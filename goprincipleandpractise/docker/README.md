# Go 项目 Docker 化构建详解

> Go 天然适合容器化：静态编译、单二进制、无运行时依赖。本文覆盖 Dockerfile 编写、多阶段构建、镜像优化、安全最佳实践。

## 目录

1. [为什么 Go 适合容器化](#1-为什么-go-适合容器化)
2. [基础镜像选择](#2-基础镜像选择)
3. [多阶段构建](#3-多阶段构建)
4. [镜像优化技巧](#4-镜像优化技巧)
5. [docker-compose 本地开发](#5-docker-compose-本地开发)
6. [安全最佳实践](#6-安全最佳实践)

---

## 1 为什么 Go 适合容器化

| 特性 | Go | Java/Python/Node |
|------|-----|-------------------|
| 编译产物 | 单一静态二进制 | JAR + JVM / 源码 + 解释器 |
| 运行时依赖 | 无（CGO_ENABLED=0） | JRE / Python / Node.js |
| 最小镜像 | scratch（0MB 基础） | 需要运行时（100-800MB） |
| 启动速度 | 毫秒级 | 秒级（JVM 预热） |
| 内存占用 | 低（无 GC overhead） | 较高 |

Go 的 `CGO_ENABLED=0` 静态编译可以生成完全自包含的二进制，放进空白镜像（scratch）就能运行。

---

## 2 基础镜像选择

### 基础镜像对比

| 镜像 | 大小 | CVE 数量 | Shell | 调试工具 | 适用场景 |
|------|------|---------|-------|---------|---------|
| `golang:1.24` | ~800MB | 多 | 有 | 完整 | 仅用于构建阶段 |
| `alpine:3.19` | ~7MB | 极少 | 有 | 基础 | 需要 shell 调试 |
| `gcr.io/distroless/static` | ~2MB | 极少 | 无 | 无 | 生产推荐 |
| `scratch` | 0MB | 0 | 无 | 无 | 极致精简 |

**选择建议**：
- **生产环境**：distroless 或 scratch
- **需要调试**：alpine（可以 exec 进容器）
- **构建阶段**：golang 官方镜像

> **反例**: [trap/fat-image/](trap/fat-image/) — 用 golang 镜像做运行时，800MB 的臃肿镜像

---

## 3 多阶段构建

### 3.1 标准两阶段（构建 + 运行）

```dockerfile
# === 阶段 1：构建 ===
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 先复制依赖文件，利用层缓存
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /app/server ./cmd/server

# === 阶段 2：运行 ===
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/server /server

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/server"]
```

### 3.2 三阶段（构建 + 测试 + 运行）

```dockerfile
# === 阶段 1：构建 ===
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /app/server ./cmd/server

# === 阶段 2：测试 ===
FROM builder AS tester
RUN go test -race -count=1 ./...
RUN go vet ./...

# === 阶段 3：运行 ===
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/server /server
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
```

> **反例**: [trap/no-multistage/](trap/no-multistage/) — 不用多阶段构建，源码和构建工具都在最终镜像中

---

## 4 镜像优化技巧

### 4.1 静态编译

```bash
CGO_ENABLED=0 GOOS=linux go build -o server .
```

`CGO_ENABLED=0` 禁用 CGO，生成纯静态二进制，不依赖任何 C 库。

### 4.2 去除调试信息

```bash
go build -ldflags="-s -w" -o server .
# -s: 去掉符号表（symbol table）
# -w: 去掉 DWARF 调试信息
# 通常减小 20-30% 的二进制体积
```

### 4.3 UPX 压缩（利弊分析）

```dockerfile
RUN apk add --no-cache upx
RUN upx --best /app/server
```

| 方面 | 优点 | 缺点 |
|------|------|------|
| 体积 | 再减 50-70% | — |
| 启动 | — | 首次启动慢（需解压） |
| 调试 | — | pprof/dlv 无法工作 |
| 安全 | — | 某些安全扫描误报 |

**建议**：一般不推荐。`-ldflags="-s -w"` 已经足够，UPX 的缺点（启动慢、无法 profile）在生产中代价较大。

### 4.4 层缓存优化

```dockerfile
# ✅ 正确：先复制依赖文件，再复制源码
COPY go.mod go.sum ./        # 这一层很少变化
RUN go mod download          # 依赖缓存命中率高

COPY . .                     # 代码变化频繁
RUN go build .               # 只有这层需要重建

# ❌ 错误：一次性复制所有文件
COPY . .                     # 任何文件变化都破坏缓存
RUN go mod download           # 每次都重新下载
RUN go build .                # 每次都全量编译
```

> **性能对比**: [performance/layer-cache/](performance/layer-cache/) — 缓存命中 vs 未命中的构建时间差异

### 4.5 .dockerignore

```
# .dockerignore
.git
.github
.vscode
.idea
*.md
docs/
bin/
tmp/
coverage.out
*.test
.env
.env.*
docker-compose*.yml
Makefile
```

> **反例**: [trap/no-dockerignore/](trap/no-dockerignore/) — 不配 .dockerignore 导致敏感文件进入镜像

---

## 5 docker-compose 本地开发

### Go 服务 + MySQL + Redis

```yaml
# docker-compose.yml
version: '3.8'

services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=mysql
      - DB_PORT=3306
      - DB_USER=root
      - DB_PASSWORD=secret
      - REDIS_URL=redis://redis:6379
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: secret
      MYSQL_DATABASE: myapp
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s
      timeout: 3s
      retries: 10

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 10

volumes:
  mysql_data:
```

### 热重载（air）

```yaml
services:
  app:
    build:
      context: .
      dockerfile: Dockerfile.dev
    volumes:
      - .:/app                    # 挂载源码
    command: air                   # 文件变更自动重编译
```

```dockerfile
# Dockerfile.dev
FROM golang:1.24-alpine
RUN go install github.com/air-verse/air@latest
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
CMD ["air"]
```

---

## 6 安全最佳实践

### 6.1 非 root 用户运行

```dockerfile
# 方式一：distroless 的 nonroot 变体
FROM gcr.io/distroless/static:nonroot
USER nonroot:nonroot

# 方式二：alpine 中创建用户
FROM alpine:3.19
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser:appgroup

# 方式三：scratch 中指定 UID
FROM scratch
COPY --from=builder /app/server /server
USER 65534:65534
ENTRYPOINT ["/server"]
```

> **反例**: [trap/root-user/](trap/root-user/) — 以 root 运行容器的安全风险

### 6.2 只读文件系统

```yaml
# docker-compose.yml
services:
  app:
    read_only: true
    tmpfs:
      - /tmp                      # 临时文件写到内存
```

### 6.3 镜像扫描

```bash
# 使用 trivy 扫描镜像漏洞
trivy image myapp:latest

# CI 中集成
- name: Scan image
  uses: aquasecurity/trivy-action@master
  with:
    image-ref: myapp:latest
    severity: 'CRITICAL,HIGH'
    exit-code: '1'
```

### 6.4 CA 证书

scratch 镜像没有 CA 证书，需要手动复制：

```dockerfile
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache ca-certificates

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/server /server
```

---

## 总结

| 要点 | 说明 |
|------|------|
| 基础镜像 | 生产用 distroless/scratch，调试用 alpine |
| 多阶段构建 | 构建阶段用 golang，运行阶段用最小镜像 |
| 静态编译 | CGO_ENABLED=0 -ldflags="-s -w" |
| 层缓存 | 先 COPY go.mod → go mod download → COPY . . |
| 安全 | 非 root、只读文件系统、镜像扫描 |
| .dockerignore | 排除 .git、.env、docs、bin 等 |

**镜像大小参考**：
| 配置 | 大小 |
|------|------|
| golang:1.24 运行 | ~800MB |
| alpine + Go binary | ~15MB |
| distroless + Go binary | ~8MB |
| scratch + Go binary | ~5MB |
| scratch + stripped + UPX | ~2MB |

**常见陷阱**：
- 用 golang 镜像做运行时：[trap/fat-image/](trap/fat-image/)
- 不用多阶段构建：[trap/no-multistage/](trap/no-multistage/)
- 以 root 运行：[trap/root-user/](trap/root-user/)
- 缺少 .dockerignore：[trap/no-dockerignore/](trap/no-dockerignore/)

**性能对比**：
- 镜像大小对比：[performance/image-size/](performance/image-size/)
- 层缓存效果：[performance/layer-cache/](performance/layer-cache/)