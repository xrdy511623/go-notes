package cachestrategy

import (
	"testing"
)

/*
基准测试：有缓存 vs 无缓存构建

运行方式：
  go test -bench=. -benchmem -benchtime=3s .

预期结果（8 个依赖，10 个包）：
  BenchmarkBuildWithoutCache-8    xxx    yyy ns/op    zzz B/op    n allocs/op
  BenchmarkBuildWithCache-8       xxx    yyy ns/op    zzz B/op    n allocs/op

  模式           模拟耗时      实际 CI 场景
  无缓存          ~8min       下载依赖(3min) + 全量编译(5min)
  有缓存(首次)    ~8min       同无缓存（冷启动）
  有缓存(后续)    ~2min       跳过下载 + 增量编译

  节省: ~75%（后续构建）

结论：
  - 首次构建无法避免全量下载，但缓存会被持久化
  - 后续构建命中缓存后，只编译变更的包
  - Jenkins Docker Agent 必须用 Named Volume 挂载缓存路径
  - 两个关键缓存目录：/go/pkg/mod（模块）+ /root/.cache/go-build（编译）
*/

func BenchmarkBuildWithoutCache(b *testing.B) {
	deps := StandardDeps()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildWithoutCache(deps, 10)
	}
}

func BenchmarkBuildWithCache_ColdStart(b *testing.B) {
	deps := StandardDeps()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache := NewCache() // 每次新缓存（冷启动）
		BuildWithCache(deps, 10, cache)
	}
}

func BenchmarkBuildWithCache_WarmCache(b *testing.B) {
	deps := StandardDeps()
	// 预热缓存
	cache := NewCache()
	BuildWithCache(deps, 10, cache)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildWithCache(deps, 10, cache)
	}
}
