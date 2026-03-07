
---
MySQL连接池深度调优与问题排查
---

- MySQL使用版本: 8.0.x
- Go版本: 1.24
- 驱动: github.com/go-sql-driver/mysql

# 1 为什么连接池至关重要

## 1.1 一次 TCP 连接的代价

每次从零创建一条 MySQL 连接，底层需要完成以下步骤：

1. TCP 三次握手（~1ms 同机房，跨区域可达数十ms）
2. MySQL 协议握手 + 认证（SSL 握手另加 ~2ms）
3. 服务端分配线程、初始化会话变量、分配内存（~0.5ms）
4. 执行查询
5. 四次挥手关闭连接

对于一次 1ms 的简单查询，连接建立和销毁的开销可能比查询本身还高。在高并发场景下（如 QPS > 1000），每次新建连接会迅速耗尽服务端的 `max_connections`，导致 `Too many connections` 拒绝服务。

## 1.2 连接池解决的核心问题

连接池在客户端维护一组已建立的空闲连接，当业务需要访问数据库时，直接从池中借用一个现有连接，用完归还。

|              | 无连接池 | 有连接池 |
| --- | --- | --- |
| 连接建立频率 | 每次查询都新建 | 仅在池为空时新建 |
| 服务端线程数 | 随请求数线性增长 | 受 MaxOpenConns 限制 |
| 响应延迟 | 查询延迟 + 连接延迟 | 仅查询延迟 |
| Too many connections 风险 | 高 | 低 |


# 2 双视角：服务端与客户端

连接池调优需要同时理解 MySQL 服务端和 Go 客户端两侧的参数，任何一侧的配置不当都会导致问题。

## 2.1 MySQL 服务端关键参数

```sql
-- 查看当前配置
SHOW VARIABLES LIKE 'max_connections';
SHOW VARIABLES LIKE 'wait_timeout';
SHOW VARIABLES LIKE 'interactive_timeout';
SHOW VARIABLES LIKE 'max_connect_errors';
SHOW VARIABLES LIKE 'thread_cache_size';
```

| 参数 | 默认值 | 含义 | 调优建议 |
| --- | --- | --- | --- |
| max_connections | 151 | 服务端允许的最大并发连接数 | 根据实例规格调整，通常 500-2000 |
| wait_timeout | 28800 (8h) | 非交互式连接的空闲超时（秒） | 生产环境建议 300-600 |
| interactive_timeout | 28800 (8h) | 交互式连接的空闲超时（秒） | 与 wait_timeout 保持一致 |
| max_connect_errors | 100 | 单主机连续连接失败次数上限，超过后拒绝该主机 | 10000+ 或配合 FLUSH HOSTS |
| thread_cache_size | -1 (auto) | 线程缓存大小，减少线程创建开销 | 通常 8-64，高并发可适当增大 |

**wait_timeout 的陷阱**：MySQL 服务端在连接空闲超过 `wait_timeout` 后，会**单方面关闭连接，但不会通知客户端**。如果客户端不知道连接已断，下一次使用该连接就会收到 `invalid connection` 或 `MySQL server has gone away` 错误。这是连接池调优中最常见的坑。

## 2.2 Go database/sql 连接池架构

Go 标准库 `database/sql` 内置了连接池，不需要第三方库。核心结构如下：

```
                  ┌──────────────────────────────────┐
                  │          sql.DB (连接池)           │
                  │                                    │
                  │  freeConn: []*driverConn  (空闲池) │
                  │  connRequests: chan connRequest     │
                  │  numOpen: int  (当前打开总数)       │
                  │  maxOpen: int  (MaxOpenConns)       │
                  │  maxIdle: int  (MaxIdleConns)       │
                  │  maxLifetime: time.Duration         │
                  │  maxIdleTime: time.Duration         │
                  └──────────────────────────────────┘
                        │           │           │
                   ┌────┴───┐ ┌────┴───┐ ┌────┴───┐
                   │ conn 1 │ │ conn 2 │ │ conn 3 │  ← TCP 连接
                   └────────┘ └────────┘ └────────┘
                        │           │           │
                   ┌────┴───────────┴───────────┴───┐
                   │        MySQL Server              │
                   └──────────────────────────────────┘
```

