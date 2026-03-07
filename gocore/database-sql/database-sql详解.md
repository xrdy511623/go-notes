# Go 标准库 database/sql 详解

`database/sql` 是 Go 标准库中操作关系型数据库的统一接口层。它不直接与任何数据库通信，而是通过驱动接口将应用代码与具体数据库实现解耦。本文将从源码层面剖析其核心设计，覆盖连接池、事务、预编译语句等关键主题。

---

## 1 sql.DB 不是连接，是连接池管理器

### 1.1 常见误解

新手最常犯的错误是把 `sql.DB` 当作一个数据库连接：

```go
// 误以为这里建立了一个数据库连接
db, err := sql.Open("mysql", "user:pass@tcp(127.0.0.1:3306)/mydb")
```

**事实上，`sql.Open()` 不会建立任何连接。** 它只是：
1. 验证驱动名称是否已注册
2. 保存 DSN（Data Source Name）
3. 返回一个 `*sql.DB` 实例

第一次真正的连接在你执行 `db.Ping()`、`db.Query()` 等操作时才会建立。`sql.DB` 是一个**并发安全的连接池管理器**，设计为在整个程序生命周期中作为长生命周期对象使用，不需要频繁创建和关闭。

```go
// 正确用法：程序启动时创建，全局复用
func main() {
    db, err := sql.Open("sqlite", ":memory:")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // 验证连接可用
    if err := db.Ping(); err != nil {
        log.Fatal("数据库不可达:", err)
    }

    // db 传递给各业务层使用，不要每次操作都 Open/Close
    startServer(db)
}
```

### 1.2 核心类型全景

通过 `go doc database/sql | grep "^type" | grep struct` 可以看到核心结构体：

| 类型 | 角色 | 生命周期 |
|------|------|---------|
| `DB` | 连接池管理器 | 程序级别，全局唯一 |
| `Conn` | 单个数据库连接（Go 1.9+） | 按需获取，用完归还 |
| `Tx` | 事务，绑定到一个连接 | Begin → Commit/Rollback |
| `Stmt` | 预编译语句 | Prepare → 多次执行 → Close |
| `Row` | 单行查询结果 | QueryRow 返回，Scan 后自动释放 |
| `Rows` | 多行查询结果 | Query 返回，**必须 Close** |

它们之间的关系：

```
DB (连接池)
 ├── Conn (单个连接，通常由 DB 自动管理)
 │    ├── Tx (事务，独占此连接)
 │    │    ├── Stmt (事务内的预编译语句)
 │    │    └── Rows/Row (事务内的查询结果)
 │    ├── Stmt (连接级别的预编译语句)
 │    └── Rows/Row (查询结果，持有连接直到 Close)
 └── Stmt (池级别的预编译语句，可跨连接复用)
```

### 1.3 驱动注册机制

`database/sql` 通过全局注册表将驱动名映射到驱动实现：

```go
// 源码: database/sql/sql.go
var (
    driversMu sync.RWMutex
    drivers   = make(map[string]driver.Driver)
)

func Register(name string, driver driver.Driver) {
    driversMu.Lock()
    defer driversMu.Unlock()
    if driver == nil {
        panic("sql: Register driver is nil")
    }
    if _, dup := drivers[name]; dup {
        panic("sql: Register called twice for driver " + name)
    }
    drivers[name] = driver
}
```

驱动通过 `init()` 函数自动注册：

```go
// github.com/go-sql-driver/mysql 的 init 函数
func init() {
    sql.Register("mysql", &MySQLDriver{})
}
```

因此应用代码只需要一个 **side-effect import**：

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"  // 仅执行 init()，注册驱动
)
```

这种设计实现了 `database/sql` 与具体驱动的完全解耦——应用代码只依赖标准库接口，更换数据库只需更改 import 和 DSN。

---

## 2 连接池内部实现

### 2.1 池的数据结构

`sql.DB` 内部维护的核心字段（简化）：

```go
type DB struct {
    connector  driver.Connector  // 创建新连接的工厂
    mu         sync.Mutex
    freeConn   []*driverConn     // 空闲连接池（切片）
    connRequests connRequestSet  // 等待连接的请求队列
    numOpen    int               // 当前打开的连接数
    maxOpen    int               // 最大打开连接数（0=无限制）
    maxIdle    int               // 最大空闲连接数（默认2）
    maxLifetime time.Duration    // 连接最大存活时间
    maxIdleTime time.Duration    // 空闲连接最大保留时间
}
```

获取连接的内部流程（`db.conn()` 方法）：

```
1. 加锁
2. 检查空闲连接池 freeConn：
   a. 有空闲连接 → 取出，检查是否过期 → 过期则关闭，重试
   b. 无空闲连接 → 步骤 3
