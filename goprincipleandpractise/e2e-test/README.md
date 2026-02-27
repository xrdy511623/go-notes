# Go 端到端测试详解

端到端测试（End-to-End Testing，简称 E2E）从**用户的视角**验证整个系统是否正常工作。它不关心内部实现，只关心：用户发起一个操作，系统是否返回了正确的结果。E2E 测试是测试金字塔的顶层，提供最高的信心，但也带来最高的成本。

---

## 1 什么是端到端测试

### 1.1 定义

端到端测试将整个应用视为一个**黑盒**，通过其公开接口（HTTP API、gRPC、CLI、UI）发起请求，验证从入口到出口的完整流程。

```
E2E 测试视角：

用户/客户端
    │
    ▼
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌──────────┐
│  API GW  │──→│ Service A│──→│ Service B│──→│ Database │
└─────────┘    └─────────┘    └─────────┘    └──────────┘
                    │                              ▲
                    ▼                              │
              ┌──────────┐    ┌──────────┐         │
              │  Redis   │    │  Kafka   │─────────┘
              └──────────┘    └──────────┘

E2E 测试只关心：输入 → 黑盒 → 输出
不关心内部走了哪条链路
```

### 1.2 与集成测试的区别

| 维度 | 集成测试 | E2E 测试 |
|------|---------|---------|
| **测试范围** | 2-3 个组件的交互 | 整个系统 |
| **环境** | 部分依赖容器化 | 完整环境（所有服务 + 中间件） |
| **视角** | 开发者（白盒/灰盒） | 用户（黑盒） |
| **验证方式** | 直接检查数据库/内部状态 | 只通过公开接口验证 |
| **速度** | 秒级 | 分钟级 |
| **定位精度** | 组件级别 | 系统级别（需进一步排查） |

### 1.3 E2E 测试能发现什么

- **跨服务通信问题**：服务间的 Protobuf 版本不兼容
- **部署配置错误**：环境变量缺失、端口冲突、TLS 证书过期
- **用户流程断裂**：注册 → 登录 → 下单 → 支付，某个环节失败
- **中间件组合问题**：认证 + 限流 + 日志的组合导致意外行为
- **数据一致性**：跨库事务、最终一致性延迟超出预期

---

## 2 为什么需要端到端测试

### 2.1 单元测试和集成测试的盲区

```
单元测试覆盖:     ■ ■ ■ ■ ■ ■ ■ ■  （每个函数）
集成测试覆盖:     ■───■  ■───■       （组件对）
E2E 测试覆盖:     ■───■───■───■───■   （完整链路）
```

一个真实的例子：所有单元测试通过，所有集成测试通过，但用户无法完成支付。原因是网关的限流配置将支付回调 webhook 误判为恶意请求并拦截了。这种跨层级的配置问题，只有 E2E 测试才能发现。

### 2.2 发布信心

E2E 测试是发布前的最后一道防线。通过验证核心用户路径（Happy Path），我们可以确信：
- 用户可以注册和登录
- 核心业务流程可以跑通
- 关键页面可以正常渲染

### 2.3 但 E2E 是昂贵的

| 成本维度 | 说明 |
|---------|------|
| **执行时间** | 单个 E2E 测试可能需要 30s-5min |
| **环境成本** | 需要完整的运行环境（多个服务 + 中间件） |
| **维护成本** | 对系统变更敏感，容易因无关变更而失败 |
| **调试成本** | 失败时需要跨多个服务排查 |
| **不稳定性** | 依赖链越长，随机失败概率越高 |

因此，E2E 测试应该**少而精**——只覆盖最关键的用户路径。

---

## 3 Go 中的端到端测试实践

### 3.1 E2E 测试的架构

```
┌──────────────────────────────────────────┐
│            E2E Test Suite                │
│                                          │
│  1. 启动完整环境 (docker-compose up)      │
│  2. 等待所有服务就绪 (health check)       │
│  3. 通过公开接口执行测试                  │
│  4. 验证响应                              │
│  5. 清理环境 (docker-compose down)        │
└──────────────────────────────────────────┘
        │                    ▲
        │  HTTP/gRPC/CLI     │  Response
        ▼                    │
┌──────────────────────────────────────────┐
│           Running System                 │
│  ┌────────┐ ┌────────┐ ┌────────┐       │
│  │ App    │ │ DB     │ │ Redis  │       │
│  │ Server │ │ (PG)   │ │        │       │
│  └────────┘ └────────┘ └────────┘       │
└──────────────────────────────────────────┘
```

