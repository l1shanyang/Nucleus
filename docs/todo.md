# Nucleus Todo

本项目是学习型 Go 后端原子脚手架。目标不是追求完整生产平台，而是在保持简洁的前提下，逐步补齐所有业务都会复用的后端基础设施。

项目约束详见 [project-constraints.md](project-constraints.md)。

## 当前状态

已具备：

- 工程命令：`Makefile`
- 配置加载：`internal/config`
- 应用装配和生命周期：`internal/app`
- HTTP 路由、中间件、统一响应：`internal/http`
- 基础错误处理：`AppError`、`ValidationError`
- 分层调用链：`handler -> service -> store -> sqlc/db`
- 数据库连接池和迁移：`internal/db`、`sql/migrations`
- sqlc 查询生成：`sql/queries`、`internal/db/sqlc`
- 本地质量门禁：`make check`
- 文档约束：`docs/project-constraints.md`

## 推进原则

- 每次只推进一个基础设施主题。
- 每个主题都要解释它解决什么后端通用问题。
- 不提前引入 GitHub CI、Kubernetes、复杂监控、队列、缓存、多租户等重型能力。
- 业务模块出现真实需求后再抽象，不为了“成熟感”提前复杂化。
- `notes` 只是 API 写法和分层调用 demo，后续基础设施设计不围绕 notes 的实际业务需求展开。

## 后续顺序

### 1. Request Log + Request ID（已完成）

目标：让每个 HTTP 请求都可追踪、可排查。

计划：

- 增加请求日志中间件。
- 记录 `request_id`、`method`、`path`、`status`、`latency`、`remote_ip`、`user_agent`。
- 将 `X-Request-ID` 回写到响应 header。
- 错误响应可关联 request id。

学习重点：

- middleware 在后端请求链路中的位置。
- request id 如何串联前端报错、后端日志和服务端排查。
- 为什么日志应该结构化，而不是字符串拼接。

验收：

- 请求结束后有结构化访问日志。
- 响应 header 包含 `X-Request-ID`。
- 测试覆盖 request id 和日志中间件的关键行为。

实现：

- 新增 `internal/http/middleware.RequestLog`。
- 复用 chi `RequestID` 生成和读取 request id。
- 在 router 全局中间件链路中接入请求日志。
- 增加 request id header、访问日志字段、默认 200 状态码测试。

### 2. 错误体系下沉（已完成）

目标：让业务错误和 HTTP 协议错误解耦。

计划：

- 扩展 service 层通用错误类型。
- 支持 `ValidationError`、`NotFoundError`、`ConflictError`、`UnauthorizedError`、`ForbiddenError`、`InternalError`。
- HTTP 层统一把 service 错误映射成状态码和错误响应。
- 避免 handler 直接判断所有业务错误细节。

学习重点：

- 为什么 service 层不应该依赖 HTTP。
- 业务错误、系统错误、协议错误的区别。
- 如何避免内部错误泄露给客户端。

验收：

- handler 中不再手写大量错误状态判断。
- service 返回的通用错误可以被 HTTP 层稳定映射。
- 包装后的错误仍可通过 `errors.As` 正确识别。

实现：

- 新增 `internal/apperror`，表达与 HTTP 无关的业务错误语义。
- service 层返回 `apperror.Validation`、`apperror.Internal` 等通用错误。
- HTTP `WrapHandler` 统一把 `apperror.Error` 映射为状态码和错误响应。
- handler 不再识别 service 私有错误类型，只负责调用 service 和写成功响应。

### 3. Request Helper + Pagination（已完成）

目标：沉淀所有 handler 都会复用的请求解析能力。

计划：

- 增加通用 query 参数解析。
- 增加分页参数结构。
- 统一 `limit`、`offset` 默认值和最大值。
- 明确列表响应 `meta` 结构。

学习重点：

- handler 如何保持“薄”。
- 为什么请求解析不应该散落在每个业务 handler 中。
- API 分页规范如何影响前后端协作。

验收：

- `NoteHandler.List` 使用通用分页 helper。
- 列表响应元数据结构稳定。
- 参数解析测试覆盖默认值、非法值、边界值。

实现：

- 新增 `handler.Pagination`、`QueryInt`、`ParsePagination`。
- 统一列表接口默认 `limit=20`、`offset=0`、最大 `limit=100`。
- `NoteHandler.List` 不再手写 query 参数解析。
- `ListResponse.Meta` 使用固定分页结构，不再使用临时 map。

### 4. TxManager（已完成）

目标：为多表写入准备统一事务边界。

计划：

