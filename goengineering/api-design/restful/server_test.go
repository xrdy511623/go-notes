package restful

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestPUTIsFullReplaceAndPatchIsPartial(t *testing.T) {
	srv := NewServer()
	auth := "Bearer tenant:t1:user:u1"

	// Create
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"name":"Alice","email":"alice@example.com","age":18}`))
	createReq.Header.Set("Authorization", auth)
	createRec := httptest.NewRecorder()
	srv.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d body=%s", createRec.Code, createRec.Body.String())
	}

	var createResp Response[User]
	if err := json.NewDecoder(createRec.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	id := createResp.Data.ID
	etag := createRec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("create: missing ETag")
	}

	// PUT full replace, age can be 0
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+id, strings.NewReader(`{"name":"Alice Full","email":"full@example.com","age":0}`))
	putReq.Header.Set("Authorization", auth)
	putReq.Header.Set("If-Match", etag)
	putRec := httptest.NewRecorder()
	srv.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("put: expected 200, got %d body=%s", putRec.Code, putRec.Body.String())
	}

	var putResp Response[User]
	_ = json.NewDecoder(putRec.Body).Decode(&putResp)
	if putResp.Data.Age != 0 {
		t.Fatalf("put: expected age=0, got %d", putResp.Data.Age)
	}
	if putResp.Data.Name != "Alice Full" || putResp.Data.Email != "full@example.com" {
		t.Fatalf("put: full replace failed: %+v", putResp.Data)
	}
	newETag := putRec.Header().Get("ETag")
	if newETag == "" || newETag == etag {
		t.Fatalf("put: expected rotated ETag, got old=%q new=%q", etag, newETag)
	}

	// PATCH partial update
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+id, strings.NewReader(`{"name":"Alice Partial"}`))
	patchReq.Header.Set("Authorization", auth)
	patchReq.Header.Set("If-Match", newETag)
	patchRec := httptest.NewRecorder()
	srv.ServeHTTP(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("patch: expected 200, got %d body=%s", patchRec.Code, patchRec.Body.String())
	}

	var patchResp Response[User]
	_ = json.NewDecoder(patchRec.Body).Decode(&patchResp)
	if patchResp.Data.Name != "Alice Partial" {
		t.Fatalf("patch: expected updated name, got %q", patchResp.Data.Name)
	}
	if patchResp.Data.Email != "full@example.com" {
		t.Fatalf("patch: expected email unchanged, got %q", patchResp.Data.Email)
	}
	if patchResp.Data.Age != 0 {
		t.Fatalf("patch: expected age unchanged at 0, got %d", patchResp.Data.Age)
	}
}

func TestIfMatchRequiredAndConflict(t *testing.T) {
	srv := NewServer()
	auth := "Bearer tenant:t1:user:u1"

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"name":"Bob","email":"bob@example.com","age":20}`))
	createReq.Header.Set("Authorization", auth)
	createRec := httptest.NewRecorder()
	srv.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", createRec.Code)
	}

	var created Response[User]
	_ = json.NewDecoder(createRec.Body).Decode(&created)
	id := created.Data.ID

	// Missing If-Match -> 412
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+id, strings.NewReader(`{"name":"Bob2","email":"bob2@example.com","age":0}`))
	putReq.Header.Set("Authorization", auth)
	putRec := httptest.NewRecorder()
	srv.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusPreconditionFailed {
		t.Fatalf("missing if-match: expected 412, got %d body=%s", putRec.Code, putRec.Body.String())
	}

	// Stale If-Match -> 412
	staleReq := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+id, strings.NewReader(`{"name":"stale"}`))
	staleReq.Header.Set("Authorization", auth)
	staleReq.Header.Set("If-Match", `"stale-v1"`)
	staleRec := httptest.NewRecorder()
	srv.ServeHTTP(staleRec, staleReq)
	if staleRec.Code != http.StatusPreconditionFailed {
		t.Fatalf("stale if-match: expected 412, got %d body=%s", staleRec.Code, staleRec.Body.String())
	}
}

