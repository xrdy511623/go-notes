// Package main 演示内部错误泄漏的反模式。
//
// ❌ 错误: 将内部错误细节直接返回给客户端
//   - 泄漏数据库表名、SQL 语句、文件路径
//   - 暴露技术栈信息（Go 版本、框架版本）
//   - 为攻击者提供有价值的信息
//
// ✅ 正确: 返回通用错误消息，内部细节只记录日志
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func main() {
	fmt.Println("=== 内部错误泄漏反模式 ===")
	fmt.Println()

	// 模拟一个数据库错误
	dbErr := fmt.Errorf("pq: relation \"users\" does not exist (SQLSTATE 42P01)")

	// ❌ 反模式: 直接暴露内部错误
	fmt.Println("❌ 反模式 — 泄漏内部错误:")
	badResp := map[string]any{
		"error":      dbErr.Error(),
		"stackTrace": "goroutine 1 [running]:\nmain.handleRequest()\n\t/app/internal/handler/user.go:42",
		"version":    "go1.24.0",
	}
	badJSON, _ := json.MarshalIndent(badResp, "  ", "  ")
	fmt.Printf("  %s\n", badJSON)
	fmt.Println()
	fmt.Println("  问题:")
	fmt.Println("  • 泄漏了数据库类型 (PostgreSQL) 和表名 (users)")
	fmt.Println("  • 泄漏了代码文件路径和行号")
	fmt.Println("  • 泄漏了 Go 版本信息")
	fmt.Println("  • 攻击者可以利用这些信息进行针对性攻击")

	// ✅ 正确模式: 通用错误消息 + 内部日志
	fmt.Println()
	fmt.Println("✅ 正确模式 — 安全的错误响应:")

	type ErrorBody struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	type ErrorResponse struct {
		Error ErrorBody `json:"error"`
	}

	// 返回给客户端的通用响应
	goodResp := ErrorResponse{
		Error: ErrorBody{
			Code:    "internal_error",
			Message: "an unexpected error occurred, please try again later",
		},
	}
	goodJSON, _ := json.MarshalIndent(goodResp, "  ", "  ")
	fmt.Printf("  客户端收到: %s\n", goodJSON)

	// 详细错误只记录到服务端日志
	logger := log.New(os.Stdout, "  [LOG] ", 0)
	logger.Printf("internal error: %v | request_id=req_abc123 | path=/api/v1/users | method=GET", dbErr)

	fmt.Println()
	fmt.Println("  要点:")
	fmt.Println("  • 客户端只看到通用错误码和消息")
	fmt.Println("  • 通过 request_id 关联客户端请求与服务端日志")
	fmt.Println("  • 详细的错误信息仅存在于服务端日志中")
	fmt.Println("  • 可以在内部监控系统中设置告警")
}
