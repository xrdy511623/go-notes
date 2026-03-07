package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

/*
陷阱：日志泄漏敏感信息

运行：go run .

预期行为：
  使用 %+v 或 %v 打印结构体时，所有字段（包括密码、token 等）都会输出。
  日志通常会被收集到 ELK/Splunk 等系统，运维人员、开发者都能看到。

  正确做法：
  - 实现 fmt.Stringer 接口，控制打印输出
  - 使用 json:"-" 标签排除敏感字段的 JSON 序列化
  - 日志中使用脱敏函数处理敏感数据
  - 错误信息不要包含内部实现细节
*/

// BadUser 没有任何保护，所有字段直接暴露
type BadUser struct {
	Name     string
	Email    string
	Password string
	Token    string
}

// GoodUser 实现了 Stringer 接口 + json 标签保护
type GoodUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"-"` // JSON 序列化时排除
	Token    string `json:"-"` // JSON 序列化时排除
}

// String 控制 fmt.Print 系列函数的输出
func (u GoodUser) String() string {
	return fmt.Sprintf("User{Name: %s, Email: %s}", u.Name, maskEmail(u.Email))
}

func main() {
	logger := log.New(os.Stdout, "[APP] ", log.LstdFlags)

	bad := BadUser{
		Name:     "张三",
		Email:    "zhangsan@example.com",
		Password: "super_secret_123",
		Token:    "eyJhbGciOiJIUzI1NiJ9.secret.payload",
	}

	good := GoodUser{
		Name:     "张三",
		Email:    "zhangsan@example.com",
		Password: "super_secret_123",
		Token:    "eyJhbGciOiJIUzI1NiJ9.secret.payload",
	}

	fmt.Println("=== 错误做法：直接打印结构体 ===")
	logger.Printf("用户登录: %+v", bad) // 密码和 token 全部暴露！
	fmt.Println()

	fmt.Printf("  fmt.Sprintf(\"%%+v\"): %+v\n", bad)
	fmt.Println("  问题：Password 和 Token 明文出现在日志中")

	badJSON, _ := json.Marshal(bad)
	fmt.Printf("  json.Marshal: %s\n", badJSON)
	fmt.Println("  问题：JSON 也包含所有敏感字段")

	fmt.Println("\n=== 正确做法：Stringer + json:\"-\" ===")
	logger.Printf("用户登录: %s", good) // 只显示脱敏后的信息
	fmt.Println()

	fmt.Printf("  fmt.Sprintf(\"%%s\"): %s\n", good)
	fmt.Printf("  fmt.Sprintf(\"%%v\"): %v\n", good)

	goodJSON, _ := json.Marshal(good)
	fmt.Printf("  json.Marshal: %s\n", goodJSON)
	fmt.Println("  Password 和 Token 不会出现在 JSON 中")

	fmt.Println("\n=== 错误信息安全 ===")
	fmt.Println("  错误做法：")
	fmt.Println("    return fmt.Errorf(\"数据库连接失败: host=prod-db:5432 user=admin password=xxx\")")
	fmt.Println("  正确做法：")
	fmt.Println("    return fmt.Errorf(\"数据库连接失败\", err)  // 不附加连接详情")
	fmt.Println("    log.Error(\"db连接失败\", \"host\", cfg.Host)   // 服务端日志记详情（无密码）")

	fmt.Println("\n总结:")
	fmt.Println("  1. 永远不要用 Sprintf(+v) 打印包含敏感字段的结构体")
	fmt.Println("  2. 实现 fmt.Stringer 接口控制打印输出")
	fmt.Println("  3. 敏感字段加 json:\"-\" 防止 JSON 泄漏")
	fmt.Println("  4. 错误信息面向用户时不暴露内部实现细节")
	fmt.Println("  5. 日志中使用脱敏函数处理邮箱、手机号等 PII")
}

// maskEmail 邮箱脱敏：zhangsan@example.com → zh****@example.com
func maskEmail(email string) string {
	for i, c := range email {
		if c == '@' {
			if i <= 2 {
				return "****" + email[i:]
			}
			return email[:2] + "****" + email[i:]
		}
	}
	return "****"
}
