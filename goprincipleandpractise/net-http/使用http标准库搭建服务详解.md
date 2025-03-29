
---
使用http标准库搭建服务详解
---

# 1 quick start

```go
// 创建一个Foo路由和处理函数
http.Handle("/foo", fooHandler)

// 创建一个bar路由和处理函数
http.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
  fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
})

// 监听8080端口
log.Fatal(http.ListenAndServe(":8080", nil))
```

代码是很简单的，一共就 5 行，但是这五行代码做了什么，为什么就能启动一个 HTTP 服务，具体的逻辑是什么样的？
要回答这些问题，就要深入理解 net/http 标准库。要不然，只会简单调用，却不知道原理，后面哪里出了问题，或者想调优，
就无从下手了。所以，我们先来看看 net/http 标准库，从代码层面搞清楚整个 HTTP 服务的主流程原理，最后再基于原理讲实现。


# 2 net/http 标准库怎么学

想要在 net/http 标准库纷繁复杂的代码层级和调用中，弄清楚主流程不是一件容易事。要快速熟悉一个标准库，就得找准方法。
一个快速掌握代码库的技巧：库函数 > 结构定义 > 结构函数。

简单来说，就是当你在阅读一个代码库的时候，不应该从上到下阅读整个代码文档，而应该先阅读整个代码库提供的对外库函数
（function），再读这个库提供的结构（struct/class），最后再阅读每个结构函数（method）。

> 库函数: 这个库提供什么功能？
> 结构定义: 整个库分为几个核心模块？
> 结构函数：每个核心模块应该提供什么能力？

为什么要这么学呢？因为这种阅读思路和代码库作者的思路是一致的。首先搞清楚这个库要提供什么功能（提供什么样的对外函数），
然后为了提供这些功能，我要把整个库分为几个核心模块（结构），最后每个核心模块，我应该提供什么样的能力（具体的结构函数）
来满足我的需求。

## 2.1 库函数（功能）

按照这个思路，我们来阅读 net/http 库，先看提供的对外库函数是为了实现哪些功能。

你直接通过 go doc net/http | grep "^func" 命令行能查询出 net/http 库所有的对外库函数：

```go
func CanonicalHeaderKey(s string) string
func DetectContentType(data []byte) string
func Error(w ResponseWriter, error string, code int)
func Get(url string) (resp *Response, err error)
func Handle(pattern string, handler Handler)
func HandleFunc(pattern string, handler func(ResponseWriter, *Request))
func Head(url string) (resp *Response, err error)
func ListenAndServe(addr string, handler Handler) error
func ListenAndServeTLS(addr, certFile, keyFile string, handler Handler) error
func MaxBytesReader(w ResponseWriter, r io.ReadCloser, n int64) io.ReadCloser
func NewRequest(method, url string, body io.Reader) (*Request, error)
func NewRequestWithContext(ctx context.Context, method, url string, body io.Reader) (*Request, error)
func NotFound(w ResponseWriter, r *Request)
func ParseHTTPVersion(vers string) (major, minor int, ok bool)
func ParseTime(text string) (t time.Time, err error)
func Post(url, contentType string, body io.Reader) (resp *Response, err error)
func PostForm(url string, data url.Values) (resp *Response, err error)
func ProxyFromEnvironment(req *Request) (*url.URL, error)
func ProxyURL(fixedURL *url.URL) func(*Request) (*url.URL, error)
func ReadRequest(b *bufio.Reader) (*Request, error)
func ReadResponse(r *bufio.Reader, req *Request) (*Response, error)
func Redirect(w ResponseWriter, r *Request, url string, code int)
func Serve(l net.Listener, handler Handler) error
func ServeContent(w ResponseWriter, req *Request, name string, modtime time.Time, ...)
func ServeFile(w ResponseWriter, r *Request, name string)
func ServeTLS(l net.Listener, handler Handler, certFile, keyFile string) error
func SetCookie(w ResponseWriter, cookie *Cookie)
func StatusText(code int) string
```

在这个库提供的方法中，我们去掉一些 New 和 Set 开头的函数，因为你从命名上可以看出，这些函数是对某个对象或者属性的设置。
剩下的函数大致可以分成三类：

