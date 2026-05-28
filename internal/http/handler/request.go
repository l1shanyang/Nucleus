package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

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
