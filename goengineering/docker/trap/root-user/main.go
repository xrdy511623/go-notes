package rootuser

/*
陷阱：以 root 用户运行容器

问题说明：
  Docker 容器默认以 root 用户运行。如果应用被攻破，
  攻击者获得的是 root 权限：

  1. 可以读写容器内所有文件
  2. 可以安装恶意软件（如果有包管理器）
  3. 如果容器配置不当（如 --privileged），可能逃逸到宿主机
  4. 可以修改网络配置，进行中间人攻击

  这不是理论风险——容器逃逸漏洞定期被发现（CVE-2019-5736 等）。

正确做法：

  # 方式 1：使用 distroless 的 nonroot 变体（推荐）
  FROM gcr.io/distroless/static:nonroot
  COPY --from=builder /app/server /server
  USER nonroot:nonroot
  ENTRYPOINT ["/server"]

  # 方式 2：在 alpine 中创建专用用户
  FROM alpine:3.19
  RUN addgroup -S appgroup && adduser -S appuser -G appgroup
  COPY --from=builder /app/server /server
  USER appuser:appgroup
  ENTRYPOINT ["/server"]

  # 方式 3：在 scratch 中指定 UID（65534 = nobody）
  FROM scratch
  COPY --from=builder /app/server /server
  USER 65534:65534
  ENTRYPOINT ["/server"]

额外加固：
  # docker-compose.yml
  services:
    app:
      read_only: true           # 只读文件系统
      security_opt:
        - no-new-privileges:true  # 禁止提权
      cap_drop:
        - ALL                    # 丢弃所有 capabilities
*/

import (
	"fmt"
	"net/http"
)

// VulnerableServer 模拟一个以 root 运行的 HTTP 服务
// 如果被攻破，攻击者拥有 root 权限
func VulnerableServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Running as root - this is dangerous!")
	})

	// 以 root 运行意味着：
	// 1. 可以绑定任何端口（包括 1-1024 的特权端口）
	// 2. 可以读写 /etc/passwd 等系统文件
	// 3. 容器逃逸后直接获得宿主机 root
	fmt.Println("WARNING: Server running as root!")
	fmt.Println("If compromised, attacker gets root access.")
}

// RootUserRisks 列出以 root 运行的风险
func RootUserRisks() []string {
	return []string{
		"容器逃逸后获得宿主机 root 权限",
		"可读写所有容器内文件",
		"可安装恶意软件",
		"可修改网络配置",
		"可访问挂载的卷中的敏感数据",
		"可以 kill 其他进程",
	}
}

// NonRootBenefits 列出使用非 root 用户的好处
func NonRootBenefits() []string {
	return []string{
		"限制攻击者可执行的操作",
		"符合最小权限原则",
		"满足安全合规要求（CIS Benchmark）",
		"减少容器逃逸的影响范围",
	}
}

// PrintComparison 打印 root vs non-root 对比
func PrintComparison() {
	fmt.Println("=== root vs non-root 容器对比 ===")
	fmt.Println("\n以 root 运行的风险：")
	for _, risk := range RootUserRisks() {
		fmt.Printf("  - %s\n", risk)
	}
	fmt.Println("\n使用非 root 用户的好处：")
	for _, benefit := range NonRootBenefits() {
		fmt.Printf("  + %s\n", benefit)
	}
}