为服务端提供创建 HTTP 服务的函数，名字中一般包含 Serve 字样，比如 Serve、ServeFile、ListenAndServe 等。
为客户端提供调用 HTTP 服务的类库，以 HTTP 的 method 同名，比如 Get、Post、Head 等。提供中转代理的一些函数，
比如 ProxyURL、ProxyFromEnvironment 等。我们现在研究的是，如何创建一个 HTTP 服务，所以关注包含 Serve 
字样的函数就可以了。

```go
// 通过监听的URL地址和控制器函数来创建HTTP服务
func ListenAndServe(addr string, handler Handler) error{}
// 通过监听的URL地址和控制器函数来创建HTTPS服务
func ListenAndServeTLS(addr, certFile, keyFile string, handler Handler) error{}
// 通过net.Listener结构和控制器函数来创建HTTP服务
func Serve(l net.Listener, handler Handler) error{}
// 通过net.Listener结构和控制器函数来创建HTTPS服务
func ServeTLS(l net.Listener, handler Handler, certFile, keyFile string) error{}
```

## 2.2 结构定义（模块）

然后，我们过一遍这个库提供的所有 struct，看看核心模块有哪些，同样使用 go doc:

```go
 go doc net/http | grep "^type"|grep struct
```

你可以看到整个库最核心的几个结构：

```go
type Client struct{ ... }
type Cookie struct{ ... }
type ProtocolError struct{ ... }
type PushOptions struct{ ... }
type Request struct{ ... } 
type Response struct{ ... }
type ServeMux struct{ ... }
type Server struct{ ... }
type Transport struct{ ... }
```

看结构的名字或者 go doc 查看结构说明文档，能逐渐了解它们的功能：Client 负责构建 HTTP 客户端；
Server 负责构建 HTTP 服务端；ServerMux 负责 HTTP 服务端路由；Transport、Request、Response、Cookie 
负责客户端和服务端传输对应的不同模块。

现在通过库方法（function）和结构体（struct），我们对整个库的结构和功能有大致印象了。整个库承担了两部分功能，
一部分是构建 HTTP 客户端，一部分是构建 HTTP 服务端。构建的 HTTP 服务端除了提供真实服务之外，也能提供代理中转服务，
它们分别由 Client 和 Server 两个数据结构负责。除了这两个最重要的数据结构之外，HTTP 协议的每个部分，
比如请求、返回、传输设置等都有具体的数据结构负责。

## 2.3 结构函数(能力)

下面从具体的需求出发，我们来阅读具体的结构函数（method）。我们当前的需求是创建 HTTP 服务，开头举了一个最简单的例子：

```go
// 创建一个Foo路由和处理函数
http.Handle("/foo", fooHandler)

// 创建一个bar路由和处理函数
http.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
  fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
})

// 监听8080端口
log.Fatal(http.ListenAndServe(":8080", nil))
```

我们跟着 http.ListenAndServe 这个函数来理一下 net/http 创建服务的主流程逻辑。阅读具体的代码逻辑用 go doc 命令明显
就不够了，你需要两个东西：

一个是可以灵活进行代码跳转的 IDE，VS Code 和 GoLand 都是非常好的工具。以我们现在要查看的 http.ListenAndServe 
这个函数为例，我们可以从上面的例子代码中，直接通过 IDE 跳转到这个函数的源码中阅读，有一个能灵活跳转的 IDE 工具
是非常必要的。

另一个是可以方便记录代码流程的笔记，这里推荐使用思维导图。

具体方法是将要分析的代码从入口处一层层记录下来，每个函数，我们只记录其核心代码，然后对每个核心代码一层层解析。记得把思维
导图的结构设置为右侧分布，这样更直观。


![listen_and_serve.png](images%2Flisten_and_serve.png)


这张图看上去层级复杂，不过不用担心，对照着思维导图，我们一层一层阅读，讲解每一层的逻辑，看清楚代码背后的设计思路。
我们先顺着 http.ListenAndServe 的脉络读。

第一层，http.ListenAndServe 本质是通过创建一个 Server 数据结构，调用 server.ListenAndServe 对外提供服务，
这一层完全是比较简单的封装，目的是，将 Server 结构创建服务的方法 ListenAndServe ，直接作为库函数对外提供，
增加库的易用性。

