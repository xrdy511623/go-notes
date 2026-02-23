
---
面向LBS应用的GEO数据类型
---

我们前面学习了 Redis 的 5 大基本数据类型：String、List、Hash、Set 和 Sorted Set，它们可以满足大多数的数据存储需求，但是在面对海量数据统计时，它们的内存开销很大，而且对于一些特殊的场景，它们是无法支持的。所以，Redis 还提供了 3 种扩展数据类型，分别是 Bitmap、HyperLogLog 和 GEO。前两种在聚合、排序、二值状态和基数统计一节已重点介绍过了，今天，我们来看看 GEO。

另外，我们介绍开发自定义的新数据类型的基本步骤。掌握了自定义数据类型的开发方法，当面临一些复杂的场景时，就不用受基本数据类型的限制，可以直接在 Redis 中增加定制化的数据类型，来满足自定义的特殊需求。

接下来，我们就先来了解下扩展数据类型 GEO 的实现原理和使用方法。

# 1 面向 LBS 应用的 GEO 数据类型
在日常生活中，我们越来越依赖搜索“附近的餐馆”、在打车软件上叫车，这些都离不开基于位置信息服务（Location-Based Service，LBS）的应用。LBS 应用访问的数据是和人或物关联的一组经纬度信息，而且要能查询相邻的经纬度范围，GEO 就非常适合应用在 LBS 服务的场景中，我们来看一下它的底层结构。


## 1.1 GEO 的底层结构
一般来说，在设计一个数据类型的底层结构时，我们首先需要知道，要处理的数据有什么访问特点。所以，我们需要先搞清楚位置信息到底是怎么被存取的。

我以叫车服务为例，来分析下 LBS 应用中经纬度的存取特点。

每一辆网约车都有一个编号（例如 33），网约车需要将自己的经度信息（例如 116.034579）和纬度信息（例如 39.000452 ）发给叫车应用。
用户在叫车的时候，叫车应用会根据用户的经纬度位置（例如经度 116.054579，纬度 39.030452），查找用户的附近车辆，并进行匹配。
等把位置相近的用户和车辆匹配上以后，叫车应用就会根据车辆的编号，获取车辆的信息，并返回给用户。
可以看到，一辆车（或一个用户）对应一组经纬度，并且随着车（或用户）的位置移动，相应的经纬度也会变化。

这种数据记录模式属于一个 key（例如车 ID）对应一个 value（一组经纬度）。当有很多车辆信息要保存时，就需要有一个集合来保存一系列的 key 和 value。Hash 集合类型可以快速存取一系列的 key 和 value，正好可以用来记录一系列车辆 ID 和经纬度的对应关系，所以，我们可以把不同车辆的 ID 和它们对应的经纬度信息存在 Hash 集合中，如下图所示：





![car-locations.png](images%2Fcar-locations.png)





同时，Hash 类型的 HSET 操作命令，会根据 key 来设置相应的 value 值，所以，我们可以用它来快速地更新车辆变化的经纬度信息。

到这里，Hash 类型看起来是一个不错的选择。但问题是，对于一个 LBS 应用来说，除了记录经纬度信息，还需要根据用户的经纬度信息在车辆的 Hash 集合中进行范围查询。一旦涉及到范围查询，就意味着集合中的元素需要有序，但 Hash 类型的元素是无序的，显然不能满足我们的要求。

我们再来看看使用 Sorted Set 类型是不是合适。

Sorted Set 类型也支持一个 key 对应一个 value 的记录模式，其中，key 就是 Sorted Set 中的元素，而 value 则是元素的权重分数。更重要的是，Sorted Set 可以根据元素的权重分数排序，支持范围查询。这就能满足 LBS 服务中查找相邻位置的需求了。

实际上，GEO 类型的底层数据结构就是用 Sorted Set 来实现的。咱们还是借着叫车应用的例子来加深下理解。

用 Sorted Set 来保存车辆的经纬度信息时，Sorted Set 的元素是车辆 ID，元素的权重分数是经纬度信息，如下图所示：





![zset-car-locations.png](images%2Fzset-car-locations.png)





这时问题来了，Sorted Set 元素的权重分数是一个浮点数（float 类型），而一组经纬度包含的是经度和纬度两个值，是没法直接保存为一个浮点数的，那具体该怎么进行保存呢？

这就要用到 GEO 类型中的 GeoHash 编码了。


## 1.2 GeoHash 的编码方法

