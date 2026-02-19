
---
MySQL是如何实现事务ACID特性的？
---

本文重点讲解 InnoDB 如何通过 undo log、redo log、锁、MVCC 等机制协同实现事务的 ACID 四大特性，
以及它们之间的依赖关系。建议先阅读第 09 篇《详解 binlog 日志、undo 日志和 redo 日志》了解各日志的底层机制，
本文在此基础上聚焦"它们如何为 ACID 服务"。


# 1 ACID 全景：四大特性与实现机制的映射

先给出全局视图，后续章节逐一深入：

```
┌──────────────────────────────────────────────────────────┐
│                      一致性 (Consistency)                  │
│        "最终目标"——数据库从一个合法状态到另一个合法状态        │
│                                                          │
│   ┌─────────────┐  ┌─────────────┐  ┌──────────────┐    │
│   │  原子性 (A)  │  │  隔离性 (I)  │  │  持久性 (D)  │    │
│   │  undo log   │  │  MVCC + 锁  │  │  redo log    │    │
│   │  回滚能力   │  │  并发控制    │  │  WAL 持久化  │    │
│   └─────────────┘  └─────────────┘  └──────────────┘    │
│          ↑                ↑                ↑             │
│          └────── A + I + D 共同保障 C ──────┘             │
└──────────────────────────────────────────────────────────┘
```

核心关系：
- **原子性**由 undo log 保障（回滚能力）。
- **持久性**由 redo log 保障（WAL + 刷盘策略）。
- **隔离性**由 MVCC（读）+ 锁（写）共同保障。
- **一致性**是最终目标，由 A + I + D 三者加上数据库约束共同保障。

**追问：为什么说一致性是"目标"而不是"手段"？**

因为一致性是一个业务层面的概念——数据满足所有业务规则和约束。原子性、隔离性、持久性是数据库
提供的技术手段，它们共同确保一致性不被破坏。没有原子性，部分执行会破坏一致性；没有隔离性，
并发冲突会破坏一致性；没有持久性，已提交的正确状态可能丢失。


# 2 原子性（Atomicity）—— undo log 的回滚能力

## 2.1 核心机制

原子性的含义是：事务中的所有操作，要么全部生效，要么全部不生效（回滚到事务开始前的状态）。

InnoDB 通过 **undo log** 实现原子性。事务执行过程中，每次修改数据之前，都会先将"反向操作"
记录到 undo log 中：

```
┌────────────────────────────────────────────────────────────┐
│                    事务执行过程                              │
│                                                            │
│  INSERT INTO t VALUES(1, 'a')                              │
│       → undo log 记录: DELETE FROM t WHERE id = 1          │
│                                                            │
│  UPDATE t SET name = 'b' WHERE id = 2                      │
│       → undo log 记录: UPDATE t SET name = 'old' WHERE id=2│
│                                                            │
│  DELETE FROM t WHERE id = 3                                │
│       → undo log 记录: INSERT INTO t VALUES(3, 'c')        │
└────────────────────────────────────────────────────────────┘
```

当事务需要回滚时，InnoDB 沿着 undo log 链逆序执行这些反向操作，将数据恢复到事务开始前的状态。

## 2.2 两种 undo log 类型

| 类型 | 触发操作 | 事务提交后是否可删除 | 原因 |
|------|---------|---------------------|------|
| insert undo log | INSERT | 立即可删除 | 新插入的行对其他事务不可见，无需为 MVCC 保留 |
| update undo log | UPDATE / DELETE | 不能立即删除 | 其他事务可能正在通过 MVCC 读取旧版本 |

**追问：为什么 insert undo log 提交后可以立即删除？**

因为 INSERT 操作产生的是一条全新的记录，在事务提交之前，这条记录对其他事务是不可见的
（由隔离性保证）。事务提交后，回滚不再需要，也不存在其他事务需要读取"插入前的版本"
（插入前根本不存在），所以 insert undo log 可以安全删除。

