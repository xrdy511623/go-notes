---
SQL优化
---

# 1 工程优化

## 1.1 基础规范

表存储引擎必须使用InnoDB;
表字符集默认使用utf8mb4;
MySQL里的utf8实际是utf8mb3，只支持最多3字节字符，不适合做新系统默认字符集;
utf8mb4可以完整覆盖Unicode字符集(包括表情符号)，是推荐默认值;

禁止使用存储过程，视图，触发器，Event;
对数据库性能影响较大的互联网业务，能让站点层和服务层干的事情，不要交到数据库层调试，排错，再者迁移比较困难，扩展性较差;
禁止在数据库中存储大文件，例如图片，音频，视频，可以将大文件存储在对象存储系统(OSS)，数据库中存储路径即可;
禁止在线上环境做数据库压力测试;
测试，开发，线上数据库环境必须隔离。

## 1.2 命名规范

库名，表名，列名必须用小写，采用下划线分隔tb_book或者t_book;
abc, Abc， ABC都是给自己埋坑;
库名，表名，列名必须见名知义，长度不要超过32个字符;
Tmp, wushan谁TM知道这些库是干嘛的;
库备份必须以bak为前缀，以日期为后缀;
从库必须以-s为后缀;
备库必须以-ss为后缀。

## 1.3 表设计规范

单实例表个数必须控制在2000个以内;
单表分表个数必须控制在1024个以内;
表必须有主键，推荐使用UNSIGNED无符号整数为主键;
删除无主键的表，如果是row模式的主从架构，从库会挂;
禁止使用外键，如果要保证数据完整性，应由应用程序实现，譬如前端设计一个下拉框，限制用户输入;
因为外键会使表之间相互耦合，影响update/delete等SQL性能，有可能造成死锁，高并发情况下容易成为数据库瓶颈。
建议将大字段，访问频度低的字段拆分到单独的表中存储，分离冷热数据。

**水平拆分和垂直拆分**

水平切分是指，以某个字段为依据（例如uid），按照一定规则（例如取模），将一个库（表）上的数据拆分到多个库
（表）上，以降低单库（表）大小，达到提升性能的目的方法，水平切分后，各个库（表）的特点是：
（1）每个库（表）的结构都一样;
（2）每个库（表）的数据都不一样，没有交集;
（3）所有库（表）的并集是全量数据。

垂直拆分是指，将一个属性较多，一行数据较大的表，将不同的属性拆分到不同的表中，以降低单库（表）大小，达到提升
性能的目的的方法，垂直切分后，各个库（表）的特点是：
（1）每个库（表）的结构都不一样;
（2）一般来说，每个库（表）的属性至少有一列交集，一般是主键;
（3）所有库（表）的并集是全量数据。

垂直切分的依据是什么？
当一个表属性很多时，如何来进行垂直拆分呢？如果没有特殊情况，拆分依据主要有几点：
（1）将长度较短，访问频率较高的属性尽量放在一个表里，这个表暂且称为主表;
（2）将字段较长，访问频率较低的属性尽量放在一个表里，这个表暂且称为扩展表;
如果1和2都满足，还可以考虑第三点：
（3）经常一起访问的属性，也可以放在一个表里。
优先考虑1和2，第3点不是必须。另，如果实在属性过多，主表和扩展表都可以有多个。

（1）水平拆分和垂直拆分都是降低单表数据量大小，提升数据库性能的常见手段;
（2）流量大，数据量大时，数据访问要有service层，并且service层不要通过join来获取主表和扩展表的属性;
（3）垂直拆分的依据，尽量把长度较短，访问频率较高的属性放在主表里;

## 1.4 列设计规范

根据业务区分tinyint/int/bigint，分别占用1/4/8字节;
根据业务区分使用char和varchar;
如果字段长度固定，或者长度相似的业务场景，适合使用char，能够减少碎片，查询性能高。
如果字段长度相差较大，或者更新较少的业务场景，适合使用varchar，能够节省存储空间。

根据业务区分使用datetime和timestamp;
前者占用5个字节，后者占用4个字节，存储年使用YEAR，存储日期使用DATE，存储时间使用datetime。

必须把字段定义为NOT NULL, 并设置默认值;
因为NULL列使用索引，索引统计，值都更加复杂，MySQL更难优化。
NULL列需要更多的存储空间。
NULL只能采用IS NULL 或IS NOT NULL，而在!, in, not in 时有大坑。

使用INT UNSIGNED存储IPv4, 不要使用char(15);
使用varchar(20)存储手机号，不要使用整数;
牵扯到国家代号，可能出现+/-/()等字符，例如+86。
而且手机号不会用来做数学运算。
varchar可以做模糊查询，例如like '138%'。

使用TINYINT来代替ENUM;
使用ENUM增加新值要进行DDL操作。

## 1.5 索引规范

唯一索引使用uniq_[字段名]来命名
非唯一索引使用idx_[字段名]来命名
单张表索引数量建议控制在5个以内:
互联网高并发业务，太多索引会影响写性能。
SQL优化器生成执行计划时，如果索引太多，会降低性能，并可能导致MySQL选择不到最优索引。
异常复杂的查询需求，可以使用ES等更为适合的方式存储。
联合索引的字段数不建议超过5个。
如果5个字段还不能极大缩小扫描行数(rows)，八成是设计有问题。
不建议在频繁更新的字段上建立索引，因为维护索引有序性也是有代价的。
不建议在区分度低的字段上建立索引，譬如gender性别字段，建立索引有额外的存储成本。
非必要不要进行JOIN联表查询，如果要进行JOIN查询，被JOIN的字段必须类型相同，并建立索引。
有没有踩过因为被JOIN的字段类型不同, 导致索引失效最终引发全表扫描的坑？
理解联合索引最左前缀原则，避免重复建设索引，如果建立了(a,b,c)三个字段的联合索引，相当于建立了(a), (a,b), (a,b,c)索引。

## 1.6 SQL规范

禁止使用 select *，只获取必要字段;
select * 会增加 cpu/io/内存/带宽 的消耗。
指定字段能有效利用覆盖索引。
指定字段查询，在表结构变更时，能保证对应用程序无影响。

insert 必须指定字段，禁止使用 insert into T values();
指定字段插入，在表结构变更时，能保证对应用程序无影响

隐式类型转换会使索引失效，导致全表扫描;
禁止在 where 条件列使用函数或者表达式;
导致不能命中索引，全表扫描。

禁止负向查询以及 % 开头的模糊查询;
导致不能命中索引，全表扫描。

谨慎使用大表 JOIN 和子查询，是否可用以执行计划和实际执行耗时为准;
同一个字段上的 OR 可以考虑改写为IN，但不是强制规则，IN 列表长度没有固定阈值，需结合数据分布和执行计划评估。

应用程序必须捕获 SQL 异常, 方便定位线上问题。

