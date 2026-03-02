package restful

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultPageSize       = 20
	maxPageSize           = 100
	defaultSortField      = "created_at"
	defaultSortOrder      = "asc"
	defaultIdempotencyTTL = 24 * time.Hour
)

// User 是用户资源模型。
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int64     `json:"-"`
	TenantID  string    `json:"-"`
	OwnerID   string    `json:"-"`
}

// CreateUserRequest 是创建用户的请求体。
type CreateUserRequest struct {
	Name  string `json:"name"  validate:"required,min=2,max=50"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age"   validate:"min=0,max=150"`
}

// ReplaceUserRequest 对应 PUT 全量替换。
type ReplaceUserRequest struct {
	Name  string `json:"name"  validate:"required,min=2,max=50"`
	Email string `json:"email" validate:"required,email"`
	Age   *int   `json:"age"   validate:"required,min=0,max=150"`
}

// PatchUserRequest 对应 PATCH 部分更新。
type PatchUserRequest struct {
	Name  *string `json:"name"  validate:"min=2,max=50"`
	Email *string `json:"email" validate:"email"`
	Age   *int    `json:"age"   validate:"min=0,max=150"`
}

// UserPatch 是存储层使用的部分更新模型。
type UserPatch struct {
	Name  *string
	Email *string
	Age   *int
}

// UpdateResult 描述更新操作结果。
type UpdateResult int

const (
	UpdateOK UpdateResult = iota
	UpdateNotFound
	UpdatePreconditionFailed
)

// UserStore 定义用户存储接口（依赖倒置）。
type UserStore interface {
	List() []User
	Get(id string) (User, bool)
	Create(user User) User
	Replace(id string, user User, expectedVersion int64) (User, UpdateResult)
	Patch(id string, patch UserPatch, expectedVersion int64) (User, UpdateResult)
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
	user.Version = 1
	s.users[user.ID] = user
	return user
}

func (s *InMemoryUserStore) Replace(id string, user User, expectedVersion int64) (User, UpdateResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.users[id]
	if !ok {
		return User{}, UpdateNotFound
	}
	if existing.Version != expectedVersion {
		return User{}, UpdatePreconditionFailed
	}

	existing.Name = user.Name
	existing.Email = user.Email
	existing.Age = user.Age
	existing.UpdatedAt = time.Now().UTC()
	existing.Version++
	s.users[id] = existing
	return existing, UpdateOK
}

func (s *InMemoryUserStore) Patch(id string, patch UserPatch, expectedVersion int64) (User, UpdateResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.users[id]
	if !ok {
		return User{}, UpdateNotFound
	}
	if existing.Version != expectedVersion {
		return User{}, UpdatePreconditionFailed
	}

	if patch.Name != nil {
		existing.Name = *patch.Name
	}
	if patch.Email != nil {
		existing.Email = *patch.Email
	}
	if patch.Age != nil {
		existing.Age = *patch.Age
	}
	existing.UpdatedAt = time.Now().UTC()
	existing.Version++
	s.users[id] = existing
	return existing, UpdateOK
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

type idempotencyRecord struct {
	Fingerprint string
	StatusCode  int
	Response    []byte
	ETag        string
	ExpiresAt   time.Time
}

// UserHandler 封装用户资源的 HTTP 处理器。
type UserHandler struct {
	store          UserStore
	idempotencyMu  sync.Mutex
	idempotency    map[string]idempotencyRecord // scoped key -> response cache
	idempotencyTTL time.Duration
}

// NewUserHandler 创建 UserHandler。
func NewUserHandler(store UserStore) *UserHandler {
	return &UserHandler{
		store:          store,
		idempotency:    make(map[string]idempotencyRecord),
		idempotencyTTL: defaultIdempotencyTTL,
	}
}

// ListUsers GET /api/v1/users
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	principal, ok := PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, ErrUnauthorizedAccess)
		return
	}

	page, limit, offset, sortField, sortOrder, fieldErrs := parseListQuery(r)
	if len(fieldErrs) > 0 {
		WriteValidationError(w, fieldErrs)
		return
	}

	all := h.store.List()
	users := make([]User, 0, len(all))
	for _, user := range all {
		if canAccessUser(principal, user) {
			users = append(users, user)
		}
	}

	sortUsers(users, sortField, sortOrder)

	total := len(users)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	paged := users[offset:end]

	meta := Meta{
		Total:  total,
		Page:   page,
		Limit:  limit,
		Offset: offset,
	}
	WriteSuccessWithMeta(w, paged, meta)
}

