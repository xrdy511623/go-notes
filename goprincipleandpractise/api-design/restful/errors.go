package restful

import (
	"fmt"
	"net/http"
)

// ErrCode 定义标准化错误码，横跨 HTTP 和 gRPC 场景。
type ErrCode string

const (
	ErrInvalidJSON      ErrCode = "invalid_json"
	ErrValidationFailed ErrCode = "validation_failed"
	ErrUnauthorized     ErrCode = "unauthorized"
	ErrForbidden        ErrCode = "forbidden"
	ErrNotFound         ErrCode = "not_found"
	ErrConflict         ErrCode = "conflict"
	ErrPrecondition     ErrCode = "precondition_failed"
	ErrRateLimited      ErrCode = "rate_limited"
	ErrInternalError    ErrCode = "internal_error"
)

// AppError 是应用层统一错误类型，同时携带面向用户的消息和内部调试信息。
type AppError struct {
	Code     ErrCode `json:"code"`
	Message  string  `json:"message"`
	Detail   string  `json:"detail,omitempty"`
	internal error   // 不序列化，仅用于日志
}

func (e *AppError) Error() string {
	if e.internal != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.internal)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.internal }

// NewAppError 创建一个 AppError，可选传入底层 error。
func NewAppError(code ErrCode, message string, internal error) *AppError {
	return &AppError{
		Code:     code,
		Message:  message,
		internal: internal,
	}
}

// WithDetail 返回带 detail 字段的副本（immutable 风格）。
func (e *AppError) WithDetail(detail string) *AppError {
	return &AppError{
		Code:     e.Code,
		Message:  e.Message,
		Detail:   detail,
		internal: e.internal,
	}
}

// HTTPStatusCode 将 ErrCode 映射到 HTTP 状态码。
func (c ErrCode) HTTPStatusCode() int {
	switch c {
	case ErrInvalidJSON:
		return http.StatusBadRequest
	case ErrValidationFailed:
		return http.StatusUnprocessableEntity
	case ErrUnauthorized:
		return http.StatusUnauthorized
	case ErrForbidden:
		return http.StatusForbidden
	case ErrNotFound:
		return http.StatusNotFound
	case ErrConflict:
		return http.StatusConflict
	case ErrPrecondition:
		return http.StatusPreconditionFailed
	case ErrRateLimited:
		return http.StatusTooManyRequests
	case ErrInternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// 预定义常用错误，避免重复创建。
var (
	ErrUserNotFound  = NewAppError(ErrNotFound, "user not found", nil)
	ErrInvalidBody   = NewAppError(ErrInvalidJSON, "request body is not valid JSON", nil)
	ErrAccessDenied  = NewAppError(ErrForbidden, "access denied", nil)
	ErrServerFailure = NewAppError(ErrInternalError, "internal server error", nil)
)
