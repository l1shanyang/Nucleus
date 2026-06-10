# FlowDesk 项目路线

FlowDesk 是基于当前 Nucleus 脚手架继续演进的后端练习项目。目标不是做一个全新的商业产品，而是用一个真实度足够高的 SaaS 化工单 / 工作流系统，系统性覆盖后端开发中的常见场景。

一句话定位：

> 一个支持多组织、多角色、工单流转、协作评论、审计日志、通知和报表的后端系统。

## 学习目标

- 熟悉从需求拆解到接口、数据库、业务规则、测试、文档的完整后端开发流程。
- 练习真实业务闭环，而不是按技术层堆功能。
- 建立可以在面试中讲清楚的工程经验：多租户、权限、事务、状态机、异步任务、报表查询、可观测性。
- 保持每次迭代足够小，让学习者亲自完成关键代码，Codex 只负责引导、拆解、审查和解释。

## 协作方式

后续每个任务按这个节奏推进：

1. 明确业务输入、业务规则、数据库状态变化、接口输出。
2. 先写最小设计：表结构、接口契约、权限边界、错误场景。
3. 学习者完成一个小切片的代码。
4. Codex 做代码审查、补充遗漏、解释关键取舍。
5. 补齐测试、OpenAPI 或 README。
6. 运行 `make test` 或更合适的验证命令。

Codex 不应该一次性独立完成完整业务模块。每次只引导一个可验证切片，例如“注册用户”、“创建 Workspace”、“创建工单”、“状态流转写 Activity”。

## 核心业务模型

第一版只保留这些核心对象：

```text
User          用户
Workspace     租户 / 组织
Member        用户在组织内的成员身份
Role          Owner / Admin / Agent / Viewer
Ticket        工单
Comment       工单评论
Activity      操作日志
Assignment    工单负责人
Notification  通知
```

不要一开始设计通用工作流引擎。先实现固定工单状态机：

```text
open -> in_progress -> resolved -> closed
open -> closed
in_progress -> open
resolved -> in_progress
```

可配置工作流放到最后阶段，等固定工单系统稳定后再升级。

## 阶段 0：工程基线

目标：让项目像真实后端项目，而不是临时 demo。

当前仓库已经基本完成这一阶段：

- 配置加载：环境变量、数据库连接、服务端口。
- 健康检查接口：`GET /healthz`。
- 数据库迁移命令。
- sqlc 生成流程。
- 统一错误响应格式。
- 请求日志 middleware。
- panic recovery middleware。
- 基础测试结构。
- Makefile 命令：`test`、`lint`、`migrate`、`rollback`、`sqlc`、`check`。
- README：启动方式、环境变量、数据库迁移、接口说明。

后续进入业务阶段前，可以补充或确认：

- OpenAPI 是否和当前 demo 接口保持一致。
- `notes` 是否继续作为 demo 保留，还是在业务模块稳定后移除。

## 阶段 1：认证与多租户

目标：建立 SaaS 系统的基础身份模型。

功能范围：

- 用户注册。
- 用户登录。
- 密码哈希。
- JWT 或 Session。
- 创建 Workspace。
- 添加成员，后续再升级为邀请成员。
- 成员角色：Owner、Admin、Agent、Viewer。
- 当前用户可以切换 Workspace。
- 所有业务数据必须带 `workspace_id`。

核心接口：

```text
POST /auth/register
POST /auth/login
GET  /me

POST   /workspaces
GET    /workspaces
GET    /workspaces/{workspaceID}
POST   /workspaces/{workspaceID}/members
GET    /workspaces/{workspaceID}/members
PATCH  /workspaces/{workspaceID}/members/{memberID}/role
DELETE /workspaces/{workspaceID}/members/{memberID}
```

重点练习：

- 鉴权 middleware。
- workspace scope 校验。
- RBAC 权限判断。
- 数据隔离。
- 数据库唯一约束设计。
- 事务处理。

建议切片：

1. 只做 `POST /auth/register`，包含密码哈希、唯一邮箱约束、用户入库、测试。
2. 再做 `POST /auth/login`，包含密码校验、token 签发、错误响应。
3. 增加鉴权 middleware 和 `GET /me`。
4. 创建 Workspace，并让创建者自动成为 Owner。
5. 增加成员列表和角色变更。

## 阶段 2：工单最小闭环

目标：让系统产生核心业务价值。

功能范围：

- 创建工单。
- 查看工单列表。
- 查看工单详情。
- 更新标题、描述、优先级。
- 指派负责人。
- 修改状态。
- 关闭工单。
- 工单列表支持分页、筛选、排序。

工单字段建议：

```text
id
workspace_id
number
title
description
status
priority
created_by
assignee_id
created_at
updated_at
resolved_at
closed_at
```

核心接口：

```text
POST   /workspaces/{workspaceID}/tickets
GET    /workspaces/{workspaceID}/tickets
GET    /workspaces/{workspaceID}/tickets/{ticketID}
PATCH  /workspaces/{workspaceID}/tickets/{ticketID}
PATCH  /workspaces/{workspaceID}/tickets/{ticketID}/assignee
POST   /workspaces/{workspaceID}/tickets/{ticketID}/transitions
```

状态流转接口只接收动作，不允许前端直接写状态：

```json
{
  "action": "start_progress"
}
```

重点练习：

- 业务状态机。
- 分页查询。
- 动态筛选。
- 局部更新。
- SQL 查询优化。
- 后续可加入 `version` 字段练习乐观并发控制。

## 阶段 3：评论、附件与操作日志

目标：引入协作系统常见能力。

功能范围：