// GetUser GET /api/v1/users/{id}
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	principal, ok := PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, ErrUnauthorizedAccess)
		return
	}

	id := r.PathValue("id")
	user, ok := h.store.Get(id)
	if !ok {
		WriteError(w, ErrUserNotFound)
		return
	}
	if !canAccessUser(principal, user) {
		WriteError(w, ErrAccessDenied)
		return
	}

	w.Header().Set("ETag", userETag(user))
	WriteSuccess(w, http.StatusOK, user)
}

// CreateUser POST /api/v1/users
// 支持 scoped Idempotency-Key + TTL + 请求指纹冲突检查。
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	principal, ok := PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, ErrUnauthorizedAccess)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		WriteError(w, ErrInvalidBody)
		return
	}

	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey != "" {
		scopeKey := buildScopedIdempotencyKey(principal, r.Method, r.URL.Path, idempotencyKey)
		fingerprint := requestFingerprint(r.Method, r.URL.Path, bodyBytes)

		record, hit, conflict := h.lookupIdempotency(scopeKey, fingerprint)
		if conflict {
			WriteError(w, ErrIdempotencyConflict.WithDetail("same Idempotency-Key used with different request body"))
			return
		}
		if hit {
			writeIdempotentReplay(w, record)
			return
		}
	}

	var req CreateUserRequest
	if err := decodeJSON(bodyBytes, &req); err != nil {
		WriteError(w, ErrInvalidBody)
		return
	}
	if errs := Validate(req); len(errs) > 0 {
		WriteValidationError(w, errs)
		return
	}

	user := h.store.Create(User{
		Name:     req.Name,
		Email:    req.Email,
		Age:      req.Age,
		TenantID: principal.TenantID,
		OwnerID:  principal.Subject,
	})

	etag := userETag(user)
	w.Header().Set("Location", "/api/v1/users/"+user.ID)
	w.Header().Set("ETag", etag)

	payload, _ := json.Marshal(Response[User]{Data: user})
	if idempotencyKey != "" {
		scopeKey := buildScopedIdempotencyKey(principal, r.Method, r.URL.Path, idempotencyKey)
		fingerprint := requestFingerprint(r.Method, r.URL.Path, bodyBytes)
		h.storeIdempotency(scopeKey, idempotencyRecord{
			Fingerprint: fingerprint,
			StatusCode:  http.StatusCreated,
			Response:    payload,
			ETag:        etag,
			ExpiresAt:   time.Now().UTC().Add(h.idempotencyTTL),
		})
	}

	writeRawJSON(w, http.StatusCreated, payload)
}

// ReplaceUser PUT /api/v1/users/{id}（全量替换）
func (h *UserHandler) ReplaceUser(w http.ResponseWriter, r *http.Request) {
	principal, ok := PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, ErrUnauthorizedAccess)
		return
	}

	id := r.PathValue("id")
	current, ok := h.store.Get(id)
	if !ok {
		WriteError(w, ErrUserNotFound)
		return
	}
	if !canAccessUser(principal, current) {
		WriteError(w, ErrAccessDenied)
		return
	}

	expectedVersion, appErr := expectedVersionFromIfMatch(r.Header.Get("If-Match"), current)
	if appErr != nil {
		WriteError(w, appErr)
		return
	}

	var req ReplaceUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidBody)
		return
	}
	if errs := Validate(req); len(errs) > 0 {
		WriteValidationError(w, errs)
		return
	}

	updated, result := h.store.Replace(id, User{
		Name:  req.Name,
		Email: req.Email,
		Age:   *req.Age,
	}, expectedVersion)

	switch result {
	case UpdateNotFound:
		WriteError(w, ErrUserNotFound)
		return
	case UpdatePreconditionFailed:
		WriteError(w, ErrPreconditionFailed.WithDetail("If-Match does not match current resource version"))
		return
	}

	w.Header().Set("ETag", userETag(updated))
	WriteSuccess(w, http.StatusOK, updated)
}