# 2 SQL语句优化

SQL 优化的完整流程是：**定位问题 SQL → 分析执行计划 → 实施优化 → 验证效果**。很多人直接从第二步开始，
但第一步"定位问题 SQL"才是入口——你得先知道哪条 SQL 慢了，才能去优化它。

## 2.1 慢查询日志（Slow Query Log）

慢查询日志是 MySQL 内置的"性能探针"，它会记录所有执行时间超过阈值的 SQL 语句，是定位慢 SQL 的第一手工具。

### 2.1.1 开启慢查询日志

```sql
-- 查看当前是否开启
SHOW VARIABLES LIKE 'slow_query_log';

-- 开启慢查询日志
SET GLOBAL slow_query_log = ON;

-- 设置慢查询阈值（单位：秒），超过该时间的SQL会被记录
-- 默认是10秒，建议根据业务设为1秒或更低
SET GLOBAL long_query_time = 1;

-- 查看慢查询日志文件位置
SHOW VARIABLES LIKE 'slow_query_log_file';

-- 是否记录未使用索引的查询（不管快慢都记录）
SET GLOBAL log_queries_not_using_indexes = ON;
```

注意：通过 `SET GLOBAL` 设置的参数在 MySQL 重启后会失效，如需持久化，应写入 my.cnf 配置文件：

```ini
[mysqld]
slow_query_log = ON
long_query_time = 1
slow_query_log_file = /var/log/mysql/slow.log
log_queries_not_using_indexes = ON
```

### 2.1.2 分析慢查询日志

慢查询日志记录了每条慢 SQL 的执行时间、扫描行数、返回行数等信息。日志量可能很大，
直接看文件效率低，应该用工具来分析。

**方法一：mysqldumpslow（MySQL 自带）**

```bash
# 按平均耗时排序，取前10条最慢的SQL
mysqldumpslow -s at -t 10 /var/log/mysql/slow.log

# 按出现次数排序，找最频繁的慢SQL
mysqldumpslow -s c -t 10 /var/log/mysql/slow.log

# 按锁等待时间排序
mysqldumpslow -s l -t 10 /var/log/mysql/slow.log
```

mysqldumpslow 会将参数相同的 SQL 归类，输出格式类似：

```
Count: 385  Time=2.10s (808s)  Lock=0.00s (0s)  Rows=20.0 (7700), user@host
  SELECT * FROM orders WHERE user_id = N AND status = N ORDER BY created_at LIMIT N, N
```

这条输出的意思是：这类 SQL 执行了 385 次，平均每次 2.10 秒，总耗时 808 秒，每次返回 20 行。

**方法二：pt-query-digest（Percona Toolkit，功能更强大）**

```bash
# 安装 Percona Toolkit 后
pt-query-digest /var/log/mysql/slow.log > slow_report.txt
```

pt-query-digest 会生成详细的报告，包括 SQL 指纹、执行次数、平均/最大/最小耗时、扫描行数分布等，
非常适合做系统性的慢查询分析。

### 2.1.3 线上慢查询排查的一般流程

1. 通过慢查询日志或监控告警，定位到具体的慢 SQL
2. 用 `EXPLAIN` 分析执行计划，找到慢的原因（全表扫描？没走索引？filesort？）
3. 针对原因做优化（加索引？改写 SQL？调整查询条件？）
4. 用 `EXPLAIN ANALYZE`（MySQL 8.0+）验证优化效果
5. 上线后持续观察慢查询日志，确认问题已解决


## 2.2 Explain工具

![explain-two.png](images%2Fexplain-two.png)

### id

表示执行顺序，数字越大的先执行，如果数字相等，则排在上面的sql语句先执行。以上面的sql语句为例，显然是id为2的子查询
sql语句(select id from tb_areas where title = "成都市")先执行，然后再执行外层的主查询sql语句。

### select_type
表示查询类型，常见的有:
SIMPLE: 表示该SQL查询语句不包含子查询或join联表查询或Union查询，是普通的sql语句；
PRIMARY: 表示此查询是复杂查询中最外层的主查询语句。
SUBQUERY: 表示此查询是子查询语句。
UNION: 表示此查询是UNION的第二个或后续查询
UNION RESULT: UNION查询的结果
DEPENDENT SUBQUERY: SELECT子查询语句依赖外层查询的结果

### type

表示存储引擎查询数据时采用的方式，是非常重要的一个属性，通过它可以判断出查询是全表扫描还是基于索引的部分扫描。
常见的属性值如下，从上到下查询性能依次增强。
ALL: 表示全表扫描，性能最差;
index: 表示基于索引的全表扫描，查询只涉及索引字段，扫描索引即可，无需访问数据行;
range: 表示使用索引范围查询，常见于>, >=, <, <=, in等等;
ref: 表示使用非唯一索引进行扫描，适用于非唯一索引的等值查询;
eq_ref: 表示使用唯一索引或主键索引扫描(需要扫描，匹配多次，常见于多表连接查询);
const: 表示使用主键或唯一索引做等值查询，常量查询;
NULL: 表示不用访问表，速度最快。

**不要机械追求type一定要达到range及以上，应该结合rows、filtered、Extra、回表成本和实际执行时间综合判断。**

### possible_keys

表示查询时可能会用到的索引

### key

表示查询时真正用到的索引，显示的是索引名

### key_len

表示查询时使用了索引的字节数，可以判断联合索引的使用是否充分。
key_len计算规则(与数据类型，字符集以及是否允许为NULL有关):

| 数据类型                 | 原始占用字节数 | 不允许为NULL | 允许为NULL |
|----------------------|---------|----------|---------|
| tinyint              | 1       | 1        | 1+1     |
| smallint             | 2       | 2        | 2+1     |
| int                  | 4       | 4        | 4+1     |
| bigint               | 8       | 8        | 8+1     |
| (utf8字符集)char(n)     | 3*n     | 3*n      | 3*n+1   |
| (utf8字符集)varchar(n)  | 3*n+2   | 3*n+2    | 3*n+2+1 |
| (utf8mb4) char(n)    | 4*n     | 4*n      | 4*n+1   |
| (utf8mb4) varchar(n) | 4*n+2   | 4*n+2    | 4*n+2+1 |
| year                 | 1       | 1        | 1+1     |
| time                 | 3       | 3        | 3+1     |
| date                 | 4       | 4        | 4+1     |
| timestamp            | 4       | 4        | 4+1     |
| datetime             | 8       | 8        | 8+1     |


| 字符集       | 占用字节数 |
|-----------|-------|
| gbk       | 2     |
| utf8      | 3     |
| utf8mb4   | 4     |
| utf16     | 2     |
| iso8859-1 | 1     |
| gb2312    | 2     |


如果字段允许为NULL，需要额外的一个字节记录是否为NULL。
索引长度上限与MySQL版本、行格式、字符集、存储引擎参数有关，不建议用固定的"768字节"作为通用结论。字符串列应结合业务做前缀索引或函数索引设计。





