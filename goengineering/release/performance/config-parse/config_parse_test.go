package configparse

import "testing"

/*
基准测试：不同配置格式的解析性能

运行方式：
  go test -bench=. -benchmem -benchtime=3s .

预期结果：
  BenchmarkParseJSON-8        xxx    yyy ns/op    zzz B/op    n allocs/op
  BenchmarkParseKeyValue-8    xxx    yyy ns/op    zzz B/op    n allocs/op
  BenchmarkParseEnvVars-8     xxx    yyy ns/op    zzz B/op    n allocs/op

  JSON 标准库高度优化，通常是最快的结构化格式。
  环境变量（直接 map 查找）最快，但不支持嵌套。
*/

func BenchmarkParseJSON(b *testing.B) {
	data := SampleJSON()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseJSON(data)
	}
}

func BenchmarkParseKeyValue(b *testing.B) {
	data := SampleYAMLLike()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseKeyValue(data)
	}
}

func BenchmarkParseEnvVars(b *testing.B) {
	envs := map[string]string{
		"SERVER_HOST":  "0.0.0.0",
		"SERVER_PORT":  "8080",
		"DB_HOST":      "localhost",
		"DB_PORT":      "5432",
		"DB_MAX_CONNS": "20",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseEnvVarsDirect(envs)
	}
}