func TestRESTListPaginationAndSorting(t *testing.T) {
	srv := NewServer()
	auth := "Bearer tenant:t1:user:u1"

	for i, body := range []string{
		`{"name":"CC","email":"c@example.com","age":30}`,
		`{"name":"AA","email":"a@example.com","age":31}`,
		`{"name":"BB","email":"b@example.com","age":32}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
		req.Header.Set("Authorization", auth)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create %d: expected 201 got %d", i, rec.Code)
		}
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/users?page=2&limit=1&sort=name&order=asc", nil)
	listReq.Header.Set("Authorization", auth)
	listRec := httptest.NewRecorder()
	srv.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list: expected 200 got %d body=%s", listRec.Code, listRec.Body.String())
	}

	var listResp Response[[]User]
	if err := json.NewDecoder(listRec.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if listResp.Meta == nil {
		t.Fatal("list: expected meta")
	}
	if listResp.Meta.Total != 3 || listResp.Meta.Page != 2 || listResp.Meta.Limit != 1 {
		t.Fatalf("list meta mismatch: %+v", listResp.Meta)
	}
	if len(listResp.Data) != 1 || listResp.Data[0].Name != "BB" {
		t.Fatalf("list page mismatch: %+v", listResp.Data)
	}
}

func TestIdempotencyScopeConflictAndTTL(t *testing.T) {
	store := NewInMemoryUserStore()
	h := NewUserHandler(store)
	h.idempotencyTTL = 10 * time.Millisecond
	srv := buildTestServer(h)

	key := "idem-k-1"
	body1 := `{"name":"Idem1","email":"idem1@example.com","age":1}`
	body2 := `{"name":"Idem2","email":"idem2@example.com","age":2}`

	// first request
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body1))
	req1.Header.Set("Authorization", "Bearer tenant:t1:user:u1")
	req1.Header.Set("Idempotency-Key", key)
	rec1 := httptest.NewRecorder()
	srv.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first request: expected 201 got %d", rec1.Code)
	}
	var first Response[User]
	_ = json.NewDecoder(rec1.Body).Decode(&first)

	// same key + same body -> replay
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body1))
	req2.Header.Set("Authorization", "Bearer tenant:t1:user:u1")
	req2.Header.Set("Idempotency-Key", key)
	rec2 := httptest.NewRecorder()
	srv.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusCreated || rec2.Header().Get("X-Idempotent-Replayed") != "true" {
		t.Fatalf("replay: expected 201 + replay header, got %d", rec2.Code)
	}
	var replay Response[User]
	_ = json.NewDecoder(rec2.Body).Decode(&replay)
	if replay.Data.ID != first.Data.ID {
		t.Fatalf("replay id mismatch: %s vs %s", replay.Data.ID, first.Data.ID)
	}

	// same key + different body (same scope) -> conflict
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body2))
	req3.Header.Set("Authorization", "Bearer tenant:t1:user:u1")
	req3.Header.Set("Idempotency-Key", key)
	rec3 := httptest.NewRecorder()
	srv.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusConflict {
		t.Fatalf("idempotency conflict: expected 409 got %d body=%s", rec3.Code, rec3.Body.String())
	}

	// same key + same body but different scope(subject) -> independent
	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body1))
	req4.Header.Set("Authorization", "Bearer tenant:t1:user:u2")
	req4.Header.Set("Idempotency-Key", key)
	rec4 := httptest.NewRecorder()
	srv.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusCreated || rec4.Header().Get("X-Idempotent-Replayed") == "true" {
		t.Fatalf("scope isolation: expected fresh create, got code=%d replay=%q", rec4.Code, rec4.Header().Get("X-Idempotent-Replayed"))
	}

	// wait ttl and retry with original scope -> new resource
	time.Sleep(20 * time.Millisecond)
	req5 := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body1))
	req5.Header.Set("Authorization", "Bearer tenant:t1:user:u1")
	req5.Header.Set("Idempotency-Key", key)
	rec5 := httptest.NewRecorder()
	srv.ServeHTTP(rec5, req5)
	if rec5.Code != http.StatusCreated {
		t.Fatalf("ttl expiry create: expected 201 got %d", rec5.Code)
	}
	var afterTTL Response[User]
	_ = json.NewDecoder(rec5.Body).Decode(&afterTTL)
	if afterTTL.Data.ID == first.Data.ID {
		t.Fatalf("ttl expiry expected new ID, still %s", afterTTL.Data.ID)
	}
}

func TestObjectLevelAuthorizationAndTenantBoundary(t *testing.T) {
	srv := NewServer()

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"name":"Owner","email":"owner@example.com","age":20}`))
	createReq.Header.Set("Authorization", "Bearer tenant:t1:user:owner1")
	createRec := httptest.NewRecorder()
	srv.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201 got %d", createRec.Code)
	}
	var created Response[User]
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	userID := created.Data.ID

	// same tenant but not owner and not admin -> forbidden
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID, nil)
	forbiddenReq.Header.Set("Authorization", "Bearer tenant:t1:user:other")
	forbiddenRec := httptest.NewRecorder()
	srv.ServeHTTP(forbiddenRec, forbiddenReq)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("same tenant non-owner: expected 403 got %d", forbiddenRec.Code)
	}

	// tenant boundary -> forbidden
	crossTenantReq := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID, nil)
	crossTenantReq.Header.Set("Authorization", "Bearer tenant:t2:user:any")
	crossTenantRec := httptest.NewRecorder()
	srv.ServeHTTP(crossTenantRec, crossTenantReq)
	if crossTenantRec.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant: expected 403 got %d", crossTenantRec.Code)
	}

	// same tenant admin -> allowed
	adminReq := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID, nil)
	adminReq.Header.Set("Authorization", "Bearer tenant:t1:admin:boss")
	adminRec := httptest.NewRecorder()
	srv.ServeHTTP(adminRec, adminReq)
	if adminRec.Code != http.StatusOK {
		t.Fatalf("admin access: expected 200 got %d", adminRec.Code)
	}
}