进入到第二层，创建服务的方法 ListenAndServe 先定义了监听信息 net.Listen，然后调用 Serve 函数。

而在第三层 Serve 函数中，用了一个 for 循环，通过 l.Accept不断接收从客户端传进来的请求连接。当接收到了一个新的请求
连接的时候，通过 srv.NewConn创建了一个连接结构（http.conn），并创建一个 Goroutine 为这个请求连接对应服务（c.serve）。

![conn_serve.png](images%2Fconn_serve.png)


在第四层，c.serve函数先判断本次 HTTP 请求是否需要升级为 HTTPs，接着创建读文本的 reader 和写文本的 buffer，
再进一步读取本次请求数据，然后第五层调用最关键的方法 serverHandler{c.server}.ServeHTTP(w, w.req) ，
来处理这次请求。这个关键方法是为了实现自定义的路由和业务逻辑，调用写法是比较有意思的：


```go
serverHandler{c.server}.ServeHTTP(w, w.req)
```

serverHandler 结构体，是标准库封装的，代表“请求对应的处理逻辑”，它只包含了一个指向总入口服务 server 的指针。
这个结构将总入口的服务结构 Server 和每个连接的处理逻辑巧妙联系在一起了，你可以看接着的第六层逻辑：

```go
// serverHandler 结构代表请求对应的处理逻辑
type serverHandler struct {
  srv *Server
}

// 具体处理逻辑的处理函数
func (sh serverHandler) ServeHTTP(rw ResponseWriter, req *Request) {
  handler := sh.srv.Handler
  if handler == nil {
    handler = DefaultServeMux
  }
  ...
  handler.ServeHTTP(rw, req)
}
```

如果入口服务 server 结构已经设置了 Handler，就调用这个 Handler 来处理此次请求，反之则使用库自带的 DefaultServerMux。这里的 serverHandler 设计，
能同时保证这个库的扩展性和易用性：你可以很方便使用默认方法处理请求，但是一旦有需求，也能自己扩展出方法处理请求。
那么 DefaultServeMux 是怎么寻找 Handler 的呢，这就是思维导图的最后一部分第七层。

DefaultServeMux.Handle 是一个非常简单的 map 实现，key 是路径（pattern），value 是这个 pattern 对应的处理函数（handler）。
它是通过 mux.match(path) 寻找对应 Handler，也就是从 DefaultServeMux 内部的 map 中直接根据 key 寻找到 value 的。


好，HTTP 库 Server 的代码流程我们就梳理完成了，整个逻辑线大致是：

创建服务 -> 监听请求 -> 创建连接 -> 处理请求

如果觉得层次比较多，对照着思维导图多看几遍就顺畅了。这里整理了一下逻辑线各层的关键结论：

第一层，标准库创建 HTTP 服务是通过创建一个 Server 数据结构完成的；
第二层，Server 数据结构在 for 循环中不断监听每一个连接；
第三层，每个连接默认开启一个 Goroutine 为其服务；
第四、五层，serverHandler 结构代表请求对应的处理逻辑，并且通过这个结构进行具体业务逻辑处理；
第六层，Server 数据结构如果没有设置处理函数 Handler，默认使用 DefaultServerMux 处理请求；
第七层，DefaultServerMux 是使用 map 结构来存储和查找路由规则。

**创建框架的 Server 结构**

现在原理弄清楚了，该下手搭 HTTP 服务了。刚刚咱也分析了主流程代码，其中第一层的关键结论就是：net/http 标准库创建服务，
实质上就是通过创建 Server 数据结构来完成的。所以接下来，我们就来创建一个 Server 数据结构。通过
go doc net/http.Server 我们可以看到 Server 的结构：

```go
type Server struct {
    // 请求监听地址
  Addr string
    // 请求核心处理函数
  Handler Handler 
  ...
}
```
其中最核心的是 Handler 这个字段，从主流程中我们知道（第六层关键结论），当 Handler 这个字段设置为空的时候，它会默认
使用 DefaultServerMux 这个路由器来填充这个值，但是我们一般都会使用自己定义的路由来替换这个默认路由。