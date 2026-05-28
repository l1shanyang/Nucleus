# 阶段四：架构分层与数据库基础

> 目标：从简单 CRUD 骨架升级成可扩展后端架构。

## 一、本次改动总览

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/store/note.go` | 新增 | Store 层：封装数据访问，隔离 sqlc |
| `internal/service/note.go` | 新增 | Service 层：承载业务逻辑 |
| `internal/http/handler/notes.go` | 改写 | Handler 瘦身，只做 HTTP 转换 |
| `internal/db/pool.go` | 改写 | 增加 BeginTx 事务支持 |
| `internal/app/app.go` | 改写 | 装配链：sqlc → store → service → handler |

## 二、架构分层

### 改之前的调用链

```
handler → sqlc (直接依赖数据库生成的类型)
```

handler 里混杂着 HTTP 协议处理和数据访问逻辑。如果要换数据库访问方式（比如从 sqlc 换成 GORM），handler 也得改。

### 改之后的调用链

```
handler → service → store → sqlc
  ↓          ↓         ↓        ↓
HTTP转换   业务逻辑   数据访问   SQL生成
```

每一层只依赖下一层的接口，不跨层调用。

### 各层职责

| 层 | 职责 | 不应该做什么 |
|----|------|-------------|
| handler | 解析 HTTP 请求、写响应、调 service | 不包含业务规则、不直接访问数据库 |
| service | 业务校验、协调多个 store 调用、事务管理 | 不知道 HTTP 的存在、不直接写 SQL |
| store | 封装数据库操作、转换 sqlc 模型为领域模型 | 不包含业务逻辑 |
| sqlc | 生成类型安全的 SQL 代码 | 不修改，由工具自动生成 |

## 三、逐项讲解

### 3.1 Store 层 — 数据访问的隔离层

**目的：** 让 handler 不直接依赖 sqlc 生成的类型。

```go
// store 层接口 — 参数是普通 Go 类型
type NoteStore interface {
    Create(ctx context.Context, title, body string) (Note, error)
    List(ctx context.Context, limit, offset int32) ([]Note, error)
    WithTx(tx pgx.Tx) NoteStore  // 事务支持
}
```

对比 sqlc 生成的接口：

```go
// sqlc 接口 — 参数是 sqlc 生成的 struct
CreateNote(ctx, sqlc.CreateNoteParams) (sqlc.Note, error)
ListNotes(ctx, sqlc.ListNotesParams) ([]sqlc.Note, error)
```

**store 的优势：**
- 接口参数更自然（直接传 title, body，不用构造 Params struct）
- 领域模型 Note 与数据库 schema 解耦（数据库加字段不影响 store 接口）
- 实现可替换（可以换成 GORM、原生 SQL 等）

**WithTx 模式：**

```go
func (s *noteStore) WithTx(tx pgx.Tx) NoteStore {
    return &noteStore{q: s.q.WithTx(tx)}
}
```

返回一个新的 store 实例，内部使用事务。原 store 不受影响。这是 sqlc 推荐的事务用法。

### 3.2 Service 层 — 业务逻辑的家

**目的：** 把业务规则从 handler 中抽出来。

```go
type NoteService struct {
    store store.NoteStore
}

func (s *NoteService) Create(ctx context.Context, input CreateInput) (store.Note, error) {
    // 业务规则：清理、校验
    input.Title = strings.TrimSpace(input.Title)
    if input.Title == "" {
        return store.Note{}, fmt.Errorf("title is required")
    }
    if len(input.Title) > 200 {
        return store.Note{}, fmt.Errorf("title must be at most 200 characters")
    }
    // 调 store
    return s.store.Create(ctx, input.Title, input.Body)
}
```

**什么时候需要 service 层？**
- 一个操作涉及多个 store 调用（如：创建笔记 + 写日志）
- 有业务规则需要执行（如：标题长度限制、去重检查）
- 需要事务管理（多步操作的原子性）

如果只是简单 CRUD，service 层会很薄。但随着业务增长，它的价值会体现出来。

### 3.3 Handler 瘦身 — 只做 HTTP 转换

**改之前：** handler 里有校验逻辑、参数规范化、直接调 sqlc。

**改之后：** handler 只做三件事：

```go
func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) error {
    // 1. 解析请求
    var req createNoteRequest
    DecodeJSON(r, &req)

    // 2. 调 service
    note, err := h.svc.Create(r.Context(), service.CreateInput{...})

    // 3. 写响应
    WriteSuccess(w, http.StatusCreated, note)
}
```

**好处：** handler 变成一个"薄壳"，容易测试，容易理解。

### 3.4 事务管理

事务保证一组操作要么全部成功，要么全部回滚。

```go
// 在 service 中使用事务
func (s *SomeService) DoSomething(ctx context.Context) error {
    tx, err := s.pool.BeginTx(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx) // 如果 Commit 了，Rollback 是 no-op

    txNoteStore := s.noteStore.WithTx(tx)
    txOtherStore := s.otherStore.WithTx(tx)

    txNoteStore.Create(ctx, ...)
    txOtherStore.Create(ctx, ...)

    return tx.Commit(ctx)
}
```

**关键点：**
- `defer tx.Rollback(ctx)` 必须在 `BeginTx` 之后立即写
- 如果 `Commit` 成功，`Rollback` 会自动跳过（pgx 的行为）
- 所有 store 都要用 `WithTx(tx)` 创建的事务版本

### 3.5 装配链 — app.go 的变化

```go
// 依赖从底向上创建
queries := sqlc.New(pool.DB())              // SQL 层
noteStore := store.NewNoteStore(queries)     // 数据访问层
noteSvc := service.NewNoteService(noteStore) // 业务逻辑层
noteHandler := handler.NewNoteHandler(noteSvc) // HTTP 层
```

每一层只接收它直接依赖的下一层。app.go 是唯一知道所有层的地方。

## 四、SQL 与 Migration 规范

### Migration 命名

```
sql/migrations/
  000001_init.up.sql      # 序号_描述.up.sql
  000001_init.down.sql    # 序号_描述.down.sql
  000002_add_users.up.sql
  000002_add_users.down.sql
```

- 序号递增，保证执行顺序
- 每个 migration 必须有对应的 up 和 down
- down 必须能完全撤销 up 的改动

### SQL 查询组织

```
sql/queries/
  notes.sql     # 一个表的查询放在一个文件里
  users.sql     # 以后加新表就加新文件
```

每个查询用 sqlc 注解标注返回类型：
- `:one` — 返回单行
- `:many` — 返回多行
- `:exec` — 不返回数据（INSERT/UPDATE/DELETE 不需要返回值时）
- `:execlastid` — 返回最后插入的 ID

## 五、本阶段学到了什么

| 知识点 | 说明 |
|--------|------|
| 分层架构 | handler → service → store → sqlc，各司其职 |
| 依赖方向 | 上层依赖下层接口，不跨层调用 |
| Store 模式 | 封装数据访问，对外暴露领域模型 |
| Service 模式 | 承载业务逻辑，协调多个 store |
| 事务管理 | BeginTx + WithTx + Commit/Rollback |
| 装配点 | app.go 是唯一创建所有层的地方 |

## 六、下一步

进入 [阶段五：测试与文档化](05-testing.md)，为各层编写测试。
