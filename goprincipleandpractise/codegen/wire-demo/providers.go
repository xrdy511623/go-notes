// Package wiredemo 演示wire依赖注入代码生成。
//
// wire是Google开源的编译期DI工具，分析Provider依赖图自动生成组装代码。
//
// 使用方式：
//
//	go install github.com/google/wire/cmd/wire@latest
//	cd wire-demo && wire
//	go test -v .
package wiredemo

import "fmt"

// ---------- 配置 ----------

type Config struct {
	DBHost  string
	DBPort  int
	AppPort int
}

// ---------- 数据库层 ----------

type Database struct {
	Host string
	Port int
}

func NewDatabase(cfg *Config) (*Database, error) {
	if cfg.DBHost == "" {
		return nil, fmt.Errorf("db host is required")
	}
	return &Database{Host: cfg.DBHost, Port: cfg.DBPort}, nil
}

// ---------- 仓储层 ----------

type UserRepo struct {
	db *Database
}

func NewUserRepo(db *Database) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) FindByID(id string) string {
	return fmt.Sprintf("user-%s from %s:%d", id, r.db.Host, r.db.Port)
}

// ---------- 服务层 ----------

type UserService struct {
	repo *UserRepo
}

func NewUserService(repo *UserRepo) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetUser(id string) string {
	return s.repo.FindByID(id)
}

// ---------- 应用入口 ----------

type App struct {
	UserSvc *UserService
	Port    int
}

func NewApp(svc *UserService, cfg *Config) *App {
	return &App{UserSvc: svc, Port: cfg.AppPort}
}
