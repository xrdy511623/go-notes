
---
go依赖管理详解
---

# 1 背景(为什么需要依赖管理？)
工程项目不可能基于标准库0~1编码搭建
管理依赖库

# 2 Go依赖管理演进
GOPATH -> Go Vendor -> Go Module

## 2.1 > GOPATH

```shell
cd go && ls
bin pkg src
```

> bin 是项目编译的二进制文件
> pkg 是项目编译的中间产物,用于加速编译
> src 项目源码

项目代码直接依赖src下的代码
go get会下载最新版本的包到src目录下

存在的问题：
在类似下面的场景中(A和B依赖于某一package的不同版本)，无法实现包(package)的多版本控制。
因为src目录下只能有一个版本存在，那A和B两个项目只能有一个编译通过，这显然无法满足我们的需求。

![GOPATH.png](images%2FGOPATH.png)

## 2.2 Go Vendor
在项目目录下增加vendor文件，所有依赖包以副本形式放在$ProjectRoot/vendor
> 依赖寻址方式: vendor -> GOPATH

![Go-Vendor.png](images%2FGo-Vendor.png)

如此，通过每个项目引入一份依赖的副本，解决了多个项目需要同一个package依赖的冲突问题。

存在的问题：

![Vendor-problem.png](images%2FVendor-problem.png)

在复杂的依赖关系下，无法控制依赖的版本。譬如项目A依赖package B和C，而B和C又分别依赖了D的不同版本v1和v2，在vendor的
管理模式下我们不能很好的控制对于D的依赖版本，一旦更新项目有可能出现依赖冲突，导致编译出错，归根结底，还是因为vendor
不能很清晰的标识依赖的版本概念。

## 2.3 Go Module
终极目标: 定义版本规则和管理项目的依赖关系

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

> 依赖标识: Module Path Version {Major}.{Minor}.{Patch}

以github.com/gin-gonic/gin 为例
github.com/gin-gonic/gin是gin包的模块路径，从这个路径可以看出从哪里找到该模块，譬如如果是github前缀则表示可以从Github仓库
找到该模块，依赖包的源代码由github托管，如果项目的子包想被单独引用，则需要通过单独的go mod init生成的mod文件进行管理。
**v1.4.9是语义化版本号，1是大版本(Major)，4是小版本(Minor)，最后一个9是 patch(修复)版本号**
不同的Major版本表示是不兼容的API，所以即使是同一个库(包)，Major版本不同也会被认为是不同的模块；Minor版本通常是新增函数
或功能，向后兼容；而patch版本一般是修复bug。

Minor版本向后兼容意味着：
新增功能不会破坏已有功能：
新增的函数、方法、类型或特性只会扩展库的功能，而不会移除或修改已有的 API。
客户端代码如果依赖旧版本的功能，在更新到新的 Minor 版本时可以继续正常运行。

接口不变：
已有的函数签名、类型定义、常量等不会改变。
不会删除已公开的任何 API。

不引入破坏性变更：
不改变已有 API 的行为（例如输入相同的参数会返回相同的结果）。

如果需要进行不向后兼容的更改，例如删除函数或改变函数行为，那么应该增加 Major 版本号。例如，从 v1.x.x 更新到 v2.0.0。

> 依赖配置-version

语义化版本
${Major}.${Minor}.${Patch}
V1.3.0
V1.2.1

> 基于commit伪版本

v1.0.1-20231109134442-10cbfed86s6y

这种基础版本前缀和语义化版本是一样的;后面的20231109134442是时间戳，是Commit提交的时间，最后的10cbfed86s6y是校验码，
包含了12位的哈希前缀，每次提交commit后Go都会默认生成一个伪版本号。

github.com/go-errors/errors v1.0.1 // indirect
**indirect表示go.mod对应的当前模块，没有直接导入该依赖模块的包，也就是不是直接依赖，是间接依赖。**
A->B->C
**那么在这个依赖链条里，A对B就是直接依赖，A对C就是间接依赖。**

github.com/uber/jaeger-client-go v2.29.1+incompatible

主版本2+模块会在模块路径增加/vN后缀，这能让go module按照不同的模块来处理同一个项目不同主版本的依赖。由于Go Module是
Go 1.11实验性引入的，所以这项规则提出之前已经有一些仓库打上了2或者更高版本的tag了，为了兼容这部分仓库，对于没有使用go.mod
文件并且主版本在2或者以上的依赖，会在版本号后加上+incompatible后缀。

![normal-go-mod.png](images%2Fnormal-go-mod.png)

![incompatible-no-go-mod.png](images%2Fincompatible-no-go-mod.png)

**什么情况下会出现 +incompatible**
未使用 Go Modules 的旧库：
如果一个库发布的版本号大于 v2（例如 v2.x.x），但它没有提供 go.mod 文件，那么 Go 会在模块版本号后添加 +incompatible，
表示它不完全兼容 Go Modules。

未遵循 Go Modules 的规则：
根据 Go Modules 的规则，如果库的 Major 版本号大于等于 2（例如 v2.x.x），模块的导入路径必须包含 Major 版本号
（即路径应为 module/path/v2）。如果库未遵守这一规则，则会被标记为 +incompatible。