3. 检查 numOpen < maxOpen？
   a. 是 → 创建新连接，numOpen++
   b. 否 → 创建等待请求，放入 connRequests 队列，阻塞等待
4. 解锁，返回连接
```

连接归还流程（`db.putConn()`）：

```
1. 检查连接是否健康
2. 有等待请求？ → 直接交给等待者
3. 空闲池未满？ → 放入 freeConn
4. 否则关闭连接，numOpen--
```

### 2.2 四个调优参数

| 参数 | 默认值 | 作用 | 设置建议 |
|------|--------|------|---------|
| `SetMaxOpenConns(n)` | 0（无限制） | 最大同时打开的连接数 | **必须设置**，推荐 25-50 |
| `SetMaxIdleConns(n)` | 2 | 最大空闲连接数 | 设为 MaxOpenConns 的 50-100% |
| `SetConnMaxLifetime(d)` | 0（不限制） | 连接最大存活时间 | 设为 DB 的 wait_timeout 的 80% |
| `SetConnMaxIdleTime(d)` | 0（不限制） | 空闲连接最大保留时间 | 5-10 分钟 |

**常见错误配置：**

```go
// 错误：不设置 MaxOpenConns
// 高并发下可能打开数千连接，导致数据库 "too many connections"
db, _ := sql.Open("mysql", dsn)

// 错误：MaxIdleConns > MaxOpenConns
// MaxIdleConns 会被自动截断为 MaxOpenConns
db.SetMaxOpenConns(10)
db.SetMaxIdleConns(20) // 实际只有 10

// 错误：MaxIdleConns 过小
// 每次请求都要新建连接，三次握手开销大
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(2) // 默认值太小
```

**推荐的生产配置：**

```go
db.SetMaxOpenConns(25)                 // 根据数据库和应用负载调整
db.SetMaxIdleConns(25)                 // 等于 MaxOpenConns，避免频繁创建
db.SetConnMaxLifetime(5 * time.Minute) // 低于数据库 wait_timeout
db.SetConnMaxIdleTime(5 * time.Minute) // 及时释放空闲连接
```

### 2.3 连接池状态机

连接在池中的状态流转：

```
                    ┌─────────────┐
         sql.Open() │  Pool 创建   │ (0 个连接)
                    │ (懒初始化)   │
                    └──────┬──────┘
                           │ 首次查询
                           ▼
                    ┌─────────────┐
          ┌─────── │   空闲 idle  │ ◀──────────┐
          │         └──────┬──────┘             │
          │ connMaxIdle-   │ 被查询/事务获取     │ 查询完成
          │ Time 超时      ▼                    │ rows.Close()
          │         ┌─────────────┐             │
          │         │  使用中 in-use│ ───────────┘
          │         └──────┬──────┘
          │                │ connMaxLifetime 超时
          │                │ 或连接错误
          ▼                ▼
    ┌─────────────────────────┐
    │      已关闭 closed       │
    │   (numOpen--, 从池移除)   │
    └─────────────────────────┘
