package performance

import "testing"

/*
select case 数量与 default 对性能的影响

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=3 -benchmem .

对比维度:
  1. 带 default 的单 case select：编译器优化为 selectnbrecv，不走 selectgo
  2. 不同 case 数量（1/2/4/8）的 select 性能差异
     - 1 case: 编译器优化为直接 chanrecv
     - 2 case: selectgo 但走快速路径
     - 4/8 case: 完整 selectgo（随机排列 + 锁排序 + 遍历）

结论:
  - 带 default 的 select 被编译器优化为非阻塞调用，性能最好
  - 1 case 的 select 被编译器优化为直接 channel 操作，与带 default 接近
  - 2 case 开始走 selectgo，性能明显下降
  - 4→8 case 性能继续下降，但幅度比 1→2 小
  - 建议：尽量减少 select 中的 case 数量，避免 case 爆炸
*/

// ---------- 带 default ----------

func BenchmarkSelectDefault1Case(b *testing.B) {
	for b.Loop() {
		selectSink = SelectDefault1Case(1000)
	}
}

// ---------- 不带 default ----------

func BenchmarkSelectNoDefault1Case(b *testing.B) {
	for b.Loop() {
		selectSink = SelectNoDefault1Case(1000)
	}
}

func BenchmarkSelectNoDefault2Case(b *testing.B) {
	for b.Loop() {
		selectSink = SelectNoDefault2Case(1000)
	}
}

func BenchmarkSelectNoDefault4Case(b *testing.B) {
	for b.Loop() {
		selectSink = SelectNoDefault4Case(1000)
	}
}

func BenchmarkSelectNoDefault8Case(b *testing.B) {
	for b.Loop() {
		selectSink = SelectNoDefault8Case(1000)
	}
}
