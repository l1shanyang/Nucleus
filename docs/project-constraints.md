# Project Constraints

## 目标

Nucleus 是一个用于学习和长期演进的 Go 后端原子脚手架。重点是帮助有经验的前端开发者理解后端项目的基本开发流程、架构分层和常用技术栈。

## 核心取向

- 简洁优先：只保留所有后端项目都会复用的通用基础能力。
- 学习优先：代码、目录和文档应服务于理解后端架构，而不是追求生产平台完整度。
- 通用优先：不内置社区、电商、后台管理、IM、游戏等具体业务能力。
- 渐进演进：需要某个阶段时再细化，不提前引入重型基础设施。

## 当前边界

- GitHub 仅作为代码存储仓库，不依赖 GitHub Actions 完成 CI。
- 质量检查通过本地 `make check` 完成。
- 工具版本直接在 Makefile 顶部维护，不使用独立 `versions.env`。
- Go 使用 `1.26.x` 级别约束，不做 patch 级强制检查。
- Docker、PostgreSQL、sqlc、migrate、golangci-lint、govulncheck 保持清晰固定版本即可。

## 架构原则

- `cmd/api` 只做程序入口。
- `internal/app` 负责依赖装配和生命周期。
- `handler -> service -> store -> sqlc/db` 是主要调用链。
- handler 只处理 HTTP 协议转换。
- service 承载业务规则和流程编排。
- store 封装数据库访问。
- sqlc 生成代码不手改。

## 后续协作要求

- 不主动引入 GitHub CI、Kubernetes、复杂监控、队列、缓存、多租户等重型能力。
- 不为了“看起来成熟”牺牲脚手架的可读性。
- 每次改造都应能解释它服务哪个学习目标或通用后端能力。
- 文档应简洁，优先解释开发流程和架构逻辑。