```

`database/sql` 内部有一个 `connectionCleaner` goroutine，定期扫描并关闭过期连接：

```go
// 源码简化
func (db *DB) connectionCleaner(d time.Duration) {
    // 定时器触发
    for {
        select {
        case <-t.C:
        case <-db.cleanerCh: // 配置变更时立即触发
        }
        db.mu.Lock()
        // 遍历 freeConn，关闭超过 maxLifetime 或 maxIdleTime 的连接
        for i, c := range db.freeConn {
            if c.createdAt + maxLifetime < now || c.returnedAt + maxIdleTime < now {
                closing = append(closing, c)
            }
        }
        db.mu.Unlock()
    }
}
```

### 2.4 db.Stats() 监控

```go
stats := db.Stats()
fmt.Printf("打开连接数: %d\n", stats.OpenConnections) // 当前打开的总连接数
fmt.Printf("使用中: %d\n", stats.InUse)               // 正在执行查询的连接
fmt.Printf("空闲: %d\n", stats.Idle)                   // 空闲连接数
fmt.Printf("等待次数: %d\n", stats.WaitCount)           // 因池满而等待的总次数
fmt.Printf("等待时长: %v\n", stats.WaitDuration)        // 等待的总时长
fmt.Printf("最大空闲关闭: %d\n", stats.MaxIdleClosed)    // 因超过 MaxIdleConns 而关闭的连接数
fmt.Printf("最大存活关闭: %d\n", stats.MaxLifetimeClosed) // 因超过 MaxLifetime 而关闭的连接数
```

生产环境建议定期采集 `db.Stats()` 并输出到监控系统：

```go
func monitorDBPool(db *sql.DB, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    for range ticker.C {
        stats := db.Stats()
        log.Printf("[DB Pool] open=%d inUse=%d idle=%d waitCount=%d waitDuration=%v",
            stats.OpenConnections, stats.InUse, stats.Idle,
            stats.WaitCount, stats.WaitDuration)
    }
}
```

**关键指标告警阈值：**
- `WaitCount` 持续增长 → MaxOpenConns 设置过小
- `InUse` 长期等于 `OpenConnections` → 连接可能泄漏
- `MaxIdleClosed` 很高 → MaxIdleConns 设置过小

---

## 3 CRUD 操作

### 3.1 查询操作

**多行查询 — db.Query()：**

```go
rows, err := db.QueryContext(ctx, "SELECT id, name, email FROM users WHERE age > ?", 18)
if err != nil {
    return fmt.Errorf("query users: %w", err)
}
defer rows.Close() // 必须在 err 检查之后

var users []User
for rows.Next() {
    var u User
    if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
        return fmt.Errorf("scan user: %w", err)
    }
    users = append(users, u)
}
// 必须检查迭代是否因错误终止
if err := rows.Err(); err != nil {
    return fmt.Errorf("rows iteration: %w", err)
}
```

**关键点：**
1. `rows` 持有底层连接，**必须 Close** 才能归还连接到池
2. `defer rows.Close()` 必须在 `err != nil` 检查**之后**（否则 rows 为 nil 时 panic）
3. `rows.Next()` 返回 false 不一定是数据结束，可能是错误——必须检查 `rows.Err()`

**单行查询 — db.QueryRow()：**

```go
var name string
err := db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = ?", 42).Scan(&name)
if err != nil {
    if errors.Is(err, sql.ErrNoRows) {
        // 没有匹配的行，不是错误，是正常业务情况
        return "", nil
    }
    return "", fmt.Errorf("query user: %w", err)
}
```

`QueryRow` 的优势：自动在 `Scan` 后关闭 `Rows`，不需要手动 `Close`，避免了忘记关闭的风险。

### 3.2 执行操作

```go
result, err := db.ExecContext(ctx,
    "INSERT INTO users (name, email) VALUES (?, ?)",
    "张三", "zhang@example.com")
if err != nil {
    return fmt.Errorf("insert user: %w", err)
}

// 获取自增 ID（注意：PostgreSQL 不支持，需用 RETURNING + QueryRow）
id, err := result.LastInsertId()
if err != nil {
    return fmt.Errorf("get insert id: %w", err)
}

