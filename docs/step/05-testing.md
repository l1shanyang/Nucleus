# 阶段五：测试与文档化

> 目标：让脚手架具备长期维护和多人协作基础。

## 一、本次改动总览

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/store/storetest/mock_note.go` | 新增 | NoteStore 的内存 mock 实现 |
| `internal/service/note_test.go` | 新增 | service 层单元测试（table-driven） |
| `internal/http/handler/notes_test.go` | 新增 | handler 层 HTTP 测试（httptest） |
| `Makefile` | 改写 | 增加 `make cover` 覆盖率报告 |
| `docs/api/openapi.yaml` | 新增 | OpenAPI 3.0 接口文档 |

## 二、Go 测试基础

### 2.1 测试文件命名

Go 的测试规则很简单：
- 文件名以 `_test.go` 结尾
- 函数名以 `Test` 开头，参数是 `*testing.T`
- 测试文件和被测文件放在同一个包（或 `_test` 包）

```go
// internal/service/note_test.go
package service_test  // _test 包，只能访问公开接口

func TestNoteService_Create(t *testing.T) {
    // ...
}
```

用 `_test` 包的好处：测试只能通过公开 API 调用，和真实使用者一样，不会误用内部实现。

### 2.2 Table-Driven Test — Go 的标准测试模式

```go
func TestNoteService_Create(t *testing.T) {
    tests := []struct {
        name    string          // 用例名称
        input   CreateInput     // 输入
        wantErr string          // 期望的错误信息（空=成功）
    }{
        {"正常创建", CreateInput{Title: "T", Body: "B"}, ""},
        {"title 为空", CreateInput{Title: "", Body: "B"}, "title is required"},
        // ...更多用例
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 执行 + 断言
        })
    }
}
```

**为什么用 table-driven？**
- 新增用例只需加一行，不用复制整个函数
- 每个用例有名字，失败时能精确定位
- 结构统一，可读性强

### 2.3 Mock — 替换外部依赖

测试 service 时不想连真实数据库。解决方法：创建一个假的 store 实现。

```go
// MockNoteStore 实现 NoteStore 接口，但用内存存储
type MockNoteStore struct {
    notes    []store.Note
    CreateErr error  // 可注入错误
}

func (m *MockNoteStore) Create(ctx, title, body) (Note, error) {
    if m.CreateErr != nil {
        return Note{}, m.CreateErr  // 模拟数据库报错
    }
    // 内存操作...
}
```

**关键设计：** mock 实现了和真实 store 相同的接口，service 完全不知道自己在用 mock。这就是接口的价值——依赖抽象而非具体实现。

### 2.4 httptest — 不启动服务器测试 HTTP

```go
// 构造请求
req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", body)
req.Header.Set("Content-Type", "application/json")

// 构造响应记录器
w := httptest.NewRecorder()

// 直接调用 handler
handler.WrapHandler(h.Create)(w, req)

// 检查响应
if w.Code != http.StatusCreated {
    t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
}
```

**好处：**
- 不需要启动真实服务器
- 不占用端口
- 测试速度快（毫秒级）
- 可以精确检查状态码、响应头、响应体

### 2.5 覆盖率

```bash
make cover
```

输出：

```
nucleus/internal/service      coverage: 100.0%
nucleus/internal/http/handler coverage: 68.3%
total:                        23.7%
```

- 100% 不是目标，但关键业务逻辑（service）应该尽量高
- handler 的 68.3% 来自于成功和错误路径都覆盖了
- 基础设施包（config、db、app）的低覆盖率是正常的，它们需要集成测试

生成 HTML 报告查看具体哪些行没覆盖：

```bash
go tool cover -html=coverage.out
```

## 三、OpenAPI 文档

`docs/api/openapi.yaml` 定义了当前所有接口的规范：

- 请求/响应格式
- 参数类型和约束
- 错误码
- 数据模型

可以用 [Swagger UI](https://editor.swagger.io/) 在线查看，也可以用 Redoc 生成静态文档。

**维护原则：** 每次新增或修改接口，同步更新 openapi.yaml。

## 四、本阶段学到了什么

| 知识点 | 说明 |
|--------|------|
| Table-driven test | Go 的标准测试模式，结构化、易扩展 |
| Mock | 用接口实现替换外部依赖，隔离测试目标 |
| httptest | 不启动服务器测试 HTTP handler |
| _test 包 | 测试只能访问公开 API，模拟真实使用场景 |
| 覆盖率 | `go test -coverprofile` 生成覆盖率数据 |
| OpenAPI | 接口的标准化描述格式 |

## 五、下一步

进入 [阶段六：生产化能力](06-production.md)，增加 Dockerfile、CI、structured logging。
