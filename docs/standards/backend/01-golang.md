# Go 语言编码规范

## 1. 格式化

```bash
# 格式化代码
go fmt ./...

# 格式化并重写
gofmt -w .
```

## 2. 命名规范

### 2.1 包名
- **小写**，单个单词
- 简短，避免下划线
- `domain`, `application`, `infrastructure`

### 2.2 变量/函数名
- **驼峰命名**
- 布尔变量：`is`, `has`, `can`, `should` 开头
- 示例：`isActive`, `hasPermission`

### 2.3 常量
- 全大写 + 下划线分隔
- 示例：`MaxRetryCount`, `DefaultTimeout`

### 2.4 接口
- `-er` 后缀
- 示例：`Reader`, `Writer`, `Handler`

### 2.5 错误类型
- `Err` 开头
- 示例：`ErrUserNotFound`, `ErrInvalidInput`

## 3. 错误处理

### 3.1 错误创建
```go
// 使用 errors.New
var ErrNotFound = errors.New("not found")

// 使用 fmt.Errorf
return fmt.Errorf("create user: %w", err)
```

### 3.2 错误检查
```go
// ✅ 正确
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// ❌ 错误：忽略错误
if err != nil {
    // do nothing
}
```

### 3.3 哨兵错误
```go
var ErrUserNotFound = errors.New("user not found")
var ErrInvalidToken = errors.New("invalid token")
```

## 4. Context

### 4.1 Context 传递
```go
// ✅ 正确：context 作为第一个参数
func DoSomething(ctx context.Context, arg string) error

// ❌ 错误
func DoSomething(arg string, ctx context.Context) error
```

### 4.2 Context 取消
```go
select {
case <-ctx.Done():
    return ctx.Err()
case result := <-ch:
    return result
}
```

## 5. 并发

### 5.1 Goroutine
```go
// ✅ 正确：知道何时终止
go func() {
    // work
}()

// ❌ 错误：泄漏
go func() {
    for {
        // work
    }
}()
```

### 5.2 Channel
```go
// 创建缓冲 channel
ch := make(chan int, 10)

// 创建无缓冲 channel
ch := make(chan int)
```

### 5.3 Mutex
```go
var mu sync.Mutex

mu.Lock()
defer mu.Unlock()
```

## 6. 结构体

### 6.1 字段标签
```go
type User struct {
    Name string `json:"name"`
    Age  int    `json:"age,omitempty"`
}
```

### 6.2 构造函数
```go
func NewUser(name string, age int) *User {
    return &User{
        Name: name,
        Age:  age,
    }
}
```

## 7. 接口设计

### 7.1 接口越小越好
```go
// ✅ 正确：小接口
type Reader interface {
    Read(p []byte) (n int, err error)
}

// ❌ 错误：大而全
type CRUD interface {
    Create()
    Read()
    Update()
    Delete()
    List()
}
```

### 7.2 接口位置
- 接口定义在**使用者**那侧
- 实现放在被使用的那侧

## 8. 依赖管理

```bash
# 添加依赖
go get github.com/pkg/errors

# 整理依赖
go mod tidy

# 查看依赖
go list -m all
```

## 9. 测试

### 9.1 表格驱动测试
```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name string
        a, b int
        want int
    }{
        {"2+3=5", 2, 3, 5},
        {"0+0=0", 0, 0, 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Add(tt.a, tt.b)
            if got != tt.want {
                t.Errorf("Add() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 9.2 Mock
```go
// 使用 interface
type UserRepository interface {
    FindByID(ctx context.Context, id string) (*User, error)
}

// 测试时注入 mock
type mockRepo struct{}
func (m *mockRepo) FindByID(ctx context.Context, id string) (*User, error) {
    return &User{ID: id, Name: "Test"}, nil
}
```

## 10. Protobuf/gRPC（可选）

如果使用 gRPC：
```protobuf
service UserService {
    rpc GetUser(GetUserRequest) returns (User);
}

message GetUserRequest {
    string user_id = 1;
}
```
