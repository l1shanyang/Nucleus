# 阶段一：工程化基础

> 目标：让项目像一个标准开源后端项目，而不是临时 demo。

本阶段不涉及业务逻辑变更，全部是"让开发体验更好、项目更规范"的基础工作。

## 一、本次改动总览

| 文件 | 操作 | 说明 |
|------|------|------|
| `.gitignore` | 改写 | 从基础版升级为完整的 Go 项目 ignore 规则 |
| `.dockerignore` | 新增 | 控制 Docker build context，避免无关文件进入镜像 |
| `.editorconfig` | 新增 | 统一不同编辑器/IDE 的代码风格 |
| `Makefile` | 改写 | 增加 build、check、tidy、clean 等命令，支持构建信息注入 |
| `.golangci.yml` | 新增 | golangci-lint 配置，启用常用 linter |
| `versions.env` | 新增 | 固定 Go 与开发工具版本，避免本地/CI 漂移 |
| `internal/version/version.go` | 新增 | 构建版本信息包 |
| `README.md` | 改写 | 更规范的项目文档 |

## 二、逐项讲解

### 2.1 .gitignore — 不该进仓库的东西

**为什么需要？**

Git 仓库是项目源码的"真相"，不属于源码的东西不应该出现在里面：
- 编译产物（`bin/`、`*.exe`）—— 每台机器编译出来的不同
- 依赖缓存（`vendor/`、`.cache/`）—— 可以通过 `go mod` 重建
- IDE 配置（`.idea/`、`.vscode/`）—— 每个人的 IDE 设置不同
- 环境变量（`.env`）—— 包含数据库密码等敏感信息
- 操作系统文件（`.DS_Store`）—— macOS 自动生成的元数据

**关键规则解读：**

```gitignore
# 编译产物
bin/
*.exe
*.dll
*.so
*.dylib

# 测试覆盖率
*.out
coverage.html

# IDE 配置 — 每人不同，不共享
.idea/
.vscode/

# 环境变量 — 含密码，绝不能提交
.env
.env.local
```

**`.env.example` 应该提交**，它告诉协作者需要哪些环境变量，但不含真实值。

### 2.2 .dockerignore — Docker 构建的"过滤器"

**为什么需要？**

当你运行 `docker build` 或 `docker compose` 时，Docker 会把当前目录打包发送给 daemon。如果目录里有 `.git`、`node_modules`、大量缓存文件，构建会很慢且镜像臃肿。

`.dockerignore` 的语法和 `.gitignore` 类似，告诉 Docker "这些文件不要发给我"。

```dockerignore
.git          # 历史记录，可能有几百 MB
docs/         # 文档不需要进入运行时镜像
*.test        # 测试二进制
```

### 2.3 .editorconfig — 统一代码风格

**为什么需要？**

团队里有人用 VS Code，有人用 GoLand，有人用 Vim。如果没有统一配置：
- 缩进可能混用 tab 和 space
- 文件末尾可能有多余空行
- 换行符可能混用 LF 和 CRLF

`.editorconfig` 是一个跨编辑器的标准，主流编辑器都原生支持或通过插件支持。

```ini
# Go 文件用 tab 缩进（Go 语言规范要求）
[*.go]
indent_style = tab

# YAML 用 2 空格
[*.{yml,yaml}]
indent_size = 2

# Makefile 必须用 tab（语法要求）
[Makefile]
indent_style = tab

# Markdown 保留尾部空格（Markdown 中两个空格 = 换行）
[*.md]
trim_trailing_whitespace = false
```

### 2.4 Makefile — 项目的"操作手册"

**为什么需要？**

Makefile 是后端项目的"命令中心"。新人来了，敲 `make help` 就知道怎么操作。比翻文档、记命令高效得多。

**本次增强点：**

#### (1) 构建信息注入

```makefile
VERSION    := $(shell git describe --tags --always --dirty)
COMMIT     := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS    := -s -w \
    -X nucleus/internal/version.Version=$(VERSION) \
    -X nucleus/internal/version.Commit=$(COMMIT) \
    -X nucleus/internal/version.BuildTime=$(BUILD_TIME)
```

`-ldflags` 是 Go 编译器的一个参数，可以在编译时把值注入到变量里。

`-s -w` 去掉调试信息和 DWARF 符号表，让二进制更小。

这样 `go build` 出来的二进制会自带版本号，不需要额外配置文件。

#### (2) build 命令

```makefile
build:
    @mkdir -p bin
    go build -ldflags '$(LDFLAGS)' -o bin/$(APP_NAME) ./cmd/api
```

