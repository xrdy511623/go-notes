package fuzzingtest

import "testing"

/*
基础 Fuzz 示例：ParseAge 边界 Bug 发现

执行命令（运行 5 秒）:

	go test -fuzz=FuzzParseAge -fuzztime=5s

预期：
  Fuzzing 引擎会自动发现输入 "150" 导致 ParseAge 错误地返回成功（应该报错）。
  crashing input 会被保存到 testdata/fuzz/FuzzParseAge/ 目录中。

重现失败:

	go test -run=FuzzParseAge/1b484383a67174f3
*/

func FuzzParseAge(f *testing.F) {
	// 种子语料：覆盖正常值、边界值、非法值
	f.Add("0")
	f.Add("1")
	f.Add("149")      // 上界（应合法）
	f.Add("-1")       // 负数（应报错）
	f.Add("abc")      // 非数字（应报错）
	f.Add("")         // 空字符串（应报错）
	f.Add("1000")     // 超大值（应报错）
	f.Add("\x80test") // 非法 UTF-8

	f.Fuzz(func(t *testing.T, ageStr string) {
		age, err := ParseAge(ageStr)
		if err != nil {
			return // ParseAge 认为输入无效，符合预期
		}

		// 不变性检查：ParseAge 返回成功时，age 必须在 [0, 149] 范围内
		// 如果 ParseAge 有 Bug 导致 150 也被接受，这里会捕获
		if age < 0 || age >= 150 {
			t.Errorf("ParseAge(%q) = %d, want 0 <= age < 150", ageStr, age)
		}
	})
}
