// Package mockgendemo 演示mockgen代码生成和gomock使用。
//
// 使用方式：
//
//	go generate ./...
//	go test -v .
package mockgendemo

import (
	"context"
	"errors"
	"fmt"
)

// ---------- 领域模型 ----------

type User struct {
	ID    string
	Name  string
	Email string
}

// ---------- 接口定义 ----------

//go:generate mockgen -source=store.go -destination=mock_store_test.go -package=mockgendemo

// UserStore 用户存储接口
type UserStore interface {
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user *User) error
}

// ---------- 业务错误 ----------

var (
	ErrNotFound      = errors.New("user not found")
	ErrAlreadyExists = errors.New("user already exists")
	ErrInvalidEmail  = errors.New("invalid email")
)

// ---------- 业务逻辑 ----------

type UserService struct {
	store UserStore
}

func NewUserService(store UserStore) *UserService {
	return &UserService{store: store}
}

// Register 注册新用户：检查邮箱是否已存在，不存在则创建
func (s *UserService) Register(ctx context.Context, name, email string) (*User, error) {
	if email == "" {
		return nil, ErrInvalidEmail
	}

	// 检查是否已存在
	existing, err := s.store.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if existing != nil {
		return nil, ErrAlreadyExists
	}

	user := &User{
		ID:    fmt.Sprintf("user_%s", name),
		Name:  name,
		Email: email,
	}

	if err := s.store.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}
