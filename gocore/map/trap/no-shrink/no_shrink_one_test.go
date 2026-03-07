package noshrink

import (
	"runtime"
	"testing"
)

/*
map 的 bucket 内存只增不减：delete 操作仅清除 key/value 并标记 tophash 为 empty，
不会释放底层 bucket 数组。即使删除全部元素，已分配的 bucket 内存仍然保留。

生产环境影响：如果 map 曾经存储过大量数据后又大量删除，这些"空桶"内存不会归还，
可能造成内存浪费。解决方案是用新 map 替换旧 map，让旧 map 被 GC 回收。

执行命令:
go test -run TestMapNoShrink -v
=== RUN   TestMapNoShrink

	no_shrink_one_test.go:23: 空 map 堆内存: 0.64 MB
	no_shrink_one_test.go:29: 填充 100w 后堆内存: 42.69 MB
	no_shrink_one_test.go:39: 删除全部后堆内存: 42.69 MB ← 内存未释放
	no_shrink_one_test.go:50: 新建 map 后堆内存: 0.64 MB ← 旧 map 被 GC 回收

--- PASS: TestMapNoShrink (0.13s)
PASS
ok      go-notes/goprincipleandpractise/map/trap/no-shrink      0.600s
*/
func TestMapNoShrink(t *testing.T) {
	const N = 1_000_000

	m := make(map[int]int)
	t.Logf("空 map 堆内存: %.2f MB", heapMB())

	for i := range N {
		m[i] = i
	}
	afterFill := heapMB()
	t.Logf("填充 %dw 后堆内存: %.2f MB", N/10_000, afterFill)

	for i := range N {
		delete(m, i)
	}

	// 必须使用 KeepAlive 确保 GC 不会提前回收 m 的内部存储
	runtime.KeepAlive(m)
	afterDelete := heapMB()
	runtime.KeepAlive(m)
	t.Logf("删除全部后堆内存: %.2f MB ← 内存未释放", afterDelete)

	// 验证：删除后内存不应大幅下降
	if afterDelete < afterFill*0.5 {
		t.Errorf("预期删除后内存不会大幅下降, 但从 %.2f MB 降到了 %.2f MB", afterFill, afterDelete)
	}

	// 解决方案：用新 map 替换旧 map，让旧 map 被 GC 回收
	m = make(map[int]int)
	afterNew := heapMB()
	runtime.KeepAlive(m)
	t.Logf("新建 map 后堆内存: %.2f MB ← 旧 map 被 GC 回收", afterNew)

}

func heapMB() float64 {
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return float64(ms.HeapInuse) / 1024 / 1024
}
