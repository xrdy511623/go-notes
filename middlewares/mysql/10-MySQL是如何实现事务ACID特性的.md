
---
MySQL是如何实现事务ACID特性的？
---

# 1 ACID 事务特性概述

ACID 事务的四个特性是：
原子性（Atomicity）：事务中的所有操作要么全部成功，要么全部失败回滚。
一致性（Consistency）：事务执行后，数据库的状态必须保持一致（例如约束、规则不能被破坏）。
隔离性（Isolation）：多个事务并发执行时，一个事务的执行不应影响其他事务。
持久性（Durability）：事务提交后，其结果应永久保存，即使系统崩溃。

InnoDB 通过多种机制（日志、锁、隔离级别、崩溃恢复等）实现这些特性。以下是具体实现方式：


# 2 InnoDB 如何实现 ACID

## 2.1 原子性（Atomicity）

Redo Log（重做日志）：
InnoDB 使用重做日志（Write-Ahead Logging, WAL）来确保事务的原子性。
在事务执行过程中，所有的更改都会先记录到重做日志中（物理日志，记录页面修改）。
redo log重做日志记录的都是未刷盘的日志，已经刷入磁盘的数据都会从redo log 这个有限大小的日志文件里删除。
redo log重做日志主要功能就是在数据库异常重启后，可以根据它将之前提交的事务的修改记录恢复数据，这就是crash-safe，
也就是崩溃恢复能力。

Undo Log（回滚日志）：
为了支持回滚，InnoDB 维护回滚日志（逻辑日志，记录数据修改前的状态）。
如果事务失败或被回滚，InnoDB 使用 Undo Log 恢复数据到事务开始前的状态。

两阶段提交（Two-Phase Commit）：
InnoDB 在事务提交时，先将修改写入重做日志（Prepare 阶段），然后将数据写入缓冲区和磁盘（Commit 阶段）。
如果中间发生故障，事务可以根据日志状态恢复或回滚。

如何工作：
事务开始时，InnoDB 为每个事务分配事务 ID，并记录操作到 Undo Log 和 Redo Log。
事务提交时，InnoDB 确保所有更改持久化到磁盘；如果失败，使用 Undo Log 回滚。

## 2.2 一致性
数据完整性约束：
InnoDB 支持外键、唯一约束、检查约束等，确保数据符合定义的规则。
例如，唯一索引防止重复值，not null 约束防止空值。
事务隔离：通过隔离级别（Read Uncommitted、Read Committed、Repeatable Read、Serializable）确保事务执行不破坏数据库一致性。

日志和回滚：
结合 Redo Log 和 Undo Log 保证数据修改的一致性，即使发生崩溃也能恢复。

如何工作：
在事务执行过程中，InnoDB 检查所有约束和规则。
如果事务违反一致性（例如插入重复主键），事务被回滚（通过 Undo Log）。
事务提交后，通过 Redo Log 确保修改持久化。

示例：
事务试图插入重复主键值，InnoDB 检测到违反唯一约束，抛出错误并回滚事务，确保数据一致性。

## 2.3 隔离性
nnoDB 使用行级锁（Record Lock）、间隙锁（Gap Lock）和下一个键锁（Next-Key Lock）来控制并发访问。
事务通过共享锁（S Lock）和排他锁（X Lock）确保隔离。

多版本并发控制（MVCC) (详见第二讲MVCC详解)：
InnoDB 利用 Undo Log 实现 MVCC，允许多个事务同时读取和修改数据。
每个行记录有多个版本（通过事务 ID 和回滚指针跟踪），不同事务看到的是不同版本的数据。

隔离级别：
Read Uncommitted：可能读到未提交数据（脏读）。
Read Committed：只读已提交数据（防止脏读）。
Repeatable Read（默认）：确保事务内重复读取相同数据一致（防止脏读和不可重复读）。
Serializable：最高隔离级别，通过锁机制完全串行化事务（防止幻读）。

如何工作：
事务开始时，InnoDB 分配事务 ID，并根据隔离级别选择锁和版本控制策略。
MVCC 通过隐藏列（DB_TRX_ID、DB_ROLL_PTR）跟踪数据版本，允许不同事务看到不同快照。
锁机制防止并发修改冲突。


## 2.4 持久性

实现机制：
Redo Log 和 Binlog：
重做日志（Redo Log）记录物理更改，确保崩溃后能重做。
二进制日志（Binlog）记录逻辑更改，用于主从复制和点恢复。

双写缓冲（Double write Buffer）：
InnoDB 使用双写缓冲，将数据先写入内存缓冲区，再写入磁盘，确保数据完整性。

崩溃恢复：
利用 Redo Log 重做未完成的事务，利用 Undo Log 回滚未提交事务。

如何工作：
事务提交时，InnoDB 将修改写入 Redo Log 和数据文件（通过双写缓冲保证一致性）。
如果系统崩溃，重启时通过 Redo Log 重放提交事务，通过 Undo Log 回滚未提交事务。