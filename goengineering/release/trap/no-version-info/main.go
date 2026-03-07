package noversioninfo

/*
陷阱：部署时不注入版本信息

问题说明：
  如果编译时不通过 ldflags 注入版本信息，运行中的程序
  无法回答"你是哪个版本？"这个基本问题。

  线上出现 bug 时的对话：
    Q: "线上跑的是哪个版本？"
    A: "不知道，看看是什么时候部署的？"
    Q: "这个 bug 是哪个版本引入的？"
    A: "不确定，得一个一个版本试..."

  没有版本信息意味着：
  1. 无法快速定位问题版本
  2. 无法确认回滚到哪个版本
  3. 无法判断修复是否已部署
  4. 监控和告警中缺少版本维度

正确做法：
  1. 定义版本变量：
     var version = "dev"
     var commit = "none"
     var buildTime = "unknown"

  2. 编译时注入：
     go build -ldflags="-X 'main.version=v1.2.3'
       -X 'main.commit=abc1234'
       -X 'main.buildTime=2024-01-15T10:30:00Z'"

  3. 暴露版本信息：
     - --version flag
     - /health 或 /version 端点
     - 启动日志
     - Prometheus metrics label
*/

import "fmt"

// App 模拟一个没有版本信息的应用
type App struct {
	Name string
}

// Start 启动应用（没有版本信息）
func (a *App) Start() {
	// 问题：启动日志中没有版本信息
	fmt.Printf("Starting %s...\n", a.Name)
	// 运维看到这行日志，无法知道是哪个版本
}

// Health 健康检查（没有版本信息）
func (a *App) Health() map[string]string {
	return map[string]string{
		"status": "ok",
		// 缺少 version, commit, buildTime
	}
}

// VersionedApp 正确的做法：包含版本信息
type VersionedApp struct {
	Name      string
	Version   string
	Commit    string
	BuildTime string
}

// Start 启动时输出版本信息
func (a *VersionedApp) Start() {
	fmt.Printf("Starting %s %s (commit: %s, built: %s)\n",
		a.Name, a.Version, a.Commit, a.BuildTime)
}

// Health 健康检查包含版本信息
func (a *VersionedApp) Health() map[string]string {
	return map[string]string{
		"status":     "ok",
		"version":    a.Version,
		"commit":     a.Commit,
		"build_time": a.BuildTime,
	}
}
