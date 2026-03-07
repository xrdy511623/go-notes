package restful

import (
	"encoding/json"
	"net/http"
)

// Response 是标准成功响应信封。
// 使用泛型参数 T 保证 data 字段的类型安全。
type Response[T any] struct {
	Data T     `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

// Meta 包含分页和观测元数据。
type Meta struct {
	Total   int    `json:"total"`
	Page    int    `json:"page"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
}

// AuditFields 定义协议层审计字段。
type AuditFields struct {
	Subject string `json:"subject,omitempty"`
	Tenant  string `json:"tenant,omitempty"`
	Role    string `json:"role,omitempty"`
}

// ErrorResponse 是标准错误响应信封。
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody 包含错误详情。
type ErrorBody struct {
	Code    ErrCode           `json:"code"`
	Message string            `json:"message"`
	Detail  string            `json:"detail,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"` // 字段级校验错误
	TraceID string            `json:"trace_id,omitempty"`
	Metric  string            `json:"metric,omitempty"`
	Audit   *AuditFields      `json:"audit,omitempty"`
}

// writeJSON 将 v 序列化为 JSON 写入 w，设置 Content-Type 和状态码。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteSuccess 写入标准成功响应。
func WriteSuccess[T any](w http.ResponseWriter, status int, data T) {
	writeJSON(w, status, Response[T]{Data: data})
}

// WriteSuccessWithMeta 写入带分页的成功响应。
func WriteSuccessWithMeta[T any](w http.ResponseWriter, data T, meta Meta) {
	if meta.TraceID == "" {
		meta.TraceID = traceIDFromHeader(w)
	}
	writeJSON(w, http.StatusOK, Response[T]{Data: data, Meta: &meta})
}

// WriteError 写入标准错误响应。
func WriteError(w http.ResponseWriter, appErr *AppError) {
	resp := ErrorResponse{
		Error: ErrorBody{
			Code:    appErr.Code,
			Message: appErr.Message,
			Detail:  appErr.Detail,
			TraceID: traceIDFromHeader(w),
			Metric:  "http_request_errors_total",
			Audit:   auditFromHeader(w),
		},
	}
	writeJSON(w, appErr.Code.HTTPStatusCode(), resp)
}

// WriteValidationError 写入字段级校验错误响应。
func WriteValidationError(w http.ResponseWriter, fields map[string]string) {
	resp := ErrorResponse{
		Error: ErrorBody{
			Code:    ErrValidationFailed,
			Message: "request validation failed",
			Fields:  fields,
			TraceID: traceIDFromHeader(w),
			Metric:  "http_request_errors_total",
			Audit:   auditFromHeader(w),
		},
	}
	writeJSON(w, http.StatusUnprocessableEntity, resp)
}

// WriteNoContent 写入 204 无内容响应。
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func traceIDFromHeader(w http.ResponseWriter) string {
	traceID := w.Header().Get("X-Trace-ID")
	if traceID != "" {
		return traceID
	}
	return w.Header().Get("X-Request-ID")
}

func auditFromHeader(w http.ResponseWriter) *AuditFields {
	subject := w.Header().Get("X-Audit-Subject")
	tenant := w.Header().Get("X-Audit-Tenant")
	role := w.Header().Get("X-Audit-Role")
	if subject == "" && tenant == "" && role == "" {
		return nil
	}
	return &AuditFields{Subject: subject, Tenant: tenant, Role: role}
}