![create-table-one.png](images%2Fcreate-table-one.png)





![explain-three.png](images%2Fexplain-three.png)





由表结构可知，c_age，c_name, c_address的联合索引的长度，也就是key_len=1+1+30x4+2+1+100x4+2+1=2+123+403=528, 
上图第一条sql使用到了联合索引u_key且使用的长度为528字节，说明使用索引充分，完全命中了联合索引u_key；但是第二条sql
就使用索引不充分了，从key_len=125字节推算，它只使用了c_age和c_name的部分索引，没有用到c_address列的索引，
key_len=1+1+3x40+2+1=2+123=125， 此时我们分析一下这条sql，只能局部命中索引的原因在于索引列使用了范围查询，且
范围查询的顺序相反，前两列都是小于等于的范围查询，最后一列是大于等于的范围查询，其查询顺序正好与前两列相反，因此只能
命中前两列的联合索引。


### rows
表示MySQL查询优化器估算出的为了得到查询结果需要扫描多少行记录，原则上rows是越少效率越高，可以直观的了解到SQL查询效率的高低。

### extra
表示很多额外信息，各种操作会在extra提示相关信息，常见的有:
Using where 表示存储引擎返回的数据还要在Server层做条件过滤，不等价于"需要回表"；
Using index 表示使用到了覆盖索引，不需要回表；
Using filesort 表示查询出来的结果需要额外排序，数据量小的在内存，大的话在磁盘排序，建议优化;
Using temporary 表示查询使用到了临时表，一般用于去重，分组等操作。

### 2.2.1 EXPLAIN ANALYZE（MySQL 8.0.18+）

`EXPLAIN` 只能告诉你优化器**计划**怎么执行，但计划和实际可能有差距。`EXPLAIN ANALYZE` 会**真正执行**
SQL 语句，并返回每个执行步骤的实际耗时、实际扫描行数等信息，是验证优化效果的终极工具。

```sql
EXPLAIN ANALYZE
SELECT id, c_name
FROM t_student
WHERE c_class_id = 6
ORDER BY c_name
LIMIT 20;
```

输出示例（MySQL 8.0 格式）：

```
-> Limit: 20 row(s)  (cost=4.25 rows=20) (actual time=0.152..0.168 rows=20 loops=1)
    -> Sort: t_student.c_name, limit input to 20 row(s) per chunk  (cost=4.25 rows=40) (actual time=0.151..0.162 rows=20 loops=1)
        -> Index lookup on t_student using idx_class_id (c_class_id=6)  (cost=4.25 rows=40) (actual time=0.087..0.118 rows=35 loops=1)
```

逐行解读：

- **最外层 Limit**：计划返回 20 行，实际也返回了 20 行（`actual rows=20`），耗时 0.168ms
- **Sort**：对结果按 c_name 排序，优化器估算输入 40 行（`rows=40`），实际输入 20 行排序后输出（内存排序，耗时 0.162ms）
- **Index lookup**：通过 idx_class_id 索引查找 c_class_id=6 的记录，优化器估算 40 行，实际扫描到 35 行（`actual rows=35`），耗时 0.118ms

关键字段含义：

| 字段 | 含义 |
|------|------|
| `cost=4.25` | 优化器估算的代价 |
| `rows=40` | 优化器估算的行数 |
| `actual time=0.087..0.118` | 实际耗时（首行时间..末行时间，单位 ms） |
| `actual rows=35` | 实际扫描/返回的行数 |
| `loops=1` | 该步骤执行了几次（在嵌套循环中可能 >1） |

**EXPLAIN 与 EXPLAIN ANALYZE 的对比使用**：

```
优化前：EXPLAIN ANALYZE 看到 actual rows=50000, actual time=320ms
加索引后：EXPLAIN ANALYZE 看到 actual rows=35, actual time=0.1ms
```

注意：`EXPLAIN ANALYZE` 会真正执行 SQL，如果是 UPDATE/DELETE 语句，会真的修改数据！
对于写操作，应该只用 `EXPLAIN`，不要用 `EXPLAIN ANALYZE`。

### 2.2.2 用证据链判断SQL是否真的优化了

建议用以下顺序判断，而不是只看某一个字段：

1. `EXPLAIN` 先看访问路径：`key`、`type`、`rows`、`Extra`；
2. `EXPLAIN ANALYZE` 看实际执行：`actual rows`、`loops`、每个算子的耗时；
3. 对比优化前后总耗时和扫描行数，确认收益；
4. 回到业务流量下做压测或灰度验证，避免只在小数据集上"看起来更快"。


## 2.3  Trace工具
在MySQL执行计划中我们发现明明这个字段建立了索引，但是有的sql不会走索引，这是因为MySQL的内部优化器认为走索引的性能比不走索引全表扫描的性能要差，
一个典型的场景是走索引查出来的数据量很大，然后还需要根据这些行记录去主键索引树回表查出完整数据，此时优化器会觉得得不偿失，不如直接全表扫描。
而优化器的选择逻辑，来自trace工具的结论。

```sql
set session optimizer_trace="enabled=on", end_markers_in_json=on;
select * from employees where name > 'a' order by position;
select * from information_schema.OPTIMIZER_TRACE;
```