// PatchUser PATCH /api/v1/users/{id}（部分更新）
func (h *UserHandler) PatchUser(w http.ResponseWriter, r *http.Request) {
	principal, ok := PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, ErrUnauthorizedAccess)
		return
	}

	id := r.PathValue("id")
	current, ok := h.store.Get(id)
	if !ok {
		WriteError(w, ErrUserNotFound)
		return
	}
	if !canAccessUser(principal, current) {
		WriteError(w, ErrAccessDenied)
		return
	}

	expectedVersion, appErr := expectedVersionFromIfMatch(r.Header.Get("If-Match"), current)
	if appErr != nil {
		WriteError(w, appErr)
		return
	}

	var req PatchUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidBody)
		return
	}
	if errs := Validate(req); len(errs) > 0 {
		WriteValidationError(w, errs)
		return
	}
	if req.Name == nil && req.Email == nil && req.Age == nil {
		WriteValidationError(w, map[string]string{"_": "at least one field must be provided for patch"})
		return
	}

	updated, result := h.store.Patch(id, UserPatch{
		Name:  req.Name,
		Email: req.Email,
		Age:   req.Age,
	}, expectedVersion)

	switch result {
	case UpdateNotFound:
		WriteError(w, ErrUserNotFound)
		return
	case UpdatePreconditionFailed:
		WriteError(w, ErrPreconditionFailed.WithDetail("If-Match does not match current resource version"))
		return
	}

	w.Header().Set("ETag", userETag(updated))
	WriteSuccess(w, http.StatusOK, updated)
}

// DeleteUser DELETE /api/v1/users/{id}
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	principal, ok := PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, ErrUnauthorizedAccess)
		return
	}

	id := r.PathValue("id")
	user, exists := h.store.Get(id)
	if !exists {
		WriteError(w, ErrUserNotFound)
		return
	}
	if !canAccessUser(principal, user) {
		WriteError(w, ErrAccessDenied)
		return
	}

	if !h.store.Delete(id) {
		WriteError(w, ErrUserNotFound)
		return
	}
	WriteNoContent(w)
}

func canAccessUser(principal Principal, user User) bool {
	if principal.TenantID == "" || user.TenantID == "" {
		return false
	}
	if principal.TenantID != user.TenantID {
		return false
	}
	if principal.Role == RoleAdmin {
		return true
	}
	return principal.Subject == user.OwnerID
}

func parseListQuery(r *http.Request) (page, limit, offset int, sortField, sortOrder string, errs map[string]string) {
	errs = make(map[string]string)
	q := r.URL.Query()

	page = 1
	limit = defaultPageSize

	if raw := strings.TrimSpace(q.Get("page")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 {
			errs["page"] = "page must be an integer >= 1"
		} else {
			page = v
		}
	}
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > maxPageSize {
			errs["limit"] = "limit must be an integer between 1 and 100"
		} else {
			limit = v
		}
	}

	offset = (page - 1) * limit
	if raw := strings.TrimSpace(q.Get("offset")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 {
			errs["offset"] = "offset must be an integer >= 0"
		} else {
			offset = v
			page = (offset / limit) + 1
		}
	}

	sortField = strings.TrimSpace(q.Get("sort"))
	if sortField == "" {
		sortField = defaultSortField
	}
	switch sortField {
	case "id", "name", "email", "age", "created_at", "updated_at":
	default:
		errs["sort"] = "sort must be one of: id,name,email,age,created_at,updated_at"
	}

	sortOrder = strings.TrimSpace(strings.ToLower(q.Get("order")))
	if sortOrder == "" {
		sortOrder = defaultSortOrder
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		errs["order"] = "order must be asc or desc"
	}

	if len(errs) == 0 {
		return page, limit, offset, sortField, sortOrder, nil
	}
	return page, limit, offset, sortField, sortOrder, errs
}

