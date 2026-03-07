package nomultistage

/*
陷阱：不使用多阶段构建

问题说明：
  单阶段 Dockerfile 把编译和运行放在同一个镜像中：

  FROM golang:1.24
  WORKDIR /app
  COPY . .
  RUN go build -o server .
  CMD ["./server"]

  最终镜像包含：
  1. Go 编译器（不需要）
  2. 项目源码（安全风险：可能包含配置、密钥）
  3. 测试文件和测试数据
  4. go module 依赖源码
  5. 编译中间产物

  安全风险：
  - 攻击者如果突破应用，可以看到全部源码
  - 可以用内置的编译器编译恶意程序
  - 测试配置中的 mock 密钥可能被利用

正确做法：多阶段构建

  FROM golang:1.24-alpine AS builder
  WORKDIR /app
  COPY go.mod go.sum ./
  RUN go mod download
  COPY . .
  RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o server .

  FROM gcr.io/distroless/static:nonroot
  COPY --from=builder /app/server /server
  USER nonroot:nonroot
  ENTRYPOINT ["/server"]

  最终镜像只包含编译好的二进制，没有源码、编译器、测试文件。
*/

import "fmt"

// SecurityRisk 描述不使用多阶段构建的安全风险
type SecurityRisk struct {
	Risk        string
	Description string
	Severity    string
}

// ListSecurityRisks 列出不使用多阶段构建的安全风险
func ListSecurityRisks() []SecurityRisk {
	return []SecurityRisk{
		{
			Risk:        "源码泄露",
			Description: "攻击者突破应用后可以阅读全部源码，找到更多攻击入口",
			Severity:    "HIGH",
		},
		{
			Risk:        "编译器可用",
			Description: "攻击者可以使用镜像内的 Go 编译器编译恶意工具",
			Severity:    "HIGH",
		},
		{
			Risk:        "敏感配置泄露",
			Description: "测试配置、mock 密钥、.env.example 等可能包含敏感信息",
			Severity:    "CRITICAL",
		},
		{
			Risk:        "攻击面扩大",
			Description: "更多的系统工具和库意味着更多可利用的漏洞",
			Severity:    "MEDIUM",
		},
	}
}

// PrintRisks 打印安全风险
func PrintRisks() {
	fmt.Println("=== 不使用多阶段构建的安全风险 ===")
	for _, risk := range ListSecurityRisks() {
		fmt.Printf("\n[%s] %s\n", risk.Severity, risk.Risk)
		fmt.Printf("  %s\n", risk.Description)
	}
}
