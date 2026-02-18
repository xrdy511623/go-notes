package noshrink

import (
	"runtime"
	"testing"
)

/*
map 的 bucket 内存只增不减：delete 操作仅清除 key/value 并标记 tophash 为 empty，
不会释放底层 bucket 数组。即使删除全部元素，已分配的 bucket 内存仍然保留。

我们将 val 设为 *int类型，运行程序。
虽然 map 的 bucket 占用内存量依然存在，但 val 改成指针被删掉后，内存占用量确实有所降低。

执行命令:
go test -run TestMapNoShrink -v
=== RUN   TestMapNoShrinkTwo

	no_shrink_two_test.go:32: 空 map 堆内存: 0.66 MB
	no_shrink_two_test.go:37: 填充 100w 后堆内存: 52.81 MB
	no_shrink_two_test.go:47: val使用指针删除全部后堆内存: 45.20 MB ← 内存释放了:7.61MB

--- PASS: TestMapNoShrinkTwo (0.16s)
PASS
*/
func TestMapNoShrinkTwo(t *testing.T) {
	const N = 1_000_000
	// 另一个解决方案是将value改为指针类型
	m1 := make(map[int]*int)
	t.Logf("空 map 堆内存: %.2f MB", heapMB())
	for i := range N {
		m1[i] = &i
	}
	afterFill := heapMB()
	t.Logf("填充 %dw 后堆内存: %.2f MB", N/10_000, afterFill)

	for i := range N {
		delete(m1, i)
	}

	// 必须使用 KeepAlive 确保 GC 不会提前回收 m 的内部存储
	runtime.KeepAlive(m1)
	afterDelete := heapMB()
	runtime.KeepAlive(m1)
	t.Logf("val使用指针删除全部后堆内存: %.2f MB ← 内存释放了:%.2fMB", afterDelete, afterFill-afterDelete)
}