### 3.2 使用 docker-compose 搭建完整测试环境

```yaml
# docker-compose.test.yml
version: "3.8"

services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://testuser:testpass@postgres:5432/testdb?sslmode=disable
      - REDIS_URL=redis://redis:6379
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 5s
      timeout: 3s
      retries: 10

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: testdb
      POSTGRES_USER: testuser
      POSTGRES_PASSWORD: testpass
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U testuser -d testdb"]
      interval: 3s
      timeout: 3s
      retries: 10

  redis:
    image: redis:7-alpine
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 3s
      timeout: 3s
      retries: 10
```

### 3.3 使用 TestMain 编排 E2E 测试

```go
//go:build e2e

package e2e_test

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "os/exec"
    "testing"
    "time"
)

var baseURL string

func TestMain(m *testing.M) {
    // 1. 启动完整环境
    if err := startEnvironment(); err != nil {
        log.Fatalf("failed to start environment: %v", err)
    }

    // 2. 等待服务就绪
    baseURL = "http://localhost:8080"
    if err := waitForReady(baseURL+"/health", 60*time.Second); err != nil {
        stopEnvironment()
        log.Fatalf("service not ready: %v", err)
    }

    // 3. 运行测试
    code := m.Run()

    // 4. 清理环境
    stopEnvironment()

    os.Exit(code)
}

func startEnvironment() error {
    cmd := exec.Command("docker-compose", "-f", "docker-compose.test.yml", "up", "-d", "--build")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

func stopEnvironment() {
    cmd := exec.Command("docker-compose", "-f", "docker-compose.test.yml", "down", "-v")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Run()
}

// waitForReady 使用轮询等待服务就绪
func waitForReady(url string, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    client := &http.Client{Timeout: 2 * time.Second}

    for time.Now().Before(deadline) {
        resp, err := client.Get(url)
        if err == nil && resp.StatusCode == http.StatusOK {
            resp.Body.Close()
            return nil
        }
        if resp != nil {
            resp.Body.Close()
        }
        time.Sleep(1 * time.Second)
    }
    return fmt.Errorf("service not ready after %v", timeout)
}
```

### 3.4 HTTP API 端到端测试

```go
//go:build e2e

package e2e_test

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "testing"
)

// TestUserJourney 测试用户完整旅程：注册 → 登录 → 查看资料 → 修改资料
func TestUserJourney(t *testing.T) {
    client := &http.Client{}

    // Step 1: 注册
    registerBody := map[string]string{
        "name":     "Alice",
        "email":    "alice@example.com",
        "password": "SecurePass123!",
    }
    registerResp := doPost(t, client, baseURL+"/api/register", registerBody)
    if registerResp.StatusCode != http.StatusCreated {
        t.Fatalf("register: expected 201, got %d", registerResp.StatusCode)
    }
    registerResp.Body.Close()

    // Step 2: 登录
    loginBody := map[string]string{
        "email":    "alice@example.com",
        "password": "SecurePass123!",
    }
    loginResp := doPost(t, client, baseURL+"/api/login", loginBody)
    if loginResp.StatusCode != http.StatusOK {
        t.Fatalf("login: expected 200, got %d", loginResp.StatusCode)
    }
    var loginResult struct {
        Token string `json:"token"`
    }
    json.NewDecoder(loginResp.Body).Decode(&loginResult)
    loginResp.Body.Close()

    if loginResult.Token == "" {
        t.Fatal("login: expected token, got empty")
    }

    // Step 3: 查看资料（带 Token）
    profileReq, _ := http.NewRequest("GET", baseURL+"/api/profile", nil)
    profileReq.Header.Set("Authorization", "Bearer "+loginResult.Token)
    profileResp, err := client.Do(profileReq)
    if err != nil {
        t.Fatalf("profile: %v", err)
    }
    defer profileResp.Body.Close()

    if profileResp.StatusCode != http.StatusOK {
        t.Fatalf("profile: expected 200, got %d", profileResp.StatusCode)
    }

    var profile struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }
    json.NewDecoder(profileResp.Body).Decode(&profile)
    if profile.Name != "Alice" {
        t.Errorf("profile: expected Alice, got %s", profile.Name)
    }
}

func doPost(t *testing.T, client *http.Client, url string, body any) *http.Response {
    t.Helper()
    jsonBody, _ := json.Marshal(body)
    resp, err := client.Post(url, "application/json", bytes.NewReader(jsonBody))
    if err != nil {
        t.Fatalf("POST %s: %v", url, err)
    }
    return resp
}
```