```json
select * from t_student where c_class_id is not null | {
 "steps": [
  {
   "join_preparation": {
    "select#": 1,
    "steps": [
     {
      "expanded_query": "/* select#1 */ select `t_student`.`c_id` AS `c_id`,`t_student`.`c_name` AS `c_name`,`t_student`.`c_gender` AS `c_gender`,`t_student`.`c_phone` AS `c_phone`,`t_student`.`c_age` AS `c_age`,`t_student`.`c_address` AS `c_address`,`t_student`.`c_cardid` AS `c_cardid`,`t_student`.`c_birth` AS `c_birth`,`t_student`.`c_class_id` AS `c_class_id` from `t_student` where (`t_student`.`c_class_id` is not null)"
     }
    ] /* steps */
   } /* join_preparation */
  },
  {
   "join_optimization": {
    "select#": 1,
    "steps": [
     {
      "condition_processing": {
       "condition": "WHERE",
       "original_condition": "(`t_student`.`c_class_id` is not null)",
       "steps": [
        {
         "transformation": "equality_propagation",
         "resulting_condition": "(`t_student`.`c_class_id` is not null)"
        },
        {
         "transformation": "constant_propagation",
         "resulting_condition": "(`t_student`.`c_class_id` is not null)"
        },
        {
         "transformation": "trivial_condition_removal",
         "resulting_condition": "(`t_student`.`c_class_id` is not null)"
        }
       ] /* steps */
      } /* condition_processing */
     },
     {
      "substitute_generated_columns": {
      } /* substitute_generated_columns */
     },
     {
      "table_dependencies": [
       {
        "table": "`t_student`",
        "row_may_be_null": false,
        "map_bit": 0,
        "depends_on_map_bits": [
        ] /* depends_on_map_bits */
       }
      ] /* table_dependencies */
     },
     {
      "ref_optimizer_key_uses": [
      ] /* ref_optimizer_key_uses */
     },
     {
      "rows_estimation": [
       {
        "table": "`t_student`",
        "range_analysis": {
         "table_scan": {
          "rows": 40,
          "cost": 6.35
         } /* table_scan */,
         "potential_range_indexes": [
          {
           "index": "PRIMARY",
           "usable": false,
           "cause": "not_applicable"
          },
          {
           "index": "idx_class_id",
           "usable": true,
           "key_parts": [
            "c_class_id",
            "c_id"
           ] /* key_parts */
          }
         ] /* potential_range_indexes */,
         "setup_range_conditions": [
         ] /* setup_range_conditions */,
         "group_index_range": {
          "chosen": false,
          "cause": "not_group_by_or_distinct"
         } /* group_index_range */,
         "skip_scan_range": {
          "potential_skip_scan_indexes": [
           {
            "index": "idx_class_id",
            "usable": false,
            "cause": "query_references_nonkey_column"
           }
          ] /* potential_skip_scan_indexes */
         } /* skip_scan_range */,
         "analyzing_range_alternatives": { // 分析各个索引的使用成本
          "range_scan_alternatives": [
           {
            "index": "idx_class_id", // 索引名
            "ranges": [
             "NULL < c_class_id"
            ] /* ranges */,
            "index_dives_for_eq_ranges": true,
            "rowid_ordered": false,
            "using_mrr": false,
            "index_only": false, // 是否使用了覆盖索引
            "rows": 32,  // 要扫描的行数
            "cost": 11.46, // 要花费的时间
            "chosen": false, // 是否选择使用这个索引
            "cause": "cost" // 不选择这个索引的原因: 开销cost比较大
           }
          ] /* range_scan_alternatives */,
          "analyzing_roworder_intersect": {
           "usable": false,
           "cause": "too_few_roworder_scans"
          } /* analyzing_roworder_intersect */
         } /* analyzing_range_alternatives */
        } /* range_analysis */
       }
      ] /* rows_estimation */
     },
     {
      "considered_execution_plans": [
       {
        "plan_prefix": [
        ] /* plan_prefix */,
        "table": "`t_student`",
        "best_access_path": {  // 最优访问路径
         "considered_access_paths": [ // 最后选择的访问路径
          {
           "rows_to_scan": 40,  // 全表扫描行数
           "access_type": "scan", // 全表扫描
           "resulting_rows": 40, // 结果行数
           "cost": 4.25, // 花费时间
           "chosen": true // 是否选择这种方式
          }
         ] /* considered_access_paths */
        } /* best_access_path */,
        "condition_filtering_pct": 100,
        "rows_for_plan": 40,
        "cost_for_plan": 4.25,
        "chosen": true
       }
      ] /* considered_execution_plans */
     },
     {
      "attaching_conditions_to_tables": {
       "original_condition": "(`t_student`.`c_class_id` is not null)",
       "attached_conditions_computation": [
       ] /* attached_conditions_computation */,
       "attached_conditions_summary": [
        {
         "table": "`t_student`",
         "attached": "(`t_student`.`c_class_id` is not null)"
        }
       ] /* attached_conditions_summary */
      } /* attaching_conditions_to_tables */
     },
     {
      "finalizing_table_conditions": [
       {
        "table": "`t_student`",
        "original_table_condition": "(`t_student`.`c_class_id` is not null)",
        "final_table_condition  ": "(`t_student`.`c_class_id` is not null)"
       }
      ] /* finalizing_table_conditions */
     },
     {
      "refine_plan": [
       {
        "table": "`t_student`"
       }
      ] /* refine_plan */
     }
    ] /* steps */
   } /* join_optimization */
  },
  {
   "join_execution": {
    "select#": 1,
    "steps": [
    ] /* steps */
   } /* join_execution */
  }
 ] /* steps */
} 
```


## 2.3 各种具体场景下的优化

### 2.3.1 尽量使用覆盖索引避免回表操作
主键索引树叶子节点存储的是主键索引值和整行数据记录信息，非主键索引树叶子节点存储的是非主键索引值和主键索引值，
所以你使用非主键索引字段过滤查询条件时，如果你需要返回的字段都在非主键索引树上时，就不需要回表操作，否则就
需要回表，常见的如select *操作，这样会导致查询性能降低。
譬如一个student表，id为主键且自增，name和age字段上建立了联合索引，此外还有gender, address等其他字段，那么:

```sql
select id, name, age from student where name = "张三" and age > 20;
select id, name, age, gender from student where name = "张三" and age > 20;
select * from student where name = "张三" and age > 20;
```

显然第一条sql语句使用到了覆盖索引，在非主键索引树上就能获取到所有信息，不需要回表，性能是ok的；而第二条和第三条需要
获取非主键索引树上id, name, age之外其他字段gender, address信息，需要回表，性能较差。

### 2.3.2 遵循最左匹配原则，用好联合索引，提高索引命中率
联合索引使用时需要遵循最左匹配原则，如果最左边的列是多个查询条件中的第一个，便可以命中联合索引，反之如果从联合索引中的
第二列及以后开始匹配查询条件，则联合索引会失效。

```sql
建立a,b,c三列的联合索引 
create index idx_a_b_c on table1(a, b, c)
能命中索引
select * from table1 where a = 10;
能命中索引
select * from table1 where a = 10 and b = 20;
能命中索引
select * from table1 where a = 10 and b = 20 and c = 30;
无法命中索引
select * from table1 where b = 10;
无法命中索引
select * from table1 where b = 10 and c = 30;
部分命中索引，可以命中a这列索引
select * from table1 where a = 10 and c = 30;
无法命中索引
select * from table1 where c = 30;
能命中索引，MySQL优化器会做内部优化，调整b和c的顺序，使其走联合索引
select * from table1 where a = 10 and c = 30 and b = 20
```

```sql
alter table t_student add index u_key (c_age, c_name, c_address);
```

需要注意的是，如果联合索引列上使用范围查询的顺序不一致，会导致联合索引使用不充分。
具体的 key_len 计算方法和这个例子的详细分析，参见前面 2.2 节 Explain 工具的 key_len 部分，此处不再重复。

如果是`select *`返回全表字段，通常会出现回表，但这不等于索引失效。  
是否使用索引，要看优化器是否仍然选择该索引作为访问路径。





![explain-four.png](images%2Fexplain-four.png)






联合索引第一列是范围查询时，通常仍可使用该索引做范围扫描；只是后续列在过滤/排序上的利用会受限，不应表述为"索引失效"。





![explain-five.png](images%2Fexplain-five.png)





