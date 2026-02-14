package set

import "strconv"

// map[string]bool vs map[string]struct{} 作为 Set 的性能对比
//
// 空结构体 struct{} 占 0 字节，bool 占 1 字节。
// 当 map 元素数量很大时，value 的内存差异会体现在总内存占用上。

type Set interface {
	Has(string) bool
	Add(string)
	Delete(string)
}

type boolSet map[string]bool

type emptyStructSet map[string]struct{}

func (s emptyStructSet) Has(key string) bool {
	_, ok := s[key]
	return ok
}

func (s emptyStructSet) Add(key string) {
	s[key] = struct{}{}
}

func (s emptyStructSet) Delete(key string) {
	delete(s, key)
}

func (s boolSet) Has(key string) bool {
	_, ok := s[key]
	return ok
}

func (s boolSet) Add(key string) {
	s[key] = true
}

func (s boolSet) Delete(key string) {
	delete(s, key)
}

// RunSetBenchmark 对 Set 执行 n 次 Add+Has+Delete 操作
func RunSetBenchmark(n int, s Set) {
	for i := range n {
		key := strconv.Itoa(i)
		s.Add(key)
		if s.Has(key) {
			s.Delete(key)
		}
	}
}