为了能高效地对经纬度进行比较，Redis 采用了业界广泛使用的 GeoHash 编码方法，这个方法的基本原理就是“二分区间，区间编码”。

当我们要对一组经纬度进行 GeoHash 编码时，我们要先对经度和纬度分别编码，然后再把经纬度各自的编码组合成一个最终编码。

首先，我们来看下经度和纬度的单独编码过程。

对于一个地理位置信息来说，它的经度范围是[-180,180]。GeoHash 编码会把一个经度值编码成一个 N 位的二进制值，我们来对经度范围[-180,180]做 N 次的二分区操作，其中 N 可以自定义。

在进行第一次二分区时，经度范围[-180,180]会被分成两个子区间：[-180,0) 和[0,180]（我称之为左、右分区）。此时，我们可以查看一下要编码的经度值落在了左分区还是右分区。如果是落在左分区，我们就用 0 表示；如果落在右分区，就用 1 表示。这样一来，每做完一次二分区，我们就可以得到 1 位编码值。

然后，我们再对经度值所属的分区再做一次二分区，同时再次查看经度值落在了二分区后的左分区还是右分区，按照刚才的规则再做 1 位编码。当做完 N 次的二分区后，经度值就可以用一个 N bit 的数来表示了。

举个例子，假设我们要编码的经度值是 116.37，我们用 5 位编码值（也就是 N=5，做 5 次分区）。

我们先做第一次二分区操作，把经度区间[-180,180]分成了左分区[-180,0) 和右分区[0,180]，此时，经度值 116.37 是属于右分区[0,180]，所以，我们用 1 表示第一次二分区后的编码值。

接下来，我们做第二次二分区：把经度值 116.37 所属的[0,180]区间，分成[0,90) 和[90, 180]。此时，经度值 116.37 还是属于右分区[90,180]，所以，第二次分区后的编码值仍然为 1。等到第三次对[90,180]进行二分区，经度值 116.37 落在了分区后的左分区[90, 135) 中，所以，第三次分区后的编码值就是 0。

按照这种方法，做完 5 次分区后，我们把经度值 116.37 定位在[112.5, 123.75]这个区间，并且得到了经度值的 5 位编码值，即 11010。这个编码过程如下图所示：





![latitude-encoding.png](images%2Flatitude-encoding.png)





对纬度的编码方式，和对经度的一样，只是纬度的范围是[-90，90]，下图显示了对纬度值 39.86 的编码过程。





![longitude-encoding.png](images%2Flongitude-encoding.png)





当一组经纬度值都编完码后，我们再把它们的各自编码值组合在一起，组合的规则是：最终编码值的偶数位上依次是经度的编码值，奇数位上依次是纬度的编码值，其中，偶数位从 0 开始，奇数位从 1 开始。

我们刚刚计算的经纬度（116.37，39.86）的各自编码值是 11010 和 10111，组合之后，第 0 位是经度的第 0 位 1，第 1 位是纬度的第 0 位 1，第 2 位是经度的第 1 位 1，第 3 位是纬度的第 1 位 0，以此类推，就能得到最终编码值 1110011101，如下图所示：






![mix-encoding.png](images%2Fmix-encoding.png)





用了 GeoHash 编码后，原来无法用一个权重分数表示的一组经纬度（116.37，39.86）就可以用 1110011101 这一个值来表示，就可以保存为 Sorted Set 的权重分数了。

当然，使用 GeoHash 编码后，我们相当于把整个地理空间划分成了一个个方格，每个方格对应了 GeoHash 中的一个分区。

举个例子。我们把经度区间[-180,180]做一次二分区，把纬度区间[-90,90]做一次二分区，就会得到 4 个分区。我们来看下它们的经度和纬度范围以及对应的 GeoHash 组合编码。

分区一：[-180,0) 和[-90,0)，编码 00；
分区二：[-180,0) 和[0,90]，编码 01；
分区三：[0,180]和[-90,0)，编码 10；
分区四：[0,180]和[0,90]，编码 11。
这 4 个分区对应了 4 个方格，每个方格覆盖了一定范围内的经纬度值，分区越多，每个方格能覆盖到的地理空间就越小，也就越精准。我们把所有方格的编码值映射到一维空间时，相邻方格的 GeoHash 编码值基本也是接近的，如下图所示：






![geohash-area-encoding.png](images%2Fgeohash-area-encoding.png)