连接的生命周期：
1. `db.Query()` / `db.Exec()` 请求连接
2. 如果空闲池有连接 → 取出使用
3. 如果空闲池为空且 `numOpen < maxOpen` → 新建连接
4. 如果空闲池为空且 `numOpen >= maxOpen` → **阻塞等待**，直到有连接归还或 context 超时
5. 使用完毕 → 归还到空闲池（如果空闲池未满）或直接关闭

## 2.3 四个核心参数

```go
db, _ := sql.Open("mysql", dsn)

db.SetMaxOpenConns(50)                 // 最大打开连接数
db.SetMaxIdleConns(25)                 // 最大空闲连接数
db.SetConnMaxLifetime(5 * time.Minute) // 连接最大存活时间
db.SetConnMaxIdleTime(3 * time.Minute) // 连接最大空闲时间
```

| 参数 | 默认值 | 作用 | 不设的后果 |
| --- | --- | --- | --- |
| MaxOpenConns | 0 (无限制) | 限制与 MySQL 的并发连接总数 | 高并发下无限建连，撑爆 max_connections |
| MaxIdleConns | 2 | 空闲池保留的连接数 | 太小→频繁新建连接；太大→占用服务端资源 |
| ConnMaxLifetime | 0 (无限制) | 连接从**创建**到强制关闭的最大时长 | 遇到服务端 wait_timeout 后出现 invalid connection |
| ConnMaxIdleTime | 0 (无限制) | 连接空闲多久后被关闭 | 低峰期大量空闲连接长期占用服务端线程 |


# 3 参数调优方法论

## 3.1 MaxOpenConns：防止撑爆服务端

**原则**：所有客户端实例的 MaxOpenConns 总和 **必须小于** MySQL 的 max_connections。

假设：
- MySQL `max_connections = 1000`
- 应用有 10 个 Pod，每个 Pod 一个 `sql.DB` 实例
- 预留 100 个连接给监控、DBA 工具、主从复制等

计算：`(1000 - 100) / 10 = 90`，MaxOpenConns 设为 **80-90**。

```go
db.SetMaxOpenConns(80)
```

**验证**：

```sql
-- 查看服务端当前活跃连接数
SHOW STATUS LIKE 'Threads_connected';

-- 查看历史最大并发连接数（是否接近 max_connections）
SHOW STATUS LIKE 'Max_used_connections';

-- 查看被拒绝的连接数（不为 0 说明 max_connections 不够）
SHOW STATUS LIKE 'Aborted_connects';
```

## 3.2 MaxIdleConns：平衡复用率与资源占用

**原则**：MaxIdleConns 设为 MaxOpenConns 的 50%-100%。

| 流量模式 | MaxIdleConns 建议 | 理由 |
| --- | --- | --- |
| 稳定高流量 | = MaxOpenConns | 避免频繁创建销毁 |
| 波动型流量 | MaxOpenConns × 50-70% | 高峰复用，低谷释放 |
| 极低流量 | 5-10 | 保持少量热连接即可 |

**常见错误**：MaxIdleConns > MaxOpenConns 时，Go 会**静默**将 MaxIdleConns 截断为 MaxOpenConns。

```go
// 错误：MaxIdleConns 比 MaxOpenConns 大，被静默截断为 50
db.SetMaxOpenConns(50)
db.SetMaxIdleConns(100) // 实际生效的是 50，不会有任何警告
```

## 3.3 ConnMaxLifetime：对齐服务端超时

**核心原则**：ConnMaxLifetime **必须严格小于** MySQL 的 wait_timeout。

如果 ConnMaxLifetime >= wait_timeout，连接可能在客户端看来"还活着"，实际上服务端已单方面关闭了它。

```go
// MySQL wait_timeout = 300s
// ConnMaxLifetime 设为比 wait_timeout 短的值，预留安全边际
db.SetConnMaxLifetime(4 * time.Minute) // 240s < 300s ✓
```

**验证**：

```sql
-- 查看空闲连接在服务端的存活时间
SELECT id, user, host, db, command, time, state
FROM information_schema.processlist
WHERE command = 'Sleep'
ORDER BY time DESC;
```

