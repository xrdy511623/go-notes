package app

import (
	"go-notes/designpattern/dependencyinjection/service"
)

// App 是应用的顶层容器，持有所有已装配的依赖。
// 在真实项目中，这里会包含多个 Service、中间件客户端、配置等。
// App 本身不负责创建依赖——依赖由外部（container 包）装配后传入。
type App struct {
	UserService *service.UserService
}