而 UPDATE/DELETE 的 undo log 必须保留，因为其他事务可能正在通过 MVCC 的版本链读取旧版本数据，
必须等到没有任何活跃事务需要这些旧版本后，才能由 purge 线程清理。

## 2.3 回滚的具体执行流程

```sql
-- 演示回滚过程
BEGIN;
UPDATE account SET balance = balance - 100 WHERE id = 1;  -- undo: balance + 100
UPDATE account SET balance = balance + 100 WHERE id = 2;  -- undo: balance - 100
-- 假设此处应用检测到业务异常
ROLLBACK;
-- InnoDB 按 undo log 逆序执行：
-- 1. UPDATE account SET balance = balance - 100 WHERE id = 2  (撤销第二步)
-- 2. UPDATE account SET balance = balance + 100 WHERE id = 1  (撤销第一步)
-- 数据恢复到事务开始前的状态
```

## 2.4 常见误解：redo log 保障原子性？

这是一个常见误区。redo log 的职责是**持久性**，不是原子性。理由如下：

- redo log 记录的是"物理修改"（某页某偏移量写入了什么值），它只会"重做"，不会"撤销"。
- 如果事务执行了一半崩溃了，redo log 不能回滚已执行的部分——它反而会把这些部分重做出来。
- 崩溃恢复时，redo log 重做 + undo log 回滚，二者配合才能保证原子性：
  redo 先恢复所有数据页到崩溃前的状态，然后 undo 将未提交事务回滚。

```
崩溃恢复流程：

1. redo log 前滚（重做）
   → 将所有已写入 redo log 的修改重做到数据页
   → 此时数据页中可能包含未提交事务的修改

2. undo log 回滚
   → 检查哪些事务未提交（没有 commit 标记）
   → 利用 undo log 将这些事务的修改撤销

结果：已提交事务的修改被保留，未提交事务的修改被撤销 → 原子性 + 持久性
```

**追问：为什么崩溃恢复要先 redo 再 undo，不能直接 undo？**

因为崩溃时，Buffer Pool 中的脏页可能只有一部分刷到了磁盘。如果不先 redo，磁盘上的数据页
状态是不完整的——有些页已刷盘，有些没有。undo log 的回滚操作需要操作完整的数据页，
所以必须先通过 redo log 将所有数据页恢复到崩溃前的状态，然后才能正确地执行 undo 回滚。


# 3 持久性（Durability）—— redo log 的 WAL 机制

## 3.1 核心机制

持久性的含义是：事务一旦提交，其修改的数据就不会丢失，即使数据库崩溃也能恢复。

InnoDB 通过 **WAL（Write-Ahead Logging）** 机制实现持久性：事务提交时，只需将 redo log
刷入磁盘，而不需要等数据页刷盘。由于 redo log 是顺序写入，比数据页的随机写入快得多。

```
┌─────────────────────────────────────────────────────┐
│                    WAL 写入流程                       │
│                                                     │
│  事务修改数据                                        │
│      │                                              │
│      ▼                                              │
│  ① 修改 Buffer Pool 中的数据页（内存）                │
│      │                                              │
│      ▼                                              │
│  ② 将修改写入 redo log buffer（内存）                 │
│      │                                              │
│      ▼                                              │
│  ③ 事务提交时，redo log buffer → 刷入磁盘（fsync）    │  ← 持久性的关键
│      │                                              │
│      ▼                                              │
│  ④ 数据页在后台异步刷盘（Checkpoint）                 │  ← 不影响持久性
│                                                     │
│  如果在④之前崩溃：                                   │
│  → 重启后通过 redo log 重做，恢复数据页               │
└─────────────────────────────────────────────────────┘
```

## 3.2 innodb_flush_log_at_trx_commit 参数

这个参数直接决定了持久性的强度：

