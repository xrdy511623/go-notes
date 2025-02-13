
---
SQL实战
---

作为后端开发人员，熟练掌握SQL语句是必须的，在面试时需要手撕代码的地方主要有两个场景，一是做算法题，另一个就是面试官提出需求要
你写出对应的SQL语句，今天我们来看看常见的SQL查询需求应该如何实现。


# case1: 查询employee表中第n高的薪水是多少？

通用的思路是：

```sql
select ifNull((select distinct salary from employee order by salary desc limit n-1, 1), null) 
as n_highest;
```

具体实现是这样的，譬如n=5
```sql
select ifNull((select distinct salary from employee order by salary desc limit 4, 1), null) as
fifth_highest;
```

# case2: 编写一个SQL查询来实现分数排名。

如果两个分数相同，则两个分数排名（Rank）相同。请注意，平分后的下一个名次应该是下一个连续的整数值。换句话说，
名次之间不应该有“间隔”。


| Id | Score |
|----|-------|
| 1  | 3.50  |
| 2  | 3.65  |
| 3  | 4.00  |
| 4  | 3.85  |
| 5  | 4.00  |
| 6  | 3.65  |


根据上述给定的Scores表，你的查询应该返回（按分数从高到低排列）:

| Score | Rank |
|-------|------|
| 4.00  | 1    |
| 4.00  | 1    |
| 3.85  | 2    |
| 3.65  | 3    |
| 3.65  | 3    |
| 3.50  | 4    |


第一个思路：第一部分是降序排列的分数,第二部分是每个分数对应的排名
比较难的是第二部分。假设现在给你一个分数X，如何算出它的排名Rank呢？
我们可以先提取出大于等于X的所有分数集合H，将H去重后的元素个数就是X的排名。比如你考了99分，但最高的就只有99分，那么去重
之后集合H里就只有99一个元素，个数为1，因此你的Rank为1。
先提取集合H：

```sql
select Score from Scores where Score >= X;
```

再去重得到排名Rank:

```sql
select count(distinct Score) from Scores where Score >= X;
```

所以连起来就是:

```sql
select a.Score as Score,
(select count(distinct b.Score) from Scores b where b.Score >= a.Score) as `Rank`
from Scores a order by a.Score desc;
```

```sql
select score, DENSE_RANK() over(order by score desc) as `Rank` from ranks;
```

第二个思路：不讲码德，直接使用窗口函数
现在给定五个成绩：99，99，85，80，75。
DENSE_RANK()，如果使用 DENSE_RANK() 进行排名会得到：1，1，2，3，4；
RANK()，如果使用 RANK() 进行排名会得到：1，1，3，4，5；
ROW_NUMBER()，如果使用 ROW_NUMBER() 进行排名会得到：1，2，3，4，5；
所以sql语句可以这样写:

```sql
select Score, DENSE_RANK() over(order by Score desc) as `Rank` from Scores;
```

# case3：部门工资最高的员工

Employee表包含所有员工信息，每个员工有其对应的Id, salary 和 department Id。

| Id | Name  | Salary | DepartmentId |
|----|-------|--------|--------------|
| 1  | Joe   | 70000  | 1            |
| 2  | Jim   | 90000  | 1            |
| 3  | Henry | 80000  | 2            |
| 4  | Sam   | 60000  | 2            |
| 5  | Max   | 90000  | 1            |

Department表包含公司所有部门的信息。

| Id | Name  |
|----|-------|
| 1  | IT    |
| 2  | Sales |


编写一个 SQL 查询，找出每个部门工资最高的员工。对于上述表，您的SQL查询应返回以下行（行的顺序无关紧要）。

| Department | Employee | Salary |
|------------|----------|--------|
| IT         | Max      | 90000  |
| IT         | Jim      | 90000  |
| Sales      | Henry    | 80000  |


思路:我们可以在Employee表中按照部门分组查询最高工资, 然后根据Employee表DepartmentId与Department表ID做
多表连接Join查询,通过where做条件筛选，筛选条件为部门ID与薪资匹配照部门分组查询最高工资的查询结果。