### 2.3.3 注意索引失效情况，根据具体场景做优化

#### 2.3.3.1 模糊查询时，%字符在查询条件开头会导致索引失效
>优化: 尽量不要把%字符放在开头，如果确有此类后缀查询业务需求，可以考虑将数据同步到ES做查询。

#### 2.3.3.2 在索引列上使用函数表达式或运算会导致索引失效
譬如:
```sql
explain select * from employees where date(hire_time) = '2022-04-22';
```

> 优化: 可以转化成范围查询

```sql
explain select * from employees where hire_time >= '2022-04-22 00:00:00' and hire_time <= '2022-04-22 23:59:59';
```

再比如:
```sql
SELECT * FROM table WHERE DATE(created_at) = DATE(NOW() - INTERVAL 1 DAY);
```

可以转化为
```sql
SELECT * FROM table WHERE created_at BETWEEN CURDATE() - INTERVAL 1 DAY AND CURDATE() - INTERVAL 1 SECOND;
```





![explain-six.png](images%2Fexplain-six.png)





#### 2.3.3.3 使用is null, is not null不一定会全表扫描

```sql
explain select * from t_student where c_class_id = 6;
explain select * from t_student where c_class_id is not null;
```





![explain-seven.png](images%2Fexplain-seven.png)





> 优化: 是否走索引取决于数据分布与选择性。`IS NULL`在很多场景可以走索引；`IS NOT NULL`如果命中行数过大，优化器可能选择全表扫描。字段应按业务语义定义NOT NULL，而不是仅为规避该写法。


#### 2.3.3.4 数据类型类型不一致会使得sql语句执行时进行隐式的类型转换最终导致索引失效





![explain-eight.png](images%2Fexplain-eight.png)





mysql默认会将字符串转化为数字，要验证这一点很简单，连接mysql后在客户端执行以下指令即可：





![verify-demo.png](images%2Fverify-demo.png)





输出1表示true, 意味着MySQL默认会将字符串"10"转化为数字10与数字9进行比较。所以，如果你对数据表中的一个字符串
类型的字段建立了索引，然后对这个索引列数据与int类型数字进行判等或大小范围操作时，MySQL就会将这个索引列数据都
转化为数字再进行比较，显然会导致索引失效。
相反，如果你对一个int类型的字段建立了索引，然后对这个索引列数据与字符串类型的数字(类似"10"这种)进行比较操作，
那么MySQL只需要将这个字符串类型的数字转化为int类型的数字和索引列数据比较就好了，查询扫描时仍然是走索引的。

譬如下面这个例子，只扫描两行数据，过滤率很高，属于走索引的典型情况。





![explain-nine.png](images%2Fexplain-nine.png)





#### 2.3.3.5 OR/IN/NOT IN/EXISTS 的优化要看执行计划，不要一刀切
`OR`、`IN`、`NOT IN`、`EXISTS/NOT EXISTS`并不天然导致索引失效，关键在于：
- 过滤条件是否可利用索引；
- 子查询是否可半连接优化（semi join）或反连接优化；
- 谓词选择性是否足够高；
- NULL语义是否让优化器保守执行。

实践建议：
- 对`IN (subquery)`、`EXISTS`、`NOT EXISTS`都跑`EXPLAIN ANALYZE`，选择扫描行数更少、耗时更低的写法；
- `NOT IN`要特别注意子查询返回NULL时的语义变化，必要时先显式过滤NULL；
- `OR`前后如果都能走索引，优化器可能做index merge；如果不能，考虑拆成`UNION ALL`再去重。


### 2.3.4 范围查询优化





![explain-ten.png](images%2Fexplain-ten.png)





![explain-eleven.png](images%2Fexplain-eleven.png)





优化: 将一个很大的查询范围拆分为多个小范围查询结果集之和。虽然需要多次查询，但在命中索引且每次扫描行数可控时，整体耗时通常优于一次大范围扫描。





### 2.3.5 多表连接查询优化

#### JOIN 算法详解

MySQL 的 JOIN 执行算法经历了多个版本的演进，理解不同算法有助于针对性优化。

**Nested Loop Join（NLJ，嵌套循环连接）**

最基础的 JOIN 算法。从驱动表逐行读取，对于每一行到被驱动表中查找匹配行。
如果被驱动表的关联字段有索引，每次查找是一次 B+ 树搜索（O(log n)），整体效率还可以接受。

```
伪代码：
for each row r1 in 驱动表:
    通过索引在被驱动表中查找满足 join 条件的行 r2
    输出 (r1, r2)
```

**Block Nested Loop Join（BNL，块嵌套循环，MySQL 5.6-5.7）**

当被驱动表的关联字段没有索引时，NLJ 每次都要全表扫描被驱动表，代价极高。
BNL 的优化是引入 `join_buffer`：先把驱动表的一批行放入 join_buffer，然后扫描一次被驱动表，
和 buffer 中的所有行做匹配，减少被驱动表的全表扫描次数。

```
伪代码：
将驱动表的一批行装入 join_buffer
for each row r2 in 被驱动表:
    与 join_buffer 中的每一行做匹配
    输出匹配的行对
```

`join_buffer_size` 默认 256KB，可以适当调大来减少扫描次数。

**Hash Join（MySQL 8.0.18+）**

MySQL 8.0.18 引入了 Hash Join，替代了 BNL。当被驱动表没有可用索引时，优化器会选择 Hash Join：
1. 对较小的表（build 端）在内存中构建一个哈希表
2. 扫描较大的表（probe 端），用哈希表做匹配

Hash Join 的优势在于匹配是 O(1) 的哈希查找，而不是 BNL 中的逐行比较，大数据量下性能好很多。

```sql
-- 可以通过 EXPLAIN FORMAT=TREE 查看是否使用了 Hash Join
EXPLAIN FORMAT=TREE
SELECT * FROM t1 INNER JOIN t2 ON t1.a = t2.a;
```

如果输出中包含 `-> Hash join`，说明使用了 Hash Join。

**三种算法对比：**

| 算法 | 适用条件 | 版本 | 性能特点 |
|------|---------|------|---------|
| NLJ | 被驱动表关联字段有索引 | 所有版本 | 好（走索引，O(n * log m)） |
| BNL | 被驱动表无索引 | 5.6-8.0.17 | 较差（减少了扫描次数，但仍是逐行比较） |
| Hash Join | 被驱动表无索引 | 8.0.18+ | 较好（O(n + m)，但需要内存构建哈希表） |

**实战建议：**

```sql
explain select * from t1 inner join t2 on t1.a = t2.a
```

t1 是大表（一万条记录），t2 是小表（一百条记录）。在 inner join 查询中，如果关联字段建立了可用索引，
通常会使用 NLJ（含 BKA 等变体），先从驱动表读取一批行，再到被驱动表按索引查找。驱动表选择由优化器
根据代价决定，不应只看 SQL 书写顺序。