所以，我们使用 Sorted Set 范围查询得到的相近编码值，在实际的地理空间上，也是相邻的方格，这就可以实现 LBS 应用“搜索附近的人或物”的功能了。

不过，需要注意的是，有的编码值虽然在大小上接近，但实际对应的方格却距离比较远。例如，我们用 4 位来做 GeoHash 编码，把经度区间[-180,180]和纬度区间[-90,90]各分成了 4 个分区，一共 16 个分区，对应了 16 个方格。编码值为 0111 和 1000 的两个方格就离得比较远，如下图所示：





![geo-neighbor-inaccurate.png](images%2Fgeo-neighbor-inaccurate.png)





所以，为了避免查询不准确问题，我们可以同时查询给定经纬度所在的方格周围的 4 个或 8 个方格。

好了，到这里，我们就知道了，GEO 类型是把经纬度所在的区间编码作为 Sorted Set 中元素的权重分数，把和经纬度相关的车辆 ID 作为 Sorted Set 中元素本身的值保存下来，这样相邻经纬度的查询就可以通过编码值的大小范围查询来实现了。接下来，我们再来聊聊具体如何操作 GEO 类型。


# 2 如何操作 GEO 类型？

Redis 在 3.2 版本中增加了 GEO 类型用于存储和查询地理位置，关于 GEO 的命令不多，主要包含以下 6 个：

geoadd：添加地理位置
geopos：查询位置信息
geodist：距离统计
georadius：查询某位置内的其他成员信息
geohash：查询位置的哈希值
zrem：删除地理位置
下面我们分别来看这些命令的使用。


## 2.1 添加地理位置
我们先用百度地图提供的经纬度查询工具，地址：

http://api.map.baidu.com/lbsapi/getpoint/index.html


找了以下 4 个地点，添加到 Redis 中：

天安门：116.404269,39.913164
月坛公园：116.36,39.922461
北京欢乐谷：116.499705,39.874635
香山公园：116.193275,39.996348

代码如下：
```shell
127.0.0.1:6379> geoadd site 116.404269 39.913164 tianan
(integer) 1
127.0.0.1:6379> geoadd site 116.36 39.922461 yuetan
(integer) 1
127.0.0.1:6379> geoadd site 116.499705 39.874635 huanle
(integer) 1
127.0.0.1:6379> geoadd site 116.193275 39.996348 xiangshan
(integer) 1
```

相关语法：

```shell
geoadd key longitude latitude member [longitude latitude member ...]
```

重点参数说明如下：
longitude 表示经度
latitude 表示纬度
member 是为此经纬度起的名字
此命令支持一次添加一个或多个位置信息。

## 2.2  查询位置信息

```shell
127.0.0.1:6379> geopos site tianan
1) 1) "116.40541702508926392"
2) "39.91316289865137179"
```

相关语法：

geopos key member [member ...]

此命令支持查看一个或多个位置信息。

## 2.3  距离统计

```shell
127.0.0.1:6379> geodist site tianan yuetan km
"3.9153"
```

注意：此命令统计的距离为两个位置的直线距离。

相关语法：
```shell
geodist key member1 member2 [unit]
```

unit 参数表示统计单位，它可以设置以下值：
m：以米为单位，默认单位；
km：以千米为单位；
mi：以英里为单位；
ft：以英尺为单位。


## 2.4  查询某位置内的其他成员信息

```shell
127.0.0.1:6379> georadius site 116.405419 39.913164 5 km
1) "tianan"
2) "yuetan"
```

此命令的意思是查询天安门（116.405419,39.913164）附近 5 公里范围内的成员列表。

相关语法：

```shell
georadius key longitude latitude radius m|km|ft|mi [WITHCOORD] [WITHDIST] [WITHHASH] [COUNT count] [ASC|DESC]
```

可选参数说明如下。

### 2.4.1 WITHCOORD

说明：返回满足条件位置的经纬度信息。

示例代码：

```shell
127.0.0.1:6379> georadius site 116.405419 39.913164 5 km withcoord
1) 1) "tianan"
2) 1) "116.40426903963088989"
2) "39.91316289865137179"
2) 1) "yuetan"
2) 1) "116.36000186204910278"
2) "39.92246025586381819"
```

### 2.4.2 WITHDIST

说明：返回满足条件位置与查询位置的直线距离。

示例代码：

```shell
127.0.0.1:6379> georadius site 116.405419 39.913164 5 km withdist
1) 1) "tianan"
2) "0.0981"
2) 1) "yuetan"
2) "4.0100"
```