| 值 | 行为 | 持久性 | 性能 |
|----|------|--------|------|
| 0 | 每秒将 log buffer 写入 OS cache 并 fsync | 可能丢失最近 1 秒数据 | 最快 |
| 1 | **每次事务提交都 fsync 到磁盘** | **不丢数据** | 最慢 |
| 2 | 每次事务提交写入 OS cache，每秒 fsync | OS 崩溃时可能丢 1 秒数据 | 中等 |

```sql
-- 查看当前设置
SHOW VARIABLES LIKE 'innodb_flush_log_at_trx_commit';

-- 生产环境推荐设置
SET GLOBAL innodb_flush_log_at_trx_commit = 1;
```

**只有设置为 1 时，才能保证严格的持久性。** 值为 0 或 2 时，存在数据丢失窗口。

**追问：设置为 1 性能是否太差？**

确实会降低吞吐量，因为每次提交都要等 fsync 完成。但这是持久性的代价。
可以通过以下方式缓解：
- **组提交（Group Commit）**：多个事务的 redo log 合并一次 fsync，大幅降低 I/O 次数。
  MySQL 5.6+ 的 binlog 组提交已经将 fsync 的开销降低了很多。
- **高性能存储**：使用 NVMe SSD，fsync 延迟从毫秒级降到微秒级。
- **适当调大 redo log 文件**：减少 Checkpoint 频率，让后台刷脏更从容。

## 3.3 Doublewrite Buffer —— 防止 partial page write

持久性还面临一个威胁：**部分页写入（partial page write）**。InnoDB 的数据页是 16KB，
而文件系统通常以 4KB 为单位写入。如果写到一半崩溃了，页面就会处于一个"半新半旧"的损坏状态。
此时 redo log 也无法修复，因为 redo log 记录的是对完整页面的修改，页面本身已损坏。

**Doublewrite Buffer 的解决方案：**

```
┌──────────────────────────────────────────────────────┐
│              Doublewrite Buffer 写入流程               │
│                                                      │
│  脏页需要刷盘                                         │
│      │                                               │
│      ▼                                               │
│  ① 先将脏页写入 Doublewrite Buffer（磁盘上的连续区域）  │
│      │                                               │
│      ▼                                               │
│  ② 确认写入完成后，再写入实际的数据文件                  │
│                                                      │
│  如果②过程中崩溃：                                    │
│  → Doublewrite Buffer 中有完整副本，用它恢复数据页      │
│  → 然后再用 redo log 继续重做                          │
└──────────────────────────────────────────────────────┘
```

```sql
-- 查看 Doublewrite Buffer 状态
SHOW VARIABLES LIKE 'innodb_doublewrite';

-- 查看 Doublewrite 的使用统计
SHOW STATUS LIKE 'Innodb_dblwr%';
```

**追问：使用支持原子写入的文件系统（如 ZFS）或 NVMe SSD，还需要 Doublewrite 吗？**

如果存储设备能保证 16KB 的原子写入，确实可以关闭 Doublewrite 以提升性能。
MySQL 8.0.20+ 允许通过 `innodb_doublewrite=OFF` 关闭。但要谨慎验证硬件特性。


# 4 隔离性（Isolation）—— MVCC + 锁的协同

## 4.1 核心机制

隔离性的含义是：并发执行的事务之间互不干扰，每个事务看到的数据视图是一致的。

InnoDB 通过两种机制实现隔离性：
- **MVCC（多版本并发控制）**：处理读-写冲突，读不加锁。
- **锁机制**：处理写-写冲突，保证并发写入的正确性。

```
┌─────────────────────────────────────────────────────┐
│                  隔离性实现机制                       │
│                                                     │
│  读-写并发           写-写并发                        │
│      │                   │                          │
│      ▼                   ▼                          │
│    MVCC                锁机制                        │
│  （快照读）         （当前读）                        │
│      │                   │                          │
│      ▼                   ▼                          │
│  undo log            Record Lock                    │
│  版本链              Gap Lock                        │
│  ReadView            Next-Key Lock                   │
│      │                   │                          │
│      ▼                   ▼                          │
│  不加锁、不阻塞      互斥等待                         │
│  高并发读            防止脏写/丢失更新                 │
└─────────────────────────────────────────────────────┘
```

