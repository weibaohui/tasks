# 项目最大并发 Agent 数配置需求文档

## 背景

当前系统派发需求时，没有限制单个项目同时运行的 Agent 数量。当项目中的需求较多时，可能同时启动大量 Agent，导致资源占用过高、LLM 调用费用激增，甚至触发渠道限流。用户需要为每个项目配置一个合理的并发上限。

## 目标

1. 在项目配置页面增加"最大并发 Agent 数"选项
2. 后端派发需求时 enforce 该限制
3. 默认值 2，可配置范围 1-10

## 功能需求

### FR-1 项目配置页增加并发限制字段
- 位置：项目配置 Drawer → "基本信息" Tab
- 控件：数字输入框（`InputNumber`）
- 约束：最小值 1，最大值 10，默认值 2
- 保存时随其他项目基本信息一起提交

### FR-2 后端领域模型扩展
- `Project` 领域模型增加 `max_concurrent_agents` 字段
- 新建项目时默认值为 2
- 更新时校验范围 1-10，非法值返回明确错误

### FR-3 派发时 enforce 并发限制
- `DispatchRequirement` 在正式启动 Agent 前，统计当前项目处于 "preparing" 或 "coding" 状态的需求数量
- 若数量 >= `max_concurrent_agents`，拒绝派发并返回 `ErrMaxConcurrentAgentsReached`
- 心跳自动派发同样受该限制约束（心跳调度器调用同一 DispatchRequirement 入口）

### FR-4 数据库兼容
- 新表结构默认 `max_concurrent_agents INTEGER NOT NULL DEFAULT 2`
- 旧数据库通过迁移脚本自动添加该列并设默认值为 2

## 非功能需求

- 测试覆盖率：新增代码测试覆盖率 >= 80%
- 前端 E2E：配置页保存后重新打开，值应正确回显
- 向后兼容：未配置过该字段的旧项目，行为等价于默认值 2
