package unittest

import (
	"errors"
	"fmt"
)

// ---------- 领域模型 ----------

// User 表示系统中的用户
type User struct {
	ID    string
	Name  string
	Email string
}

// ---------- 接口定义（面向行为，而非实现） ----------

// UserStore 定义用户持久化行为
// 在 Go 中，接口应在使用方定义，而非实现方。
type UserStore interface {
	GetByID(id string) (*User, error)
	Save(user *User) error
}

// ---------- 手写 Stub（最简单的 Mock 方式） ----------

// StubUserStore 是 UserStore 的手写 Stub 实现。
// 通过设置函数字段来控制返回值，适合小型项目或简单场景。
//
// 对比三种 Mock 方式：
//   - 手写 Stub：零依赖，灵活，适合接口方法少的场景（推荐）
//   - gomock：代码生成，适合接口方法多且需要严格调用验证的场景
//   - gomonkey：运行时 patch，不依赖接口，但不支持 -race，慎用
type StubUserStore struct {
	GetByIDFunc func(id string) (*User, error)
	SaveFunc    func(user *User) error
}

func (s *StubUserStore) GetByID(id string) (*User, error) {
	if s.GetByIDFunc != nil {
		return s.GetByIDFunc(id)
	}
	return nil, errors.New("GetByIDFunc not set")
}

func (s *StubUserStore) Save(user *User) error {
	if s.SaveFunc != nil {
		return s.SaveFunc(user)
	}
	return errors.New("SaveFunc not set")
}

// ---------- 业务层 ----------

// 常见错误
var (
	ErrUserNotFound = errors.New("user not found")
	ErrEmptyName    = errors.New("name must not be empty")
	ErrEmptyEmail   = errors.New("email must not be empty")
	ErrStoreFailed  = errors.New("store operation failed")
)

// UserService 封装用户相关业务逻辑，通过构造函数注入 UserStore 依赖。
type UserService struct {
	store UserStore
}

// NewUserService 创建 UserService 实例。依赖注入使测试可以替换底层存储。
func NewUserService(store UserStore) *UserService {
	return &UserService{store: store}
}

// GetUser 按 ID 查找用户，未找到时返回 ErrUserNotFound。
func (s *UserService) GetUser(id string) (*User, error) {
	user, err := s.store.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("get user %s: %w", id, err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// CreateUser 校验并持久化新用户。
func (s *UserService) CreateUser(name, email string) (*User, error) {
	if name == "" {
		return nil, ErrEmptyName
	}
	if email == "" {
		return nil, ErrEmptyEmail
	}

	user := &User{
		ID:    fmt.Sprintf("user_%s", name), // 简化 ID 生成
		Name:  name,
		Email: email,
	}
	if err := s.store.Save(user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}
