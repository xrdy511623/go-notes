
---
WebSocket 详解
---

# 1 什么是 WebSocket？

WebSocket 是一种在单个 TCP 连接上进行**全双工通信**的协议（RFC 6455）。与 HTTP 的请求-响应模型不同，
WebSocket 允许服务器主动向客户端推送数据，客户端也可以随时向服务器发送数据，非常适合实时场景。

```
传统 HTTP:
  Client --request-->  Server
  Client <--response-- Server    （每次都要重新建立/复用连接，且必须由客户端发起）

WebSocket:
  Client --upgrade-->  Server    （一次 HTTP 握手）
  Client <==双向==>    Server    （之后长连接，双方随时发送）
```

## 1.1 WebSocket 与 HTTP 的关系

WebSocket 并不是替代 HTTP 的协议，而是**建立在 HTTP 之上**的升级协议：

1. 客户端发送一个带有 `Upgrade: websocket` 头的 HTTP 请求
2. 服务器返回 `101 Switching Protocols`
3. 之后 TCP 连接不再走 HTTP 协议，而是 WebSocket 帧协议

```
GET /chat HTTP/1.1
Host: server.example.com
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
Sec-WebSocket-Version: 13

HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
```

`Sec-WebSocket-Key` 是客户端生成的随机 Base64 字符串，服务器将其与固定 GUID
`258EAFA5-E914-47DA-95CA-5AB5DC11E5B0` 拼接后取 SHA-1 再 Base64，返回 `Sec-WebSocket-Accept`。
这不是加密，只是为了防止非 WebSocket 客户端误连。

## 1.2 WebSocket 帧结构

握手完成后，数据以**帧（frame）**为单位传输：

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-------+-+-------------+-------------------------------+
|F|R|R|R| opcode|M| Payload len |    Extended payload length    |
|I|S|S|S|  (4)  |A|     (7)     |           (16/64)             |
|N|V|V|V|       |S|             |   (if payload len==126/127)   |
| |1|2|3|       |K|             |                               |
+-+-+-+-+-------+-+-------------+-------------------------------+
|     Masking-key (if MASK=1)   |                               |
+-------------------------------+-------------------------------+
|                     Payload Data                              |
+---------------------------------------------------------------+
```

关键字段：
- **FIN**: 是否是消息的最后一帧（支持消息分片）
- **opcode**: 0x1=文本帧, 0x2=二进制帧, 0x8=关闭, 0x9=Ping, 0xA=Pong
- **MASK**: 客户端→服务器的帧必须 mask（防止缓存投毒攻击），服务器→客户端不需要
- **Payload len**: 7bit 可表示 0-125 字节，126 表示接下来 2 字节是长度，127 表示接下来 8 字节是长度

## 1.3 为什么需要 WebSocket？

| 方案 | 延迟 | 服务器资源 | 适用场景 |
|------|------|-----------|---------|
| HTTP 轮询 | 高（轮询间隔） | 高（大量无效请求） | 简单场景 |
| HTTP 长轮询 | 中 | 中（连接挂起） | 兼容性要求高 |
| SSE (Server-Sent Events) | 低 | 低 | 服务器→客户端单向推送 |
| **WebSocket** | **极低** | **低** | **双向实时通信** |

典型使用场景：
- 即时通讯（聊天室、在线客服）
- 实时数据推送（股票行情、体育比分、监控面板）
- 协同编辑（在线文档、白板）
- 多人游戏
- IoT 设备通信

# 2 Go 中的 WebSocket 库

Go 标准库 `net/http` 不直接支持 WebSocket，需要使用第三方库。主流选择有两个：

| 特性 | gorilla/websocket | nhooyr.io/websocket (coder/websocket) |
|------|------------------|--------------------------------------|
| 维护状态 | 归档（2022），社区 fork 活跃 | 活跃维护 |
| API 风格 | 底层，手动管理连接 | 高层，context 集成 |
| 并发写安全 | ❌ 需自行加锁 | ✅ 内置并发安全 |
| 压缩 | ✅ permessage-deflate | ✅ permessage-deflate |
| WASM 支持 | ❌ | ✅ |
| 性能 | 略高（少一层抽象） | 优秀 |
| `context.Context` | 部分支持 | 全面集成 |

**选型建议**：
- 新项目推荐 `nhooyr.io/websocket`（又名 `github.com/coder/websocket`），API 更符合现代 Go 惯例
- 已有项目使用 `gorilla/websocket` 的无需迁移，生态成熟

本文两个库都会覆盖，以 `gorilla/websocket` 为主（使用最广泛），并给出 `nhooyr.io/websocket` 的对比示例。

# 3 基础用法：gorilla/websocket

## 3.1 服务端

```go
package main

