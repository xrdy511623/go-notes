# Go 集成测试详解

集成测试（Integration Testing）验证的是多个组件组合在一起时的行为是否正确。与单元测试关注单个函数不同，集成测试关注的是组件之间的**交互边界**——数据库查询是否真的能跑通、HTTP Handler 加上中间件后行为是否正确、缓存和数据库的一致性是否成立。

---

## 1 什么是集成测试

### 1.1 定义

集成测试是对**多个组件协同工作**进行验证的测试。它不再用 Mock 替代外部依赖，而是让被测代码与真实（或接近真实）的依赖交互。

```
单元测试:     函数 A ──→ [Mock B]
集成测试:     函数 A ──→ 真实 B ──→ 真实数据库
端到端测试:   用户请求 ──→ 网关 ──→ 服务A ──→ 服务B ──→ 数据库
```

### 1.2 集成测试在测试金字塔中的位置

```
          /    E2E    \         ← 少量，验证关键用户路径
         /-----------\
        / Integration  \       ← 适量，验证组件交互
       /----------------\
      /   Unit Tests      \    ← 大量，验证业务逻辑
     /---------------------\
```

集成测试位于金字塔的中间层。它比单元测试慢，但比端到端测试快；比单元测试覆盖面广，但比端到端测试更容易定位问题。

### 1.3 集成测试能发现什么单元测试发现不了的 Bug

| 缺陷类型 | 单元测试 | 集成测试 |
|---------|---------|---------|
| 接口参数不匹配 | Mock 掩盖了问题 | ✅ 真实调用暴露 |
| SQL 语法/语义错误 | Mock 返回预设值 | ✅ 真实 DB 报错 |
| 序列化/反序列化不一致 | 各自测试通过 | ✅ 联调暴露 |
| 中间件链顺序错误 | Handler 单独正常 | ✅ 完整链路暴露 |
| 连接池配置不当 | 无法模拟 | ✅ 真实连接池行为 |
| 事务隔离级别问题 | Mock 无此概念 | ✅ 真实数据库行为 |

**典型场景**：你在单元测试中 Mock 了 `UserRepository.FindByEmail()`，让它返回一个 User 对象。测试通过了。但实际上你的 SQL 写成了 `WHERE email = ?` 但列名实际是 `user_email`。这个 Bug 只有集成测试才能发现。

---

## 2 为什么需要集成测试

### 2.1 Mock 的局限性

单元测试通过 Mock 隔离依赖，这是正确的做法。但 Mock 本身就是一种**假设**——你假设外部依赖的行为是你 Mock 出来的样子。如果这个假设错了，单元测试照样全绿，但系统就是跑不通。

```go
// 单元测试：Mock 返回固定值，测试通过
mockRepo.On("FindByEmail", "test@example.com").Return(&User{ID: 1, Name: "test"}, nil)

// 但真实 SQL 可能是错的：
// SELECT * FROM users WHERE email = ?     ← 你以为的
// SELECT * FROM users WHERE user_email = ? ← 实际的列名
```

### 2.2 组件边界是 Bug 的高发区

据统计，生产环境中大量的 Bug 出现在组件边界：
- **序列化边界**：JSON/Protobuf 的 field tag 写错
- **数据库边界**：SQL 语法在特定数据库版本上不兼容
- **缓存边界**：缓存 key 的构造规则与读取规则不一致
- **消息队列边界**：生产者和消费者对消息格式的理解不一致

### 2.3 性价比优于 E2E

集成测试的性价比是最高的：

| 维度 | 单元测试 | 集成测试 | E2E 测试 |
|------|---------|---------|---------|
| 执行速度 | 毫秒级 | 秒级 | 分钟级 |
| 环境依赖 | 无 | 容器化依赖 | 完整环境 |
| 定位精度 | 函数级 | 组件级 | 系统级 |
| 维护成本 | 低 | 中 | 高 |
| 发现 Bug 类型 | 逻辑错误 | 交互错误 | 系统级错误 |

---

## 3 Go 中的集成测试实践

### 3.1 用 Build Tag 隔离集成测试

集成测试依赖外部资源（数据库、Redis 等），不应该在普通的 `go test ./...` 中运行。Go 提供了 Build Tag 来隔离它们：

```go
//go:build integration

package user_test

import "testing"

func TestUserRepository_Create(t *testing.T) {
    // 这个测试只有在 go test -tags=integration 时才会运行
}
```

运行方式：

```bash
# 只运行单元测试（默认）
go test ./...

# 运行集成测试
go test -tags=integration ./...

# 同时运行单元测试和集成测试
go test -tags=integration -count=1 ./...
```

