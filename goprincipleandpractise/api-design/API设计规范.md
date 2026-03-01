# API 设计规范

> 本文系统讲解 RESTful/gRPC API 设计的核心原则与实践，涵盖路由设计、版本管理、错误码体系、请求校验、认证授权等内容。所有示例均有可运行代码，见同目录下的 `restful/`、`grpc/`、`performance/`、`trap/` 子目录。

---

## 目录

1. [设计哲学与核心原则](#1-设计哲学与核心原则)
2. [RESTful API 设计](#2-restful-api-设计)
3. [路由设计](#3-路由设计)
4. [版本管理](#4-版本管理)
5. [错误码体系](#5-错误码体系)
6. [请求校验框架](#6-请求校验框架)
7. [gRPC API 设计](#7-grpc-api-设计)
8. [认证与授权](#8-认证与授权)
9. [最佳实践与检查清单](#9-最佳实践与检查清单)

---

## 1. 设计哲学与核心原则

### 1.1 契约优先（Contract First）

在写一行代码之前，先定义好 API 契约：

- **请求格式**: URL、方法、头部、请求体
- **响应格式**: 状态码、响应数据、字段命名
- **错误模型**: 错误码、错误消息、字段级错误

契约是 API 消费者和提供者之间的协议，一旦发布就应保持稳定。

### 1.2 向后兼容

**只做加法，不做减法。** 已发布的字段、端点、错误码不应移除或改变语义。

### 1.3 稳定语义

状态码、字段命名、错误码的含义必须一致且可预测：

- `404` 永远意味着"资源不存在"
- `401` 永远意味着"未认证"
- `email` 字段永远是字符串类型

### 1.4 安全是一等公民

安全不是事后添加的功能：

- 认证/授权在设计阶段就要确定
- 输入校验是强制性的
- 敏感数据不应出现在 URL 或错误响应中
- 限流是必须的

### 1.5 API 评分卡（14 项）

| # | 检查项 | 说明 |
|---|--------|------|
| 1 | 资源命名 | 复数名词、小写、kebab-case |
| 2 | HTTP 方法 | 正确使用 GET/POST/PUT/PATCH/DELETE |
| 3 | 状态码 | 语义准确（201 Created、204 No Content 等） |
| 4 | 错误格式 | 统一信封、包含错误码和消息 |
| 5 | 字段级错误 | 校验失败时返回具体字段和原因 |
| 6 | 分页 | 列表接口支持分页参数 |
| 7 | 版本管理 | 明确的版本策略 |
| 8 | 认证 | 使用标准认证机制（Bearer Token、API Key） |
| 9 | 授权 | 正确区分 401 和 403 |
| 10 | 限流 | 配置限流并返回 Retry-After |
| 11 | 幂等性 | POST 创建支持 Idempotency-Key |
| 12 | CORS | 跨域配置正确 |
| 13 | 输入校验 | 系统边界处校验所有输入 |
| 14 | 错误不泄漏 | 内部错误不暴露给客户端 |

---

## 2. RESTful API 设计

### 2.1 URL 与资源规则

```
GET    /api/v1/users              → 列出用户
GET    /api/v1/users/{id}         → 获取用户详情
POST   /api/v1/users              → 创建用户
PUT    /api/v1/users/{id}         → 全量更新用户
PATCH  /api/v1/users/{id}         → 部分更新用户
DELETE /api/v1/users/{id}         → 删除用户

GET    /api/v1/users/{id}/orders  → 获取用户的订单（嵌套资源）
POST   /api/v1/orders/{id}/cancel → 显式动作端点（例外）
```

**命名规则：**
- 资源名使用复数名词（`/users`，不是 `/user`）
- URL 使用小写、kebab-case
- 不在 URL 中使用动词（见 [`trap/verb-url/`](trap/verb-url/main.go)）
- 嵌套层级不超过 2 层

### 2.2 HTTP 方法与状态码映射

| 方法 | 语义 | 幂等 | 安全 | 典型状态码 |
|------|------|------|------|-----------|
| GET | 读取 | ✅ | ✅ | 200 |
| POST | 创建/动作 | ❌ | ❌ | 201 + Location |
| PUT | 全量替换 | ✅ | ❌ | 200 |
| PATCH | 部分更新 | ❌ | ❌ | 200 |
| DELETE | 删除 | ✅ | ❌ | 204 |

**常用状态码：**

| 状态码 | 含义 | 使用场景 |
|--------|------|---------|
| 200 OK | 成功 | GET、PUT、PATCH |
| 201 Created | 已创建 | POST 创建资源 |
| 204 No Content | 无内容 | DELETE 成功 |
| 400 Bad Request | 请求格式错误 | JSON 解析失败 |
| 401 Unauthorized | 未认证 | 缺少或无效的 Token |
| 403 Forbidden | 未授权 | Token 有效但无权限 |
| 404 Not Found | 资源不存在 | 查询/修改不存在的资源 |
| 409 Conflict | 冲突 | 唯一键冲突 |
| 412 Precondition Failed | 前置条件失败 | ETag/If-Match 不匹配 |
| 422 Unprocessable Entity | 校验失败 | 字段校验不通过 |
| 429 Too Many Requests | 限流 | 请求频率超限 |
| 500 Internal Server Error | 服务器错误 | 未预期的内部错误 |

### 2.3 请求响应信封

**成功响应：**

```json
{
  "data": {
    "id": "usr_000001",
    "name": "Alice",
    "email": "alice@example.com"
  }
}
```

**带分页的成功响应：**

```json
{
  "data": [...],
  "meta": {
    "total": 100,
    "page": 1,
    "limit": 20
  }
}
```

**错误响应：**

```json
{
  "error": {
    "code": "validation_failed",
    "message": "request validation failed",
    "fields": {
      "email": "must be a valid email address",
      "name": "must be at least 2 characters"
    }
  }
}
```

> 实现见 [`restful/response.go`](restful/response.go)

### 2.4 分页、过滤与排序

```
GET /api/v1/users?page=2&limit=20&sort=created_at&order=desc&status=active
```

参数约定：
- `page` / `offset`: 分页位置
- `limit`: 每页数量（设置上限，如 100）
- `sort`: 排序字段
- `order`: `asc` 或 `desc`
- 过滤参数使用字段名（`status=active`）

### 2.5 幂等性

POST 请求天然非幂等，但通过 `Idempotency-Key` 头部可以实现幂等：

```
POST /api/v1/orders
Idempotency-Key: order-abc-123
```

服务端缓存 `(Idempotency-Key → Response)`，相同 Key 的重复请求直接返回缓存响应。

> 实现见 [`restful/handler.go`](restful/handler.go) CreateUser 方法
> 反模式见 [`trap/missing-idempotency/`](trap/missing-idempotency/main.go)

---

## 3. 路由设计

### 3.1 Go 1.22+ 路由语法

Go 1.22 引入了增强的 `http.ServeMux`，支持方法匹配和路径参数：

```go
mux := http.NewServeMux()

// 方法 + 路径模式
mux.Handle("GET /api/v1/users", listHandler)
mux.Handle("POST /api/v1/users", createHandler)
mux.Handle("GET /api/v1/users/{id}", getHandler)
mux.Handle("PUT /api/v1/users/{id}", updateHandler)
mux.Handle("DELETE /api/v1/users/{id}", deleteHandler)

// 提取路径参数
func getHandler(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    // ...
}
```

### 3.2 路由层次

```
/                           → 根（通常重定向到 API 文档）
/healthz                    → 健康检查（不需要认证）
/api/v1/                    → API 版本 1
/api/v1/users               → 用户资源
/api/v1/users/{id}          → 特定用户
/api/v1/users/{id}/orders   → 用户的订单（嵌套资源）
/api/v2/                    → API 版本 2（与 v1 共存）
```

### 3.3 中间件链

中间件的执行顺序至关重要：

```
请求 → Recovery → CORS → Logging → RateLimit → Auth → Handler
响应 ← Recovery ← CORS ← Logging ← RateLimit ← Auth ← Handler
```

```go
// Chain 组合中间件，从左到右执行
public := Chain(Recovery, CORS, Logging, limiter.Middleware)
protected := Chain(Recovery, CORS, Logging, limiter.Middleware, Auth(nil))
```

**顺序原则：**
1. **Recovery** 必须最外层 — 捕获所有 panic
2. **CORS** 在认证之前 — OPTIONS 预检不应被认证拦截
3. **Logging** 在业务逻辑之前 — 记录所有请求（包括被拒绝的）
4. **RateLimit** 在认证之前 — 防止暴力破解
5. **Auth** 最靠近 Handler — 只保护需要认证的路由

> 实现见 [`restful/middleware.go`](restful/middleware.go) 和 [`restful/server.go`](restful/server.go)

---

## 4. 版本管理

### 4.1 版本策略对比

| 策略 | 示例 | 优点 | 缺点 |
|------|------|------|------|
| URL 路径 | `/api/v1/users` | 直观、易缓存 | URL 变化大 |
| Header | `Accept-Version: v1` | URL 不变 | 不直观、难调试 |
| 媒体类型 | `Accept: application/vnd.myapi.v1+json` | 标准化 | 复杂 |

**推荐使用 URL 路径版本**，因为：
- 最直观，团队成员一眼就能看出版本
- 容易在负载均衡器/网关层路由
- 便于缓存（不同 URL = 不同缓存键）

### 4.2 Breaking vs Non-Breaking 变更

| 变更类型 | 是否 Breaking | 处理方式 |
|----------|:------------:|---------|
| 新增可选字段 | ❌ | 直接添加 |
| 新增端点 | ❌ | 直接添加 |
| 新增可选查询参数 | ❌ | 直接添加 |
| 移除字段 | ✅ | 新版本 |
| 重命名字段 | ✅ | 新版本 |
| 修改字段类型 | ✅ | 新版本 |
| 修改状态码语义 | ✅ | 新版本 |
| 移除端点 | ✅ | 新版本 + 弃用期 |

### 4.3 弃用策略（Sunset）

```
HTTP/1.1 200 OK
Sunset: Sat, 01 Mar 2025 00:00:00 GMT
Deprecation: true
Link: </api/v2/users>; rel="successor-version"
```

弃用流程：
1. 发布新版本 (v2)
2. 在旧版本 (v1) 响应中添加 `Sunset` 和 `Deprecation` 头部
3. 在文档中标注弃用时间表
4. 监控旧版本流量
5. 流量降至 0 或到达 Sunset 日期后下线

---

## 5. 错误码体系

### 5.1 标准错误码

| 错误码 | HTTP 状态码 | gRPC Code | 含义 |
|--------|:----------:|:---------:|------|
| `invalid_json` | 400 | InvalidArgument | 请求体 JSON 格式错误 |
| `validation_failed` | 422 | InvalidArgument | 字段校验失败 |
| `unauthorized` | 401 | Unauthenticated | 未认证 |
| `forbidden` | 403 | PermissionDenied | 已认证但无权限 |
| `not_found` | 404 | NotFound | 资源不存在 |
| `conflict` | 409 | AlreadyExists | 资源冲突（如唯一键） |
| `precondition_failed` | 412 | FailedPrecondition | 前置条件不满足 |
| `rate_limited` | 429 | ResourceExhausted | 请求频率超限 |
| `internal_error` | 500 | Internal | 内部错误 |

### 5.2 AppError 实现

```go
type AppError struct {
    Code     ErrCode `json:"code"`
    Message  string  `json:"message"`
    Detail   string  `json:"detail,omitempty"`
    internal error   // 不序列化，仅用于日志
}
```

关键设计：
- `Code` 是机器可读的字符串码（不是数字）
- `Message` 是面向用户的友好消息
- `Detail` 是可选的额外信息
- `internal` 不序列化，仅存在于服务端日志

> 实现见 [`restful/errors.go`](restful/errors.go)

### 5.3 HTTP/gRPC 双向映射

```go
func (c ErrCode) HTTPStatusCode() int {
    switch c {
    case ErrNotFound:      return 404
    case ErrUnauthorized:  return 401
    // ...
    }
}

func GRPCCodeToHTTP(code codes.Code) int {
    switch code {
    case codes.NotFound:        return 404
    case codes.Unauthenticated: return 401
    // ...
    }
}
```

> 反模式见 [`trap/inconsistent-error/`](trap/inconsistent-error/main.go)
> 反模式见 [`trap/leak-internal-error/`](trap/leak-internal-error/main.go)

### 5.4 可观测性

错误应该可以被监控和告警：

```go
// 结构化日志
log.Printf("[ERROR] code=%s method=%s path=%s internal=%v request_id=%s",
    appErr.Code, r.Method, r.URL.Path, appErr.internal, requestID)
```

客户端通过 `request_id` 关联请求与服务端日志。

---

## 6. 请求校验框架

### 6.1 系统边界校验原则

> 在系统边界校验所有输入，信任内部代码。

- **系统边界**: HTTP handler、gRPC 服务入口、消息队列消费者
- **内部代码**: 服务层、领域层——由调用方保证参数合法性

### 6.2 Struct Tag + 反射实现

```go
type CreateUserRequest struct {
    Name  string `json:"name"  validate:"required,min=2,max=50"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age"   validate:"min=0,max=150"`
}

errs := Validate(req) // map[string]string
if len(errs) > 0 {
    WriteValidationError(w, errs)
    return
}
```

支持的规则：

| 规则 | 示例 | 说明 |
|------|------|------|
| `required` | `validate:"required"` | 非零值 |
| `email` | `validate:"email"` | 合法邮箱格式 |
| `min=N` | `validate:"min=2"` | 字符串最小长度/数字最小值 |
| `max=N` | `validate:"max=50"` | 字符串最大长度/数字最大值 |

> 实现见 [`restful/validator.go`](restful/validator.go)

### 6.3 校验 vs 业务规则

| 类别 | 示例 | 处理位置 |
|------|------|---------|
| 格式校验 | "email 格式不合法" | Handler 层（校验框架） |
| 业务规则 | "该邮箱已被注册" | Service 层（业务逻辑） |

格式校验返回 `422 Unprocessable Entity`，业务规则返回 `409 Conflict`。

### 6.4 性能考量

反射校验相比手动校验有额外开销，但在 Web 应用场景下完全可接受：

```bash
go test -bench=. -benchmem ./performance/validation/
```

> 基准测试见 [`performance/validation/`](performance/validation/)

---

## 7. gRPC API 设计

### 7.1 Proto 设计原则

虽然本示例使用手写 Go struct（避免 protoc 工具链依赖），但 proto 设计原则同样适用：

- **消息类型用单数**: `User`，不是 `Users`
- **请求/响应成对**: `CreateUserRequest` → `User`
- **列表接口返回专用响应**: `ListUsersResponse` 包含分页信息
- **ID 字段使用 string**: 允许不同 ID 生成策略

### 7.2 gRPC Status Code 映射

gRPC 使用 `google.golang.org/grpc/status` 包返回结构化错误：

```go
// 参数校验失败
return nil, status.Error(codes.InvalidArgument, "name is required")

// 资源不存在
return nil, status.Errorf(codes.NotFound, "user %q not found", id)

// 唯一键冲突
return nil, status.Errorf(codes.AlreadyExists, "email already registered")
```

| gRPC Code | HTTP 等价 | 含义 |
|-----------|:---------:|------|
| OK | 200 | 成功 |
| InvalidArgument | 400 | 参数错误 |
| Unauthenticated | 401 | 未认证 |
| PermissionDenied | 403 | 未授权 |
| NotFound | 404 | 不存在 |
| AlreadyExists | 409 | 已存在 |
| FailedPrecondition | 412 | 前置条件失败 |
| ResourceExhausted | 429 | 资源耗尽 |
| Internal | 500 | 内部错误 |
| Unavailable | 503 | 不可用 |

> 实现见 [`grpc/service.go`](grpc/service.go) 和 [`grpc/server.go`](grpc/server.go)

### 7.3 拦截器（Interceptor）

gRPC 拦截器等价于 HTTP 中间件：

```go
grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        RecoveryInterceptor,   // 捕获 panic
        LoggingInterceptor,    // 记录请求日志
        AuthInterceptor(token), // 验证 token
    ),
)
```

> 实现见 [`grpc/server.go`](grpc/server.go)

### 7.4 gRPC vs REST 选型

| 维度 | REST | gRPC |
|------|------|------|
| 协议 | HTTP/1.1 (JSON) | HTTP/2 (Protobuf) |
| 性能 | 较低 | 较高（二进制、流式） |
| 浏览器支持 | 原生 | 需要 gRPC-Web |
| 可调试性 | curl/浏览器 | 需要专用工具 |
| 流式传输 | SSE/WebSocket | 原生双向流 |
| 适用场景 | 公开 API、前端 | 微服务内部通信 |

**选型建议：**
- **公开 API / 面向前端** → REST
- **微服务间内部通信** → gRPC
- **需要流式传输** → gRPC
- **需要浏览器直接调用** → REST

> 序列化性能对比见 [`performance/json-vs-protobuf/`](performance/json-vs-protobuf/)

---

## 8. 认证与授权

### 8.1 Bearer Token 认证

```
Authorization: Bearer <token>
```

认证中间件提取 Token 并验证：

```go
func Auth(tokenValidator func(token string) bool) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            auth := r.Header.Get("Authorization")
            if !strings.HasPrefix(auth, "Bearer ") {
                WriteError(w, NewAppError(ErrUnauthorized, "missing Authorization", nil))
                return
            }
            token := strings.TrimPrefix(auth, "Bearer ")
            if !tokenValidator(token) {
                WriteError(w, NewAppError(ErrUnauthorized, "invalid token", nil))
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

> 实现见 [`restful/middleware.go`](restful/middleware.go)

### 8.2 401 vs 403

| 状态码 | 含义 | 场景 |
|--------|------|------|
| 401 Unauthorized | 未认证 | 没有 Token、Token 过期、Token 无效 |
| 403 Forbidden | 已认证但无权限 | 普通用户访问管理员接口 |

**关键区别**: 401 可以通过重新登录解决，403 表示即使重新登录也无权访问。

### 8.3 限流与 Retry-After

```go
if len(valid) >= rl.limit {
    w.Header().Set("Retry-After", "60")
    WriteError(w, NewAppError(ErrRateLimited, "rate limit exceeded", nil))
    return
}
```

限流策略：
- **固定窗口**: 简单，但窗口边界有流量突增
- **滑动窗口**: 更平滑，实现稍复杂
- **令牌桶**: `golang.org/x/time/rate`，生产推荐
- **分布式限流**: Redis + Lua 脚本

> 实现见 [`restful/middleware.go`](restful/middleware.go)

---

## 9. 最佳实践与检查清单

### 9.1 API 评分卡（14 项）

在发布每个 API 端点前，逐一检查：

- [ ] **资源命名**: 复数名词、小写、无动词
- [ ] **HTTP 方法**: 语义正确（GET 读取、POST 创建）
- [ ] **状态码**: 精确匹配（201 创建、204 删除、422 校验失败）
- [ ] **错误格式**: 使用统一错误信封
- [ ] **字段级错误**: 校验失败时返回具体字段
- [ ] **分页**: 列表接口支持 page/limit
- [ ] **版本**: URL 路径包含版本号
- [ ] **认证**: 受保护端点要求 Bearer Token
- [ ] **授权**: 正确区分 401/403
- [ ] **限流**: 配置限流并返回 429 + Retry-After
- [ ] **幂等性**: POST 创建支持 Idempotency-Key
- [ ] **CORS**: 跨域头部正确配置
- [ ] **输入校验**: 所有输入在系统边界校验
- [ ] **错误不泄漏**: 内部错误细节不暴露给客户端

### 9.2 陷阱速查表

| 陷阱 | 示例代码 | 说明 |
|------|---------|------|
| 动词 URL | [`trap/verb-url/`](trap/verb-url/main.go) | URL 中不应包含动词 |
| 错误格式不一致 | [`trap/inconsistent-error/`](trap/inconsistent-error/main.go) | 所有接口必须使用相同错误信封 |
| 缺少幂等性 | [`trap/missing-idempotency/`](trap/missing-idempotency/main.go) | POST 创建应支持幂等重试 |
| 内部错误泄漏 | [`trap/leak-internal-error/`](trap/leak-internal-error/main.go) | 不要暴露堆栈/SQL/路径 |

### 9.3 性能考量

| 场景 | 基准测试 | 建议 |
|------|---------|------|
| 序列化格式选择 | [`performance/json-vs-protobuf/`](performance/json-vs-protobuf/) | 内部通信用二进制，外部 API 用 JSON |
| 校验框架选择 | [`performance/validation/`](performance/validation/) | Web 应用使用反射校验即可 |

运行基准测试：

```bash
# 序列化对比
go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem \
  ./goprincipleandpractise/api-design/performance/json-vs-protobuf/

# 校验对比
go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem \
  ./goprincipleandpractise/api-design/performance/validation/
```

### 9.4 运行所有示例

```bash
# 编译检查
go build ./goprincipleandpractise/api-design/...

# 运行测试
go test -race ./goprincipleandpractise/api-design/...

# 运行陷阱示例
go run ./goprincipleandpractise/api-design/trap/verb-url/
go run ./goprincipleandpractise/api-design/trap/inconsistent-error/
go run ./goprincipleandpractise/api-design/trap/missing-idempotency/
go run ./goprincipleandpractise/api-design/trap/leak-internal-error/
```