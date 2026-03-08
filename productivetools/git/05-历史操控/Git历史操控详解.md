# Git 历史操控详解

> 本文是「Git 系统学习指南」系列的第 05 章，聚焦于 Git 历史的查看、搜索、修改和恢复。掌握这些能力意味着你可以精确操控项目的时间线——从高效定位 bug 到安全地回滚错误提交，再到跨分支迁移特定变更。

## 目录

- [1 git log 进阶](#1-git-log-进阶)
  - [1.1 格式化输出](#11-格式化输出)
  - [1.2 过滤与搜索](#12-过滤与搜索)
  - [1.3 实用组合示例](#13-实用组合示例)
- [2 git blame](#2-git-blame)
  - [2.1 基本用法](#21-基本用法)
  - [2.2 忽略格式化提交](#22-忽略格式化提交)
- [3 git bisect](#3-git-bisect)
  - [3.1 原理](#31-原理)
  - [3.2 手动 bisect](#32-手动-bisect)
  - [3.3 自动 bisect](#33-自动-bisect)
- [4 git reflog](#4-git-reflog)
  - [4.1 什么是 reflog](#41-什么是-reflog)
  - [4.2 用 reflog 恢复误操作](#42-用-reflog-恢复误操作)
  - [4.3 恢复误删的分支](#43-恢复误删的分支)
- [5 git reset 三种模式详解](#5-git-reset-三种模式详解)
  - [5.1 --soft](#51---soft)
  - [5.2 --mixed（默认）](#52---mixed默认)
  - [5.3 --hard](#53---hard)
  - [5.4 三种模式对比](#54-三种模式对比)
  - [5.5 HEAD 引用语法](#55-head-引用语法)
- [6 git revert](#6-git-revert)
  - [6.1 revert vs reset](#61-revert-vs-reset)
  - [6.2 基本用法](#62-基本用法)
  - [6.3 使用建议](#63-使用建议)
- [7 git cherry-pick](#7-git-cherry-pick)
  - [7.1 基本用法](#71-基本用法)
  - [7.2 转移多个 commit](#72-转移多个-commit)
  - [7.3 处理冲突](#73-处理冲突)
  - [7.4 跨仓库 cherry-pick](#74-跨仓库-cherry-pick)
  - [7.5 常用选项](#75-常用选项)
- [8 危险操作的安全网](#8-危险操作的安全网)
  - [8.1 ORIG_HEAD](#81-orig_head)
  - [8.2 备份分支](#82-备份分支)
  - [8.3 恢复操作的标准流程](#83-恢复操作的标准流程)
- [9 小结](#9-小结)
- [10 动手练习](#10-动手练习)

## 1 git log 进阶

### 1.1 格式化输出

```bash
git log --oneline                      # 精简单行，只展示短 hash 和 message
git log --format="%h %an %ar %s"       # 自定义格式
```

常用占位符：

| 占位符 | 含义 |
|--------|------|
| `%H` / `%h` | 完整 / 短 commit hash |
| `%an` | 作者名称 |
| `%ae` | 作者邮箱 |
| `%ar` | 相对时间（如 "2 days ago"） |
| `%ad` | 绝对时间（受 `--date=` 控制） |
| `%s` | commit message 首行 |

**图形化全分支历史**（强烈推荐配成 alias）：

```bash
git log --graph --all --oneline --decorate

# 配置 alias 后一键使用
git config --global alias.lg "log --graph --all --oneline --decorate"
git lg
```

### 1.2 过滤与搜索

```bash
# 按作者过滤
git log --author="john"

# 按时间范围
git log --since="2024-01-01" --until="2024-06-01"

# 搜索 commit message
git log --grep="fix deadlock"

# pickaxe：搜索代码内容的新增或删除
# 当某段代码"消失"了，用 -S 可以找到是哪个 commit 删掉的
git log -S "func ParseConfig"

# 正则搜索代码变更（比 -S 更灵活）
git log -G "err\s*!=\s*nil"

# 限定文件范围
git log -- src/parser/lexer.go

# 跟踪文件重命名前的历史
git log --follow old_name.go
```

**`-S` 与 `-G` 的区别**：`-S` 匹配的是「变更前后该字符串出现次数发生了变化」的 commit；`-G` 匹配的是「diff 中包含匹配该正则的行」的 commit。简单查找用 `-S`，需要模糊匹配用 `-G`。

### 1.3 实用组合示例

```bash
# 查看某人在过去一个月内对指定目录的提交
git log --author="john" --since="1 month ago" --oneline -- pkg/handler/

# 查看涉及某函数新增或删除的提交，并显示 diff
git log -S "func NewServer" -p --oneline

# 查找包含 "fix" 关键字的合并提交
git log --grep="fix" --merges --oneline

# 查看两个 tag 之间的变更摘要
git log v1.2.0..v1.3.0 --oneline --stat
```

## 2 git blame

### 2.1 基本用法

```bash
git blame filename                # 逐行追溯文件的最后修改者和 commit
git blame -L 10,20 filename      # 只看第 10-20 行
git blame -L :funcName filename   # 只看某个函数范围
```

### 2.2 忽略格式化提交

大规模代码格式化（如 `gofmt`、`prettier`、统一代码风格）之后，`blame` 会显示格式化的 commit 而非真正的逻辑变更作者。解决方案：

```bash
# 忽略单个格式化提交
git blame --ignore-rev <format-commit-hash> filename

# 推荐：维护一个忽略列表文件
echo "<format-commit-hash>" >> .git-blame-ignore-revs
git blame --ignore-revs-file .git-blame-ignore-revs filename

# 设为默认行为，团队共享
git config blame.ignoreRevsFile .git-blame-ignore-revs
```

> **团队实践**：将 `.git-blame-ignore-revs` 提交到仓库中，所有成员配置 `blame.ignoreRevsFile` 后，GitHub 的 blame 视图也会自动识别该文件。

## 3 git bisect

### 3.1 原理

`git bisect` 使用**二分查找**在「好」和「坏」的 commit 之间定位引入 bug 的那个 commit。

- 时间复杂度 O(log n)：1000 个 commit 只需约 10 步
- 无需人工逐个排查，效率远超线性搜索

### 3.2 手动 bisect

```bash
# 1. 启动 bisect
git bisect start

# 2. 标记当前版本有 bug
git bisect bad

# 3. 标记一个已知没有 bug 的版本
git bisect good v1.0.0

# 4. Git 自动 checkout 中间的 commit，你测试后标记：
git bisect good   # 该版本没有 bug
# 或
git bisect bad    # 该版本有 bug

# 5. 重复步骤 4，直到 Git 输出：
#    "abc1234 is the first bad commit"

# 6. 结束 bisect，回到原来的分支
git bisect reset
```

### 3.3 自动 bisect

如果你有一个可以自动检测 bug 的脚本或测试用例，可以让 Git 全自动完成二分查找：

```bash
git bisect start HEAD v1.0.0
git bisect run go test ./pkg/parser/...
```

约定：脚本返回 **0 = good**，**非 0 = bad**。Git 会自动重复「checkout → 执行脚本 → 判断好坏」的过程，直到找到第一个引入 bug 的 commit。

> **注意**：脚本返回 125 表示该 commit 无法测试（如编译失败），Git 会跳过它继续查找。

## 4 git reflog

### 4.1 什么是 reflog

reflog（reference log）记录了 HEAD 和分支引用的**每一次移动**（commit、reset、checkout、rebase、merge 等）。

- 即使 commit 已经「丢失」（不在任何分支上），reflog 仍保留记录
- 默认保留 **90 天**，是你执行危险操作后的**最后一道安全网**
- reflog 是**本地**的，不会推送到远程

### 4.2 用 reflog 恢复误操作

```bash
# 查看 reflog
git reflog

# 输出示例：
# a1b2c3d HEAD@{0}: reset: moving to HEAD~3
# e4f5g6h HEAD@{1}: commit: feat: add config parser
# i7j8k9l HEAD@{2}: commit: fix: resolve nil pointer
# ...

# 找到误操作前的 HEAD 位置，恢复到那个状态
git reset --hard HEAD@{1}
```

### 4.3 恢复误删的分支

```bash
git branch -D feature-auth              # 不小心删了
git reflog | grep "feature-auth"        # 找到该分支最后一个 commit
git branch feature-auth <commit-hash>   # 重建分支
```

## 5 git reset 三种模式详解

`git reset` 通过移动 HEAD 指针来「回退」历史，但它对暂存区和工作区的影响取决于所使用的模式。

### 5.1 --soft

```bash
git reset --soft HEAD~1
```

- **只移动 HEAD 指针**，暂存区和工作区都不变
- 被撤销的 commit 的修改保留在暂存区中，随时可以重新 commit
- **用途**：想修改最近的 commit message 或将多个 commit 合并为一个

### 5.2 --mixed（默认）

```bash
git reset HEAD~1        # 等同于 git reset --mixed HEAD~1
```

- **移动 HEAD 指针 + 重置暂存区**，工作区不变
- 被撤销的修改从暂存区退回到工作区（变为 unstaged 状态）
- **用途**：撤销 commit 和 `git add`，重新选择要暂存的文件

### 5.3 --hard

```bash
git reset --hard HEAD~1
```

- **移动 HEAD 指针 + 重置暂存区 + 重置工作区**
- 被撤销的所有修改彻底丢弃
- **用途**：完全回退到某个历史版本，干净利落

> ⚠️ **危险操作**：`--hard` 会永久丢失未提交的工作区修改（已提交的可以通过 reflog 找回）。执行前务必确认没有需要保留的未提交代码。

### 5.4 三种模式对比

| 模式 | HEAD | 暂存区 | 工作区 | 典型用途 |
|------|------|--------|--------|----------|
| `--soft` | ✅ 移动 | ❌ 不变 | ❌ 不变 | 重新组织 commit |
| `--mixed` | ✅ 移动 | ✅ 重置 | ❌ 不变 | 撤销 add + commit |
| `--hard` | ✅ 移动 | ✅ 重置 | ✅ 重置 | 完全回退（⚠️ 危险） |

### 5.5 HEAD 引用语法

```
HEAD       当前版本
HEAD^      上一个版本（等同于 HEAD~1）
HEAD^^     上两个版本（等同于 HEAD~2）
HEAD~n     上 n 个版本
HEAD^2     第二个父提交（仅对合并提交有意义）
```

> **`^` 与 `~` 的区别**：`~` 沿第一父提交向上回溯 n 代；`^` 用于选择第几个父提交。对于普通 commit 只有一个父提交，两者没有区别；对于 merge commit，`HEAD^1` 是被合并到的分支，`HEAD^2` 是被合并进来的分支。

## 6 git revert

### 6.1 revert vs reset

| 对比项 | `git reset` | `git revert` |
|--------|-------------|--------------|
| 原理 | 移动 HEAD 指针，改写历史 | 生成新的反向提交，保留历史 |
| 历史影响 | 目标 commit 之后的历史被丢弃 | 完整保留所有历史 |
| 适用场景 | 未推送的本地修改 | 已推送到远程的公共分支 |
| 安全性 | 已推送后使用会导致团队冲突 | 安全，不影响他人 |

**核心原则**：已经 push 到远程公共分支的 commit，**永远用 revert，不要用 reset**。

### 6.2 基本用法

```bash
# 撤销某个 commit（生成一个新的反向提交）
git revert <commit-hash>

# 撤销多个连续的 commit（左开右闭，不含 older-hash）
git revert <older-hash>..<newer-hash>

# 只修改工作区，不自动生成 commit（适合批量 revert 后统一提交）
git revert -n <commit-hash>

# 撤销合并提交（需要通过 -m 指定保留哪个 parent）
# -m 1 表示保留合并时的目标分支（通常是 main）
git revert -m 1 <merge-commit-hash>
```

### 6.3 使用建议

- 已推送到远程的公共分支，**必须**用 revert 而非 reset
- revert 产生的 commit 可以再被 revert（即 "revert the revert"），用于恢复之前被撤销的功能
- 使用 `-n` 选项批量 revert 多个 commit 时，最后统一提交可以保持历史整洁

## 7 git cherry-pick

`cherry-pick` 可以将任意分支上的指定 commit「摘取」到当前分支，不需要合并整个分支。

### 7.1 基本用法

```bash
# 将指定 commit 应用到当前分支（自动生成新 commit）
git cherry-pick <commit-hash>

# 只应用变更到工作区和暂存区，不自动提交
git cherry-pick -n <commit-hash>
```

### 7.2 转移多个 commit

```bash
# 转移多个不连续的 commit
git cherry-pick <hash1> <hash2> <hash3>

# 转移连续的 commit（左开右闭，不包含 A）
git cherry-pick A..B

# 转移连续的 commit（包含 A）
git cherry-pick A^..B
```

### 7.3 处理冲突

cherry-pick 和 merge 一样可能产生冲突：

```bash
git add .
git cherry-pick --continue   # 解决冲突后继续
git cherry-pick --abort       # 放弃，回到操作前的状态
git cherry-pick --quit        # 退出但保留当前工作区状态
```

### 7.4 跨仓库 cherry-pick

```bash
git remote add other-repo https://github.com/org/other-repo.git
git fetch other-repo
git log other-repo/main --oneline
git cherry-pick <commit-hash>
```

### 7.5 常用选项

| 选项 | 作用 |
|------|------|
| `-x` | 在 commit message 末尾追加 `(cherry picked from commit ...)` 来源信息 |
| `-s` | 追加 `Signed-off-by` 签名行 |
| `-n` | 不自动提交，只应用变更到暂存区 |
| `-m <parent>` | 处理合并提交时指定保留哪个 parent |

> **推荐**：团队协作时始终加 `-x`，方便追溯 commit 来源。

## 8 危险操作的安全网

Git 提供了多层保护机制，帮助你在执行危险操作后安全回退。

### 8.1 ORIG_HEAD

Git 在执行 `merge`、`rebase`、`reset` 等可能改变大量历史的操作前，会自动把当前 HEAD 保存到 `ORIG_HEAD`：

```bash
# 撤销刚才的 merge / rebase / reset
git reset --hard ORIG_HEAD
```

`ORIG_HEAD` 只保存最近一次危险操作前的状态，如果你连续执行了多次操作，需要借助 reflog。

### 8.2 备份分支

在执行 rebase 或 reset 等不可逆操作前，养成手动创建备份分支的习惯：

```bash
# 创建备份
git branch backup-before-rebase

# 执行危险操作
git rebase -i HEAD~5

# 如果出了问题，可以恢复
git reset --hard backup-before-rebase
```

### 8.3 恢复操作的标准流程

当操作出错需要恢复时，按以下顺序排查：

1. **查 reflog**：`git reflog` 找到目标状态的 commit hash
2. **确认目标**：`git show <hash>` 确认这就是你要恢复的状态
3. **执行恢复**：
   - 回退当前分支：`git reset --hard <hash>`
   - 恢复为新分支：`git branch recovered <hash>`

## 9 小结

| 命令 | 核心能力 | 关键记忆点 |
|------|----------|------------|
| `git log` | 历史搜索 | `-S` 搜代码、`--grep` 搜 message、`--graph` 看拓扑 |
| `git blame` | 逐行追溯 | `--ignore-revs-file` 跳过格式化提交 |
| `git bisect` | 二分定位 bug | `bisect run` 配合测试脚本全自动查找 |
| `git reflog` | 恢复丢失的 commit | 你的最后一道安全网，90 天有效 |
| `git reset` | 回退历史 | `--soft` 保留暂存、`--mixed` 保留工作区、`--hard` 全部清除 |
| `git revert` | 安全撤销 | 已推送到远程的 commit 只能用 revert |
| `git cherry-pick` | 摘取 commit | `-x` 追加来源信息，冲突时 `--continue` / `--abort` |

**两条黄金法则**：

1. 已推送到远程公共分支的历史，**不要用 reset，要用 revert**
2. 执行任何危险操作前，先 `git reflog` 确认有路可退，或手动创建备份分支

## 10 动手练习

1. **reflog 救命演练**：在一个测试仓库中做 3 个 commit，然后执行 `git reset --hard HEAD~3` "丢掉"它们。接着用 `git reflog` 找到丢失的 commit，用 `git reset --hard <hash>` 恢复。

2. **bisect 实战**：创建一个包含 10+ 个 commit 的仓库，在某个中间 commit 中故意引入一个 bug（比如让某个测试失败）。使用 `git bisect start` + `git bisect run go test ./...`（或其他测试命令）自动定位到引入 bug 的 commit。

3. **reset 三模式对比**：做一个 commit 后，分别尝试 `--soft`、`--mixed`、`--hard` reset，每次都用 `git status` 和 `git diff` 观察暂存区和工作区的状态变化，切身感受三者的区别。

4. **cherry-pick 跨分支迁移**：创建两个分支 A 和 B，在 B 上做 3 个 commit，然后切到 A，用 `git cherry-pick` 只选取其中 1 个 commit 过来。用 `git log --oneline --all --graph` 查看结果。

> 上一章：[04 - Git 远程协作详解](../04-远程协作/Git远程协作详解.md) · 下一章：[06 - Git 工作流与提交规范](../06-工作流与规范/Git工作流与提交规范.md)
