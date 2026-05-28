package handler

import (
	"encoding/json"
	"net/http"
)

// SuccessResponse 业务成功响应（单条数据）。
type SuccessResponse struct {
	Data any `json:"data"`
}

// ListResponse 列表响应，带分页元数据。
type ListResponse struct {
	Data any            `json:"data"`
	Meta map[string]any `json:"meta,omitempty"`
}

// ErrorBody 错误响应体。
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse 业务错误响应。
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// writeJSON 写入 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, `{"error":{"code":"INTERNAL","message":"failed to encode response"}}`, http.StatusInternalServerError)
	}
}

// WriteSuccess 写入成功响应。
func WriteSuccess(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, SuccessResponse{Data: data})
}

// WriteList 写入列表响应。
func WriteList(w http.ResponseWriter, data any, meta map[string]any) {
	writeJSON(w, http.StatusOK, ListResponse{Data: data, Meta: meta})
}

// WriteError 写入业务错误响应。
func WriteError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, ErrorResponse{
		Error: ErrorBody{Code: code, Message: message},
	})
}