如果看到大量 `time` 值接近或超过 `wait_timeout` 的 Sleep 连接，说明客户端的 ConnMaxLifetime 未正确设置。

## 3.4 ConnMaxIdleTime：低峰期资源回收

ConnMaxIdleTime 控制空闲连接多久后被关闭，主要用于低峰期释放不必要的连接。

```go
db.SetConnMaxIdleTime(3 * time.Minute)
```

**与 ConnMaxLifetime 的区别**：

| 维度 | ConnMaxLifetime | ConnMaxIdleTime |
| --- | --- | --- |
| 计时起点 | 连接**创建**时开始 | 连接**最后一次归还到空闲池**时开始 |
| 作用 | 防止使用服务端已关闭的死连接 | 低峰期回收不必要的空闲连接 |
| 高频连接 | 到期后强制关闭（即使一直在用） | 因为持续使用，不会触发 |
| 低频连接 | 同上 | 空闲一段时间后被回收 |

建议：ConnMaxIdleTime < ConnMaxLifetime，让空闲连接更早回收。


# 4 常见故障与排查

## 4.1 故障一：Too many connections

**现象**：应用报错 `Error 1040: Too many connections`

**排查步骤**：

```sql
-- Step 1: 确认当前连接数和上限
SHOW STATUS LIKE 'Threads_connected';
SHOW VARIABLES LIKE 'max_connections';

-- Step 2: 查看连接来源分布，定位哪个服务/账号占用最多
SELECT user, substring_index(host, ':', 1) as client_ip, db, command,
       count(*) as cnt
FROM information_schema.processlist
GROUP BY user, client_ip, db, command
ORDER BY cnt DESC;

-- Step 3: 查看长时间 Sleep 的连接（疑似泄漏或空闲未回收）
SELECT id, user, host, db, time, info
FROM information_schema.processlist
WHERE command = 'Sleep' AND time > 300
ORDER BY time DESC
LIMIT 20;
```

**常见原因与解决方案**：

| 原因 | 诊断依据 | 解决方案 |
| --- | --- | --- |
| 未设 MaxOpenConns | processlist 中单服务连接数远超预期 | 设置合理的 MaxOpenConns |
| 连接泄漏（Rows 未 Close） | Sleep 连接持续增长不释放 | 检查代码中未 Close 的 Rows/Stmt（详见 4.3） |
| Pod 扩容后连接总数超限 | Threads_connected ≈ max_connections | 重新计算 MaxOpenConns = (max_conn - 预留) / Pod 数 |
| 慢查询占用连接时间过长 | processlist 中大量 Query 状态连接 | 优化慢查询，缩短单连接占用时间 |

## 4.2 故障二：invalid connection / MySQL server has gone away

**现象**：应用**偶发**报错 `driver: bad connection` 或 `MySQL server has gone away`，通常在低峰期之后的第一波请求集中出现。

**根因**：客户端从空闲池取出的连接已被服务端关闭（wait_timeout 到期），但客户端不知道。

**排查**：

```sql
-- 确认服务端 wait_timeout
SHOW VARIABLES LIKE 'wait_timeout';

-- 查看被服务端主动关闭的连接数（如果持续增长，说明有空闲连接超时）
SHOW STATUS LIKE 'Aborted_clients';
```

同时检查应用代码中的 ConnMaxLifetime 配置。如果 ConnMaxLifetime 为 0 或大于 wait_timeout，就是这个问题。

**解决方案**：

```go
// 假设 wait_timeout = 300
db.SetConnMaxLifetime(4 * time.Minute) // 240s，留 60s 安全边际
```

**database/sql 的自动重试机制**：Go 1.9+ 的 `database/sql` 在检测到 `driver.ErrBadConn` 时，会自动丢弃该连接并重试（最多 `maxBadConnRetries = 2` 次）。但这存在两个局限：

1. 依赖驱动正确返回 `driver.ErrBadConn`，某些边缘场景下驱动可能返回其他错误
2. 如果池中连续多个连接都已死（低谷期全部超时），2 次重试仍然失败

因此，**不应依赖自动重试，而应通过 ConnMaxLifetime 从源头避免**。

## 4.3 故障三：连接泄漏

**现象**：Threads_connected 持续上升，即使流量恒定。最终触发 `Too many connections`。重启应用后恢复，过一段时间又复现。

