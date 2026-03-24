# 后端规范

Go 技术栈相关规范。

## 规范列表

| 文件 | 说明 |
|------|------|
| `01-golang.md` | Go 语言编码规范 |
| `02-ddd.md` | DDD 架构约束 |
| `03-api.md` | HTTP API 设计规范 |

## 快速命令

```bash
# 格式化
go fmt ./...

# 静态分析
go vet ./...

# 测试
go test ./... -v

# 覆盖率
go test -coverprofile=coverage.out ./...
```
