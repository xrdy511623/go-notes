package performance

import (
	"testing"
)

/*
String interning 内存节省对比

执行命令:

	go test -run '^$' -bench 'Intern' -benchmem .

对比维度:
  1. 无驻留: 每个字符串独立分配
  2. sync.Map 驻留: 相同内容共享一份内存

结论:
  - 对于大量重复字符串（如 CSV 字段值、日志级别、HTTP header 名称），
    interning 可以显著减少内存分配次数和总内存占用
  - 首次驻留有 sync.Map 操作的额外开销
  - 命中缓存后，Intern 返回已有字符串，避免重复分配
  - 适用场景: 解析大量结构化数据时的重复字段值
*/

func BenchmarkInternWithPool(b *testing.B) {
	inputs := SimulateRepeatedStrings(10000)
	var interner StringInterner
	b.ResetTimer()
	for b.Loop() {
		for _, s := range inputs {
			internSink = interner.Intern(s)
		}
	}
}

func BenchmarkInternWithoutPool(b *testing.B) {
	inputs := SimulateRepeatedStrings(10000)
	b.ResetTimer()
	for b.Loop() {
		for _, s := range inputs {
			internSink = NoIntern(s)
		}
	}
}
