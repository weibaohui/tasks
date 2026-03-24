# 测试规范

## 1. 测试金字塔

```
        E2E (少量)
       /          \
   Integration    Integration
   /      \        /      \
Unit     Unit    Unit     Unit
(大量)   (大量)  (大量)   (大量)
```

| 层级 | 数量 | 执行速度 | Mock |
|------|------|----------|------|
| Unit | 大量 | < 1ms | 所有外部依赖 |
| Integration | 中等 | < 100ms | 部分依赖 |
| E2E | 少量 | 秒级 | 无 |

## 2. 单元测试

### 2.1 要求
- **覆盖率**：核心业务逻辑 ≥ 80%
- **执行速度**：< 1ms 每测试
- **隔离性**：Mock 所有外部依赖

### 2.2 Mock 要求

| 类型 | 必须 Mock |
|------|-----------|
| HTTP 请求 | `httpx`, `requests` |
| 数据库 | 使用测试 fixtures（SQLite in-memory） |
| 文件 I/O | `open`, `Path.read_text` |
| 外部 API | 模拟客户端 |

### 2.3 命名规范

```go
// Go: Test<函数名>_<场景>_<预期>
func TestNewAgent_WithEmptyName_ReturnsError(t *testing.T) {...}
func TestCreateTask_WithValidInput_Success(t *testing.T) {...}

// TypeScript: it("should <behavior> when <condition>")
it("should return error when name is empty", () => {...})
```

## 3. 集成测试

### 3.1 要求
- 使用真实测试依赖
- 测试结束后清理数据
- 标记：`@pytest.mark.integration`

### 3.2 数据库测试

```go
// 使用 SQLite in-memory
db, err := sql.Open("sqlite3", ":memory:")
defer db.Close()

// 使用事务回滚
tx, _ := db.Begin()
defer tx.Rollback()
```

## 4. E2E 测试

### 4.1 适用场景
- 完整用户流程
- UI 组件交互
- API 端到端验证

### 4.2 工具
使用 `playwright-cli`

### 4.3 会话隔离（必须）

**并行测试时必须使用会话隔离**：
```bash
PW_SESSION="${PILOT_SESSION_ID:-default}"
playwright-cli -s="$PW_SESSION" open <url>
playwright-cli -s="$PW_SESSION" snapshot
playwright-cli -s="$PW_SESSION" close
```

### 4.4 工作流

```bash
# 1. 打开浏览器
playwright-cli -s="$PW_SESSION" open http://localhost:3000

# 2. 获取元素
playwright-cli -s="$PW_SESSION" snapshot

# 3. 交互操作
playwright-cli -s="$PW_SESSION" fill e1 "text"
playwright-cli -s="$PW_SESSION" click e2

# 4. 验证结果
playwright-cli -s="$PW_SESSION" snapshot

# 5. 关闭
playwright-cli -s="$PW_SESSION" close
```

## 5. TDD 流程

```
Red     →  写一个失败的测试
Green   →  写最简单的代码让测试通过
Refactor → 改进代码质量，保持测试通过
```

### 5.1 TDD 适用场景
- 新功能开发
- API 端点
- 业务逻辑
- Bug 修复（先写测试复现）

### 5.2 TDD 不适用场景
- 文档更新
- 配置变更
- 代码格式化

## 6. 测试文件位置

| 类型 | Go | TypeScript |
|------|-----|------------|
| 单元测试 | `*_test.go` 同包 | `*.test.ts` 同目录 |
| 集成测试 | `*_integration_test.go` | `*.integration.test.ts` |
| E2E | `e2e/` 目录 | `e2e/` 目录 |

## 7. 测试覆盖检查

```bash
# Go
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 查看覆盖率
go tool cover -func=coverage.out
```

## 8. 验证清单

### 8.1 代码变更
- [ ] 核心逻辑有单元测试
- [ ] 测试覆盖率 ≥ 80%
- [ ] `go test ./...` 通过
- [ ] `go vet ./...` 无警告

### 8.2 前端变更
- [ ] UI 流程用 E2E 验证
- [ ] 表单验证正常
- [ ] 成功/错误状态正确

### 8.3 Bug 修复
- [ ] 先写测试复现 bug
- [ ] 修复后测试通过
- [ ] 验证修复有效

## 9. 常见问题

### 9.1 测试太慢
- 检查是否 Mock 了所有外部依赖
- 检查是否有不必要的 sleep
- 检查数据库连接是否复用

### 9.2 测试不稳定
- 检查是否有竞态条件
- 检查是否正确清理测试数据
- 检查时间相关逻辑

### 9.3 测试无法复现
- 使用确定性的测试数据
- 避免依赖外部时间/随机数