// 获取影响行数
affected, err := result.RowsAffected()
if err != nil {
    return fmt.Errorf("get affected rows: %w", err)
}
fmt.Printf("插入成功, id=%d, 影响 %d 行\n", id, affected)
```

**重要：非 SELECT 语句（INSERT/UPDATE/DELETE）必须用 `Exec`，不要用 `Query`。** 用 `Query` 执行非 SELECT 语句会返回一个 `*Rows`，如果不消费和关闭，连接将泄漏。详见 `trap/query-for-exec/`。

### 3.3 Scan 的类型映射

Go 与 SQL 类型的对应关系：

| SQL 类型 | Go 类型 | NULL 处理 |
|----------|---------|----------|
| INTEGER | `int64` | `sql.NullInt64` 或 `*int64` |
| REAL/FLOAT | `float64` | `sql.NullFloat64` 或 `*float64` |
| TEXT/VARCHAR | `string` | `sql.NullString` 或 `*string` |
| BOOLEAN | `bool` | `sql.NullBool` 或 `*bool` |
| DATETIME | `time.Time` | `sql.NullTime` 或 `*time.Time` |
| BLOB | `[]byte` | 天然可为 nil |

**处理 NULL 的三种方式：**

```go
// 方式一：sql.Null* 类型
var name sql.NullString
err := row.Scan(&name)
if name.Valid {
    fmt.Println(name.String)
}

// 方式二：指针类型（更简洁）
var name *string
err := row.Scan(&name)
if name != nil {
    fmt.Println(*name)
}

// 方式三：Go 1.22+ 泛型 sql.Null[T]
var name sql.Null[string]
err := row.Scan(&name)
if name.Valid {
    fmt.Println(name.V)
}
```

### 3.4 Columns() 和动态扫描

当列数未知时，可以动态扫描：

```go
rows, err := db.QueryContext(ctx, query)
if err != nil {
    return err
}
defer rows.Close()

columns, err := rows.Columns()
if err != nil {
    return err
}

// 创建与列数等长的 interface{} 切片
values := make([]interface{}, len(columns))
valuePtrs := make([]interface{}, len(columns))
for i := range values {
    valuePtrs[i] = &values[i]
}

for rows.Next() {
    if err := rows.Scan(valuePtrs...); err != nil {
        return err
    }
    for i, col := range columns {
        fmt.Printf("%s = %v\n", col, values[i])
    }
}
return rows.Err()
```

---

## 4 事务

### 4.1 基本用法

```go
tx, err := db.BeginTx(ctx, nil) // nil 使用默认隔离级别
if err != nil {
    return err
}
// 关键：defer Rollback 保证异常路径回滚
defer tx.Rollback()

_, err = tx.ExecContext(ctx, "UPDATE accounts SET balance = balance - ? WHERE id = ?", amount, fromID)
if err != nil {
    return fmt.Errorf("debit: %w", err)
}

_, err = tx.ExecContext(ctx, "UPDATE accounts SET balance = balance + ? WHERE id = ?", amount, toID)
if err != nil {
    return fmt.Errorf("credit: %w", err)
}

// Commit 成功后，之前的 defer Rollback 会变成 no-op（对已提交的事务调用 Rollback 返回 ErrTxDone）
return tx.Commit()
```

### 4.2 关键模式：defer Rollback

**为什么 `defer tx.Rollback()` 是安全的？**

- 如果 `Commit()` 已执行成功，`Rollback()` 返回 `sql.ErrTxDone`，不做任何操作
- 如果中途发生 panic，`defer` 确保事务被回滚
- 如果 `Commit()` 之前 return 了错误，`defer` 确保回滚

```go
// 标准事务模板
func transferMoney(ctx context.Context, db *sql.DB, from, to int64, amount float64) (err error) {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback() // 安全网：Commit 后变为 no-op

    // ... 业务逻辑 ...

    return tx.Commit()
}
```

**反模式——不要在 Rollback 后检查错误并影响返回值：**

```go
// 错误：Rollback 在 Commit 之后执行时返回 ErrTxDone，不应覆盖返回值
defer func() {
    if err := tx.Rollback(); err != nil {
        log.Println("rollback failed:", err) // 这里会误报
    }
}()
```

### 4.3 隔离级别

```go
tx, err := db.BeginTx(ctx, &sql.TxOptions{
    Isolation: sql.LevelSerializable,
    ReadOnly:  true, // 只读事务，可能获得更好的性能
})
```

| 隔离级别 | 常量 | 脏读 | 不可重复读 | 幻读 |
|---------|------|------|-----------|------|
| Read Uncommitted | `sql.LevelReadUncommitted` | 可能 | 可能 | 可能 |
| Read Committed | `sql.LevelReadCommitted` | 不可能 | 可能 | 可能 |
| Repeatable Read | `sql.LevelRepeatableRead` | 不可能 | 不可能 | 可能 |
| Serializable | `sql.LevelSerializable` | 不可能 | 不可能 | 不可能 |

> 注意：实际行为取决于数据库引擎。MySQL 的 Repeatable Read 通过 MVCC 也能避免幻读；SQLite 只有 Serializable 和 Deferred/Immediate/Exclusive 三种模式。

### 4.4 事务中的查询

**事务绑定到单个连接。** 事务内的所有操作必须使用 `tx.Query/tx.Exec`，不能使用 `db.Query/db.Exec`：

```go
tx, _ := db.Begin()