```sql
select d.name as `Department`, e.Name as `Employee`,
e.Salary as `Salary` from Employee e join Department d on e.DepartmentId = d.Id 
where (e.DepartmentId, e.Salary) in
(select DepartmentId, max(Salary) from Employee group by DepartmentId );
```

# case4:超过经理收入的员工

Employee表包含所有员工，他们的经理也属于员工。每个员工都有一个Id，此外还有一列对应员工的经理的Id。

| Id | Name  | Salary | ManagerId |
|----|-------|--------|-----------|
| 1  | Joe   | 70000  | 3         |
| 2  | Henry | 80000  | 4         |
| 3  | Sam   | 60000  | NULL      |
| 4  | Max   | 90000  | NULL      |


给定以上的Employee表，编写一个SQL查询，该查询可以获取收入超过他们经理的员工的姓名。在上面的表格中，Joe是唯一一个收入超过
他的经理的员工。


思路：自连接查询解决
```sql
select a.Name as `Employee` from Employee a inner join Employee b 
on a.ManagerId = b.Id where a.Salary > b.Salary;
```

下面这样更直观些
```sql
select a.Name as `Employee`, a.Salary as `employee_salary`, a.ManagerId as `manager_id`, b.Name as `manager`, 
b.Salary as `manager_salary` from Employee a inner join Employee b on a.ManagerId = b.Id where a.Salary > b.Salary;
```

# case5: 部门工资前三的所有员工

编写一个SQL 查询，找出每个部门获得前三高工资的所有员工。

| Id | Name  | Salary | DepartmentId |
|----|-------|--------|--------------|
| 1  | Joe   | 85000  | 1            |
| 2  | Henry | 80000  | 2            |
| 3  | Sam   | 60000  | 2            |
| 4  | Max   | 90000  | 1            |
| 5  | Janet | 69000  | 1            |
| 6  | Randy | 85000  | 1            |
| 7  | Will  | 70000  | 1            |


Department表包含公司所有部门的信息。

| Id | Name  |
|----|-------|
| 1  | IT    |
| 2  | Sales |


例如，根据上述给定的表，查询结果应返回：

| Department | Employee | Salary |
|------------|----------|--------|
| IT         | Max      | 90000  |
| IT         | Randy    | 85000  |
| IT         | Joe      | 85000  |
| IT         | Will     | 70000  |
| Sales      | Henry    | 80000  |
| Sales      | Sam      | 60000  |


思路一：我们先找出公司里前3高的薪水，意思是不超过三个值比这些值大

```sql
SELECT e1.Salary
FROM Employee AS e1
WHERE 3 >
      (SELECT count(DISTINCT e2.Salary)
       FROM  Employee AS e2
       WHERE e1.Salary < e2.Salary AND e1.DepartmentId = e2.DepartmentId) ;
```

结合前3高的薪水，再把表Department 和表 Employee 连接，获得各个部门工资前三高的员工。

```sql
select d.Name as `Department`, e1.Name as `Employee`, e1.Salary as `Salary`
from Department d inner join Employee e1 on d.Id = e1.DepartmentId where 3 > (
select count(distinct e2.Salary) from Employee e2 where e2.Salary > e1.Salary
and e1.DepartmentId = e2.DepartmentId)
```


思路二:利用窗口函数可以快速实现既分组又排序的需求。
1）考察如何使用窗口函数及专用窗口函数排名的区别：rank, dense_rank, row_number
2）经典topN问题：每组最大的N条记录。这类问题涉及到“既要分组，又要排序”的情况，要能想到用窗口函数来实现。


```sql
select d.Name as Department, e.Name as Employee, e.Salary from (
select *, dense_rank() over(partition by DepartmentId order by Salary desc) 
as r from Employee)e 
inner join Department d on e.DepartmentId = d.Id and e.r <= 3;
```

# case6: 寻找重复的电子邮箱

编写一个SQL查询，查找Person表中所有重复的电子邮箱。

| Id | Email        |
|----|--------------|
| 1  | zs@gmail.com |
| 2  | ls@163.com   |
| 3  | zs@gmail.com |

根据以上输入，你的查询应返回以下结果：

| Email        |
|--------------|
| zs@gmail.com |


