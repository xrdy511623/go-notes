package cachecomparison

/*
性能对比：有缓存 vs 无缓存的 CI 构建

本实验模拟 CI 环境下有缓存和无缓存的构建时间差异。

运行基准测试：
  go test -bench=. -benchmem -benchtime=3s .

关键结论：
  - 缓存命中时，构建速度提升 5-10 倍
  - go mod cache 避免重复下载（节省网络 I/O）
  - go build cache 避免重复编译（节省 CPU）
  - CI 中必须配置两级缓存：GOMODCACHE + GOCACHE
*/

import (
	"crypto/sha256"
	"math/rand"
)

// Module 模拟一个 Go module 依赖
type Module struct {
	Path    string
	Version string
	Size    int
}

// StandardDeps 返回标准规模的依赖列表
func StandardDeps() []Module {
	modules := make([]Module, 80)
	for i := range modules {
		modules[i] = Module{
			Path:    "example.com/dep",
			Version: "v1.0.0",
			Size:    (i + 1) * 512,
		}
	}
	return modules
}

// downloadModule 模拟从网络下载模块
func downloadModule(m Module) []byte {
	data := make([]byte, m.Size)
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}
	_ = sha256.Sum256(data) // 校验哈希
	return data
}

// compileModule 模拟编译模块
func compileModule(data []byte) [32]byte {
	hash := sha256.Sum256(data)
	for i := 0; i < 50; i++ {
		hash = sha256.Sum256(hash[:])
	}
	return hash
}

// loadCached 模拟从缓存加载
func loadCached(m Module) []byte {
	data := make([]byte, m.Size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return data
}

// NoCacheBuild 无缓存构建：下载 + 编译全部依赖
func NoCacheBuild(deps []Module) int {
	total := 0
	for _, dep := range deps {
		data := downloadModule(dep)
		_ = compileModule(data)
		total += len(data)
	}
	return total
}

// CachedBuild 有缓存构建：缓存加载 + 跳过已编译的
func CachedBuild(deps []Module) int {
	total := 0
	for _, dep := range deps {
		data := loadCached(dep)
		total += len(data)
	}
	return total
}
