package ldflagsvsembed

/*
性能对比：ldflags 注入 vs go:embed 读取版本信息

两种在运行时获取版本信息的方式：

  方式 1：ldflags 注入（推荐）
  var version = "dev"
  // go build -ldflags="-X 'main.version=v1.2.3'"

  方式 2：go:embed 嵌入版本文件
  //go:embed VERSION
  var version string

运行基准测试：
  go test -bench=. -benchmem -benchtime=3s .

关键结论：
  - ldflags 注入的变量是普通字符串变量，读取零开销
  - go:embed 嵌入的也是编译时确定的字符串，读取零开销
  - 两者运行时性能完全相同
  - 差异在于工程实践：
    - ldflags 更灵活（CI 中动态注入）
    - go:embed 需要维护 VERSION 文件
    - ldflags 是业界标准做法
*/

import (
	"strings"
)

// === 方式 1：ldflags 注入 ===

// 由 ldflags 在编译时注入
var (
	ldflagsVersion   = "dev"
	ldflagsCommit    = "none"
	ldflagsBuildTime = "unknown"
)

// GetLdflagsVersion 获取 ldflags 注入的版本信息
func GetLdflagsVersion() string {
	return ldflagsVersion
}

// GetLdflagsFullInfo 获取完整版本信息
func GetLdflagsFullInfo() string {
	return ldflagsVersion + " " + ldflagsCommit + " " + ldflagsBuildTime
}

// === 方式 2：go:embed 嵌入（模拟）===

// 模拟 go:embed 嵌入的版本文件内容
// 实际使用时：
//
//	//go:embed VERSION
//	var embedVersion string
var embedVersion = "v1.2.3\n"

// GetEmbedVersion 获取嵌入的版本信息
func GetEmbedVersion() string {
	return strings.TrimSpace(embedVersion)
}

// === 方式 3：runtime/debug.ReadBuildInfo ===

// GetBuildInfoVersion 模拟通过 debug.ReadBuildInfo 获取版本
// 实际代码中会调用 debug.ReadBuildInfo()
func GetBuildInfoVersion() string {
	// debug.ReadBuildInfo() 返回的信息包括：
	// - GoVersion: "go1.24"
	// - Main.Version: "(devel)" 或 module version
	// - Settings: 包含 vcs.revision, vcs.time 等
	return "go1.24 (devel)"
}

// CompareApproaches 对比三种方式
func CompareApproaches() map[string]map[string]string {
	return map[string]map[string]string{
		"ldflags": {
			"机制":    "链接器在编译时替换变量值",
			"灵活性":   "高：CI 中动态注入任意值",
			"维护成本":  "低：无需维护文件",
			"信息丰富度": "高：可注入 version/commit/buildTime",
			"适用场景":  "所有 Go 项目（业界标准）",
		},
		"go:embed": {
			"机制":    "编译器嵌入文件内容到二进制",
			"灵活性":   "低：需要先更新 VERSION 文件",
			"维护成本":  "中：需要维护 VERSION 文件",
			"信息丰富度": "低：通常只有版本号",
			"适用场景":  "简单场景或配合 ldflags 使用",
		},
		"debug.ReadBuildInfo": {
			"机制":    "运行时读取 Go 工具链嵌入的构建信息",
			"灵活性":   "自动：无需任何配置",
			"维护成本":  "零",
			"信息丰富度": "中：GoVersion + VCS 信息",
			"适用场景":  "作为 ldflags 的补充",
		},
	}
}
