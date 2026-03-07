package multiparam

import "testing"

/*
多参数 Fuzz 示例

执行命令:

	go test -fuzz=FuzzFormatRecord -fuzztime=10s

Go fuzzing f.Add() 支持的参数类型:
  string, []byte, bool, byte, rune,
  int, int8, int16, int32, int64,
  uint, uint8, uint16, uint32, uint64,
  float32, float64

注意:
  - f.Add() 的参数类型和数量必须与 f.Fuzz(func(t, ...) 完全匹配
  - 不支持 struct、slice（除 []byte）、map 等复合类型
  - 引擎会独立变异每个参数
*/

func FuzzFormatRecord(f *testing.F) {
	// 多参数种子: (string, int, bool)
	f.Add("Alice", 1, false)
	f.Add("Bob", 999999, true)
	f.Add("", 0, false)
	f.Add("中文名字", 42, true)
	f.Add("\x00\xff", -1, false) // 特殊字节

	f.Fuzz(func(t *testing.T, name string, id int, uppercase bool) {
		formatted := FormatRecord(name, id, uppercase)

		// 不变性1: 输出不应为空
		if formatted == "" {
			t.Error("FormatRecord returned empty string")
		}

		// 不变性2: 输出应该可以被 ParseRecord 解析（round-trip）
		parsedName, parsedID, err := ParseRecord(formatted)
		if err != nil {
			t.Fatalf("ParseRecord(%q) failed: %v", formatted, err)
		}

		// 不变性3: 解析后的 id 在合法范围内
		if parsedID < 0 || parsedID > 999999 {
			t.Errorf("parsed id %d out of range", parsedID)
		}

		// 不变性4: 解析后的 name 非空
		if parsedName == "" {
			t.Error("parsed name is empty")
		}
	})
}
