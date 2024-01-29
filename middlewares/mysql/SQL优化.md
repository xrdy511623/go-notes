---
SQL优化
---

# 1 工程优化

## 1.1 基础规范

表存储引擎必须使用InnoDB
表字符集默认使用utf8, 必要时使用utf8mb4
utf8是通用的，无乱码风险，汉字三个字节，英文1个字节；
utf8mb4是utf8的超集，需要存储4字节的数据如表情符号时，使用它。

禁止使用存储过程，视图，触发器，Event
对数据库性能影响较大，互联网业务，能让站点层和服务层干的事情，不要交到数据库层调试，排错，再者迁移比较困难，扩展性较差。
禁止在数据库中存储大文件，例如图片，音频，视频，可以将大文件存储在对象存储系统(OSS)，数据库中存储路径即可。
禁止在线上环境做数据库压力测试。
测试，开发，线上数据库环境必须隔离。

## 1.2 命名规范

库名，表名，列名必须用小写，采用下划线分隔tb_book或者t_book；
abc, Abc， ABC都是给自己埋坑。
库名，表名，列名必须见名知义，长度不要超过32字符；
Tmp, wushan谁TM知道这些库是干嘛的
库备份必须以bak为前缀，以日期为后缀。
从库必须以-s为后缀。
备库必须以-ss为后缀。

## 1.3 表设计规范

单实例表个数必须控制在2000个以内。
单表分表个数必须控制在1024个以内。
表必须有主键，推荐使用UNSIGNED无符号整数为主键。
删除无主键的表，如果是row模式的主从架构，从库会挂。
禁止使用外键，如果要保证数据完整性，应由应用程序实现，譬如前端设计一个下拉框，限制用户输入。
因为外键会使表之间相互耦合，影响update/delete等SQL性能，有可能造成死锁，高并发情况下容易成为数据库瓶颈。
建议将大字段，访问频度低的字段拆分到单独的表中存储，分离冷热数据。

水平拆分和垂直拆分
水平切分是指，以某个字段为依据（例如uid），按照一定规则（例如取模），将一个库（表）上的数据拆分到多个库
（表）上，以降低单库（表）大小，达到提升性能的目的方法，水平切分后，各个库（表）的特点是：
（1）每个库（表）的结构都一样。
（2）每个库（表）的数据都不一样，没有交集。
（3）所有库（表）的并集是全量数据。
垂直拆分是指，将一个属性较多，一行数据较大的表，将不同的属性拆分到不同的表中，以降低单库（表）大小，达到提升
性能的目的的方法，垂直切分后，各个库（表）的特点是：
（1）每个库（表）的结构都不一样
（2）一般来说，每个库（表）的属性至少有一列交集，一般是主键。
（3）所有库（表）的并集是全量数据。
垂直切分的依据是什么？
当一个表属性很多时，如何来进行垂直拆分呢？如果没有特殊情况，拆分依据主要有几点：
（1）将长度较短，访问频率较高的属性尽量放在一个表里，这个表暂且称为主表。
（2）将字段较长，访问频率较低的属性尽量放在一个表里，这个表暂且称为扩展表。
如果1和2都满足，还可以考虑第三点：
（3）经常一起访问的属性，也可以放在一个表里。
优先考虑1和2，第3点不是必须。另，如果实在属性过多，主表和扩展表都可以有多个。

（1）水平拆分和垂直拆分都是单表降低数据量大小，提升数据库性能的常见手段。
（2）流量大，数据量大时，数据访问要有service层，并且service层不要通过join来获取主表和扩展表的属性
（3）垂直拆分的依据，尽量把长度较短，访问频率较高的属性放在主表里。

## 1.4 列设计规范

