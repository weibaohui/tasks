# Git 工作流规范

## 1. 分支管理

### 1.1 分支命名

```
feat/功能描述          # 新功能
fix/问题描述           # 缺陷修复
test/测试改进          # 测试相关
refactor/重构描述      # 重构
docs/文档描述          # 文档
```

**示例**：
```bash
feat/user-authentication
fix/conversation-record-empty
test/add-agent-coverage
```

### 1.2 分支操作

```bash
# 创建分支
git checkout -b feat/功能名称

# 切换分支
git checkout main
git pull origin main

# 删除分支
git branch -d feat/功能名称
```

## 2. Commit 规范

### 2.1 格式

```
<type>: <简短描述>

<详细说明（可选）>
```

### 2.2 Type 类型

| Type | 说明 | 示例 |
|------|------|------|
| feat | 新功能 | `feat: 添加用户登录功能` |
| fix | 缺陷修复 | `fix: 修复对话记录为空问题` |
| test | 测试相关 | `test: 添加用户注册测试` |
| refactor | 重构 | `refactor: 简化 Agent 工厂方法` |
| docs | 文档 | `docs: 更新 API 文档` |
| style | 格式 | `style: 格式化代码` |
| chore | 构建/工具 | `chore: 升级依赖版本` |

### 2.3 规则

- **使用中文**描述
- **简洁**，不超过 50 字
- 说明**为什么**而非**做了什么**
- 第一行是摘要，后面可加详细说明

**示例**：
```bash
# 好
git commit -m "fix: 修复对话记录中user_code为空的问题

问题原因：PreLLMCall中extractScope在Metadata设置之前调用
修复：调整extractScope调用顺序"

# 不好
git commit -m "update code"
git commit -m "fix bug"
```

## 3. PR 规范

### 3.1 创建 PR

```bash
# 推送分支
git push -u origin HEAD

# 创建 PR
gh pr create --title "fix: 修复对话记录为空问题"
```

### 3.2 PR 描述模板

```markdown
## Summary
一句话描述修改内容

## Root Cause
问题根因分析

## Changes
- 修改点 1
- 修改点 2

## Test Plan
- [ ] 测试项 1
- [ ] 测试项 2
```

### 3.3 PR 合并

```bash
# 确保分支是最新的
git checkout main
git pull origin main

# 合并
gh pr merge --admin --merge

# 删除分支
git branch -d feat/功能名称
```

## 4. 工作流程

### 4.1 Feature 工作流

```bash
# 1. 从 main 创建分支
git checkout -b feat/功能名称

# 2. 开发并提交
git add .
git commit -m "feat: 添加功能"

# 3. 推送
git push -u origin HEAD

# 4. 创建 PR
gh pr create --title "feat: 添加XXX功能"

# 5. Code Review
# ...

# 6. 合并
gh pr merge --admin --merge

# 7. 更新本地
git checkout main
git pull origin main
```

### 4.2 Hotfix 工作流

```bash
# 1. 从 main 创建分支
git checkout -b fix/紧急问题

# 2. 快速修复并提交
git commit -m "fix: 紧急修复XXX问题"

# 3. 推送并创建 PR
git push -u origin HEAD
gh pr create --title "fix: 紧急修复XXX问题" --label "hotfix"

# 4. 优先 Review 和合并
```

## 5. 规则

### 5.1 必须遵守
- ✅ commit 前运行 `go fmt` / `go vet`
- ✅ 每个 commit 聚焦一个变更
- ✅ PR 描述清晰，包含测试计划
- ✅ 合并前确保测试通过

### 5.2 禁止
- ❌ `git push -f`（除非明确需要）
- ❌ 混合多个无关变更
- ❌ commit message 含糊不清
- ❌ 直接提交到 main 分支

## 6. 常见问题

### 6.1 修改上一个 commit
```bash
git commit --amend
# 修改 message 或添加文件
```

### 6.2 撤销未推送的 commit
```bash
git reset --soft HEAD~1   # 保留更改
git reset --hard HEAD~1   # 删除更改
```

### 6.3 查看提交历史
```bash
git log --oneline -10     # 最近 10 条
git log --graph --oneline  # 图形化
```
