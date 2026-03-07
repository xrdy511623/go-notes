package nodockerignore

/*
陷阱：缺少 .dockerignore 文件

问题说明：
  Docker build 会将 context（通常是整个项目目录）发送给 Docker daemon。
  如果没有 .dockerignore 文件，以下内容都会进入构建上下文：

  1. .git/ 目录（几十到几百 MB）
  2. .env 文件（包含数据库密码、API Key 等敏感信息！）
  3. IDE 配置（.idea/, .vscode/）
  4. 测试产物（coverage.out, *.test）
  5. 本地构建产物（bin/, dist/）
  6. node_modules/（如果有前端）
  7. vendor/（如果使用 vendor 模式）
  8. 文档和图片（docs/, images/）

  后果：
  - 构建上下文传输慢（.git 可能几百 MB）
  - 敏感信息进入镜像（.env 中的密码！）
  - 镜像体积增大
  - 缓存失效（.git 经常变化，导致 COPY . . 缓存 miss）

正确做法：创建 .dockerignore 文件

  # .dockerignore
  .git
  .github
  .vscode
  .idea
  *.md
  docs/
  bin/
  tmp/
  coverage.out
  *.test
  .env
  .env.*
  docker-compose*.yml
  Makefile
  LICENSE

特别注意：
  .env 文件绝对不能进入镜像！
  即使在多阶段构建中，如果 .env 在构建阶段被 COPY 进去，
  通过 docker history 仍然可以看到。
*/

import "fmt"

// DangerousFiles 列出不应该进入 Docker 镜像的文件
type DangerousFile struct {
	Pattern  string
	Risk     string
	Severity string
}

// ListDangerousFiles 返回应该在 .dockerignore 中排除的文件
func ListDangerousFiles() []DangerousFile {
	return []DangerousFile{
		{".env", "包含数据库密码、API Key 等敏感信息", "CRITICAL"},
		{".env.*", "环境特定的敏感配置", "CRITICAL"},
		{".git/", "Git 历史可能包含曾提交过的密钥", "HIGH"},
		{"**/*_test.go", "测试文件可能包含 mock 密钥", "MEDIUM"},
		{".github/", "CI 配置可能暴露基础设施信息", "MEDIUM"},
		{".vscode/", "IDE 配置可能包含远程连接信息", "LOW"},
		{".idea/", "IDE 配置", "LOW"},
		{"coverage.out", "覆盖率数据", "LOW"},
		{"*.md", "文档，增加镜像体积", "LOW"},
	}
}

// RecommendedDockerignore 返回推荐的 .dockerignore 内容
func RecommendedDockerignore() string {
	return `.git
.github
.gitlab-ci.yml
.vscode
.idea
.env
.env.*
*.md
docs/
bin/
dist/
tmp/
vendor/
coverage.out
*.test
*.prof
*.out
docker-compose*.yml
Makefile
Dockerfile*
LICENSE
.dockerignore`
}

// PrintDangerousFiles 打印危险文件列表
func PrintDangerousFiles() {
	fmt.Println("=== 不应进入 Docker 镜像的文件 ===")
	for _, f := range ListDangerousFiles() {
		fmt.Printf("\n[%s] %s\n", f.Severity, f.Pattern)
		fmt.Printf("  风险：%s\n", f.Risk)
	}
	fmt.Println("\n推荐的 .dockerignore：")
	fmt.Println(RecommendedDockerignore())
}
