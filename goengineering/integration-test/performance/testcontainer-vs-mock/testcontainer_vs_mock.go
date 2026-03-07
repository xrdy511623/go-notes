package testcontainer_vs_mock

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// testcontainer_vs_mock 对比两种集成测试策略的执行速度：
//
// 1. Testcontainer 方式：启动真实 PostgreSQL 容器，执行真实 SQL
// 2. Mock 方式：使用内存中的 Mock 实现，不涉及网络和磁盘 I/O
//
// 结论：Mock 方式快 10-100 倍，但无法验证真实 SQL 语义。
// 最佳实践是两者结合——单元测试用 Mock，集成测试用 Testcontainer。

// UserRepository 定义用户仓库接口
type UserRepository interface {
	Create(ctx context.Context, name, email string) (int64, error)
	FindByID(ctx context.Context, id int64) (string, string, error)
}

// MockUserRepository 内存 Mock 实现
type MockUserRepository struct {
	mu     sync.RWMutex
	users  map[int64]mockUser
	nextID int64
}

type mockUser struct {
	Name  string
	Email string
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:  make(map[int64]mockUser),
		nextID: 1,
	}
}

func (r *MockUserRepository) Create(_ context.Context, name, email string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := r.nextID
	r.nextID++
	r.users[id] = mockUser{Name: name, Email: email}
	return id, nil
}

func (r *MockUserRepository) FindByID(_ context.Context, id int64) (string, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.users[id]
	if !ok {
		return "", "", fmt.Errorf("user %d not found", id)
	}
	return u.Name, u.Email, nil
}

// SimulatedDBRepository 模拟真实数据库操作的延迟
// （真实 testcontainer 场景中会使用 database/sql 连接容器）
type SimulatedDBRepository struct {
	mu      sync.RWMutex
	users   map[int64]mockUser
	nextID  int64
	latency time.Duration // 模拟网络和磁盘 I/O 延迟
}

func NewSimulatedDBRepository(latency time.Duration) *SimulatedDBRepository {
	return &SimulatedDBRepository{
		users:   make(map[int64]mockUser),
		nextID:  1,
		latency: latency,
	}
}

func (r *SimulatedDBRepository) Create(_ context.Context, name, email string) (int64, error) {
	time.Sleep(r.latency) // 模拟 INSERT 延迟
	r.mu.Lock()
	defer r.mu.Unlock()
	id := r.nextID
	r.nextID++
	r.users[id] = mockUser{Name: name, Email: email}
	return id, nil
}

func (r *SimulatedDBRepository) FindByID(_ context.Context, id int64) (string, string, error) {
	time.Sleep(r.latency) // 模拟 SELECT 延迟
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.users[id]
	if !ok {
		return "", "", fmt.Errorf("user %d not found", id)
	}
	return u.Name, u.Email, nil
}
