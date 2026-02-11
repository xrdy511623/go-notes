
---
MVCC(多版本并发控制)详解
---

# 1 版本链

对于使用InnoDB存储引擎的表来说，它的聚簇索引记录中都包含两个必要的隐藏列（row_id并不是必要的，我们创建的表中有主键或者
非NULL的UNIQUE键时都不会包含row_id列）：
> trx_id：每次一个事务对某条聚簇索引记录进行改动时，都会把该事务的事务id赋值给trx_id隐藏列。
> roll_pointer：每次对某条聚簇索引记录进行改动时，都会把旧的版本写入到undo日志中，然后这个隐藏列就相当于一个指针，可以
通过它来找到该记录修改前的信息。

比方说我们的表hero现在只包含一条记录：





![mvcc-one.png](images%2Fmvcc-one.png)





假设插入该记录的事务id为80，那么此刻该条记录的示意图如下所示：





![mvcc-two.png](images%2Fmvcc-two.png)





假设之后两个事务id分别为100、200的事务对这条记录进行UPDATE操作，操作流程如下：





![mvcc-three.png](images%2Fmvcc-three.png)





每次对记录进行改动，都会记录一条undo日志，每条undo日志也都有一个roll_pointer属性（INSERT操作对应的undo日志没有
该属性，因为该记录并没有更早的版本），可以将这些undo日志都连起来，串成一个链表，所以现在的情况就像下图一样：





![mvcc-four.png](images%2Fmvcc-four.png)





对该记录每次更新后，都会将旧值放到一条undo日志中，就算是该记录的一个旧版本，随着更新次数的增多，所有的版本都会被
roll_pointer属性连接成一个链表，我们把这个链表称之为版本链，版本链的头节点就是当前记录最新的值。另外，每个版本中
还包含生成该版本时对应的事务id，这个信息很重要，我们稍后就会用到。


# 2 ReadView(读视图)

对于使用READ UNCOMMITTED隔离级别的事务来说，由于可以读到未提交事务修改过的记录，所以直接读取记录的最新版本就好了；
对于使用SERIALIZABLE隔离级别的事务来说，设计InnoDB的人规定使用加锁的方式来访问记录；对于使用READ COMMITTED和
REPEATABLE READ隔离级别的事务来说，都必须保证读到已经提交了的事务修改过的记录，也就是说假如另一个事务已经修改了记录
但是尚未提交，是不能直接读取最新版本的记录的，核心问题就是：需要判断一下版本链中的哪个版本是当前事务可见的。为此，
设计InnoDB的人提出了一个ReadView的概念，这个ReadView中主要包含4个比较重要的内容：
> m_ids：表示在生成ReadView时当前系统中活跃的读写事务的事务id列表。(也就是正在执行尚未提交的事务id列表)
> min_trx_id：表示在生成ReadView时当前系统中活跃的读写事务中最小的事务id，也就是m_ids中的最小值。
> max_trx_id：表示生成ReadView时系统中应该分配给下一个事务的id值。
> creator_trx_id：表示生成该ReadView事务的事务id。

注意：
只有在对表中的记录做改动时（执行INSERT、DELETE、UPDATE这些语句时）才会为事务分配事务id，否则在一个只读事务中的事务id值都默认为0。

有了这个ReadView，这样在访问某条记录时，只需要按照下边的步骤判断记录的某个版本是否可见：
- 如果被访问版本的trx_id属性值与ReadView中的creator_trx_id值相同，意味着当前事务在访问它自己修改过的记录，所以该版本可以被当前事务访问。
- 如果被访问版本的trx_id属性值小于ReadView中的min_trx_id值，表明生成该版本的事务在当前事务生成ReadView前已经提交，所以该版本可以被当前事务访问。
- 如果被访问版本的trx_id属性值大于或等于ReadView中的max_trx_id值，表明生成该版本的事务是在当前事务生成ReadView后才开启，所以该版本不可以被当前事务访问。
- 如果被访问版本的trx_id属性值在ReadView的min_trx_id和max_trx_id之间，那就需要判断一下trx_id属性值是不是在m_ids列表中，如果在，
说明创建ReadView时生成该版本的事务还是活跃的，该版本不可以被访问；如果不在，说明创建ReadView时生成该版本的事务已经被提交，该版本可以被访问。

