# Agent 管理界面重构测试文档

## 测试范围

本次重构主要涉及前端 UI 交互和菜单结构，后端无变更。测试重点为：
1. 左侧菜单正确移除 Skills、MCP 入口
2. Agent 列表按类型分组过滤正确
3. 新建 Agent 两步流程交互正常
4. 个人助理编辑页内嵌资源入口可用
5. 现有 Agent 编辑功能不受影响

## 单元测试

### UT-1: 列表过滤逻辑
**目标**: `AgentManagementPage` 的分组过滤逻辑
**输入**: 混合类型的 Agent 数组
**预期**:
- `activeGroup='all'` → 返回全部 Agent
- `activeGroup='assistant'` → 仅返回 `agent_type='BareLLM'`
- `activeGroup='coding'` → 返回 `CodingAgent` + `OpenCodeAgent`

### UT-2: AgentTypeSelector 渲染
**目标**: `AgentTypeSelector` 组件
**输入**: 弹窗 visible=true
**预期**:
- 渲染 3 个类型卡片（个人助理、Claude Code、OpenCode）
- 每个卡片显示标题和描述
- 初始无选中态

### UT-3: 编辑抽屉 agent_type 禁用
**目标**: `AgentEditDrawer` 新建模式行为
**输入**: `mode='create'`, `agentType='CodingAgent'`
**预期**:
- `agent_type` Select 禁用或隐藏
- active tab 为 `claudecode`

## 集成测试 / E2E 测试

### E2E-1: 菜单结构验证
**步骤**:
1. 登录系统
2. 查看左侧菜单
**预期**:
- 菜单中存在 `Agent 工坊`
- 菜单中不存在 `Skills 管理`
- 菜单中不存在 `MCP 管理`

### E2E-2: Agent 列表分组切换
**步骤**:
1. 进入 Agent 工坊
2. 确保列表中有 BareLLM 和 CodingAgent 两种类型的 Agent
3. 点击 `个人助理` 分段器
**预期**: 列表仅显示 BareLLM 类型的 Agent

### E2E-3: 新建个人助理完整流程
**步骤**:
1. 点击 `新建 Agent`
2. 选择 `个人助理`
3. 填写名称、描述
4. 切换到 `技能工具` 标签
5. 确认有 Skills / MCP 快捷入口
6. 保存
**预期**: 新建成功，列表中出现该 Agent，类型为 `BareLLM`

### E2E-4: 新建 Claude Code Agent
**步骤**:
1. 点击 `新建 Agent`
2. 选择 `Claude Code`
3. 填写基本信息
4. 切换到 `Claude Code 配置` 标签，填写配置
5. 保存
**预期**: 新建成功，类型为 `CodingAgent`

### E2E-5: 编辑现有 Agent 不受影响
**步骤**:
1. 列表中点击任意已有 Agent 的 `编辑`
2. 修改名称
3. 保存
**预期**: 更新成功，页面刷新后数据正确

### E2E-6: 直接访问 Skills / MCP 页面
**步骤**:
1. 在地址栏直接输入 `/skills`（或对应路由）
2. 页面应正常渲染
**预期**: 页面可访问，功能正常（路由未删除，仅移除菜单入口）

## 回归测试清单

- [ ] 现有 Agent 的启用/停用切换正常
- [ ] 现有 Agent 的"设为默认"正常
- [ ] 现有 Agent 的删除正常
- [ ] 移动端菜单和列表页无样式崩坏
- [ ] 前端 `pnpm run build` 无 TypeScript 错误
- [ ] 后端 `go test ./...` 全部通过