### 3.5 CLI 端到端测试

```go
//go:build e2e

package e2e_test

import (
    "os/exec"
    "strings"
    "testing"
)

// TestCLI_Version 测试 CLI 工具的版本输出
func TestCLI_Version(t *testing.T) {
    cmd := exec.Command("./myapp", "version")
    output, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("exit error: %v, output: %s", err, output)
    }

    if !strings.Contains(string(output), "v1.") {
        t.Errorf("expected version string, got: %s", output)
    }
}

// TestCLI_MigrateAndSeed 测试数据库迁移和数据填充命令
func TestCLI_MigrateAndSeed(t *testing.T) {
    // Step 1: 执行迁移
    migrateCmd := exec.Command("./myapp", "migrate", "--up")
    migrateOutput, err := migrateCmd.CombinedOutput()
    if err != nil {
        t.Fatalf("migrate failed: %v, output: %s", err, migrateOutput)
    }

    // Step 2: 执行数据填充
    seedCmd := exec.Command("./myapp", "seed", "--env=test")
    seedOutput, err := seedCmd.CombinedOutput()
    if err != nil {
        t.Fatalf("seed failed: %v, output: %s", err, seedOutput)
    }

    // Step 3: 验证数据
    checkCmd := exec.Command("./myapp", "user", "list", "--format=json")
    checkOutput, err := checkCmd.CombinedOutput()
    if err != nil {
        t.Fatalf("check failed: %v, output: %s", err, checkOutput)
    }

    if !strings.Contains(string(checkOutput), "admin@example.com") {
        t.Errorf("expected seed data in output, got: %s", checkOutput)
    }
}

// TestCLI_InvalidArgs 测试错误参数处理
func TestCLI_InvalidArgs(t *testing.T) {
    cmd := exec.Command("./myapp", "nonexistent-command")
    output, err := cmd.CombinedOutput()

    // 应该返回非零退出码
    if err == nil {
        t.Error("expected non-zero exit code for invalid command")
    }

    // 错误信息应该有帮助性
    if !strings.Contains(string(output), "unknown command") {
        t.Errorf("expected helpful error message, got: %s", output)
    }
}
```

### 3.6 测试数据与环境管理

#### 数据隔离策略

```go
// 方案一：每个测试使用唯一前缀
func uniqueEmail(t *testing.T) string {
    t.Helper()
    return fmt.Sprintf("e2e_%d_%s@test.com", time.Now().UnixNano(), t.Name())
}

// 方案二：每个测试套件使用独立数据库
// docker-compose.test.yml 中为每次运行创建新数据库
// TEST_DB_NAME=testdb_$(date +%s) docker-compose up

// 方案三：测试前后清理
func cleanupTestData(t *testing.T, client *http.Client) {
    t.Helper()
    req, _ := http.NewRequest("POST", baseURL+"/api/admin/cleanup", nil)
    req.Header.Set("X-Test-Cleanup-Key", os.Getenv("TEST_CLEANUP_KEY"))
    resp, err := client.Do(req)
    if err != nil {
        t.Logf("cleanup warning: %v", err)
        return
    }
    resp.Body.Close()
}
```

---

## 4 什么时候写端到端测试

### 4.1 应该写 E2E 测试的场景

| 场景 | 原因 | 示例 |
|------|------|------|
| **核心业务流程** | 这些流程出问题直接影响收入 | 注册→登录→下单→支付 |
| **跨服务交互** | 无法在单个服务内验证 | 订单服务→库存服务→物流服务 |
| **部署后冒烟测试** | 验证部署成功 | 健康检查 + 核心 API 可达 |
| **认证/授权链路** | 涉及多个中间件和服务 | OAuth → Token → 权限校验 |

### 4.2 不应该写 E2E 测试的场景

| 场景 | 应该用什么 |
|------|-----------|
| 单个函数的边界条件 | 单元测试 |
| 数据库查询正确性 | 集成测试 |
| 所有错误码覆盖 | 单元测试 + 集成测试 |
| 性能基准 | Benchmark |
| 输入格式校验 | 单元测试 + Fuzzing |

### 4.3 经验法则

