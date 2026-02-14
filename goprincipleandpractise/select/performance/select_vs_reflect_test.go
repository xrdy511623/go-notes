package performance

import "testing"

/*
静态 select vs reflect.Select 性能对比

执行命令:

	go test -run '^$' -bench '^BenchmarkReflect' -benchtime=3s -count=3 -benchmem .

对比维度:
  1. 2-case: 静态 select vs reflect.Select
  2. 4-case: 静态 select vs reflect.Select

结论:
  - reflect.Select 比静态 select 慢 5-10x 以上
  - reflect.Select 每次调用产生额外的内存分配（reflect.Value 装箱等）
  - 仅在 case 数量需要在运行时动态确定时才使用 reflect.Select
  - 静态场景永远优先用编译期 select
*/

func BenchmarkReflectStaticSelect2(b *testing.B) {
	for b.Loop() {
		selectSink = StaticSelect2(1000)
	}
}

func BenchmarkReflectReflectSelect2(b *testing.B) {
	for b.Loop() {
		selectSink = ReflectSelect2(1000)
	}
}

func BenchmarkReflectStaticSelect4(b *testing.B) {
	for b.Loop() {
		selectSink = StaticSelect4(1000)
	}
}

func BenchmarkReflectReflectSelect4(b *testing.B) {
	for b.Loop() {
		selectSink = ReflectSelect4(1000)
	}
}