CI 流水线中的典型配置：

```yaml
# .github/workflows/test.yml
jobs:
  unit-test:
    steps:
      - run: go test -race ./...

  integration-test:
    needs: unit-test
    services:
      postgres:
        image: postgres:16
    steps:
      - run: go test -tags=integration -race ./...
```

### 3.2 使用 TestMain 管理测试生命周期

集成测试通常需要在所有测试开始前准备环境，在所有测试结束后清理资源。`TestMain` 是最佳入口：

```go
//go:build integration

package repo_test

import (
    "database/sql"
    "log"
    "os"
    "testing"

    _ "github.com/lib/pq"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
    // ===== Setup =====
    var err error
    testDB, err = sql.Open("postgres", os.Getenv("TEST_DATABASE_URL"))
    if err != nil {
        log.Fatalf("failed to connect to test database: %v", err)
    }

    // 执行数据库迁移
    if err := runMigrations(testDB); err != nil {
        log.Fatalf("failed to run migrations: %v", err)
    }

    // ===== Run Tests =====
    code := m.Run()

    // ===== Teardown =====
    testDB.Close()

    os.Exit(code)
}
```

### 3.3 使用 testcontainers-go 管理外部依赖

手动管理外部服务（安装、启动、配置）既繁琐又不可重复。`testcontainers-go` 通过 Docker 自动管理容器生命周期，让集成测试真正做到**一键运行**。

#### PostgreSQL 容器示例

```go
//go:build integration

package repo_test

import (
    "context"
    "database/sql"
    "log"
    "os"
    "testing"

    _ "github.com/lib/pq"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
    ctx := context.Background()

    // 启动 PostgreSQL 容器
    pgContainer, err := postgres.Run(ctx,
        "postgres:16-alpine",
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("testuser"),
        postgres.WithPassword("testpass"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2),
        ),
    )
    if err != nil {
        log.Fatalf("failed to start postgres container: %v", err)
    }

    // 获取连接字符串
    connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        log.Fatalf("failed to get connection string: %v", err)
    }

    testDB, err = sql.Open("postgres", connStr)
    if err != nil {
        log.Fatalf("failed to connect: %v", err)
    }

    // 运行测试
    code := m.Run()

    // 清理
    testDB.Close()
    pgContainer.Terminate(ctx)

    os.Exit(code)
}
```

#### Redis 容器示例

```go
import (
    "github.com/testcontainers/testcontainers-go/modules/redis"
)

func setupRedis(ctx context.Context) (testcontainers.Container, string) {
    redisContainer, err := redis.Run(ctx, "redis:7-alpine")
    if err != nil {
        log.Fatalf("failed to start redis: %v", err)
    }

    endpoint, err := redisContainer.Endpoint(ctx, "")
    if err != nil {
        log.Fatalf("failed to get redis endpoint: %v", err)
    }

    return redisContainer, endpoint
}
```

### 3.4 数据库集成测试

#### 事务回滚隔离模式

最推荐的数据库测试隔离策略是：**每个测试在独立事务中运行，测试结束后回滚**。

```go
// txTestHelper 封装事务隔离的测试辅助函数
func txTestHelper(t *testing.T, testDB *sql.DB, fn func(tx *sql.Tx)) {
    t.Helper()

    tx, err := testDB.Begin()
    if err != nil {
        t.Fatalf("failed to begin transaction: %v", err)
    }

    // 无论测试成功还是失败，都回滚
    defer tx.Rollback()

    fn(tx)
}

func TestUserRepo_Create(t *testing.T) {
    txTestHelper(t, testDB, func(tx *sql.Tx) {
        repo := NewUserRepository(tx)

        user, err := repo.Create(context.Background(), &User{
            Name:  "Alice",
            Email: "alice@example.com",
        })

        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if user.ID == 0 {
            t.Error("expected user ID to be set")
        }
        if user.Name != "Alice" {
            t.Errorf("expected name Alice, got %s", user.Name)
        }
    })
    // 事务已回滚，数据库状态干净
}
```

#### 完整的 Repository 测试示例

```go
func TestUserRepo_FindByEmail(t *testing.T) {
    txTestHelper(t, testDB, func(tx *sql.Tx) {
        repo := NewUserRepository(tx)
        ctx := context.Background()

        // Arrange: 插入测试数据
        _, err := repo.Create(ctx, &User{
            Name:  "Bob",
            Email: "bob@example.com",
        })
        if err != nil {
            t.Fatalf("setup failed: %v", err)
        }

        // Act
        found, err := repo.FindByEmail(ctx, "bob@example.com")

        // Assert
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if found == nil {
            t.Fatal("expected user, got nil")
        }
        if found.Name != "Bob" {
            t.Errorf("expected name Bob, got %s", found.Name)
        }

        // 测试不存在的用户
        notFound, err := repo.FindByEmail(ctx, "nobody@example.com")
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if notFound != nil {
            t.Errorf("expected nil, got %+v", notFound)
        }
    })
}
```

