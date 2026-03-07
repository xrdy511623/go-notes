package performance

import "testing"

/*
对比直接方法调用与接口动态分派的性能开销。

执行命令:

	go test -run '^$' -bench '^BenchmarkDispatch' -benchtime=3s -count=5 -benchmem .

Apple M4(Go 1.24.5)下预期:

	DirectCall     ~0.3 ns/op   (编译器可能内联)
	InterfaceCall  ~1.5 ns/op   (itab查表+间接调用)
	TypeAssertion  ~1.0 ns/op   (hash比较)
	TypeSwitch     ~2.0 ns/op   (多分支hash比较)
	EmptyInterface ~0.5 ns/op   (装箱/拆箱)

结论:
 1. 接口调用比直接调用慢约1-2ns，在绝大多数场景可忽略。
 2. 编译器对直接调用可做内联优化，接口调用无法内联。
 3. type switch开销随case数增加而略增。
*/

var sinkInt int

func BenchmarkDispatchDirect(b *testing.B) {
	d := DirectAdder{}
	for b.Loop() {
		sinkInt = d.Add(1, 2)
	}
}

func BenchmarkDispatchInterface(b *testing.B) {
	var iface Adder = DirectAdder{}
	for b.Loop() {
		sinkInt = iface.Add(1, 2)
	}
}

func BenchmarkDispatchTypeAssertionSafe(b *testing.B) {
	var iface interface{} = DirectAdder{}
	for b.Loop() {
		if v, ok := iface.(Adder); ok {
			sinkInt = v.Add(1, 2)
		}
	}
}

func BenchmarkDispatchTypeSwitch(b *testing.B) {
	var iface Describer = IntType{V: 42}
	for b.Loop() {
		switch v := iface.(type) {
		case IntType:
			sinkInt = v.V
		case StrType:
			sinkInt = len(v.V)
		case FloatType:
			sinkInt = int(v.V)
		}
	}
}

var sinkInterface interface{}

func BenchmarkDispatchEmptyInterface(b *testing.B) {
	x := 42
	for b.Loop() {
		sinkInterface = x             // 装箱
		sinkInt = sinkInterface.(int) // 拆箱
	}
}
