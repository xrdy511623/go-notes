---
SQL优化
---

# 1 工程优化

## 1.1 基础规范

表存储引擎必须使用InnoDB;
表字符集默认使用utf8, 必要时使用utf8mb4;
utf8是通用的，无乱码风险，汉字三个字节，英文1个字节;
utf8mb4是utf8的超集，需要存储4字节的数据如表情符号时使用它;

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

禁止大表 JOIN 和子查询;
同一个字段上的 OR 必须改写为IN，IN 的值必须少于 50 个。

应用程序必须捕获 SQL 异常, 方便定位线上问题。

# 2 SQL语句优化

## 2.1 Explain工具

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
Using where 表示需要回表查询；
Using index 表示使用到了覆盖索引，不需要回表；
Using filesort 表示查询出来的结果需要额外排序，数据量小的在内存，大的话在磁盘排序，建议优化;
Using temporary 表示查询使用到了临时表，一般用于去重，分组等操作。

## 2.2  Trace工具
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





![explain-three.png](images%2Fexplain-three.png)





由表结构可知，c_age，c_name, c_address的联合索引的长度，也就是key_len=1+1+30x4+2+1+100x4+2+1=2+123+403=528,
上图第一条sql使用到了联合索引u_key且使用的长度为528字节，说明使用索引充分，完全命中了联合索引u_key；但是第二条sql
就使用索引不充分了，从key_len=125字节推算，它只使用了c_age和c_name的部分索引，没有用到c_address列的索引，
key_len=1+1+3x40+2+1=2+123=125， 此时我们分析一下这条sql，只能局部命中索引的原因在于索引列使用了范围查询，且
范围查询的顺序相反，前两列都是小于等于的范围查询，最后一列是大于等于的范围查询，其查询顺序正好与前两列相反，因此只能
命中前两列的联合索引。


但如果是select * 返回全表字段等情况导致需要回表时，就只有联合索引第一列为等值查询时，后面的列使用范围查询才能命中索引，
否则，索引会失效。





![explain-four.png](images%2Fexplain-four.png)






如果是select * 返回全表字段等情况导致需要回表时，只要是联合索引第一列为范围查询，就会导致索引失效。





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





#### 2.3.3.5 少使用or或in，not in,  exist，MySQL内部优化器可能不走索引
当in后面的子查询语句结果集数据比较多时，可能会导致索引失效。
优化: 将in后面的子查询语句结果集进行拆分，比如有10万条数据，可以拆分为一次查询1000条，查询100次，运用多线程进行结果整合。
主键索引使用not in 关键字查询数据范围可以命中索引，而普通索引使用not in关键字查询数据范围则会导致索引失效；使用
not exists关键字会导致索引失效。
or前后的字段必须都有索引才会走索引，只要其中有一个字段没索引都会导致索引失效。


### 2.3.4 范围查询优化





![explain-ten.png](images%2Fexplain-ten.png)





![explain-eleven.png](images%2Fexplain-eleven.png)





优化: 将一个很大的查询范围拆分为多个小范围查询结果集之和，这样虽然需要多次查询才能得到最终结果，但是不会导致索引
失效，总体查询时间仍然比全表扫描要少很多。





### 2.3.5 多表连接查询优化
```sql
explain select * from t1 inner join t2 on t1.a = t2.a
```
t1是大表(一万条记录)，t2是小表(一百条记录)，在inner join查询中，如果关联字段建立了索引，且字段类型相同，MySQL
就会使用NLJ(Nested Loop Join)算法，去找小表(数据量比较小的表)作为驱动表，先从驱动表读取一行数据，然后拿着这
一行数据去被驱动表(数据量比较大的表)中做查询。这样的大表和小表是由MySQL内部优化器来决定的，跟sql语句中中表的书写顺序无关。

