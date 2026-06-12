package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"

	"nucleus/internal/apperror"
)

// AppError 表示 HTTP 层可以直接转换为响应的错误。
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

func InvalidParam(msg string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Code: "INVALID_PARAM", Message: msg}
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
			status, code, message := errorResponse(err)
			logHandlerError(r, status, code, err)
			WriteError(w, status, code, message)
		}
	}
}

func errorResponse(err error) (status int, code, message string) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Status, appErr.Code, appErr.Message
	}

	var serviceErr *apperror.Error
	if errors.As(err, &serviceErr) {
		return mapAppError(serviceErr)
	}

	// 非预期错误统一返回 500，不暴露内部信息
	return http.StatusInternalServerError, "INTERNAL", "internal server error"
}

func logHandlerError(r *http.Request, status int, code string, err error) {
	attrs := []slog.Attr{
		slog.String("request_id", chimw.GetReqID(r.Context())),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Int("status", status),
		slog.String("code", code),
		slog.String("error", err.Error()),
	}

	level := slog.LevelWarn
	if status >= http.StatusInternalServerError {
		level = slog.LevelError
	}

	slog.LogAttrs(r.Context(), level, "http handler error", attrs...)
}

func mapAppError(err *apperror.Error) (status int, code, message string) {
	switch err.Kind {
	case apperror.KindValidation:
		return http.StatusBadRequest, err.Code, err.Message
	case apperror.KindNotFound:
		return http.StatusNotFound, err.Code, err.Message
	case apperror.KindConflict:
		return http.StatusConflict, err.Code, err.Message
	case apperror.KindUnauthorized:
		return http.StatusUnauthorized, err.Code, err.Message
	case apperror.KindForbidden:
		return http.StatusForbidden, err.Code, err.Message
	case apperror.KindInternal:
		return http.StatusInternalServerError, "INTERNAL", safeInternalMessage(err.Message)
	default:
		return http.StatusInternalServerError, "INTERNAL", "internal server error"
	}
}

func safeInternalMessage(message string) string {
	if message == "" {
		return "internal server error"
	}
	return message
}