### 2.4.3 WITHHASH

说明：返回满足条件位置的哈希信息。

示例代码：

```shell
127.0.0.1:6379> georadius site 116.405419 39.913164 5 km withhash
1) 1) "tianan"
2) (integer) 4069885552230465
2) 1) "yuetan"
2) (integer) 4069879797297521
```

### 2.4.4 COUNT count

说明：指定返回满足条件位置的个数。

例如，指定返回一条满足条件的信息，代码如下：

```shell
127.0.0.1:6379> georadius site 116.405419 39.913164 5 km count 1
1) "tianan"
```

### 2.4.5 ASC|DESC

说明：从近到远|从远到近排序返回。

示例代码：

```shell
127.0.0.1:6379> georadius site 116.405419 39.913164 5 km desc
1) "yuetan"
2) "tianan"

127.0.0.1:6379> georadius site 116.405419 39.913164 5 km asc
1) "tianan"
2) "yuetan"
```

当然以上这些可选参数也可以一起使用，例如以下代码：

```shell
127.0.0.1:6379> georadius site 116.405419 39.913164 5 km withdist desc
1) 1) "yuetan"
   2) "4.0100"
2) 1) "tianan"
   2) "0.0981"
```

## 2.5 查询哈希值

```shell
127.0.0.1:6379> geohash site tianan
1) "wx4g0cgp000"
```

相关语法：

```shell
geohash key member [member ...]
```

此命令支持查询一个或多个地址的哈希值。

## 2.6 删除地理位置

```shell
127.0.0.1:6379> zrem site tianan
(integer) 1
```

相关语法：

```shell
zrem key member [member ...]
```

此命令支持删除一个或多个位置信息。


## 2.8 GEOSEARCH（Redis 6.2+）

从 Redis 6.2 开始，`GEORADIUS` 和 `GEORADIUSBYMEMBER` 已被标记为**废弃**（deprecated），推荐使用 `GEOSEARCH` 命令替代。`GEOSEARCH` 不仅支持圆形范围查询，还新增了**矩形范围查询**。

```shell
GEOSEARCH key
  [FROMMEMBER member | FROMLONLAT longitude latitude]
  [BYRADIUS radius m|km|ft|mi | BYBOX width height m|km|ft|mi]
  [ASC | DESC]
  [COUNT count [ANY]]
  [WITHCOORD] [WITHDIST] [WITHHASH]
```

示例：

```shell
# 圆形范围查询（等价于 GEORADIUS）
127.0.0.1:6379> GEOSEARCH site FROMLONLAT 116.405419 39.913164 BYRADIUS 5 km ASC
1) "tianan"
2) "yuetan"

# 矩形范围查询（新能力）：查询以天安门为中心，10km × 8km 矩形内的成员
127.0.0.1:6379> GEOSEARCH site FROMLONLAT 116.405419 39.913164 BYBOX 10 8 km ASC
1) "tianan"
2) "yuetan"
3) "huanle"

# 以某个成员为中心查询
127.0.0.1:6379> GEOSEARCH site FROMMEMBER tianan BYRADIUS 5 km WITHDIST ASC
1) 1) "tianan"
   2) "0.0000"
2) 1) "yuetan"
   2) "3.9153"
```

对应的 `GEOSEARCHSTORE` 命令可以将查询结果存储到一个新的 Sorted Set 中：

```shell
# 将结果存入 nearby_sites 键
GEOSEARCHSTORE nearby_sites site FROMLONLAT 116.405419 39.913164 BYRADIUS 5 km ASC
```

> **迁移建议**：新代码应统一使用 `GEOSEARCH` / `GEOSEARCHSTORE`，避免使用已废弃的 `GEORADIUS` 系列命令。


## 2.9 GeoHash 编码精度

GeoHash 编码的位数（或字符数）直接决定了定位精度。位数越多，方格越小，精度越高：

| GeoHash 字符数 | bit 位数 | 方格宽度 | 方格高度 | 适用场景 |
|---------------|---------|---------|---------|---------|
| 1 | 5 | ~5000 km | ~5000 km | 洲际级别 |
| 2 | 10 | ~1250 km | ~625 km | 国家级别 |
| 3 | 15 | ~156 km | ~156 km | 大区域 |
| 4 | 20 | ~39 km | ~19.5 km | 城市级别 |
| 5 | 25 | ~4.9 km | ~4.9 km | 城区级别 |
| 6 | 30 | ~1.2 km | ~0.61 km | 街道级别 |
| 7 | 35 | ~153 m | ~153 m | 小区/建筑物级别 |
| 8 | 40 | ~38 m | ~19 m | 精确定位 |
| 9 | 45 | ~4.8 m | ~4.8 m | 门牌级别 |

