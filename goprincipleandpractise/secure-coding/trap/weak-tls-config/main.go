package main

import (
	"crypto/tls"
	"fmt"
)

/*
陷阱：不安全的 TLS 配置

运行：go run .

预期行为：
  InsecureSkipVerify = true 会跳过服务端证书验证，使连接容易被中间人攻击（MITM）。
  MinVersion 设为 TLS 1.0/1.1 允许使用已知有漏洞的协议版本。

  Go 的 crypto/tls 默认配置已经足够安全（TLS 1.2+，验证证书），
  但开发者常因调试方便或对接旧系统而降低安全级别，然后忘记改回来。

  正确做法：
  - 永远不要在生产环境使用 InsecureSkipVerify = true
  - MinVersion 至少设为 tls.VersionTLS12（Go 默认值）
  - 推荐 MinVersion = tls.VersionTLS13（如客户端都支持）
  - 生产环境使用 CipherSuites 白名单限制弱密码套件
*/

func main() {
	fmt.Println("=== 错误做法 1：InsecureSkipVerify ===")
	badSkipVerify()

	fmt.Println("\n=== 错误做法 2：允许 TLS 1.0 ===")
	badMinVersion()

	fmt.Println("\n=== 正确做法：生产推荐配置 ===")
	goodConfig()

	fmt.Println("\n=== TLS 版本对照表 ===")
	printVersionTable()

	fmt.Println("\n总结:")
	fmt.Println("  1. 绝不在生产环境使用 InsecureSkipVerify = true")
	fmt.Println("  2. MinVersion 至少 TLS 1.2（Go 默认），推荐 TLS 1.3")
	fmt.Println("  3. Go 默认 TLS 配置已足够安全，不要画蛇添足降级")
	fmt.Println("  4. gosec G402 检测 InsecureSkipVerify，G302 检测弱 TLS 版本")
	fmt.Println("  5. 调试完毕后必须移除 InsecureSkipVerify")
}

func badSkipVerify() {
	cfg := &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // 故意演示反例
	}
	fmt.Printf("  InsecureSkipVerify = %v\n", cfg.InsecureSkipVerify)
	fmt.Println("  风险：跳过证书验证，任何人都可以冒充服务器（中间人攻击）")
	fmt.Println("  常见借口：'测试环境用的' → 最终上线忘记改回来")
}

func badMinVersion() {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS10, //nolint:gosec // 故意演示反例
	}
	fmt.Printf("  MinVersion = 0x%04x (TLS 1.0)\n", cfg.MinVersion)
	fmt.Println("  风险：TLS 1.0 存在 BEAST、POODLE 等已知漏洞")
	fmt.Println("  TLS 1.1 也已被废弃（RFC 8996, 2021）")
}

func goodConfig() {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		// TLS 1.3 密码套件由 Go 自动选择，无需（也不能）手动指定
	}
	fmt.Printf("  MinVersion = 0x%04x (TLS 1.2)\n", cfg.MinVersion)
	fmt.Printf("  InsecureSkipVerify = %v (默认 false)\n", cfg.InsecureSkipVerify)
	fmt.Printf("  CipherSuites: %d 个 AEAD 密码套件\n", len(cfg.CipherSuites))
	fmt.Println("  Go 默认已不支持 TLS 1.0/1.1，不设 MinVersion 也安全")
}

func printVersionTable() {
	versions := []struct {
		name   string
		value  uint16
		status string
	}{
		{"TLS 1.0", tls.VersionTLS10, "已废弃 ❌ BEAST/POODLE 漏洞"},
		{"TLS 1.1", tls.VersionTLS11, "已废弃 ❌ RFC 8996 (2021)"},
		{"TLS 1.2", tls.VersionTLS12, "安全 ✅ Go 默认最低版本"},
		{"TLS 1.3", tls.VersionTLS13, "最安全 ✅ 推荐，更快握手"},
	}
	for _, v := range versions {
		fmt.Printf("  0x%04x  %s  %s\n", v.value, v.name, v.status)
	}
}