## 4.2 MVCC 如何实现"读不加锁"

InnoDB 中每行记录都有两个隐藏列：

| 隐藏列 | 含义 |
|--------|------|
| DB_TRX_ID | 最后一次修改该行的事务 ID |
| DB_ROLL_PTR | 回滚指针，指向 undo log 中该行的上一个版本 |

多次修改同一行会形成一条**版本链**：

```
当前行: name='Charlie', DB_TRX_ID=300
    │
    ▼ (DB_ROLL_PTR)
undo log: name='Bob', DB_TRX_ID=200
    │
    ▼ (DB_ROLL_PTR)
undo log: name='Alice', DB_TRX_ID=100
```

事务在执行 SELECT 时（快照读），会创建一个 **ReadView**，其中记录：

```
ReadView {
    m_ids:        [200, 300]    // 创建 ReadView 时所有活跃（未提交）事务的 ID 列表
    min_trx_id:   200           // m_ids 中的最小值
    max_trx_id:   301           // 下一个将分配的事务 ID（当前最大 ID + 1）
    creator_trx_id: 400         // 创建该 ReadView 的事务自己的 ID
}
```

可见性判断规则：

```
对于版本链中的某个版本，其 DB_TRX_ID = trx_id：

1. trx_id < min_trx_id
   → 该版本在 ReadView 创建前已提交 → 可见

2. trx_id >= max_trx_id
   → 该版本在 ReadView 创建后才出现 → 不可见

3. min_trx_id <= trx_id < max_trx_id
   → 检查 trx_id 是否在 m_ids 中：
     - 在 m_ids 中 → 该事务未提交 → 不可见
     - 不在 m_ids 中 → 该事务已提交 → 可见

4. trx_id == creator_trx_id
   → 自己的修改 → 可见
```

```sql
-- 演示 MVCC 快照读

-- Session A (trx_id = 100)
BEGIN;
SELECT name FROM users WHERE id = 1;  -- 读到 'Alice'

-- Session B (trx_id = 200)
BEGIN;
UPDATE users SET name = 'Bob' WHERE id = 1;
COMMIT;

-- Session A (仍然读到 'Alice'，因为 ReadView 创建时 trx_id=200 还未提交)
SELECT name FROM users WHERE id = 1;  -- 仍然是 'Alice'（RR 隔离级别下）
COMMIT;
```

## 4.3 RC 与 RR 的 ReadView 差异

| 隔离级别 | ReadView 创建时机 | 效果 |
|---------|-------------------|------|
| READ COMMITTED (RC) | **每次** SELECT 都创建新的 ReadView | 能看到其他事务最新提交的数据 |
| REPEATABLE READ (RR) | 事务中**第一次** SELECT 时创建，后续复用 | 整个事务看到一致的快照 |

这解释了为什么 RR 隔离级别下，同一事务内多次读取结果一致——因为使用的是同一个 ReadView。

**追问：RR 隔离级别下，InnoDB 能否防止幻读？**

部分能。
- **快照读**（普通 SELECT）：通过 MVCC 的 ReadView 机制，可以防止幻读，因为看到的始终是同一个快照。
- **当前读**（SELECT ... FOR UPDATE / INSERT / UPDATE / DELETE）：通过 **Next-Key Lock**
 （Record Lock + Gap Lock）防止幻读——锁住记录和记录之间的间隙，阻止其他事务在该范围内插入新记录。

