package multiparam

import (
	"fmt"
	"strings"
)

// 多参数 Fuzz 示例
//
// Go fuzzing 的 f.Add() 支持多个参数，引擎会独立变异每个参数。
// 支持的类型: string, []byte, int, int8-64, uint, uint8-64, float32, float64, bool, rune

// FormatRecord 根据参数格式化一条记录
// uppercase: 是否大写 name
// repeat: 重复次数（限制在合理范围内）
func FormatRecord(name string, id int, uppercase bool) string {
	if name == "" {
		name = "anonymous"
	}

	// 限制 name 长度，防止内存爆炸
	if len(name) > 1000 {
		name = name[:1000]
	}

	if uppercase {
		name = strings.ToUpper(name)
	}

	// 限制 id 范围
	if id < 0 {
		id = 0
	}
	if id > 999999 {
		id = 999999
	}

	return fmt.Sprintf("[%06d] %s", id, name)
}

// ParseRecord 反向解析 FormatRecord 的输出
func ParseRecord(s string) (name string, id int, err error) {
	if len(s) < 9 || s[0] != '[' || s[7] != ']' || s[8] != ' ' {
		return "", 0, fmt.Errorf("invalid format: %q", s)
	}

	_, err = fmt.Sscanf(s[1:7], "%d", &id)
	if err != nil {
		return "", 0, fmt.Errorf("invalid id: %w", err)
	}

	name = s[9:]
	return name, id, nil
}
