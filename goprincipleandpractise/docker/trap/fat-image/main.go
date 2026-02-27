package fatimage

/*
陷阱：使用 golang 基础镜像作为运行时

问题说明：
  很多新手写 Dockerfile 时直接用 golang 镜像运行编译好的二进制：

  FROM golang:1.24
  COPY . .
  RUN go build -o /app .
  CMD ["/app"]

  这会导致最终镜像包含：
  - Go 编译器和标准库源码（~500MB）
  - 操作系统工具链（gcc, make, etc.）
  - 所有 go module 依赖的源码
  - 你的项目源码（包括测试文件）
  - 总计：~800MB

  而实际运行只需要一个编译好的二进制文件（通常 5-20MB）。

  800MB 的镜像意味着：
  - 拉取镜像慢（CI/CD 部署延迟）
  - 存储成本高（镜像仓库占用空间）
  - 攻击面大（包含编译器，可被用于编译恶意代码）
  - CVE 数量多（更多软件包 = 更多漏洞）

正确做法：使用多阶段构建（multi-stage build）

  # 阶段 1：编译
  FROM golang:1.24-alpine AS builder
  WORKDIR /app
  COPY go.mod go.sum ./
  RUN go mod download
  COPY . .
  RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/server .

  # 阶段 2：运行（最终镜像只有二进制）
  FROM scratch
  COPY --from=builder /app/server /server
  CMD ["/server"]

  最终镜像大小：~5MB（vs 800MB）
*/

import "fmt"

// ImageContents 列出不同基础镜像中包含的内容
type ImageContents struct {
	BaseImage   string
	Size        string
	Includes    []string
	CVECount    string
	CanDebug    bool
	Recommended bool
}

// FatImage 展示 golang 基础镜像的臃肿内容
func FatImage() ImageContents {
	return ImageContents{
		BaseImage: "golang:1.24",
		Size:      "~800MB",
		Includes: []string{
			"Go 编译器和工具链",
			"标准库完整源码",
			"GCC 编译器",
			"libc 和其他 C 库",
			"Git",
			"项目源码和测试文件",
			"go module 依赖源码",
			"build cache",
		},
		CVECount:    "100+（包含大量系统工具）",
		CanDebug:    true,
		Recommended: false,
	}
}

// SlimImage 展示 scratch 镜像的精简内容
func SlimImage() ImageContents {
	return ImageContents{
		BaseImage: "scratch",
		Size:      "~5MB",
		Includes: []string{
			"编译好的 Go 二进制（仅此一项）",
		},
		CVECount:    "0（没有操作系统组件）",
		CanDebug:    false,
		Recommended: true,
	}
}

// PrintComparison 打印镜像大小对比
func PrintComparison() {
	fat := FatImage()
	slim := SlimImage()

	fmt.Println("=== Docker 镜像大小对比 ===")
	fmt.Printf("\n%-20s %s\n", "golang:1.24", fat.Size)
	fmt.Println("  包含内容：")
	for _, item := range fat.Includes {
		fmt.Printf("    - %s\n", item)
	}
	fmt.Printf("  CVE: %s\n", fat.CVECount)

	fmt.Printf("\n%-20s %s\n", "scratch", slim.Size)
	fmt.Println("  包含内容：")
	for _, item := range slim.Includes {
		fmt.Printf("    - %s\n", item)
	}
	fmt.Printf("  CVE: %s\n", slim.CVECount)

	fmt.Println("\n结论：生产环境请使用多阶段构建 + scratch/distroless")
}
