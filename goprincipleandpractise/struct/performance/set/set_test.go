package set

import "testing"

/*
map[string]bool vs map[string]struct{} 作为 Set 的性能对比

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=3 -benchmem .

结论:
  - 两者性能几乎相同（map 操作主导了开销）, 空结构体有约4.68%的性能优势
  - emptyStructSet 的 B/op 略低（value 占 0 字节 vs 1 字节）
  - 当元素数量极大时，struct{} 的内存优势更明显
  - 更重要的是语义优势：struct{} 明确表示"只关心 key，value 无意义"
*/

func BenchmarkBoolSet(b *testing.B) {
	for b.Loop() {
		s := make(boolSet)
		RunSetBenchmark(10000, s)
	}
}

func BenchmarkEmptyStructSet(b *testing.B) {
	for b.Loop() {
		s := make(emptyStructSet)
		RunSetBenchmark(10000, s)
	}
}