还有一个常见坑：**连接字段的字符集/排序规则/数据类型不一致**，可能触发隐式转换导致索引失效。
例如 t1.a 是 `utf8mb4` 而 t2.a 是 `utf8`，JOIN 时 MySQL 会对 t2.a 做隐式转换，
导致 t2 上的索引无法使用，退化为全表扫描。

因此，JOIN 键建议统一数据类型、字符集、排序规则，并优先统一为 utf8mb4 体系。

关联字段没建立索引前是全表扫描 ALL





![explain-twelve.png](images%2Fexplain-twelve.png)





关联字段建立索引后则是ref





![explain-thirteen.png](images%2Fexplain-thirteen.png)





### 2.3.6 小表驱动大表优化
>在使用多表连接查询时，inner join通常更容易被优化器重排；left/right join受语义约束，重排空间更小，确实要关注表顺序和索引设计。
但"必须小表在左/右"不是绝对规则，最终以执行计划和实际耗时为准。

>in 和 exists没有绝对优劣，通常要看子查询是否可改写成半连接、是否命中索引以及结果集基数。
推荐同时测试两种写法，保留EXPLAIN ANALYZE成本更低的一种。

譬如:
```sql
select * from bigTable where id in (select id from smallTable)
```

相反，如果是左表为小表，右表为大表的场景，那就应该使用exists查询，譬如:
```sql
select * from smallTable where exists (select 1 from bigTable where bigTable.id = smallTable.id)
```

### 2.3.7 深分页优化

深分页是一个非常常见的性能问题。当 OFFSET 很大时，MySQL 需要扫描 OFFSET + LIMIT 行数据，然后丢弃前 OFFSET 行，只返回 LIMIT 行。
OFFSET 越大，浪费的扫描就越多。

```sql
-- 当 offset=10000 时，MySQL 实际要扫描 10010 行，丢弃前 10000 行
explain select * from employees limit 10000, 10
```

有以下三种优化方案：

**方案一：主键连续时直接用 WHERE 条件**

如果主键索引是连续的（自增且没有删除），可以直接用 WHERE 代替 OFFSET：

```sql
explain select * from employees where id > 10000 limit 10;
```

这样 MySQL 直接从 id=10001 开始扫描，只需要扫描 10 行。

**方案二：延迟关联（Deferred Join）**

如果主键索引不连续，可以先在子查询中利用覆盖索引快速定位到目标主键，再回表获取完整数据：

```sql
select * from employees a
inner join (select id from employees limit 10000, 10) b
on a.id = b.id
```

子查询 `select id from employees limit 10000, 10` 只需要扫描主键索引（覆盖索引），不需要回表，
速度快得多。定位到 10 个 id 后，再通过 JOIN 回表获取完整数据，只回表 10 次。





![explain-fourteen.png](images%2Fexplain-fourteen.png)




**方案三：游标分页（Cursor-based Pagination，推荐）**

游标分页是实际业务中最推荐的深分页方案。核心思路是：记住上一页最后一条记录的排序键值，
下一页从该值开始查询，彻底消除 OFFSET。

```sql
-- 第一页：正常查询
SELECT * FROM orders WHERE user_id = 100 ORDER BY id LIMIT 20;
-- 假设第一页最后一条记录的 id = 258

-- 第二页：用上一页最后一条的 id 作为游标
SELECT * FROM orders WHERE user_id = 100 AND id > 258 ORDER BY id LIMIT 20;
-- 假设第二页最后一条记录的 id = 312

-- 第三页：
SELECT * FROM orders WHERE user_id = 100 AND id > 312 ORDER BY id LIMIT 20;
```

游标分页的优势：
- 无论翻到第多少页，每次查询只扫描 LIMIT 行，性能恒定
- 充分利用索引（`WHERE user_id=100 AND id>258` 可以走 `(user_id, id)` 的联合索引）

游标分页的局限：
- 不支持"跳页"（不能直接跳到第 100 页），只能上一页/下一页
- 需要前端配合传递游标值（上一页最后一条记录的排序键）
- 排序键必须是唯一的（否则可能漏数据），如果排序字段不唯一，需要加上主键作为二级排序

```sql
-- 如果按 created_at 排序，created_at 可能重复，需要加上 id 作为二级排序
SELECT * FROM orders
WHERE user_id = 100
  AND (created_at, id) > ('2024-01-15 10:00:00', 258)
ORDER BY created_at, id
LIMIT 20;
```

**三种方案的适用场景：**

| 方案 | 适用场景 | 局限 |
|------|---------|------|
| WHERE id > N | 主键连续自增 | 有删除的表不适用 |
| 延迟关联 | 需要支持跳页的管理后台 | OFFSET 极大时仍有性能瓶颈 |
| 游标分页 | 移动端/信息流/无限滚动 | 不支持跳页 |



### 2.3.8 Count 优化
对于count的优化应该是架构层面的优化，因为count的统计在一个产品中会经常出现，而且每个用户都会访问，所以对于访问
频率过高的数据建议维护在缓存中。

### 2.3.9 order by 优化
如果explain中的Extra信息出现了Using filesort意味着sql语句执行时进行了文件排序，原因当然是没有命中索引，优化方案
是让排序字段遵循最左匹配原则，避免文件排序。
order by多个列排序在遵循联合索引的最左匹配原则时，是可以走索引的；或者where条件列与order by列合在一起遵循
最左匹配原则，也是可以命中索引的；如果中间有断层，比如跳过了第二个字段，也是可以命中索引的，只是排序时效率比较低，需要
走一次filesort文件排序。





![explain-fifteen.png](images%2Fexplain-fifteen.png)





那么，什么时候索引会失效呢？
> 即使使用select *导致回表，也不等于索引失效；但回表会增加随机IO，可能让优化器改选其他执行路径。





![explain-sixteen.png](images%2Fexplain-sixteen.png)





> 对不同的索引列做order by会导致索引失效

譬如下面这个案例c_age对应了(c_age,c_name,c_address)三个列的联合索引，c_class_id对应了c_class_id列的
索引，对这两个索引列做order by排序就会导致索引失效，因为这种情况MySQL无法做到这两列数据的全局有序。





![explain-seventeen.png](images%2Fexplain-seventeen.png)





> 不遵循最左匹配原则，8.0及以上版本做了优化，可以走索引，但是排序时效率比较低，需要走一次filesort文件排序。





![explain-eighteen.png](images%2Fexplain-eighteen.png)





>  order by多个排序字段排序顺序不同会导致索引失效，但是8.0及以上版本做了优化，可以走索引，但是排序时效率
比较低，需要走一次filesort文件排序。





![explain-nineteen.png](images%2Fexplain-nineteen.png)





> filesort 文件排序的原理

