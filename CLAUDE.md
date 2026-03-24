# 项目 Claude Code 规范

## 项目概述

任务管理系统（tasks），包含：
- **后端**：Go 语言，基于 DDD 架构
- **前端**：React + TypeScript + Ant Design
- **数据库**：SQLite

## 开发规范

### 1. 问题处理流程

**发现问题 → 复现确认 → 定位根因 → 修复代码 → 测试验证 → 提交代码**

**复盘要点**：
- 先复现再修复，不要猜测
- 找到根因再动手
- 小步提交，每次 commit 聚焦一个变更

### 2. 代码质量

**Go 项目**：
```bash
go fmt ./...
go vet ./...
go test ./... -v
go test -coverprofile=coverage.out ./...
```

**前端项目**：
```bash
bun run dev      # 开发
bun run build    # 构建
bun test        # 测试
```

### 3. E2E 测试

使用 `playwright-cli`，**必须使用会话隔离**：
```bash
PW_SESSION="${PILOT_SESSION_ID:-default}"
playwright-cli -s="$PW_SESSION" open <url>
playwright-cli -s="$PW_SESSION" snapshot
playwright-cli -s="$PW_SESSION" close
```

### 4. Git 工作流

**分支命名**：
```
feat/功能描述
fix/问题描述
test/测试改进
```

**提交格式**：
```
<type>: <简短描述>

<详细说明（可选）>
```

**PR 流程**：
```bash
git checkout -b fix/问题描述
# 修改代码
git push -u origin HEAD
gh pr create --title "..."
gh pr merge --admin --merge
```

### 5. 数据库问题排查

```bash
sqlite3 tasks.db

# 查看表结构
.schema table_name

# 查看空值
SELECT * FROM table_name WHERE user_code IS NULL;
```

## 目录结构

```
.
├── backend/              # Go 后端
│   ├── cmd/            # 入口点
│   ├── domain/         # 领域模型
│   ├── application/    # 应用服务
│   ├── infrastructure/ # 基础设施
│   │   ├── persistence/  # 持久化
│   │   ├── hook/        # 钩子系统
│   │   └── llm/         # LLM 集成
│   └── interfaces/     # 接口层
│       └── http/        # HTTP 处理
│
├── frontend/            # React 前端
│   └── src/
│       ├── api/        # API 调用
│       ├── components/ # 组件
│       ├── pages/      # 页面
│       ├── stores/     # 状态管理
│       └── types/      # 类型定义
│
└── docs/               # 文档
```

## 关键文件

| 文件 | 用途 |
|------|------|
| `backend/domain/hook.go` | Hook 系统定义 |
| `backend/infrastructure/hook/hooks/conversation_record.go` | 对话记录 Hook |
| `backend/pkg/channel/processor.go` | 消息处理器 |
| `frontend/src/pages/ConversationRecordsPage.tsx` | 对话记录页面 |

## 测试规范

### 单元测试
- 核心业务逻辑 ≥ 80% 覆盖率
- Mock 所有外部依赖（HTTP、DB、文件 I/O）

### TDD 流程
```
Red  → 写失败的测试
Green → 写最简单的代码通过测试
Refactor → 改进代码质量
```

### E2E 验证
- 前端 UI 变更必须用 `playwright-cli` 验证
- 验证清单：
  - [ ] 主流程可完成
  - [ ] 表单验证正常
  - [ ] 成功状态正确显示
  - [ ] 导航正常
  - [ ] 错误状态正确渲染

## 沟通规范

### Commit Message
- 使用中文
- 简洁，不超过 50 字
- 说明**为什么**而非**做了什么**

### PR 描述
```markdown
## Summary
<一句话描述>

## Root Cause
<问题根因>

## Changes
<具体修改>

## Test Plan
- [ ] 测试项 1
- [ ] 测试项 2
```

## 决策原则

1. **先验证再修复** - 复现问题，确认根因
2. **小步提交** - 每次 commit 聚焦一个变更
3. **测试覆盖** - 核心逻辑必须有测试
4. **PR 审查** - 代码合并前必须 review
5. **文档更新** - 规范变更需同步更新文档

## 快速命令

```bash
# 后端开发
cd backend
go build -o server ./cmd/server
go test ./... -v

# 前端开发
cd frontend
bun run dev

# E2E 测试
playwright-cli open http://localhost:3000

# 数据库
sqlite3 backend/tasks.db
```

## 注意事项

1. **不要猜测** - 先查日志、数据、代码
2. **不要跳过测试** - 修复后必须运行测试
3. **不要忽略警告** - `go vet` 的警告要认真对待
4. **会话隔离** - 并行测试时必须使用 `-s=$PILOT_SESSION_ID`