思路一: 利用临时表

```sql
select Email from (select Email, count(*) as count from Person group by Email) t 
where t.count > 1;
```

思路二：利用分组加过滤，不需要引入临时表，性能更优

```sql
select Email from Person group by Email having count(Email) > 1;
```

# case7: 从不订购的客户

两个表，Customers表和Orders表。编写一个SQL查询，找出所有从不订购任何东西的客户。
Customers表：

| Id | Name  |
|----|-------|
| 1  | Joe   |
| 2  | Henry |
| 3  | Sam   |
| 4  | Max   |


Orders表：

| Id | CustomerId |
|----|------------|
| 1  | 3          |
| 2  | 1          |


根据给定上述数据，你的查询SQL应该返回以下结果:

| Customer |
|----------|
| Henry    |
| Max      |


思路一: 利用in子查询过滤解决

```sql
select Name as `customer` from Customers where Id not in 
(select distinct CustomerId from Orders);
```

思路二: 使用左连接left join解决

```sql
select a.Name as `customers` from Customers a left join Orders b 
on a.Id = b.CustomerId where b.CustomerId is null;
```

# case8: 删除重复的电子邮箱

编写一个 SQL 查询，来删除Person表中所有重复的电子邮箱，重复的邮箱里只保留Id最小的那个。

| Id | Email        |
|----|--------------|
| 1  | zs@gmail.com |
| 2  | ls@163.com   |
| 3  | zs@gmail.com |


在运行你的sql语句之后，上面的 Person 表应返回以下几行:

| Id | Email        |
|----|--------------|
| 1  | zs@gmail.com |
| 2  | ls@163.com   |

思路: 自连接查询解决

```sql
delete p1 from Person p1 join Person p2 where p1.Email = p2.Email and p1.Id > p2.Id;
```

# case9: 连续出现的数字

编写一个 SQL 查询，查找所有至少连续出现三次的数字。

| Id | Num |
|----|-----|
| 1  | 1   |
| 2  | 1   |
| 3  | 1   |
| 4  | 2   |
| 5  | 1   |
| 6  | 2   |
| 7  | 2   |


在运行你的sql语句之后，应返回:

| ConsecutiveNums |
|-----------------|
| 1               |


思路: 多表自连接查询解决

```sql
select distinct l1.Num as ConsecutiveNums from logs l1 join logs l2 
join logs l3 where l1.Id = l2.Id -1 and l2.Id = l3.Id - 1 
and l1.Num = l2.Num and l2.Num = l3.Num;
```

# case10: 查询两门及以上课程不及格的同学的学号和姓名

studentScore表

| studentNo | courseNo | score |
|-----------|----------|-------|
| 1         | 1        | 56    |
| 1         | 2        | 90    |
| 1         | 3        | 89    |
| 2         | 2        | 50    |
| 2         | 3        | 47    |
| 3         | 1        | 80    |
| 3         | 2        | 70    |
| 3         | 3        | 65    |


student表

| studentNo | name    | birthday   | gender |
|-----------|---------|------------|--------|
| 1         | zangsan | 1989-01-01 | male   |
| 2         | lisi    | 1990-12-21 | female |
| 3         | wangwu  | 1991-07-16 | male   |
| 4         | zhaoliu | 1990-05-20 | female |


course表

| courseNo | courseNo | teacherNo |
|----------|----------|-----------|
| 1        | 语文       | 2         |
| 2        | 数学       | 1         |
| 3        | 英语       | 3         |


teacher表

| teacherNo | teacherName |
|-----------|-------------|
| 1         | qianqi      |
| 2         | sunba       |
| 3         | zoujiu      |



思路: 先where过滤出不及格的数据而后根据课程号进行聚合分组，最后对聚合的结果进行条件过滤。

```sql
select studentNo, name from student where studentNo in 
(select studentNo from studentScore where score < 60 
group by studentNo having count(courseNo) >=2);
```

如果还需要获取这些同学不及格科目的成绩的话，就需要联表查询了
```sql
select s.studentNo, s.name, ss.courseNo, ss.score from student s inner join studentScore ss 
on s.studentNo = ss.studentNo where s.studentNo in 
(select studentNo from studentScore where score < 60 
group by studentNo having count(courseNo) >=2);
```

