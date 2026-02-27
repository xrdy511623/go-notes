package cachecomparison

import "testing"

/*
基准测试：缓存 vs 无缓存构建

运行方式：
  go test -bench=. -benchmem -benchtime=3s .

预期结果（80 个依赖）：
  BenchmarkNoCacheBuild-8    xxx    yyy ns/op    zzz B/op
  BenchmarkCachedBuild-8     xxx    yyy ns/op    zzz B/op

  缓存构建通常比无缓存快 5-10 倍。
*/

func BenchmarkNoCacheBuild(b *testing.B) {
	deps := StandardDeps()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NoCacheBuild(deps)
	}
}

func BenchmarkCachedBuild(b *testing.B) {
	deps := StandardDeps()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CachedBuild(deps)
	}
}
