package nocache

/*
陷阱：CI 流水线中不配置依赖缓存

问题说明：
  Go 项目在 CI 中如果不配置缓存，每次 pipeline 运行都会：
  1. 重新下载所有 go module 依赖（go mod download）
  2. 重新编译所有依赖包（go build cache miss）

  典型的中等规模 Go 项目（50-200 个依赖）：
  - 无缓存：go mod download 30-120s + go build 60-300s = 2-7 分钟浪费
  - 有缓存：go mod download 0s + go build 5-15s = 几乎零额外开销

  一天跑 20 次 CI，一年浪费 200+ 小时等待时间。

正确做法（GitHub Actions）：

  # 方式一：setup-go 内置缓存（推荐）
  - uses: actions/setup-go@v5
    with:
      go-version: '1.24'
      cache: true

  # 方式二：手动配置缓存
  - uses: actions/cache@v4
    with:
      path: |
        ~/go/pkg/mod
        ~/.cache/go-build
      key: go-${{ runner.os }}-${{ hashFiles('go.sum') }}
      restore-keys: go-${{ runner.os }}-

正确做法（GitLab CI）：

  cache:
    key: ${CI_COMMIT_REF_SLUG}
    paths:
      - /go/pkg/mod/
*/

import (
	"crypto/sha256"
	"math/rand"
)

// Dependency 模拟一个外部依赖包
type Dependency struct {
	Name    string
	Version string
	Size    int
}

// SimulateDependencies 模拟中等规模 Go 项目的依赖列表
func SimulateDependencies(count int) []Dependency {
	deps := make([]Dependency, count)
	names := []string{
		"gin-gonic/gin", "go-sql-driver/mysql", "redis/go-redis",
		"grpc/grpc-go", "uber-go/zap", "stretchr/testify",
		"spf13/viper", "prometheus/client_golang",
	}
	for i := range deps {
		deps[i] = Dependency{
			Name:    names[i%len(names)],
			Version: "v1.0.0",
			Size:    (i + 1) * 1024,
		}
	}
	return deps
}

// DownloadDependency 模拟从网络下载依赖（无缓存）
func DownloadDependency(dep Dependency) []byte {
	data := make([]byte, dep.Size)
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}
	// 模拟哈希校验
	_ = sha256.Sum256(data)
	return data
}

// LoadFromCache 模拟从本地缓存加载依赖
func LoadFromCache(dep Dependency) []byte {
	data := make([]byte, dep.Size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return data
}

// BuildWithoutCache 无缓存构建：下载 + 编译所有依赖
func BuildWithoutCache(deps []Dependency) int {
	total := 0
	for _, dep := range deps {
		data := DownloadDependency(dep)
		total += len(data)
		_ = sha256.Sum256(data) // 模拟编译
	}
	return total
}

// BuildWithCache 有缓存构建：从缓存加载，跳过编译
func BuildWithCache(deps []Dependency) int {
	total := 0
	for _, dep := range deps {
		data := LoadFromCache(dep)
		total += len(data)
	}
	return total
}
