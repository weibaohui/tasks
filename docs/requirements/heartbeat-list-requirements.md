# 心跳列表改造需求文档

## 背景

当前系统的心跳机制是项目级别的单一配置：一个项目只有一个心跳模板、一个调度间隔和一个执行 Agent。随着业务发展，一个心跳 Prompt 里需要同时处理"需求派发、PR 审查、优化建议"等多项任务，导致：

1. Prompt 越来越臃肿，难以维护
2. 无法单独开关或调整某一项任务
3. 所有心跳生成的需求类型都是 `heartbeat`，无法利用已有的 `pr_review`/`optimization` 等类型和对应状态机

因此需要将"单心跳"改造为"心跳列表"，每个心跳只负责一件事。

## 目标

1. 将 `Project` 上的心跳配置剥离为独立的 `Heartbeat` 聚合
2. 支持一个项目配置多个心跳，每个心跳独立管理
3. 每个心跳可独立配置：名称、开关、调度间隔、Agent、Prompt 模板、生成的需求类型
4. 向后兼容：旧项目的单心跳数据自动迁移为默认心跳记录

## 功能需求

### FR-1 独立 Heartbeat 领域模型
- 新增 `Heartbeat` 领域实体，包含字段：`id`、`project_id`、`name`、`enabled`、`interval_minutes`、`md_content`、`agent_code`、`requirement_type`、`sort_order`、`created_at`、`updated_at`
- `NewHeartbeat()` 校验：`name` 非空、`interval_minutes >= 1`、`agent_code` 非空
- `Heartbeat` 提供 `RenderPrompt(project *Project) string` 方法，支持 `${project.id}` 等模板变量

### FR-2 从 Project 剥离心跳字段
- `Project` 移除字段：`heartbeat_enabled`、`heartbeat_interval_minutes`、`heartbeat_md_content`、`agent_code`
- 移除 `Project.UpdateHeartbeatConfig` 方法
- `Project` 保留全局派发配置：`dispatch_channel_code`、`dispatch_session_key`、`max_concurrent_agents`

### FR-3 Heartbeat 仓储与数据迁移
- 新增 `heartbeats` 表及索引
- 新增 `HeartbeatRepository` 接口及 SQLite 实现
- 启动时执行迁移：将旧项目的心跳字段转换为一条默认 `Heartbeat` 记录（若 `heartbeat_enabled=1` 且 `agent_code` 非空）
- `projects` 表旧列保留不删除，但代码层不再读写

### FR-4 调度器按心跳粒度调度
- `HeartbeatScheduler` 改为按 `Heartbeat` 注册 cron 任务，而非按 `Project`
- 每个心跳独立调度，互不干扰
- 支持单条心跳的刷新/移除，不需要重启整个 cron

### FR-5 心跳应用服务
- 新增 `HeartbeatApplicationService`，提供 Heartbeat 的 CRUD 命令
- 每次增删改后同步刷新调度器

### FR-6 HTTP 接口
- 新增 Heartbeat REST API：列表、创建、获取、更新、删除
- `ProjectHandler` 移除心跳相关字段的接收和返回
- 路由注册到 `/api/heartbeats`

### FR-7 CLI 命令重构
- `taskmanager project heartbeat` 子命令重构为基于心跳 ID 的 CRUD：
  - `list <project_id>`：列出项目心跳
  - `create <project_id>`：创建心跳
  - `update <heartbeat_id>`：更新心跳
  - `delete <heartbeat_id>`：删除心跳
  - `enable/disable <heartbeat_id>`：开关心跳

### FR-8 前端心跳列表
- 项目配置页面移除原有"心跳模板"大文本框
- 新增"心跳列表"区域，展示项目下所有心跳
- 支持新增、编辑、删除、开关、排序
- 心跳编辑弹窗包含：名称、间隔、Agent、需求类型、Prompt 编辑器
- 不同需求类型提供不同的默认 Prompt 模板

### FR-9 需求类型绑定
- 每个心跳配置 `requirement_type`，执行时创建对应该类型的需求
- 支持类型：`heartbeat`、`normal`、`pr_review`、`optimization`

## 非功能需求

- 测试覆盖率：新增代码测试覆盖率 >= 80%
- 向后兼容：旧数据库启动后自动迁移，原有心跳行为不变
- DDD 分层：Heartbeat 作为独立聚合，不破坏 domain → application → infrastructure → interfaces 的分层