> 一个典型微服务的 E2E 测试数量：**5-20 个**。
> 只测试"黄金路径"（Golden Path）和最关键的错误场景。
> 如果你的 E2E 测试超过 50 个，大概率应该把一部分下沉到集成测试。

---

## 5 E2E 测试的挑战与应对

### 5.1 测试速度慢

**问题**：E2E 测试依赖完整环境，启动慢、执行慢。

**应对策略**：
```bash
# 1. 环境复用：测试间共享环境，只在开始和结束时启动/销毁
# TestMain 中 docker-compose up，所有测试共享

# 2. 选择性执行：只在 merge 到主分支时运行完整 E2E
# PR 阶段只运行冒烟测试（标记为 smoke 的子集）
go test -tags=e2e -run='TestSmoke' ./...

# 3. 并行执行独立的测试
func TestOrderFlow(t *testing.T)   { t.Parallel(); ... }
func TestProfileFlow(t *testing.T) { t.Parallel(); ... }
```

### 5.2 测试不稳定（Flaky Tests）

**问题**：E2E 测试因网络、时序、资源竞争等原因随机失败。

**应对策略**：

```go
// ❌ 错误：固定等待
time.Sleep(5 * time.Second)

// ✅ 正确：轮询等待
func waitForCondition(t *testing.T, check func() bool, timeout time.Duration) {
    t.Helper()
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if check() {
            return
        }
        time.Sleep(200 * time.Millisecond)
    }
    t.Fatal("condition not met within timeout")
}

// 使用
waitForCondition(t, func() bool {
    resp, _ := http.Get(baseURL + "/api/orders/" + orderID)
    return resp != nil && resp.StatusCode == http.StatusOK
}, 30*time.Second)
```

**Flaky Test 管理**：
1. **隔离**：将不稳定测试标记为 `//go:build flaky`，单独运行
2. **重试**：CI 中对 E2E 测试允许 1-2 次重试
3. **追踪**：记录每个 E2E 测试的历史通过率，低于 95% 的需要修复

### 5.3 调试困难

**问题**：E2E 测试失败时，问题可能在任何一个服务中。

**应对策略**：

```go
// 1. 每个步骤记录详细日志
func TestOrderFlow(t *testing.T) {
    t.Log("Step 1: Creating user...")
    user := createUser(t)
    t.Logf("Step 1 OK: user=%+v", user)

    t.Log("Step 2: Placing order...")
    order := placeOrder(t, user.Token)
    t.Logf("Step 2 OK: order=%+v", order)
    // ...
}

// 2. 失败时保存响应体
func assertStatus(t *testing.T, resp *http.Response, expected int) {
    t.Helper()
    if resp.StatusCode != expected {
        body, _ := io.ReadAll(resp.Body)
        t.Fatalf("expected status %d, got %d, body: %s",
            expected, resp.StatusCode, body)
    }
}

// 3. 使用 trace ID 关联日志
// 每个 E2E 请求携带唯一 trace ID，方便在日志系统中追踪
```

### 5.4 环境管理

```bash
# 临时环境：每次 CI 运行创建新环境
docker-compose -f docker-compose.test.yml -p "e2e-${CI_RUN_ID}" up -d

# 运行测试
go test -tags=e2e -timeout=600s ./e2e/...

# 无论测试成功失败，都清理环境
docker-compose -f docker-compose.test.yml -p "e2e-${CI_RUN_ID}" down -v
```

---

## 6 E2E 测试的最佳实践

### 6.1 测试用户行为，而非实现细节

```go
// ❌ 错误：E2E 测试关注内部实现
func TestCreateOrder_Bad(t *testing.T) {
    createOrder(t)

    // 直接查数据库验证 → 这是集成测试该做的事
    var count int
    testDB.QueryRow("SELECT COUNT(*) FROM orders").Scan(&count)
    assert.Equal(t, 1, count)
}

// ✅ 正确：通过公开 API 验证
func TestCreateOrder_Good(t *testing.T) {
    order := createOrder(t)

    // 通过 API 查询验证
    fetched := getOrder(t, order.ID)
    assert.Equal(t, "pending", fetched.Status)
}
```

### 6.2 使用 Build Tag 隔离

```go
//go:build e2e

// E2E 测试只在显式指定 tag 时运行
// go test -tags=e2e ./e2e/...
```

### 6.3 测试独立性

每个 E2E 测试应该可以独立运行，不依赖其他测试的执行结果：

