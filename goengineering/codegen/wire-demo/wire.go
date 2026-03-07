//go:build wireinject

package wiredemo

import "github.com/google/wire"

// InitializeApp 是wire的Injector声明。
// wire会分析所有Provider的输入输出类型，自动生成依赖组装代码。
//
// 这个函数体会被wire生成的代码完全替换。
func InitializeApp(cfg *Config) (*App, error) {
	wire.Build(
		NewDatabase,    // *Config → *Database, error
		NewUserRepo,    // *Database → *UserRepo
		NewUserService, // *UserRepo → *UserService
		NewApp,         // *UserService, *Config → *App
	)
	return nil, nil
}
