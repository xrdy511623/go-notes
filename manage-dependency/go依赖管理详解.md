
---
go依赖管理详解
---

1 背景(为什么需要依赖管理？)
工程项目不可能基于标准库0~1编码搭建
管理依赖库

2 Go依赖管理严禁
GOPATH -> Go Vendor -> Go Module

2.1 > GOPATH

```shell
cd go && ls
bin pkg src
```

bin 是项目编译的二进制文件
pkg 是项目编译的中间产物,用于加速编译
src 项目源码

项目代码直接依赖src下的代码
go get会下载最新版本的包到src目录下

存在的问题：
在类似下面的场景中(A和B依赖于某一package的不同版本)，无法实现包(package)的多版本控制。
因为src目录下只能有一个版本存在，那A和B两个项目只能有一个编译通过，显然无法满足我们的需求。

![GOPATH.png](images%2FGOPATH.png)

2.2 > Go Vendor
在项目目录下增加vendor文件，所有依赖包以副本形式放在$ProjectRoot/vendor
依赖寻址方式: vendor -> GOPATH

![Go-Vendor.png](images%2FGo-Vendor.png)

如此，通过每个项目引入一份依赖的副本，解决了多个项目需要同一个package依赖的冲突问题。

存在的问题：

![Vendor-problem.png](images%2FVendor-problem.png)

在复杂的依赖关系下，无法控制依赖的版本。譬如项目A依赖package B和C，而B和C又分别依赖了D的不同版本v1和v2，在vendor的
管理模式下我们不能很好的控制对于D的依赖版本，一旦更新项目有可能出现依赖冲突，导致编译出错，归根结底，还是因为vendor
不能很清晰的标识依赖的版本概念。

2.3 > Go Module
终极目标: 定义版本规则和管理项目依赖关系

通过go.mod文件管理依赖包版本
通过go get/go mod指令工具管理依赖包

依赖管理三要素:
使用go.mod文件来配置文件，描述依赖；
使用Proxy中心仓库来管理依赖库；
使用go get/mod本地工具管理依赖包

![go-mod.png](images%2Fgo-mod.png)

module mxshop-api   依赖管理的基本单元(项目)

go 1.19            原生库(go版本)

require (         单元依赖
    github.com/alibaba/sentinel-golang v1.0.2
    github.com/aliyun/alibaba-cloud-sdk-go v1.61.1140
    github.com/dgrijalva/jwt-go v3.2.0+incompatible
    github.com/fsnotify/fsnotify v1.4.9
    github.com/gin-gonic/gin v1.9.1
)

依赖标识: Module Path Version {Major}.{Minor}.{Patch}
以github.com/gin-gonic/gin 为例
github.com/gin-gonic/gin是gin包的模块路径，从这个路径可以看出从哪里找到该模块，譬如如果是github前缀则表示可以从Github仓库
找到该模块，依赖包的源代码由github托管，如果项目的子包想被单独引用，则需要通过单独的go mod init生成的mod文件进行管理。
v1.4.9是语义化版本号，1是大版本(Major)，9是小版本(Minor)，最后一个1是
patch(修复)版本号
不同的Major版本表示是不兼容的API，所以即使是同一个库(包)，Major版本不同也会被认为是不同的模块；Minor版本通常是新增函数
或功能，向后兼容；而patch版本一般是修复bug。

依赖配置-version
语义化版本
${Major}.${Minor}.${Patch}
V1.3.0
V1.2.1

基于commit伪版本
v1.0.1-20231109134442-10cbfed86s6y

这种基础版本前缀和语义化版本是一样的;后面的20231109134442是时间戳，是Commit提交的时间，最后的10cbfed86s6y是校验码，
包含了12位的哈希前缀，每次提交commit后Go都会默认生成一个伪版本号。

github.com/go-errors/errors v1.0.1 // indirect
indirect表示go.mod对应的当前模块，没有直接导入该依赖模块的包，也就是不是直接依赖，是间接依赖。
A->B->C
那么在这个依赖链条里，A对B就是直接依赖，A对C就是间接依赖。

github.com/uber/jaeger-client-go v2.29.1+incompatible

主版本2+模块会在模块路径增加/vN后缀，这能让go module按照不同的模块来处理同一个项目不同主版本的依赖。由于Go Module是
Go 1.11实验性引入的，所以这项规则提出之前已经有一些仓库打上了2或者更高版本的tag了，为了兼容这部分仓库，对于没有go.mod
文件并且主版本在2或者以上的依赖，会在版本号后加上+incompatible后缀。

如果X项目依赖了A、B两个项目，且A、B分别依赖了C项目的v1.3、V1.4两个版本，最终编译时所使用的C项目的版本是?
A v1.3
B v1.4
C A用到C时用v1.3编译，B用到C时用v1.4编译