- 在 db/store 层增加轻量事务管理器。
- 提供 `WithTx(ctx, fn)` 风格接口。
- 确保 commit / rollback 逻辑集中管理。
- 保持当前单表 CRUD 简洁，不强行复杂化。

学习重点：

- 什么场景需要事务。
- 为什么事务边界通常由 service 控制。
- 如何让多个 store 共用同一个事务。

验收：

- 有统一事务入口。
- 有事务成功提交和失败回滚测试。
- 后续业务 service 可以自然接入事务。

实现：

- 新增 `internal/db.TxManager` 和 `WithTx(ctx, fn)`。
- 事务开启失败、业务回调失败、提交失败都返回可追踪错误。
- 回调成功时提交事务，回调失败或提交失败时统一回滚。
- 当前单表 Note CRUD 不强行接入事务，保留后续多表写入时使用。

### 5. 数据库错误映射（已完成）

目标：隔离 pgx/PostgreSQL 底层错误，向 service 暴露稳定错误语义。

计划：

- 映射 `pgx.ErrNoRows`。
- 映射 PostgreSQL 唯一约束冲突。
- 映射外键约束冲突。
- 保留原始错误用于日志排查。

学习重点：

- 数据库约束如何转成业务语义。
- 为什么不能把底层数据库错误直接返回给客户端。
- store 层在错误隔离中的职责。

验收：

- store 层返回稳定业务错误。
- handler 不感知 pgx / pgconn 细节。
- 测试覆盖常见数据库错误映射。

实现：

- 新增 store 层数据库错误映射 helper。
- 将 `pgx.ErrNoRows` 映射为 `apperror.NotFound`。
- 将 PostgreSQL 唯一约束和外键约束错误映射为 `apperror.Conflict`。
- service 层遇到已映射的 `apperror.Error` 时直接向上返回，未知错误才包装为 `Internal`。

### 6. Store 集成测试基础（已完成）

目标：打通真实 PostgreSQL 下的数据库测试能力。

计划：

- 增加 `internal/db/dbtest` 或等价测试 helper。
- 使用测试数据库跑 migration。
- 为 `NoteStore` 增加真实数据库集成测试。
- 明确本地执行方式。

学习重点：

- 单元测试和集成测试的区别。
- migration、sqlc、pgx 在测试中的协作方式。
- 如何清理测试数据。

验收：

- 可以本地运行 store 集成测试。
- 测试使用真实 PostgreSQL。
- 不影响普通 `make test` 的轻量体验，必要时单独命令运行。

实现：

- 新增 `internal/db/dbtest`，统一连接测试数据库、执行 migration、清理测试表。
- 新增带 `integration` build tag 的 store 集成测试。
- 新增 `make test-integration`，通过 `TEST_DATABASE_URL` 显式指定测试数据库。
- 普通 `make test` 不运行集成测试，保持日常反馈轻量。

### 7. OpenAPI 维护规范（已完成）

目标：让 API 文档成为前后端协作契约。

计划：

- 规范新增接口时如何更新 `docs/api/openapi.yaml`。
- 统一错误响应引用。
- 统一分页响应 schema。
- 在 README 或 docs 中说明维护规则。

学习重点：

- OpenAPI 在前后端协作中的作用。
- 为什么接口文档应该和代码一起演进。
- 手写 OpenAPI 的边界和成本。

验收：

- 当前 notes 接口文档和实际响应一致。
- 错误和分页 schema 可复用。
- 有简短维护说明。

实现：

- 补充 `docs/api/README.md`，说明 OpenAPI 维护规则。
- `openapi.yaml` 增加可复用 `PaginationMeta` 和 `ErrorBody`。
- notes demo 的列表分页、错误响应和当前代码结构保持一致。
- README 增加 API 契约文档入口。

### 8. Version Endpoint

目标：暴露构建信息，理解 Go 二进制构建参数。

计划：

- 增加 `GET /version`。
- 返回 `version`、`commit`、`build_time`、`go_version`。
- 复用已有 `internal/version`。

学习重点：

- `-ldflags` 如何向 Go 二进制注入构建信息。
- 如何确认当前运行的是哪个构建版本。
- 运维端点和业务 API 的边界。

验收：

- `/version` 返回构建信息。
- OpenAPI 同步更新。
- 有 handler 测试。

## 暂不推进

以下能力暂时不作为当前脚手架目标：

- GitHub CI
- Kubernetes
- Prometheus metrics
- OpenTelemetry tracing
- Redis cache
- 消息队列
- 多租户
- 认证权限系统
- 后台管理
- 复杂代码生成框架

这些能力不是不重要，而是等真实业务需要时再引入。
