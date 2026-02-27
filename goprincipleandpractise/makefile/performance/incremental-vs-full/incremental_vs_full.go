package incrementalvsfull

/*
性能对比：增量构建 vs 全量构建

Go 编译器内置了构建缓存（GOCACHE），只重新编译有变更的包及其依赖。
Makefile 中的 clean + build 会清除缓存导致全量重建。

本实验模拟：
  1. 全量构建：所有包都需要编译（类似 make clean && make build）
  2. 增量构建：只编译变更的包（类似直接 make build）

运行基准测试：
  go test -bench=. -benchmem -benchtime=3s .

关键结论：
  - 增量构建通常只需全量构建 5-20% 的时间
  - 不要在 CI 中无脑执行 make clean && make build
  - 利用 CI 缓存保留 GOCACHE 和 GOMODCACHE
  - 只在依赖变更或需要 reproducible build 时全量重建
*/

import (
	"crypto/sha256"
	"fmt"
)

// Package 模拟一个 Go 包
type Package struct {
	Name     string
	Source   []byte
	Modified bool // 是否有修改
}

// BuildCache 模拟构建缓存
type BuildCache struct {
	compiled map[string][32]byte
}

// NewBuildCache 创建空缓存
func NewBuildCache() *BuildCache {
	return &BuildCache{compiled: make(map[string][32]byte)}
}

// NewPackages 创建 n 个模拟包，其中 modifiedCount 个被修改
func NewPackages(n, modifiedCount int) []Package {
	pkgs := make([]Package, n)
	for i := range pkgs {
		source := make([]byte, 4096)
		for j := range source {
			source[j] = byte((i*11 + j*7) % 256)
		}
		pkgs[i] = Package{
			Name:     fmt.Sprintf("pkg%d", i),
			Source:   source,
			Modified: i < modifiedCount,
		}
	}
	return pkgs
}

// compilePackage 模拟编译单个包
func compilePackage(pkg Package) [32]byte {
	hash := sha256.Sum256(pkg.Source)
	for i := 0; i < 100; i++ {
		hash = sha256.Sum256(hash[:])
	}
	return hash
}

// FullBuild 全量构建：编译所有包，忽略缓存
func FullBuild(pkgs []Package) map[string][32]byte {
	results := make(map[string][32]byte, len(pkgs))
	for _, pkg := range pkgs {
		results[pkg.Name] = compilePackage(pkg)
	}
	return results
}

// IncrementalBuild 增量构建：只编译修改过的包，其余从缓存读取
func IncrementalBuild(pkgs []Package, cache *BuildCache) map[string][32]byte {
	results := make(map[string][32]byte, len(pkgs))
	for _, pkg := range pkgs {
		if pkg.Modified {
			// 需要重新编译
			result := compilePackage(pkg)
			cache.compiled[pkg.Name] = result
			results[pkg.Name] = result
		} else if cached, ok := cache.compiled[pkg.Name]; ok {
			// 缓存命中，直接使用
			results[pkg.Name] = cached
		} else {
			// 首次编译
			result := compilePackage(pkg)
			cache.compiled[pkg.Name] = result
			results[pkg.Name] = result
		}
	}
	return results
}
