package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const (
	DefaultPageLimit = 20
	MaxPageLimit     = 100
	DefaultOffset    = 0
)

// Pagination 是列表接口通用的分页参数。
type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// DecodeJSON 从请求体解析 JSON 到 dst，自动限制 body 大小（1MB）。
func DecodeJSON(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1MB

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return BadRequest("invalid json body")
	}

	return nil
}

// TrimString 清理字符串字段：去首尾空格。
func TrimString(s string) string {
	return strings.TrimSpace(s)
}

// Require 验证必填字段，空值返回 BadRequest。
func Require(fields map[string]string) error {
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			return BadRequest(fmt.Sprintf("%s is required", name))
		}
	}
	return nil
}

// QueryInt 读取整数 query 参数，未传时返回默认值。
func QueryInt(r *http.Request, key string, defaultValue int) (int, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return defaultValue, nil
	}

	v, err := strconv.Atoi(value)
	if err != nil {
		return 0, InvalidParam(key + " must be an integer")
	}

	return v, nil
}

// ParsePagination 解析列表接口通用分页参数。
func ParsePagination(r *http.Request) (Pagination, error) {
	limit, err := QueryInt(r, "limit", DefaultPageLimit)
	if err != nil {
		return Pagination{}, err
	}
	if limit <= 0 {
		return Pagination{}, InvalidParam("limit must be greater than 0")
	}
	if limit > MaxPageLimit {
		limit = MaxPageLimit
	}

	offset, err := QueryInt(r, "offset", DefaultOffset)
	if err != nil {
		return Pagination{}, err
	}
	if offset < 0 {
		return Pagination{}, InvalidParam("offset must be a non-negative integer")
	}

	return Pagination{Limit: limit, Offset: offset}, nil
}
