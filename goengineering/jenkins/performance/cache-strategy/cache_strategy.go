package cachestrategy

/*
性能对比：Jenkins 中 Go 模块缓存 vs 无缓存

Jenkins 使用 Docker Agent 时，每次构建默认在全新容器中启动。
如果不挂载缓存 Volume，每次都要重新下载 Go 模块 + 重新编译，浪费大量时间。

本实验模拟：
  1. 无缓存：每次从零下载依赖 + 编译所有包
  2. 有缓存：命中模块缓存 + 增量编译

运行基准测试：
  go test -bench=. -benchmem -benchtime=3s ./...

关键结论：
  - Go 模块缓存（GOMODCACHE）避免重复下载，节省 1-3 分钟
  - Go 构建缓存（GOCACHE）避免重复编译，节省 2-5 分钟
  - 两者结合可以让 CI 构建时间从 8 分钟降到 2 分钟
  - Docker Agent 需用 Named Volume 持久化缓存

Jenkins 缓存配置示例：

  // ❌ 无缓存：每次干净构建
  agent {
      docker { image 'golang:1.24' }
  }

  // ✅ 有缓存：挂载 Named Volume
  agent {
      docker {
          image 'golang:1.24'
          args '-v go-mod-cache:/go/pkg/mod -v go-build-cache:/root/.cache/go-build'
      }
  }
*/

import (
	"crypto/sha256"
	"fmt"
	"sync"
)

// Dependency 模拟一个 Go 模块依赖
type Dependency struct {
	Module  string
	Version string
	Size    int // 模拟大小（字节）
}

// StandardDeps 返回一组典型 Go 项目的依赖
func StandardDeps() []Dependency {
	return []Dependency{
		{Module: "github.com/gin-gonic/gin", Version: "v1.10.0", Size: 8192},
		{Module: "gorm.io/gorm", Version: "v1.25.0", Size: 6144},
		{Module: "go.uber.org/zap", Version: "v1.27.0", Size: 4096},
		{Module: "github.com/redis/go-redis/v9", Version: "v9.7.0", Size: 5120},
		{Module: "github.com/stretchr/testify", Version: "v1.9.0", Size: 3072},
		{Module: "google.golang.org/grpc", Version: "v1.68.0", Size: 10240},
		{Module: "github.com/golang-jwt/jwt/v5", Version: "v5.2.0", Size: 2048},
		{Module: "github.com/spf13/viper", Version: "v1.19.0", Size: 4096},
	}
}

// download 模拟下载一个依赖（CPU 密集操作代替网络延迟）
func download(dep Dependency) [32]byte {
	data := make([]byte, dep.Size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	hash := sha256.Sum256(data)
	for i := 0; i < 50; i++ {
		hash = sha256.Sum256(hash[:])
	}
	return hash
}

// compilePkg 模拟编译一个包
func compilePkg(source []byte) [32]byte {
	hash := sha256.Sum256(source)
	for i := 0; i < 80; i++ {
		hash = sha256.Sum256(hash[:])
	}
	return hash
}

// Cache 模拟依赖和构建缓存
type Cache struct {
	mu       sync.RWMutex
	modCache map[string][32]byte
	buildHit map[string][32]byte
}

// NewCache 创建空缓存
func NewCache() *Cache {
	return &Cache{
		modCache: make(map[string][32]byte),
		buildHit: make(map[string][32]byte),
	}
}

// BuildWithoutCache 模拟无缓存构建：下载所有依赖 + 编译所有包
func BuildWithoutCache(deps []Dependency, pkgCount int) [][32]byte {
	// 下载所有依赖
	for _, dep := range deps {
		download(dep)
	}
	// 编译所有包
	results := make([][32]byte, pkgCount)
	for i := 0; i < pkgCount; i++ {
		source := make([]byte, 2048)
		for j := range source {
			source[j] = byte((i*7 + j*13) % 256)
		}
		results[i] = compilePkg(source)
	}
	return results
}

// BuildWithCache 模拟有缓存构建：跳过已缓存的依赖和包
func BuildWithCache(deps []Dependency, pkgCount int, cache *Cache) [][32]byte {
	// 只下载未缓存的依赖
	cache.mu.RLock()
	for _, dep := range deps {
		key := dep.Module + "@" + dep.Version
		if _, ok := cache.modCache[key]; !ok {
			cache.mu.RUnlock()
			hash := download(dep)
			cache.mu.Lock()
			cache.modCache[key] = hash
			cache.mu.Unlock()
			cache.mu.RLock()
		}
	}
	cache.mu.RUnlock()

	// 只编译变更的包（模拟：只有第一个包变更了）
	results := make([][32]byte, pkgCount)
	for i := 0; i < pkgCount; i++ {
		key := fmt.Sprintf("pkg%d", i)
		cache.mu.RLock()
		cached, ok := cache.buildHit[key]
		cache.mu.RUnlock()

		if ok && i > 0 { // 假设只有 pkg0 变更了
			results[i] = cached
		} else {
			source := make([]byte, 2048)
			for j := range source {
				source[j] = byte((i*7 + j*13) % 256)
			}
			hash := compilePkg(source)
			results[i] = hash
			cache.mu.Lock()
			cache.buildHit[key] = hash
			cache.mu.Unlock()
		}
	}
	return results
}