**根因**：应用代码中未正确关闭 `*sql.Rows`，导致连接借出后无法归还。

**典型泄漏代码**：

```go
// 泄漏场景 1：忘记关闭 Rows
func leakRows(db *sql.DB) {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return
    }
    // 缺少 defer rows.Close()
    for rows.Next() {
        var id int
        rows.Scan(&id)
        if id > 100 {
            return // 提前退出循环，rows 未关闭，连接无法归还
        }
    }
}

// 泄漏场景 2：defer 写在 err 检查之前
func leakPanic(db *sql.DB) {
    rows, err := db.Query("SELECT id FROM users")
    defer rows.Close() // 如果 err != nil，rows 为 nil → panic
    if err != nil {
        return
    }
}

// 泄漏场景 3：Rows.Err() 未检查
func leakSilent(db *sql.DB) []int {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return nil
    }
    defer rows.Close()

    var ids []int
    for rows.Next() {
        var id int
        rows.Scan(&id)
        ids = append(ids, id)
    }
    // 缺少 rows.Err() 检查，如果迭代中途网络断开，错误被吞掉
    return ids
}
```

**唯一正确的写法**：

```go
func correctUsage(db *sql.DB) ([]int, error) {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return nil, err
    }
    defer rows.Close() // 必须在 err 检查之后

    var ids []int
    for rows.Next() {
        var id int
        if err := rows.Scan(&id); err != nil {
            return nil, err
        }
        ids = append(ids, id)
    }
    if err := rows.Err(); err != nil { // 检查迭代过程中是否有错误
        return nil, err
    }
    return ids, nil
}
```

**诊断连接泄漏**：

```go
stats := db.Stats()
log.Printf("Open=%d InUse=%d Idle=%d WaitCount=%d",
    stats.OpenConnections, stats.InUse, stats.Idle, stats.WaitCount)
```

| 指标 | 含义 | 泄漏信号 |
| --- | --- | --- |
| InUse | 正在使用的连接数 | 持续上升，不随请求结束下降 |
| Idle | 空闲连接数 | 接近 0 而 InUse 接近 MaxOpenConns |
| WaitCount | 等待获取连接的累计次数 | 快速增长 |
| WaitDuration | 等待获取连接的累计耗时 | 持续增大 |

**泄漏信号的典型特征**：InUse 持续增长而 Idle 持续为 0。正常情况下 InUse 应随请求波动，请求结束后下降。

## 4.4 故障四：连接风暴（Connection Storm）

**现象**：应用重启或流量突增时，短时间内大量新建连接，导致 MySQL CPU 飙升、响应变慢甚至触发 `max_connect_errors`。

**排查**：

```sql
-- 查看线程创建速率
SHOW STATUS LIKE 'Connections';       -- 累计建立的连接总数
SHOW STATUS LIKE 'Threads_created';   -- 累计创建的线程数
SHOW STATUS LIKE 'Threads_cached';    -- 当前缓存的线程数

-- 如果 Threads_created 增速远快于预期，说明在短时间内大量建连
```

**原因分析**：
1. 应用启动时池为空，第一波请求并发触发大量新建连接
2. 服务端 `thread_cache_size` 不够，每个新连接都要创建新线程
3. 所有 Pod 同时重启（如 K8s 滚动更新配置不当）

**解决方案**：

方案一：**启动时预热连接池**

```go
func warmupPool(db *sql.DB, n int) error {
    conns := make([]*sql.Conn, 0, n)
    ctx := context.Background()
    for i := 0; i < n; i++ {
        conn, err := db.Conn(ctx)
        if err != nil {
            break
        }
        conns = append(conns, conn)
    }
    for _, conn := range conns {
        conn.Close() // 归还到空闲池，而非关闭 TCP 连接
    }
    return nil
}

// 应用启动时调用
warmupPool(db, db.Stats().MaxOpenConnections/2)
```

方案二：**服务端增大 thread_cache_size**

```sql
SET GLOBAL thread_cache_size = 64;
```

方案三：**K8s 滚动重启策略**，确保 `maxUnavailable=1`，避免所有 Pod 同时重建连接


# 5 监控体系

## 5.1 服务端监控指标