`@mkdir -p bin` — `@` 表示不打印命令本身，`-p` 表示目录存在也不报错。

`-o bin/nucleus-api` — 输出到 `bin/` 目录，这是 Go 项目的惯例。

#### (3) check 命令

```makefile
check: version-check fmt lint test vuln
```

一条命令跑完版本检查、格式化、静态检查、测试和漏洞扫描。可以作为 CI 的入口命令，也可以在提交前手动跑一下确认没问题。

#### (4) tidy 命令

```makefile
tidy:
    go mod tidy
    go mod verify
```

`go mod tidy` 会删除 `go.mod` 里不需要的依赖，添加代码里用到但 `go.mod` 里没有的依赖。

`go mod verify` 检查本地缓存的模块是否被篡改。

#### (5) help 命令

```makefile
help:
    @grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
        awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
```

自动从 Makefile 中提取所有带 `## 注释` 的 target，格式化输出。好处是：加新命令时只要写上 `## 说明`，help 就会自动更新。

### 2.5 versions.env — 工具版本清单

**为什么需要？**

CI 和本地最容易出问题的地方就是工具版本不同。例如本地安装了 `golangci-lint 2.x`，但 CI 使用 `1.x`，同一份配置就可能在一个环境通过、另一个环境失败。

`versions.env` 固定这些版本：

```env
GO_VERSION=1.26.3
GOLANGCI_LINT_VERSION=v2.12.2
SQLC_VERSION=v1.31.1
MIGRATE_VERSION=v4.19.1
GOVULNCHECK_VERSION=v1.3.0
POSTGRES_VERSION=17
```

Makefile 和 CI 都读取这个文件。本地执行 `make lint`、`make sqlc-gen`、`make migrate-up` 时，不再使用本机随机安装的版本，而是使用仓库声明的固定版本。

### 2.6 .golangci.yml — 静态分析配置

**为什么需要？**

`go vet` 只能检查一些基本问题。golangci-lint 是 Go 社区最流行的 linter 聚合工具，内置了几十个 linter，可以发现更多问题。

**启用的 linter 说明：**

| Linter | 作用 |
|--------|------|
| `errcheck` | 检查是否忽略了 error 返回值 |
| `govet` | go vet 的超集，检查可疑代码 |
| `staticcheck` | 高级静态分析，发现性能和正确性问题 |
| `unused` | 检查未使用的变量、函数、参数 |
| `ineffassign` | 检查无效赋值（赋值后从未使用） |
| `gosimple` | 建议更简洁的写法 |
| `gocritic` | 代码风格和最佳实践检查 |
| `gofmt` | 检查是否格式化 |
| `goimports` | 检查 import 排序 |
| `misspell` | 检查英文拼写错误 |

**注意事项：**

```yaml
issues:
  exclude-dirs:
    - vendor
    - internal/db/sqlc   # sqlc 生成的代码不做 lint
```

`internal/db/sqlc` 是 sqlc 自动生成的，格式和风格由 sqlc 控制，人工 lint 没有意义。

### 2.7 internal/version — 构建版本信息

```go
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildTime = "unknown"
)
```

这些变量有默认值，开发时 `go run` 得到的是 `dev`。

通过 Makefile 的 `-ldflags` 注入后，编译出的二进制会携带真实版本信息。

后续可以在 `/healthz` 或新增 `/version` 接口暴露这些信息，方便运维确认部署版本。

## 三、验证

完成以上改动后，可以执行以下命令验证：

```bash
# 查看所有可用命令
make help

# 确认格式化和检查通过
make fmt
make lint

# 构建二进制
make build

# 查看版本信息
./bin/nucleus-api --version  # (需要后续阶段添加 flag 解析)
```

## 四、本阶段学到了什么

| 知识点 | 说明 |
|--------|------|
| `.gitignore` | 控制哪些文件不进入版本管理 |
| `.dockerignore` | 控制哪些文件不进入 Docker 构建上下文 |
| `.editorconfig` | 跨编辑器统一代码风格 |
| Makefile | 项目的命令入口，新人看 help 就能上手 |
| `versions.env` | 固定本地、CI、Docker 构建使用的工具版本 |
| `-ldflags` | Go 编译时注入变量的机制 |
| `go mod tidy` | 保持依赖声明干净 |
| golangci-lint | Go 静态分析工具，比 `go vet` 更强大 |
| 构建信息 | 版本号、commit、构建时间，运维排查必备 |

## 五、下一步

进入 [阶段二：应用启动与配置体系](02-startup-config.md)，将项目从"能跑"升级成"可管理、可配置、可部署"。