# case11: 查询所有课程成绩都小于90分的学生的学号, 姓名

思路：临时表, 多表连接查询和子查询解决

```sql
select s.studentNo, s.name from student s inner join 
(select studentNo, count(courseNo) as cnt from studentScore group
by studentNo) t on s.studentNo = t.studentNo where s.studentNo in 
(select studentNo from studentScore where score < 90 group by studentNo 
having count(courseNo) = t.cnt);
```

# case12: 查询没有学全所有课的学生的学号、姓名

思路：第一步，统计所有学生所修课程的数量count，筛选出count小于课程总数的记录；
第二步，在学生表中通过范围查询统计出学号在第一步结果的学号范围的学生。

```sql
select s.studentNo, s.name from student s inner join (select studentNo, 
count(courseNo) as cnt from studentScore group by studentNo) t 
on s.studentNo = t.studentNo where t.cnt < (select count(distinct courseNo) 
from studentScore);
```

# case13: 日期函数的使用

a 1990年出生的学生

```sql
select name, birthday from student where year(birthday) = 1990;
```

b 计算学生的年龄

```sql
select studentNo, name, timestampdiff(year, birthday, now()) as age from student;
```

c 查询本月过生日的学生

```sql
select name, birthday from student where month(birthday) = month(now());
```

# case14: 多表连接查询练习

a 查询所有学生的学号、姓名、选课数、总成绩

```sql
select s1.studentNo, s1.name, s2.count, s2.sumScore from student s1 
inner join (select studentNo, count(courseNo) as count, sum(score) as sumScore
from studentScore group by studentNo) s2 on s1.studentNo = s2.studentNo;
```

b 查询平均成绩大于70的所有学生的学号、姓名和平均成绩

```sql
select s.studentNo, s.name, t.avgScore from student s inner join 
(select studentNo, avg(score) as avgScore from studentScore 
group by studentNo) t 
on s.studentNo = t.studentNo where t.avg_score > 70;
```

c 查询所有学生的选课情况：学号，姓名，课程号，课程名称
由于学生表与课程表没有直接的关系，所以需要借助成绩表作为中介进行三表连接.

```sql
select a.studentNo, a.name, c.courseNo, c.courseName from student
a inner join studentScore b on a.studentNo = b.studentNo
inner join course c on b.courseNo = c.courseNo;
```

d 查询课程编号为3且课程成绩在80分以上的学生的学号和姓名

```sql
elect s.studentNo, s.name, t.courseNo, t.score from student s inner join 
(select studentNo, courseNo, score from studentScore where courseNo
 = 3 and score > 80) t on s.studentNo = t.studentNo;
```

e 查询课程编号为2且课程成绩小于90，按分数降序排序的学生信息，课程号及其对应的成绩信息

```sql
select a.*, b.courseNo, b.score from student a inner join 
(select studentNo, courseNo, score from studentScore 
where courseNo = 2 and score < 90) b 
on a.studentNo = b.studentNo order by b.score desc;
```

f 查询课程名称为"数学", 且分数低于90的学生姓名和分数

```sql
select a.name, c.courseName, b.score from student a inner join studentScore b 
on a.studentNo = b.studentNo inner join course c on b.courseNo = c.courseNo 
where c.courseName = "数学" and b.score < 90;
```

g 查询至少有一门课程成绩在70分以上的学生姓名、课程名称和分数

```sql
select a.name, c.courseName, b.score from student a inner join studentScore b 
on a.studentNo = b.studentNo inner join course c on b.courseNo = c.courseNo 
where b.score > 70;
```

h 查询有两门及其以上课程不及格的学生的学号，姓名及其平均成绩
加了一个平均成绩，就需要多一次联表查询
比较:查询有两门及其以上课程不及格的学生的学号，姓名

```sql
select studentNo, name from student where studentNo in 
(select studentNo from studentScore where score < 60 
group by studentNo having count(courseNo) >=2);
```