Redis GEO 内部使用 **52 bit** 精度的 GeoHash（Sorted Set 的 double 类型 score 的有效精度），对应的定位精度约为 **0.6 米**，对绝大多数 LBS 应用场景已足够。

> `GEOHASH` 命令返回的是 Base32 编码的 11 字符字符串（如 `wx4g0cgp000`），可以在 http://geohash.org/ 上可视化查看对应位置。


可以看到，使用 GEO 数据类型可以非常轻松地操作经纬度这种信息。

虽然我们有了 5 种基本类型和 3 种扩展数据类型，但是有些场景下，我们对数据类型会有特殊需求，例如，我们需要一个数据类型既能像 Hash 那样支持快速的单键查询，又能像 Sorted Set 那样支持范围查询，此时，我们之前学习的这些数据类型就无法满足需求了。那么，接下来，我就再向你介绍下 Redis 扩展数据类型的终极版——自定义的数据类型。这样，你就可以定制符合自己需求的数据类型了，不管你的应用场景怎么变化，你都不用担心没有合适的数据类型。


# 3 如何自定义数据类型？

为了实现自定义数据类型，首先，我们需要了解 Redis 的基本对象结构 RedisObject，因为 Redis 键值对中的每一个值都是用 RedisObject 保存的。

我们知道，RedisObject 包括元数据和指针。其中，元数据的一个功能就是用来区分不同的数据类型，指针用来指向具体的数据类型的值。所以，要想开发新数据类型，我们就先来了解下 RedisObject 的元数据和指针。

## 3.1 Redis 的基本对象结构
RedisObject 的内部组成包括了 type、encoding、lru 和 refcount 4 个元数据，以及 1 个*ptr指针。

- type：表示值的类型，涵盖了我们前面学习的五大基本类型；
- encoding：是值的编码方式，用来表示 Redis 中实现各个基本类型的底层数据结构，例如 SDS、压缩列表、哈希表、跳表等；
- lru：记录了这个对象最后一次被访问的时间，用于淘汰过期的键值对；
- refcount：记录了对象的引用计数；
- *ptr：是指向数据的指针。





![redis-object.png](images%2Fredis-object.png)





RedisObject 结构借助*ptr指针，就可以指向不同的数据类型，例如，*ptr指向一个 SDS 或一个跳表，就表示键值对中的值是 String 类型或 Sorted Set 类型。所以，我们在定义了新的数据类型后，也只要在 RedisObject 中设置好新类型的 type 和 encoding，再用*ptr指向新类型的实现，就行了。

## 3.2 开发一个新的数据类型
了解了 RedisObject 结构后，定义一个新的数据类型也就不难了。首先，我们需要为新数据类型定义好它的底层结构、type 和 encoding 属性值，然后再实现新数据类型的创建、释放函数和基本命令。

接下来，我们以开发一个名字叫作 NewTypeObject 的新数据类型为例，来解释下具体的 4 个操作步骤。


### 3.2.1 第一步：定义新数据类型的底层结构

我们用 newtype.h 文件来保存这个新类型的定义，具体定义的代码如下所示：

```c
struct NewTypeObject {
    struct NewTypeNode *head;
    size_t len;
}NewTypeObject;
```

其中，NewTypeNode 结构就是我们自定义的新类型的底层结构。我们为底层结构设计两个成员变量：一个是 Long 类型的 value 值，用来保存实际数据；一个是*next指针，指向下一个 NewTypeNode 结构。

```c
struct NewTypeNode {
    long value;
    struct NewTypeNode *next;
};
```


从代码中可以看到，NewTypeObject 类型的底层结构其实就是一个 Long 类型的单向链表。当然，你还可以根据自己的需求，把 NewTypeObject 的底层结构定义为其他类型。例如，如果我们想要 NewTypeObject 的查询效率比链表高，就可以把它的底层结构设计成一颗 B+ 树。


### 3.2.2 第二步：在 RedisObject 的 type 属性中，增加这个新类型的定义