答案是B，选择最低的兼容版本。因为Minor是向后兼容的，所以用1.4一定是在1.3的基础上新增函数或功能，使用1.3那么B就无法
使用1.4新增的功能，无法通过编译，而使用1.4则不影响1.3原有功能的使用，所以会使用v1.4版本。

接下来讲一下go module的以来分发，也就是从哪里下载，如何下载的问题。
github是比较常见的代码托管平台，而Go Module系统中定义的依赖，最终可以对应到多版本代码管理系统中某一项目的特定提交或版本，
这样的话，对于go.mod文件中定义的依赖，则可以直接从对应仓库中下载指定软件依赖，从而完成依赖分发。

但直接使用版本管理仓库下载依赖，存在多个问题，首先无法保证构建确定性；软件作者可以在代码平台直接增加/删除软件版本，导致
下次构建使用其他版本的依赖，或者找不到依赖版本，无法保证依赖可用性；依赖软件作者可以直接在代码平台删除软件，导致依赖不可用，
大幅增加第三方代码托管平台压力。

![go-proxy.png](images%2Fgo-proxy.png)

而go proxy就是解决这些问题的方案，Go Proxy是一个服务站点，它会缓存源站中的软件内容，缓存的软件版本不会改变，并且在源站
软件删除之后依然可用，从而实现了不可变的和高可用的依赖分发；使用go proxy后，构建时会直接从go proxy站点拉取依赖。

```shell
GOPROXY="https://proxy1.cn,https://proxy2.cn,direct"
```
上面的proxy1和proxy2表示服务站点URL列表，direct表示源站
Proxy1 -> Proxy2 -> Direct

接下来讲一下go proxy的使用，Go Modules通过GOPROXY环境变量控制如何使用go proxy；go proxy是一个go proxy站点URL列表，
可以使用direct表示源站。对于上面这个示例配置，整体的依赖寻址路径，会优先从proxy1下载依赖，如果proxy1不存在，会从
proxy2寻找，如果proxy2中也不存在则会回到源站中直接下载依赖，缓存到proxy站点中。

go get工具

![go-get.png](images%2Fgo-get.png)

go mod init 初始化，创建go.mod文件
go mod download 下载模块到本地缓存
tidy  增加需要的依赖，删除不需要的依赖

git commit提交代码前尽量执行下go mod tidy，减少构建时无效依赖包的拉取。

2.4 > 案例分析

case 1 执行go get -u xxx后项目编译不通过。
直接原因: 升级了github.com/apache/thrift但是0.14.x与0.13.x两个版本不兼容
根本原因: 在执行go get时错误使用了-u参数更新了依赖的依赖。
![go-get-case.png](images%2Fgo-get-case.png)

经验: 如无必要，不使用-u参数

case 2 为什么一些工程不使用go mod?
![go-mod-indirect.png](images%2Fgo-mod-indirect.png)

出现indirect的原因是:
在go mod推广前，major版本已>1
便于go get拉到v3版本

经验: 在复杂的业务场景中，新项目尽量不使用>1的major版本号

case 3 部分项目不适用go mod导致的复杂场景: 最终参与编译的x是v1还是v2?
![complex-case.png](images%2Fcomplex-case.png)

在A没有依赖C的情况下，会使用x的v2版本；
但由于C的存在，使得x的依赖被指定为了v1版本。

经验:
无论依赖库是否使用go mod，我们得项目中都应该使用go mod；
特殊情况可以手动指定indirect依赖。

case 4 删除tag/branch/commit后导致依赖报错，无法通过编译
![case-tag.png](images%2Fcase-tag.png)

如何解决？
若只有本项目依赖，在删除go.mod/go.sum中的条目后，go get更新到最新tag;
若依赖项中也依赖，则replace该依赖至正常tag，彻底解决需go get更新依赖链路上的所有项目到最新tag；

更可怕的是，删除tag后在另外一个commit上重新打相同的tag...
这时只能清理本地缓存，重新拉取。

case5 循环依赖陷阱
循环依赖如何产生？
两个package之间是不能互相import导入的；
但两个不同工程的不同package之间是可以互相import的。
![loop.png](images%2Floop.png)

循环依赖一旦形成，内部所有依赖的所有版本都会一直保留。

a> 某基础库A与基础库B存在循环依赖，且依赖链中B的某个版本依赖了A的某个分支上上的commit。某天，A的该分支
在清理分支时被删除，导致众多项目无法编译上线。

![loop-one.png](images%2Floop-one.png)

b> 某团队内部俩公共库A、B存在循环依赖，此时某基础库C的某版本存在高风险bug，为收敛问题删除了一些tag，导致两个
公共库被迫replace掉依赖C，且所有依赖该基础库的服务都需要更新依赖，尽管C的那个有问题的版本已经不再被A、B依赖。

经验: 公共库之间应该分工明确，避免大杂烩，避免循环依赖。