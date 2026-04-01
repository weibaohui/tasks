# 项目 Claude Code 规范

## 项目概述

- **后端**：Go + DDD 架构
- **前端**：React + TypeScript + Ant Design
- **数据库**：SQLite

## 核心原则

1. **先复现再修复** - 不要猜测，先找证据
2. **小步提交** - 每次 commit 聚焦一个变更
3. **测试覆盖** - 核心逻辑必须有测试
4. **分层清晰** - domain → application → infrastructure → interfaces
5. **依赖倒置** - 高层模块不依赖低层模块，都依赖抽象
6. 必须严格按照 DDD.md中的要求编写代码

## 红线（禁止）

### 代码编写要求
1. AI 严禁在main分支直接写代码，必须先创建合适的分支
2. AI 严禁在没有文档的情况下写代码，必须先编写需求、设计、测试文档。

### DDD 原则
- ❌ `infrastructure` 引用 `interfaces`
- ❌ `domain` 引用其他层
- ❌ 应用服务包含业务逻辑（应该是贫血模型）
- ❌ 跨聚合直接修改（应该通过聚合根方法）
- ❌ 领域层引入技术细节（HTTP、DB 等）

### 代码质量
- ❌ `go vet` 有警告不修复
- ❌ 测试覆盖率低于 80%
- ❌ E2E 测试跳过（前端 UI 变更必须验证）

### Git
- ❌ 严禁在 main 分支直接提交代码，所有变更必须在新分支开展
- ❌ 混合多个无关变更到一个 commit
- ❌ commit message 含糊不清

## 开发流程

```
发现问题 → 复现确认 → 定位根因 → 修复代码 → 测试验证 → 提交代码
```


# 调试命令

 
## 常用命令

```bash
# 启动开发环境（后端 + 前端）
make dev

# 停止开发环境
make stop

# 后端编译和测试
cd backend && go build ./cmd/server && go test ./...

# 前端开发
# 使用pnpm 代替 npm
cd frontend && pnpm run dev

# E2E（必须会话隔离）
PW_SESSION="${PILOT_SESSION_ID:-default}"
playwright-cli -s="$PW_SESSION" open http://localhost:3000

# 数据库（位置在 ~/.taskmanager/config.yaml 中配置）
sqlite3 ~/.taskmanager/data.db
```

## 调试与日志

**日志文件（开发环境）：**
| 文件 | 内容 |
|------|------|
| `backend/logs/air.log` | Air 运行日志 + 应用所有 stdout/stderr |
| `backend/logs/air_build.log` | Air 构建日志 |

**查看日志：**
```bash
tail -f backend/logs/air.log          # 实时跟踪应用日志
tail -f backend/logs/air_build.log   # 实时跟踪构建日志
```

## 详细文档

| 文档 | 内容 |
|-------|------|
| `DDD.md` | DDD 架构约束与最佳实践 |
| `docs/development-workflow.md` | 开发流程、测试规范、问题排查 |
| `docs/*.md` | 其他专项文档 |

## 关键文件

| 文件 | 用途 |
|------|------|
| `backend/domain/hook.go` | Hook 系统定义 |
| `backend/infrastructure/hook/hooks/conversation_record.go` | 对话记录 Hook |
| `backend/pkg/channel/processor.go` | 消息处理器 |
| `frontend/src/pages/ConversationRecordsPage.tsx` | 对话记录页面 |