平均成绩需要统计学生成绩表中所有课程的平均成绩，如果在上面SQL的聚合子查询中加上平均成绩的聚合查询，
得出的是不及格课程的平均成绩，与题意不符。

```sql
select s.studentNo, s.name, c.avgScore from student s inner join
(select studentNo from studentScore where score < 60
group by studentNo having count(courseNo) >= 2) b on s.studentNo = b.studentNo
inner join (select studentNo, avg(score) as avgScore from studentScore group by 
studentNo) c on b.studentNo = c.studentNo;
```

i 查询至少有两门课程成绩在80分及以上的所有学生的学号,姓名和平均成绩
思路与上一题相同，坑也一样

```sql
select s.studentNo, s.name, c.avgScore from student s inner join 
(select studentNo from studentScore where score >= 80
group by studentNo having count(courseNo) >= 2) b on s.studentNo = b.studentNo
inner join (select studentNo, avg(score) as avgScore from studentScore 
group by studentNo) c on b.studentNo = c.studentNo;
```

j 查询任何一门课程成绩均在70分以上的学生姓名、课程名称和分数
思路: 通过where条件过滤出课程成绩在70分以上的数据，再通过学号分组，筛选出课程数等于该学生所学所有课程数的数据。

```sql
select a.studentNo, a.name, c.courseName, b.score from student a inner join 
studentScore b on a.studentNo = b.studentNo inner join course c on b.courseNo = c.courseNo 
inner join (select studentNo, count(courseNo) as cnt from studentScore
group by studentNo) t on b.studentNo = t.studentNo where a.studentNo in 
(select studentNo from studentScore where score > 70 group by studentNo 
having count(courseNo) = t.cnt);
```

k 查询学过编号为1的课程并且也学过编号为2的课程的学生学号,姓名

```sql
select studentNo, name from student where studentNo in 
(select a.studentNo from (select studentNo from studentScore
where courseNo = 1) a inner join (select studentNo from studentScore 
where courseNo = 2) b on a.studentNo = b.studentNo);
```

l 进阶 查询学过编号为1的课程并且也学过编号为2的课程的学生的学号, 姓名，以及两门课程的成绩

```sql
select s.studentNo, s.name, t.a1, t.a2, t.b1, t.b2 from student s 
inner join (select a.studentNo, a.courseNo as a1, a.score as a2, 
b.courseNo as b1, b.score as b2 from (select * from score where courseNo = 1) 
a inner join (select * from score where courseNo = 2) b 
on a.studentNo = b.studentNo) t on s.studentNo = t.studentNo;
```

m 查询所有老师所教课程的信息及课程的平均分信息，按平均分从高到低降序排列

```sql
select c.courseName, t.teacherName, s.avgScore from course c inner join
(select courseNo, avg(score) as avgScore from
studentScore group by courseNo) s on c.courseNo = s.courseNo
inner join teacher t on c.teacherNo = t.teacherNo order by s.avgScore desc;
```

n 查询课程不同但成绩相同的学生的学号, 姓名, 课程编号, 成绩信息

```sql
select s.studentNo, s.name, t.courseNo, t.score from student s inner join
(select distinct a.studentNo, a.courseNo, a.score from studentScore a 
inner join studentScore b on a.studentNo = b.studentNo 
where a.courseNo != b.courseNo and a.score = b.score) t 
on s.studentNo = t.studentNo;
```

o 扩展:查询同一门课程成绩相同学生的学生学号, 姓名, 课程编号, 成绩信息

```sql
select s.studentNo, s.name, t.courseNo, t.score from student s inner join
(select distinct a.studentNo, a.courseNo, a.score from studentScore a 
inner join studentScore b on a.courseNo = b.courseNo where a.studentNo != b.studentNo
and a.score = b.score) t on s.studentNo = t.studentNo;
```


p 查询课程编号为1的课程成绩比课程编号为2的课程成绩低的所有学生的学号

```sql
select a.studentNo, a.courseNo, a.score, b.courseNo, b.score from
(select * from studentScore where courseNo = 1) a inner join 
(select * from studentScore where courseNo = 2) b on a.studentNo = b.studentNo 
where a.score < b.score;
```

