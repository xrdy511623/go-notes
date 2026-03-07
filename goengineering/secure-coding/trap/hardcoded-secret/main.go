package main

import (
	"fmt"
	"os"
)

/*
陷阱：硬编码密钥

运行：go run .

预期行为：
  将 API 密钥、数据库密码等敏感信息直接写在源代码中，一旦代码提交到版本控制，
  所有有仓库访问权限的人（包括公开仓库的所有人）都能看到这些密钥。
  即使后来删除，git 历史中仍然保留。

  正确做法：
  - 使用环境变量（os.Getenv / os.LookupEnv）
  - 使用 .env 文件 + .gitignore
  - 生产环境使用密钥管理服务（Vault、AWS Secrets Manager 等）
  - 启动时 fail-fast：缺少必要密钥则立即退出

  gosec G101 规则可检测疑似硬编码密钥。
*/

func main() {
	fmt.Println("=== 错误做法：硬编码密钥 ===")
	badExample()

	fmt.Println("\n=== 正确做法：环境变量 + fail-fast ===")
	goodExample()

	fmt.Println("\n总结:")
	fmt.Println("  1. 绝不在源代码中硬编码密钥、密码、token")
	fmt.Println("  2. 使用 os.LookupEnv 读取环境变量，缺失时 fail-fast")
	fmt.Println("  3. .env 文件必须加入 .gitignore")
	fmt.Println("  4. gosec G101 可自动检测疑似硬编码密钥")
	fmt.Println("  5. 如果密钥已经提交到 git，必须立即轮换（rotation）")
}

func badExample() {
	// 错误：硬编码密钥
	const apiKey = "sk-proj-abc123def456" //nolint:gosec // 故意演示反例
	const dbPassword = "super_secret_pwd" //nolint:gosec // 故意演示反例

	fmt.Printf("  API Key: %s (直接写在代码中！)\n", apiKey)
	fmt.Printf("  DB Password: %s (提交到 git 后所有人可见！)\n", dbPassword)
	fmt.Println("  风险：")
	fmt.Println("    - 代码推到 GitHub 后密钥公开泄漏")
	fmt.Println("    - 即使删除代码，git 历史仍保留")
	fmt.Println("    - 所有 clone 过仓库的人都有密钥副本")
}

func goodExample() {
	// 正确：从环境变量读取，缺失时 fail-fast
	requiredEnvs := []string{"API_KEY", "DB_PASSWORD"}

	for _, env := range requiredEnvs {
		value, ok := os.LookupEnv(env)
		if !ok {
			fmt.Printf("  %s: 未设置（生产环境应 log.Fatalf 退出）\n", env)
			continue
		}
		// 打印时脱敏
		masked := maskSecret(value)
		fmt.Printf("  %s: %s (已脱敏)\n", env, masked)
	}

	fmt.Println("\n  推荐的 fail-fast 模式：")
	fmt.Println("    apiKey, ok := os.LookupEnv(\"API_KEY\")")
	fmt.Println("    if !ok {")
	fmt.Println("        log.Fatal(\"API_KEY environment variable is required\")")
	fmt.Println("    }")
}

// maskSecret 脱敏：只显示前 4 位和后 4 位
func maskSecret(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}
