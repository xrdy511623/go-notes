# Go 安全编码实践

> 覆盖 Go 安全编码七大领域，每个章节配有可运行的反例（trap/）和性能基准（performance/）。

## 目录

1. [随机数安全](#1-随机数安全)
2. [SQL 注入防御](#2-sql-注入防御)
3. [敏感数据处理](#3-敏感数据处理)
4. [密钥与配置管理](#4-密钥与配置管理)
5. [TLS 配置](#5-tls-配置)
6. [密码学原语](#6-密码学原语)
7. [gosec 静态安全扫描](#7-gosec-静态安全扫描)

---

## 1. 随机数安全

### math/rand vs crypto/rand

| 特性 | math/rand | crypto/rand |
|------|-----------|-------------|
| 算法 | PRNG (PCG/ChaCha8) | CSPRNG (OS 熵源) |
| 可预测性 | 已知种子可重现 | 不可预测 |
| 性能 | ~30 ns/16B | ~90 ns/16B |
| 适用场景 | 模拟、测试、游戏 | token、密钥、nonce、salt |

### CSPRNG 原理

Go 的 `crypto/rand.Reader` 在不同 OS 上的实现：
- **Linux**: `getrandom(2)` 系统调用（内核 3.17+），回退到 `/dev/urandom`
- **macOS**: `getentropy(2)`
- **Windows**: `RtlGenRandom`

这些 OS 原语收集硬件中断、磁盘 I/O 时序等熵源，经 CSPRNG 扩展后提供给用户态。

### Token 生成最佳实践

```go
import (
    "crypto/rand"
    "encoding/base64"
    "encoding/hex"
)

// 方式 1：hex 编码（32 字节 → 64 字符）
func generateToken() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}

// 方式 2：base64 URL 安全编码（32 字节 → 43 字符）
func generateTokenBase64() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
}
```

> **反例**: [trap/math-rand-for-token/](trap/math-rand-for-token/) — math/rand 生成 token 可被预测
>
> **性能对比**: [performance/rand-crypto-vs-math/](performance/rand-crypto-vs-math/) — crypto/rand 仅慢 1.5-3x，绝对开销 <100ns

---

## 2. SQL 注入防御

### 注入机制

SQL 注入的根本原因是**代码和数据混合**。使用字符串拼接时，用户输入成为 SQL 语句的一部分，可以改变语句语义：

```go
// 危险：用户输入直接拼接到 SQL
query := fmt.Sprintf("SELECT * FROM users WHERE name = '%s'", userInput)

// 攻击者输入: ' OR '1'='1
// 实际执行: SELECT * FROM users WHERE name = '' OR '1'='1'  → 返回所有用户
```

### 参数化查询

参数化查询将 SQL 结构和数据分离，数据库驱动负责正确处理参数：

```go
// 安全：参数化查询
db.Query("SELECT * FROM users WHERE name = ?", userInput)

// 参数化的原理：
// 1. SQL 语句先编译为执行计划（结构固定）
// 2. 参数作为数据传入，不参与 SQL 解析
// 3. 无论参数内容如何，都不会改变 SQL 结构
```

### 特殊场景安全写法

**LIKE 子句**：
```go
// 错误
query := fmt.Sprintf("SELECT * FROM users WHERE name LIKE '%%%s%%'", input)

// 正确（SQLite / MySQL）
db.Query("SELECT * FROM users WHERE name LIKE '%' || ? || '%'", input)

// 正确（PostgreSQL）
db.Query("SELECT * FROM users WHERE name LIKE '%' || $1 || '%'", input)
```

**IN 子句**：
```go
// 动态生成占位符
ids := []int{1, 2, 3}
placeholders := strings.Repeat("?,", len(ids))
placeholders = placeholders[:len(placeholders)-1] // 去掉末尾逗号

args := make([]any, len(ids))
for i, id := range ids {
    args[i] = id
}

query := fmt.Sprintf("SELECT * FROM users WHERE id IN (%s)", placeholders)
db.Query(query, args...)
```

**ORDER BY**：
```go
// ORDER BY 不能参数化（它是 SQL 结构，不是数据）
// 必须用白名单验证
allowedColumns := map[string]bool{"name": true, "created_at": true, "id": true}
if !allowedColumns[sortColumn] {
    sortColumn = "id" // 默认值
}
query := fmt.Sprintf("SELECT * FROM users ORDER BY %s", sortColumn)
```

> **反例**: [trap/sql-injection-sprintf/](trap/sql-injection-sprintf/) — SQLite 实机演示注入攻击

---

## 3. 敏感数据处理

### 日志脱敏

```go
// 问题：%+v 打印所有字段
log.Printf("用户: %+v", user)
// 输出: {Name:张三 Email:zhang@example.com Password:secret Token:eyJ...}

// 解决：实现 fmt.Stringer 接口
type User struct {
    Name     string
    Email    string
    Password string `json:"-"`
    Token    string `json:"-"`
}

func (u User) String() string {
    return fmt.Sprintf("User{Name: %s, Email: %s}", u.Name, maskEmail(u.Email))
}
```

### json:"-" 标签

```go
type User struct {
    Name     string `json:"name"`
    Password string `json:"-"`     // json.Marshal 时完全排除
    Token    string `json:"-"`     // 不会出现在 API 响应中
    Internal string `json:"-"`     // 内部字段不暴露
}
```

### 内存中清零敏感数据

```go
func processPassword(password []byte) error {
    defer func() {
        // 用完后立即清零，减少内存中的暴露窗口
        for i := range password {
            password[i] = 0
        }
    }()
    // ... 处理密码
    return nil
}
```

注意：Go 的 GC 可能在清零前复制数据，这不是完美方案。对安全性要求极高的场景（如密钥管理），考虑使用 `mlock` 或专用库。

### 错误信息安全

```go
// 错误：暴露内部细节
return fmt.Errorf("连接 %s:%d 失败，用户 %s: %w", host, port, user, err)

// 正确：对外通用错误，对内记日志
log.Error("db连接失败", "host", host, "port", port, "error", err)
return fmt.Errorf("服务暂时不可用: %w", ErrServiceUnavailable)
```

> **反例**: [trap/sensitive-data-in-log/](trap/sensitive-data-in-log/) — %+v 泄漏密码和 token

---

## 4. 密钥与配置管理

### os.LookupEnv fail-fast 模式

```go
func mustEnv(key string) string {
    value, ok := os.LookupEnv(key)
    if !ok {
        log.Fatalf("环境变量 %s 未设置", key)
    }
    if value == "" {
        log.Fatalf("环境变量 %s 为空", key)
    }
    return value
}

func main() {
    apiKey := mustEnv("API_KEY")
    dbURL  := mustEnv("DATABASE_URL")
    // 启动时就失败，而不是运行到一半才发现缺配置
}
```

### .env + .gitignore

```bash
# .env（不提交到 git）
API_KEY=sk-proj-abc123
DATABASE_URL=postgres://user:pass@localhost/db

# .gitignore
.env
.env.local
.env.*.local
```

### 密钥脱敏打印

```go
func maskSecret(s string) string {
    if len(s) <= 8 {
        return "****"
    }
    return s[:4] + "****" + s[len(s)-4:]
}

// 输出: sk-p****f456
```

### gosec G101

gosec G101 规则通过正则匹配变量名中的 `password`、`secret`、`token`、`key` 等关键词，检测疑似硬编码密钥。

```go
// G101 会标记
const apiKey = "sk-proj-abc123"

// 如果确实是非敏感常量，可用 nolint 标注
const tokenPrefix = "Bearer" //nolint:gosec // 不是密钥
```

> **反例**: [trap/hardcoded-secret/](trap/hardcoded-secret/) — 硬编码密钥 vs 环境变量 fail-fast

---

## 5. TLS 配置

### Go 默认安全级别

Go 1.22+ 的 `crypto/tls` 默认配置：
- MinVersion = TLS 1.2（不支持 TLS 1.0/1.1）
- 只启用 AEAD 密码套件（GCM、ChaCha20-Poly1305）
- 证书验证默认开启

**大多数情况下，Go 的默认值就是安全的，不需要额外配置。**

### 生产推荐配置

```go
tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS12,
    CipherSuites: []uint16{
        // TLS 1.2 密码套件（TLS 1.3 由 Go 自动管理）
        tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
        tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
    },
    // CurvePreferences 默认值已足够安全，通常不需要设置
}

srv := &http.Server{
    Addr:      ":443",
    TLSConfig: tlsConfig,
}
```

### InsecureSkipVerify 的后果

```go
// 危险：跳过证书验证
client := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: true, // 中间人可以冒充任何服务器！
        },
    },
}
```

InsecureSkipVerify 关闭了 TLS 最核心的保护——身份认证。攻击者可以拦截流量并返回自签名证书，客户端会照单全收。

**合理的替代方案：**
- 开发环境：使用 `mkcert` 生成本地可信证书
- 自签名证书：将 CA 证书加入 `tls.Config.RootCAs`
- 测试代码：使用 `httptest.NewTLSServer()` 配合 `srv.Client()`

### mTLS（双向 TLS）

```go
// 服务端要求客户端证书
tlsConfig := &tls.Config{
    ClientAuth: tls.RequireAndVerifyClientCert,
    ClientCAs:  certPool, // 信任的客户端 CA 证书池
    MinVersion: tls.VersionTLS12,
}
```

mTLS 适用于微服务间通信、API 网关等内部系统互联场景。

### TLS 版本对照

| 版本 | 状态 | 说明 |
|------|------|------|
| TLS 1.0 | 已废弃 | BEAST/POODLE 漏洞 |
| TLS 1.1 | 已废弃 | RFC 8996 (2021) |
| TLS 1.2 | 安全 | Go 默认最低版本 |
| TLS 1.3 | 最安全 | 更快握手（1-RTT），更强密码套件 |

> **反例**: [trap/weak-tls-config/](trap/weak-tls-config/) — InsecureSkipVerify 和 TLS 1.0
>
> **性能对比**: [performance/tls-overhead/](performance/tls-overhead/) — HTTPS 复用连接后开销仅增加约 20%

---

## 6. 密码学原语

### 密码存储：bcrypt vs Argon2id

| 特性 | bcrypt | Argon2id |
|------|--------|----------|
| 年代 | 1999 | 2015（Password Hashing Competition 冠军） |
| 抗 GPU | 一般 | 强（内存硬化） |
| 参数 | cost（默认 10） | time、memory、threads |
| 性能 | ~100ms (cost=10) | ~200ms (64MB memory) |
| 推荐 | 通用选择 | 新项目首选 |

```go
// bcrypt
import "golang.org/x/crypto/bcrypt"

hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
err = bcrypt.CompareHashAndPassword(hash, []byte(password))

// argon2id
import "golang.org/x/crypto/argon2"

salt := make([]byte, 16)
rand.Read(salt)
hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
// time=1, memory=64MB, threads=4, keyLen=32
```

### 对称加密：AES-GCM

```go
import "crypto/aes"
import "crypto/cipher"

// AES-GCM 提供加密 + 认证（AEAD）
block, _ := aes.NewCipher(key) // key: 16/24/32 字节
aead, _ := cipher.NewGCM(block)

// 加密
nonce := make([]byte, aead.NonceSize()) // 12 字节
rand.Read(nonce)
ciphertext := aead.Seal(nonce, nonce, plaintext, nil)
// nonce 前置到密文中，解密时取出

// 解密
nonce = ciphertext[:aead.NonceSize()]
plaintext, err := aead.Open(nil, nonce, ciphertext[aead.NonceSize():], nil)
```

**关键要点：**
- nonce 必须唯一（每次加密生成新的随机 nonce）
- GCM 同时提供保密性和完整性，不需要额外 HMAC
- key 必须用 crypto/rand 生成或由 KDF 派生

### HMAC 消息认证

```go
import "crypto/hmac"
import "crypto/sha256"

// 生成 HMAC
mac := hmac.New(sha256.New, secretKey)
mac.Write(message)
signature := mac.Sum(nil)

// 验证 HMAC（必须使用 ConstantTimeCompare 防止时序攻击）
mac2 := hmac.New(sha256.New, secretKey)
mac2.Write(message)
expected := mac2.Sum(nil)
if !hmac.Equal(signature, expected) { // 内部使用 subtle.ConstantTimeCompare
    return ErrInvalidSignature
}
```

### subtle.ConstantTimeCompare

```go
import "crypto/subtle"

// 错误：直接比较（时序攻击）
if string(sig) == string(expected) { ... }

// 正确：常量时间比较
if subtle.ConstantTimeCompare(sig, expected) != 1 { ... }
```

时序攻击原理：普通字符串比较遇到不匹配字节立即返回，攻击者通过测量响应时间可逐字节猜测正确值。ConstantTimeCompare 无论是否匹配，耗时相同。

> **反例**: [trap/md5-sha1-for-passwords/](trap/md5-sha1-for-passwords/) — MD5 一秒百万次 vs bcrypt 一次 100ms
>
> **性能对比**: [performance/hash-algorithm-cost/](performance/hash-algorithm-cost/) — 哈希迭代次数与耗时的关系

---

## 7. gosec 静态安全扫描

### 安装与使用

```bash
# 安装
go install github.com/securego/gosec/v2/cmd/gosec@latest

# 扫描当前项目
gosec ./...

# 指定输出格式
gosec -fmt=json -out=results.json ./...

# 只检查特定规则
gosec -include=G101,G201,G401 ./...

# 排除特定规则
gosec -exclude=G104 ./...
```

### golangci-lint 集成

```yaml
# .golangci.yml
linters:
  enable:
    - gosec

linters-settings:
  gosec:
    includes:
      - G101  # 硬编码凭证
      - G201  # SQL 拼接
      - G301  # 目录权限过大
      - G302  # 文件权限过大
      - G401  # 弱加密算法 (MD5/SHA1)
      - G402  # InsecureSkipVerify
      - G501  # 导入黑名单 (crypto/md5 等)
```

### 核心规则速查表

| 规则 | 描述 | 严重性 | 示例 |
|------|------|--------|------|
| **G101** | 硬编码凭证 | HIGH | `const password = "secret"` |
| **G102** | 绑定所有网络接口 | MEDIUM | `net.Listen("tcp", ":8080")` |
| **G104** | 未检查错误返回值 | MEDIUM | `f.Close()` 不检查 err |
| **G107** | HTTP 请求 URL 拼接 | MEDIUM | `http.Get(userInput)` |
| **G108** | pprof 自动暴露 | HIGH | `import _ "net/http/pprof"` |
| **G201** | SQL 字符串拼接 | HIGH | `fmt.Sprintf("...%s", input)` |
| **G202** | SQL 字符串拼接（Exec） | HIGH | 同 G201，针对 Exec |
| **G301** | 目录权限 > 0750 | MEDIUM | `os.Mkdir("dir", 0777)` |
| **G302** | 文件权限 > 0600 | MEDIUM | `os.WriteFile("f", d, 0666)` |
| **G304** | 文件路径注入 | MEDIUM | `os.Open(userInput)` |
| **G401** | 使用弱哈希 MD5/SHA1 | MEDIUM | `md5.Sum(data)` |
| **G402** | TLS InsecureSkipVerify | HIGH | `InsecureSkipVerify: true` |
| **G501** | 导入弱加密包 | MEDIUM | `import "crypto/md5"` |
| **G601** | 隐式内存别名（循环） | MEDIUM | Go 1.22+ 已修复 |

### nolint 规范

```go
// 明确说明原因，审查者可验证
const tokenType = "Bearer" //nolint:gosec // G101: 不是密钥，是协议常量

// 反例：不加原因的 nolint（不好）
const secret = "abc" //nolint:gosec
```

**nolint 使用原则：**
1. 每个 nolint 必须附带原因
2. 原因应解释为什么这里是安全的
3. Code Review 时重点审查 nolint 注释
4. 定期审计 nolint 数量，警惕滥用

---

## 安全编码检查清单

在提交代码前，确认以下事项：

- [ ] 安全随机数使用 `crypto/rand`
- [ ] SQL 查询使用参数化（? 占位符）
- [ ] 日志不包含密码、token 等敏感信息
- [ ] 敏感字段有 `json:"-"` 标签
- [ ] 密钥从环境变量读取，缺失时 fail-fast
- [ ] TLS 配置未使用 InsecureSkipVerify
- [ ] 密码存储使用 bcrypt 或 argon2id
- [ ] 密码学比较使用 `subtle.ConstantTimeCompare` 或 `hmac.Equal`
- [ ] gosec 扫描无 HIGH 级别告警
- [ ] 错误信息不暴露内部实现细节