// 正确：使用 tx 执行查询
rows, err := tx.QueryContext(ctx, "SELECT id FROM users WHERE status = ?", "active")

// 错误：使用 db 执行查询——这会使用另一个连接，看不到事务中的未提交更改！
// rows, err := db.QueryContext(ctx, "SELECT id FROM users WHERE status = ?", "active")
```

详见 `trap/tx-db-mixed/`。

---

## 5 预编译语句

### 5.1 Prepare 的工作原理

预编译语句将 SQL 解析和执行计划缓存在数据库服务端，之后只需传参数：

```
第一次: db.Prepare("SELECT * FROM users WHERE id = ?")
  → 驱动发送 PREPARE 命令到数据库
  → 数据库解析 SQL，生成执行计划，返回 statement ID
  → 返回 *sql.Stmt

后续调用: stmt.Query(42)
  → 驱动发送 EXECUTE 命令 + 参数
  → 数据库直接用缓存的执行计划，跳过解析
```

**在 `database/sql` 连接池层面的特殊行为：**

`*sql.Stmt` 不绑定到特定连接。当你调用 `stmt.Query()` 时：
1. 从池中获取一个连接
2. 检查该连接是否已经 PREPARE 过这条语句
3. 如果没有 → 在这个连接上重新执行 PREPARE
4. 执行 EXECUTE

这意味着同一个 `Stmt` 可能在多个连接上都做过 PREPARE。

### 5.2 连接重新 prepare

当连接因超时被关闭、新连接被创建时，`database/sql` 会在新连接上透明地重新 prepare。这个行为对用户透明，但带来额外开销：

```go
// 如果连接池中有 25 个连接，最坏情况下这条语句会在 25 个连接上各 prepare 一次
stmt, _ := db.Prepare("SELECT * FROM users WHERE id = ?")
```

### 5.3 何时使用 Prepare

**应该使用 Prepare 的场景：** 同一条 SQL 在循环或高频路径中反复执行：

```go
// 正确：循环外 Prepare，循环内只传参数
stmt, err := db.PrepareContext(ctx, "INSERT INTO logs (level, message) VALUES (?, ?)")
if err != nil {
    return err
}
defer stmt.Close()

for _, log := range logs {
    _, err := stmt.ExecContext(ctx, log.Level, log.Message)
    if err != nil {
        return fmt.Errorf("insert log: %w", err)
    }
}
```

**不需要 Prepare 的场景：** 一次性查询或低频操作：

```go
// 直接执行即可，不需要 Prepare 的开销
row := db.QueryRowContext(ctx, "SELECT count(*) FROM users")
```

**事务内的 Prepare：**

```go
tx, _ := db.BeginTx(ctx, nil)
defer tx.Rollback()

// 方式一：在事务内新建
stmt, err := tx.PrepareContext(ctx, "INSERT INTO orders (user_id, amount) VALUES (?, ?)")

