// Package main 演示错误响应格式不一致的反模式。
//
// ❌ 错误: 不同接口返回不同格式的错误
//
//	{"error": "not found"}                    ← 字符串
//	{"code": 404, "msg": "user not found"}    ← 数字码 + msg
//	{"errors": ["invalid email"]}             ← 数组
//
// ✅ 正确: 统一错误信封
//
//	{"error": {"code": "not_found", "message": "user not found"}}
package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	fmt.Println("=== 错误响应格式不一致反模式 ===")
	fmt.Println()

	// ❌ 反模式: 三个接口三种错误格式
	fmt.Println("❌ 反模式 — 不一致的错误格式:")
	badErrors := []any{
		map[string]string{"error": "not found"},
		map[string]any{"code": 404, "msg": "user not found"},
		map[string][]string{"errors": {"invalid email", "name too short"}},
	}
	for i, e := range badErrors {
		data, _ := json.MarshalIndent(e, "  ", "  ")
		fmt.Printf("  接口 %d: %s\n", i+1, data)
	}

	fmt.Println()
	fmt.Println("  问题:")
	fmt.Println("  • 客户端需要为每个接口写不同的错误解析逻辑")
	fmt.Println("  • 无法统一错误处理中间件")
	fmt.Println("  • 前端需要 if/else 判断错误格式")

	// ✅ 正确模式: 统一错误信封
	fmt.Println()
	fmt.Println("✅ 正确模式 — 统一错误信封:")

	type ErrorBody struct {
		Code    string            `json:"code"`
		Message string            `json:"message"`
		Detail  string            `json:"detail,omitempty"`
		Fields  map[string]string `json:"fields,omitempty"`
	}
	type ErrorResponse struct {
		Error ErrorBody `json:"error"`
	}

	goodErrors := []ErrorResponse{
		{Error: ErrorBody{Code: "not_found", Message: "user not found"}},
		{Error: ErrorBody{Code: "validation_failed", Message: "request validation failed",
			Fields: map[string]string{"email": "invalid format", "name": "too short"}}},
		{Error: ErrorBody{Code: "unauthorized", Message: "invalid token"}},
	}

	for _, e := range goodErrors {
		data, _ := json.MarshalIndent(e, "  ", "  ")
		fmt.Printf("  %s\n", data)
	}

	fmt.Println()
	fmt.Println("  优点:")
	fmt.Println("  • 客户端只需一套错误解析逻辑")
	fmt.Println("  • 字段级错误通过 fields 字段传递")
	fmt.Println("  • 可以统一错误监控和告警")
}
