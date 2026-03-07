package ldflagsvsembed

import "testing"

/*
基准测试：ldflags vs go:embed 版本信息读取性能

运行方式：
  go test -bench=. -benchmem -benchtime=3s .

预期结果：
  BenchmarkGetLdflagsVersion-8       xxx    ~0 ns/op    0 B/op    0 allocs/op
  BenchmarkGetLdflagsFullInfo-8      xxx    ~yyy ns/op  zzz B/op  n allocs/op
  BenchmarkGetEmbedVersion-8         xxx    ~yyy ns/op  zzz B/op  n allocs/op

  两者运行时性能几乎相同：
  - 简单变量读取：零分配，纳秒级
  - 字符串拼接（FullInfo）：有分配开销
  - 差异不在性能，在工程实践

结论：优先选择 ldflags（灵活性更好），性能不是考量因素。
*/

func BenchmarkGetLdflagsVersion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetLdflagsVersion()
	}
}

func BenchmarkGetLdflagsFullInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetLdflagsFullInfo()
	}
}

func BenchmarkGetEmbedVersion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetEmbedVersion()
	}
}

func BenchmarkGetBuildInfoVersion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetBuildInfoVersion()
	}
}