import (
    "log"
    "net/http"

    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    // 生产环境务必校验 Origin
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("upgrade failed: %v", err)
        return
    }
    defer conn.Close()

    for {
        msgType, msg, err := conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err,
                websocket.CloseGoingAway,
                websocket.CloseNormalClosure) {
                log.Printf("read error: %v", err)
            }
            break
        }
        log.Printf("received: %s", msg)

        if err := conn.WriteMessage(msgType, msg); err != nil {
            log.Printf("write error: %v", err)
            break
        }
    }
}

func main() {
    http.HandleFunc("/ws", echoHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## 3.2 客户端

```go
package main

import (
    "fmt"
    "log"

    "github.com/gorilla/websocket"
)

func main() {
    conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
    if err != nil {
        log.Fatal("dial:", err)
    }
    defer conn.Close()

    // 发送消息
    err = conn.WriteMessage(websocket.TextMessage, []byte("Hello WebSocket"))
    if err != nil {
        log.Fatal("write:", err)
    }

    // 读取响应
    _, msg, err := conn.ReadMessage()
    if err != nil {
        log.Fatal("read:", err)
    }
    fmt.Printf("received: %s\n", msg)
}
```

## 3.3 使用 nhooyr.io/websocket

同样的 echo 服务器，用 `nhooyr.io/websocket` 实现：

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"

    "nhooyr.io/websocket"
)

func echoHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
        // 生产环境务必校验 Origin
        InsecureSkipVerify: true,
    })
    if err != nil {
        log.Printf("accept failed: %v", err)
        return
    }
    defer conn.CloseNow()

    ctx := r.Context()
    for {
        msgType, data, err := conn.Read(ctx)
        if err != nil {
            break
        }
        if err := conn.Write(ctx, msgType, data); err != nil {
            break
        }
    }
    conn.Close(websocket.StatusNormalClosure, "")
}