相反，如果关联字段没有建立索引，或者建立了索引但是字段的类型不同或者长度不同，那么会导致索引失效，此时MySQL会创建一个
join buffer内存缓冲区，把小表数据存进来(为什么不存大表，因为缓冲区大小限制，以及存数据消耗性能的考虑)，将内存缓冲区的
100行记录去和大表中的1万行记录进行比较，比较的过程依然是在内存中进行的。join buffer内存缓冲区起到了提高join查询效率的效果。
此时使用的是Block Nested Loop Join(BNLJ)算法。

还有一个很坑的地方，就是多表join连接查询时，连接字段的两个表的字符集也必须相同，否则索引也会失效，譬如A表字符集是utf8, 
join的表B字符集是utf8mb4，那么即使连接字段名，类型都完全相同，也都建立了索引，这个索引也会失效，也会走全表扫描。
为什么呢？
字符集utf8mb4是utf8的超集，所以当这两个类型的字符串做比较时，MySQL内部的操作是先将utf8字符串转化成utf8mb4字符集，
再做比较。这个转换过程，其实就是在被驱动表的索引字段上加函数操作，优化器会放弃走索引树。

关联字段没建立索引前是全表扫描ALL





![explain-twelve.png](images%2Fexplain-twelve.png)





关联字段建立索引后则是ref





![explain-thirteen.png](images%2Fexplain-thirteen.png)





### 2.3.6 小表驱动大表优化
>在使用多表连接查询时，尽量使用inner join，这样MySQL优化器会自动选择小表来驱动大表，如果使用left join 
或者 right join 要注意表的书写顺序，如果是左连接，则必须左表为小表，如果是右连接，则必须右表为小表，否则
会出现大表驱动小表，导致性能问题。

>in 和 exists优化，也要遵循小表驱动大表的原则以提高查询性能，in查询适合左表为大表，右表为小表的场景。

譬如:
```sql
select * from bigTable where id in (select id from smallTable)
```

相反，如果是左表为小表，右表为大表的场景，那就应该使用exists查询，譬如:
```sql
select * from smallTable where exists (select 1 from bigTable where bigTable.id = smallTable.id)
```

### 2.3.7 小表驱动大表优化
```sql
explain select * from employees limit 10000, 10
```

如果主键索引是连续的情况下可以这样优化:
```sql
explain select * from employees where id > 10000 limit 10;
```

如果主键索引不连续，则可以这样优化:
```sql
select * from employees a inner join (select id from employees limit 10000, 10) b on a.id = b.id
```





![explain-fourteen.png](images%2Fexplain-fourteen.png)





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
> 虽然order by 或者where + order by列顺序遵循最左匹配原则，但是使用了select * 导致需要回表会使联合索引失效。





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





> filesort文件排序的原理
在执行文件排序的时候，会把查询的数据量大小与系统变量: max_length_for_sort_data的大小进行比较
(默认是1024个字节), 如果比系统变量小，那么执行单路排序，否则执行双路排序。

单路排序
把所有的数据扔到sort_buffer内存缓冲区中，进行排序；
双路排序
取数据的排序字段和主键字段，扔到内存缓冲区，排序完成后，根据主键字段做一次回表查询，获取完整数据。

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
show global variables like 'inndodb_buffer_pool_size';
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
> 如果对group by语句的结果没有排序要求，需要在语句后加上order by null，否则MySQL会在分组后进行排序;
> 尽量让group by 过程用上表的索引(对分组字段建立索引)，确认方法是explain的extra里没有出现Using temporary和Using filesort。
> 如果group by 需要统计的数据量不大，尽量使用内存临时表sort buffer；也可以通过适当调大tmp_table_size参数，来避免用到磁盘临时表。
> 如果数据量是在太大，使用SQL_BIG_RESULT这个提示，来告诉优化器直接使用排序算法得到group by的结果。





![explain-twenty.png](images%2Fexplain-twenty.png)





![explain-twentyone.png](images%2Fexplain-twentyone.png)