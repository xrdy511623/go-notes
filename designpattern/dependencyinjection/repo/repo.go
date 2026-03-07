package repo

import (
	"errors"
	"fmt"
	"sync"
)

// User 表示用户领域模型。
type User struct {
	ID    string
	Name  string
	Email string
}

// UserRepo 定义用户数据访问接口。
// 所有持久层实现（MySQL、内存、MongoDB……）都必须满足此契约，
// 上层业务代码只依赖该接口，不依赖具体实现。
type UserRepo interface {
	FindByID(id string) (*User, error)
	Save(u *User) error
}

// ─── MySQL 实现 ──────────────────────────────────────────────

// mysqlRepo 是 UserRepo 的 MySQL 实现。
// 此处为演示用途，仅模拟数据库行为。
type mysqlRepo struct {
	dsn string
}

// NewMySQL 创建基于 MySQL 的 UserRepo。
// dsn 为数据源名称，生产环境下会用于建立数据库连接。
func NewMySQL(dsn string) UserRepo {
	return &mysqlRepo{dsn: dsn}
}

func (r *mysqlRepo) FindByID(id string) (*User, error) {
	if id == "" {
		return nil, errors.New("repo: id must not be empty")
	}
	// 模拟从 MySQL 读取数据
	return &User{
		ID:    id,
		Name:  "user-" + id,
		Email: id + "@example.com",
	}, nil
}

func (r *mysqlRepo) Save(u *User) error {
	if u == nil {
		return errors.New("repo: user must not be nil")
	}
	if u.ID == "" {
		return errors.New("repo: user ID must not be empty")
	}
	if u.Name == "" {
		return errors.New("repo: user Name must not be empty")
	}
	if u.Email == "" {
		return errors.New("repo: user Email must not be empty")
	}
	// 模拟写入 MySQL — 生产环境会执行 INSERT/UPDATE
	return nil
}

// ─── InMemory 实现（用于测试） ───────────────────────────────

// inMemoryRepo 是 UserRepo 的内存实现，适用于单元测试。
// 数据存储在 map 中，无需外部依赖。
type inMemoryRepo struct {
	mu    sync.RWMutex
	store map[string]*User
}

// NewInMemory 创建基于内存 map 的 UserRepo，用于测试场景。
func NewInMemory() UserRepo {
	return &inMemoryRepo{
		store: make(map[string]*User),
	}
}

func (r *inMemoryRepo) FindByID(id string) (*User, error) {
	if id == "" {
		return nil, errors.New("repo: id must not be empty")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, ok := r.store[id]
	if !ok {
		return nil, fmt.Errorf("repo: user %q not found", id)
	}
	// 返回副本，防止调用方修改内部状态
	cp := *u
	return &cp, nil
}

func (r *inMemoryRepo) Save(u *User) error {
	if u == nil {
		return errors.New("repo: user must not be nil")
	}
	if u.ID == "" {
		return errors.New("repo: user ID must not be empty")
	}
	if u.Name == "" {
		return errors.New("repo: user Name must not be empty")
	}
	if u.Email == "" {
		return errors.New("repo: user Email must not be empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	cp := *u
	r.store[u.ID] = &cp
	return nil
}
