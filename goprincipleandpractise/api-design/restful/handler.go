package restful

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// User 是用户资源模型。
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateUserRequest 是创建用户的请求体。
type CreateUserRequest struct {
	Name  string `json:"name"  validate:"required,min=2,max=50"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age"   validate:"min=0,max=150"`
}

// UpdateUserRequest 是更新用户的请求体。
type UpdateUserRequest struct {
	Name  string `json:"name"  validate:"min=2,max=50"`
	Email string `json:"email" validate:"email"`
	Age   int    `json:"age"   validate:"min=0,max=150"`
}

// UserStore 定义用户存储接口（依赖倒置）。
type UserStore interface {
	List() []User
	Get(id string) (User, bool)
	Create(user User) User
	Update(id string, user User) (User, bool)
	Delete(id string) bool
}

// InMemoryUserStore 是基于内存的 UserStore 实现，用于示例和测试。
type InMemoryUserStore struct {
	mu    sync.RWMutex
	users map[string]User
	seq   int
}

// NewInMemoryUserStore 创建一个空的内存用户存储。
func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{users: make(map[string]User)}
}

func (s *InMemoryUserStore) List() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]User, 0, len(s.users))
	for _, u := range s.users {
		result = append(result, u)
	}
	return result
}

func (s *InMemoryUserStore) Get(id string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	return u, ok
}

func (s *InMemoryUserStore) Create(user User) User {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	user.ID = idFromSeq(s.seq)
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now
	s.users[user.ID] = user
	return user
}

func (s *InMemoryUserStore) Update(id string, user User) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.users[id]
	if !ok {
		return User{}, false
	}
	if user.Name != "" {
		existing.Name = user.Name
	}
	if user.Email != "" {
		existing.Email = user.Email
	}
	if user.Age != 0 {
		existing.Age = user.Age
	}
	existing.UpdatedAt = time.Now().UTC()
	s.users[id] = existing
	return existing, true
}

func (s *InMemoryUserStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.users[id]
	if ok {
		delete(s.users, id)
	}
	return ok
}

func idFromSeq(seq int) string {
	return "usr_" + padInt(seq, 6)
}

func padInt(n, width int) string {
	s := ""
	for v := n; v > 0; v /= 10 {
		s = string(rune('0'+v%10)) + s
	}
	for len(s) < width {
		s = "0" + s
	}
	return s
}

// UserHandler 封装用户资源的 HTTP 处理器。
type UserHandler struct {
	store         UserStore
	idempotencyMu sync.RWMutex
	idempotency   map[string][]byte // Idempotency-Key → 响应缓存
}

// NewUserHandler 创建 UserHandler。
func NewUserHandler(store UserStore) *UserHandler {
	return &UserHandler{
		store:       store,
		idempotency: make(map[string][]byte),
	}
}

// ListUsers GET /api/v1/users
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users := h.store.List()
	WriteSuccess(w, http.StatusOK, users)
}

// GetUser GET /api/v1/users/{id}
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	user, ok := h.store.Get(id)
	if !ok {
		WriteError(w, ErrUserNotFound)
		return
	}
	WriteSuccess(w, http.StatusOK, user)
}

// CreateUser POST /api/v1/users
// 支持 Idempotency-Key 头部实现幂等创建。
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// 幂等性检查
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey != "" {
		h.idempotencyMu.RLock()
		cached, ok := h.idempotency[idempotencyKey]
		h.idempotencyMu.RUnlock()
		if ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("X-Idempotent-Replayed", "true")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write(cached)
			return
		}
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidBody)
		return
	}

	if errs := Validate(req); len(errs) > 0 {
		WriteValidationError(w, errs)
		return
	}

	user := h.store.Create(User{
		Name:  req.Name,
		Email: req.Email,
		Age:   req.Age,
	})

	// 缓存幂等响应
	if idempotencyKey != "" {
		resp := Response[User]{Data: user}
		data, _ := json.Marshal(resp)
		h.idempotencyMu.Lock()
		h.idempotency[idempotencyKey] = data
		h.idempotencyMu.Unlock()
	}

	WriteSuccess(w, http.StatusCreated, user)
}

// UpdateUser PUT /api/v1/users/{id}
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidBody)
		return
	}

	if errs := Validate(req); len(errs) > 0 {
		WriteValidationError(w, errs)
		return
	}

	user, ok := h.store.Update(id, User{
		Name:  req.Name,
		Email: req.Email,
		Age:   req.Age,
	})
	if !ok {
		WriteError(w, ErrUserNotFound)
		return
	}

	WriteSuccess(w, http.StatusOK, user)
}

// DeleteUser DELETE /api/v1/users/{id}
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !h.store.Delete(id) {
		WriteError(w, ErrUserNotFound)
		return
	}
	WriteNoContent(w)
}