- 给工单添加评论。
- 评论列表。
- 编辑 / 删除自己的评论。
- 工单操作日志。
- 记录状态变更、指派变更、标题变更、优先级变更。
- 附件第一版只保存元数据，不急着接对象存储。

核心接口：

```text
POST   /workspaces/{workspaceID}/tickets/{ticketID}/comments
GET    /workspaces/{workspaceID}/tickets/{ticketID}/comments
PATCH  /workspaces/{workspaceID}/comments/{commentID}
DELETE /workspaces/{workspaceID}/comments/{commentID}

GET    /workspaces/{workspaceID}/tickets/{ticketID}/activities
```

重点练习：

- append-only 事件记录。
- 审计日志。
- 事务内写业务数据和 Activity。
- 权限细分：谁能编辑评论、谁能删除评论。
- 文件元数据建模。

## 阶段 4：通知与异步任务

目标：进入真实后端工程能力区。

功能范围：

- 工单被指派时通知负责人。
- 评论中提及成员时通知。
- 工单状态变化时通知相关人。
- 通知已读 / 未读。
- 异步发送邮件，第一版可以 mock。
- 后台 worker 处理通知任务。

第一步先使用数据库 outbox，不急着引入 Redis 或消息队列：

```text
notification_events
id
workspace_id
event_type
payload
status: pending / processing / sent / failed
retry_count
next_retry_at
```

第二步再考虑 Redis、Asynq、NATS 或 RabbitMQ。

重点练习：

- Outbox Pattern。
- 后台 worker。
- 失败重试。
- 幂等处理。
- 定时扫描任务。
- 事务边界设计。

## 阶段 5：报表与查询优化

目标：练习复杂查询、聚合统计和缓存策略。

功能范围：

- 工单数量统计。
- 不同状态分布。
- 不同优先级分布。
- 成员处理工单数量。
- 平均解决时长。
- 超时工单数量。
- 最近 7 / 30 天趋势。

核心接口：

```text
GET /workspaces/{workspaceID}/reports/overview
GET /workspaces/{workspaceID}/reports/tickets/status
GET /workspaces/{workspaceID}/reports/tickets/priority
GET /workspaces/{workspaceID}/reports/agents/performance
```

重点练习：

- SQL 聚合。
- 索引设计。
- 查询性能分析。
- 缓存策略。
- 读模型设计。
- 报表接口与业务事实源分离。

报表是 read model，不能让报表表成为业务事实源。

## 阶段 6：高级权限、安全与可运维

目标：让项目更像生产系统。

功能范围：

- API rate limit。
- refresh token。
- 密码重置流程，邮件可以 mock。
- 操作审计后台。
- workspace 配额限制，例如成员数、工单数。
- 软删除。
- 数据导出。
- OpenAPI 文档。
- Docker Compose 一键启动。
- GitHub Actions：test + lint。
- 结构化日志。
- metrics：请求耗时、错误数、worker 成功 / 失败数。

重点练习：

- 安全边界。
- 可观测性。
- CI。
- 部署体验。
- 后端项目工程化。

## 阶段 7：可配置工作流

目标：把固定状态机升级为可配置工作流。

新增概念：

```text
Workflow
WorkflowState
WorkflowTransition
TicketWorkflowInstance
```

能力范围：

- workspace 可以创建自定义流程。
- 每个流程有多个状态。
- 每个状态允许指定下一步。
- transition 可以限制角色。
- ticket 绑定一个 workflow。

这个阶段复杂度高，但简历价值也高。应等固定工单系统稳定后再做，否则早期会被抽象拖慢。

## 推荐开发顺序

```text
0. 工程基线
1. 用户认证
2. Workspace / Member / Role
3. Ticket CRUD
4. Ticket 状态流转
5. Comment
6. Activity Log
7. Notification Outbox
8. Worker
9. Report
10. Permission Hardening
11. Docker / CI / OpenAPI
12. 可配置工作流
```

每一步都要配套：

- migration
- sqlc query
- handler
- service / usecase
- test
- README 或接口文档更新

## 代码分层建议

保持当前项目的简单分层，不提前引入复杂 DDD：

```text
cmd/api
internal/config
internal/app
internal/http
internal/auth
internal/workspace
internal/ticket
internal/comment
internal/activity
internal/notification
internal/report
internal/store
sql/migrations
sql/queries
```

当前项目已经采用 `handler -> service -> store -> sqlc/db` 调用链。业务扩展优先复用这条链路：

- handler 只处理 HTTP 协议转换。
- service 承载业务规则和流程编排。
- store 封装数据库访问。
- sqlc 生成代码不手改。

## 两周最小闭环

近期目标：

> 两周内完成“认证 + Workspace + 工单创建 / 流转 + Activity Log”这个最小闭环。

完成后，项目就具备一个 SaaS 后端雏形：

- 有用户身份。
- 有租户边界。
- 有核心业务对象。
- 有状态流转规则。
- 有事务一致性。
- 有审计日志可追踪业务变化。

## 简历表述方向

项目名称：

```text
FlowDesk - Multi-tenant Workflow Ticket System
```

可讲亮点：

- 使用 Go、Chi、pgx、sqlc、PostgreSQL 构建多租户工单系统。
- 设计 Workspace 级数据隔离与 RBAC 权限模型。
- 实现工单状态机，约束非法状态流转。
- 使用事务保证工单更新、评论、审计日志的一致性。
- 基于 Outbox Pattern 实现异步通知与失败重试。
- 设计报表接口，支持工单状态分布、成员绩效和解决时长统计。
- 使用 Docker Compose、golang-migrate、golangci-lint、CI 流程提升工程化质量。
