package main

import (
	"testing"
)

/*
结果差异非常明显，GetLastBySlice 耗费了100.14MB 内存，也就是说，申请的100 个1MB大小的内存没有被回收。
因为切片虽然只使用了最后2个元素，但是因为与原来1M的切片引用了相同的底层数组，底层数组得不到释放，因此，
最终100MB的内存始终得不到释放。而GetLastByCopy仅消耗了 3.14 MB 的内存。这是因为，通过copy，指向
了一个新的底层数组，当origin不再被引用后，内存会被垃圾回收(garbage collector, GC)。
如果我们在循环中，显示地调用runtime.GC()，效果会更加地明显:
=== RUN   TestGetLastBySlice
    slice_memory_release.go:43: 100.23 MB
--- PASS: TestGetLastBySlice (0.31s)
=== RUN   TestGetLastByCopy
    slice_memory_release.go:43: 0.23 MB
--- PASS: TestGetLastByCopy (0.24s)
PASS
*/

func TestGetLastBySlice(t *testing.T) { testGetLast(t, GetLastBySlice) }
func TestGetLastByCopy(t *testing.T)  { testGetLast(t, GetLastByCopy) }