这个定义是在 Redis 的 server.h 文件中。比如，我们增加一个叫作 OBJ_NEWTYPE 的宏定义，用来在代码中指代 NewTypeObject 这个新类型。


```c
#define OBJ_STRING 0    /* String object. */
#define OBJ_LIST 1      /* List object. */
#define OBJ_SET 2       /* Set object. */
#define OBJ_ZSET 3      /* Sorted set object. */
…
#define OBJ_NEWTYPE 7
```

### 3.2.3 第三步：开发新类型的创建和释放函数

Redis 把数据类型的创建和释放函数都定义在了 object.c 文件中。所以，我们可以在这个文件中增加 NewTypeObject 的创建函数 createNewTypeObject，如下所示：

```c
robj *createNewTypeObject(void){
    NewTypeObject *h = newtypeNew();
    robj *o = createObject(OBJ_NEWTYPE,h);
    return o;
}
```


createNewTypeObject 分别调用了 newtypeNew 和 createObject 两个函数，下面分别来介绍下。

先说 newtypeNew 函数。它是用来为新数据类型初始化内存结构的。这个初始化过程主要是用 zmalloc 做底层结构分配空间，以便写入数据。

```c
NewTypeObject *newtypeNew(void){
    NewTypeObject *n = zmalloc(sizeof(*n));
    n->head = NULL;
    n->len = 0;
    return n;
}
```


newtypeNew 函数涉及到新数据类型的具体创建，而 Redis 默认会为每个数据类型定义一个单独文件，实现这个类型的创建和命令操作，例如，t_string.c 和 t_list.c 分别对应 String 和 List 类型。按照 Redis 的惯例，我们就把 newtypeNew 函数定义在名为 t_newtype.c 的文件中。

createObject 是 Redis 本身提供的 RedisObject 创建函数，它的参数是数据类型的 type 和指向数据类型实现的指针*ptr。

我们给 createObject 函数中传入了两个参数，分别是新类型的 type 值 OBJ_NEWTYPE，以及指向一个初始化过的 NewTypeObjec 的指针。这样一来，创建的 RedisObject 就能指向我们自定义的新数据类型了。

```c
robj *createObject(int type, void *ptr) {
    robj *o = zmalloc(sizeof(*o));
    o->type = type;
    o->ptr = ptr;
    ...
    return o;
}
```


对于释放函数来说，它是创建函数的反过程，是用 zfree 命令把新结构的内存空间释放掉。


### 3.2.4 第四步：开发新类型的命令操作

简单来说，增加相应的命令操作的过程可以分成三小步：

在 t_newtype.c 文件中增加命令操作的实现。比如说，我们定义 ntinsertCommand 函数，由它实现对 NewTypeObject 单向链表的插入操作：

```c
void ntinsertCommand(client *c){
//基于客户端传递的参数，实现在NewTypeObject链表头插入元素
}
```


在 server.h 文件中，声明我们已经实现的命令，以便在 server.c 文件引用这个命令，例如：

```c
void ntinsertCommand(client *c){
//基于客户端传递的参数，实现在NewTypeObject链表头插入元素
}
```


在 server.c 文件中的 redisCommandTable 里面，把新增命令和实现函数关联起来。例如，新增的 ntinsert 命令由 ntinsertCommand 函数实现，我们就可以用 ntinsert 命令给 NewTypeObject 数据类型插入元素了。

```c
struct redisCommand redisCommandTable[] = {
...
{"ntinsert",ntinsertCommand,2,"m",...}
}
```


此时，我们就完成了一个自定义的 NewTypeObject 数据类型，可以实现基本的命令操作了。当然，如果你还希望新的数据类型能被持久化保存，我们还需要在 Redis 的 RDB 和 AOF 模块中增加对新数据类型进行持久化保存的代码。

## 3.3 更现代的方式：Redis Module API

上面介绍的方法需要修改 Redis 源码并重新编译，维护成本高。从 **Redis 4.0** 开始，Redis 提供了 **Module API**，允许以动态加载模块（`.so` 文件）的方式扩展 Redis 功能，**无需修改 Redis 源码**。

Module API 支持：
- 定义新的数据类型（包含 RDB 持久化和 AOF 重写回调）
- 注册新的命令
- 访问 Redis 内部的键空间
- 订阅键空间通知

加载模块的方式：

```bash
# 在 redis.conf 中配置
loadmodule /path/to/mymodule.so

# 或在运行时动态加载
MODULE LOAD /path/to/mymodule.so
```