根据业务区分tinyint/int/bigint，分别占用1/4/8字节。
根据业务区分使用char和varchar:
如果字段长度固定，或者长度相似的业务场景，适合使用char，能够减少碎片，查询性能高；
如果字段长度相差较大，或者更新较少的业务场景，适合使用varchar，能够节省存储空间。
根据业务区分使用datetime和timestamp:
前者占用5个字节，后者占用4个字节，存储年使用YEAR，存储日期使用DATE，存储时间使用datetime。
必须把字段定义为NOT NULL, 并设置默认值。
因为NULL列使用索引，索引统计，值都更加复杂，MySQL更难优化。
NULL列需要更多的存储空间。
NULL只能采用IS NULL 或IS NOT NULL，而在!, in, not in 时有大坑。
使用INT UNSIGNED存储IPv4, 不要使用char(15)。
使用varchar(20)存储手机号，不要使用整数。
牵扯到国家代号，可能出现+/-/()等字符，例如+86
而且手机号不会用来做数学运算
varchar可以做模糊查询，例如like '138%'
使用TINYINT来代替ENUM。
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
不建议在频繁更新的字段上建立索引，因为维护索引有序也是有代价的。
不建议在区分度低的字段上建立索引，譬如gender性别字段，建立索引有额外的存储成本。
非必要不要进行JOIN联表查询，如果要进行JOIN查询，被JOIN的字段必须类型相同，并建立索引。
有没有踩过因为被JOIN的字段类型不同, 导致索引失效最终引发全表扫描的坑吗？
理解联合索引最左前缀原则，避免重复建设索引，如果建立了(a,b,c)三个字段的联合索引，相当于建立了(a), (a,b), (a,b,c)索引。

## 1.6 SQL规范

禁止使用 select *，只获取必要字段
select * 会增加 cpu/io/内存/带宽 的消耗
指定字段能有效利用覆盖索引
指定字段查询，在表结构变更时，能保证对应用程序无影响
insert 必须指定字段，禁止使用 insert into T values()
指定字段插入，在表结构变更时，能保证对应用程序无影响
隐式类型转换会使索引失效，导致全表扫描
禁止在 where 条件列使用函数或者表达式
导致不能命中索引，全表扫描
禁止负向查询以及 % 开头的模糊查询
导致不能命中索引，全表扫描
禁止大表 JOIN 和子查询
同一个字段上的 OR 必须改写为IN，IN 的值必须少于 50 个
应用程序必须捕获 SQL 异常
方便定位线上问题

# 2 SQL语句优化

## 2.1 Explain工具

![explain-two.png](images%2Fexplain-two.png)

### id

表示执行顺序，数字越大的先执行，如果数字相等，则排在上面的sql语句先执行。以上面的sql语句为例，显然是id为2的子查询
sql语句(select id from tb_areas where title = "成都市")先执行，然后再执行外层的主查询sql语句。

select_type
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
ALL: 表示全表扫描，性能最差
index: 表示基于索引的全表扫描，先扫描索引树再扫描全表数据
range: 表示使用索引范围查询，常见于>, >=, <, <=, in等等
ref: 表示使用非唯一索引进行扫描
eq_ref: 表示使用唯一索引或主键索引扫描(需要扫描，匹配多次，常见于多表连接查询)
const: 表示使用主键或唯一索引做等值查询，常量查询
NULL: 表示不用访问表，速度最快

**为了提高查询性能，我们需要type属性的值是range及以上。**

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
| (utfbmb4) char(n)    | 4*n     | 4*n      | 4*n+1   |
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
索引最大长度为768个字节，当字符串过长时，MySQL会做一个类似左前缀索引的处理，将前半部分的字符提取出来作为索引。





![create-table-one.png](images%2Fcreate-table-one.png)





![explain-three.png](images%2Fexplain-three.png)





由表结构可知，c_age，c_name, c_address的联合索引的长度，也就是key_len=1+1+30*4+2+1+100*4+2+1=2+123+403=528, 
上图第一条sql使用到了联合索引u_key且使用的长度为528字节，说明使用索引充分，完全命中了联合索引u_key；但是第二条sql
就使用索引不充分了，从key_len=125字节推算，它只使用了c_age和c_name的部分索引，没有用到c_address列的索引，
key_len=1+1+3*40+2+1=2+123=125， 此时我们分析一下这条sql，只能局部命中索引的原因在于索引列使用了范围查询，且
范围查询的顺序相反，前两列都是小于等于的范围查询，最后一列是大于等于的范围查询，其查询顺序正好与前两列相反，因此只能
命中前两列的联合索引。