用伪代码表示这个可见性判断算法：

```
function is_visible(trx_id, ReadView):
    if trx_id == creator_trx_id:       // 自己修改的，当然可见
        return true
    if trx_id < min_trx_id:            // 生成 ReadView 前就已提交
        return true
    if trx_id >= max_trx_id:           // 生成 ReadView 后才开启的事务
        return false
    if trx_id in m_ids:                // 生成 ReadView 时还在活跃（未提交）
        return false
    else:                              // 在 [min_trx_id, max_trx_id) 区间但已提交
        return true

function read_record(record, ReadView):
    version = record                    // 从版本链头节点（最新版本）开始
    while version != null:
        if is_visible(version.trx_id, ReadView):
            return version              // 找到第一个可见版本，返回
        version = version.roll_pointer  // 沿版本链向前回溯
    return NOT_VISIBLE                  // 所有版本都不可见，该记录对当前事务不存在
```

如果某个版本的数据对当前事务不可见的话，那就顺着版本链找到下一个版本的数据，继续按照上边的步骤判断可见性，依此类推，直到版本链中的最后一个版本。
如果最后一个版本也不可见的话，那么就意味着该条记录对该事务完全不可见，查询结果就不包含该记录。

> 读取已提交与可重复读两种隔离级别在生成readview上的差异

在MySQL中，READ COMMITTED和REPEATABLE READ隔离级别的的一个非常大的区别就是它们生成ReadView的时机不同。我们还是以表hero为例来，
假设现在表hero中只有一条由事务id为80的事务插入的一条记录：





![mvcc-one.png](images%2Fmvcc-one.png)





接下来看一下READ COMMITTED和REPEATABLE READ所谓的生成ReadView的时机不同到底不同在哪里。

## 2.1 READ COMMITTED(读取已提交) —— 每次读取数据前都生成一个ReadView

比方说现在系统里有两个事务id分别为100、200的事务在执行：

```sql
# Transaction 100
BEGIN;

UPDATE hero SET name = '关羽' WHERE number = 1;

UPDATE hero SET name = '张飞' WHERE number = 1;
```

```sql
# Transaction 200
BEGIN;
# 更新了一些别的表的记录
...
```

再次强调一遍，事务执行过程中，只有在第一次真正修改记录时（比如使用INSERT、DELETE、UPDATE语句），才会被分配一个单独的
事务id，这个事务id是递增的。所以我们才在Transaction 200中更新一些别的表的记录，目的是让它分配事务id。

此刻，表hero中number为1的记录得到的版本链表如下所示：





![mvcc-five.png](images%2Fmvcc-five.png)





假设现在有一个使用READ COMMITTED隔离级别的事务开始执行：

```sql
# 使用READ COMMITTED隔离级别的事务
BEGIN;
# SELECT1：Transaction 100、200未提交
SELECT * FROM hero WHERE number = 1; # 得到的列name的值为'刘备'
```

这个SELECT1的执行过程如下：
- 在执行SELECT语句时会先生成一个ReadView，ReadView的m_ids列表的内容就是[100, 200]，min_trx_id为100，max_trx_id为201，creator_trx_id为0。
- 然后从版本链中挑选可见的记录，从图中可以看出，最新版本的列name的内容是'张飞'，该版本的trx_id值为100，在m_ids列表内，所以不符合可见性要求，根据roll_pointer跳到下一个版本。
- 下一个版本的列name的内容是'关羽'，该版本的trx_id值也为100，也在m_ids列表内，所以也不符合要求，继续跳到下一个版本。
- 下一个版本的列name的内容是'刘备'，该版本的trx_id值为80，小于ReadView中的min_trx_id值100，所以这个版本是符合要求的，最后返回给用户的版本就是这条列name为'刘备'的记录。

