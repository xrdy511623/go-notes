package incrementalvsfull

import (
	"fmt"
	"testing"
)

/*
基准测试：增量构建 vs 全量构建

运行方式：
  go test -bench=. -benchmem -benchtime=3s .

预期结果（20 个包，2 个修改）：
  BenchmarkFullBuild-8                 xxx    yyy ns/op
  BenchmarkIncrementalBuild/modified=1  xxx    yyy ns/op
  BenchmarkIncrementalBuild/modified=2  xxx    yyy ns/op
  BenchmarkIncrementalBuild/modified=5  xxx    yyy ns/op

  修改比例     相对全量构建耗时
  1/20 (5%)    ~5%
  2/20 (10%)   ~10%
  5/20 (25%)   ~25%

结论：
  日常开发中通常只修改 1-3 个包，增量构建的优势巨大。
  不要在 Makefile 中把 clean 作为 build 的前置依赖！
*/

func BenchmarkFullBuild(b *testing.B) {
	pkgs := NewPackages(20, 0) // 全部需要编译
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FullBuild(pkgs)
	}
}

func BenchmarkIncrementalBuild(b *testing.B) {
	modifiedCounts := []int{1, 2, 5, 10}

	for _, mc := range modifiedCounts {
		b.Run(fmt.Sprintf("modified=%d", mc), func(b *testing.B) {
			pkgs := NewPackages(20, mc)
			// 预热缓存：先全量编译一次
			cache := NewBuildCache()
			warmupPkgs := NewPackages(20, 0)
			IncrementalBuild(warmupPkgs, cache)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				IncrementalBuild(pkgs, cache)
			}
		})
	}
}