### rows
表示MySQL查询优化器估算出的为了得到查询结果需要扫描多少行记录，原则上rows是越少效率越高，可以直观的了解到SQL查询效率的高低。

### extra
表示很多额外信息，各种操作会在extra提示相关信息，常见的有:
Using where 表示需要回表查询；
Using index 表示使用到了覆盖索引，不需要回表；
Using filesort 表示查询出来的结果需要额外排序，数据量小的在内存，大的话在磁盘排序，建议优化;
Using temporary 表示查询使用到了临时表，一般用于去重，分组等操作。

## 2.2  Trace工具
在MySQL执行计划中我们发现明明这个字段建立了索引，但是有的sql不会走索引，这是因为MySQL的内部优化器认为走索引的性能比不走索引全表扫描的性能要差，
一个典型的场景是走索引查出来的数据量很大，然后还需要根据这些行记录去主键索引树回表查出完整数据，此时优化器会觉得得不偿失，不如直接全表扫描。
而优化器的选择逻辑，依据来自于trace工具的结论。

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





![explain-three.png](images%2Fexplain-three.png)





由表结构可知，c_age，c_name, c_address的联合索引的长度，也就是key_len=1+1+30*4+2+1+100*4+2+1=2+123+403=528,
上图第一条sql使用到了联合索引u_key且使用的长度为528字节，说明使用索引充分，完全命中了联合索引u_key；但是第二条sql
就使用索引不充分了，从key_len=125字节推算，它只使用了c_age和c_name的部分索引，没有用到c_address列的索引，
key_len=1+1+3*40+2+1=2+123=125， 此时我们分析一下这条sql，只能局部命中索引的原因在于索引列使用了范围查询，且
范围查询的顺序相反，前两列都是小于等于的范围查询，最后一列是大于等于的范围查询，其查询顺序正好与前两列相反，因此只能
命中前两列的联合索引。


但如果是select * 返回全表字段等情况导致需要回表时，就只有联合索引第一列为等值查询时，后面的列使用范围查询才能命中索引，
否则，索引会失效。





![explain-four.png](images%2Fexplain-four.png)






如果是select * 返回全表字段等情况导致需要回表时，只要是联合索引第一列为范围查询，就会导致索引失效。





![explain-five.png](images%2Fexplain-five.png)





### 2.3.3 注意索引失效情况，根据具体场景做优化

#### 2.3.3.1 模糊查询时，%字符在查询条件开头会导致索引失效
>优化: 尽量不要%字符放在开头，如果确有此类后缀查询业务需求，可以考虑将数据同步到ES做查询。

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





#### 2.3.3.3 使用is null, is not null会导致全表扫描

```sql
explain select * from t_student where c_class_id = 6;
explain select * from t_student where c_class_id is not null;
```





![explain-seven.png](images%2Fexplain-seven.png)





> 优化: 必须把字段定义为NOT NULL, 并设置默认值。


#### 2.3.3.4 数据类型类型不一致会使得sql语句执行时进行隐式的类型转换最终导致索引失效





![explain-eight.png](images%2Fexplain-eight.png)





mysql默认会将字符串转化为数字，要验证这一点很简单，连接mysql后在客户端执行以下指令即可：





![verify-demo.png](images%2Fverify-demo.png)





输出1表示true, 意味着MySQL默认会将字符串"10"转化为数字10与数字9进行比较。所以，如果你对数据表中的一个字符串
类型的字段建立了索引，然后对这个索引列数据与int类型数字进行判等或大小范围操作时，MySQL就会将这个索引列数据都
转化为数字再进行比较，显然会导致索引失效。
相反，如果你对一个int类型的字段建立了索引，然后对这个索引列数据与字符串类型的数字(类似"10"这种)进行比较操作，
那么MySQL只需要将这个字符串类型的数字转化为int类型的数字和索引列数据比较就好了，查询扫描时仍然是走索引的。

譬如下面这个例子，只扫描两行数据，几乎过滤掉了100%的数据，是走了索引的。





![explain-nine.png](images%2Fexplain-nine.png)