之后，我们把事务id为100的事务提交一下，就像这样：

```sql
# Transaction 100
BEGIN;

UPDATE hero SET name = '关羽' WHERE number = 1;

UPDATE hero SET name = '张飞' WHERE number = 1;

COMMIT;
```

然后再到事务id为200的事务中更新一下表hero中number为1的记录：

```sql
# Transaction 200
BEGIN;
# 更新了一些别的表的记录
...

UPDATE hero SET name = '赵云' WHERE number = 1;

UPDATE hero SET name = '诸葛亮' WHERE number = 1;
```

此刻，表hero中number为1的记录的版本链就长这样：





![mvcc-six.png](images%2Fmvcc-six.png)





然后再到刚才使用READ COMMITTED隔离级别的事务中继续查找这个number为1的记录，如下：

```sql
# 使用READ COMMITTED隔离级别的事务
BEGIN;
# SELECT1：Transaction 100、200均未提交
SELECT * FROM hero WHERE number = 1; # 得到的列name的值为'刘备'
# SELECT2：Transaction 100提交，Transaction 200未提交
SELECT * FROM hero WHERE number = 1; # 得到的列name的值为'张飞'
```

这个SELECT2的执行过程如下：
- 在执行SELECT语句时会又会单独生成一个ReadView，该ReadView的m_ids列表的内容就是[200]（事务id为100的那个事务已经提交了，所以再次生成快照时就没有它了），min_trx_id为200，max_trx_id为201，creator_trx_id为0。
- 然后从版本链中挑选可见的记录，从图中可以看出，最新版本的列name的内容是'诸葛亮'，该版本的trx_id值为200，在m_ids列表内，所以不符合可见性要求，根据roll_pointer跳到下一个版本。
- 下一个版本的列name的内容是'赵云'，该版本的trx_id值为200，也在m_ids列表内，所以也不符合要求，继续跳到下一个版本。
- 下一个版本的列name的内容是'张飞'，该版本的trx_id值为100，小于ReadView中的min_trx_id值200，所以这个版本是符合要求的，最后返回给用户的版本就是这条列name为'张飞'的记录。

依此类推，如果之后事务id为200的记录也提交了，再次在使用READ COMMITTED隔离级别的事务中查询表hero中number值为1的记录时，得到的结果就是'诸葛亮'了，
具体流程我们就不分析了。总结一下就是：使用READ COMMITTED隔离级别的事务在每次查询开始时都会生成一个独立的ReadView。


## 2.2 REPEATABLE READ(可重复读) —— 只在第一次读取数据时生成一个ReadView，以后复用此ReadView

对于使用REPEATABLE READ隔离级别的事务来说，只会在第一次执行查询语句时生成一个ReadView，之后的查询就不会重复生成了，而是复用这个ReadView。
我们还是用例子看一下是什么效果。
比方说现在系统里有两个事务id分别为100、200的事务在执行：

```sql
# Transaction 100
BEGIN;

UPDATE hero SET name = '关羽' WHERE number = 1;

UPDATE hero SET name = '张飞' WHERE number = 1;
```

```sql
# Transaction 200
BEGIN;
# 更新了一些别的表的记录
...
```

此刻，表hero中number为1的记录得到的版本链表如下所示：





![mvcc-seven.png](images%2Fmvcc-seven.png)





假设现在有一个使用REPEATABLE READ隔离级别的事务开始执行：

