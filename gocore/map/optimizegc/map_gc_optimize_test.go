package optimizegc

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

/*
当 map 中缓存的数据比较多时，为了避免 GC 开销，我们可以将 map 中的 key-value 类型设计成非指针类型
且大小不超过 128 字节，从而避免 GC 扫描。

 go test -v
=== RUN   TestSmallBatchStringGCDuration
    map_gc_optimize_test.go:34: size 1000 GC duration: 258.417µs
--- PASS: TestSmallBatchStringGCDuration (0.00s)
=== RUN   TestBigBatchStringGCDuration
    map_gc_optimize_test.go:44: size 5000000 GC duration: 44.597333ms
--- PASS: TestBigBatchStringGCDuration (1.46s)
=== RUN   TestBigBatchIntGCDuration
    map_gc_optimize_test.go:65: size 5000000 GC duration: 471.958µs
--- PASS: TestBigBatchIntGCDuration (0.40s)
=== RUN   TestSmallStruct
    map_gc_optimize_test.go:80: size 5000000 GC duration: 859.709µs
--- PASS: TestSmallStruct (0.66s)
=== RUN   TestBigStruct
    map_gc_optimize_test.go:95: size 5000000 GC duration: 44.831416ms
--- PASS: TestBigStruct (0.77s)
PASS
ok      go-notes/goprincipleandpractise/map/optimizegc  3.551s

为什么使用int类型作为键值对的map性能会好很多呢？
因为：
- int 是纯值类型，无指针
- 如果 map 只包含非指针类型，Go GC 根本不会扫描它
- 只会扫描 map 结构本身（几个 page）
几乎没有扫描成本。
所以对比结果特别惊人：
map[string]string:   44.59ms
map[int]int:         0.5 ms
性能差了 88倍以上！
这是 Go 官方推荐避免指针类型 key/value 的根本原因。

为什么结构体大小达到128字节后，明明只增加了1字节，GC压力便大幅飙升？

因为超过 128 字节后：
- Go 会把 struct 当作 大对象
- 它会逃逸到堆上（escape to heap）
- GC 必须扫描每个 struct 的完整 129 字节
- 5M 个 struct = 645 MB 的扫描量
所以立刻拉爆 GC 开销。
*/

func GenerateStringMap(size int) map[string]string {
	// 在这里执行一些可能会触发GC的操作，例如创建大量对象等
	// 以下示例创建一个较大的map并填充数据
	m := make(map[string]string)
	for i := 0; i < size; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("val_%d", i)
		m[key] = value

	}
	return m
}

// TestSmallBatchStringGCDuration 测试小规模数据gc时长
func TestSmallBatchStringGCDuration(t *testing.T) {
	size := 1000
	m := GenerateStringMap(size)
	runtime.GC()
	gcCost := timeGC()
	t.Logf("size %d GC duration: %v\n", size, gcCost)
	_ = m["1"]
}

// TestBigBatchStringGCDuration 测试大规模数据gc时长
func TestBigBatchStringGCDuration(t *testing.T) {
	size := 5000000
	m := GenerateStringMap(size)
	runtime.GC()
	gcCost := timeGC()
	t.Logf("size %d GC duration: %v\n", size, gcCost)
	_ = m["1"]
}

func GenerateIntMap(size int) map[int]int {
	// 在这里执行一些可能会触发GC的操作，例如创建大量对象等
	// 以下示例创建一个较大的map并填充数据
	m := make(map[int]int)
	for i := 0; i < size; i++ {
		m[i] = i

	}
	return m
}

// 测试key-value非指针类型,int的gc开销
func TestBigBatchIntGCDuration(t *testing.T) {
	size := 5000000
	m := GenerateIntMap(size)
	runtime.GC()
	gcCost := timeGC()
	t.Logf("size %d GC duration: %v\n", size, gcCost)
	_ = m[1]
}

func TestSmallStruct(t *testing.T) {
	type SmallStruct struct {
		data [128]byte
	}
	m := make(map[int]SmallStruct)
	size := 5000000
	for i := 0; i < size; i++ {
		m[i] = SmallStruct{}
	}
	runtime.GC()
	gcCost := timeGC()
	t.Logf("size %d GC duration: %v\n", size, gcCost)
	_ = m[1]
}

func TestBigStruct(t *testing.T) {
	type BigStruct struct {
		data [129]byte
	}
	m := make(map[int]BigStruct)
	size := 5000000
	for i := 0; i < size; i++ {
		m[i] = BigStruct{}
	}
	runtime.GC()
	gcCost := timeGC()
	t.Logf("size %d GC duration: %v\n", size, gcCost)
	_ = m[1]
}

func timeGC() time.Duration {
	// 记录GC开始时间
	gcStartTime := time.Now()
	// 手动触发GC，以便更准确地测量此次操作相关的GC时长
	runtime.GC()
	// 计算总的GC时长s
	gcCost := time.Since(gcStartTime)
	return gcCost
}
