# 阶段三：HTTP 基础设施

> 目标：建立所有业务都能复用的 API 层规范。

## 一、本次改动总览

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/http/handler/response.go` | 改写 | 统一 envelope 响应结构 |
| `internal/http/handler/errors.go` | 新增 | AppError 类型 + WrapHandler 错误捕获 |
| `internal/http/handler/request.go` | 新增 | DecodeJSON / TrimString / Require |
| `internal/http/handler/notes.go` | 改写 | 适配新的 error-return 模式 |
| `internal/http/middleware/cors.go` | 新增 | CORS 跨域中间件 |
| `internal/http/middleware/security.go` | 新增 | 安全响应头中间件 |
| `internal/http/router/router.go` | 改写 | 路由分组 + 挂载中间件 |

## 二、逐项讲解

### 2.1 统一响应格式 — 所有 API 返回一致的结构

**改之前：** 每个 handler 自己决定返回什么格式，前端无法统一处理。

**改之后：** 三种标准响应结构：

```json
// 成功（单条）
{"data": {"id": 1, "title": "...", "body": "...", "created_at": "..."}}

// 成功（列表）
{"data": [...], "meta": {"limit": 20, "offset": 0}}

// 错误
{"error": {"code": "VALIDATION_ERROR", "message": "title is required"}}
```

**为什么要 envelope？**

没有 envelope 时，前端拿到 `{"id":1,...}` 不知道这是成功还是错误，得靠 HTTP 状态码区分。有了 envelope：
- 成功一定有 `data` 字段
- 错误一定有 `error` 字段
- 列表带 `meta` 元数据（分页、总数等）

前端可以写一个通用的响应处理函数，不用每个接口单独处理。

**healthz/readyz 为什么不走这个格式？**

它们是运维端点，不是业务 API。K8s、负载均衡器、监控系统只关心 HTTP 状态码，不解析 body。保持简单即可。

### 2.2 统一错误处理 — handler 返回 error

**改之前：** handler 自己调 `WriteError(w, status, code, msg)`，每个错误都要写三行。

**改之后：** handler 只需返回 error：

```go
// 之前
if err != nil {
    WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to create note")
    return
}

// 之后
if err != nil {
    return Internal("failed to create note")
}
```

**AppError 类型：**

```go
type AppError struct {
    Status  int    // HTTP 状态码（不序列化到 JSON）
    Code    string // 业务错误码
    Message string // 人类可读信息
}
```

**构造函数：**

```go
BadRequest("title is required")    // 400 + BAD_REQUEST
NotFound("note not found")         // 404 + NOT_FOUND
Conflict("note already exists")    // 409 + CONFLICT
Internal("database error")         // 500 + INTERNAL
```

**WrapHandler 的作用：**

```go
func WrapHandler(fn func(w, r) error) http.HandlerFunc {
    return func(w, r) {
        if err := fn(w, r); err != nil {
            // AppError → 用它的 Status 和 Code
            // 普通 error → 统一 500，不暴露内部信息
        }
    }
}
```

**关键安全点：** 非 AppError 的普通 error 不会暴露给前端。数据库报错、panic 信息等只返回 `"internal server error"`。

### 2.3 请求解析工具 — 一行完成 decode + 校验

**DecodeJSON：**

```go
func DecodeJSON(r *http.Request, dst any) error {
    r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 限制 1MB
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()  // 拒绝多余字段
    return dec.Decode(dst)
}
```

三个防护：
- **1MB 限制** — 防止恶意大 body 耗尽内存
- **DisallowUnknownFields** — 防止前端传了错误字段名（typo）却静默忽略
- **BadRequest 返回** — 解析失败直接返回 400，handler 不用重复写

**Require：**

```go
Require(map[string]string{"title": req.Title, "body": req.Body})
```

批量校验必填字段，返回格式化好的错误信息。比逐个 if 判断更简洁。

### 2.4 中间件 — CORS 和安全头

**CORS 中间件：**

跨域请求时，浏览器会先发一个 OPTIONS 预检请求。CORS 中间件处理这个请求，告诉浏览器"这个来源是允许的"。

```go
r.Use(middleware.CORS(cfg.HTTP.CORSOrigins...))  // 从配置读取允许来源
```

本地可以配置为 `*`，生产环境应该通过 `CORS_ALLOWED_ORIGINS=https://app.example.com` 指定具体来源。

**安全头中间件：**

| Header | 作用 |
|--------|------|
| `X-Content-Type-Options: nosniff` | 防止浏览器猜测 MIME 类型 |
| `X-Frame-Options: DENY` | 防止页面被嵌入 iframe（点击劫持） |
| `X-XSS-Protection: 0` | 禁用浏览器旧版 XSS 过滤器（CSP 更安全） |
| `Referrer-Policy` | 控制 Referer 头发送策略 |

### 2.5 路由分组

```go
// 全局中间件（所有请求都经过）
r.Use(chimw.RequestID)
r.Use(chimw.RealIP)
r.Use(chimw.Recoverer)
r.Use(chimw.Timeout(30 * time.Second))
r.Use(middleware.SecurityHeaders)
r.Use(middleware.CORS(cfg.HTTP.CORSOrigins...))

// 运维端点
r.Get("/healthz", healthHandler.Live)
r.Get("/readyz", healthHandler.Ready)

// 业务 API
r.Route("/api/v1", func(r chi.Router) {
    r.Post("/notes", handler.WrapHandler(noteHandler.Create))
    r.Get("/notes", handler.WrapHandler(noteHandler.List))
})
```

三层分明：全局中间件 → 运维端点 → 业务 API。以后加新的业务路由，都放在 `/api/v1` 下面。

## 三、本阶段学到了什么

| 知识点 | 说明 |
|--------|------|
| Envelope 模式 | 统一响应结构让前端可以写通用处理逻辑 |
| 错误码体系 | 机器可读的 code + 人类可读的 message |
| Handler 返回 error | 业务逻辑只关注"发生了什么错误"，不关注"怎么写响应" |
| WrapHandler | 统一的错误→响应转换层，安全地隐藏内部错误 |
| 请求校验 | body 限制、未知字段拒绝、必填校验，handler 不重复写 |
| CORS | 跨域请求的浏览器安全机制 |
| 安全头 | 防点击劫持、MIME 嗅探等常见攻击 |

## 四、下一步

进入 [阶段四：架构分层与数据库基础](04-architecture-db.md)，引入 service 层和 store 层，让 handler 不再直接操作数据库。