```sql
-- 当前读防止幻读示例

-- Session A
BEGIN;
SELECT * FROM users WHERE age > 20 FOR UPDATE;
-- 此时 InnoDB 不仅锁住所有 age > 20 的行（Record Lock），
-- 还锁住 age > 20 的间隙（Gap Lock），
-- 合称 Next-Key Lock

-- Session B
INSERT INTO users (name, age) VALUES ('Dave', 25);
-- 阻塞！因为 age=25 落在 Session A 锁住的间隙中

-- Session A
SELECT * FROM users WHERE age > 20 FOR UPDATE;
-- 结果不变，没有幻读
COMMIT;
```

## 4.4 锁机制概览

InnoDB 的锁按粒度和用途分类：

```
┌──────────────────────────────────────────────────────────┐
│                     InnoDB 锁分类                         │
│                                                          │
│  按粒度：                                                 │
│  ├── 表级锁：意向共享锁(IS)、意向排他锁(IX)               │
│  └── 行级锁：                                             │
│      ├── Record Lock  —— 锁单条记录                      │
│      ├── Gap Lock     —— 锁记录之间的间隙                 │
│      └── Next-Key Lock —— Record + Gap（左开右闭区间）    │
│                                                          │
│  按模式：                                                 │
│  ├── 共享锁 (S Lock)  —— 读锁，允许其他事务加 S 锁       │
│  └── 排他锁 (X Lock)  —— 写锁，不允许其他事务加任何锁    │
│                                                          │
│  兼容矩阵：                                               │
│          S Lock    X Lock                                │
│  S Lock    ✓         ✗                                   │
│  X Lock    ✗         ✗                                   │
└──────────────────────────────────────────────────────────┘
```

```sql
-- 查看当前锁等待情况
SELECT * FROM performance_schema.data_lock_waits;

-- 查看当前持有的锁
SELECT * FROM performance_schema.data_locks;

-- 查看 InnoDB 锁等待状态
SHOW ENGINE INNODB STATUS\G
```

## 4.5 四种隔离级别对比

| 隔离级别 | 脏读 | 不可重复读 | 幻读 | 实现方式 | 性能 |
|---------|------|-----------|------|---------|------|
| READ UNCOMMITTED | 可能 | 可能 | 可能 | 直接读最新数据，不用 MVCC | 最高 |
| READ COMMITTED | 不可能 | 可能 | 可能 | 每次 SELECT 新建 ReadView | 高 |
| **REPEATABLE READ** | 不可能 | 不可能 | 快照读不可能，当前读靠 Next-Key Lock | 首次 SELECT 建 ReadView + Next-Key Lock | **默认** |
| SERIALIZABLE | 不可能 | 不可能 | 不可能 | 所有 SELECT 自动加 S Lock | 最低 |

```sql
-- 查看当前隔离级别
SELECT @@transaction_isolation;

-- 设置隔离级别（会话级）
SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ;
```


# 5 一致性（Consistency）—— A + I + D 的最终目标

## 5.1 一致性的含义

一致性是指：事务将数据库从一个合法状态转变为另一个合法状态。所谓"合法状态"，
是指满足所有预定义的约束和业务规则。

一致性与其他三个特性不同：A、I、D 是数据库提供的**技术手段**，而 C 是这些手段要达成的**目标**。

## 5.2 数据库层面的一致性保障

InnoDB 提供的约束机制：

```sql
-- 主键约束：防止重复
CREATE TABLE orders (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    amount DECIMAL(10,2) NOT NULL CHECK (amount > 0),
    status ENUM('pending', 'paid', 'cancelled') NOT NULL DEFAULT 'pending',
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- 尝试违反约束
INSERT INTO orders (id, user_id, amount) VALUES (1, 999, 100.00);
-- ERROR 1452: Cannot add or update a child row:
-- a foreign key constraint fails

INSERT INTO orders (id, user_id, amount) VALUES (1, 1, -50.00);
-- ERROR 3819: Check constraint 'orders_chk_1' is violated
```

## 5.3 AID 如何共同保障一致性

以银行转账为例——A 向 B 转账 100 元：

```sql
BEGIN;
UPDATE accounts SET balance = balance - 100 WHERE id = 'A';
UPDATE accounts SET balance = balance + 100 WHERE id = 'B';
COMMIT;
```

