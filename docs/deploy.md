# 部署说明

## 构建镜像

```bash
make docker-build
```

产物：`nucleus-api:latest`（约 10MB，scratch 基础镜像）。

## 环境变量

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `APP_ENV` | 否 | `local` | 运行环境：local / test / production |
| `HTTP_PORT` | 否 | `8080` | 监听端口 |
| `CORS_ALLOWED_ORIGINS` | 否 | `*` | 允许的跨域来源，多个来源用逗号分隔 |
| `DATABASE_URL` | **是** | — | PostgreSQL 连接串 |
| `DB_MAX_OPEN_CONNS` | 否 | `10` | 最大打开连接数 |
| `DB_MIN_CONNS` | 否 | `1` | 最小连接数 |
| `DB_MAX_IDLE_TIME` | 否 | `15m` | 最大空闲时间 |
| `LOG_LEVEL` | 否 | `info` | 日志级别：debug / info / warn / error |

## Docker 运行

```bash
docker run -d \
  --name nucleus-api \
  -p 8080:8080 \
  -e APP_ENV=production \
  -e CORS_ALLOWED_ORIGINS=https://app.example.com \
  -e DATABASE_URL=postgres://user:pass@db:5432/nucleus?sslmode=disable \
  nucleus-api:latest
```

## Docker Compose 运行

```bash
# 启动数据库
make db-up

# 执行迁移
make migrate-up

# 启动服务（本地 Go 或 Docker）
make run
```

## 健康检查

```bash
# 存活探针（不检查依赖）
curl http://localhost:8080/healthz

# 就绪探针（检查数据库连接）
curl http://localhost:8080/readyz
```

## 数据库迁移

```bash
# 升级
make migrate-up

# 回滚一个版本
make migrate-down
```

## 质量门禁

```bash
make check   # fmt + lint + test + vuln
make cover   # 测试覆盖率报告
```