// 方式二：从已有 Stmt 创建事务版本
existingStmt, _ := db.Prepare("INSERT INTO orders (user_id, amount) VALUES (?, ?)")
txStmt := tx.Stmt(existingStmt) // 复用 existingStmt 在当前事务连接上
```

### 5.4 Stmt 必须 Close

`Stmt.Close()` 的作用：
1. 释放 `database/sql` 内部对这条语句的跟踪
2. 在所有曾 prepare 过这条语句的连接上发送 DEALLOCATE

**不关闭 Stmt 会导致：**
- 数据库服务端预编译语句数量持续增长
- `database/sql` 内部 map 持续增长
- 连接上残留的 prepared statement 占用内存

详见 `trap/stmt-leak/`。

---

## 6 Context 集成

Go 1.8 为 `database/sql` 的所有操作添加了 `Context` 变体。

### 6.1 所有操作的 Context 变体

| 原始方法 | Context 变体 |
|---------|-------------|
| `db.Ping()` | `db.PingContext(ctx)` |
| `db.Query()` | `db.QueryContext(ctx, ...)` |
| `db.QueryRow()` | `db.QueryRowContext(ctx, ...)` |
| `db.Exec()` | `db.ExecContext(ctx, ...)` |
| `db.Prepare()` | `db.PrepareContext(ctx, ...)` |
| `db.Begin()` | `db.BeginTx(ctx, opts)` |

**建议：始终使用 Context 变体。** 无 Context 的版本内部使用 `context.Background()`，无法被取消。

### 6.2 超时控制

```go
// 单次查询 5 秒超时
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