```sql
# 使用REPEATABLE READ隔离级别的事务
BEGIN;
# SELECT1：Transaction 100、200未提交
SELECT * FROM hero WHERE number = 1; # 得到的列name的值为'刘备'
```
这个SELECT1的执行过程如下：
- 在执行SELECT语句时会先生成一个ReadView，ReadView的m_ids列表的内容就是[100, 200]，min_trx_id为100，max_trx_id为201，creator_trx_id为0。
- 然后从版本链中挑选可见的记录，从图中可以看出，最新版本的列name的内容是'张飞'，该版本的trx_id值为100，在m_ids列表内，所以不符合可见性要求，根据roll_pointer跳到下一个版本。
- 下一个版本的列name的内容是'关羽'，该版本的trx_id值也为100，也在m_ids列表内，所以也不符合要求，继续跳到下一个版本。
- 最后一个版本的列name内容是'刘备'，该版本的trx_id值是80，小于min_trx_id(100)，所以对该事务是可见的，因此读到的name列值为刘备。


之后，我们把事务id为100的事务提交一下，就像这样：

```sql
# Transaction 100
BEGIN;

UPDATE hero SET name = '关羽' WHERE number = 1;

UPDATE hero SET name = '张飞' WHERE number = 1;

COMMIT;
```

然后再到事务id为200的事务中更新一下表hero中number为1的记录：

```sql
# Transaction 200
BEGIN;
# 更新了一些别的表的记录
...

UPDATE hero SET name = '赵云' WHERE number = 1;

UPDATE hero SET name = '诸葛亮' WHERE number = 1;
```

此刻，表hero中number为1的记录的版本链就长这样：





![mvcc-eight.png](images%2Fmvcc-eight.png)





然后再到刚才使用REPEATABLE READ隔离级别的事务中继续查找这个number为1的记录，如下：

```sql
# 使用REPEATABLE READ隔离级别的事务
BEGIN;
# SELECT1：Transaction 100、200均未提交
SELECT * FROM hero WHERE number = 1; # 得到的列name的值为'刘备'
# SELECT2：Transaction 100提交，Transaction 200未提交
SELECT * FROM hero WHERE number = 1; # 得到的列name的值仍为'刘备'
```

这个SELECT2的执行过程如下：
- 因为当前事务的隔离级别为REPEATABLE READ，而之前在执行SELECT1时已经生成过ReadView了，所以此时直接复用之前的ReadView，之前的ReadView的m_ids列表的内容就是[100, 200]，min_trx_id为100，max_trx_id为201，creator_trx_id为0。
- 然后从版本链中挑选可见的记录，从图中可以看出，最新版本的列name的内容是'诸葛亮'，该版本的trx_id值为200，在m_ids列表内，所以不符合可见性要求，根据roll_pointer跳到下一个版本。
- 下一个版本的列name的内容是'赵云'，该版本的trx_id值为200，也在m_ids列表内，所以也不符合要求，继续跳到下一个版本。
- 下一个版本的列name的内容是'张飞'，该版本的trx_id值为100，而m_ids列表中是包含值为100的事务id的，所以该版本也不符合要求，同理下一个列name的内容是'关羽'的版本也不符合要求。继续跳到下一个版本。
- 下一个版本的列name的内容是'刘备'，该版本的trx_id值为80，小于ReadView中的min_trx_id值100，所以这个版本是符合要求的，最后返回给用户的版本就是这条列name为'刘备'的记录。
也就是说两次SELECT查询得到的结果是重复的，记录的列name值都是'刘备'，这就是可重复读的含义。如果我们之后再把事务id为200的记录提交了，
然后再到刚才使用REPEATABLE READ隔离级别的事务中继续查找这个number为1的记录，得到的结果还是'刘备'， 具体执行过程大家可以自己分析一下。

# 3 工程实践中的MVCC边界

上边讲清楚了原理，但在生产里最容易踩坑的地方是：哪些读是快照读，哪些读会加锁并读取最新版本。

## 3.1 快照读 vs 当前读