q 查询学过zoujiu老师所教的课程的所有同学的学号、姓名，课程名称和成绩。

```sql
select a.studentNo, a.name, c.courseName, b.score, d.teacherName from student 
a inner join studentScore b on a.studentNo = b.studentNo inner join course c 
on b.courseNo = c.courseNo inner join teacher d on c.teacherNo = d.teacherNo
where d.teacherName = "zoujiu";
```

r 进阶一: 查询选修zoujiu老师所授课程学生中成绩最高的学生姓名及其成绩

```sql
select a.studentNo, a.name, c.courseName, b.score, d.teacherName from student 
a inner join studentScore b on a.studentNo = b.studentNo inner join course c 
on b.courseNo = c.courseNo inner join teacher d on c.teacherNo = d.teacherNo
where d.teacherName = "zoujiu" order by b.score desc limit 1;
```

s 进阶二: 查询没学过zoujiu老师讲授课程的所有学生姓名

```sql
select name from student where studentNo not in 
(select studentNo from studentScore s inner join course c on s.courseNo
= c.courseNo inner join teacher t on c.teacherNo = t.teacherNo 
where t.teacherName = "zoujiu");
```

t 查询至少有一门课与学号为1的学生所学课程相同的学生的学号和姓名

```sql
select studentNo, name from student where studentNo in (select distinct 
studentNo from studentScore where courseNo in ( select courseNo from studentScore
where studentNo = 1)) and studentNo != 1;
```

# case 15 分段统计

a 统计出每门课程的及格人数和不及格人数

```sql
select courseNo, sum(case when score >= 60 then 1 else 0 end) as qualified, 
sum(case when score < 60 then 1 else 0 end) as unqualified from studentScore
group by courseNo;
```

b 进阶:按平均成绩从高到低显示所有学生的所有课程的成绩以及平均成绩以及名次

```sql
select s.studentNo, 
max(case when c.courseName  = "语文" then s.score else 0 end) as "语文",
max(case when c.courseName  = "数学" then s.score else 0 end) as "数学",
max(case when c.courseName  = "英语" then s.score else 0 end) as "英语", 
avg(s.score), row_number() over (order by avg(s.score) desc) as `rank`
from studentScore s inner join course c on s.courseNo = c.courseNo 
group by s.studentNo;
```


# case16: （窗口函数强化）查询所有课程成绩第2名到第3名的学生及该课程成绩

```sql
select b.name, a.courseNo, a.score from (select courseNo, studentNo, score,
row_number () over(partition by courseNo order by score desc) as ranking
from studentScore) a inner join student b on a.studentNo = b.studentNo
where a.ranking in( 2,3);
```

# case17: 查询北京和上海工作岗位数量及其占比

```sql
select bj_job_count, concat(round((bj_job_count/job_count)*100, 2),'%') 
as bj_job_rate, sh_job_count, concat(round((sh_job_count/job_count)*100, 2),'%') 
as sh_job_rate from (select count(distinct url_object_id) as bj_job_count 
from lagou_job where location like "北京%") a join 
(select count(distinct url_object_id) as job_count from lagou_job) b 
join (select count(distinct url_object_id) as sh_job_count from lagou_job 
where location like "上海%") c;
```

# case18: 查询各城市工作岗位数量及其占比，按岗位数降序排序，取前10名

```sql
select work_city, number, concat(round(number/total*100.00, 2), '%') as rate 
from (select * from (select work_city, count(url_object_id) as number from 
lagou_job group by work_city) t1 inner join (select count(url_object_id) as 
total from lagou_job) t2 on 1 = 1) t order by number desc limit 10;
```

# case19: 查询所有商家的所有品牌的销量信息，按商家升序排序，按销量降序排序

```sql
select storeId, brandId, count(orderId) as cnt from sales group by storeId, 
brandId order by storeId, cnt desc;
```

# case20: 两数据列运算产生新数据列，并且要按照这个新数据列排序

a, b都是数据列，现在要产生一个指标数据列，值为a/b，同时要处理除0错误，若b为0，显示0

```sql
select a, b, case when b = 0 then 0 else a / b end as ratio from table order by ratio desc;
```