目前已有许多基于 Module API 开发的成熟模块，例如：

| 模块 | 功能 | 说明 |
|------|------|------|
| **RedisJSON** | JSON 数据类型 | 原生支持 JSON 路径查询和修改 |
| **RediSearch** | 全文搜索 | 支持二级索引、聚合、自动补全 |
| **RedisTimeSeries** | 时序数据 | 高效存储和查询时序数据 |
| **RedisBloom** | 概率数据结构 | 布隆过滤器、Cuckoo 过滤器、Top-K 等 |
| **RedisGraph** | 图数据库 | 基于 Cypher 查询语言的图引擎 |

> **建议**：对于新项目，优先考虑使用 Module API 来扩展 Redis 功能，而不是修改 Redis 源码。Module API 更易于维护和升级，且与 Redis 版本解耦。


# 4 GEO 使用注意事项

在实际应用中使用 Redis GEO 时，需要注意以下几个问题：

## 4.1 数据量与内存

GEO 底层是 Sorted Set，每个成员需要存储元素值和 score（8 字节的 double）。如果存储千万级 POI（Point of Interest），内存消耗和 `GEOSEARCH` 的查询性能都需要评估。

**优化建议**：按城市或区域分 key 存储，例如 `geo:beijing`、`geo:shanghai`，避免单 key 过大。

## 4.2 GeoHash 的边界问题

GeoHash 编码将二维空间映射到一维，虽然相邻方格的编码值大多接近，但**存在边界跳变**的情况（如前文提到的 0111 和 1000 编码值接近但方格距离远）。

Redis 内部已经通过**同时搜索目标方格及其周围 8 个相邻方格**来解决这个问题，使用 `GEOSEARCH` 命令时不需要额外处理。

## 4.3 距离计算的精度

`GEODIST` 使用的是 **Haversine 公式**，假设地球为标准球体。在以下情况下误差会增大：

- **极地区域**：纬度接近 ±90° 时，经度方向的距离被压缩，GeoHash 方格变形严重
- **超长距离**：跨越半个地球时，Haversine 公式的误差可达 0.5%

对于绝大多数 LBS 应用（搜索附近的人/店/车），这些误差完全可以忽略。

## 4.4 经度范围的特殊性

GeoHash 编码中，经度范围是 [-180, 180]。当位置横跨 **180° 经线**（国际日期变更线）时，两个地理上相邻的位置编码值会出现巨大差异。不过这种场景在实际业务中极少遇到。


# 5 小结

| 主题 | 核心要点 |
|------|---------|
| GEO 底层结构 | 基于 Sorted Set 实现，经纬度通过 GeoHash 编码为 52bit 整数作为 score |
| GeoHash 编码 | “二分区间 + 区间编码 + 经纬度交叉组合”，将二维坐标映射到一维可排序值 |
| 编码精度 | 52bit 精度约 0.6 米，GeoHash 字符数越多方格越小越精确 |
| 核心命令 | `GEOADD`（添加）、`GEOSEARCH`（范围查询，6.2+ 推荐）、`GEODIST`（距离）、`GEOHASH`（编码）|
| 命令废弃 | `GEORADIUS` / `GEORADIUSBYMEMBER` 在 Redis 6.2 后废弃，用 `GEOSEARCH` 替代 |
| 边界问题 | Redis 内部搜索目标方格 + 周围 8 个方格，已自动解决 GeoHash 边界跳变问题 |
| 扩展数据类型 | 两种途径：基于现有类型 + 编码（如 GEO、Bitmap）；Module API 动态加载模块（推荐） |

**GEO 类型常用命令速查**：

| 命令 | 作用 | 时间复杂度 |
|------|------|-----------|
| `GEOADD key lng lat member` | 添加/更新位置 | O(logN) |
| `GEOPOS key member` | 查询位置坐标 | O(1) |
| `GEODIST key m1 m2 [unit]` | 两点直线距离 | O(1) |
| `GEOHASH key member` | 查询 GeoHash 编码 | O(1) |
| `GEOSEARCH key FROMLONLAT/FROMMEMBER BYRADIUS/BYBOX` | 范围查询（圆形/矩形） | O(N+logM) |
| `GEOSEARCHSTORE dest key ...` | 范围查询并存储结果 | O(N+logM) |
| `ZREM key member` | 删除位置（复用 Sorted Set 命令） | O(logN) |