- 快照读（consistent read）：普通`SELECT`（不带`FOR UPDATE`/`FOR SHARE`）使用`ReadView` + undo版本链来决定可见性。
- 当前读（current read）：`SELECT ... FOR UPDATE`、`SELECT ... FOR SHARE`、`UPDATE`、`DELETE`会读取当前最新版本并加锁，不会沿版本链回退到历史版本。
- 结果差异：在`REPEATABLE READ`中，同一事务里普通`SELECT`可能一直看到老快照；但`UPDATE`或`SELECT ... FOR UPDATE`面对的是"当前值 + 锁冲突"，这也是很多"查询看到A，更新却被B影响"的根源。

**具体例子：同一事务内快照读与当前读的分裂**

```sql
-- 初始状态：hero 表 number=1 的 name='刘备'

-- -------- Session A (REPEATABLE READ) --------
BEGIN;
SELECT * FROM hero WHERE number = 1;
-- 结果：name='刘备'（快照读，生成 ReadView）

-- -------- Session B --------
UPDATE hero SET name = '曹操' WHERE number = 1;
COMMIT;

-- -------- 回到 Session A --------
SELECT * FROM hero WHERE number = 1;
-- 结果：name='刘备'（快照读，复用 ReadView，看不到 Session B 的修改）

SELECT * FROM hero WHERE number = 1 FOR UPDATE;
-- 结果：name='曹操'！（当前读，读取最新已提交版本并加 X 锁）

UPDATE hero SET name = '孙权' WHERE number = 1;
-- 实际修改的是 name='曹操' 那行，而不是快照中的 '刘备'

SELECT * FROM hero WHERE number = 1;
-- 结果：name='孙权'（自己修改的，creator_trx_id 匹配，可见）
COMMIT;
```

关键点：`SELECT ... FOR UPDATE` 和 `UPDATE` 都是当前读，它们跳过了 ReadView 机制，直接读取并锁定最新版本。
这就是为什么在同一个 RR 事务里，普通 SELECT 看到的和 FOR UPDATE 看到的可能不一样。

## 3.2 next-key lock / gap lock 与幻读防护

- `REPEATABLE READ`下，锁定读和范围更新通常使用next-key lock（记录锁+间隙锁）来阻止其他事务向范围内插入新行，从而抑制幻读。
- 这种防护依赖索引范围访问。条件没走到合适索引时，锁范围可能扩大，甚至退化为更重的锁，吞吐会明显下降。
- `READ COMMITTED`下，gap lock总体更保守（常见场景不启用，仅在外键检查、唯一键冲突检查等场景保留），因此并发更高，但幻读防护更弱。
- 普通`SELECT`是快照读，不加锁。它"看不到后续新插入行"并不等于阻止了幻读写入，只是读取了同一时点快照。

**RR 下的经典幻读边界案例**

很多人认为 REPEATABLE READ 完全解决了幻读，但实际上存在一个经典的边界情况：

```sql
-- 初始状态：hero 表只有 number=1 (name='刘备')

-- -------- Session A (REPEATABLE READ) --------
BEGIN;
SELECT * FROM hero WHERE number > 0;
-- 结果：只有 1 行 (number=1, name='刘备')

-- -------- Session B --------
INSERT INTO hero(number, name) VALUES(2, '曹操');
COMMIT;

-- -------- 回到 Session A --------
SELECT * FROM hero WHERE number > 0;
-- 结果：仍然只有 1 行（快照读，ReadView 复用，看不到 number=2）

-- 关键操作：对"看不到"的那行做 UPDATE
UPDATE hero SET name = '魏王曹操' WHERE number = 2;
-- 影响行数：1（UPDATE 是当前读，能看到并修改 Session B 插入的行！）

SELECT * FROM hero WHERE number > 0;
-- 结果：2 行！(number=1, name='刘备') 和 (number=2, name='魏王曹操')
-- 幻读出现了！因为 number=2 这行被当前事务 UPDATE 过，trx_id 变成了自己的，
-- 所以 is_visible 判断 creator_trx_id 匹配，返回 true
COMMIT;
```