func sortUsers(users []User, sortField, sortOrder string) {
	desc := sortOrder == "desc"
	sort.SliceStable(users, func(i, j int) bool {
		cmp := compareUsers(users[i], users[j], sortField)
		if cmp == 0 {
			cmp = strings.Compare(users[i].ID, users[j].ID)
		}
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})
}

func compareUsers(a, b User, field string) int {
	switch field {
	case "id":
		return strings.Compare(a.ID, b.ID)
	case "name":
		return strings.Compare(a.Name, b.Name)
	case "email":
		return strings.Compare(a.Email, b.Email)
	case "age":
		return cmpInt(a.Age, b.Age)
	case "updated_at":
		return cmpTime(a.UpdatedAt, b.UpdatedAt)
	case "created_at":
		fallthrough
	default:
		return cmpTime(a.CreatedAt, b.CreatedAt)
	}
}

func cmpTime(a, b time.Time) int {
	switch {
	case a.Before(b):
		return -1
	case a.After(b):
		return 1
	default:
		return 0
	}
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func userETag(user User) string {
	return "\"" + user.ID + "-v" + strconv.FormatInt(user.Version, 10) + "\""
}

func expectedVersionFromIfMatch(ifMatch string, current User) (int64, *AppError) {
	if strings.TrimSpace(ifMatch) == "" {
		return 0, ErrPreconditionFailed.WithDetail("missing If-Match header")
	}
	if !ifMatchMatches(ifMatch, userETag(current)) {
		return 0, ErrPreconditionFailed.WithDetail("If-Match does not match current resource version")
	}
	return current.Version, nil
}

func ifMatchMatches(rawHeader, etag string) bool {
	tokens := strings.Split(rawHeader, ",")
	for _, token := range tokens {
		t := strings.TrimSpace(token)
		if t == "*" || t == etag {
			return true
		}
	}
	return false
}

func decodeJSON(data []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}

func buildScopedIdempotencyKey(principal Principal, method, path, idemKey string) string {
	return principal.TenantID + "|" + principal.Subject + "|" + method + "|" + path + "|" + idemKey
}

func requestFingerprint(method, path string, body []byte) string {
	sum := sha256.Sum256(append(append([]byte(method+"\n"+path+"\n"), body...), '\n'))
	return hex.EncodeToString(sum[:])
}

func (h *UserHandler) lookupIdempotency(scopedKey, fingerprint string) (idempotencyRecord, bool, bool) {
	h.idempotencyMu.Lock()
	defer h.idempotencyMu.Unlock()

	now := time.Now().UTC()
	h.pruneExpiredIdempotencyLocked(now)

	record, ok := h.idempotency[scopedKey]
	if !ok {
		return idempotencyRecord{}, false, false
	}
	if record.Fingerprint != fingerprint {
		return idempotencyRecord{}, false, true
	}
	return record, true, false
}

func (h *UserHandler) storeIdempotency(scopedKey string, record idempotencyRecord) {
	h.idempotencyMu.Lock()
	defer h.idempotencyMu.Unlock()
	h.pruneExpiredIdempotencyLocked(time.Now().UTC())
	h.idempotency[scopedKey] = record
}

func (h *UserHandler) pruneExpiredIdempotencyLocked(now time.Time) {
	for key, record := range h.idempotency {
		if now.After(record.ExpiresAt) {
			delete(h.idempotency, key)
		}
	}
}

func writeIdempotentReplay(w http.ResponseWriter, record idempotencyRecord) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Idempotent-Replayed", "true")
	if record.ETag != "" {
		w.Header().Set("ETag", record.ETag)
	}
	w.WriteHeader(record.StatusCode)
	_, _ = w.Write(record.Response)
}

func writeRawJSON(w http.ResponseWriter, status int, payload []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}
