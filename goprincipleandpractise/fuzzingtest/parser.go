package fuzzingtest

import (
	"fmt"
	"strconv"
)

// ParseAge 将字符串解析为年龄（整数），期望范围 0-149。
// 注意：此处故意使用 age > 150 而非 age > 149，留下一个边界 Bug 供 Fuzzing 发现。
func ParseAge(ageStr string) (int, error) {
	if ageStr == "" {
		return 0, fmt.Errorf("age string cannot be empty")
	}
	age, err := strconv.Atoi(ageStr)
	if err != nil {
		return 0, fmt.Errorf("not a valid integer: %w", err)
	}
	if age < 0 || age > 150 { // Bug: 应该是 age > 149，导致 150 被错误接受
		return 0, fmt.Errorf("age %d out of reasonable range (0-149)", age)
	}
	return age, nil
}
