# I/O 重定向与管道

> 本文是「Shell 脚本实战指南」系列的第 05 章，系统讲解 Shell 中 I/O 重定向、管道、`tee`、高级文件描述符操作、进程替换与命名管道的原理与实战用法。

---

## 目录

- [1 文件描述符基础](#1-文件描述符基础)
  - [1.1 三个标准文件描述符](#11-三个标准文件描述符)
  - [1.2 文件描述符的本质](#12-文件描述符的本质)
- [2 输出重定向](#2-输出重定向)
  - [2.1 覆盖写与追加写](#21-覆盖写与追加写)
  - [2.2 标准错误重定向](#22-标准错误重定向)
  - [2.3 合并 stdout 和 stderr](#23-合并-stdout-和-stderr)
  - [2.4 重定向顺序陷阱](#24-重定向顺序陷阱)
  - [2.5 丢弃输出](#25-丢弃输出)
  - [2.6 实战：日志分离](#26-实战日志分离)
- [3 输入重定向](#3-输入重定向)
  - [3.1 从文件读取](#31-从文件读取)
  - [3.2 Here Document 与 Here String](#32-here-document-与-here-string)
  - [3.3 实战：批量操作](#33-实战批量操作)
- [4 管道](#4-管道)
  - [4.1 管道的工作原理](#41-管道的工作原理)
  - [4.2 管道链](#42-管道链)
  - [4.3 管道与子进程](#43-管道与子进程)
  - [4.4 |& 同时传递 stdout 和 stderr](#44--同时传递-stdout-和-stderr)
  - [4.5 实战：日志分析管道](#45-实战日志分析管道)
- [5 tee 命令](#5-tee-命令)
  - [5.1 基本用法](#51-基本用法)
  - [5.2 管道中间调试](#52-管道中间调试)
  - [5.3 实战：构建日志同时显示和保存](#53-实战构建日志同时显示和保存)
- [6 高级文件描述符操作](#6-高级文件描述符操作)
  - [6.1 exec 打开与关闭文件描述符](#61-exec-打开与关闭文件描述符)
  - [6.2 自定义 fd 实现多文件读写](#62-自定义-fd-实现多文件读写)
  - [6.3 实战：日志同时写文件与发送远程](#63-实战日志同时写文件与发送远程)
- [7 进程替换](#7-进程替换)
  - [7.1 语法与原理](#71-语法与原理)
  - [7.2 进程替换 vs 管道](#72-进程替换-vs-管道)
  - [7.3 实战示例](#73-实战示例)
- [8 命名管道（FIFO）](#8-命名管道fifo)
  - [8.1 创建与使用](#81-创建与使用)
  - [8.2 实战：简单进程间通信](#82-实战简单进程间通信)
- [9 实用技巧与陷阱](#9-实用技巧与陷阱)
  - [9.1 管道中的变量赋值](#91-管道中的变量赋值)
  - [9.2 PIPESTATUS](#92-pipestatus)
  - [9.3 set -o pipefail](#93-set--o-pipefail)
  - [9.4 常见陷阱汇总](#94-常见陷阱汇总)
- [10 小结](#10-小结)
- [11 动手练习](#11-动手练习)

---

## 1 文件描述符基础

### 1.1 三个标准文件描述符

每个进程启动时，内核自动打开三个**文件描述符（File Descriptor, fd）**：

| fd | 名称 | 默认指向 | 用途 |
|----|------|----------|------|
| 0 | **stdin**（标准输入） | 键盘/终端 | 读取用户输入 |
| 1 | **stdout**（标准输出） | 终端 | 输出正常结果 |
| 2 | **stderr**（标准错误） | 终端 | 输出错误信息 |

stdout 与 stderr 默认都显示在终端上，但它们是**两条独立的流**，可以分别重定向到不同的目标。

### 1.2 文件描述符的本质

文件描述符是进程级的整数索引，指向内核维护的**打开文件表（Open File Table）**。Shell 重定向的本质就是修改 fd 的指向。

```
 进程                    内核
┌─────────────┐     ┌──────────────────┐
│  fd 表      │     │  打开文件表       │
│  0 ──────────┼────►│  /dev/tty (终端)  │
│  1 ──────────┼────►│  /dev/tty (终端)  │
│  2 ──────────┼────►│  /dev/tty (终端)  │
│  3 ──────────┼────►│  /var/log/app.log │
│  ...        │     │  ...              │
└─────────────┘     └──────────────────┘
```

重定向 `>output.log` 所做的事情，就是把 fd 1 从终端改为指向 `output.log` 文件。

---

## 2 输出重定向

### 2.1 覆盖写与追加写

**`>`（覆盖写）** 会清空目标文件后写入；**`>>`（追加写）** 在文件末尾追加。

```bash
# 覆盖写入——文件已有内容会被清空
echo "first line" > output.txt

# 追加写入——在已有内容后面添加
echo "second line" >> output.txt
```

> ⚠️ **注意**：`>` 在命令执行前就会截断文件。如果写 `cat file > file`，`file` 会先被清空，然后 `cat` 读到的是空文件，结果**丢失所有数据**。

### 2.2 标准错误重定向

`2>` 将 stderr 重定向到文件，`2>>` 追加。

```bash
# 只重定向错误输出到文件
ls /nonexistent 2> error.log

# 追加错误输出
ls /another_bad_path 2>> error.log
```

### 2.3 合并 stdout 和 stderr

两种等价写法：

```bash
# 写法一：&> （Bash 推荐写法）
command &> all.log

# 写法二：先重定向 stdout，再把 stderr 指向 stdout
command > all.log 2>&1
```

`2>&1` 的含义：把 fd 2 指向 fd 1 **当前**指向的目标。因此它必须出现在 `>` 之后。

### 2.4 重定向顺序陷阱

这是高频面试题，也是最常见的坑。

```bash
# ✅ 正确：stdout → file，stderr → stdout（即 file）
command > file 2>&1

# ❌ 错误：stderr → stdout（此时仍是终端），然后 stdout → file
command 2>&1 > file
```

用下图说明求值顺序：

```
命令: command 2>&1 > file

步骤 1 (初始状态):     fd1 → 终端,  fd2 → 终端
步骤 2 (执行 2>&1):    fd1 → 终端,  fd2 → 终端    ← fd2 复制了 fd1 的当前指向（终端）
步骤 3 (执行 > file):  fd1 → file,  fd2 → 终端    ← 只有 fd1 改了

结果：stdout 进了 file，stderr 仍然打到终端。
```

```
命令: command > file 2>&1

步骤 1 (初始状态):     fd1 → 终端,  fd2 → 终端
步骤 2 (执行 > file):  fd1 → file,  fd2 → 终端
步骤 3 (执行 2>&1):    fd1 → file,  fd2 → file    ← fd2 复制了 fd1 的当前指向（file）

结果：stdout 和 stderr 都进了 file。✅
```

> **提示**：记住一条规则——**`2>&1` 是"复制当时 fd1 的指向"，而不是"永远跟随 fd1"**。所以它必须放在 `>` 之后才能捕获 stderr。

### 2.5 丢弃输出

**`/dev/null`** 是一个特殊文件，写入的内容全部被丢弃。

```bash
# 丢弃 stdout
command > /dev/null

# 丢弃 stderr
command 2> /dev/null

# 丢弃所有输出
command &> /dev/null
```

### 2.6 实战：日志分离

将正常输出和错误输出写入不同文件，是生产环境常见需求。

```bash
#!/bin/bash
# deploy.sh —— 部署脚本，分离正常日志与错误日志

LOG_DIR="/var/log/myapp"
mkdir -p "$LOG_DIR"

# stdout → info.log，stderr → error.log
./deploy_task.sh \
    > "$LOG_DIR/info_$(date +%Y%m%d).log" \
    2> "$LOG_DIR/error_$(date +%Y%m%d).log"

# 检查是否有错误
if [ -s "$LOG_DIR/error_$(date +%Y%m%d).log" ]; then
    echo "部署过程中存在错误，请检查错误日志"
fi
```

---

## 3 输入重定向

### 3.1 从文件读取

**`<`** 将文件内容作为命令的 stdin。

```bash
# 从文件读取并排序
sort < unsorted.txt

# 等价于 sort unsorted.txt，但本质不同：
#   sort < file  → Shell 打开文件，fd0 指向它
#   sort file    → sort 自己打开文件
```

### 3.2 Here Document 与 Here String

**Here Document（`<<`）** 允许在脚本中内联多行输入：

```bash
# 用 Here Document 给 cat 提供多行输入
cat << 'EOF'
server {
    listen 80;
    server_name example.com;
}
EOF
```

定界符加引号（`'EOF'`）会禁止变量展开；不加引号则会展开变量。

**Here String（`<<<`）** 将单个字符串作为 stdin 传入：

```bash
# 将字符串传给 grep
grep "error" <<< "this is an error message"

# 常用于避免 echo | command 的写法
bc <<< "3.14 * 2"
```

> **提示**：Here Document 的详细用法见第 2 章「引用展开与运算」。

### 3.3 实战：批量操作

**批量 SQL 执行：**

```bash
# 从文件执行 SQL
mysql -u root -p mydb < schema.sql

# 用 Here Document 执行多条 SQL
mysql -u root -p mydb << 'SQL'
CREATE TABLE IF NOT EXISTS users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(64) NOT NULL
);
INSERT INTO users (name) VALUES ('alice'), ('bob');
SQL
```

**从文件逐行读取配置：**

```bash
# config.txt 格式：key=value
while IFS='=' read -r key value; do
    echo "配置项: $key = $value"
done < config.txt
```

---

## 4 管道

### 4.1 管道的工作原理

**管道（Pipe, `|`）** 将前一个命令的 stdout 连接到后一个命令的 stdin。内核在两个进程之间创建一个匿名管道缓冲区。

```
┌──────────┐  stdout   ┌──────────┐  stdout   ┌──────────┐
│ command1 ├──────────►│ command2 ├──────────►│ command3 │
└──────────┘   pipe1   └──────────┘   pipe2   └──────────┘
   fd1 → pipe1           fd0 ← pipe1
                          fd1 → pipe2           fd0 ← pipe2
```

管道只连接 stdout；stderr 默认不经过管道，仍然打到终端。

### 4.2 管道链

多级管道可以组合形成强大的数据处理流水线：

```bash
# 统计当前目录下各文件扩展名的数量，按数量降序排列
find . -type f -name '*.*' | \
    sed 's/.*\.//' | \
    sort | \
    uniq -c | \
    sort -rn
```

### 4.3 管道与子进程

管道中的**每个命令都在独立的子 Shell 中执行**。这意味着管道内的变量赋值、`cd` 等操作不会影响父 Shell。

```bash
# 常见陷阱：count 在父 Shell 中仍然是 0
count=0
echo -e "a\nb\nc" | while read -r line; do
    count=$((count + 1))
done
echo "count = $count"   # 输出 0，不是 3！
```

原因：`while` 循环在管道的子 Shell 中执行，`count` 的修改不会回传。解决方案见 [9.1 节](#91-管道中的变量赋值)。

### 4.4 |& 同时传递 stdout 和 stderr

Bash 4.0+ 引入 **`|&`**，等价于 `2>&1 |`，将 stdout 和 stderr 一起传给下一个命令。

```bash
# 同时过滤 stdout 和 stderr 中包含 "error" 的行
./build.sh |& grep -i "error"

# 等价写法
./build.sh 2>&1 | grep -i "error"
```

### 4.5 实战：日志分析管道

分析 Nginx 访问日志，找出访问量最高的 10 个 IP：

```bash
# 假设日志格式：IP - - [时间] "请求" 状态码 大小 ...
cat /var/log/nginx/access.log | \
    awk '{print $1}' | \
    sort | \
    uniq -c | \
    sort -rn | \
    head -10
```

| 阶段 | 命令 | 作用 |
|------|------|------|
| 1 | `awk '{print $1}'` | 提取第一列（IP 地址） |
| 2 | `sort` | 排序（`uniq` 要求输入有序） |
| 3 | `uniq -c` | 去重并计数 |
| 4 | `sort -rn` | 按数量降序排序 |
| 5 | `head -10` | 取前 10 行 |

查找过去一小时内的 5xx 错误：

```bash
# 提取状态码为 5xx 的请求路径，按频率排序
awk '$9 ~ /^5/ {print $7}' /var/log/nginx/access.log | \
    sort | \
    uniq -c | \
    sort -rn | \
    head -20
```

---

## 5 tee 命令

### 5.1 基本用法

**`tee`** 从 stdin 读取数据，同时输出到 stdout **和**文件。相当于一个 T 形分流器。

```bash
# 输出到终端的同时保存到文件
echo "Hello" | tee output.txt

# 追加模式（-a）
echo "World" | tee -a output.txt

# 同时写入多个文件
echo "data" | tee file1.txt file2.txt file3.txt
```

```
               ┌──► stdout（终端）
stdin ──► tee ─┤
               └──► file（磁盘）
```

### 5.2 管道中间调试

在复杂管道中插入 `tee` 可以观察中间数据，定位问题。

```bash
# 在管道中间用 tee 保存中间结果用于调试
cat access.log | \
    awk '{print $1}' | \
    tee /tmp/debug_ips.txt | \
    sort | \
    uniq -c | \
    sort -rn | \
    head -10
```

调试完毕后删除 `tee` 行即可，不影响管道逻辑。

### 5.3 实战：构建日志同时显示和保存

```bash
#!/bin/bash
# build_with_log.sh —— 构建项目并同时在终端显示和保存日志

LOG_FILE="build_$(date +%Y%m%d_%H%M%S).log"

# stdout 和 stderr 合并后通过 tee 分流
make all 2>&1 | tee "$LOG_FILE"

# 检查构建结果（使用 PIPESTATUS 获取 make 的退出码）
if [ "${PIPESTATUS[0]}" -ne 0 ]; then
    echo "构建失败，日志已保存到 $LOG_FILE"
    exit 1
fi
echo "构建成功"
```

---

## 6 高级文件描述符操作

### 6.1 exec 打开与关闭文件描述符

**`exec`** 可以在不启动新进程的情况下操作文件描述符。

| 语法 | 含义 |
|------|------|
| `exec 3>file` | 打开 fd 3，用于**写入** file |
| `exec 3>>file` | 打开 fd 3，用于**追加写入** file |
| `exec 3<file` | 打开 fd 3，用于**读取** file |
| `exec 3<>file` | 打开 fd 3，用于**读写** file |
| `exec 3>&-` | **关闭**写入方向的 fd 3 |
| `exec 3<&-` | **关闭**读取方向的 fd 3 |

```bash
#!/bin/bash
# 打开 fd 3 写入日志文件
exec 3> /tmp/custom.log

echo "普通输出到终端"
echo "这行写入自定义日志" >&3
echo "这行也写入自定义日志" >&3

# 用完关闭
exec 3>&-
```

### 6.2 自定义 fd 实现多文件读写

同时从两个文件读取数据，逐行合并：

```bash
#!/bin/bash
# merge_files.sh —— 逐行合并两个文件

exec 3< file_a.txt
exec 4< file_b.txt

while IFS= read -r line_a <&3; do
    IFS= read -r line_b <&4
    echo "$line_a | $line_b"
done

exec 3<&-
exec 4<&-
```

同时写入多个目标：

```bash
#!/bin/bash
# multi_output.sh —— 分级日志

exec 3> info.log
exec 4> debug.log

log_info()  { echo "[INFO]  $(date '+%H:%M:%S') $*" >&3; }
log_debug() { echo "[DEBUG] $(date '+%H:%M:%S') $*" >&4; }

log_info  "服务启动"
log_debug "加载配置文件 /etc/app.conf"
log_info  "监听端口 8080"
log_debug "初始化数据库连接池，大小=10"

exec 3>&-
exec 4>&-
```

### 6.3 实战：日志同时写文件与发送远程

利用 `exec` + `tee` + 进程替换实现一路写本地、一路发远程：

```bash
#!/bin/bash
# dual_log.sh —— 日志同时写本地文件和发送到远程 syslog

LOCAL_LOG="/var/log/myapp/app.log"
REMOTE_HOST="logserver.example.com"

# fd 1 通过 tee 分流：一路写本地文件，stdout 保持终端输出
exec > >(tee -a "$LOCAL_LOG")
# fd 2 合并到 fd 1
exec 2>&1

echo "应用启动于 $(date)"
echo "配置加载完成"
```

---

## 7 进程替换

### 7.1 语法与原理

**进程替换（Process Substitution）** 是 Bash 特有的功能，用 `<(command)` 或 `>(command)` 将命令的输出/输入伪装成一个文件名。

```bash
# <(command) 产生一个可读的伪文件
echo <(ls)          # 输出类似 /dev/fd/63

# 可以像普通文件一样使用
cat <(echo "hello from process substitution")
```

内核实现方式：Bash 创建一个匿名管道或 `/dev/fd/N`，将命令的输出连接到该 fd，然后把 `/dev/fd/N` 作为文件名传给外层命令。

### 7.2 进程替换 vs 管道

| 对比项 | 管道 `|` | 进程替换 `<()` |
|--------|----------|----------------|
| 传递方式 | stdin → stdin | 伪文件路径 |
| 可使用个数 | 一个 stdin | 多个并行 |
| 是否影响变量 | 是（子 Shell） | 否（不创建子 Shell） |
| 适用场景 | 线性流水线 | 需要多个输入源的命令 |

关键优势：管道只能连一个 stdin，而进程替换可以给命令提供**多个文件参数**。

### 7.3 实战示例

**比较两个命令的输出差异：**

```bash
# diff 要求两个文件参数，管道做不到，进程替换可以
diff <(ls /dir1) <(ls /dir2)

# 比较排序前后的差异
diff <(cat file.txt) <(sort file.txt)
```

**合并多个文件的不同列：**

```bash
# 从两个文件各取一列，按行合并
paste <(cut -f1 users.tsv) <(cut -f3 scores.tsv)
```

**对比两台服务器的配置：**

```bash
diff <(ssh server1 cat /etc/nginx/nginx.conf) \
     <(ssh server2 cat /etc/nginx/nginx.conf)
```

**用 `>(command)` 实现输出分流：**

```bash
# 将 stdout 同时发送到 gzip 压缩和 wc 统计
command | tee >(gzip > output.gz) >(wc -l > linecount.txt) > /dev/null
```

---

## 8 命名管道（FIFO）

### 8.1 创建与使用

**命名管道（Named Pipe / FIFO）** 是文件系统中的一种特殊文件，提供先进先出的进程间通信通道。与匿名管道不同，它有文件名，不相关的进程也能通过它通信。

```bash
# 创建命名管道
mkfifo /tmp/myfifo

# 查看文件类型（p 表示管道）
ls -l /tmp/myfifo
# prw-r--r--  1 user  group  0 Mar  9 10:00 /tmp/myfifo
```

读写命名管道的行为是**阻塞的**：写入端会阻塞直到有读取端打开管道，反之亦然。

```bash
# 终端 1：写入（会阻塞，等待读取端）
echo "message from writer" > /tmp/myfifo

# 终端 2：读取
cat /tmp/myfifo
# 输出: message from writer
```

### 8.2 实战：简单进程间通信

用命名管道实现一个简单的生产者-消费者模型：

```bash
#!/bin/bash
# producer.sh —— 生产者
PIPE="/tmp/task_pipe"
mkfifo "$PIPE" 2>/dev/null

for i in $(seq 1 5); do
    echo "task-$i" > "$PIPE"
    echo "已发送: task-$i"
done
```

```bash
#!/bin/bash
# consumer.sh —— 消费者
PIPE="/tmp/task_pipe"

while read -r task < "$PIPE"; do
    echo "处理中: $task"
    sleep 1
    echo "完成: $task"
done

# 清理
rm -f "$PIPE"
```

> ⚠️ **注意**：命名管道用完后需要手动删除（`rm`）。脚本中应在 `trap` 中注册清理逻辑，避免残留。

---

## 9 实用技巧与陷阱

### 9.1 管道中的变量赋值

管道中的命令在子 Shell 中执行，变量修改无法传回父 Shell。

**解决方案一：进程替换替代管道**

```bash
count=0
while read -r line; do
    count=$((count + 1))
done < <(echo -e "a\nb\nc")
echo "count = $count"   # 输出 3 ✅
```

**解决方案二：`lastpipe` 选项（Bash 4.2+）**

```bash
#!/bin/bash
shopt -s lastpipe

count=0
echo -e "a\nb\nc" | while read -r line; do
    count=$((count + 1))
done
echo "count = $count"   # 输出 3 ✅
```

`lastpipe` 使管道的最后一个命令在当前 Shell（而非子 Shell）中执行。

> ⚠️ **注意**：`lastpipe` 仅在脚本中生效，交互式 Shell 中默认启用 job control，`lastpipe` 不起作用。

### 9.2 PIPESTATUS

**`PIPESTATUS`** 是 Bash 数组，保存最近一条管道中**每个命令**的退出码。

```bash
false | true | false
echo "${PIPESTATUS[@]}"    # 输出: 1 0 1
echo "${PIPESTATUS[0]}"    # 第一个命令的退出码: 1
echo "${PIPESTATUS[2]}"    # 第三个命令的退出码: 1
```

普通的 `$?` 只返回管道中**最后一个命令**的退出码：

```bash
false | true
echo $?                    # 输出: 0（只看 true 的退出码）
```

### 9.3 set -o pipefail

默认情况下，管道的退出码是最后一个命令的退出码。启用 **`pipefail`** 后，管道中**任一命令**失败，整个管道就返回非零退出码。

```bash
set -o pipefail

false | true
echo $?    # 输出: 1（false 失败了）

true | true
echo $?    # 输出: 0
```

> **提示**：在生产脚本开头加上 `set -euo pipefail` 是最佳实践。`-e` 遇错退出，`-u` 未定义变量报错，`-o pipefail` 管道失败即退出。

### 9.4 常见陷阱汇总

| 陷阱 | 说明 | 正确做法 |
|------|------|----------|
| `cat file > file` | `>` 先截断文件，`cat` 读到空文件 | 用临时文件或 `sponge` |
| `command 2>&1 > file` | stderr 仍到终端 | `command > file 2>&1` |
| 管道中变量赋值 | 子 Shell 中的修改不影响父 Shell | 用进程替换或 `lastpipe` |
| 只看 `$?` | 只反映最后一个命令的退出码 | 用 `PIPESTATUS` 或 `pipefail` |
| `echo $var \| read x` | `read` 在子 Shell，`x` 读不到 | `read x <<< "$var"` |
| FIFO 未清理 | 残留的命名管道可能阻塞后续脚本 | 在 `trap EXIT` 中 `rm -f` |
| `set -e` + 管道 | 中间命令失败不会触发退出 | 同时启用 `pipefail` |

---

## 10 小结

1. **fd 是抽象层**——Shell 重定向的本质是修改 fd 0/1/2 的指向，理解这一点就掌握了所有重定向语法的底层逻辑。
2. **`>` 与 `>>` 的区别是覆盖与追加**——生产环境几乎总是用 `>>`，防止日志丢失。
3. **`2>&1` 的位置决定行为**——它是"复制当时 fd1 的指向"，必须放在 `>` 之后才能合并 stderr。
4. **管道只传 stdout**——stderr 不经过管道，需要时用 `|&` 或 `2>&1 |`。
5. **管道中的命令运行在子 Shell**——变量赋值不会影响父 Shell，用进程替换或 `lastpipe` 解决。
6. **`tee` 是管道的调试利器**——在任意位置插入 `tee` 可以观察中间数据。
7. **`exec` 操作 fd 不启动新进程**——适合在脚本内持久打开/关闭自定义 fd。
8. **进程替换 `<()` 把命令输出变成文件名**——解决了管道只能连一个 stdin 的限制。
9. **命名管道（FIFO）实现无亲缘进程间通信**——但需要注意阻塞行为和清理。
10. **`set -euo pipefail` 是健壮脚本的标配**——让管道中的错误不被静默吞掉。

---

## 11 动手练习

**练习 1：日志分级输出**

编写脚本 `log_splitter.sh`，执行一个会同时产生 stdout 和 stderr 的命令（如 `find / -name "*.conf" -print 2>&1`），将 stdout 保存到 `found.log`，stderr 保存到 `errors.log`，同时在终端显示两者的合并输出。

提示：结合 `exec`、`tee` 和自定义 fd。

**练习 2：管道数据处理**

有一个 CSV 文件 `sales.csv`，格式为 `日期,产品,数量,单价`。用一行管道命令完成：提取产品名称列 → 排序 → 去重计数 → 按数量降序排列 → 取前 5 名。

验证：在管道中间插入 `tee /tmp/debug.txt` 观察中间结果是否符合预期。

**练习 3：进程替换对比**

使用 `diff` + 进程替换，比较 `/etc` 下两个不同目录（或两台机器）的文件列表差异。尝试用管道实现同样的功能，体会为什么进程替换更方便。

**练习 4：健壮管道脚本**

编写脚本 `safe_pipeline.sh`，启用 `set -euo pipefail`，构建一个 3 级以上的管道。在管道中间故意插入一个会失败的命令（如 `false`），观察 `PIPESTATUS` 数组和 `$?` 的值。分别测试启用和未启用 `pipefail` 时的行为差异。

---

> 上一章：[04 - 函数与脚本组织](../04-函数与脚本组织/函数与脚本组织.md) | 下一章：[06 - 文本处理三剑客](../06-文本处理三剑客/文本处理三剑客.md)