```sql
-- 核心指标
SHOW STATUS LIKE 'Threads_connected';      -- 当前连接数
SHOW STATUS LIKE 'Threads_running';        -- 当前活跃线程数（正在执行查询）
SHOW STATUS LIKE 'Max_used_connections';   -- 历史最高连接数
SHOW STATUS LIKE 'Aborted_connects';       -- 连接建立失败次数（客户端问题）
SHOW STATUS LIKE 'Aborted_clients';        -- 被服务端关闭的连接次数（超时/异常）
```

**告警规则**：

| 指标 | 告警阈值 | 含义 |
| --- | --- | --- |
| Threads_connected / max_connections | > 80% | 连接数即将耗尽 |
| Threads_running | > CPU 核数 × 2 | 活跃线程过多，可能有慢查询 |
| Aborted_clients | 5分钟增量 > 50 | 大量连接被服务端超时关闭 |
| Aborted_connects | 5分钟增量 > 0 | 有客户端连接失败 |

## 5.2 客户端监控指标（Go DBStats）

```go
stats := db.Stats()
```

| 指标 | 告警阈值建议 |
| --- | --- |
| OpenConnections | > MaxOpenConns × 90% |
| InUse | 持续 > MaxOpenConns × 80% |
| WaitCount | 5分钟增量 > 100 |
| WaitDuration / WaitCount | 平均等待 > 100ms |
| MaxIdleClosed | 5分钟增量异常飙升（空闲回收过于频繁） |
| MaxLifetimeClosed | 5分钟增量异常飙升（生命周期设置过短） |

## 5.3 Prometheus 接入示例

```go
var (
    dbOpenConns = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "db_open_connections",
        Help: "Number of open connections to the database",
    })
    dbInUse = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "db_in_use_connections",
        Help: "Number of connections currently in use",
    })
    dbWaitTotal = promauto.NewCounter(prometheus.CounterOpts{
        Name: "db_wait_total",
        Help: "Total number of times a connection was waited for",
    })
)

func collectDBMetrics(db *sql.DB) {
    ticker := time.NewTicker(15 * time.Second)
    for range ticker.C {
        stats := db.Stats()
        dbOpenConns.Set(float64(stats.OpenConnections))
        dbInUse.Set(float64(stats.InUse))
        dbWaitTotal.Add(float64(stats.WaitCount))
    }
}
```

关键 Grafana 面板：
- **连接数三线图**：OpenConnections / InUse / Idle 随时间变化
- **等待排队趋势**：WaitCount 增量 + 平均 WaitDuration
- **连接关闭原因分布**：MaxIdleClosed / MaxLifetimeClosed / MaxIdleTimeClosed


# 6 实战案例分析

## case 1: 未设置 MaxOpenConns 导致雪崩

**背景**：某服务使用默认配置（MaxOpenConns=0），正常运行数月。某天下游表缺少索引，单次查询从 5ms 上升到 800ms。

**故障链**：
1. 慢查询导致每个连接的占用时间从 5ms → 800ms，归还速度急剧下降
2. 连接池无上限，`database/sql` 持续新建连接，10 秒内 numOpen 从 50 → 900+
3. MySQL `Threads_connected` 逼近 `max_connections(1000)`
4. 共享同一 MySQL 实例的其他服务全部报 `Too many connections`
5. 全站级联故障

**排查命令**：

```sql
SHOW STATUS LIKE 'Threads_connected';
-- 结果: 987

SELECT user, count(*) as cnt
FROM information_schema.processlist
GROUP BY user ORDER BY cnt DESC;
-- 结果: 问题服务账号占 920 个连接

-- 查看这些连接在执行什么
SELECT info, count(*) as cnt
FROM information_schema.processlist
WHERE user = 'app_user' AND command = 'Query'
GROUP BY info ORDER BY cnt DESC LIMIT 5;
-- 结果: 全是同一条慢查询
```

**修复**：

```go
db.SetMaxOpenConns(80)
db.SetMaxIdleConns(40)
db.SetConnMaxLifetime(4 * time.Minute)
```

**教训**：`MaxOpenConns = 0`（默认值）在生产环境是危险的。慢查询 + 无上限连接池 = 雪崩。MaxOpenConns 是连接池的**熔断器**，它限制了故障的爆炸半径。

