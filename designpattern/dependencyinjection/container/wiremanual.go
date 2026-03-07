package container

import (
	"go-notes/designpattern/dependencyinjection/app"
	"go-notes/designpattern/dependencyinjection/repo"
	"go-notes/designpattern/dependencyinjection/service"
)

// Config 包含应用启动所需的外部配置。
// 将配置集中到一个结构体，便于管理和校验。
type Config struct {
	MySQLDSN string
}

// InitializeApp 手动组装依赖图，等价于 Wire 生成的代码。
//
// 依赖图：
//
//	Config.MySQLDSN
//	     └─→ repo.NewMySQL(dsn) ─→ UserRepo
//	                                   └─→ service.NewUserService(repo) ─→ UserService
//	                                                                          └─→ app.App
//
// 手动装配的优点：无需代码生成，依赖关系一目了然，IDE 可直接跳转。
func InitializeApp(cfg Config) *app.App {
	userRepo := repo.NewMySQL(cfg.MySQLDSN)
	userSvc := service.NewUserService(userRepo)
	return &app.App{
		UserService: userSvc,
	}
}

// InitializeTestApp 使用内存实现组装测试用依赖图。
// 测试不需要真实数据库连接，用 InMemory 替换 MySQL 即可实现完全隔离。
func InitializeTestApp() *app.App {
	userRepo := repo.NewInMemory()
	userSvc := service.NewUserService(userRepo)
	return &app.App{
		UserService: userSvc,
	}
}