一致性要求：无论何时，A 和 B 的余额之和不变。

```
┌─────────────────────────────────────────────────────────────┐
│                  AID 如何保障一致性                           │
│                                                             │
│  场景1：事务执行到一半崩溃                                    │
│    → 原子性(A) 保证：undo log 回滚，A 的钱没扣、B 的钱没加    │
│    → 一致性保持：总余额不变 ✓                                 │
│                                                             │
│  场景2：事务提交后崩溃，数据页未刷盘                           │
│    → 持久性(D) 保证：redo log 重做，A 扣了 100、B 加了 100    │
│    → 一致性保持：总余额不变 ✓                                 │
│                                                             │
│  场景3：转账同时有人查询 A 和 B 的余额                        │
│    → 隔离性(I) 保证：查询看到的是事务开始前的快照              │
│    → 一致性保持：看到的总余额一致 ✓                           │
│                                                             │
│  场景4：A 余额不足                                           │
│    → 约束检查 + 原子性(A)：CHECK 约束失败 → 事务回滚          │
│    → 一致性保持：不会出现负余额 ✓                             │
└─────────────────────────────────────────────────────────────┘
```

**追问：仅靠数据库的 AID 就能保证一致性吗？**

不能。一致性还依赖**应用层**的正确性。例如：

```sql
-- 应用层 bug：只扣了 A 的钱，忘了给 B 加钱
BEGIN;
UPDATE accounts SET balance = balance - 100 WHERE id = 'A';
-- 忘了 UPDATE B
COMMIT;
```

这个事务从数据库角度看是完全合法的——原子性、隔离性、持久性都满足。
但一致性被破坏了（总余额减少了 100）。数据库无法检测这种业务逻辑错误。

所以一致性 = 数据库约束（主键、外键、CHECK、NOT NULL）+ AID 技术手段 + 应用层逻辑正确性。


# 6 两阶段提交 —— redo log 与 binlog 的一致性协议

## 6.1 为什么需要两阶段提交

InnoDB 的 redo log 和 MySQL Server 层的 binlog 是两个独立的日志系统。
事务提交时，二者必须保持一致——要么都写入，要么都不写入。否则：

```
场景1：先写 redo log，再写 binlog
  → redo log 写成功，binlog 写失败
  → 主库通过 redo log 恢复了数据
  → 从库没有 binlog，数据缺失 → 主从不一致！

场景2：先写 binlog，再写 redo log
  → binlog 写成功，redo log 写失败
  → 主库崩溃恢复后丢失数据
  → 从库通过 binlog 同步了数据 → 主从不一致！
```

## 6.2 两阶段提交流程

```
┌──────────────────────────────────────────────────────────────────┐
│                      两阶段提交流程                               │
│                                                                  │
│  UPDATE t SET c = c + 1 WHERE id = 2                             │
│                                                                  │
│  ① 执行器调用 InnoDB 引擎，更新 Buffer Pool 中的数据页             │
│  ② InnoDB 将修改记录到 redo log，状态标记为 prepare               │
│     ──────────────── Prepare 阶段完成 ────────────────            │
│  ③ 执行器生成 binlog，写入 binlog 文件                            │
│  ④ 执行器调用 InnoDB 提交，redo log 状态改为 commit               │
│     ──────────────── Commit 阶段完成 ─────────────────            │
│                                                                  │
│  关键：binlog 写入成功是提交的"决策点"                             │
└──────────────────────────────────────────────────────────────────┘
```

## 6.3 崩溃恢复时的判断逻辑

```
崩溃恢复时，对每条 redo log 记录：

┌─────────────────────────────┐
│  redo log 是 commit 状态？   │
│           │                 │
│     是    │     否          │
│     ↓     │     ↓          │
│   提交    │  有对应 binlog？ │
│           │     │          │
│           │  是  │  否     │
│           │  ↓   │  ↓     │
│           │ 提交 │ 回滚   │
└─────────────────────────────┘
```