## case 2: ConnMaxLifetime 与 wait_timeout 不匹配

**背景**：某服务配置了 `MaxOpenConns=50, MaxIdleConns=50`，但未设 ConnMaxLifetime。MySQL `wait_timeout=300`。服务在凌晨低谷期后的第一波早高峰请求集中报错 `invalid connection`。

**故障链**：
1. 凌晨流量极低，50 个连接长时间空闲在客户端池中
2. 超过 300 秒后，MySQL 服务端单方面关闭全部 50 个连接
3. 早高峰第一波请求从池中取出这些已死连接
4. 驱动发送查询后收到 TCP RST，返回 `driver: bad connection`
5. `database/sql` 自动重试（最多 2 次），但池中连续多个连接都已死，连续重试仍失败

**验证**：

```sql
-- Aborted_clients 在凌晨时段持续增长
SHOW STATUS LIKE 'Aborted_clients';

-- 查看空闲连接已超过 wait_timeout
SELECT id, user, time FROM information_schema.processlist
WHERE command = 'Sleep' ORDER BY time DESC;
-- 结果: time 列均接近或超过 300
```

**修复**：

```go
db.SetConnMaxLifetime(4 * time.Minute)  // 240s < 300s(wait_timeout)
db.SetConnMaxIdleTime(2 * time.Minute)  // 低谷期更快回收空闲连接
```

修复后，连接在空闲 240 秒后由客户端主动关闭并替换，永远不会等到服务端的 300 秒超时。

## case 3: Rows 未关闭导致连接泄漏

**背景**：某服务上线后运行正常，但每隔 12-24 小时就会出现所有数据库请求超时。重启后恢复，过一段时间又复现。

**诊断过程**：

```go
// 定时输出 DBStats
stats := db.Stats()
log.Printf("Open=%d InUse=%d Idle=%d Wait=%d",
    stats.OpenConnections, stats.InUse, stats.Idle, stats.WaitCount)
```

观察到的数据变化：

```
// 上线 1 小时后
Open=30 InUse=12 Idle=18 Wait=0        // 正常

// 上线 6 小时后
Open=50 InUse=42 Idle=8  Wait=156      // InUse 持续升高

// 上线 12 小时后
Open=50 InUse=50 Idle=0  Wait=89432    // 池满，所有请求排队等待
```

同时在 MySQL 侧查看：

```sql
SELECT command, count(*) FROM information_schema.processlist
WHERE user = 'app_user' GROUP BY command;
-- 结果:
-- Sleep   50
-- Query   0
```

**关键发现**：客户端 InUse=50（全部"在使用"），但 MySQL 侧全部是 Sleep（没有活跃查询）。这说明客户端认为连接在使用中（未归还），但实际上没有任何查询在执行。这是**连接泄漏**的典型表现。

代码审查发现问题函数：

```go
func searchUsers(db *sql.DB, keyword string) ([]User, error) {
    rows, err := db.Query("SELECT id, name FROM users WHERE name LIKE ?",
        "%"+keyword+"%")
    if err != nil {
        return nil, err
    }
    // 缺少 defer rows.Close()

    var users []User
    for rows.Next() {
        var u User
        if err := rows.Scan(&u.ID, &u.Name); err != nil {
            return nil, err // 提前 return，rows 未关闭
        }
        users = append(users, u)
    }
    return users, nil
}
```

**修复**：在 `if err != nil` 之后加上 `defer rows.Close()`。

## case 4: 配置参考表

| 场景 | MaxOpenConns | MaxIdleConns | ConnMaxLifetime | ConnMaxIdleTime |
| --- | --- | --- | --- | --- |
| 高频 OLTP (QPS>1000) | 80-100 | 60-80 | 4min | 3min |
| 普通 Web (QPS 100-500) | 30-50 | 15-30 | 5min | 3min |
| 低频后台任务 | 10-20 | 5-10 | 5min | 2min |
| 定时 Cron 任务 | 5-10 | 2-5 | 5min | 1min |

以上假设 MySQL `wait_timeout=300`。如果 wait_timeout 不同，需相应调整 ConnMaxLifetime 使其严格小于 wait_timeout。


# 7 进阶：连接池源码中的关键设计