```go
// 每个测试创建自己的数据
func TestOrderFlow(t *testing.T) {
    user := registerNewUser(t) // 自己创建用户
    login(t, user)
    order := placeOrder(t, user)
    payOrder(t, order)
    // ...
}
```

### 6.4 合理设置超时

```go
// 单个 E2E 测试的超时
func TestPaymentFlow(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    // 整个测试在 ctx 的超时范围内执行
    // ...
}

// 命令行全局超时
// go test -tags=e2e -timeout=600s ./e2e/...
```

---

## 7 整合四种测试类型的策略

这是本文最核心的部分。单元测试、模糊测试、集成测试和 E2E 测试各有所长，合理组合才能最大化地发现 Bug、提高代码质量。

### 7.1 测试金字塔的实践

```
            /   E2E   \          ← 5-10%: 关键用户路径
           /───────────\
          / Integration  \       ← 15-25%: 组件交互
         /────────────────\
        /   Unit Tests      \    ← 60-70%: 业务逻辑
       /─────────────────────\
      /    Fuzzing Tests       \ ← 对解析器/编解码器持续运行
     /──────────────────────────\
```

各层的职责：

| 层级 | 占比 | 验证什么 | 由谁编写 | 运行频率 |
|------|------|---------|---------|---------|
| **E2E** | 5-10% | 关键用户旅程完整可用 | 开发 + QA | 合入主分支 / 发布前 |
| **集成测试** | 15-25% | 组件交互正确 | 开发 | 每次 CI |
| **单元测试** | 60-70% | 业务逻辑正确 | 开发 | 每次 CI（最先运行） |
| **Fuzzing** | 持续运行 | 边界/异常输入不崩溃 | 开发 | 夜间/持续 |

### 7.2 什么时候写哪种测试（决策树）

面对一个新功能或 Bug 修复，按照下面的决策流程选择测试类型：

```
你要测试的是什么？
│
├─ 纯计算函数（无外部依赖）
│  └→ 单元测试
│     └→ 如果处理用户输入/解析器 → 额外加 Fuzzing
│
├─ 依赖数据库/缓存的函数
│  ├→ 单元测试（Mock 依赖，验证逻辑分支）
│  └→ 集成测试（真实依赖，验证 SQL/缓存操作）
│
├─ HTTP Handler / API 端点
│  ├→ 单元测试（httptest.NewRecorder，验证 Handler 逻辑）
│  ├→ 集成测试（httptest.NewServer + 中间件 + 真实 DB）
│  └→ E2E（如果是核心用户路径的一部分）
│
├─ 消息队列生产者/消费者
│  ├→ 单元测试（Mock MQ，验证消息构造和处理逻辑）
│  └→ 集成测试（testcontainer 启动 Kafka/RabbitMQ）
│
├─ 跨服务的完整业务流程
│  └→ E2E 测试
│
├─ 序列化/反序列化（JSON/Protobuf/自定义协议）
│  ├→ 单元测试（已知的 input/output pairs）
│  └→ Fuzzing（随机输入发现边界 Bug）
│
└─ CLI 工具
   ├→ 单元测试（核心逻辑函数）
   └→ E2E（os/exec 运行编译后的二进制）
```

### 7.3 一个具体的例子

假设我们有一个"用户注册"功能，涉及以下组件：

```
HTTP Handler → 参数校验 → UserService → UserRepository → PostgreSQL
                  │                           │
                  └→ 密码加密                  └→ 唯一性约束检查
```

我们应该这样分配测试：

| 组件 | 单元测试 | Fuzzing | 集成测试 | E2E |
|------|---------|---------|---------|-----|
| 参数校验 | ✅ 各种非法输入 | ✅ 随机字符串 | | |
| 密码加密 | ✅ 已知 hash 对比 | | | |
| UserService 逻辑 | ✅ Mock Repo | | | |
| UserRepository SQL | | | ✅ 真实 PG | |
| 唯一性约束 | | | ✅ 真实 PG | |
| 注册完整流程 | | | | ✅ HTTP 请求 |

### 7.4 CI/CD 中的测试编排