**为什么会这样？**

```
1. Session A 的 ReadView 中看不到 number=2（Session B 的 trx_id 在 m_ids 中或 >= max_trx_id）
2. UPDATE 是当前读，不走 ReadView，直接操作最新版本，成功修改了 number=2
3. 修改后，number=2 这行的 trx_id 变成了 Session A 自己的事务 id
4. 再次 SELECT 时，is_visible 判断 trx_id == creator_trx_id → true，可见！
5. 于是 Session A 看到了之前看不到的行 —— 幻读
```

**防御措施：** 如果业务逻辑依赖"范围内不能有新行"，必须使用 `SELECT ... FOR UPDATE` 加锁读。
它会加 next-key lock / gap lock，阻止其他事务在锁定范围内插入新行，从根本上阻止幻读：

```sql
-- Session A
BEGIN;
SELECT * FROM hero WHERE number > 0 FOR UPDATE;
-- 加锁：对 number > 0 的范围加 next-key lock

-- Session B
INSERT INTO hero(number, name) VALUES(2, '曹操');
-- 阻塞！因为 number=2 落在 Session A 的 gap lock 范围内
```

## 3.3 undo purge 与长事务问题

- `UPDATE`/`DELETE`产生undo版本，后台purge线程会在“没有活跃ReadView再需要这些旧版本”后回收。
- 长事务（尤其长时间不提交的`REPEATABLE READ`事务）会拖住purge，导致历史版本积压（History list length升高）、undo膨胀、一致性读变慢。
- 实践建议：事务尽量短小；避免“事务中等待用户输入/远程调用”；批处理按主键分批提交。

可以用以下命令排查：

```sql
-- 观察长事务
SELECT trx_id, trx_started, trx_state, trx_query
FROM information_schema.innodb_trx
ORDER BY trx_started;

-- 观察历史版本积压（History list length）
SHOW ENGINE INNODB STATUS\G

-- MySQL 8 可选指标（开启innodb_metrics时更直观）
SELECT name, count
FROM information_schema.innodb_metrics
WHERE name = 'trx_rseg_history_len';
```


## 3.4 二级索引与 MVCC

前面的版本链讨论都基于聚簇索引记录，但实际查询经常走二级索引。二级索引的叶子节点**不存储 trx_id 和 roll_pointer**（它们只存在于聚簇索引记录中），那 InnoDB 是怎么对二级索引做可见性判断的？

**判断流程：**

```
1. 扫描二级索引叶子页，找到满足条件的索引记录
2. 检查该索引页的 PAGE_MAX_TRX_ID（页级别的最大事务 id）
   - 如果 PAGE_MAX_TRX_ID < ReadView.min_trx_id
     → 说明该页上所有记录的最后修改事务都已提交，整页可见，无需回表检查
   - 否则 → 需要回聚簇索引逐行检查版本链
3. 回表到聚簇索引，用该记录的 trx_id + 版本链做标准的可见性判断
4. 如果可见，返回该记录；如果不可见，跳过
```

**性能影响：**

- **PAGE_MAX_TRX_ID 优化**：这是一个页级别的快速过滤。在大多数读多写少的场景下，二级索引页很少被并发修改，
  PAGE_MAX_TRX_ID 通常远小于 ReadView.min_trx_id，大量页可以跳过回表检查，这是一个很重要的优化。
- **覆盖索引的限制**：即使查询的字段都在二级索引中（理论上是覆盖索引），如果 PAGE_MAX_TRX_ID 检查不通过，
  InnoDB 仍然必须回表到聚簇索引做可见性判断。这就是为什么在 EXPLAIN 中看到 `Using index`（覆盖索引），
  实际执行时仍可能有回表开销——EXPLAIN 不考虑 MVCC 可见性。