## 7.1 连接获取的完整流程

`database/sql` 的连接获取逻辑（简化）：

```
db.conn(ctx, strategy) →
  ├── 检查 freeConn 空闲池
  │     ├── 有空闲连接 → 检查是否过期(lifetime/idletime) → 未过期则返回
  │     └── 无空闲连接 ↓
  ├── numOpen < maxOpen ?
  │     ├── 是 → 新建连接 → 返回
  │     └── 否 ↓
  └── 创建 connRequest 放入等待队列
        └── 阻塞等待，直到：
              ├── 有连接归还 → 唤醒
              ├── ctx 超时 → 返回 context.DeadlineExceeded
              └── db 关闭 → 返回 ErrDBClosed
```

**关键隐患**：当 MaxOpenConns 已满时，后续请求会阻塞等待。如果请求的 context 没有设置超时，会**无限阻塞**。这在连接泄漏场景下尤其致命 — 泄漏导致池满，池满导致所有请求永久阻塞。

```go
// 危险：无超时查询，连接池满时无限阻塞
rows, err := db.Query("SELECT ...")

// 安全：带超时查询，连接池满时最多等 3 秒
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()
rows, err := db.QueryContext(ctx, "SELECT ...")
```

## 7.2 连接健康检查

`database/sql` **不会在取出空闲连接时主动 ping**（出于性能考虑），而是通过以下机制间接保障连接可用：

1. **ConnMaxLifetime**：取出时检查创建时间，过期则丢弃
2. **ConnMaxIdleTime**：取出时检查空闲时间，过期则丢弃
3. **ErrBadConn 重试**：驱动返回坏连接错误时，自动重试最多 2 次

这意味着在极端场景下（如 MySQL 重启、网络闪断），池中所有连接可能同时失效。应用层应做好错误处理，配合 context 超时使用。

`db.Ping()` / `db.PingContext()` 可以在应用启动时验证连接可用性：

```go
db, _ := sql.Open("mysql", dsn)
if err := db.PingContext(ctx); err != nil {
    log.Fatalf("数据库连接失败: %v", err)
}
```

但 `Ping` 不应被用于定期健康检查（有 ConnMaxLifetime 即可），否则会产生不必要的网络开销。

## 7.3 connectionCleaner 后台协程

`database/sql` 会启动一个后台 goroutine（connectionCleaner），定期清理过期连接：

- 检查间隔 = `min(ConnMaxLifetime, ConnMaxIdleTime)`
- 遍历 freeConn，关闭超过 ConnMaxLifetime 或 ConnMaxIdleTime 的连接
- **仅清理空闲池中的连接，不会中断正在使用的连接**

这解释了为什么 ConnMaxLifetime 是"软限制"：一个连接如果一直在被使用（未归还），即使超过了 ConnMaxLifetime，也不会被强制关闭。只有在归还后，下一次被取出时才会检查并丢弃。

如果 ConnMaxLifetime 和 ConnMaxIdleTime 都为 0（默认值），connectionCleaner **不会启动**，空闲池中的连接永远不会被主动清理。


# 8 总结

## 8.1 配置 Checklist

| # | 检查项 | 通过标准 |
| --- | --- | --- |
| 1 | MaxOpenConns 已设置且 > 0 | 所有客户端总和 < max_connections |
| 2 | MaxIdleConns 已设置 | MaxIdleConns <= MaxOpenConns |
| 3 | ConnMaxLifetime 已设置 | **严格小于** MySQL wait_timeout |
| 4 | ConnMaxIdleTime 已设置 | 小于 ConnMaxLifetime |
| 5 | 所有 db.Query 返回的 Rows 有 defer Close | 在 err 检查**之后** defer |
| 6 | 查询使用 context 超时 | QueryContext / ExecContext |
| 7 | DBStats 已接入监控 | Grafana 面板可查看 InUse/Idle/WaitCount |
| 8 | rows.Err() 已检查 | 迭代结束后检查错误 |

## 8.2 一句话记忆

> **连接池调优 = 设上限（MaxOpenConns）+ 控生命周期（ConnMaxLifetime < wait_timeout）+ 防泄漏（defer rows.Close()）+ 可观测（DBStats 监控）**
