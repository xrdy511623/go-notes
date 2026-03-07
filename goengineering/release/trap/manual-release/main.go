package manualrelease

/*
陷阱：手动发布流程

问题说明：
  手动发布流程依赖人的记忆和操作，每一步都可能出错：

  1. 忘记运行测试 → 发布有 bug 的版本
  2. 忘记更新 CHANGELOG → 用户不知道改了什么
  3. 版本号打错 → v1.2.3 变成 v1.23
  4. 忘记某个平台的交叉编译 → Linux arm64 用户无法使用
  5. 忘记发布 Docker 镜像 → 容器用户无法升级
  6. 忘记通知团队 → 运维不知道有新版本
  7. 二进制中忘记注入版本号 → 线上无法确认版本

  手动发布的典型步骤：
    1. 确认测试全部通过        ← 可能忘记
    2. 更新版本号               ← 可能打错
    3. 更新 CHANGELOG           ← 可能遗漏
    4. 提交并打 tag             ← 可能 tag 格式错误
    5. 多平台交叉编译           ← 可能漏掉某个平台
    6. 上传到 GitHub Release    ← 可能传错文件
    7. 构建 Docker 镜像         ← 可能忘记
    8. 推送到镜像仓库           ← 可能忘记打 latest 标签
    9. 更新 Homebrew formula    ← 可能忘记
   10. 通知团队                  ← 可能忘记

  以上 10 个步骤，每个都可能出错，手动执行的可靠性极低。

正确做法：使用 goreleaser + GitHub Actions 全自动化

  开发者只需：
    git tag v1.2.3
    git push origin v1.2.3

  CI 自动完成剩下的所有步骤。
*/

import "fmt"

// ReleaseStep 发布步骤
type ReleaseStep struct {
	Step        int
	Description string
	Automated   bool   // 是否可以自动化
	FailureRisk string // 手动操作的失败风险
}

// ManualReleaseSteps 手动发布的所有步骤
func ManualReleaseSteps() []ReleaseStep {
	return []ReleaseStep{
		{1, "运行全量测试", true, "忘记运行，发布有 bug 的版本"},
		{2, "更新版本号", true, "打错版本号，如 v1.23 而非 v1.2.3"},
		{3, "更新 CHANGELOG", true, "遗漏重要变更，用户不知道改了什么"},
		{4, "提交代码并打 Git tag", true, "tag 格式错误，不符合 semver"},
		{5, "多平台交叉编译", true, "遗漏某个平台，如 linux/arm64"},
		{6, "上传到 GitHub Release", true, "传错文件或缺少 checksum"},
		{7, "构建 Docker 镜像", true, "忘记构建或使用错误的 Dockerfile"},
		{8, "推送镜像到仓库", true, "忘记打 latest 标签"},
		{9, "更新 Homebrew formula", true, "忘记更新或 SHA 不匹配"},
		{10, "通知团队", true, "忘记通知，运维不知道有新版本"},
	}
}

// AutomatedRelease 自动化发布只需两步
func AutomatedRelease() []string {
	return []string{
		"git tag -a v1.2.3 -m 'Release v1.2.3'",
		"git push origin v1.2.3",
		"// 剩下的全部由 CI + goreleaser 自动完成",
	}
}

// PrintComparison 打印手动 vs 自动对比
func PrintComparison() {
	fmt.Println("=== 手动发布 vs 自动发布 ===")
	fmt.Println("\n手动发布（10 个步骤，每个都可能出错）：")
	for _, step := range ManualReleaseSteps() {
		fmt.Printf("  %2d. %-25s 风险：%s\n",
			step.Step, step.Description, step.FailureRisk)
	}

	fmt.Println("\n自动发布（2 个步骤）：")
	for _, cmd := range AutomatedRelease() {
		fmt.Printf("  $ %s\n", cmd)
	}
}
