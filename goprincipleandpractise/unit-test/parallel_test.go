package unittest

import (
	"fmt"
	"sync/atomic"
	"testing"
)

// ---------- t.Parallel()：并行测试执行 ----------
//
// 调用 t.Parallel() 后，该测试不再阻塞父测试，
// 而是与其他同样调用了 t.Parallel() 的测试并行运行。
//
// 执行规则：
// 1. 顶层 TestXxx 调用 t.Parallel() → 与其他并行顶层测试同时执行
// 2. 子测试调用 t.Parallel() → 与同一父测试下的其他并行子测试同时执行
// 3. 父测试在所有并行子测试完成后才标记为完成
//
// Go 1.22+ 语义变化：
// Go 1.22 起，for range 循环变量每次迭代都会创建新的作用域，
// 因此不再需要 `tt := tt` 捕获。但本仓库使用 Go 1.24，
// 为教学目的保留旧写法说明。详见 trap/loop-capture/。

func TestParallel_TableDriven(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{"double 1", 1, 2},
		{"double 2", 2, 4},
		{"double 3", 3, 6},
		{"double 0", 0, 0},
		{"double -1", -1, -2},
	}

	for _, tt := range tests {
		// Go 1.22 之前必须这样做：
		//   tt := tt // 捕获循环变量，避免闭包引用最后一个值
		// Go 1.22+（本仓库使用 Go 1.24）不再需要，但写了也无害。
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // 标记子测试为并行
			got := tt.input * 2
			if got != tt.want {
				t.Errorf("double(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// TestParallel_SharedState 演示并行测试中安全共享状态的方式
func TestParallel_SharedState(t *testing.T) {
	// 使用 atomic 保证并发安全
	var counter int64

	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("increment_%d", i), func(t *testing.T) {
			t.Parallel()
			atomic.AddInt64(&counter, 1)
		})
	}

	// 注意：父测试中紧跟 for 循环后的代码，在并行子测试"调度"后即执行，
	// 但此时子测试可能尚未运行完毕。验证需要放在 t.Cleanup 中，
	// 因为 Cleanup 在所有子测试完成后才执行。
	t.Cleanup(func() {
		if v := atomic.LoadInt64(&counter); v != 100 {
			t.Errorf("counter = %d, want 100", v)
		}
	})
}

// TestParallel_Independent 演示多个独立的并行顶层测试
func TestParallel_Independent(t *testing.T) {
	t.Parallel() // 与其他顶层并行测试同时执行

	// 这个测试可以与 TestParallel_TableDriven 等同时运行
	result := 42 * 2
	if result != 84 {
		t.Errorf("got %d, want 84", result)
	}
}