rows, err := db.QueryContext(ctx, "SELECT * FROM large_table WHERE complex_condition = ?", val)
if err != nil {
    // 如果超时，err 会包含 context.DeadlineExceeded
    return fmt.Errorf("query: %w", err)
}
defer rows.Close()
```

### 6.3 取消传播

在 HTTP handler 中，请求的 context 天然支持取消传播：

```go
func getUserHandler(w http.ResponseWriter, r *http.Request) {
    // r.Context() 在客户端断开时自动取消
    ctx := r.Context()

    rows, err := db.QueryContext(ctx, "SELECT * FROM users WHERE department = ?", dept)
    if err != nil {
        if errors.Is(err, context.Canceled) {
            // 客户端已断开，无需返回响应
            return
        }
        http.Error(w, "查询失败", http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    // ...
}
```

### 6.4 注意事项

- **context 取消后连接的状态：** 当 context 被取消时，数据库驱动可能会将连接标记为 bad。`database/sql` 不会将 bad 连接放回池中，而是关闭它。高频取消（如客户端大量断开）可能导致连接频繁创建和销毁。

- **不要跨查询复用已超时的 context：**

```go
// 错误：context 可能在第一个查询后就超时了
ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
defer cancel()
db.QueryContext(ctx, query1) // 用了 1.5 秒
db.QueryContext(ctx, query2) // 只剩 0.5 秒，可能不够

// 正确：每个查询独立控制超时
ctx1, cancel1 := context.WithTimeout(ctx, 2*time.Second)
defer cancel1()
db.QueryContext(ctx1, query1)

ctx2, cancel2 := context.WithTimeout(ctx, 2*time.Second)
defer cancel2()
db.QueryContext(ctx2, query2)
```

---

## 7 生产最佳实践

### 7.1 连接池调优

经验公式（PostgreSQL 社区推荐）：

```
MaxOpenConns = (CPU 核心数 * 2) + 有效磁盘数
```

例如：4 核 CPU + 1 SSD = 4*2 + 1 = **9 个连接**。

> 这只是起始值。实际值需要根据查询复杂度、I/O 等待比例、外部服务调用等因素调整。

**完整的生产配置示例：**

```go
func newDB(dsn string) (*sql.DB, error) {
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }

    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(25)
    db.SetConnMaxLifetime(5 * time.Minute)
    db.SetConnMaxIdleTime(5 * time.Minute)

    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()
    if err := db.PingContext(ctx); err != nil {
        db.Close()
        return nil, fmt.Errorf("ping database: %w", err)
    }

    return db, nil
}
```

### 7.2 健康检查

```go
// 启动时验证连接
if err := db.Ping(); err != nil {
    log.Fatal("数据库不可达:", err)
}
```

**不要** 在每次查询前调用 `Ping()`——连接池会自动处理坏连接（检测到错误后丢弃，下次取新连接）。

### 7.3 优雅关闭

```go
func main() {
    db, _ := sql.Open("sqlite", "file:app.db")
    defer db.Close() // 等待所有活跃查询完成后关闭所有连接

    // 配合信号处理
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    // 启动服务，使用 ctx 控制生命周期
    srv := &http.Server{Addr: ":8080"}
    go func() {
        <-ctx.Done()
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        srv.Shutdown(shutdownCtx)
    }()
    srv.ListenAndServe()

    // 服务关闭后，defer db.Close() 执行
}
```

### 7.4 连接泄漏检测

连接泄漏的典型症状：`Stats().InUse` 持续增长，`WaitCount` 持续增长。

```go
func detectLeak(db *sql.DB) {
    prev := db.Stats()
    time.Sleep(30 * time.Second)
    curr := db.Stats()

    if curr.InUse > prev.InUse+5 {
        log.Printf("WARNING: 可能存在连接泄漏! InUse: %d → %d", prev.InUse, curr.InUse)
    }
    if curr.WaitCount > prev.WaitCount+100 {
        log.Printf("WARNING: 连接等待过多! WaitCount: %d → %d", prev.WaitCount, curr.WaitCount)
    }
}
```

### 7.5 错误处理

```go
// ErrNoRows 是业务信号，不是系统错误
var user User
err := db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = ?", id).Scan(&user.Name)
switch {
case errors.Is(err, sql.ErrNoRows):
    return nil, nil // 用户不存在
case err != nil:
    return nil, fmt.Errorf("query user %d: %w", id, err)
default:
    return &user, nil
}
```

**重试策略：** 网络错误（连接断开、超时）可以重试，但需注意：
- 写操作（INSERT/UPDATE/DELETE）重试需要幂等保证
- 使用 `context.WithTimeout` 控制总超时
- 不要无限重试，设置最大次数

---

## 8 常见陷阱

本节汇总了使用 `database/sql` 时最常见的错误。每个陷阱在 `trap/` 目录下有可运行的示例。

### 8.1 不关闭 Rows（trap/rows-not-closed/）

**现象：** 程序运行一段时间后查询卡死。
**原因：** `db.Query()` 返回的 `*Rows` 持有一个连接。不调用 `rows.Close()`，连接永远不会归还到池中。
**修复：** 始终 `defer rows.Close()`。

### 8.2 用 Query 执行非 SELECT（trap/query-for-exec/）

**现象：** 连接数持续增长。
**原因：** `db.Query("DELETE FROM ...")` 返回 `*Rows`，如果不消费和关闭，连接泄漏。
**修复：** 非 SELECT 语句使用 `db.Exec()`。

### 8.3 defer rows.Close() 放在错误检查之前（trap/defer-before-error-check/）

**现象：** 程序 panic: `nil pointer dereference`。
**原因：** 当 `db.Query()` 返回错误时，`rows` 为 nil，`defer rows.Close()` 解引用 nil 指针。
**修复：** 先检查错误，再 defer Close。

### 8.4 忽略 rows.Err()（trap/rows-err-ignored/）

**现象：** 查询结果不完整，但没有报错。
**原因：** `rows.Next()` 因错误返回 false 与正常结束返回 false 无法区分，必须检查 `rows.Err()`。
**修复：** 在 `for rows.Next()` 循环后检查 `rows.Err()`。

### 8.5 Stmt 泄漏（trap/stmt-leak/）

**现象：** 数据库报 prepared statement 数量超限。
**原因：** 循环中反复 `Prepare` 但不 `Close`，数据库和客户端都累积资源。
**修复：** `defer stmt.Close()` 或在循环外 Prepare、循环结束后 Close。

### 8.6 事务中混用 db 和 tx（trap/tx-db-mixed/）

**现象：** 事务中的查询看不到刚 INSERT 的数据。
**原因：** `db.Query()` 使用池中的另一个连接，不在事务中，看不到未提交的更改。
**修复：** 事务中所有操作使用 `tx.Query/tx.Exec`。

### 8.7 不限制最大连接数（trap/unlimited-connections/）

**现象：** 高并发时数据库报 "too many connections"。
**原因：** `MaxOpenConns` 默认为 0（无限制），高并发下每个 goroutine 都会打开一个新连接。
**修复：** 必须设置 `db.SetMaxOpenConns()`。
