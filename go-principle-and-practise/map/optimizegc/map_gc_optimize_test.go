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

// TestSmallBatchGCDuration 测试小规模数据gc时长
func TestSmallBatchGCDuration(t *testing.T) {
	size := 1000
	m := GenerateStringMap(size)
	runtime.GC()
	gcCost := timeGC()
	t.Logf("size %d GC duration: %v\n", size, gcCost)
	_ = m["1"]
}

// TestBigBatchGCDuration 测试大规模数据gc时长
func TestBigBatchGCDuration(t *testing.T) {
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
	// 计算总的GC时长
	gcCost := time.Since(gcStartTime)
	return gcCost
}