### 3.5 HTTP API 集成测试

与单元测试中使用 `httptest.NewRecorder()` 不同，集成测试应启动一个真实的 HTTP Server，测试完整的请求链路：

```go
func TestAPI_CreateAndGetUser(t *testing.T) {
    // 构建完整的应用（包含中间件、路由、数据库连接）
    app := NewApp(testDB)

    // 启动测试服务器
    server := httptest.NewServer(app.Router())
    defer server.Close()

    client := server.Client()

    // Step 1: 创建用户
    body := `{"name": "Charlie", "email": "charlie@example.com"}`
    resp, err := client.Post(
        server.URL+"/api/users",
        "application/json",
        strings.NewReader(body),
    )
    if err != nil {
        t.Fatalf("POST /api/users failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        t.Fatalf("expected 201, got %d", resp.StatusCode)
    }

    var created User
    if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
        t.Fatalf("failed to decode response: %v", err)
    }

    // Step 2: 查询刚创建的用户
    resp2, err := client.Get(
        fmt.Sprintf("%s/api/users/%d", server.URL, created.ID),
    )
    if err != nil {
        t.Fatalf("GET /api/users/%d failed: %v", created.ID, err)
    }
    defer resp2.Body.Close()

    if resp2.StatusCode != http.StatusOK {
        t.Fatalf("expected 200, got %d", resp2.StatusCode)
    }

    var fetched User
    if err := json.NewDecoder(resp2.Body).Decode(&fetched); err != nil {
        t.Fatalf("failed to decode response: %v", err)
    }

    if fetched.Name != "Charlie" {
        t.Errorf("expected Charlie, got %s", fetched.Name)
    }
}
```

### 3.6 gRPC 集成测试

使用 `bufconn` 实现内存级 gRPC 连接，避免占用真实端口：

```go
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func setupGRPCServer(t *testing.T) *grpc.ClientConn {
    t.Helper()

    lis := bufconn.Listen(bufSize)

    server := grpc.NewServer(
        grpc.UnaryInterceptor(loggingInterceptor),
    )
    pb.RegisterUserServiceServer(server, NewUserService(testDB))

    go func() {
        if err := server.Serve(lis); err != nil {
            t.Errorf("server exited with error: %v", err)
        }
    }()

    t.Cleanup(func() {
        server.GracefulStop()
        lis.Close()
    })

    conn, err := grpc.NewClient(
        "passthrough://bufnet",
        grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
            return lis.DialContext(ctx)
        }),
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        t.Fatalf("failed to dial: %v", err)
    }
    t.Cleanup(func() { conn.Close() })

    return conn
}

func TestGRPC_CreateUser(t *testing.T) {
    conn := setupGRPCServer(t)
    client := pb.NewUserServiceClient(conn)

    resp, err := client.CreateUser(context.Background(), &pb.CreateUserRequest{
        Name:  "Dave",
        Email: "dave@example.com",
    })
    if err != nil {
        t.Fatalf("CreateUser failed: %v", err)
    }
    if resp.User.Name != "Dave" {
        t.Errorf("expected Dave, got %s", resp.User.Name)
    }
}
```

### 3.7 测试数据管理

#### 工厂函数 vs 固定 Fixture

| 方式 | 优点 | 缺点 | 适用场景 |
|------|------|------|---------|
| 固定 Fixture (SQL 文件) | 数据直观 | 耦合度高，维护成本大 | 只读测试 |
| 工厂函数 | 灵活、可组合 | 需要编写代码 | 大多数场景 |
| Builder 模式 | 可读性极好 | 代码量更大 | 复杂对象 |

推荐使用工厂函数：

```go
// testutil/factory.go
func NewTestUser(t *testing.T, overrides ...func(*User)) *User {
    t.Helper()
    user := &User{
        Name:      fmt.Sprintf("user_%d", time.Now().UnixNano()),
        Email:     fmt.Sprintf("test_%d@example.com", time.Now().UnixNano()),
        CreatedAt: time.Now(),
    }
    for _, fn := range overrides {
        fn(user)
    }
    return user
}

// 使用
user := NewTestUser(t, func(u *User) {
    u.Name = "SpecificName"
})
```