对应上面三个阶段可能的崩溃时机：

| 崩溃时机 | redo log 状态 | binlog | 恢复动作 | 原因 |
|---------|--------------|--------|---------|------|
| ② 之后，③ 之前 | prepare | 无 | 回滚 | binlog 未写入，从库不会有此数据 |
| ③ 之后，④ 之前 | prepare | 有 | 提交 | binlog 已写入，从库会同步此数据 |
| ④ 之后 | commit | 有 | 提交 | 正常完成 |

**追问：为什么 binlog 写入成功就要提交，即使 redo log 还是 prepare 状态？**

因为 binlog 一旦写入，就可能已经被从库读取并执行了。此时如果主库回滚，就会导致主从不一致。
所以 binlog 的写入是事务提交的"决策点"（point of no return）。

**追问：如何验证两阶段提交是否正常工作？**

```sql
-- 查看 binlog 事件，确认每个事务都有完整的 BEGIN...COMMIT
SHOW BINLOG EVENTS IN 'mysql-bin.000001' LIMIT 20;

-- 查看 InnoDB 日志状态
SHOW ENGINE INNODB STATUS\G
-- 关注 LOG 部分的 Log sequence number 和 Last checkpoint at
```


# 7 完整的事务执行流程

将 ACID 的所有机制串联起来，一条 UPDATE 语句在事务中的完整执行流程：

```sql
BEGIN;
UPDATE users SET name = 'Bob' WHERE id = 1;
COMMIT;
```

```
┌─────────────────────────────────────────────────────────────────┐
│                    完整执行流程                                   │
│                                                                 │
│  BEGIN                                                          │
│  ├─ 分配事务 ID (trx_id)                                        │
│  │                                                              │
│  UPDATE users SET name = 'Bob' WHERE id = 1                     │
│  ├─ ① 在 Buffer Pool 中查找 id=1 的数据页（不在则从磁盘读入）     │
│  ├─ ② 对 id=1 的记录加 X Lock（排他锁）        ← 隔离性          │
│  ├─ ③ 将旧值 name='Alice' 写入 undo log        ← 原子性          │
│  │     同时在行记录中更新 DB_TRX_ID 和 DB_ROLL_PTR               │
│  ├─ ④ 在 Buffer Pool 中修改 name → 'Bob'                        │
│  ├─ ⑤ 将修改写入 redo log buffer                ← 持久性          │
│  │                                                              │
│  COMMIT                                                         │
│  ├─ ⑥ redo log 刷盘，标记 prepare               ← 两阶段提交     │
│  ├─ ⑦ 生成 binlog，写入 binlog 文件                              │
│  ├─ ⑧ redo log 标记 commit                                      │
│  ├─ ⑨ 释放 X Lock                                               │
│  └─ ⑩ 事务结束                                                   │
│                                                                 │
│  后台：                                                          │
│  ├─ Checkpoint 机制异步将脏页刷入磁盘                             │
│  └─ purge 线程在无事务引用时清理 undo log                         │
└─────────────────────────────────────────────────────────────────┘
```

这个流程清晰地展示了 ACID 四大特性如何在一条语句的执行中协同工作：
- **undo log（步骤③）** 保障原子性——可以回滚。
- **redo log（步骤⑤⑥⑧）** 保障持久性——崩溃后可恢复。
- **X Lock（步骤②⑨）+ MVCC** 保障隔离性——并发不冲突。
- 三者共同保障一致性——数据始终处于合法状态。


# 8 实践：ACID 相关的关键配置与监控

## 8.1 关键配置参数

