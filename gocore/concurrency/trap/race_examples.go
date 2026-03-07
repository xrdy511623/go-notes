package trap

import "sync"

// RaceCounter 未保护的共享计数器——典型数据竞争
// 用 go test -race 可检测到
func RaceCounter() int {
	counter := 0
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter++ // 数据竞争：多个goroutine并发读写counter
		}()
	}

	wg.Wait()
	return counter // 结果不确定，可能小于100
}

// RaceMap map并发读写——fatal error，无法recover
// 注意：这个函数可能导致程序直接崩溃，仅作演示
func RaceMap() {
	m := make(map[string]int)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func(v int) {
			defer wg.Done()
			m["key"] = v // 并发写
		}(i)
		go func() {
			defer wg.Done()
			_ = m["key"] // 并发读
		}()
	}

	wg.Wait()
}

// RaceSliceAppend slice并发append——数据竞争
func RaceSliceAppend() []int {
	var s []int
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			s = append(s, v) // 数据竞争：append修改slice header
		}(i)
	}

	wg.Wait()
	return s // 长度不确定，可能有数据丢失
}

// SafeSliceIndex 安全写法：预分配+索引写入
func SafeSliceIndex() []int {
	s := make([]int, 100)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			s[idx] = idx // 各写各的索引，无竞争
		}(i)
	}

	wg.Wait()
	return s
}
