package restful

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthz(t *testing.T) {
	srv := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestUserCRUD(t *testing.T) {
	srv := NewServer()
	auth := "Bearer demo-token"

	// ── Create ───────────────────────────────────
	body := `{"name":"Alice","email":"alice@example.com","age":30}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req.Header.Set("Authorization", auth)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var createResp Response[User]
	if err := json.NewDecoder(rec.Body).Decode(&createResp); err != nil {
		t.Fatalf("create: decode error: %v", err)
	}
	userID := createResp.Data.ID
	if userID == "" {
		t.Fatal("create: user ID is empty")
	}

	// ── List ─────────────────────────────────────
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rec.Code)
	}

	// ── Get ──────────────────────────────────────
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID, nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", rec.Code)
	}

	var getResp Response[User]
	if err := json.NewDecoder(rec.Body).Decode(&getResp); err != nil {
		t.Fatalf("get: decode error: %v", err)
	}
	if getResp.Data.Name != "Alice" {
		t.Fatalf("get: expected name Alice, got %s", getResp.Data.Name)
	}

	// ── Update ───────────────────────────────────
	updateBody := `{"name":"Alice Updated"}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/users/"+userID, strings.NewReader(updateBody))
	req.Header.Set("Authorization", auth)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	// ── Delete ───────────────────────────────────
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+userID, nil)
	req.Header.Set("Authorization", auth)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", rec.Code)
	}

	// ── Get after delete → 404 ───────────────────
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID, nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("get deleted: expected 404, got %d", rec.Code)
	}
}

func TestErrorResponses(t *testing.T) {
	srv := NewServer()

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		auth       string
		wantStatus int
		wantCode   ErrCode
	}{
		{
			name:       "unauthorized without token",
			method:     http.MethodPost,
			path:       "/api/v1/users",
			body:       `{"name":"Bob","email":"bob@example.com"}`,
			wantStatus: http.StatusUnauthorized,
			wantCode:   ErrUnauthorized,
		},
		{
			name:       "unauthorized with bad token",
			method:     http.MethodPost,
			path:       "/api/v1/users",
			body:       `{"name":"Bob","email":"bob@example.com"}`,
			auth:       "Bearer wrong-token",
			wantStatus: http.StatusUnauthorized,
			wantCode:   ErrUnauthorized,
		},
		{
			name:       "invalid JSON body",
			method:     http.MethodPost,
			path:       "/api/v1/users",
			body:       `{not json}`,
			auth:       "Bearer demo-token",
			wantStatus: http.StatusBadRequest,
			wantCode:   ErrInvalidJSON,
		},
		{
			name:       "validation: missing required fields",
			method:     http.MethodPost,
			path:       "/api/v1/users",
			body:       `{"age":25}`,
			auth:       "Bearer demo-token",
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   ErrValidationFailed,
		},
		{
			name:       "validation: invalid email",
			method:     http.MethodPost,
			path:       "/api/v1/users",
			body:       `{"name":"Bob","email":"not-email"}`,
			auth:       "Bearer demo-token",
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   ErrValidationFailed,
		},
		{
			name:       "not found user",
			method:     http.MethodGet,
			path:       "/api/v1/users/nonexistent",
			wantStatus: http.StatusNotFound,
			wantCode:   ErrNotFound,
		},
		{
			name:       "delete not found",
			method:     http.MethodDelete,
			path:       "/api/v1/users/nonexistent",
			auth:       "Bearer demo-token",
			wantStatus: http.StatusNotFound,
			wantCode:   ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}

			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			var errResp ErrorResponse
			if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if errResp.Error.Code != tt.wantCode {
				t.Errorf("error code: got %s, want %s", errResp.Error.Code, tt.wantCode)
			}
		})
	}
}

func TestIdempotency(t *testing.T) {
	srv := NewServer()
	auth := "Bearer demo-token"
	body := `{"name":"Idempotent User","email":"idem@example.com"}`

	// 第一次请求
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req.Header.Set("Authorization", auth)
	req.Header.Set("Idempotency-Key", "unique-key-123")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("first request: expected 201, got %d", rec.Code)
	}
	firstBody := rec.Body.String()

	// 第二次请求（相同 Idempotency-Key）
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req.Header.Set("Authorization", auth)
	req.Header.Set("Idempotency-Key", "unique-key-123")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("second request: expected 201, got %d", rec.Code)
	}
	if rec.Header().Get("X-Idempotent-Replayed") != "true" {
		t.Error("second request: expected X-Idempotent-Replayed header")
	}

	// 响应内容应该相同
	var first, second Response[User]
	_ = json.Unmarshal([]byte(firstBody), &first)
	_ = json.Unmarshal(rec.Body.Bytes(), &second)
	if first.Data.ID != second.Data.ID {
		t.Errorf("idempotent response mismatch: %s vs %s", first.Data.ID, second.Data.ID)
	}
}

func TestCORS(t *testing.T) {
	srv := NewServer()

	// CORS headers are set on normal requests too (not just OPTIONS preflight).
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS Allow-Origin header")
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("missing CORS Allow-Methods header")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		wantErr bool
		fields  []string // 期望出错的字段
	}{
		{
			name: "valid request",
			input: CreateUserRequest{
				Name:  "Alice",
				Email: "alice@example.com",
				Age:   30,
			},
			wantErr: false,
		},
		{
			name:    "missing name and email",
			input:   CreateUserRequest{Age: 25},
			wantErr: true,
			fields:  []string{"name", "email"},
		},
		{
			name: "name too short",
			input: CreateUserRequest{
				Name:  "A",
				Email: "a@example.com",
			},
			wantErr: true,
			fields:  []string{"name"},
		},
		{
			name: "invalid email format",
			input: CreateUserRequest{
				Name:  "Bob",
				Email: "not-an-email",
			},
			wantErr: true,
			fields:  []string{"email"},
		},
		{
			name: "age too large",
			input: CreateUserRequest{
				Name:  "Old",
				Email: "old@example.com",
				Age:   200,
			},
			wantErr: true,
			fields:  []string{"age"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(tt.input)
			hasErr := len(errs) > 0
			if hasErr != tt.wantErr {
				t.Errorf("Validate() hasErr = %v, want %v; errs = %v", hasErr, tt.wantErr, errs)
			}
			for _, field := range tt.fields {
				if _, ok := errs[field]; !ok {
					t.Errorf("expected error for field %q, got: %v", field, errs)
				}
			}
		})
	}
}