当 MySQL 无法利用索引完成排序时，就会使用 filesort。filesort 并不一定意味着使用了磁盘文件，
在数据量小的时候是在内存中完成排序的。

**两种排序算法：**

MySQL 会比较查询需要返回的所有列的总长度与系统变量 `max_length_for_sort_data` 的大小（默认 1024 字节）：

- **单路排序（全字段排序）**：如果总长度 ≤ `max_length_for_sort_data`，把需要返回的所有字段都放入
  `sort_buffer` 中排序，排完直接返回结果，不需要回表。
- **双路排序（rowid 排序）**：如果总长度 > `max_length_for_sort_data`，只把排序字段和主键放入
  `sort_buffer`，排完后再根据主键回表获取其他字段。多了一次回表，但 `sort_buffer` 能装下更多行。

**sort_buffer 的作用：**

`sort_buffer_size` 参数控制每个连接可用于排序的内存大小（默认 256KB）。
- 如果需要排序的数据量 ≤ `sort_buffer_size`，在内存中完成排序
- 如果数据量 > `sort_buffer_size`，MySQL 会使用磁盘临时文件做外部排序（归并排序），性能急剧下降

```sql
-- 查看 sort_buffer 大小
SHOW VARIABLES LIKE 'sort_buffer_size';

-- 适当调大（注意这是 per-connection 的，不是全局共享的）
-- 如果并发连接很多，调太大会导致内存不够
SET SESSION sort_buffer_size = 2 * 1024 * 1024;  -- 调大到 2MB
```

**监控排序情况：**

```sql
-- 查看排序相关的状态变量
SHOW STATUS LIKE 'Sort_%';
```

| 变量 | 含义 |
|------|------|
| `Sort_merge_passes` | 使用磁盘临时文件做归并排序的次数，如果这个值很大，说明 sort_buffer 太小 |
| `Sort_rows` | 已排序的行数 |
| `Sort_scan` | 通过全表扫描完成的排序次数 |
| `Sort_range` | 通过范围扫描完成的排序次数 |

如果 `Sort_merge_passes` 持续增长，说明大量排序操作溢出到了磁盘，应该考虑：
1. 给排序字段加索引，让 ORDER BY 走索引而不走 filesort
2. 适当调大 `sort_buffer_size`
3. 减少 SELECT 的字段数量（让更多行能放进 sort_buffer）

### 2.3.10 MySQL优化器有可能走错了索引，需要手动纠正，可以通过force index 指定索引。

```sql
explain select * from user force index(idx_age) where age=60;
```

### 2.3.11 连接数过小
MySQL的server层里有一个连接管理，它的作用是管理客户端和MySQL之间的长连接。如果客户端与server层只有一条连接，
那么在执行SQL查询后，只能阻塞等待结果返回，如果有大量查询同时并发请求，那么后面的请求都需要等到前面的请求执行完成，才能开始执行。
连接数过小的问题，受数据库和客户端两侧同时限制。

**数据库连接数过小**
MySQL最大连接数默认是100，最大可以达到16384。
可以通过设置MySQL的max_connections参数，更改数据库的最大连接数。

```sql
set global max_connections = 500;
show variables like 'max_connections';
```

**应用层连接数过小**
MySQL客户端，也就是应用层与MySQL底层的连接，是基于TCP协议的长连接，而TCP协议，需要经过三次握手和四次挥手来实现连接
建立和关闭。如果每次执行SQL都重新建立一个新的连接的话，那就要不断的握手和挥手，非常耗时。所以一般会建立一个长连接池，
连接用完后，再塞回到连接池里，下次要执行SQL时，再从里面捞一条连接出来，实现连接复用，避免频繁通过握手和挥手建立和关闭连接。

一般的ORM库都会实现连接池，譬如gorm是这么设置连接池的。

```go
func Init() {
    db, err := gorm.Open(mysql.Open(conn), config)
    sqlDB, err := db.DB()
    // 设置空闲连接池中连接的最大数量
    sqlDB.SetMaxIdleConns(200)
    // 设置打开数据库的最大连接数量
    sqlDB.SetMaxOpenConns(1000)
}
```


### 2.3.12  buffer pool太小
在数据库查询流程里，在InnoDB存储引擎取数据时，为了加速，会有一层内存buffer pool， 用于将磁盘数据页加载到内存页
中，只要查询到buffer pool 里有，就可以直接返回，速度就很快了，否则就得走磁盘IO，那就慢了。
也就是说，如果我的buffer pool越大，那我们在其中存放的数据页就越多，相应的，SQL查询时就更可能命中buffer pool, 
查询速度自然就更快了。

可以通过下面的命令查询bp的大小，单位是Byte。
```sql
show global variables like 'innodb_buffer_pool_size';
```
可以通过下面的指令调大一些:
```sql
set global innodb_buffer_pool_size = 536870912;
```
这样就把bp调大到512Mb了。

问题又来了，怎么知道buffer pool 是不是太小了？
这个我们可以看看buffer pool 的缓存命中率

```sql
// 查看bp相关信息
show status like "Innodb_buffer_pool_%";
其中Innodb_buffer_pool_read_requests 表示读请求的次数
Innodb_buffer_pool_reads 表示从物理磁盘中读取数据的请求次数。
所以bp的命中率可以通过以下公式得到
rate = 1 - (Innodb_buffer_pool_reads/Innodb_buffer_pool_read_requests) * 100%
```

一般情况下，bp的命中率都在99%以上，如果低于这个值，就需要考虑加大bp的值了。比较好的做法是将这个bp命中率指标加到监控里，
这样晚上SQL查询慢发了邮件告警，第二天早上上班查看邮件就能定位到原因，很nice。


### 2.3.13 group by 优化
> 在MySQL 5.7等旧版本里，group by可能伴随额外排序，常见写法是显式加order by null；在MySQL 8.0里默认行为已有变化，不要把这条当作通用强规则。
> 尽量让group by 过程用上表的索引(对分组字段建立索引)，确认方法是explain的extra里没有出现Using temporary和Using filesort。
> 如果group by 需要统计的数据量不大，尽量使用内存临时表sort buffer；也可以通过适当调大tmp_table_size参数，来避免用到磁盘临时表。
> 如果数据量是在太大，使用SQL_BIG_RESULT这个提示，来告诉优化器直接使用排序算法得到group by的结果。





![explain-twenty.png](images%2Fexplain-twenty.png)





![explain-twentyone.png](images%2Fexplain-twentyone.png)


# 3 版本差异与调优方法

## 3.1 MySQL 5.7 vs 8.0 差异速查