```
┌─────────┐    ┌──────────────────┐    ┌───────────────┐    ┌──────────┐
│ git push │──→│  Unit Tests      │──→│  Integration  │──→│   E2E    │
│          │    │  + Fuzzing Seed  │    │    Tests      │    │  Tests   │
│          │    │  (< 2 min)       │    │  (< 5 min)    │    │ (< 10m)  │
└─────────┘    └──────────────────┘    └───────────────┘    └──────────┘
                     │ fail                  │ fail              │ fail
                     ▼                       ▼                   ▼
               快速反馈:              组件级定位:           系统级问题:
               "这个函数逻辑         "SQL 语法错误"        "支付链路不通"
                有 Bug"              "缓存 key 冲突"       "配置缺失"
```

Makefile 示例：

```makefile
.PHONY: test test-unit test-integration test-e2e test-fuzz test-all

# 单元测试（最快，每次 push 都运行）
test-unit:
	go test -race -count=1 ./...

# Fuzzing 种子回归（作为单元测试的一部分）
test-fuzz-seed:
	go test -run='Fuzz' ./...

# 集成测试（需要 Docker）
test-integration:
	go test -tags=integration -race -timeout=300s ./...

# E2E 测试（需要完整环境）
test-e2e:
	docker-compose -f docker-compose.test.yml up -d --build --wait
	go test -tags=e2e -timeout=600s ./e2e/... || (docker-compose -f docker-compose.test.yml down -v; exit 1)
	docker-compose -f docker-compose.test.yml down -v

# 长时间 Fuzzing（夜间运行）
test-fuzz:
	go test -fuzz=. -fuzztime=30m ./pkg/parser/...

# CI 完整流水线
test-all: test-unit test-fuzz-seed test-integration test-e2e
```

### 7.5 覆盖率策略

不同测试类型的覆盖率目标不同：

| 测试类型 | 覆盖率目标 | 度量方式 |
|---------|-----------|---------|
| **单元测试** | 80%+ 行覆盖率 | `go test -cover ./...` |
| **集成测试** | 关键路径 100% | 按接口/查询清单检查 |
| **E2E 测试** | 核心旅程 100% | 按用户故事清单检查 |
| **Fuzzing** | 解析器/编解码器 | 持续运行，关注崩溃发现 |

获取综合覆盖率：

```bash
# 合并单元测试和集成测试的覆盖率
go test -cover -coverprofile=unit.out ./...
go test -tags=integration -cover -coverprofile=integration.out ./...

# 使用 gocovmerge 合并
go install github.com/wadey/gocovmerge@latest
gocovmerge unit.out integration.out > merged.out
go tool cover -func=merged.out
```

### 7.6 团队协作中的测试策略

| 角色 | 职责 |
|------|------|
| **开发** | 单元测试 + 集成测试（代码提交前） |
| **开发** | Fuzzing 测试（对解析器、编解码器） |
| **开发 + QA** | E2E 测试（核心旅程） |
| **QA** | E2E 测试维护和扩展 |
| **SRE** | 冒烟测试（部署后验证） |

Code Review 测试检查清单：

- [ ] 新功能是否有单元测试？覆盖率是否 >= 80%？
- [ ] Mock 的依赖是否有对应的集成测试验证？
- [ ] 如果涉及解析/编解码，是否有 Fuzzing 测试？
- [ ] 如果是核心用户路径，是否有/更新了 E2E 测试？
- [ ] 测试是否可以独立运行、可重复执行？

### 7.7 四种测试的互补关系总结

```
                发现 Bug 的能力
                ┌─────────────────────────────────────┐
  逻辑错误      │ ████████████████████  单元测试        │
                │ ██████              集成测试        │
                │ ██                  E2E             │
                │                                     │
  边界/异常输入  │ ████                单元测试        │
                │                     集成测试        │
                │                     E2E             │
                │ ████████████████████  Fuzzing         │
                │                                     │
  交互/集成错误  │                     单元测试        │
                │ ████████████████████  集成测试        │
                │ ████████████        E2E             │
                │                     Fuzzing         │
                │                                     │
  系统级/配置   │                     单元测试        │
                │ ████                集成测试        │
                │ ████████████████████  E2E             │
                │                     Fuzzing         │
                └─────────────────────────────────────┘
```

- **单元测试**：逻辑错误的主力发现者，快速、精准、成本低
- **Fuzzing**：边界和异常输入的主力发现者，发现人类想不到的 corner case
- **集成测试**：交互错误的主力发现者，验证 Mock 背后的真实行为
- **E2E 测试**：系统级问题的最后防线，验证整个系统端到端可用

四者缺一不可，合理搭配才能构建起完整的质量防线。