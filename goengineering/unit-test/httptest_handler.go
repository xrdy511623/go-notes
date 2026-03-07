package unittest

import (
	"encoding/json"
	"net/http"
)

// ---------- HTTP Handler（依赖 UserStore 接口） ----------

// UserHandler 提供用户相关的 HTTP API。
// 通过依赖注入 UserStore 接口，使 handler 可用 httptest 进行单元测试。
type UserHandler struct {
	store UserStore
}

// NewUserHandler 创建 UserHandler 实例。
func NewUserHandler(store UserStore) *UserHandler {
	return &UserHandler{store: store}
}

// apiError 是统一的 JSON 错误响应结构
type apiError struct {
	Error string `json:"error"`
}

// apiSuccess 是统一的 JSON 成功响应结构
type apiSuccess struct {
	Data interface{} `json:"data"`
}

// GetUser 处理 GET /users?id=xxx 请求。
// 演示 httptest.NewRecorder 和 httptest.NewServer 两种测试方式。
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "missing id parameter"})
		return
	}

	user, err := h.store.GetByID(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "internal error"})
		return
	}
	if user == nil {
		writeJSON(w, http.StatusNotFound, apiError{Error: "user not found"})
		return
	}

	writeJSON(w, http.StatusOK, apiSuccess{Data: user})
}

// CreateUser 处理 POST /users 请求，body 为 JSON。
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}

	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "invalid JSON body"})
		return
	}

	if req.Name == "" || req.Email == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "name and email are required"})
		return
	}

	user := &User{
		ID:    "generated-id",
		Name:  req.Name,
		Email: req.Email,
	}
	if err := h.store.Save(user); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "failed to save user"})
		return
	}

	writeJSON(w, http.StatusCreated, apiSuccess{Data: user})
}

// writeJSON 将 v 序列化为 JSON 并写入响应
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
