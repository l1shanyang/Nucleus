# 阶段二：应用启动与配置体系

> 目标：把项目从"能跑"升级成"可管理、可配置、可部署"。

## 一、本次改动总览

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/config/config.go` | 改写 | 拆分为 App/HTTP/Database/Log 四组，增加校验和环境判断方法 |
| `.env.example` | 改写 | 新增所有可配置项及说明 |
| `internal/app/app.go` | 新增 | 应用层：组装依赖、管理生命周期 |
| `internal/db/pool.go` | 改写 | 包装为 Pool 类型，暴露 Ping/Close/DB |
| `internal/http/handler/health.go` | 改写 | 拆分为 liveness (/healthz) 和 readiness (/readyz) |
| `internal/http/router/router.go` | 改写 | 路由注册两个健康检查端点 |
| `cmd/api/main.go` | 改写 | 瘦身为 22 行纯入口 |

## 二、逐项讲解

### 2.1 增强 config — 从硬编码到可配置

**改之前：**

```go
type Config struct {
    AppEnv      string
    HTTPPort    string
    DatabaseURL string
}
```

只有 3 个字段，超时、连接池大小全部硬编码在代码里。每次调整都要改代码、重新编译。

**改之后：**

```go
type Config struct {
    App      AppConfig      // 环境
    HTTP     HTTPConfig     // 服务端超时
    Database DatabaseConfig // 连接池参数
    Log      LogConfig      // 日志级别
}
```

所有值从环境变量读取，有合理默认值。不设环境变量也能跑起来，设了就覆盖。

**关键设计 — 辅助函数：**

```go
func getInt(key string, fallback int) int
func getDuration(key string, fallback time.Duration) time.Duration
```

环境变量都是 string，这两个函数负责类型转换。解析失败时静默使用默认值，不会因为 `DB_MAX_OPEN_CONNS=abc` 就崩溃。

**校验函数 `validate()`：**

```go
func (c Config) validate() error {
    // DATABASE_URL 必填
    // APP_ENV 必须是 local/test/production
    // LOG_LEVEL 必须是 debug/info/warn/error
    // HTTP_PORT 必须是合法端口号
}
```

校验在 `Load()` 里完成，启动时就能发现问题，而不是运行时才报错。

### 2.2 环境判断方法

```go
func (a AppConfig) IsLocal() bool      { return a.Env == "local" }
func (a AppConfig) IsTest() bool       { return a.Env == "test" }
func (a AppConfig) IsProduction() bool { return a.Env == "production" }
```

业务代码中按环境分支：

```go
if cfg.App.IsProduction() {
    // 不返回详细错误信息
} else {
    // 返回完整错误堆栈方便调试
}
```

比到处写 `cfg.App.Env == "production"` 更清晰，改环境名时只需改一处。

### 2.3 抽出 app 层 — main.go 从 73 行到 22 行

**核心思想：职责分离。**

```
main.go    → 加载配置，创建 App，运行
app.App    → 组装依赖（db、handler、router、server）
app.Run()  → 管理生命周期（启动、信号监听、关闭）
```

**App 结构体：**

```go
type App struct {
    cfg    config.Config    // 配置
    pool   *db.Pool         // 数据库
    server *http.Server     // HTTP 服务
}
```

所有依赖在一个地方可见。以后加 Redis、消息队列，都加在这个结构体里。

**New() vs Run()：**

- `New()` — 组装：创建 pool、创建 handler、创建 server。任何一个环节失败就返回错误。
- `Run()` — 运行：启动 server、等待信号、执行 shutdown。

分离的好处：测试时可以只调 `New()` 检查组装是否正确，不需要真的启动 server。

**db.Pool 包装类型：**

```go
type Pool struct {
    inner *pgxpool.Pool
}

func (p *Pool) Ping(ctx context.Context) error
func (p *Pool) Close()
func (p *Pool) DB() *pgxpool.Pool
```

不直接暴露 pgxpool.Pool 的全部方法，只暴露需要的。好处：
- 后续加日志、metrics 只改这一个地方
- 业务代码不依赖 pgx 的具体类型
- 实现接口（如 `handler.Pinger`）更自然

### 2.4 依赖生命周期 — 关闭顺序很重要

```
启动顺序：config → db pool → handler → router → http server
关闭顺序：http server → db pool（反序）
```

为什么反序？因为 server 可能还有正在处理的请求，这些请求可能还在用数据库连接。先关 server（停止接受新请求），等在途请求处理完，再关数据库连接。

```go
func (a *App) shutdown() error {
    // 1. 停止 HTTP server（拒绝新请求，等待在途请求完成）
    a.server.Shutdown(ctx)
    // 2. 关闭数据库连接池
    a.pool.Close()
}
```

### 2.5 Liveness vs Readiness — 两个探针的区别

| | Liveness (/healthz) | Readiness (/readyz) |
|---|---|---|
| 回答的问题 | 进程还活着吗？ | 能处理请求吗？ |
| 检查什么 | 什么都不检查 | 检查数据库连接 |
| 失败意味着 | 进程卡死，需要重启 | 依赖不可用，等一等再试 |
| K8s 行为 | 重启容器 | 从 Service 摘除，不重启 |

```go
// Liveness — 只要进程能响应就 OK
func (h *HealthHandler) Live(w http.ResponseWriter, _ *http.Request) {
    writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

// Readiness — 检查数据库
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
    if err := h.db.Ping(r.Context()); err != nil {
        writeJSON(w, http.StatusServiceUnavailable, ...)
        return
    }
    writeJSON(w, http.StatusOK, ...)
}
```

**Pinger 接口：**

```go
type Pinger interface {
    Ping(ctx context.Context) error
}
```

HealthHandler 不直接依赖 db.Pool，而是依赖 Pinger 接口。这样：
- 测试时可以用 mock
- 以后加 Redis 健康检查，只需扩展接口

## 三、验证

```bash
# 确认编译通过
go build ./...

# 运行服务后测试
curl http://localhost:8080/healthz     # {"status":"ok"}
curl http://localhost:8080/readyz      # {"status":"ok","database":"ok"}

# 停掉数据库后再测试 readiness
curl http://localhost:8080/readyz      # 503 {"status":"not_ready","database":"unavailable"}
```

## 四、本阶段学到了什么

| 知识点 | 说明 |
|--------|------|
| 结构化配置 | 把配置按职责分组，而不是一个大扁平结构 |
| 环境变量驱动 | 所有参数从环境变量读取，代码不含环境特定值 |
| 启动校验 | 在 `Load()` 阶段就发现配置错误，不要等到运行时 |
| App 层 | 把初始化逻辑从 main.go 拆出来，main 只做入口 |
| 依赖包装 | 不直接暴露第三方库的类型，用自定义类型包一层 |
| Graceful shutdown | 先停 HTTP 再关 DB，顺序不能反 |
| Liveness vs Readiness | 两个探针回答不同的问题，K8s 用法不同 |
| 接口解耦 | HealthHandler 依赖 Pinger 接口，不依赖具体实现 |

## 五、下一步

进入 [阶段三：HTTP 基础设施](03-http-infra.md)，建立统一的 API 规范：响应格式、错误处理、请求校验、中间件。
