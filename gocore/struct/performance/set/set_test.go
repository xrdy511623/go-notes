package set

import (
	"strconv"
	"testing"
)

/*
map[string]bool vs map[string]struct{} 作为 Set 的性能对比

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=3 -benchmem .

结论:
  - 本基准已把 key 预生成，避免 strconv.Itoa 的分配成本干扰对比
  - 两者性能通常接近，差异以 map 操作为主，不同环境波动较大
  - B/op 差异往往很小，不应机械解读为稳定收益
  - 当元素数量极大时，struct{} 在内存占用上仍更具语义与理论优势
  - 更重要的是语义优势：struct{} 明确表示"只关心 key，value 无意义"
*/

func buildKeys(n int) []string {
	keys := make([]string, n)
	for i := range n {
		keys[i] = strconv.Itoa(i)
	}
	return keys
}

func BenchmarkBoolSet(b *testing.B) {
	keys := buildKeys(10000)
	b.ReportAllocs()
	for b.Loop() {
		s := make(boolSet, len(keys))
		RunSetBenchmark(keys, s)
	}
}

func BenchmarkEmptyStructSet(b *testing.B) {
	keys := buildKeys(10000)
	b.ReportAllocs()
	for b.Loop() {
		s := make(emptyStructSet, len(keys))
		RunSetBenchmark(keys, s)
	}
}