---

## 4 什么时候写集成测试

### 4.1 决策规则

**写集成测试的信号**：

1. **你在单元测试中 Mock 了某个依赖** → 写一个集成测试验证真实依赖的行为
2. **涉及数据库查询** → 特别是复杂 JOIN、事务、存储过程
3. **HTTP Handler + 中间件链** → 认证、限流、日志等中间件的组合行为
4. **缓存逻辑** → 缓存穿透、缓存一致性
5. **消息队列** → 生产者/消费者的序列化一致性
6. **配置加载** → 配置文件解析和默认值覆盖

**不需要写集成测试的场景**：

1. **纯计算函数** → 单元测试足够
2. **简单 CRUD** → 如果 ORM 已被充分验证
3. **UI 展示逻辑** → 不涉及后端交互

### 4.2 经验法则

> 如果你在单元测试中 Mock 了某个东西，就该写一个集成测试来验证那个"真实的东西"。

---

## 5 集成测试 vs 单元测试的权衡

### 5.1 对比表

| 维度 | 单元测试 | 集成测试 |
|------|---------|---------|
| **速度** | 毫秒级，可运行数千个 | 秒级，通常数十到数百个 |
| **可靠性** | 极高（无外部依赖） | 高（容器化后接近确定性） |
| **维护成本** | 低 | 中（需要维护容器配置） |
| **调试难度** | 低（失败精准定位） | 中（可能涉及多个组件） |
| **覆盖深度** | 函数内部逻辑 | 组件交互边界 |
| **环境要求** | 无 | Docker |
| **CI 时间** | < 1 分钟 | 2-5 分钟 |

### 5.2 冰淇淋锥反模式

```
错误的测试结构（冰淇淋锥）：       正确的测试结构（金字塔）：

   ████████████  E2E                  /  E2E  \
   ██████████  集成                  / 集成测试 \
   ████████  单元                  / 单元测试    \
```

冰淇淋锥反模式是指：大量的 E2E 测试，少量的单元测试。这导致测试套件运行缓慢、难以维护、定位问题困难。正确做法是遵循测试金字塔。

---

## 6 集成测试的最佳实践

### 6.1 测试隔离

每个测试必须拥有干净的状态，不依赖其他测试的执行结果：

```go
// ✅ 正确：每个测试在独立事务中
func TestA(t *testing.T) {
    txTestHelper(t, testDB, func(tx *sql.Tx) {
        // 只操作 tx，测试结束自动回滚
    })
}

// ❌ 错误：测试之间共享数据
var sharedUserID int64
func TestCreate(t *testing.T) { sharedUserID = createUser() }
func TestGet(t *testing.T)    { getUser(sharedUserID) } // 依赖 TestCreate
```

### 6.2 并行执行

利用 `t.Parallel()` 加速集成测试，但需确保每个测试有独立的数据空间：

```go
func TestUserRepo_Create(t *testing.T) {
    t.Parallel() // 并行执行
    txTestHelper(t, testDB, func(tx *sql.Tx) {
        // 独立事务，互不干扰
    })
}

func TestUserRepo_Delete(t *testing.T) {
    t.Parallel()
    txTestHelper(t, testDB, func(tx *sql.Tx) {
        // 独立事务，互不干扰
    })
}
```

### 6.3 超时管理

集成测试涉及网络和 I/O，必须设置合理的超时：

```go
func TestSlowQuery(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    result, err := repo.HeavyQuery(ctx)
    if err != nil {
        t.Fatalf("query failed: %v", err)
    }
    // ...
}
```

### 6.4 CI/CD 集成

```yaml
# GitHub Actions 示例
integration-test:
  runs-on: ubuntu-latest
  services:
    postgres:
      image: postgres:16-alpine
      env:
        POSTGRES_DB: testdb
        POSTGRES_USER: testuser
        POSTGRES_PASSWORD: testpass
      ports:
        - 5432:5432
      options: >-
        --health-cmd pg_isready
        --health-interval 10s
        --health-timeout 5s
        --health-retries 5
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.22'
    - run: go test -tags=integration -race -timeout=300s ./...
      env:
        TEST_DATABASE_URL: postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable
```

### 6.5 测试命名规范

```go
// 格式: Test<Layer>_<Method>_<Scenario>
func TestUserRepo_Create_DuplicateEmail(t *testing.T)   {}
func TestUserRepo_FindByEmail_NotFound(t *testing.T)    {}
func TestOrderAPI_PlaceOrder_InsufficientStock(t *testing.T) {}
```