示例分析
库未使用 Go Module： 假设 github.com/uber/jaeger-client-go 发布了 v2.29.1 版本，但它没有 go.mod 文件。
由于缺少 go.mod，Go Modules 无法判断模块是否完全支持模块化，因此会标记为 +incompatible。

库未调整导入路径： 如果该库的导入路径仍是 github.com/uber/jaeger-client-go（没有包含 /v2），即使有 go.mod，
也会标记为 +incompatible。

使用 +incompatible 的模块
即使标记为 +incompatible，这些模块仍然可以正常使用，但需要注意：
导入路径不会包含版本号（例如，直接使用 github.com/uber/jaeger-client-go）。
需要手动检查兼容性，确保该模块与您的代码或其他依赖项不会冲突。

如何解决 +incompatible？
如果您是模块的维护者，可以通过以下方式消除 +incompatible：
添加 go.mod 文件：
使用 go mod init 命令生成 go.mod 文件，明确声明模块化支持。

调整导入路径：
如果 Major 版本号 >= 2，更新模块路径为 module/path/v2 并在 go.mod 中声明，例如：

```go
module github.com/uber/jaeger-client-go/v2
```

Go Module 内部使用 **MVS（Minimal Version Selection，最小版本选择）** 算法来决定最终使用的依赖版本：在所有满足约束的版本中，选择能满足所有依赖方需求的**最低**版本，而非最新版本。这使得构建结果确定且可复现。

如果X项目依赖了A、B两个项目，且A、B分别依赖了C项目的v1.3、V1.4两个版本，最终编译时所使用的C项目的版本是?
A v1.3
B v1.4
C A用到C时用v1.3编译，B用到C时用v1.4编译

答案是B，选择最低地兼容版本。因为Minor是向后兼容的，所以用1.4一定是在1.3的基础上新增函数或功能，使用1.3那么B就无法
使用1.4新增的功能，无法通过编译，而使用1.4则不影响1.3原有功能的使用，所以会使用v1.4版本。

**接下来讲一下go module的依赖分发，也就是从哪里下载，如何下载的问题**
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

> go get工具

![go-get.png](images%2Fgo-get.png)

go mod init 初始化，创建go.mod文件
go mod download 下载模块到本地缓存
tidy  增加需要的依赖，删除不需要的依赖

git commit提交代码前尽量执行下go mod tidy，减少构建时无效依赖包的拉取。

## 2.4 案例分析

### 2.4.1 执行go get -u xxx后项目编译不通过。
直接原因: 升级了github.com/apache/thrift但是0.14.x与0.13.x两个版本不兼容
根本原因: 在执行go get时错误使用了-u参数更新了依赖的依赖。
![go-get-case.png](images%2Fgo-get-case.png)

**经验: 如无必要，不要使用-u参数**

### 2.4.2 为什么一些工程不使用go mod?
![go-mod-indirect.png](images%2Fgo-mod-indirect.png)

出现indirect的原因是:
在go mod推广前，major版本已>1
便于go get拉到v3版本

**经验: 在复杂的业务场景中，新项目尽量不使用>1的major版本号**

### 2.4.3 部分项目不适用go mod导致的复杂场景: 最终参与编译的x是v1还是v2?
![complex-case.png](images%2Fcomplex-case.png)

在A没有依赖C的情况下，会使用x的v2版本；
但由于C的存在，使得x的依赖被指定为了v1版本。

**经验:**
**无论依赖库是否使用go mod，我们的项目中都应该使用go mod；
特殊情况可以手动指定indirect依赖。**

### 2.4.4 删除tag/branch/commit后导致依赖报错，无法通过编译
![case-tag.png](images%2Fcase-tag.png)

如何解决？
若只有本项目依赖，在删除go.mod/go.sum中的条目后，go get更新到最新tag;
若依赖项中也依赖，则replace该依赖至正常tag，彻底解决需go get更新依赖链路上的所有项目到最新tag；

更可怕的是，删除tag后在另外一个commit上重新打相同的tag...
这时只能清理本地缓存，重新拉取。

### 2.4.5 循环依赖陷阱
循环依赖如何产生？
两个package之间是不能互相import导入的；
但两个不同工程的不同package之间是可以互相import的。
![loop.png](images%2Floop.png)

循环依赖一旦形成，内部所有依赖的所有版本都会一直保留。

> 某基础库A与基础库B存在循环依赖，且依赖链中B的某个版本依赖了A的某个分支上的commit。某天，A的该分支
在清理分支时被删除，导致众多项目无法编译上线。

![loop-one.png](images%2Floop-one.png)

> 某团队内部俩公共库A、B存在循环依赖，此时某基础库C的某版本存在高风险bug，为收敛问题删除了一些tag，导致两个
公共库被迫replace掉依赖C，且所有依赖该基础库的服务都需要更新依赖，尽管C的那个有问题的版本已经不再被A、B依赖。

**经验: 公共库之间应该分工明确，避免大杂烩，避免循环依赖。**