```sql
-- 例：二级索引 idx_name 覆盖了查询所需的所有字段
EXPLAIN SELECT name FROM hero WHERE name = '刘备';
-- Extra: Using index（覆盖索引，理论上不需要回表）

-- 但如果此时有并发事务正在修改 hero 表，PAGE_MAX_TRX_ID 较大，
-- InnoDB 仍需回聚簇索引检查版本链，实际产生了回表 IO
```

这也解释了一个常见的性能疑问：为什么同样的覆盖索引查询，在高并发写入期间会比低并发时慢——不是索引失效了，
而是 MVCC 可见性检查迫使更多的回表操作。

**小结**

从上边的描述中我们可以看出来，所谓的MVCC（Multi-Version Concurrency Control ，多版本并发控制）指的就是在使用
READ COMMITTED、REPEATABLE READ这两种隔离级别的事务在执行普通的SELECT操作时访问记录的版本链的过程，这样可以
使不同事务的读-写、写-读操作并发执行，从而提升系统性能。READ COMMITTED、REPEATABLE READ这两个隔离级别的一个很大不同就是：
生成ReadView的时机不同，READ COMMITTED在每一次进行普通SELECT操作前都会生成一个ReadView，而REPEATABLE READ只在第一次
进行普通SELECT操作前生成一个ReadView，之后的查询操作都重复使用这个ReadView就好了。

# 4 追问

1. **ReadView 的 max_trx_id 为什么是"下一个待分配的事务 id"而不是"当前最大事务 id + 1"？**
   在并发场景下，事务 id 的分配是全局递增的，但分配和提交的顺序不一定一致。max_trx_id 取的是生成 ReadView 瞬间的全局下一个待分配值，
   这保证了任何在此之后才分配 id 的事务一定不可见。如果用"当前最大 id + 1"，可能遗漏某些已分配但还未出现在 m_ids 中的事务。

2. **为什么 UPDATE 是当前读而不是基于快照修改？**
   如果 UPDATE 基于快照版本修改，就会产生"丢失更新"问题：事务 A 读到旧快照，基于旧值计算新值并写入，
   覆盖了事务 B 已经提交的修改。当前读 + 行锁保证了 UPDATE 始终操作最新版本，配合锁机制串行化冲突写入。

3. **长事务为什么会导致查询变慢，而不仅仅是 undo 空间膨胀？**
   长事务持有的 ReadView 会拖住 purge，导致版本链不断增长。当其他事务做快照读时，需要沿版本链逐个回溯判断可见性，
   版本链越长，回溯越久。极端情况下，一条记录可能有上万个历史版本，每次 SELECT 都要遍历整条链，查询延迟急剧上升。

4. **RC 隔离级别下是否还有 MVCC？为什么很多互联网公司选择 RC 而不是 RR？**
   RC 同样使用 MVCC，只是每次 SELECT 都重新生成 ReadView，所以能看到其他事务最新提交的结果。
   互联网公司选择 RC 的主要原因：① gap lock 在 RC 下基本不启用，死锁概率大幅降低，并发吞吐更高；
   ② 业务层通常已有幂等/乐观锁机制，不依赖数据库层的幻读防护；③ 事务间的可见性更直觉，排查问题更简单。
   代价是失去了可重复读保证和数据库层面的幻读防护，需要业务层自行处理。

5. **MVCC 为什么不能替代锁？二者是什么关系？**
   MVCC 解决的是**读-写并发**：读不阻塞写，写不阻塞读。但它不能解决**写-写并发**——两个事务同时修改同一行，
   必须通过行锁串行化。所以 InnoDB 的并发控制是 **MVCC + 2PL（两阶段锁）** 的混合方案：
   读操作走 MVCC（快照读，无锁），写操作走 2PL（加行锁，事务提交时释放）。
   `SELECT ... FOR UPDATE` 是显式地把一个读操作从 MVCC 路径切换到加锁路径，
   适用于"读完之后要基于读到的值做写入"的场景（如扣库存、转账）。
