# API Documentation

`openapi.yaml` 是本项目的 HTTP API 契约文档。它用于帮助前端、后端和调用方对齐接口路径、请求参数、响应结构和错误格式。

## 维护规则

- 新增或修改接口时，同步更新 `openapi.yaml`。
- handler 响应结构变化时，同步更新 `components.schemas`。
- 列表接口统一复用 `PaginationMeta`。
- 错误响应统一复用 `ErrorResponse`。
- `notes` 只是 API 写法 demo，OpenAPI 维护规则面向后续任意业务模块。

## 当前约定

- 成功单条响应：

```json
{
  "data": {}
}
```

- 成功列表响应：

```json
{
  "data": [],
  "meta": {
    "limit": 20,
    "offset": 0
  }
}
```

- 错误响应：

```json
{
  "error": {
    "code": "BAD_REQUEST",
    "message": "invalid json body"
  }
}
```