func main() {
    http.HandleFunc("/ws", echoHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

注意关键差异：
- `websocket.Accept` 替代了 `upgrader.Upgrade`，无需预创建 Upgrader
- 读写方法接受 `context.Context`，可以方便地设置超时和取消
- `conn.Write` 是并发安全的，不需要额外加锁

# 4 生产级架构：聊天室实现

一个完整的聊天室需要解决以下问题：
1. 连接管理（注册/注销）
2. 消息广播
3. 并发写安全
4. 心跳保活
5. 优雅关闭

## 4.1 Hub 模式

Hub（消息中心）是 WebSocket 应用最经典的架构模式，gorilla/websocket 官方示例也采用此模式：

```
                    ┌─────────┐
                    │   Hub   │
                    │         │
          register  │ clients │  broadcast
  Client ─────────→ │   map   │ ←───────── Client
                    │         │
         unregister │         │
  Client ─────────→ │         │
                    └─────────┘
                         │
                    broadcast
                    ┌────┼────┐
                    ▼    ▼    ▼
                   C1   C2   C3
```

```go
type Hub struct {
    clients    map[*Client]struct{}
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
}

func NewHub() *Hub {
    return &Hub{
        clients:    make(map[*Client]struct{}),
        broadcast:  make(chan []byte, 256),
        register:   make(chan *Client),
        unregister: make(chan *Client),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = struct{}{}

        case client := <-h.unregister:
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }

        case message := <-h.broadcast:
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    // 发送缓冲区满，说明客户端处理不过来，断开连接
                    close(client.send)
                    delete(h.clients, client)
                }
            }
        }
    }
}
```

**为什么 Hub.Run 是单 goroutine？**

Hub 的 `clients` map 只在 `Run` 这一个 goroutine 中访问，不需要加锁。
所有对 clients 的操作都通过 channel 序列化。这是 Go 的经典并发模式：
**"Don't communicate by sharing memory; share memory by communicating."**

## 4.2 Client 模式

每个 WebSocket 连接对应一个 Client，Client 启动两个 goroutine：
- **readPump**: 从 WebSocket 读消息，发到 Hub 的 broadcast channel
- **writePump**: 从自己的 send channel 读消息，写到 WebSocket

```go
type Client struct {
    hub  *Hub
    conn *websocket.Conn
    send chan []byte
}

const (
    writeWait      = 10 * time.Second
    pongWait       = 60 * time.Second
    pingPeriod     = (pongWait * 9) / 10  // 必须小于 pongWait
    maxMessageSize = 4096
)

func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()
    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })
    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            break
        }
        c.hub.broadcast <- message
    }
}

func (c *Client) writePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()
    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                // Hub 关闭了 send channel
                c.conn.WriteMessage(websocket.CloseMessage, nil)
                return
            }
            if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
                return
            }

        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
```

**为什么要分 readPump 和 writePump？**

gorilla/websocket 的 `Conn` 不是并发安全的：
- 同一时刻只能有一个 goroutine 调用写方法
- 同一时刻只能有一个 goroutine 调用读方法
- 但**一个读 + 一个写可以并发**

因此标准做法是每个连接恰好两个 goroutine，一个专门读，一个专门写。
writePump 从 `send` channel 接收消息，保证写操作的串行化。

# 5 心跳机制（Ping/Pong）

WebSocket 协议内建了 Ping/Pong 帧，用于检测连接是否存活：

```
Server ──Ping──→ Client
Server ←──Pong── Client （自动回复，无需应用层处理）
```

**为什么需要心跳？**

1. **检测死连接**：客户端网络断开（如断网、进隧道），TCP 层不会立即感知。没有心跳，服务器的
   goroutine 和内存会被"僵尸连接"永远占用
2. **防止中间代理超时**：很多反向代理（Nginx）和负载均衡器会关闭长时间无数据的连接，
   默认 60s-120s。心跳可以保持连接活跃
3. **客户端检测延迟**：客户端可以通过 Pong 的返回时间估算网络延迟

**心跳参数设置建议**：

```go
const (
    // Pong 超时：在这个时间内没有收到 Pong，认为连接已死
    pongWait = 60 * time.Second

    // Ping 间隔：必须小于 pongWait，否则可能误判
    // 推荐设为 pongWait 的 80%-90%
    pingPeriod = (pongWait * 9) / 10

    // 写超时：每次写操作的最大等待时间
    writeWait = 10 * time.Second
)
```

# 6 JSON 消息处理

实际应用中，WebSocket 传输的通常是结构化数据。gorilla/websocket 提供了 JSON 辅助函数：

```go
type Message struct {
    Type    string `json:"type"`
    Content string `json:"content"`
    From    string `json:"from"`
    Time    int64  `json:"time"`
}

// 读取 JSON 消息
func readJSON(conn *websocket.Conn) (*Message, error) {
    var msg Message
    if err := conn.ReadJSON(&msg); err != nil {
        return nil, fmt.Errorf("read json: %w", err)
    }
    return &msg, nil
}

// 发送 JSON 消息
func writeJSON(conn *websocket.Conn, msg *Message) error {
    if err := conn.WriteJSON(msg); err != nil {
        return fmt.Errorf("write json: %w", err)
    }
    return nil
}
```

**注意**：`ReadJSON` / `WriteJSON` 内部会创建 `json.Encoder` / `json.Decoder`，
对于高频消息场景，可以考虑预分配缓冲区或使用更高效的 JSON 库（如 `json-iterator`、`sonic`）。

# 7 连接关闭与错误处理

WebSocket 关闭是有协议规范的，不能简单地断开 TCP：

## 7.1 关闭握手

```
发起方 ──Close Frame──→ 对方
发起方 ←──Close Frame── 对方     （确认关闭）
                         TCP 断开
```

关闭帧包含一个状态码和可选的原因文本：

| 状态码 | 含义 | 使用场景 |
|-------|------|---------|
| 1000 | 正常关闭 | 任务完成 |
| 1001 | Going Away | 服务器关闭/客户端离开页面 |
| 1002 | 协议错误 | 收到无效帧 |
| 1003 | 不支持的数据类型 | 期望文本收到二进制 |
| 1006 | 异常关闭 | 未发送 Close 帧就断开（不会出现在帧中） |
| 1008 | 策略违反 | 消息违反了服务器策略 |
| 1009 | 消息过大 | 超过服务器设定的限制 |
| 1011 | 服务器内部错误 | 类似 HTTP 500 |

## 7.2 优雅关闭

```go
func gracefulClose(conn *websocket.Conn) {
    // 发送关闭帧
    deadline := time.Now().Add(writeWait)
    msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye")
    conn.WriteControl(websocket.CloseMessage, msg, deadline)

    // 等待对方的关闭帧（或超时）
    time.Sleep(time.Second)
    conn.Close()
}
```

## 7.3 错误分类处理

```go
func handleError(err error) {
    if websocket.IsCloseError(err,
        websocket.CloseNormalClosure,
        websocket.CloseGoingAway) {
        // 正常关闭，不需要报错
        return
    }
    if websocket.IsUnexpectedCloseError(err) {
        // 异常关闭，需要记录日志
        log.Printf("unexpected close: %v", err)
        return
    }
    // 其他错误
    log.Printf("websocket error: %v", err)
}
```

# 8 安全性

## 8.1 Origin 校验（防止 CSRF / 跨站 WebSocket 劫持）

浏览器在建立 WebSocket 连接时会发送 `Origin` 头，但 WebSocket 不受同源策略约束。
如果不校验 Origin，恶意网站可以在用户不知情的情况下，用用户的 Cookie 连接你的 WebSocket 服务器。

```go
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        origin := r.Header.Get("Origin")
        // 只允许来自受信任的域
        return origin == "https://yourdomain.com"
    },
}
```

**生产环境必须设置 `CheckOrigin`，默认的 gorilla/websocket 会拒绝跨域请求。**
设置 `return true` 只适用于开发阶段。

## 8.2 认证

WebSocket 握手是标准 HTTP 请求，可以携带 Cookie 和自定义 Header：

```go
// 方式一：通过 URL 参数传递 token（不推荐用于敏感 token）
// ws://server/ws?token=xxx

// 方式二：通过 Cookie（推荐，浏览器自动携带）
func wsHandler(w http.ResponseWriter, r *http.Request) {
    cookie, err := r.Cookie("session_id")
    if err != nil || !validateSession(cookie.Value) {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    conn, err := upgrader.Upgrade(w, r, nil)
    // ...
}

// 方式三：通过自定义 Header（非浏览器客户端）
dialer := websocket.Dialer{}
header := http.Header{}
header.Set("Authorization", "Bearer "+token)
conn, _, err := dialer.Dial("ws://server/ws", header)
```

## 8.3 消息限制

```go
// 限制单条消息大小，防止内存耗尽攻击
conn.SetReadLimit(4096) // 4KB

// 配合 WriteBufferSize 使用
var upgrader = websocket.Upgrader{
    ReadBufferSize:  4096,
    WriteBufferSize: 4096,
}
```

## 8.4 TLS（WSS）

生产环境应该使用 `wss://`（WebSocket over TLS），可以通过 Nginx/Caddy 反向代理实现，
也可以直接在 Go 中使用 `http.ListenAndServeTLS`。

# 9 与 Nginx 配合部署

WebSocket 需要特殊的 Nginx 配置来支持协议升级：

```nginx
upstream ws_backend {
    server 127.0.0.1:8080;
}

server {
    listen 443 ssl;
    server_name yourdomain.com;

    location /ws {
        proxy_pass http://ws_backend;
        proxy_http_version 1.1;

        # 关键：传递 Upgrade 头
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # 传递客户端真实 IP
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

        # 超时设置：必须大于心跳间隔
        # 默认 60s，如果心跳间隔是 54s，这里应该设更大
        proxy_read_timeout 120s;
        proxy_send_timeout 120s;
    }
}
```

**常见坑**：
- 忘记设置 `proxy_http_version 1.1`，HTTP/1.0 不支持 Upgrade
- `proxy_read_timeout` 太短，导致心跳还没到就被 Nginx 断开
- 忘记传递 `Upgrade` 和 `Connection` 头

# 10 水平扩展

单机 WebSocket 服务的连接数有限（通常 10 万级别），水平扩展需要解决**跨节点消息路由**问题。

## 10.1 问题

```
Client A ──连接──→ Node 1
Client B ──连接──→ Node 2

A 发消息给 B → Node 1 收到，但 B 在 Node 2 上，怎么办？
```

## 10.2 解决方案：消息总线

```
              ┌─────────────────────────────┐
              │     Redis Pub/Sub / Kafka   │
              └──────┬──────────┬───────────┘
                     │          │
              ┌──────▼──┐  ┌───▼──────┐
              │  Node 1 │  │  Node 2  │
              │ Hub + WS│  │ Hub + WS │
              └─────────┘  └──────────┘
```

每个节点：
1. 将收到的消息发布到 Redis Pub/Sub
2. 订阅 Redis Pub/Sub，将收到的消息广播给本节点的客户端

```go
func (h *Hub) RunWithRedis(ctx context.Context, rdb *redis.Client, channel string) {
    // 订阅 Redis
    sub := rdb.Subscribe(ctx, channel)
    ch := sub.Channel()

    for {
        select {
        case client := <-h.register:
            h.clients[client] = struct{}{}

        case client := <-h.unregister:
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }

        case message := <-h.broadcast:
            // 发布到 Redis，而不是直接广播
            rdb.Publish(ctx, channel, message)

        case msg := <-ch:
            // 从 Redis 收到消息，广播给本节点的客户端
            for client := range h.clients {
                select {
                case client.send <- []byte(msg.Payload):
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }

        case <-ctx.Done():
            return
        }
    }
}
```

# 11 性能优化

## 11.1 缓冲区复用

每个 WebSocket 连接默认需要独立的读/写缓冲区，大量连接时内存开销显著：

```
10 万连接 × (4KB 读 + 4KB 写) = 800MB 仅缓冲区
```

gorilla/websocket 支持通过 `BufferPool` 复用缓冲区：

```go
var bufferPool = &sync.Pool{
    New: func() interface{} {
        buf := make([]byte, 0, 1024)
        return bytes.NewBuffer(buf)
    },
}

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    WriteBufferPool: bufferPool,
}
```

详细的性能对比见 [performance/](performance/) 目录。

## 11.2 消息压缩

WebSocket 支持 `permessage-deflate` 扩展进行消息压缩：

```go
// gorilla/websocket
var upgrader = websocket.Upgrader{
    EnableCompression: true,
}

// nhooyr.io/websocket
conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
    CompressionMode: websocket.CompressionContextTakeover,
})
```

**注意**：压缩会增加 CPU 开销，适用于大文本消息（如 JSON），不适用于已压缩的二进制数据（如图片）。
生产环境应进行 benchmark 评估压缩的收益。

## 11.3 连接数优化

Go 处理 WebSocket 连接的主要瓶颈是每个连接占两个 goroutine（readPump + writePump）。
对于百万级连接场景，可以考虑：

1. **epoll/kqueue 优化**：使用 `gobwas/ws` + `netpoll`，只在有数据时创建 goroutine
2. **减少 goroutine**：空闲连接不需要持续读取，可以用 epoll 监听可读事件
3. **连接数限制**：限制单机最大连接数，超出后拒绝或转发

```go
var (
    connCount int64
    maxConns  int64 = 50000
)

func wsHandler(w http.ResponseWriter, r *http.Request) {
    if atomic.LoadInt64(&connCount) >= maxConns {
        http.Error(w, "too many connections", http.StatusServiceUnavailable)
        return
    }
    atomic.AddInt64(&connCount, 1)
    defer atomic.AddInt64(&connCount, -1)

    conn, err := upgrader.Upgrade(w, r, nil)
    // ...
}
```

# 12 测试

## 12.1 使用 httptest

```go
func TestEchoHandler(t *testing.T) {
    // 创建测试服务器
    server := httptest.NewServer(http.HandlerFunc(echoHandler))
    defer server.Close()

    // 将 http:// 转换为 ws://
    wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

    // 连接
    conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
    if err != nil {
        t.Fatalf("dial: %v", err)
    }
    defer conn.Close()

    // 发送
    want := "hello"
    if err := conn.WriteMessage(websocket.TextMessage, []byte(want)); err != nil {
        t.Fatalf("write: %v", err)
    }

    // 读取
    _, got, err := conn.ReadMessage()
    if err != nil {
        t.Fatalf("read: %v", err)
    }
    if string(got) != want {
        t.Errorf("got %q, want %q", got, want)
    }
}
```

## 12.2 并发测试

```go
func TestConcurrentConnections(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(echoHandler))
    defer server.Close()

    wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
    const numClients = 100

    var wg sync.WaitGroup
    wg.Add(numClients)

    for i := 0; i < numClients; i++ {
        go func(id int) {
            defer wg.Done()
            conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
            if err != nil {
                t.Errorf("client %d: dial: %v", id, err)
                return
            }
            defer conn.Close()

            msg := fmt.Sprintf("hello from client %d", id)
            if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
                t.Errorf("client %d: write: %v", id, err)
                return
            }

            _, got, err := conn.ReadMessage()
            if err != nil {
                t.Errorf("client %d: read: %v", id, err)
                return
            }
            if string(got) != msg {
                t.Errorf("client %d: got %q, want %q", id, got, msg)
            }
        }(i)
    }
    wg.Wait()
}
```

# 13 常见问题排查

| 问题 | 可能原因 | 排查方法 |
|------|---------|---------|
| 连接建立后立即断开 | Nginx 超时、CheckOrigin 拒绝 | 检查 Nginx 日志和 CheckOrigin |
| 消息接收延迟高 | 写缓冲区满、对端处理慢 | 监控 send channel 长度 |
| 内存持续增长 | 连接泄漏（没有正确注销） | pprof 查看 goroutine 数量 |
| panic: concurrent write | 多 goroutine 写同一连接 | 确保只有 writePump 写连接 |
| 连接突然断开无报错 | 中间代理超时 | 检查心跳是否正常工作 |
| CPU 占用高 | 消息压缩开销 | 关闭压缩或降低压缩级别 |

# 14 总结

| 要点 | 建议 |
|------|------|
| 库选择 | 新项目用 `nhooyr.io/websocket`，存量项目 `gorilla/websocket` 也可 |
| 架构模式 | Hub + Client (readPump/writePump) |
| 并发安全 | gorilla: 一读一写两个 goroutine；nhooyr: 内置安全 |
| 心跳 | Ping 间隔 < Pong 超时，通常 54s / 60s |
| 安全 | CheckOrigin 必须配置，生产用 WSS |
| 部署 | Nginx proxy_http_version 1.1 + Upgrade 头 |
| 扩展 | Redis Pub/Sub 做跨节点消息路由 |
| 性能 | 缓冲区池化，高连接数考虑 epoll |
| 测试 | httptest 创建测试服务器 |

**推荐阅读**：
- [RFC 6455 - The WebSocket Protocol](https://tools.ietf.org/html/rfc6455)
- [gorilla/websocket 官方聊天室示例](https://github.com/gorilla/websocket/tree/main/examples/chat)
- [性能对比实验](performance/)
- [常见陷阱](trap/)