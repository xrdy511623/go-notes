package mockperf

import "testing"

// ---------- 基准测试：接口调用 vs 直接调用 ----------
//
// 运行方式：
//   go test -bench=. -benchmem ./goprincipleandpractise/unit-test/performance/
//   go test -bench=. -benchtime=3s -count=5 -benchmem ./goprincipleandpractise/unit-test/performance/
//
// 预期结果：两者性能差异在纳秒级，证明接口抽象的开销可忽略。

/*
go test -bench=^Bench -benchtime=3s -benchmem .
goos: darwin
goarch: arm64
pkg: go-notes/goprincipleandpractise/unit-test/performance
cpu: Apple M4
BenchmarkDirectCall-10                  1000000000               3.000 ns/op           0 B/op          0 allocs/op
BenchmarkInterfaceCall-10               1000000000               3.000 ns/op           0 B/op          0 allocs/op
BenchmarkInterfaceCall_NoInline-10      1000000000               3.000 ns/op           0 B/op          0 allocs/op
BenchmarkDirectCall_NoInline-10         1000000000               3.000 ns/op           0 B/op          0 allocs/op
PASS
ok      go-notes/goprincipleandpractise/unit-test/performance   12.143s
*/

func BenchmarkDirectCall(b *testing.B) {
	for b.Loop() {
		directAdd(1, 2)
	}
}

func BenchmarkInterfaceCall(b *testing.B) {
	var calc Calculator = RealCalculator{}
	for b.Loop() {
		calc.Add(1, 2)
	}
}

// BenchmarkInterfaceCall_NoInline 防止编译器内联优化，
// 更真实地反映接口调用开销
func BenchmarkInterfaceCall_NoInline(b *testing.B) {
	var calc Calculator = RealCalculator{}
	var result int
	for b.Loop() {
		result = calc.Add(result, 1)
	}
	_ = result // 防止编译器优化掉整个循环
}

func BenchmarkDirectCall_NoInline(b *testing.B) {
	var result int
	for b.Loop() {
		result = directAdd(result, 1)
	}
	_ = result
}