```sql
-- 持久性相关
SHOW VARIABLES LIKE 'innodb_flush_log_at_trx_commit';  -- 推荐 1
SHOW VARIABLES LIKE 'sync_binlog';                      -- 推荐 1
SHOW VARIABLES LIKE 'innodb_doublewrite';                -- 推荐 ON

-- 隔离性相关
SELECT @@transaction_isolation;                          -- 默认 REPEATABLE-READ
SHOW VARIABLES LIKE 'innodb_lock_wait_timeout';          -- 默认 50 秒

-- 原子性相关（undo 表空间）
SHOW VARIABLES LIKE 'innodb_undo_tablespaces';           -- 默认 2（MySQL 8.0+）
SHOW VARIABLES LIKE 'innodb_max_undo_log_size';          -- undo 表空间自动截断阈值
SHOW VARIABLES LIKE 'innodb_undo_log_truncate';          -- 是否启用自动截断

-- redo log 相关
SHOW VARIABLES LIKE 'innodb_redo_log_capacity';          -- MySQL 8.0.30+ 统一配置
SHOW VARIABLES LIKE 'innodb_log_file_size';              -- 8.0.30 之前
SHOW VARIABLES LIKE 'innodb_log_files_in_group';         -- 8.0.30 之前
```

## 8.2 "双 1 配置"

生产环境中，为了保证数据不丢失（严格的持久性），推荐"双 1 配置"：

```sql
innodb_flush_log_at_trx_commit = 1   -- 每次事务提交 redo log 都 fsync
sync_binlog = 1                       -- 每次事务提交 binlog 都 fsync
```

这样即使主机突然断电，已提交的事务也不会丢失，并且 binlog 完整，不会导致主从不一致。

代价是性能下降（更多的 fsync），但通过组提交和 SSD 可以有效缓解。

## 8.3 监控查询

```sql
-- 查看当前活跃事务
SELECT * FROM information_schema.INNODB_TRX;

-- 查看长事务（运行超过 60 秒）
SELECT trx_id, trx_state, trx_started,
       TIMESTAMPDIFF(SECOND, trx_started, NOW()) AS duration_sec,
       trx_query
FROM information_schema.INNODB_TRX
WHERE TIMESTAMPDIFF(SECOND, trx_started, NOW()) > 60;

-- 查看锁等待
SELECT * FROM performance_schema.data_lock_waits;

-- 查看 undo log 使用情况
SELECT name, subsystem, count
FROM information_schema.INNODB_METRICS
WHERE name LIKE '%undo%' OR name LIKE '%purge%';

-- 查看 redo log 写入量
SHOW STATUS LIKE 'Innodb_os_log_written';
```

**追问：为什么要监控长事务？**

长事务有三大危害：
1. **undo log 膨胀**：长事务持有的旧版本数据无法被 purge 清理，导致 undo 表空间不断增长。
2. **锁持有时间长**：阻塞其他事务，导致锁等待链变长，甚至死锁。
3. **版本链过长**：其他事务通过 MVCC 读取时，需要沿版本链回溯更多节点，影响查询性能。


# 9 总结

InnoDB 通过精密的多机制协同实现了事务的 ACID 特性：

| 特性 | 核心机制 | 关键组件 |
|------|---------|---------|
| 原子性 (A) | undo log 回滚 | insert undo / update undo、版本链 |
| 一致性 (C) | A + I + D + 约束 | 主键、外键、CHECK、NOT NULL + 业务逻辑 |
| 隔离性 (I) | MVCC（读）+ 锁（写） | ReadView、版本链、Record/Gap/Next-Key Lock |
| 持久性 (D) | redo log WAL | innodb_flush_log_at_trx_commit=1、Doublewrite |

四大特性不是孤立的，它们通过 redo log、undo log、binlog 和锁机制紧密交织：
- undo log 同时服务于原子性（回滚）和隔离性（MVCC 版本链）。
- redo log 同时服务于持久性（WAL）和崩溃恢复时配合 undo 保障原子性。
- 两阶段提交保证 redo log 与 binlog 的一致性，确保主从数据不分叉。
- 一致性是最终目标，由其他三个特性加上数据库约束和应用逻辑共同保障。
