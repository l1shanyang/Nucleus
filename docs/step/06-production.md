# 阶段六：生产化能力

> 目标：让项目具备真实部署和线上运行的基础条件。

## 一、本次改动总览

| 文件 | 操作 | 说明 |
|------|------|------|
| `Dockerfile` | 新增 | 多阶段构建，scratch 运行镜像（10MB） |
| `internal/log/logger.go` | 新增 | 结构化日志（slog），按环境切换格式 |
| `cmd/api/main.go` | 改写 | 启动时初始化 slog，替代 log 包 |
| `internal/app/app.go` | 改写 | 所有日志改为 slog 结构化输出 |
| `.github/workflows/ci.yml` | 新增 | GitHub Actions CI 流程 |
| `Makefile` | 改写 | 增加 docker-build、vuln 命令 |
| `docs/deploy.md` | 新增 | 部署说明文档 |

## 二、逐项讲解

### 2.1 生产 Dockerfile — 多阶段构建

**为什么要多阶段？**

Go 编译出的是静态二进制，运行时不需要 Go 工具链、源码、依赖缓存。多阶段构建把编译和运行分开：

```
阶段 1 (builder): golang:${GO_VERSION}-alpine
  → 安装依赖、编译二进制

阶段 2 (runtime): scratch (~0MB)
  → 只复制编译产物
  → 最终镜像约 10MB
```

**scratch 镜像：** 空镜像，没有 shell、没有包管理器、没有任何系统文件。只有你的二进制和 SSL 证书。

**GO_VERSION：** Dockerfile 通过 build arg 接收 Go 版本，`make docker-build` 从 `versions.env` 注入。这样本地构建和 CI 使用同一套 Go 工具链。

**CGO_ENABLED=0：** 禁用 CGO，确保编译出纯 Go 静态二进制，不依赖系统 C 库。

**SSL 证书：** scratch 没有证书，需要从 builder 阶段复制 `ca-certificates.crt`，否则 HTTPS 请求会失败。

### 2.2 Structured Logging — slog

**改之前：** `log.Printf("server listening on :%s (%s)", port, env)`

输出：`2026/05/29 12:00:00 server listening on :8080 (local)`

**改之后：** `slog.Info("server starting", "port", "8080", "env", "local")`

local 环境输出（文本，人可读）：
```
time=2026-05-29T12:00:00+08:00 level=INFO msg="server starting" app=nucleus env=local port=8080
```

production 环境输出（JSON，机器可解析）：
```json
{"time":"2026-05-29T12:00:00+08:00","level":"INFO","msg":"server starting","app":"nucleus","env":"local","port":"8080"}
```

**为什么用 slog？**
- Go 1.21+ 标准库，不需要引入第三方库
- 结构化 key-value 比字符串拼接更利于日志检索
- JSON 格式可以直接被 ELK、Loki 等日志系统采集
- 按环境自动切换格式（local 用文本，production 用 JSON）

### 2.3 CI 流程 — GitHub Actions

```yaml
jobs:
  test:
    # 0. 从 versions.env 读取工具版本
    # 1. gofmt 检查（代码格式化）
    # 2. go vet（静态分析）
    # 3. go test（带 race 检测的单元测试）
    # 4. go build（编译验证）
    # 5. govulncheck（依赖漏洞扫描）

  lint:
    # 使用 versions.env 中固定版本的 golangci-lint
```

**PostgreSQL service container：** CI 中需要数据库的测试用。GitHub Actions 可以在 job 启动时自动拉起一个 PostgreSQL 容器，测试完自动销毁。

**race 检测：** `go test -race` 会检测并发数据竞争。开发阶段可能不开（慢），但 CI 必须开。

### 2.4 依赖安全扫描 — govulncheck

```bash
make vuln
```

govulncheck 会检查项目依赖中是否有已知的安全漏洞。它从 Go 官方漏洞数据库获取数据。

CI 中也会运行，如果发现漏洞，PR 无法合并。

### 2.5 部署文档

`docs/deploy.md` 包含：
- 构建命令
- 环境变量说明
- Docker 运行方式
- 健康检查端点
- 迁移命令
- 质量门禁命令

## 三、本阶段学到了什么

| 知识点 | 说明 |
|--------|------|
| 多阶段构建 | 编译和运行分离，最小化镜像体积 |
| scratch 镜像 | 空镜像，最安全、最小 |
| slog | Go 标准库结构化日志，JSON 格式利于生产环境 |
| GitHub Actions | CI/CD 自动化，PR 质量守护 |
| 版本固定 | `versions.env` 统一本地、CI、Docker 的工具版本 |
| govulncheck | 依赖漏洞扫描，上线前必查 |
| service container | CI 中使用 PostgreSQL 等外部依赖 |

## 四、项目完整架构

```
nucleus/
├── cmd/api/                  # 程序入口
├── internal/
│   ├── app/                  # 应用组装与生命周期
│   ├── config/               # 配置加载与校验
│   ├── log/                  # 结构化日志
│   ├── version/              # 构建版本信息
│   ├── db/                   # 数据库连接池
│   ├── db/sqlc/              # sqlc 生成代码
│   ├── service/              # 业务逻辑层
│   ├── store/                # 数据访问层
│   │   └── storetest/        # 测试用 mock
│   └── http/
│       ├── handler/          # HTTP 处理器
│       ├── middleware/        # 中间件
│       └── router/           # 路由定义
├── sql/
│   ├── migrations/           # 数据库迁移
│   └── queries/              # SQL 查询定义
├── docs/
│   ├── api/                  # OpenAPI 文档
│   ├── step/                 # 分步讲解文档
│   ├── deploy.md             # 部署说明
│   └── todo.md               # 项目推进方案
├── Dockerfile                # 生产镜像
├── docker-compose.yml        # 本地开发环境
├── versions.env              # 工具版本清单
├── .github/workflows/ci.yml  # CI 流程
└── Makefile                  # 命令入口
```

## 五、六个阶段回顾

| 阶段 | 主题 | 核心收获 |
|------|------|----------|
| 一 | 工程化基础 | .gitignore、Makefile、构建信息、golangci-lint |
| 二 | 启动与配置 | 结构化配置、环境区分、app 层、graceful shutdown |
| 三 | HTTP 基础设施 | 统一响应/错误、请求校验、CORS、安全头 |
| 四 | 架构分层 | handler → service → store → sqlc、事务管理 |
| 五 | 测试与文档 | table-driven test、mock、httptest、OpenAPI |
| 六 | 生产化 | Dockerfile、slog、CI、漏洞扫描、部署文档 |
