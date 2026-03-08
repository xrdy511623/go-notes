# Git 工作流与提交规范

> 本文是「Git 系统学习指南」系列第六章，聚焦团队协作中的工作流选型与提交规范。
> 代码管理不仅是技术问题，更是团队协作问题。选择合适的工作流和提交规范，能显著降低协作摩擦。

---

## 目录

- [1 主流工作流对比](#1-主流工作流对比)
  - [1.1 Git Flow](#11-git-flow)
  - [1.2 GitHub Flow](#12-github-flow)
  - [1.3 Trunk-based Development](#13-trunk-based-development)
  - [1.4 选型建议](#14-选型建议)
- [2 Conventional Commits 提交规范](#2-conventional-commits-提交规范)
  - [2.1 格式](#21-格式)
  - [2.2 type 类型](#22-type-类型)
  - [2.3 scope（可选）](#23-scope可选)
  - [2.4 subject 规则](#24-subject-规则)
  - [2.5 body（可选）](#25-body可选)
  - [2.6 footer（可选）](#26-footer可选)
  - [2.7 完整示例](#27-完整示例)
- [3 工具链](#3-工具链)
  - [3.1 commitizen](#31-commitizen)
  - [3.2 commitlint](#32-commitlint)
  - [3.3 自动生成 CHANGELOG](#33-自动生成-changelog)
  - [3.4 Go 项目中的实践](#34-go-项目中的实践)
- [4 Pull Request / Merge Request 最佳实践](#4-pull-request--merge-request-最佳实践)
  - [4.1 PR 粒度](#41-pr-粒度)
  - [4.2 PR 描述模板](#42-pr-描述模板)
  - [4.3 Code Review 要点](#43-code-review-要点)
  - [4.4 CI 门禁](#44-ci-门禁)
  - [4.5 GitHub 合并策略选型](#45-github-合并策略选型)
- [5 分支命名规范](#5-分支命名规范)
  - [5.1 推荐格式](#51-推荐格式)
  - [5.2 示例](#52-示例)
  - [5.3 分支保护规则](#53-分支保护规则)
- [6 小结](#6-小结)
- [7 动手练习](#7-动手练习)

## 1 主流工作流对比

### 1.1 Git Flow

Git Flow 由 Vincent Driessen 在 2010 年提出，是最经典的分支管理模型。它定义了五种分支角色：

```
时间线 ──────────────────────────────────────────────────────►

main        ●───────────────────●─────────────●──────────► (生产)
             \                 /↑            /↑
hotfix        \           ----  |           / |
               \         /      |          /  |
develop         ●───●───●───●───●───●───●─●───●──────────► (开发主线)
                 \ /       / \         / \
feature           ●───●───●   ●───●──●   \
                                           \
release                                     ●───●─────────► (发布分支)

分支说明：
┌──────────────┬────────────────────────────────────────────┐
│ main         │ 生产分支，只接受 merge，每个 merge 打 tag   │
│ develop      │ 开发主线，集成所有已完成的功能              │
│ feature/*    │ 从 develop 创建，完成后合回 develop         │
│ release/*    │ 从 develop 创建，合回 main + develop        │
│ hotfix/*     │ 从 main 创建，合回 main + develop           │
└──────────────┴────────────────────────────────────────────┘
```

**优点：**

- 分支职责清晰，适合有明确版本发布周期的项目
- 生产环境和开发环境完全隔离，线上问题可通过 hotfix 快速修复
- release 分支提供了发布前的缓冲区，适合做最后的集成测试和版本号修改

**缺点：**

- 分支多、流程长，合并频繁，维护成本高
- 不适合持续部署（CD）场景——每次发布都要走 release 分支流程
- 长期存在的 feature 分支容易产生大量合并冲突

---

### 1.2 GitHub Flow

GitHub Flow 是 GitHub 团队在实践中总结出的极简工作流，核心只有一条长期分支 `main`：

```
简化流程：

1. main 分支始终保持可部署状态
2. 新功能从 main 创建 feature 分支
3. 在 feature 分支上持续提交，推送到远程
4. 开发完成后提 Pull Request
5. 团队成员进行 Code Review
6. Review 通过、CI 绿灯后合并到 main
7. 合并后立即触发自动部署
```

**优点：**

- 流程简单直接，上手成本低
- 天然适合持续部署，合并即发布
- 强制 Code Review，提高代码质量

**缺点：**

- 没有 develop 缓冲层，main 的每次合并都直接面向生产
- 对 CI/CD 和自动化测试要求高——如果测试覆盖不足，有引入缺陷的风险
- 不适合需要同时维护多个版本的项目

---

### 1.3 Trunk-based Development

Trunk-based Development（主干开发）是 Google、Meta 等大厂广泛采用的工作流。

**核心理念：**

- 所有开发者直接在 main（trunk）上提交，或使用生命周期极短的分支（< 1 天）
- 通过 Feature Flag（功能开关）控制未完成功能的可见性
- 依赖强大的 CI 基础设施保证主干始终可用
- 小批量、高频率提交，每次提交都是可发布的增量

**适用场景：**

- 高频发布（日均多次部署）
- 团队成员经验丰富，具备良好的工程纪律
- CI/CD 基础设施成熟，自动化测试覆盖率高

---

### 1.4 选型建议

| 维度 | Git Flow | GitHub Flow | Trunk-based |
|------|----------|-------------|-------------|
| 团队规模 | 中大型 | 小中型 | 任意 |
| 发布频率 | 按版本 | 按需 | 持续 |
| CI/CD 要求 | 低 | 中 | 高 |
| 分支复杂度 | 高 | 低 | 极低 |
| 学习曲线 | 陡峭 | 平缓 | 中等（需理解 Feature Flag） |
| 适合项目 | 客户端 / 有版本号的产品 | SaaS / Web 服务 | 成熟的微服务 |

> **实践建议：** 大多数 Web 后端团队推荐从 GitHub Flow 起步。当团队工程成熟度提升后，可逐步过渡到 Trunk-based Development。Git Flow 更适合移动端 App 等有明确版本发布节奏的项目。

---

## 2 Conventional Commits 提交规范

Conventional Commits 是一套建立在 commit message 之上的轻量级约定，为自动化工具（CHANGELOG 生成、语义化版本号推算）提供结构化信息。

### 2.1 格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

其中 `type` 和 `subject` 是必填项，`scope`、`body`、`footer` 为可选。

### 2.2 type 类型

| type | 说明 | 示例 |
|------|------|------|
| feat | 新功能 | `feat(auth): add OAuth2 login` |
| fix | 修复 bug | `fix(api): handle nil pointer in GetUser` |
| docs | 文档变更 | `docs: update API README` |
| refactor | 重构（非新功能、非修 bug） | `refactor(db): simplify connection pool logic` |
| perf | 性能优化 | `perf(query): add index on user_id` |
| test | 测试相关 | `test(auth): add login edge case` |
| chore | 构建 / 工具 / 依赖变更 | `chore: upgrade Go to 1.24` |
| style | 代码风格（不影响逻辑） | `style: run gofmt` |
| ci | CI 配置 | `ci: add golangci-lint step` |

### 2.3 scope（可选）

scope 用于指明本次变更影响的模块或功能域，帮助读者快速定位变更范围：

- 模块名：`auth`, `api`, `db`, `config`
- 包名：`handler`, `repository`, `middleware`
- 功能域：`connection-pool`, `rate-limit`

### 2.4 subject 规则

- **不超过 50 字符**——GitHub 等平台在列表视图中会截断过长的标题
- 使用**动词原形**开头：add, fix, update, remove, refactor
- **不加句号**
- 描述"做了什么"而非"改了哪个文件"

```
# ✓ 好的 subject
feat(auth): add JWT token refresh mechanism
fix(api): prevent race condition in cache update

# ✗ 差的 subject
feat(auth): added JWT token refresh mechanism.     ← 过去式 + 句号
fix: fix bug                                        ← 没有具体信息
update code                                         ← 缺少 type，描述模糊
```

### 2.5 body（可选）

body 用于详细描述变更的动机和前后对比，适合以下场景：

- 解释**为什么**需要这个变更（而非重复描述做了什么）
- 记录设计决策和权衡取舍
- 每行不超过 72 字符（方便在终端中查看 `git log`）

### 2.6 footer（可选）

- `BREAKING CHANGE:` 不兼容变更说明（会触发 major 版本号升级）
- `Closes #123` 关联并自动关闭 issue
- `Refs:` 相关链接或参考文档

### 2.7 完整示例

```
feat(connection-pool): add idle connection cleanup

Previously idle connections were never cleaned up, leading to
resource exhaustion under sustained load. This adds a background
goroutine that periodically closes connections idle for more than
ConnMaxIdleTime.

Closes #245
```

```
fix(middleware): correct timeout context propagation

The request context was not properly propagated to downstream
service calls, causing premature cancellation when the middleware
timeout was shorter than the downstream RPC deadline.

BREAKING CHANGE: RequestContext() now returns a derived context
instead of the original request context.

Refs: https://github.com/org/repo/issues/189
```

---

## 3 工具链

### 3.1 commitizen

commitizen 提供交互式命令行引导，帮助开发者生成规范的 commit message：

```bash
# 安装
npm install -g commitizen cz-conventional-changelog

# 初始化项目配置
echo '{ "path": "cz-conventional-changelog" }' > .czrc

# 使用（代替 git commit）
git cz
```

执行 `git cz` 后，会依次询问 type、scope、subject、body、footer，最终拼装成一条符合规范的 commit message。

### 3.2 commitlint

commitlint 在 `commit-msg` Git Hook 中校验 message 是否符合规范，不合规的提交会被直接拦截：

```bash
# 安装
npm install -D @commitlint/cli @commitlint/config-conventional

# 创建配置文件
echo "module.exports = { extends: ['@commitlint/config-conventional'] };" \
  > commitlint.config.js

# 配合 husky 注册 hook
npx husky add .husky/commit-msg 'npx commitlint --edit "$1"'
```

### 3.3 自动生成 CHANGELOG

conventional-changelog 工具能基于 commit type 自动归类变更日志：

- `feat` → **Features**
- `fix` → **Bug Fixes**
- `BREAKING CHANGE` → **Breaking Changes**

```bash
# 安装
npm install -g conventional-changelog-cli

# 生成 CHANGELOG（追加模式）
conventional-changelog -p angular -i CHANGELOG.md -s
```

配合语义化版本（SemVer），工具可自动推算版本号：

- 存在 `BREAKING CHANGE` → major（x.0.0）
- 存在 `feat` → minor（0.x.0）
- 仅有 `fix` → patch（0.0.x）

### 3.4 Go 项目中的实践

Go 项目通常不使用 npm 生态，但可以借助以下方式集成：

- 项目根目录放置 `.czrc`，团队中有 Node 环境的成员可直接使用 `git cz`
- 使用 **goreleaser** 发布——它可解析 conventional commits 自动生成 release notes
- 纯 Go 团队可使用 `go-commitlinter` 或在 CI 中通过脚本校验 commit message 格式

---

## 4 Pull Request / Merge Request 最佳实践

### 4.1 PR 粒度

- **一个 PR 解决一个问题**——一个功能 / 一个 bug / 一次重构
- 控制在 **200-400 行变更**以内——超过 400 行的 PR，review 质量会显著下降
- 大功能拆分为多个串行 PR，通过 feature flag 控制未完成部分
- 避免在功能 PR 中夹带格式化或重构——单独提 PR

### 4.2 PR 描述模板

建议在仓库的 `.github/PULL_REQUEST_TEMPLATE.md` 中配置统一模板：

```markdown
## Summary
<!-- 1-3 句话描述本次变更的目的 -->

## Changes
<!-- 列出具体的改动点 -->
- 
- 

## Test Plan
<!-- 如何验证变更是正确的 -->
- [ ] 单元测试通过
- [ ] 手动验证场景 X

## Screenshots（if applicable）
<!-- UI 变更请附截图 -->
```

### 4.3 Code Review 要点

**Reviewer 角度：**

- **关注设计和逻辑**，而非代码风格——风格问题交给 linter
- 优先指出 **bug 和安全隐患**，其次是可维护性建议
- 用"建议"而非"要求"的语气——`nit:` 前缀表示非阻塞建议
- 如果改动过大，建议作者拆分 PR 而非勉强 review

**Author 角度：**

- PR 提交前先做 self-review
- 对复杂逻辑主动添加 review comment 说明意图
- 及时回复 reviewer 的问题，不要让 PR 挂太久

### 4.4 CI 门禁

合并到主分支前，CI 应至少包含以下检查：

```yaml
# GitHub Actions 示例
steps:
  - name: Lint
    run: golangci-lint run ./...

  - name: Test
    run: go test -race -coverprofile=coverage.out ./...

  - name: Build
    run: go build ./...

  - name: Coverage Gate
    run: |
      # 增量覆盖率不低于 80%
      go tool cover -func=coverage.out
```

---

### 4.5 GitHub 合并策略选型

GitHub 提供三种合并按钮，每种策略对提交历史的影响截然不同：

| 策略 | 对应命令 | 历史效果 | 适用场景 |
|------|----------|----------|----------|
| **Create a merge commit** | `git merge --no-ff` | 保留分支所有 commit + 一个 merge commit | 需要保留完整开发历史的团队 |
| **Squash and merge** | `git merge --squash` | 将 PR 的所有 commit 压缩为一个 commit 合入 | 要求主分支每个 commit 对应一个完整功能 |
| **Rebase and merge** | `git rebase` + `git merge --ff` | 将 PR 的 commit 逐个 rebase 到目标分支末尾 | 追求线性历史，每个 commit 都有独立意义 |

**三种策略的历史对比**：

```
初始状态:
          D --- E  (feature PR, 2 commits)
         /
A --- B --- C  (main)

===== Create a merge commit =====
          D --- E
         /       \
A --- B --- C --- M  (main, 保留了 D 和 E)

===== Squash and merge =====
A --- B --- C --- DE'  (main, D+E 被压缩为一个 commit)

===== Rebase and merge =====
A --- B --- C --- D' --- E'  (main, 线性历史)
```

**选型建议**：

- **Squash and merge（推荐大多数团队）**：PR 内部的 WIP commit 不会污染主分支，主分支的每个 commit 都对应一个完整的 PR，`git log --oneline` 干净整洁。配合 Conventional Commits 规范，PR 标题即为 squash 后的 commit message。
- **Create a merge commit**：适合需要追溯 PR 内部每个 commit 的场景，如开源项目中希望保留贡献者的完整提交记录。
- **Rebase and merge**：适合每个 commit 都经过精心整理（rebase -i）的团队，要求开发者在提 PR 前自行 squash 和 reword。

> **团队统一**：在 GitHub 仓库的 Settings → General → Pull Requests 中，可以禁用不需要的合并策略，确保团队只使用约定的方式。

---

## 5 分支命名规范

### 5.1 推荐格式

```
<type>/<short-description>
```

使用小写字母，单词间用短横线 `-` 连接，避免使用下划线或驼峰。

### 5.2 示例

| 类型 | 命名 | 说明 |
|------|------|------|
| 新功能 | `feature/oauth-login` | 功能分支 |
| 修复 | `fix/nil-pointer-getuser` | Bug 修复 |
| 重构 | `refactor/db-connection-pool` | 重构 |
| 发布 | `release/v1.2.0` | 发布分支 |
| 热修复 | `hotfix/critical-auth-bypass` | 紧急修复 |
| 文档 | `docs/update-api-readme` | 文档更新 |

### 5.3 分支保护规则

在 GitHub / GitLab 中，应为核心分支配置保护规则：

| 规则 | main / master | develop |
|------|---------------|---------|
| 禁止直接推送 | ✅ | ✅ |
| 必须通过 PR 合并 | ✅ | ✅ |
| 要求至少 1 人 approve | ✅ | 建议 |
| 要求 CI 全部通过 | ✅ | ✅ |
| 禁止 force push | ✅ | ✅ |
| 要求线性历史（rebase） | 可选 | 可选 |

---

## 6 小结

工作流选择、提交规范、PR 实践构成了团队 Git 协作的**三大支柱**：

1. **工作流**决定了代码如何从开发流向生产——根据团队规模、发布频率、CI/CD 成熟度选择合适的模型
2. **提交规范**让每一次 commit 都有迹可循——结构化的 message 是自动化 CHANGELOG 和版本管理的基础
3. **PR 实践**保障了代码质量的最后一道防线——合理的粒度、清晰的描述、严格的 CI 门禁缺一不可

三者不是孤立的规则，而是一套有机的协作体系。从 Conventional Commits 开始落地，逐步完善 CI 门禁和 PR 模板，最终选定适合团队的工作流——这是一条务实的演进路径。

## 7 动手练习

1. **工作流模拟**：在一个测试仓库中模拟 GitHub Flow 全流程——创建 feature 分支 → 提交 3 个 commit → push 到远程 → 用 `gh pr create` 或 GitHub 网页创建 PR → 合并 → 删除分支。

2. **Conventional Commits 练习**：写出以下场景对应的 commit message（注意 type、scope、subject 的选择）：
   - 给用户模块添加了邮箱验证功能
   - 修复了订单查询接口的空指针问题
   - 将 Go 版本从 1.23 升级到 1.24
   - 对数据库连接池代码进行了重构

3. **GitHub 合并策略对比**：创建一个 PR 包含 3 个 commit，分别用三种合并策略（merge commit / squash / rebase）合并到不同分支，用 `git log --oneline --graph --all` 对比三种策略产生的历史差异。

4. **CI 门禁搭建**：在一个 Go 项目中配置 `.github/workflows/ci.yml`，包含 lint、test、build 三个步骤，验证 PR 提交时 CI 自动运行。

> 上一章：[05 - Git 历史操控详解](../05-历史操控/Git历史操控详解.md) · 下一章：[07 - Git 进阶技巧](../07-进阶技巧/Git进阶技巧.md)
