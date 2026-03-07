package nolintinci

/*
陷阱：CI 中不配置 linter

问题说明：
  没有 linter 的 CI 流水线只能发现编译错误和测试失败，
  但无法发现以下代码质量问题：

  1. error shadow — 内层 err 遮蔽外层 err
  2. unchecked error — 忽略了函数返回的 error
  3. ineffectual assignment — 赋值后未使用
  4. HTTP body 未关闭 — 资源泄漏
  5. 安全问题 — gosec 能发现的漏洞

  这些问题编译能过、测试能过，但会在生产环境中引发 bug。

正确做法：
  CI 中配置 golangci-lint，它聚合了 100+ 个 linter：

  - uses: golangci/golangci-lint-action@v6
    with:
      version: latest
      args: --timeout=5m

  推荐开启的 linter：
  - errcheck: 未处理的错误
  - govet: go vet 检查
  - staticcheck: 高级静态分析
  - bodyclose: HTTP body 未关闭
  - gosec: 安全问题
*/

import (
	"fmt"
	"net/http"
	"os"
)

// === 问题 1：Error Shadow（错误遮蔽）===

// ProcessFile 演示 error shadow 问题
// golangci-lint 的 govet 会检测到 shadow
func ProcessFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// 错误：这里的 err 遮蔽了外层的 err
	// 如果 Read 失败，外层 err 仍然是 nil
	buf := make([]byte, 1024)
	if _, err := f.Read(buf); err != nil {
		// 这里的 err 是新的局部变量
		fmt.Println("read error:", err)
		// 没有 return，继续执行
	}

	// 外层 err 仍然是 Open 成功时的 nil
	return nil
}

// === 问题 2：Unchecked Error（未检查错误）===

// WriteConfig 演示未检查错误
// golangci-lint 的 errcheck 会报告
func WriteConfig(path string, data []byte) {
	f, _ := os.Create(path) // 错误：忽略了 Create 的 error
	f.Write(data)           // 错误：忽略了 Write 的 error
	f.Close()               // 错误：忽略了 Close 的 error
	// 如果磁盘满了或权限不足，这里静默失败
	// 用户以为配置已保存，实际上没有
}

// === 问题 3：HTTP Body 未关闭（资源泄漏）===

// FetchURL 演示 HTTP response body 未关闭
// golangci-lint 的 bodyclose 会检测到
func FetchURL(url string) (int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	// 错误：没有 defer resp.Body.Close()
	// 每次调用都泄漏一个 TCP 连接
	// 高并发下很快耗尽文件描述符
	return resp.StatusCode, nil
}

// === 正确写法 ===

// ProcessFileSafe 正确处理错误
func ProcessFileSafe(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	buf := make([]byte, 1024)
	if _, err = f.Read(buf); err != nil { // 用 = 而非 :=，不遮蔽
		return fmt.Errorf("read file: %w", err)
	}
	return nil
}

// WriteConfigSafe 正确检查所有错误
func WriteConfigSafe(path string, data []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// FetchURLSafe 正确关闭 HTTP body
func FetchURLSafe(url string) (int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}
