
---
MySQL查询一行数据也慢，为什么？
---

# 第一类：查询长时间不返回
一般碰到这种情况的话，大概率是表 t 被锁住了。接下来分析原因的时候，一般都是首先执行一下 show processlist 命令，看看当前语句处于什么状态。
然后我们再针对每种状态，去分析它们产生的原因、如何复现，以及如何处理。

## 等MDL锁
如下图所示，就是使用 show processlist 命令查看 Waiting for table metadata lock 的示意图。





![wait-mdl-lock.png](images%2Fwait-mdl-lock.png)





出现这个状态表示的是，现在有一个线程正在表 t 上请求或者持有 MDL 写锁，把 select 语句堵住了。
这个场景复现就是session A 通过 lock table 命令持有表 t 的 MDL 写锁，而 session B 的查询需要获取 MDL 读锁。所以，session B 进入等待状态。
这类问题的处理方式，就是找到谁持有 MDL 写锁，然后把它 kill 掉。
但是，由于在 show processlist 的结果里面，session A 的 Command 列是“Sleep”，导致查找起来很不方便。不过有了 performance_schema 和 sys 
系统库以后，就方便多了。（MySQL 启动时需要设置 performance_schema=on，相比于设置为 off 会有 10% 左右的性能损失)
通过查询 sys.schema_table_lock_waits 这张表，我们就可以直接找出造成阻塞的 process id，把这个连接用 kill 命令断开即可。





![find-blocking-pid.png](images%2Ffind-blocking-pid.png)





## 等 flush
接下来，我给你举另外一种查询被堵住的情况。
我在表 t 上，执行下面的 SQL 语句：

```sql
select * from information_schema.processlist where id=1;
```

我查出来这个线程的状态是 Waiting for table flush。





![wait-flush.png](images%2Fwait-flush.png)





出现 Waiting for table flush 状态的可能情况是：有一个 flush tables 命令被别的语句堵住了，然后它又堵住了我们的 select 语句。

现在，我们一起来复现一下这种情况，复现步骤如下图所示：





![repeat.png](images%2Frepeat.png)





在 session A 中，我故意每行都调用一次 sleep(1)，这样这个语句默认要执行 10 万秒，在这期间表 t 一直是被
session A“打开”着。然后，session B 的 flush tables t 命令再要去关闭表 t，就需要等 session A 的查询结束。
这样，session C 要再次查询的话，就会被 flush 命令堵住了。

## 等行锁
现在，经过了表级锁的考验，我们的 select 语句终于来到引擎里了。

```sql
select * from t where id=1 lock in share mode; 
```

由于访问 id=1 这个记录时要加读锁，如果这时候已经有一个事务在这行记录上持有一个写锁，我们的 select 语句就会被堵住。





![wait-record-lock.png](images%2Fwait-record-lock.png)





显然，session A 启动了事务，占有写锁，还不提交，是导致 session B 被堵住的原因。
这个问题并不难分析，但问题是怎么查出是谁占着这个写锁。如果你用的是 MySQL 5.7 版本，可以通过 sys.innodb_lock_waits 表查到。
查询方法是：

```sql
select * from sys.innodb_lock_waits where locked_table=`'test'.'t'`\G
```





![analyze-lock-nine.png](images%2Fanalyze-lock-nine.png)





可以看到，这个信息很全，4 号线程是造成堵塞的罪魁祸首。而干掉这个罪魁祸首的方式，就是 KILL QUERY 4 或 KILL 4。
不过，这里不应该显示“KILL QUERY 4”。这个命令表示停止 4 号线程当前正在执行的语句，而这个方法其实是没有用的。
因为占有行锁的是 update 语句，这个语句已经是之前执行完成了的，现在执行 KILL QUERY，无法让这个事务去掉 id=1 上的行锁。
实际上，KILL 4 才有效，也就是说直接断开这个连接。这里隐含的一个逻辑就是，连接被断开的时候，会自动回滚这个连接里面正在
执行的线程，也就释放了 id=1 上的行锁。


# 第二类 查询慢
经过了重重封“锁”，我们再来看看一些查询慢的例子。
先来看一条你一定知道原因的 SQL 语句：
```sql
select * from t where c=50000 limit 1;
```

由于字段 c 上没有索引，这个语句只能走 id 主键顺序扫描，因此需要扫描 5 万行。

然后再看这个:
```sql
select * from t where id=1；
```





![repeat-one.png](images%2Frepeat-one.png)





你看到了，session A 先用 start transaction with consistent snapshot 命令启动了一个事务，之后 session B 才开始执行 update 语句。
session B 执行完 100 万次 update 语句后，id=1 这一行处于什么状态呢？你可以从下图中找到答案。





![version-chain.png](images%2Fversion-chain.png)





session B 更新完 100 万次，生成了 100 万个回滚日志 (undo log)。
带 lock in share mode 的 SQL 语句，是当前读，因此会直接读到 1000001 这个结果，所以速度很快；而
select * from t where id=1 这个语句，是一致性读，因此需要从 1000001 开始，依次执行 undo log，
执行了 100 万次以后，才将 1 这个结果返回。