package service

import (
	"fmt"
	"sync/atomic"

	"go-notes/designpattern/dependencyinjection/repo"
)

// UserService 提供用户业务逻辑。
// 它通过构造函数接收 UserRepo 接口，不关心底层使用的是 MySQL 还是内存实现。
type UserService struct {
	repo repo.UserRepo
}

// NewUserService 通过构造函数注入 UserRepo 依赖。
func NewUserService(r repo.UserRepo) *UserService {
	return &UserService{repo: r}
}

// GetUser 根据 ID 查询用户。
func (s *UserService) GetUser(id string) (*repo.User, error) {
	u, err := s.repo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("service: get user: %w", err)
	}
	return u, nil
}

// CreateUser 创建新用户。
// 使用简单的计数器生成 ID，生产环境应使用 UUID。
func (s *UserService) CreateUser(name, email string) (*repo.User, error) {
	if name == "" {
		return nil, fmt.Errorf("service: name must not be empty")
	}
	if email == "" {
		return nil, fmt.Errorf("service: email must not be empty")
	}
	u := &repo.User{
		ID:    generateID(name),
		Name:  name,
		Email: email,
	}
	if err := s.repo.Save(u); err != nil {
		return nil, fmt.Errorf("service: create user: %w", err)
	}
	return u, nil
}

// UpdateEmail 更新指定用户的邮箱地址。
func (s *UserService) UpdateEmail(id, newEmail string) error {
	if newEmail == "" {
		return fmt.Errorf("service: new email must not be empty")
	}
	u, err := s.repo.FindByID(id)
	if err != nil {
		return fmt.Errorf("service: update email: %w", err)
	}
	u.Email = newEmail
	if err := s.repo.Save(u); err != nil {
		return fmt.Errorf("service: update email: %w", err)
	}
	return nil
}

var idSeq int64

// generateID 生成简易唯一 ID（仅用于演示，生产环境应使用 UUID）。
func generateID(name string) string {
	n := atomic.AddInt64(&idSeq, 1)
	return fmt.Sprintf("u-%s-%d", name, n)
}