# 3 go.sum 与安全校验

go.mod 描述的是"要用哪个版本"，而 **go.sum 记录的是"这个版本的内容是否被篡改"**。

go.sum 中的每一行形如：
```
github.com/gin-gonic/gin v1.9.1 h1:4idEAncQnU5cB7BeOkPtxjfCSye0AAm1R0RVIqJ+Jmg=
github.com/gin-gonic/gin v1.9.1/go.mod h1:hPrL7YrpYKXt5YId3A/Tnip5kqbEAP+KLuI3SUcPTeU=
```
- 第一列：模块路径和版本
- `h1:` 后面：该版本 zip 包的 SHA-256 哈希值
- `/go.mod` 行：go.mod 文件本身的哈希值

**go.sum 的作用**
- 每次下载依赖时，Go 工具链会验证下载内容的哈希是否与 go.sum 中记录的一致。
- 防止依赖被悄悄替换（供应链攻击）。
- 应当提交到版本控制系统，保证团队所有人、CI 构建使用完全相同的依赖内容。

**GONOSUMCHECK 与 GONOSUMDB**

Go 维护了一个公共的 checksum 数据库 `sum.golang.org`，用于二次验证哈希值。对于私有模块，不应将代码信息发送到公共 checksum 数据库：

```shell
# 跳过指定模块的 checksum 校验（逗号分隔，支持通配符）
GONOSUMDB=*.internal.company.com,github.com/company/*

# 同时控制 GOPROXY 和 GONOSUMDB
GOPRIVATE=*.internal.company.com
```

`GOPRIVATE` 是 `GONOSUMDB` 和 `GONOPROXY` 的组合简写，配置后该模式匹配的模块会绕过代理和 checksum 数据库，直接访问源站。


# 4 go.mod 进阶指令

除了 `require`，go.mod 还支持以下指令：

## 4.1 replace

将某个依赖替换为另一个版本或本地路径，常用于：
- 临时使用 fork 版本修复 bug
- 本地调试修改依赖库
- 解决 2.4.4 中删除 tag 的问题

```go
replace (
    // 替换为另一个版本
    github.com/foo/bar v1.2.0 => github.com/myfork/bar v1.2.1

    // 替换为本地路径（本地调试时常用）
    github.com/foo/bar v1.2.0 => ../bar
)
```

注意：`replace` 只对当前模块生效，不会传递给上游依赖方。

## 4.2 exclude

明确排除某个版本，Go 工具链在解析依赖时会跳过该版本，选择下一个可用版本：

```go
exclude (
    github.com/foo/bar v1.2.3  // 该版本有严重 bug，禁止使用
)
```

## 4.3 retract（Go 1.16+）

供模块**作者**使用，标记已发布版本中有问题的版本，提醒使用者升级：

```go
retract (
    v1.3.0         // 该版本存在数据竞争问题
    [v1.4.0, v1.4.5]  // 该区间内所有版本存在安全漏洞
)
```

与 `exclude` 的区别：`retract` 是作者在自己模块中声明，`exclude` 是使用者在自己模块中声明。


# 5 go work 工作区（Go 1.18+）

在同时开发多个相关模块时（例如主项目 + 正在修改的依赖库），过去只能用 `replace` 临时指向本地路径，且容易误提交。Go 1.18 引入了 **workspace 模式**，通过 `go.work` 文件优雅地解决这个问题。

```shell
# 在多模块的父目录下初始化 workspace
go work init ./myapp ./mylib

# 添加更多模块到 workspace
go work use ./another-module
```

生成的 `go.work` 文件：
```
go 1.21

use (
    ./myapp
    ./mylib
)
```

workspace 生效后，`myapp` 对 `mylib` 的引用会优先使用本地路径，无需修改任何 `go.mod`。

**注意：`go.work` 不应提交到版本控制系统**，它是开发者本地的临时配置（加入 `.gitignore`）。


# 6 常用命令速查

| 命令 | 作用 |
|------|------|
| `go mod init <module>` | 初始化模块，生成 go.mod |
| `go mod tidy` | 增加缺失依赖，删除无用依赖，更新 go.sum |
| `go mod download` | 下载所有依赖到本地缓存（`$GOPATH/pkg/mod`）|
| `go mod vendor` | 将依赖复制到项目 vendor 目录（用于离线构建）|
| `go mod verify` | 验证本地缓存的依赖是否与 go.sum 一致 |
| `go mod graph` | 打印完整的依赖图 |
| `go mod why <pkg>` | 解释为什么需要某个依赖 |
| `go get <pkg>@<version>` | 添加/升级/降级依赖到指定版本 |
| `go get <pkg>@none` | 从 go.mod 中移除某个依赖 |
| `go list -m all` | 列出所有直接和间接依赖 |

**提交代码前的清单**
- `go mod tidy` 确保 go.mod / go.sum 整洁
- 将 go.mod 和 go.sum **都提交**到版本控制
- 不要提交 go.work（本地工作区配置）
- 如无必要，不使用 `go get -u`（会连带升级间接依赖）