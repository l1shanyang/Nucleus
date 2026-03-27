# Nucleus Backend (Go)

一个用于长期演进的后端原子项目骨架，技术栈：

- Go
- Chi
- PostgreSQL
- pgx/v5
- sqlc
- golang-migrate

## 1. 本地准备

### 必需工具

- Go (建议 1.23+)
- Docker / Docker Compose
- sqlc
- migrate（可选，未安装会自动使用 Docker 版本）

macOS 可用：

```bash
brew install go sqlc golang-migrate
```

或者使用 `make deps`（依赖 Go 已安装）：

```bash
make deps
```

## 2. 初始化环境变量

```bash
cp .env.example .env
```

然后导入环境变量（zsh/bash）：

```bash
set -a
source .env
set +a
```

## 3. 启动数据库并迁移

```bash
make db-up
make migrate-up
```

`make migrate-up` 会自动选择：

- 本机安装了 `migrate`：走本机命令
- 本机没安装 `migrate`：自动走 `docker compose run --rm migrate ...`

## 4. 生成查询代码（可选）

`internal/db/sqlc` 目录已经包含一份最小可运行实现。后续可用 sqlc 重新生成：

```bash
make sqlc-gen
```

## 5. 启动服务

```bash
make run
```

默认监听：`http://localhost:8080`

`make run` 会自动选择：

- 本机安装了 Go：走 `go run ./cmd/api`
- 本机没安装 Go：自动走 `docker compose up api`

## 6. 快速验证

健康检查：

```bash
curl http://localhost:8080/healthz
```

创建笔记：

```bash
curl -X POST http://localhost:8080/api/v1/notes \
  -H "Content-Type: application/json" \
  -d '{"title":"First Note","body":"Hello Nucleus"}'
```

查询笔记：

```bash
curl "http://localhost:8080/api/v1/notes?limit=20&offset=0"
```

## 目录说明

```text
cmd/api                 # 程序入口
internal/config         # 配置加载
internal/db             # 数据库连接
internal/db/sqlc        # 查询层（sqlc 生成或兼容实现）
internal/http/handler   # HTTP handler
internal/http/router    # 路由
sql/migrations          # 数据库迁移
sql/queries             # SQL 查询定义
```

## 后续建议

下一步推荐按这个顺序扩展：

1. 增加用户表与基础认证
2. 引入 structured logging 与 request tracing
3. 加入单元测试与 API 集成测试
4. 设计 service 层与事务边界