| 主题 | MySQL 5.7 常见行为 | MySQL 8.0 常见行为 | 实践建议 |
|---|---|---|---|
| 默认字符集 | 常见是utf8mb4(由发行版配置决定) | utf8mb4 | 新系统统一utf8mb4，避免utf8mb3遗留 |
| JOIN算法 | 以NLJ/BNL等为主 | 在部分场景可选Hash Join | 用EXPLAIN ANALYZE确认最终算法 |
| GROUP BY与排序 | 容易遇到"顺带排序"的历史认知 | 行为与旧版本有差异 | 是否加order by null要按版本和执行计划决定 |
| 索引长度限制 | 与行格式/参数强相关 | 与行格式/参数强相关 | 不要写固定数值结论，按版本文档+DDL验证 |
| 优化器能力 | 相对保守 | 谓词、连接、排序优化更丰富 | 规则要条件化，避免绝对化口号 |

## 3.2 条件化结论模板（替代口号）

把“总是/必须/一定”改成以下格式更可靠：

1. 前提：数据量、索引、版本、字符集/排序规则；
2. 现象：`EXPLAIN`/`EXPLAIN ANALYZE`看到什么；
3. 结论：该场景为什么快/慢；
4. 反例：在什么条件下该结论不成立；
5. 验证：上线前用真实流量模型压测确认。

示例：

- 不推荐写：`is not null一定全表扫描`；
- 推荐写：`is not null在低选择性场景可能全表扫描，是否走索引以EXPLAIN ANALYZE为准`。

# 4 SQL 优化的边界：何时该引入异构数据源

SQL 优化不是万能的。MySQL 作为 OLTP 型关系数据库，其核心设计目标是**事务处理**（高并发的短事务读写），
而不是复杂的多维度搜索。当你发现以下场景时，应该在架构层面考虑引入 Elasticsearch 等异构数据源来承担查询职责。

## 4.1 索引不是免费的：写放大问题

每新增一个二级索引，InnoDB 在执行 INSERT / UPDATE / DELETE 时都需要额外维护该索引的 B+ 树有序性。
具体代价包括：

- **写放大**：一次业务写入变成 1（聚簇索引）+ N（N 个二级索引）次 B+ 树写操作
- **页分裂概率增大**：索引越多，随机写导致的页分裂越频繁，空间利用率下降
- **Change Buffer 压力**：二级索引的变更先写入 Change Buffer，索引越多，Buffer 占用越大，merge 越频繁
- **DDL 成本**：添加/删除索引需要重建表（`ALGORITHM=INPLACE` 也需要遍历全量数据）

因此，OLTP 系统通常建议**单表索引不超过 5-6 个**，联合索引字段不超过 3-4 个。
这是一个工程权衡：索引太少查询慢，索引太多写入慢。

## 4.2 MySQL 力不从心的查询场景

以下场景下，无论怎么优化 SQL 和索引，MySQL 都难以提供令人满意的性能：

### 4.2.1 多维度组合查询

电商搜索场景：用户可以按品牌、价格区间、颜色、尺码、好评率、销量、发货地……数十个维度任意组合筛选。

```sql
-- 假设有 10 个筛选维度，用户可能使用其中任意 2-5 个
SELECT * FROM products
WHERE brand = 'Apple'
  AND price BETWEEN 1000 AND 5000
  AND color IN ('黑色', '白色')
  AND rating >= 4.5
ORDER BY sales DESC
LIMIT 20;
```

如果要为所有可能的查询组合建索引，索引数量会爆炸（10 个字段的组合数是 $C_{10}^2 + C_{10}^3 + ... = 1013$）。
实际上不可能建这么多索引，也承受不了这些索引带来的写放大。

### 4.2.2 前缀模糊匹配 / 全文检索

```sql
-- 前缀模糊：% 在开头，B+ 树索引完全无法使用
SELECT * FROM articles WHERE title LIKE '%分布式%';

-- 全文搜索：即使 MySQL 5.6+ 支持 InnoDB FULLTEXT INDEX，
-- 其分词能力（尤其是中文）、相关性排序、性能都远不如专业搜索引擎
SELECT * FROM articles WHERE MATCH(content) AGAINST('微服务架构' IN BOOLEAN MODE);
```

B+ 树索引是按照键值**从左到右**有序排列的，`LIKE '%keyword%'` 无法利用这种有序性，
必然退化为全表扫描。MySQL 的 FULLTEXT INDEX 虽然能应对简单场景，但在中文分词、同义词扩展、
相关性打分、聚合分析等方面远不如 Elasticsearch。

### 4.2.3 需要复杂聚合与分析的查询

```sql
-- 按多个维度做聚合统计
SELECT category, brand, DATE(created_at),
       COUNT(*), AVG(price), SUM(sales)
FROM products
WHERE created_at >= '2024-01-01'
GROUP BY category, brand, DATE(created_at);
```

这类分析型查询（OLAP）涉及大量数据的扫描和聚合，MySQL 的行存储引擎天然不擅长。

## 4.3 MySQL + ES 的协作架构

在实际生产环境中，常见的做法是 **MySQL 做写入和事务，ES 做复杂查询**：

```
                  ┌──── 写入 ────→ MySQL（主库，OLTP）
                  │                    │
    业务层 ───────┤                    │ binlog / 业务双写 / Canal
                  │                    ↓
                  └──── 查询 ────→ Elasticsearch（搜索/聚合）
```

**数据同步方案：**

| 方案 | 原理 | 优点 | 缺点 |
|------|------|------|------|
| 业务双写 | 写 MySQL 后同步写 ES | 实现简单 | 一致性难保证，业务侵入性强 |
| 监听 binlog | 用 Canal / Debezium 监听 MySQL binlog，异步写入 ES | 对业务无侵入，一致性较好 | 有秒级延迟，需维护同步组件 |
| 定时全量/增量同步 | 定时任务扫描 MySQL 变更写入 ES | 实现简单 | 延迟大，不适合实时场景 |

生产环境推荐 **Canal / Debezium 监听 binlog** 的方案，对业务代码零侵入，延迟通常在秒级以内。

**适用边界判断：**

| 判断维度 | 继续用 MySQL | 考虑引入 ES |
|----------|-------------|-------------|
| 查询维度数量 | ≤ 3 个，可用联合索引覆盖 | > 5 个，任意组合 |
| 模糊匹配 | 后缀匹配 `LIKE 'abc%'` | 前缀/中间匹配 `LIKE '%abc%'` |
| 全文检索 | 不需要 | 需要分词、相关性排序 |
| 数据量 | 单表百万级以内 | 单表千万级以上的复杂查询 |
| 一致性要求 | 强一致（事务） | 可接受秒级延迟 |
| 写入频率 | 高频写入，索引少 | 读多写少 |

> **核心原则：MySQL 保证数据正确性（ACID 事务），ES 解决查询灵活性。二者是互补关系，不是替代关系。**
> 在业务早期数据量小、查询简单时，MySQL 单体足够；当查询复杂度和数据量增长到 MySQL 索引方案无法兼顾读写性能时，
> 就是引入 ES 的合理时机。这是一个架构演进决策，不是 SQL 优化能解决的问题。
