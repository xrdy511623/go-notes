package performance

import (
	"sync"
)

var internSink string

// StringInterner 使用 sync.Map 实现的字符串驻留池
// 对于重复出现的字符串，共享同一份内存而不是每次创建新副本
type StringInterner struct {
	pool sync.Map
}

// Intern 返回与 s 内容相同的驻留字符串
// 如果池中已有相同内容的字符串，则返回池中的版本（共享内存）
func (si *StringInterner) Intern(s string) string {
	if v, ok := si.pool.Load(s); ok {
		return v.(string)
	}
	si.pool.Store(s, s)
	return s
}

// NoIntern 不做任何驻留，直接返回（基准对照）
func NoIntern(s string) string {
	return s
}

// SimulateRepeatedStrings 模拟生成重复字符串的场景
// 例如从 CSV/JSON 中读取的字段名或枚举值
func SimulateRepeatedStrings(n int) []string {
	values := []string{"pending", "active", "inactive", "deleted", "archived"}
	result := make([]string, n)
	for i := range n {
		// 模拟从外部输入获取字符串（每次都是新的 string）
		result[i] = string([]byte(values[i%len(values)]))
	}
	return result
}
