# Git 进阶技巧

> 本文是「Git 系统学习指南」系列第七章（最终章），覆盖 Git 的高级功能，面向需要处理复杂协作场景、大型仓库或自动化流程的开发者。

---

## 目录

- [1 Git Hooks](#1-git-hooks)
  - [1.1 什么是 Hooks](#11-什么是-hooks)
  - [1.2 常用客户端 Hooks](#12-常用客户端-hooks)
  - [1.3 pre-commit 实战](#13-pre-commit-实战)
  - [1.4 commit-msg 实战](#14-commit-msg-实战)
  - [1.5 使用 Husky / Lefthook 管理 Hooks（团队协作推荐）](#15-使用-husky--lefthook-管理-hooks团队协作推荐)
- [2 Git Submodule](#2-git-submodule)
  - [2.1 什么是 Submodule](#21-什么是-submodule)
  - [2.2 基本操作](#22-基本操作)
  - [2.3 Submodule 的痛点](#23-submodule-的痛点)
- [3 Git Subtree（Submodule 的替代方案）](#3-git-subtreesubmodule-的替代方案)
  - [3.1 Subtree vs Submodule](#31-subtree-vs-submodule)
  - [3.2 基本操作](#32-基本操作)
- [4 Git Worktree](#4-git-worktree)
  - [4.1 使用场景](#41-使用场景)
  - [4.2 基本操作](#42-基本操作)
  - [4.3 实际工作流](#43-实际工作流)
- [5 Sparse Checkout](#5-sparse-checkout)
  - [5.1 使用场景](#51-使用场景)
  - [5.2 操作方式](#52-操作方式)
- [6 .gitattributes](#6-gitattributes)
  - [6.1 换行符处理](#61-换行符处理)
  - [6.2 自定义 Diff 驱动](#62-自定义-diff-驱动)
  - [6.3 Git LFS（Large File Storage）](#63-git-lfslarge-file-storage)
- [7 性能优化](#7-性能优化)
  - [7.1 浅克隆（Shallow Clone）](#71-浅克隆shallow-clone)
  - [7.2 部分克隆（Partial Clone）](#72-部分克隆partial-clone)
  - [7.3 git maintenance（Git 2.30+）](#73-git-maintenancegit-230)
  - [7.4 其他优化技巧](#74-其他优化技巧)
- [8 小结](#8-小结)
- [9 动手练习](#9-动手练习)

## 1 Git Hooks

### 1.1 什么是 Hooks

Git Hooks 是存放在 `.git/hooks/` 目录下的可执行脚本，在特定 Git 事件（commit、push、merge 等）发生时**自动触发**。Hooks 分为两类：

- **客户端 Hooks**：在本地操作（commit、merge、rebase）时触发，仅影响当前开发者。
- **服务端 Hooks**：在远程仓库接收推送时触发（pre-receive、update、post-receive），由仓库管理员配置。

启用一个 hook 只需将 `.git/hooks/` 下对应的 `.sample` 后缀去掉，并确保文件有可执行权限：

```bash
cp .git/hooks/pre-commit.sample .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

### 1.2 常用客户端 Hooks

| Hook | 触发时机 | 典型用途 |
|------|----------|----------|
| `pre-commit` | `git commit` 执行前 | 代码格式化、lint 检查 |
| `commit-msg` | 编辑 commit message 后 | 校验 commit 格式 |
| `pre-push` | `git push` 执行前 | 运行测试、阻止推送到 main |
| `prepare-commit-msg` | 编辑器打开前 | 自动填充模板（如 JIRA 编号） |
| `post-merge` | `git merge` 完成后 | 自动安装依赖 |

所有客户端 hooks 在脚本返回**非零退出码**时会中止对应操作，这是其拦截机制的核心。

### 1.3 pre-commit 实战

以下示例适用于 Go 项目，在每次 commit 前自动检查代码格式和静态分析：

```bash
#!/bin/sh
# .git/hooks/pre-commit

STAGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$')
if [ -z "$STAGED_GO_FILES" ]; then
    exit 0
fi

# gofmt 格式检查
UNFORMATTED=$(gofmt -l $STAGED_GO_FILES)
if [ -n "$UNFORMATTED" ]; then
    echo "以下文件未格式化（请运行 gofmt -w）："
    echo "$UNFORMATTED"
    exit 1
fi

# go vet 静态分析
go vet ./...
if [ $? -ne 0 ]; then
    echo "go vet 发现问题，请修复后再提交"
    exit 1
fi
```

### 1.4 commit-msg 实战

强制团队使用 Conventional Commits 格式：

```bash
#!/bin/sh
# .git/hooks/commit-msg

MSG=$(cat "$1")
PATTERN="^(feat|fix|docs|refactor|perf|test|chore|style|ci|build|revert)(\(.+\))?: .{1,72}"

if ! echo "$MSG" | grep -qE "$PATTERN"; then
    echo "Commit message 不符合 Conventional Commits 规范"
    echo "格式: <type>(<scope>): <subject>"
    echo "示例: feat(auth): 添加 JWT 刷新令牌机制"
    exit 1
fi
```

### 1.5 使用 Husky / Lefthook 管理 Hooks（团队协作推荐）

传统 `.git/hooks/` 目录**不会被 Git 跟踪**，新成员 clone 后没有任何 hooks。解决方案是将 hooks 配置化并纳入版本控制：

| 工具 | 生态 | 特点 |
|------|------|------|
| [Husky](https://github.com/typicode/husky) | Node.js | 最流行，配合 lint-staged 使用 |
| [Lefthook](https://github.com/evilmartians/lefthook) | Go（单二进制） | 无依赖，速度快，适合 Go 项目 |
| [pre-commit](https://pre-commit.com/) | Python | 插件生态丰富，支持多语言 |

以 Lefthook 为例（`go install github.com/evilmartians/lefthook@latest` 安装）：

```yaml
# lefthook.yml — 纳入版本控制，团队统一
pre-commit:
  parallel: true
  commands:
    gofmt:
      glob: "*.go"
      run: gofmt -l {staged_files} && test -z "$(gofmt -l {staged_files})"
    govet:
      run: go vet ./...

commit-msg:
  commands:
    conventional:
      run: 'echo "$1" | grep -qE "^(feat|fix|docs|refactor|perf|test|chore)(\(.+\))?: .+"'
```

---

## 2 Git Submodule

### 2.1 什么是 Submodule

Git Submodule 允许在一个 Git 仓库中**嵌入另一个独立的 Git 仓库**。主仓库只记录子仓库的 URL 和一个固定的 commit hash，不存储子仓库的实际文件内容。

典型场景：共享库、协议定义（proto）、前后端分仓。

### 2.2 基本操作

```bash
# 添加子模块（会生成 .gitmodules 文件）
git submodule add git@github.com:team/shared-lib.git libs/shared

# 克隆含子模块的仓库（推荐 --recursive 一步到位）
git clone --recursive git@github.com:team/main-repo.git

# 如果已经 clone 了，补充初始化子模块
git submodule init
git submodule update

# 更新子模块到远程最新 commit
git submodule update --remote
```

删除子模块需要三步：

```bash
git submodule deinit libs/shared       # 取消注册
git rm libs/shared                     # 从工作区和索引中移除
rm -rf .git/modules/libs/shared        # 清理缓存
```

### 2.3 Submodule 的痛点

- **操作繁琐**：每次 clone 都要带 `--recursive`，漏了就报错。
- **容易遗忘更新**：子模块更新后，主仓库必须手动提交新的 commit hash。
- **CI/CD 额外配置**：大多数 CI 系统不会自动初始化子模块。
- **嵌套 submodule**：子模块中再有子模块，复杂度指数级增长。

---

## 3 Git Subtree（Submodule 的替代方案）

### 3.1 Subtree vs Submodule

| 维度 | Submodule | Subtree |
|------|-----------|---------|
| 原理 | 引用外部仓库的 commit | 将外部仓库代码**合并**到主仓库 |
| 使用者体验 | 需要额外命令（init / update） | 无感，普通 clone 即可 |
| 更新方式 | `git submodule update --remote` | `git subtree pull` |
| 历史记录 | 子仓库保持独立历史 | 合并到主仓库历史中 |
| 回推修改 | 在子仓库目录内直接 push | `git subtree push` |
| 推荐场景 | 子仓库需独立开发和版本控制 | 只需引入代码，不需频繁同步 |

**选择建议**：子仓库有独立团队和发版周期 → submodule；只是引入代码且会直接修改 → subtree。

### 3.2 基本操作

```bash
# 添加远程引用（方便后续操作）
git remote add shared-lib git@github.com:team/shared-lib.git

# 添加 subtree（--squash 将子仓库历史压缩为一个 commit）
git subtree add --prefix=libs/shared shared-lib main --squash

# 拉取子仓库的最新更新
git subtree pull --prefix=libs/shared shared-lib main --squash

# 将主仓库中对子目录的修改推送回子仓库
git subtree push --prefix=libs/shared shared-lib main
```

> `--squash` 可以避免子仓库的完整历史涌入主仓库，保持 log 整洁。

---

## 4 Git Worktree

### 4.1 使用场景

正在 feature 分支开发到一半，突然需要修一个紧急 bug。传统做法是 `git stash` → 切分支 → 修复 → 切回来 → `git stash pop`，繁琐且易出错。

**Git Worktree** 允许从同一个仓库创建多个独立工作目录，每个目录检出不同分支，共享同一个 `.git` 对象数据库。无需 stash，多个分支**并行工作**。

### 4.2 基本操作

```bash
# 创建新的 worktree（检出已有分支）
git worktree add ../hotfix hotfix/critical-bug

# 创建新的 worktree（同时创建新分支）
git worktree add -b hotfix/login-fix ../hotfix-login main

# 列出所有 worktree
git worktree list

# 完成工作后，删除 worktree
git worktree remove ../hotfix

# 清理已被手动删除的 worktree 引用
git worktree prune
```

### 4.3 实际工作流

```
~/project/          ← 主工作区，日常 feature 开发
~/project-hotfix/   ← 临时工作区，处理紧急 bug
~/project-review/   ← 临时工作区，review 同事的 PR
```

在主目录继续 feature 开发的同时，打开另一个终端进入 `~/project-hotfix/` 修复 bug，两边互不干扰。修复完成后 `git worktree remove` 清理即可。

> **注意**：同一个分支不能同时在多个 worktree 中检出。

---

## 5 Sparse Checkout

### 5.1 使用场景

在大型 monorepo 中，开发者通常只关心自己负责的服务。Sparse Checkout 允许只检出仓库的**部分目录**，大幅节省磁盘空间和克隆时间。

### 5.2 操作方式

```bash
# 结合 partial clone 和 sparse checkout（推荐组合）
git clone --filter=blob:none --sparse git@github.com:org/monorepo.git
cd monorepo

# 选择需要的目录（cone 模式，Git 2.25+）
git sparse-checkout set services/user-api libs/common

# 追加更多目录
git sparse-checkout add services/order-api

# 查看当前配置
git sparse-checkout list

# 恢复完整检出
git sparse-checkout disable
```

---

## 6 .gitattributes

`.gitattributes` 文件用于为路径设置属性，控制 Git 对特定文件的处理方式。应**纳入版本控制**，确保团队统一。

### 6.1 换行符处理

跨平台团队最常见的问题是换行符不一致（Windows 用 CRLF，Unix/macOS 用 LF）：

```
# .gitattributes
* text=auto

*.go text eol=lf
*.sh text eol=lf
*.yml text eol=lf
*.md text eol=lf

*.bat text eol=crlf
*.cmd text eol=crlf

*.png binary
*.jpg binary
*.woff2 binary
```

### 6.2 自定义 Diff 驱动

```
# .gitattributes — 对 proto 文件使用自定义 diff
*.proto diff=proto
```

```bash
# 配置 diff 驱动，在 hunk header 中显示 message/service 名称
git config diff.proto.xfuncname "^(message|service|rpc|enum) .*"
```

### 6.3 Git LFS（Large File Storage）

Git 不擅长处理大文件，每次修改都会在历史中保留完整副本。Git LFS 将大文件替换为轻量级指针，实际内容存储在专用服务器上。

```bash
git lfs install
git lfs track "*.psd"
git lfs track "*.zip"
```

以上命令会自动在 `.gitattributes` 中添加：

```
*.psd filter=lfs diff=lfs merge=lfs -text
*.zip filter=lfs diff=lfs merge=lfs -text
```

---

## 7 性能优化

当仓库历史达到数万次提交、或包含大量大文件时，常规操作会变得缓慢。

### 7.1 浅克隆（Shallow Clone）

```bash
# 只拉取最近 1 次提交（CI/CD 常用）
git clone --depth 1 <url>

# 后续需要完整历史时
git fetch --unshallow
```

**局限**：浅克隆无法执行依赖完整历史的操作（如 `git bisect`）。

### 7.2 部分克隆（Partial Clone）

Git 2.22+ 引入，按需下载对象：

```bash
# Blobless clone：不下载文件内容，访问时按需获取（适合日常开发）
git clone --filter=blob:none <url>

# Treeless clone：连目录树也不下载（适合 CI/CD）
git clone --filter=tree:0 <url>
```

与 sparse-checkout 组合使用效果最佳（见第 5 节）。

### 7.3 git maintenance（Git 2.30+）

后台维护机制，自动执行优化任务：

```bash
git maintenance start    # 启用（注册到系统定时任务）
git maintenance run      # 手动执行一次
git maintenance stop     # 停用
```

| 任务 | 作用 | 频率 |
|------|------|------|
| `commit-graph` | 构建 commit 图缓存，加速 `log`、`merge-base` | 每小时 |
| `prefetch` | 后台预拉取远程分支数据 | 每小时 |
| `gc` | 垃圾回收，压缩对象数据库 | 每天 |
| `loose-objects` | 将松散对象打包 | 每天 |
| `incremental-repack` | 增量重新打包，优化磁盘占用 | 每天 |

### 7.4 其他优化技巧

```bash
git commit-graph write --reachable    # 手动更新 commit-graph
git count-objects -vH                 # 查看仓库对象统计
git gc --aggressive --prune=now       # 压缩仓库
```

---

## 8 小结

本章介绍了 Git 的进阶功能，覆盖了从自动化到性能优化的完整链路：

| 场景 | 工具 | 核心价值 |
|------|------|----------|
| 自动化流程 | **Git Hooks** | commit / push 时自动执行检查 |
| 多仓库管理 | **Submodule / Subtree** | 引用和管理外部代码 |
| 并行开发 | **Worktree** | 多分支同时工作，无需 stash |
| 大仓库优化 | **Sparse Checkout** | 只检出需要的目录 |
| 团队配置统一 | **.gitattributes** | 统一换行符、diff 策略、LFS |
| 性能优化 | **浅克隆 / 部分克隆 / maintenance** | 加速克隆和日常操作 |

---

## 9 动手练习

1. **编写 pre-commit hook**：为一个 Go 项目编写 `.git/hooks/pre-commit` 脚本，要求在 commit 前自动运行 `gofmt -l` 和 `go vet`，任一失败则阻止提交。测试方法：故意提交一个未格式化的文件，确认 hook 阻止了提交。

2. **Worktree 并行开发**：在当前仓库创建一个 worktree 检出另一个分支（`git worktree add ../my-hotfix hotfix-branch`），在两个目录中分别做修改和提交，体会无需 stash 的并行开发。完成后用 `git worktree remove` 清理。

3. **Sparse Checkout 实践**：找一个大型开源 monorepo（如 `github.com/golang/go`），用 `git clone --filter=blob:none --sparse` 克隆，然后用 `git sparse-checkout set src/cmd/go` 只检出部分目录，对比完整克隆的体积差异。

4. **Submodule 生命周期**：创建两个仓库 A 和 B，将 B 作为 A 的 submodule 添加，修改 B 并更新 A 中的 submodule 引用，最后从 A 中删除 submodule。完整走一遍 add → update → remove 流程。

> 上一章：[06 - Git 工作流与提交规范](../06-工作流与规范/Git工作流与提交规范.md)

---

> **至此，「Git 系统学习指南」全部 7 个章节完结。** 建议按顺序学习：
>
> 1. [Git 原理与配置](../01-git原理与配置/Git原理与配置.md)
> 2. [Git 日常操作命令详解](../02-日常操作命令/Git日常操作命令详解.md)
> 3. [Git 分支与合并详解](../03-分支与合并/Git分支与合并详解.md)
> 4. [Git 远程协作详解](../04-远程协作/Git远程协作详解.md)
> 5. [Git 历史操控详解](../05-历史操控/Git历史操控详解.md)
> 6. [Git 工作流与提交规范](../06-工作流与规范/Git工作流与提交规范.md)
> 7. [Git 进阶技巧](../07-进阶技巧/Git进阶技巧.md)（本章）
>
> 每章的命令都建议动手实践，在真实仓库中反复练习，才能真正内化为肌肉记忆。
