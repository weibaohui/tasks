# 开发工作流规范

## 1. 问题处理流程

### 1.1 发现问题
- **现象观察**：用户反馈或日志/监控发现异常
- **初步验证**：数据库查询、日志分析确认问题存在
- **问题记录**：描述问题现象、影响范围、复现步骤

### 1.2 根因定位
**追踪路径示例**：
```
用户反馈 → 数据库验证 → 代码追踪
```

**定位方法**：
1. **数据库验证**：确认数据状态
2. **日志追踪**：查看关键节点的日志输出
3. **代码追踪**：从入口到实现，逐步确认

### 1.3 修复方案
**原则**：
- 先复现再修复
- 找到根因再动手，不要猜测
- 小步修改，每次 commit 聚焦一个变更

### 1.4 测试验证
| 测试类型 | 用途 | 工具 |
|---------|------|------|
| 单元测试 | 核心逻辑验证 | `go test` |
| 集成测试 | 模块交互验证 | `go test -tags=integration` |
| E2E 测试 | 完整流程验证 | `playwright-cli` |

### 1.5 提交规范
**分支命名**：
```
feat/功能描述
fix/问题描述
test/测试改进
refactor/重构
```

**Commit 格式**：
```
<type>: <简短描述>

<详细说明（可选）>
```

**Type 类型**：
- `feat`: 新功能
- `fix`: 缺陷修复
- `test`: 测试相关
- `refactor`: 重构
- `docs`: 文档
- `chore`: 构建/工具

## 2. PR 工作流

### 2.1 分支管理
```bash
# 创建功能分支
git checkout -b feat/功能名称
git push -u origin feat/功能名称

# 创建修复分支
git checkout -b fix/问题描述
git push -u origin fix/问题描述
```

### 2.2 PR 流程
```
创建分支 → 提交代码 → Push → 创建 PR → Code Review → 合并 → 删除分支
```

**PR 描述模板**：
```markdown
## Summary
- <简要描述>

## Root Cause
- <问题根因>

## Changes
- <修改内容>

## Test Plan
- [ ] 测试项 1
- [ ] 测试项 2
```

## 3. 代码质量检查

### 3.1 Go 项目
```bash
# 格式化
gofmt -w .

# 静态分析
go vet ./...

# 运行测试
go test ./... -v

# 覆盖率
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 3.2 前端项目
```bash
# 安装依赖
bun install

# 运行测试
bun test

# 开发模式
bun run dev

# 构建
bun run build
```

### 3.3 E2E 测试
```bash
# 会话隔离（并行测试时必须）
PW_SESSION="${PILOT_SESSION_ID:-default}"

# 打开浏览器
playwright-cli -s="$PW_SESSION" open <url>

# 交互操作
playwright-cli -s="$PW_SESSION" fill <ref> <text>
playwright-cli -s="$PW_SESSION" click <ref>

# 验证
playwright-cli -s="$PW_SESSION" snapshot

# 关闭
playwright-cli -s="$PW_SESSION" close
```

## 4. 数据库问题排查

### 4.1 连接数据库
```bash
sqlite3 tasks.db
```

### 4.2 常用查询
```sql
-- 查看表结构
.schema table_name

-- 查看数据
SELECT * FROM table_name LIMIT 10;

-- 按条件查询
SELECT * FROM table_name WHERE field = 'value';

-- 统计
SELECT field, COUNT(*) FROM table_name GROUP BY field;

-- 查看空值
SELECT * FROM table_name WHERE user_code IS NULL;
```

## 5. 日志追踪

### 5.1 后端日志
- **位置**：`backend/backend.log`
- **级别**：DEBUG / INFO / WARN / ERROR
- **格式**：时间 + 级别 + 位置 + 消息 + 字段

### 5.2 关键日志点
| 模块 | 日志 | 用途 |
|------|------|------|
| Hook | `ConversationRecord: saved` | 记录保存成功 |
| LLM | `GenerateWithTools` | LLM 调用 |
| Tool | `tool_call` / `tool_result` | 工具执行 |

## 6. 测试规范

### 6.1 单元测试要求
- **覆盖率**：核心业务逻辑 ≥ 80%
- **隔离性**：不依赖外部资源，使用 mock
- **独立性**：每个测试可单独运行

### 6.2 TDD 流程
```
Red → Green → Refactor

1. Red：写一个失败的测试
2. Green：写最简单的代码让它通过
3. Refactor：改进代码质量，保持测试通过
```

### 6.3 Mock 要求
| 类型 | 必须 Mock |
|------|----------|
| HTTP 请求 | `httpx`, `requests` |
| 数据库 | SQLite (用测试 fixtures) |
| 文件 I/O | `open`, `Path.read_text` |
| 外部 API | 模拟客户端 |

## 7. 问题复盘

### 7.1 复盘内容
1. **问题描述**：什么出了问题
2. **根因分析**：为什么出问题
3. **修复方案**：如何修复的
4. **预防措施**：如何避免再发生
5. **经验总结**：学到了什么

### 7.2 复盘模板
```markdown
## 问题：[标题]

### 现象
<描述问题现象>

### 根因
<分析根因>

### 修复
<修复方案>

### 预防
<预防措施>

### 教训
<经验教训>
```

## 8. 项目结构

```
.
├── backend/              # Go 后端
│   ├── cmd/            # 入口
│   ├── domain/         # 领域模型
│   ├── application/    # 应用服务
│   ├── infrastructure/  # 基础设施
│   │   ├── persistence/  # 持久化
│   │   ├── hook/        # 钩子
│   │   └── llm/         # LLM
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

## 9. 常用命令速查

```bash
# 后端
cd backend
go build -o server ./cmd/server
go test ./... -v
go vet ./...

# 前端
cd frontend
bun run dev
bun run build

# Git
git checkout -b feat/xxx
git add .
git commit -m "feat: description"
git push -u origin HEAD
gh pr create --title "PR Title"
gh pr merge --admin --merge

# 数据库
sqlite3 tasks.db ".schema table_name"
```

## 10. 沟通协作

### 10.1 Commit Message 规范
- 使用中文描述（项目内）
- 简洁明了，不超过 50 字
- 说明 **为什么** 而非 **做了什么**

### 10.2 PR 描述要点
1. **Summary**：一句话描述
2. **Changes**：具体改了什么
3. **Test Plan**：如何验证
4. **Screenshots**：如有 UI 变更

### 10.3 遇到问题
1. 先自查：日志、数据、代码
2. 再讨论：带着证据问
3. 定方案：明确后再动手
