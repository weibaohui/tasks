# 项目最大并发 Agent 数配置测试文档

## 测试范围

1. 后端 Domain 层：Project 模型新增字段的默认值、校验、快照往返
2. 后端 Application 层：项目服务更新字段、派发服务并发限制
3. 前端：配置页表单渲染、保存、回显
4. E2E：完整用户路径验证

## 单元测试

### UT-1: Project 默认最大并发数为 2
**目标**: `domain.NewProject`
**输入**: 合法项目参数
**预期**: `project.MaxConcurrentAgents() == 2`

### UT-2: Project 设置合法并发数
**目标**: `project.SetMaxConcurrentAgents`
**输入**: `1`, `5`, `10`
**预期**: 全部成功，`updatedAt` 更新

### UT-3: Project 拒绝非法并发数
**目标**: `project.SetMaxConcurrentAgents`
**输入**: `0`, `11`, `-1`
**预期**: 返回 `ErrProjectMaxConcurrentAgentsInvalid`，字段值不变

### UT-4: Project 快照往返保留并发数
**目标**: `ToSnapshot` / `FromSnapshot`
**输入**: 设置 `max_concurrent_agents = 3` 后生成快照
**预期**: 从快照恢复的项目 `MaxConcurrentAgents() == 3`

### UT-5: 项目服务更新最大并发数
**目标**: `ProjectApplicationService.UpdateProject`
**输入**: `UpdateProjectCommand{MaxConcurrentAgents: ptr(4)}`
**预期**: 更新成功，返回的项目 `MaxConcurrentAgents() == 4`

### UT-6: 项目服务拒绝非法并发数
**目标**: `ProjectApplicationService.UpdateProject`
**输入**: `UpdateProjectCommand{MaxConcurrentAgents: ptr(0)}`
**预期**: 返回 `ErrProjectMaxConcurrentAgentsInvalid`

### UT-7: 派发服务达到并发上限时拒绝
**目标**: `RequirementDispatchService.DispatchRequirement`
**前置条件**:
  - 项目 `max_concurrent_agents = 2`
  - `requirementRepo.Count(preparing+coding)` 返回 2
**预期**: 返回 `ErrMaxConcurrentAgentsReached`

### UT-8: 派发服务未达上限时允许继续
**目标**: `RequirementDispatchService.DispatchRequirement`
**前置条件**:
  - 项目 `max_concurrent_agents = 2`
  - `requirementRepo.Count(preparing+coding)` 返回 1
**预期**: 不返回并发限制错误，流程继续

## E2E 测试

### E2E-1: 项目配置页保存并回显最大并发 Agent 数
**步骤**:
1. 登录系统，进入"项目需求"页
2. 创建一个项目
3. 点击项目卡片的"设置"图标打开配置 Drawer
4. 在"基本信息" Tab 中，将"最大并发 Agent 数"修改为 3
5. 点击"保存基本信息"
6. 关闭 Drawer，重新打开
**预期**:
- 保存成功提示出现
- 重新打开后"最大并发 Agent 数"显示为 3

### E2E-2: 派发需求受并发限制
**步骤**:
1. 创建一个项目，配置"最大并发 Agent 数"为 1
2. 创建两个需求
3. 派发第一个需求（手动或 API 直接修改 DB 使其状态为 preparing/coding）
4. 尝试派发第二个需求
**预期**:
- 第二个需求派发失败，提示"max concurrent agents limit reached for project"
