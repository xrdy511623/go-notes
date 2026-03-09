# Shell 基础与变量

> 本文是「Shell 脚本实战指南」系列的第 01 章，涵盖 Shell 的本质、脚本编写与执行方式、变量体系（普通变量、环境变量、特殊变量、数组）以及字符串操作，为后续章节打下坚实基础。

---

## 目录

- [1 Shell 是什么](#1-shell-是什么)
  - [1.1 Shell 在操作系统中的角色](#11-shell-在操作系统中的角色)
  - [1.2 常见 Shell 种类](#12-常见-shell-种类)
- [2 第一个 Shell 脚本](#2-第一个-shell-脚本)
  - [2.1 Shebang 行](#21-shebang-行)
  - [2.2 创建、赋权、执行](#22-创建赋权执行)
  - [2.3 四种执行方式对比](#23-四种执行方式对比)
  - [2.4 内建命令 vs 外部命令](#24-内建命令-vs-外部命令)
- [3 变量基础](#3-变量基础)
  - [3.1 定义与赋值](#31-定义与赋值)
  - [3.2 引用变量](#32-引用变量)
  - [3.3 删除变量](#33-删除变量)
  - [3.4 只读变量](#34-只读变量)
- [4 变量作用域](#4-变量作用域)
  - [4.1 局部变量 vs 环境变量](#41-局部变量-vs-环境变量)
  - [4.2 export 导出到子进程](#42-export-导出到子进程)
  - [4.3 local 关键字](#43-local-关键字)
- [5 特殊变量](#5-特殊变量)
  - [5.1 退出状态 $?](#51-退出状态-)
  - [5.2 进程相关变量](#52-进程相关变量)
  - [5.3 脚本参数](#53-脚本参数)
  - [5.4 $@ vs $* 的区别](#54--vs--的区别)
- [6 环境变量与配置文件](#6-环境变量与配置文件)
  - [6.1 常用环境变量](#61-常用环境变量)
  - [6.2 配置文件加载顺序](#62-配置文件加载顺序)
  - [6.3 永久设置环境变量](#63-永久设置环境变量)
- [7 数组](#7-数组)
  - [7.1 索引数组](#71-索引数组)
  - [7.2 关联数组](#72-关联数组)
  - [7.3 实战：批量处理服务器 IP](#73-实战批量处理服务器-ip)
- [8 declare 与变量属性](#8-declare-与变量属性)
- [9 字符串操作](#9-字符串操作)
  - [9.1 长度与子串](#91-长度与子串)
  - [9.2 默认值替换](#92-默认值替换)
  - [9.3 模式删除](#93-模式删除)
  - [9.4 模式替换](#94-模式替换)
  - [9.5 实战：从路径提取信息](#95-实战从路径提取信息)
- [10 小结](#10-小结)
- [11 动手练习](#11-动手练习)

---

## 1 Shell 是什么

### 1.1 Shell 在操作系统中的角色

**Shell**（壳）是用户与操作系统内核之间的命令解释器。用户输入命令，Shell 解析后调用内核提供的系统调用，内核完成实际工作后将结果返回给 Shell，再展示给用户。

```
┌──────────────────────────────────────────────┐
│                   用户 (User)                │
│          键盘输入 / 终端 / 脚本文件           │
└──────────────┬───────────────────────────────┘
               │ 命令 / 脚本
               ▼
┌──────────────────────────────────────────────┐
│              Shell (bash / zsh)              │
│  ┌────────────────────────────────────────┐  │
│  │  词法分析 → 语法解析 → 命令查找 → 执行 │  │
│  └────────────────────────────────────────┘  │
└──────────────┬───────────────────────────────┘
               │ 系统调用 (fork / exec / open …)
               ▼
┌──────────────────────────────────────────────┐
│             内核 (Kernel)                    │
│  进程管理 / 内存管理 / 文件系统 / 设备驱动   │
└──────────────────────────────────────────────┘
```

Shell 本身也是一个普通的用户态程序，可被替换。

### 1.2 常见 Shell 种类

| Shell | 路径 | 特点 |
|-------|------|------|
| **sh** (Bourne Shell) | `/bin/sh` | POSIX 标准，语法最精简 |
| **bash** (Bourne Again Shell) | `/bin/bash` | Linux 默认，兼容 sh，功能丰富 |
| **zsh** | `/bin/zsh` | macOS 默认（Catalina+），插件生态强大 |
| **dash** | `/usr/bin/dash` | Debian/Ubuntu 的 `/bin/sh` 实现，启动极快 |
| **fish** | `/usr/bin/fish` | 开箱即用的补全与高亮，语法不兼容 POSIX |

查看当前使用的 Shell 和系统支持的所有 Shell：

```bash
# 当前登录 Shell
echo $SHELL

# 当前运行的 Shell 进程
echo $0

# 系统所有可用 Shell
cat /etc/shells
```

> **提示**：`$SHELL` 反映的是登录 Shell（写在 `/etc/passwd` 中），而 `$0` 反映的是当前实际运行的 Shell 进程，两者可能不同。

---

## 2 第一个 Shell 脚本

### 2.1 Shebang 行

脚本第一行以 `#!` 开头，称为 **Shebang**（也叫 hashbang），告诉系统用哪个解释器执行本文件。

```bash
#!/bin/bash
```

更推荐的写法：

```bash
#!/usr/bin/env bash
```

`env` 会在 `$PATH` 中搜索 `bash`，因此不依赖 bash 的绝对路径。在不同发行版中 bash 可能位于 `/bin/bash` 或 `/usr/local/bin/bash`，用 `env` 可以做到跨平台兼容。

### 2.2 创建、赋权、执行

```bash
# 创建脚本
cat > hello.sh << 'EOF'
#!/usr/bin/env bash
echo "Hello, Shell!"
EOF

# 添加可执行权限
chmod u+x hello.sh

# 执行
./hello.sh
```

### 2.3 四种执行方式对比

```bash
# 方式 1：显式指定解释器
bash hello.sh

# 方式 2：作为可执行文件（需要 +x 权限和 Shebang）
./hello.sh

# 方式 3：source 在当前 Shell 进程中执行
source hello.sh

# 方式 4：点号，source 的简写
. hello.sh
```

| 执行方式 | 是否开子进程 | 是否需要执行权限 | 对当前 Shell 的影响 |
|---------|:----------:|:--------------:|:------------------:|
| `bash script.sh` | 是 | 否 | 无 |
| `./script.sh` | 是 | **是** | 无 |
| `source script.sh` | 否 | 否 | **有** |
| `. script.sh` | 否 | 否 | **有** |

用一个例子直观感受区别：

```bash
# 创建测试脚本
cat > test_cd.sh << 'EOF'
#!/usr/bin/env bash
cd /tmp && pwd
EOF
chmod u+x test_cd.sh

# 当前目录
pwd
# 输出: /home/user

# 子进程执行 —— 当前目录不受影响
./test_cd.sh
# 输出: /tmp
pwd
# 输出: /home/user   ← 没有变化

# 当前进程执行 —— 当前目录被改变
source test_cd.sh
# 输出: /tmp
pwd
# 输出: /tmp          ← 已经切换
```

> ⚠️ **注意**：`source` 会直接影响当前 Shell 环境（变量、工作目录等），在执行未知脚本时请优先使用 `bash script.sh` 或 `./script.sh`。

### 2.4 内建命令 vs 外部命令

**内建命令**（builtin）由 Shell 自身实现，不产生子进程；**外部命令**（external）是独立的可执行文件，Shell 通过 `fork + exec` 启动。

```bash
# cd 是内建命令
type cd
# 输出: cd is a shell builtin

# ls 是外部命令
type ls
# 输出: ls is /bin/ls（路径可能因系统而异）

# 查看所有内建命令
enable -a
```

内建命令的关键优势：无需创建子进程，且能直接修改当前 Shell 状态（如 `cd` 改变工作目录、`export` 设置环境变量）。

---

## 3 变量基础

### 3.1 定义与赋值

Shell 变量赋值使用 `=`，**等号两边不能有空格**。

```bash
# 正确
name="John"

# 错误！Shell 会把 name 当作命令，= 和值当作参数
name = "John"
# bash: name: command not found
```

> ⚠️ **注意**：等号两边不能有空格，这是 Shell 新手最常犯的错误。与大多数编程语言不同，Shell 依靠空格来分隔命令和参数。

几种赋值方式：

```bash
# 直接赋值
host="192.168.1.1"

# 使用 let 进行算术赋值
let count=10+20
echo $count    # 30

# 将命令本身赋值给变量
cmd=ls

# 将命令的输出赋值给变量（推荐 $() 而非反引号）
file_list=$(ls /etc)
current_date=$(date +%Y-%m-%d)
```

变量命名规则：字母、数字、下划线，不能以数字开头。

### 3.2 引用变量

使用 `$var` 或 `${var}` 引用变量值。

```bash
name="world"

# 两种方式等价
echo $name
echo ${name}
```

当变量名与紧随其后的字符可能产生歧义时，**必须**使用花括号：

```bash
fruit="apple"

# 想输出 "apples"
echo "$fruits"     # 空！Shell 认为变量名是 fruits（未定义）
echo "${fruit}s"   # apples ✓
```

> **提示**：养成始终使用 `${var}` 的习惯，可以避免很多意想不到的错误。

### 3.3 删除变量

```bash
name="John"
echo $name   # John

unset name
echo $name   # （空）
```

`unset` 删除后变量不再存在，与空值 `name=""` 不同——后者变量仍然存在，只是值为空。

### 3.4 只读变量

```bash
readonly PI=3.14159

PI=3.0
# bash: PI: readonly variable

unset PI
# bash: unset: PI: cannot unset: readonly variable
```

只读变量在当前 Shell 生命周期内无法修改或删除。

---

## 4 变量作用域

### 4.1 局部变量 vs 环境变量

Shell 变量默认是**局部变量**（shell variable），仅在当前 Shell 进程可见。**环境变量**（environment variable）会被子进程继承。

```
┌─────────────────────────┐
│     父 Shell 进程        │
│  name="John"  (局部)    │ ──× 子进程看不到
│  export APP="web" (环境) │ ──→ 子进程可继承
└──────────┬──────────────┘
           │ fork
           ▼
┌─────────────────────────┐
│     子 Shell 进程        │
│  echo $name  → (空)     │
│  echo $APP   → web      │
└─────────────────────────┘
```

### 4.2 export 导出到子进程

```bash
# 定义局部变量
greeting="hello"

# 启动子 Shell，尝试读取
bash -c 'echo "greeting=$greeting"'
# 输出: greeting=          ← 子进程看不到

# 导出为环境变量
export greeting

# 再次启动子 Shell
bash -c 'echo "greeting=$greeting"'
# 输出: greeting=hello     ← 子进程可以看到
```

也可以在定义时直接导出：

```bash
export APP_PORT=8080
```

> **提示**：环境变量只能向下传递（父→子），子进程中修改环境变量不会影响父进程。

### 4.3 local 关键字

`local` 用于在函数内部定义局部变量，防止污染全局作用域。这里先做简单演示，详细用法见第 04 章。

```bash
demo() {
    local msg="inside function"
    echo $msg
}

demo
# 输出: inside function

echo $msg
# 输出: （空）—— msg 仅存在于函数内部
```

---

## 5 特殊变量

### 5.1 退出状态 $?

每条命令执行后都会返回一个**退出状态码**（exit status），`0` 表示成功，非 `0` 表示失败。

```bash
ls /etc/passwd
echo $?    # 0（成功）

ls /nonexistent
echo $?    # 2（失败，文件不存在）
```

> **提示**：`$?` 的值只在紧接命令之后有效，下一条命令执行后就会被覆盖。如需保留，应立即赋值给变量：`ret=$?`。

### 5.2 进程相关变量

| 变量 | 含义 | 示例 |
|------|------|------|
| `$$` | 当前 Shell 进程的 PID | `echo $$` |
| `$!` | 最近一个后台进程的 PID | `sleep 10 & echo $!` |
| `$PPID` | 父进程的 PID | `echo $PPID` |

```bash
echo "当前 PID: $$"
sleep 30 &
echo "后台进程 PID: $!"
echo "父进程 PID: $PPID"
```

### 5.3 脚本参数

| 变量 | 含义 |
|------|------|
| `$0` | 脚本名（或当前 Shell 名称） |
| `$1` ~ `$9` | 第 1 到第 9 个位置参数 |
| `${10}` | 第 10 个及以后的参数（必须用花括号） |
| `$#` | 参数个数 |
| `$@` | 所有参数（每个参数独立） |
| `$*` | 所有参数（合并为一个字符串） |

```bash
#!/usr/bin/env bash
echo "脚本名: $0"
echo "参数个数: $#"
echo "第一个参数: $1"
echo "第二个参数: $2"
echo "所有参数: $@"
```

执行：

```bash
bash params.sh foo bar baz
# 脚本名: params.sh
# 参数个数: 3
# 第一个参数: foo
# 第二个参数: bar
# 所有参数: foo bar baz
```

### 5.4 $@ vs $* 的区别

不加引号时 `$@` 和 `$*` 行为相同。**关键区别在双引号中**：

- `"$@"` 展开为多个独立的字符串：`"$1" "$2" "$3" ...`
- `"$*"` 展开为一个整体字符串：`"$1c$2c$3c..."`（`c` 是 `$IFS` 的第一个字符，默认空格）

```bash
#!/usr/bin/env bash

echo '--- 使用 "$@" ---'
for arg in "$@"; do
    echo "  [$arg]"
done

echo '--- 使用 "$*" ---'
for arg in "$*"; do
    echo "  [$arg]"
done
```

执行：

```bash
bash test_args.sh "hello world" foo bar
# --- 使用 "$@" ---
#   [hello world]
#   [foo]
#   [bar]
# --- 使用 "$*" ---
#   [hello world foo bar]
```

`"$@"` 保持了每个参数的边界，`"$*"` 将所有参数合并为一个字符串。

> ⚠️ **注意**：几乎所有场景都应使用 `"$@"` 而非 `"$*"`，以正确处理含空格的参数。

---

## 6 环境变量与配置文件

### 6.1 常用环境变量

| 变量 | 含义 | 示例值 |
|------|------|--------|
| `PATH` | 可执行文件搜索路径 | `/usr/local/bin:/usr/bin:/bin` |
| `HOME` | 当前用户家目录 | `/home/john` |
| `USER` | 当前用户名 | `john` |
| `SHELL` | 登录 Shell 路径 | `/bin/bash` |
| `LANG` | 系统语言与编码 | `en_US.UTF-8` |
| `PS1` | 命令提示符格式 | `\u@\h:\w\$` |
| `TERM` | 终端类型 | `xterm-256color` |
| `EDITOR` | 默认编辑器 | `vim` |

```bash
# 查看所有环境变量
env

# 查看特定变量
echo $PATH

# 临时修改 PATH
export PATH="$HOME/bin:$PATH"
```

### 6.2 配置文件加载顺序

bash 的配置文件分为两大场景：**login shell**（如 SSH 登录、`su - user`）和 **non-login shell**（如打开新终端窗口、执行 `bash`）。

```
Login Shell 加载链：
┌──────────────────┐
│  /etc/profile    │  ← 全局，所有用户
└───────┬──────────┘
        ▼
┌──────────────────┐
│  /etc/profile.d/ │  ← 模块化全局配置
│  *.sh            │
└───────┬──────────┘
        ▼
┌──────────────────────┐
│  ~/.bash_profile     │  ← 用户级，login 时读取
│  (或 ~/.bash_login   │     三者按顺序找到第一个即停
│   或 ~/.profile)     │
└───────┬──────────────┘
        ▼
┌──────────────────┐
│  ~/.bashrc       │  ← 通常被 ~/.bash_profile source 引入
└───────┬──────────┘
        ▼
┌──────────────────┐
│  /etc/bashrc     │  ← 通常被 ~/.bashrc source 引入
└──────────────────┘

Non-login Shell 加载链：
┌──────────────────┐
│  ~/.bashrc       │  ← 仅读取此文件
└───────┬──────────┘
        ▼
┌──────────────────┐
│  /etc/bashrc     │  ← 通常被 ~/.bashrc source 引入
└──────────────────┘
```

> **提示**：正因为 non-login shell 只加载 `~/.bashrc`，最稳妥的做法是把自定义配置写在 `~/.bashrc`，然后在 `~/.bash_profile` 中 source 它。

### 6.3 永久设置环境变量

```bash
# 方式 1：写入 ~/.bashrc（推荐，login 和 non-login 都生效）
echo 'export GOPATH="$HOME/go"' >> ~/.bashrc

# 方式 2：写入 /etc/profile.d/ 下的独立文件（对所有用户生效）
sudo tee /etc/profile.d/go.sh << 'EOF'
export GOROOT=/usr/local/go
export PATH="$GOROOT/bin:$PATH"
EOF

# 使改动立即生效（仅对当前终端）
source ~/.bashrc
```

> ⚠️ **注意**：不要直接修改 `/etc/profile`，应在 `/etc/profile.d/` 下创建独立 `.sh` 文件，便于管理和回退。

---

## 7 数组

bash 从 4.0 起支持**索引数组**（indexed array）和**关联数组**（associative array）。

### 7.1 索引数组

```bash
# 定义
fruits=("apple" "banana" "cherry")

# 读取单个元素（索引从 0 开始）
echo ${fruits[0]}    # apple
echo ${fruits[2]}    # cherry

# 读取所有元素
echo ${fruits[@]}    # apple banana cherry

# 数组长度
echo ${#fruits[@]}   # 3

# 追加元素
fruits+=("durian")
echo ${fruits[@]}    # apple banana cherry durian

# 修改元素
fruits[1]="blueberry"

# 删除元素
unset fruits[2]
echo ${fruits[@]}    # apple blueberry durian

# 遍历
for f in "${fruits[@]}"; do
    echo "水果: $f"
done
```

> **提示**：遍历数组时务必加双引号 `"${arr[@]}"`，否则含空格的元素会被拆分。

### 7.2 关联数组

关联数组相当于其他语言中的字典 / Map，使用前必须用 `declare -A` 声明。

```bash
# 声明关联数组
declare -A user_roles

# 赋值
user_roles["alice"]="admin"
user_roles["bob"]="developer"
user_roles["carol"]="viewer"

# 读取
echo ${user_roles["alice"]}   # admin

# 所有键
echo ${!user_roles[@]}        # alice bob carol

# 所有值
echo ${user_roles[@]}         # admin developer viewer

# 遍历
for key in "${!user_roles[@]}"; do
    echo "$key -> ${user_roles[$key]}"
done
```

### 7.3 实战：批量处理服务器 IP

```bash
#!/usr/bin/env bash

servers=("192.168.1.10" "192.168.1.11" "192.168.1.12" "10.0.0.5")

for ip in "${servers[@]}"; do
    if ping -c 1 -W 2 "$ip" &>/dev/null; then
        echo "[OK]   $ip 可达"
    else
        echo "[FAIL] $ip 不可达"
    fi
done
```

---

## 8 declare 与变量属性

`declare`（等价于 `typeset`）用于显式设置变量的属性。

| 选项 | 作用 | 等价写法 |
|------|------|----------|
| `declare -i var` | 整数变量，赋值时自动做算术运算 | — |
| `declare -r var` | 只读变量 | `readonly var` |
| `declare -a var` | 索引数组 | — |
| `declare -A var` | 关联数组 | — |
| `declare -x var` | 导出为环境变量 | `export var` |
| `declare -l var` | 值自动转为小写（bash 4+） | — |
| `declare -u var` | 值自动转为大写（bash 4+） | — |

```bash
# 整数变量
declare -i num
num=5+3
echo $num      # 8（自动运算，而非字符串 "5+3"）

# 大小写转换
declare -l lower_str
lower_str="Hello World"
echo $lower_str    # hello world

declare -u upper_str
upper_str="Hello World"
echo $upper_str    # HELLO WORLD

# 查看变量属性
declare -p num
# declare -i num="8"
```

> **提示**：`declare -x` 与 `export` 完全等价。在函数内部 `declare` 默认创建局部变量，行为等同于 `local`。

---

## 9 字符串操作

bash 内置了丰富的字符串操作语法，无需调用外部命令（如 `sed`、`awk`），效率更高。

### 9.1 长度与子串

```bash
str="Hello, Shell!"

# 字符串长度
echo ${#str}             # 13

# 子串截取: ${var:offset:length}
echo ${str:0:5}          # Hello
echo ${str:7}            # Shell!（省略 length，截取到末尾）
echo ${str: -6}          # Shell!（注意冒号后有空格，否则会被解析为默认值语法）
echo ${str:7:5}          # Shell
```

### 9.2 默认值替换

当变量未定义或为空时，可以用默认值语法提供兜底。

| 语法 | 含义 |
|------|------|
| `${var:-default}` | var 未设置或为空 → 返回 default，**不修改** var |
| `${var:=default}` | var 未设置或为空 → 返回 default，**同时将** var **设为** default |
| `${var:+alt}` | var **已设置且非空** → 返回 alt，否则返回空 |
| `${var:?error}` | var 未设置或为空 → 输出 error 并退出脚本 |

```bash
# :- 只返回默认值，不修改变量
unset name
echo ${name:-"anonymous"}   # anonymous
echo $name                  # （空）

# := 返回默认值并同时赋值
echo ${name:="anonymous"}   # anonymous
echo $name                  # anonymous

# :+ 变量有值时返回替代值
role="admin"
echo ${role:+"[已设置]"}    # [已设置]

# :? 变量为空时报错退出
unset db_host
echo ${db_host:?"错误: db_host 未设置"}
# bash: db_host: 错误: db_host 未设置
```

> **提示**：去掉冒号（如 `${var-default}`）只检查变量是否未定义，不检查是否为空字符串。带冒号的版本更常用。

### 9.3 模式删除

从字符串的头部或尾部删除匹配的模式（支持通配符 `*`、`?`、`[]`）。

| 语法 | 含义 | 助记 |
|------|------|------|
| `${var#pattern}` | 从**头部**删除**最短**匹配 | `#` 在键盘左侧 → 从左删 |
| `${var##pattern}` | 从**头部**删除**最长**匹配 | 贪婪 |
| `${var%pattern}` | 从**尾部**删除**最短**匹配 | `%` 在键盘右侧 → 从右删 |
| `${var%%pattern}` | 从**尾部**删除**最长**匹配 | 贪婪 |

```bash
path="/home/john/projects/app/main.go"

# 从头部删除最短匹配 */
echo ${path#*/}      # home/john/projects/app/main.go

# 从头部删除最长匹配 */（提取文件名）
echo ${path##*/}     # main.go

# 从尾部删除最短匹配 .*（去掉扩展名）
echo ${path%.*}      # /home/john/projects/app/main

# 从尾部删除最长匹配 /*（不常用）
echo ${path%%/*}     # （空，因为路径以 / 开头）
```

### 9.4 模式替换

```bash
text="foo-bar-baz-foo"

# 替换第一个匹配
echo ${text/foo/FOO}     # FOO-bar-baz-foo

# 替换所有匹配
echo ${text//foo/FOO}    # FOO-bar-baz-FOO

# 仅匹配开头（类似 ^）
echo ${text/#foo/FOO}    # FOO-bar-baz-foo

# 仅匹配结尾（类似 $）
echo ${text/%foo/FOO}    # foo-bar-baz-FOO

# 删除匹配（替换为空）
echo ${text//foo/}       # -bar-baz-
```

### 9.5 实战：从路径提取信息

```bash
#!/usr/bin/env bash

filepath="/var/log/nginx/access.log.gz"

# 提取目录名（等价于 dirname）
dir=${filepath%/*}
echo "目录: $dir"          # /var/log/nginx

# 提取文件名（等价于 basename）
filename=${filepath##*/}
echo "文件名: $filename"    # access.log.gz

# 提取扩展名
ext=${filename##*.}
echo "扩展名: $ext"         # gz

# 去掉最后一个扩展名
name_no_ext=${filename%.*}
echo "去扩展名: $name_no_ext"  # access.log

# 去掉所有扩展名
base=${filename%%.*}
echo "基础名: $base"        # access

# 替换路径中的目录
new_path=${filepath/nginx/apache}
echo "新路径: $new_path"    # /var/log/apache/access.log.gz
```

> **提示**：纯 bash 字符串操作比调用 `dirname`、`basename`、`sed` 等外部命令快得多，在循环中处理大量路径时尤其明显。

---

## 10 小结

1. **Shell 是命令解释器**，位于用户与内核之间，常用 bash 和 zsh。
2. **Shebang 行**推荐使用 `#!/usr/bin/env bash` 以提高可移植性。
3. **四种执行方式**：`bash script.sh` 和 `./script.sh` 在子进程运行，`source` 和 `.` 在当前进程运行，对环境的影响截然不同。
4. **变量赋值等号两侧不能有空格**，这是最常见的新手陷阱。
5. **变量默认只在当前进程可见**，通过 `export` 导出后子进程才能继承，但子进程无法反向影响父进程。
6. **特殊变量**（`$?`、`$$`、`$@`、`$#` 等）是编写健壮脚本的基础。
7. **`"$@"` 几乎总是优于 `"$*"`**，因为它能正确保持参数边界。
8. **配置文件加载**分 login shell 和 non-login shell 两条链，自定义变量建议写在 `~/.bashrc`。
9. **数组**分为索引数组和关联数组（`declare -A`），遍历时记得加双引号。
10. **字符串操作**（默认值、模式删除、模式替换）是 bash 内置能力，无需外部命令，性能更优。

---

## 11 动手练习

**练习 1：脚本执行方式验证**

编写一个脚本 `scope_test.sh`，在其中定义变量 `MY_VAR="from_script"` 并 `cd /tmp`。分别用 `bash`、`./`、`source` 三种方式执行，执行后在终端检查 `echo $MY_VAR` 和 `pwd`，记录三次结果的差异，解释原因。

**练习 2：参数处理脚本**

编写一个脚本 `greet.sh`，接受 2~N 个参数。要求：
- 第一个参数为问候语，其余参数为人名
- 若参数不足 2 个，使用 `${var:?msg}` 输出错误信息并退出
- 用 `"$@"` 遍历所有人名并输出格式为 `问候语, 人名!`
- 测试含空格的参数是否正确处理

**练习 3：路径解析工具**

编写脚本 `pathinfo.sh`，接受一个文件路径作为参数，使用纯 bash 字符串操作（不调用 `dirname`、`basename`）输出：目录名、文件名、扩展名、无扩展名的文件名。对 `/var/log/app.log.gz` 和 `script.sh` 两种输入分别测试。

**练习 4：服务器巡检脚本**

定义一个包含 5 个 IP 地址的索引数组和一个关联数组（IP → 用途描述），遍历数组对每个 IP 执行 `ping -c 1`，输出格式为 `[OK/FAIL] IP (用途)`。

---

> 下一章：[02 - 引用展开与运算](../02-引用展开与运算/引用展开与运算.md)
