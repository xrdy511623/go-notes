package unittest

import (
	"testing"
	"time"
)

// ---------- testing.Short()：长短测试分类 ----------
//
// 使用 `go test -short` 时，testing.Short() 返回 true。
// 耗时的测试可以用 t.Skip 跳过，实现快速反馈循环。
//
// -short 标志 vs Build Tags 对比：
//
// | 方式         | 用法                                    | 适用场景                   |
// |-------------|----------------------------------------|--------------------------|
// | -short      | testing.Short() + t.Skip()             | 按耗时分类，最常用          |
// | build tags  | //go:build integration                 | 按测试类型分类（单元/集成）  |
// | -run        | go test -run 'Pattern'                 | 按名称选择性运行           |

func TestShort_FastTest(t *testing.T) {
	// 快速测试：无论是否 -short 都会运行
	result := 1 + 1
	if result != 2 {
		t.Errorf("1+1 = %d, want 2", result)
	}
}

func TestShort_SlowIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow integration test in short mode")
	}

	// 模拟耗时操作（实际场景：数据库查询、外部 API 调用等）
	time.Sleep(100 * time.Millisecond)

	// 执行慢速测试逻辑
	result := expensiveComputation()
	if result != 42 {
		t.Errorf("got %d, want 42", result)
	}
}

func TestShort_AnotherSlowTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: requires external service")
	}

	// 模拟需要外部依赖的测试
	time.Sleep(50 * time.Millisecond)
	assertEqual(t, true, true) // placeholder assertion
}

// expensiveComputation 模拟一个耗时计算
func expensiveComputation() int {
	time.Sleep(50 * time.Millisecond)
	return 42
}
