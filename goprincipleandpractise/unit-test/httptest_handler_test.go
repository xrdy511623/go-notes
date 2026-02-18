package unittest

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------- httptest.NewRecorder：单元测试风格 ----------
//
// 不启动真实 HTTP 服务器，直接调用 handler 函数，
// 通过 httptest.ResponseRecorder 捕获响应。
// 适合快速、隔离的单元测试。

func TestGetUser_Recorder(t *testing.T) {
	tests := []struct {
		store      *StubUserStore
		name       string
		query      string
		method     string
		wantBody   string
		wantStatus int
	}{
		{
			name: "成功获取用户",
			store: &StubUserStore{
				GetByIDFunc: func(id string) (*User, error) {
					return &User{ID: "1", Name: "Alice", Email: "alice@example.com"}, nil
				},
			},
			query:      "?id=1",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantBody:   `"Name":"Alice"`,
		},
		{
			name:       "缺少 id 参数",
			store:      &StubUserStore{},
			query:      "",
			method:     http.MethodGet,
			wantStatus: http.StatusBadRequest,
			wantBody:   "missing id",
		},
		{
			name: "用户不存在",
			store: &StubUserStore{
				GetByIDFunc: func(id string) (*User, error) {
					return nil, nil
				},
			},
			query:      "?id=999",
			method:     http.MethodGet,
			wantStatus: http.StatusNotFound,
			wantBody:   "not found",
		},
		{
			name:       "方法不允许",
			store:      &StubUserStore{},
			query:      "?id=1",
			method:     http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   "method not allowed",
		},
		{
			name: "存储层错误",
			store: &StubUserStore{
				GetByIDFunc: func(id string) (*User, error) {
					return nil, errors.New("db down")
				},
			},
			query:      "?id=1",
			method:     http.MethodGet,
			wantStatus: http.StatusInternalServerError,
			wantBody:   "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewUserHandler(tt.store)

			req := httptest.NewRequest(tt.method, "/users"+tt.query, nil)
			rec := httptest.NewRecorder()

			handler.GetUser(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			body := rec.Body.String()
			if !strings.Contains(body, tt.wantBody) {
				t.Errorf("body = %q, want containing %q", body, tt.wantBody)
			}
		})
	}
}

// ---------- httptest.NewServer：集成测试风格 ----------
//
// 启动一个真实的 HTTP 服务器（绑定到随机端口），
// 使用标准 http.Client 发送请求。
// 适合测试完整的 HTTP 请求/响应周期，包括中间件、路由等。

func TestGetUser_Server(t *testing.T) {
	store := &StubUserStore{
		GetByIDFunc: func(id string) (*User, error) {
			if id == "1" {
				return &User{ID: "1", Name: "Alice", Email: "alice@example.com"}, nil
			}
			return nil, nil
		},
	}
	handler := NewUserHandler(store)

	// 启动测试服务器
	srv := httptest.NewServer(http.HandlerFunc(handler.GetUser))
	defer srv.Close() // 测试结束后自动关闭

	// 使用标准 HTTP 客户端
	resp, err := http.Get(srv.URL + "?id=1")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result apiSuccess
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// result.Data 是 map[string]interface{} 因为 JSON 解码
	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected data type: %T", result.Data)
	}
	if data["Name"] != "Alice" {
		t.Errorf("Name = %v, want Alice", data["Name"])
	}
}

func TestCreateUser_Recorder(t *testing.T) {
	tests := []struct {
		store      *StubUserStore
		name       string
		method     string
		body       string
		wantBody   string
		wantStatus int
	}{
		{
			name: "成功创建用户",
			store: &StubUserStore{
				SaveFunc: func(user *User) error { return nil },
			},
			method:     http.MethodPost,
			body:       `{"name":"Bob","email":"bob@example.com"}`,
			wantStatus: http.StatusCreated,
			wantBody:   `"Name":"Bob"`,
		},
		{
			name:       "非 POST 方法",
			store:      &StubUserStore{},
			method:     http.MethodGet,
			body:       "",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "无效 JSON",
			store:      &StubUserStore{},
			method:     http.MethodPost,
			body:       `{invalid`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid JSON",
		},
		{
			name:       "缺少必填字段",
			store:      &StubUserStore{},
			method:     http.MethodPost,
			body:       `{"name":"","email":""}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewUserHandler(tt.store)

			req := httptest.NewRequest(tt.method, "/users", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.CreateUser(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantBody != "" {
				body := rec.Body.String()
				if !strings.Contains(body, tt.wantBody) {
					t.Errorf("body = %q, want containing %q", body, tt.wantBody)
				}
			}
		})
	}
}
