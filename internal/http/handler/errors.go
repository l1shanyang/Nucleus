package handler

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError 业务错误，包含 HTTP 状态码和错误码。
type AppError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// --- 常用错误构造函数 ---

func BadRequest(msg string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Code: "BAD_REQUEST", Message: msg}
}

func NotFound(msg string) *AppError {
	return &AppError{Status: http.StatusNotFound, Code: "NOT_FOUND", Message: msg}
}

func Conflict(msg string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: "CONFLICT", Message: msg}
}

func Internal(msg string) *AppError {
	return &AppError{Status: http.StatusInternalServerError, Code: "INTERNAL", Message: msg}
}

// WrapHandler 包装返回 error 的 handler，统一处理错误响应。
func WrapHandler(fn func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			var appErr *AppError
			if errors.As(err, &appErr) {
				WriteError(w, appErr.Status, appErr.Code, appErr.Message)
			} else {
				// 非 AppError 统一返回 500，不暴露内部信息
				WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
			}
		}
	}
}
