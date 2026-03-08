# Git 日常操作命令详解

> 本文是「Git 系统学习指南」系列第 02 篇，聚焦日常开发中最高频使用的 Git 命令。
> 掌握这些命令，足以应对 90% 的日常开发场景。

---

## 目录

- [1 初始化与克隆](#1-初始化与克隆)
  - [1.1 git init](#11-git-init)
  - [1.2 git clone](#12-git-clone)
- [2 文件跟踪与提交](#2-文件跟踪与提交)
  - [2.1 git add](#21-git-add)
  - [2.2 git commit](#22-git-commit)
  - [2.3 git commit 的注意事项](#23-git-commit-的注意事项)
- [3 状态与差异](#3-状态与差异)
  - [3.1 git status](#31-git-status)
  - [3.2 git diff](#32-git-diff)
- [4 查看历史](#4-查看历史)
  - [4.1 git log](#41-git-log)
  - [4.2 git show](#42-git-show)
- [5 暂存管理（git stash）](#5-暂存管理git-stash)
  - [5.1 基本用法](#51-基本用法)
  - [5.2 进阶用法](#52-进阶用法)
  - [5.3 典型场景](#53-典型场景)
- [6 撤销与恢复](#6-撤销与恢复)
  - [6.1 git restore（推荐，Git 2.23+）](#61-git-restore推荐git-223)
  - [6.2 git checkout（传统方式）](#62-git-checkout传统方式)
  - [6.3 注意事项](#63-注意事项)
- [7 文件操作](#7-文件操作)
  - [7.1 git mv](#71-git-mv)
  - [7.2 git rm](#72-git-rm)
- [8 小结](#8-小结)
- [9 动手练习](#9-动手练习)

## 1 初始化与克隆

### 1.1 git init

```bash
# 在当前目录初始化一个 Git 仓库
git init

# 默认分支是 master，可用以下命令重命名为 main
git branch -m main
```

裸仓库（bare repository）不包含工作区，通常用作远程中转仓库：

```bash
git init --bare my-project.git
```

> **提示**：裸仓库只存储 `.git` 目录内容，无法直接编辑文件，适合部署在服务器上作为共享仓库。

### 1.2 git clone

```bash
# 克隆远程仓库
git clone https://github.com/user/repo.git

# 浅克隆，只拉取最近 1 次提交，大幅减少下载量
git clone --depth 1 https://github.com/user/repo.git

# 指定分支克隆
git clone --branch develop https://github.com/user/repo.git

# 只克隆单分支（不下载其他分支的引用）
git clone --single-branch --branch main https://github.com/user/repo.git
```

> **实践建议**：CI/CD 环境中推荐 `--depth 1 --single-branch`，能显著加快构建速度。

## 2 文件跟踪与提交

### 2.1 git add

```bash
# 暂存指定文件
git add filename

# 暂存当前目录下所有变更（新增、修改），受 .gitignore 过滤
git add .

# 暂存所有变更，包括删除的文件
git add -A

# 交互式暂存：按 hunk 选择要暂存的代码块
git add -p
```

`git add -p` 是高手必备技能。它会逐个展示代码变更块（hunk），你可以选择：
- `y` — 暂存当前 hunk
- `n` — 跳过当前 hunk
- `s` — 将当前 hunk 拆分为更小的块
- `e` — 手动编辑 hunk

这让你能够把一个文件中的不同修改拆分到不同的 commit 中，保持提交粒度干净。

### 2.2 git commit

```bash
# 提交并附带消息
git commit -m "fix: resolve null pointer in user service"

# 跳过 git add，直接提交所有已跟踪文件的修改（不含新文件）
git commit -am "chore: update config defaults"

# 修改最近一次提交的 message 或内容（未 push 时使用）
git commit --amend -m "fix: correct typo in error message"

# 创建 fixup commit，配合 rebase --autosquash 自动合并
git commit --fixup=abc1234
# 后续执行：git rebase -i --autosquash abc1234~1
```

> **注意**：`--amend` 会改写提交历史。如果已经 push 到远程，不要使用，除非你清楚后果并愿意 force push。

### 2.3 git commit 的注意事项

**一次 commit 应该是一个逻辑完整的变更**。遵循以下原则：

1. **原子性** — 不要把不相关的修改混在一个 commit 里。修 bug 和加新功能应该是独立的 commit。
2. **描述"为什么"而不是"做了什么"** — 代码 diff 已经展示了"做了什么"，commit message 应该解释动机。
3. **遵循团队规范** — 推荐 Conventional Commits 格式：`type(scope): description`。

好的 commit message 示例：

```
fix(auth): prevent token refresh race condition

Two concurrent requests could both trigger a refresh, causing one to
use an invalidated token. Add mutex to serialize refresh calls.
```

## 3 状态与差异

### 3.1 git status

```bash
# 查看工作区状态总览
git status

# 简洁模式，适合快速浏览
git status -s

# 显示分支信息
git status -sb
```

简洁模式的标记含义：`M` = 已修改，`A` = 新增，`D` = 已删除，`??` = 未跟踪，`R` = 重命名。
左列表示暂存区状态，右列表示工作区状态。例如 `MM` 表示文件在暂存后又被修改。

### 3.2 git diff

```bash
# 工作区 vs 暂存区（查看还未 add 的改动）
git diff

# 暂存区 vs HEAD（查看已 add 但还未 commit 的改动）
git diff --cached
# 等价写法
git diff --staged

# 工作区 vs HEAD（查看所有未提交的改动）
git diff HEAD

# 两个分支之间的差异
git diff main..feature/login

# 仅显示统计信息（哪些文件改了、改了多少行）
git diff --stat

# 查看指定文件的差异
git diff -- src/main.go
```

> **技巧**：`git diff main...feature` （三个点）表示 feature 分支自从从 main 分出来之后的变更，排除 main 后续的新提交。这在 code review 时非常有用。

## 4 查看历史

### 4.1 git log

```bash
# 单行显示，简洁直观
git log --oneline

# 图形化显示分支历史（最常用的组合之一）
git log --oneline --graph --all --decorate

# 按作者过滤
git log --author="john"

# 按时间范围过滤
git log --since="2024-01-01" --until="2024-06-01"

# 跟踪文件的完整历史，包括重命名前的记录
git log --follow src/utils.go

# 显示每次提交的完整 diff
git log -p

# 搜索 commit message 中的关键词
git log --grep="hotfix"

# 搜索代码变更内容（pickaxe），找出哪个 commit 增加或删除了某段代码
git log -S "func HandleRequest"
```

> **实践建议**：建议在 `~/.gitconfig` 中配置别名：
> ```
> [alias]
>     lg = log --oneline --graph --all --decorate
> ```
> 之后只需 `git lg` 即可查看美观的分支图。

### 4.2 git show

```bash
# 查看某次提交的详细内容（元信息 + diff）
git show abc1234

# 查看某次提交中特定文件的内容
git show abc1234:src/main.go
```

## 5 暂存管理（git stash）

### 5.1 基本用法

```bash
# 暂存当前所有未提交的修改（工作区 + 暂存区）
git stash

# 带描述信息的暂存，方便后续识别
git stash save "WIP: refactoring auth module"

# 查看暂存列表
git stash list

# 恢复最近的暂存并从列表中删除
git stash pop

# 恢复指定暂存，保留在列表中（可多次 apply）
git stash apply stash@{2}

# 删除指定暂存
git stash drop stash@{1}

# 清空所有暂存
git stash clear
```

### 5.2 进阶用法

```bash
# 包含未跟踪文件（默认 stash 只保存已跟踪文件的修改）
git stash -u

# 只暂存未 add 的部分，保留暂存区内容不变
git stash --keep-index

# 查看某个暂存的详细 diff
git stash show -p stash@{0}

# 从暂存创建新分支（自动 apply 并 drop）
git stash branch feature/new-idea stash@{0}
```

### 5.3 典型场景

**场景一：临时切换分支修 bug**

正在开发新功能，突然需要切到 main 分支修复线上 bug：

```bash
git stash save "WIP: new feature halfway"
git checkout main
# ... 修复 bug，提交 ...
git checkout feature/my-work
git stash pop
```

**场景二：pull 前暂存本地修改**

本地有未提交的修改，直接 pull 可能冲突：

```bash
git stash
git pull --rebase
git stash pop
# 如果 pop 时出现冲突，手动解决后 git stash drop
```

## 6 撤销与恢复

### 6.1 git restore（推荐，Git 2.23+）

```bash
# 丢弃工作区的修改，恢复到暂存区的版本
git restore filename

# 取消暂存（从暂存区移回工作区），保留文件内容不变
git restore --staged filename

# 恢复文件到指定版本
git restore --source=HEAD~1 filename
```

### 6.2 git checkout（传统方式）

```bash
# 丢弃工作区修改（等同于 git restore filename）
git checkout -- filename
```

### 6.3 注意事项

| 操作 | 推荐命令 | 传统命令 |
|------|----------|----------|
| 丢弃工作区修改 | `git restore file` | `git checkout -- file` |
| 取消暂存 | `git restore --staged file` | `git reset HEAD file` |
| 切换分支 | `git switch branch` | `git checkout branch` |

Git 2.23 引入 `restore` 和 `switch`，将 `checkout` 的两种职责拆分开，语义更清晰。

> **警告**：丢弃工作区修改是**不可恢复的**。未暂存、未 commit 的内容一旦 restore 就永久丢失了。操作前请确认。

## 7 文件操作

### 7.1 git mv

```bash
# 重命名文件（Git 会自动记录重命名历史）
git mv old_name.go new_name.go

# macOS/Windows 下仅改大小写时需要 --force
git mv --force myFile.go MyFile.go
```

> **说明**：直接用 `mv` 命令重命名再 `git add` 也可以，Git 会自动检测重命名。但 `git mv` 是一步到位的写法。

### 7.2 git rm

```bash
# 删除文件并将删除操作暂存（下次 commit 时生效）
git rm filename

# 从版本控制中移除，但保留本地文件（常用于误提交的配置文件）
git rm --cached .env
```

> **典型场景**：不小心把 `.env` 提交到了仓库，用 `git rm --cached .env` 移除跟踪，然后把 `.env` 加入 `.gitignore`。

## 8 小结

以下是本章命令速查表：

| 场景 | 命令 | 说明 |
|------|------|------|
| 初始化仓库 | `git init` | 创建本地仓库 |
| 克隆仓库 | `git clone <url>` | 克隆远程仓库到本地 |
| 暂存文件 | `git add .` | 暂存所有变更 |
| 交互式暂存 | `git add -p` | 按代码块选择性暂存 |
| 提交 | `git commit -m "msg"` | 提交暂存区内容 |
| 修改提交 | `git commit --amend` | 修改最近一次提交 |
| 查看状态 | `git status -sb` | 简洁显示工作区状态 |
| 查看差异 | `git diff --cached` | 查看已暂存的改动 |
| 查看历史 | `git log --oneline --graph` | 图形化查看提交历史 |
| 搜索代码变更 | `git log -S "keyword"` | 查找引入/删除特定代码的提交 |
| 查看提交详情 | `git show <commit>` | 查看某次提交的完整内容 |
| 暂存修改 | `git stash -u` | 暂存所有修改（含未跟踪文件） |
| 恢复暂存 | `git stash pop` | 恢复最近的暂存内容 |
| 丢弃修改 | `git restore <file>` | 丢弃工作区修改 |
| 取消暂存 | `git restore --staged <file>` | 将文件从暂存区移回工作区 |
| 重命名文件 | `git mv old new` | 重命名并保留历史 |
| 移除跟踪 | `git rm --cached <file>` | 取消版本控制但保留本地文件 |

## 9 动手练习

1. **交互式暂存**：在一个文件中做两处不相关的修改，使用 `git add -p` 将它们分别提交为两个独立的 commit。

2. **stash 实战**：模拟"正在开发，突然要切分支修 bug"的场景——在 feature 分支做一些修改（不 commit），用 `git stash -u` 暂存，切到 main 修复一个问题，切回来 `git stash pop` 恢复。

3. **pickaxe 搜索**：在仓库中用 `git log -S "某个函数名"` 找出该函数是在哪个 commit 中首次引入的。再试试 `git log -G "err\s*!=\s*nil"` 搜索所有涉及错误检查的变更。

4. **restore 练习**：修改一个文件后先 `git add`，再继续修改。此时文件在暂存区和工作区的版本不同。分别用 `git diff` 和 `git diff --cached` 查看差异，然后用 `git restore --staged` 和 `git restore` 逐步撤销。

> 上一章：[01 - Git 原理与配置](../01-git原理与配置/Git原理与配置.md) · 下一章：[03 - Git 分支与合并详解](../03-分支与合并/Git分支与合并详解.md)
