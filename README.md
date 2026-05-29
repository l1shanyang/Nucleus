# Nucleus

一个用于学习和长期演进的 Go 后端项目骨架。技术栈：Chi + PostgreSQL + pgx + sqlc。

## 技术栈

| 层 | 技术 |
|---|------|
| HTTP Router | [chi/v5](https://github.com/go-chi/chi) |
| Database Driver | [pgx/v5](https://github.com/jackc/pgx) |
| SQL Code Gen | [sqlc](https://sqlc.dev/) |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) |
| Lint | [golangci-lint](https://golangci-lint.run/) |

## 快速开始

### 前置条件

- Go 1.26.3
- Docker / Docker Compose
- sqlc / migrate / golangci-lint / govulncheck（`make deps` 可安装固定版本）

项目工具版本统一维护在 `versions.env`。本地、CI、Docker 构建都应以该文件为准，避免使用本机已有工具或 `latest` 带来的版本漂移。

### 1. 初始化环境

```bash
cp .env.example .env
set -a && source .env && set +a
```

### 2. 启动数据库 & 迁移

```bash
make db-up
make migrate-up
```

### 3. 运行

```bash
make run
```

服务默认监听 `http://localhost:8080`。

### 4. 验证

```bash
# 健康检查
curl http://localhost:8080/healthz

# 创建笔记
curl -X POST http://localhost:8080/api/v1/notes \
  -H "Content-Type: application/json" \
  -d '{"title":"First Note","body":"Hello Nucleus"}'

# 查询笔记
curl "http://localhost:8080/api/v1/notes?limit=20&offset=0"
```

## 开发命令

```bash
make help           # 查看所有命令
make versions       # 查看固定工具版本
make version-check  # 检查本地和仓库声明的工具版本是否匹配
make build          # 构建二进制到 bin/
make run            # 本地运行
make test           # 运行测试
make fmt            # 格式化代码
make lint           # 使用固定版本 golangci-lint 静态检查
make check          # version-check + fmt + lint + test + vuln
make tidy           # 整理 go.mod 依赖
make clean          # 清理构建产物
make deps           # 安装固定版本开发工具
make db-up          # 启动 PostgreSQL
make db-down        # 停止所有容器
make migrate-up     # 执行迁移
make migrate-down   # 回滚一个迁移
make sqlc-gen       # 重新生成 sqlc 代码
```

## 项目结构

```
cmd/api/                  # 程序入口
internal/
  app/                    # 应用装配与生命周期
  config/                 # 配置加载
  log/                    # 结构化日志
  version/                # 构建版本信息
  db/                     # 数据库连接池
  db/sqlc/                # sqlc 生成的查询代码
  service/                # 业务逻辑层
  store/                  # 数据访问层
  http/handler/           # HTTP 处理器
  http/middleware/        # HTTP 中间件
  http/router/            # 路由定义
sql/
  migrations/             # 数据库迁移文件
  queries/                # SQL 查询定义
docs/                     # 项目文档
```

## 配置

通过环境变量配置，参见 `.env.example`：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `APP_ENV` | `local` | 运行环境 |
| `HTTP_PORT` | `8080` | 监听端口 |
| `CORS_ALLOWED_ORIGINS` | `*` | 允许的跨域来源，多个来源用逗号分隔 |
| `DATABASE_URL` | (必填) | PostgreSQL 连接串 |
| `DB_MAX_OPEN_CONNS` | `10` | 数据库连接池最大连接数 |
| `DB_MIN_CONNS` | `1` | 数据库连接池最小连接数 |
| `DB_MAX_IDLE_TIME` | `15m` | 数据库连接最大空闲时间 |
| `LOG_LEVEL` | `info` | 日志级别 |

## 工具版本

`versions.env` 是工具链版本的单一来源：

| 变量 | 用途 |
|------|------|
| `GO_VERSION` | 本地检查、CI setup-go、Docker build arg |
| `GOLANGCI_LINT_VERSION` | CI lint action、`make lint` |
| `SQLC_VERSION` | `make sqlc-gen`、`make deps` |
| `MIGRATE_VERSION` | `make migrate-up/down`、`make deps` |
| `GOVULNCHECK_VERSION` | `make vuln`、CI vulnerability check |
| `POSTGRES_VERSION` | 本地/CI PostgreSQL 版本约定 |

## 文档

项目推进方案详见 [docs/todo.md](docs/todo.md)，每阶段的讲解文档在 [docs/step/](docs/step/) 目录。