func TestErrorResponseContainsTraceIDAndAudit(t *testing.T) {
	srv := NewServer()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{not-json}`))
	req.Header.Set("Authorization", "Bearer tenant:t1:user:u1")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", rec.Code)
	}
	var errResp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errResp.Error.TraceID == "" {
		t.Fatal("expected error.trace_id")
	}
	if errResp.Error.Audit == nil || errResp.Error.Audit.Subject == "" {
		t.Fatalf("expected error.audit fields, got %+v", errResp.Error.Audit)
	}
}

func TestV1DeprecationHeaders(t *testing.T) {
	srv := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer tenant:t1:user:u1")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Header().Get("Deprecation") != "true" {
		t.Fatal("expected Deprecation header")
	}
	if rec.Header().Get("Sunset") == "" {
		t.Fatal("expected Sunset header")
	}
	if !strings.Contains(rec.Header().Get("Link"), "successor-version") {
		t.Fatalf("expected Link successor-version, got %q", rec.Header().Get("Link"))
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		wantErr bool
		fields  []string
	}{
		{
			name: "valid create",
			input: CreateUserRequest{
				Name:  "Alice",
				Email: "alice@example.com",
				Age:   30,
			},
			wantErr: false,
		},
		{
			name: "valid put with age zero",
			input: ReplaceUserRequest{
				Name:  "Alice",
				Email: "alice@example.com",
				Age:   intPtr(0),
			},
			wantErr: false,
		},
		{
			name:    "put missing required age",
			input:   ReplaceUserRequest{Name: "Alice", Email: "alice@example.com"},
			wantErr: true,
			fields:  []string{"age"},
		},
		{
			name: "invalid patch email",
			input: PatchUserRequest{
				Email: strPtr("bad-email"),
			},
			wantErr: true,
			fields:  []string{"email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(tt.input)
			hasErr := len(errs) > 0
			if hasErr != tt.wantErr {
				t.Errorf("Validate() hasErr=%v want=%v errs=%v", hasErr, tt.wantErr, errs)
			}
			for _, field := range tt.fields {
				if _, ok := errs[field]; !ok {
					t.Errorf("expected error for field %q, got %v", field, errs)
				}
			}
		})
	}
}

func buildTestServer(handler *UserHandler) http.Handler {
	mux := http.NewServeMux()
	limiter := NewRateLimiter(100, 60*time.Second)
	chain := Chain(Recovery, CORS, Trace, Logging, limiter.Middleware, Auth(nil), DeprecationHeaders(time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC), "/api/v2/users"))

	mux.Handle("GET /api/v1/users", chain(http.HandlerFunc(handler.ListUsers)))
	mux.Handle("POST /api/v1/users", chain(http.HandlerFunc(handler.CreateUser)))
	mux.Handle("GET /api/v1/users/{id}", chain(http.HandlerFunc(handler.GetUser)))
	mux.Handle("PUT /api/v1/users/{id}", chain(http.HandlerFunc(handler.ReplaceUser)))
	mux.Handle("PATCH /api/v1/users/{id}", chain(http.HandlerFunc(handler.PatchUser)))
	mux.Handle("DELETE /api/v1/users/{id}", chain(http.HandlerFunc(handler.DeleteUser)))
	return mux
}

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }
