# 心跳列表改造测试文档

## 测试范围

1. 后端 Domain 层：Heartbeat 模型的创建、校验、快照、Prompt 渲染
2. 后端 Application 层：Heartbeat 服务 CRUD、调度器按心跳粒度调度
3. 后端 Infrastructure 层：Heartbeat 仓储 CRUD、数据迁移
4. 前端：心跳列表的增删改查、编辑弹窗、模板切换
5. E2E：完整用户路径验证

## 单元测试

### UT-1: Heartbeat 创建成功
**目标**: `domain.NewHeartbeat`
**输入**: 合法参数（name="需求派发", interval=30, agentCode="scheduler", requirementType="normal"）
**预期**: 返回非 nil `Heartbeat`，字段值正确，`enabled` 默认为 true

### UT-2: Heartbeat 拒绝空名称
**目标**: `domain.NewHeartbeat`
**输入**: name=""
**预期**: 返回错误，错误信息包含 name required

### UT-3: Heartbeat 拒绝非法间隔
**目标**: `domain.NewHeartbeat`
**输入**: intervalMinutes=0
**预期**: 返回错误

### UT-4: Heartbeat 拒绝空 Agent
**目标**: `domain.NewHeartbeat`
**输入**: agentCode=""
**预期**: 返回错误

### UT-5: Heartbeat Update 校验
**目标**: `Heartbeat.Update`
**输入**: 更新 intervalMinutes 为 -1
**预期**: 返回错误，原有字段不变

### UT-6: Heartbeat Prompt 渲染
**目标**: `Heartbeat.RenderPrompt`
**前置条件**: `mdContent` 包含 `${project.id}`、`${project.name}`、`${timestamp}`
**输入**: 一个 `Project`
**预期**: 返回字符串中模板变量被正确替换

### UT-7: Heartbeat 快照往返
**目标**: `ToSnapshot` / `FromSnapshot`
**输入**: 创建一个 Heartbeat 后生成快照，再用快照恢复
**预期**: 所有字段一致

### UT-8: 调度器 Start 加载所有启用的心跳
**目标**: `HeartbeatScheduler.Start`
**前置条件**: `heartbeatRepo.FindAllEnabled` 返回 3 条记录
**预期**: `cron` 中注册了 3 个任务

### UT-9: 调度器 RefreshSchedule 更新单条心跳
**目标**: `HeartbeatScheduler.RefreshSchedule`
**前置条件**: 某心跳 ID 已注册，修改其 interval 后调用 Refresh
**预期**: 该心跳的 cron 表达式被更新，其他心跳不受影响

### UT-10: 心跳服务创建后刷新调度
**目标**: `HeartbeatApplicationService.CreateHeartbeat`
**前置条件**: mock `scheduler.RefreshSchedule`
**预期**: 创建成功后调用 `RefreshSchedule`

### UT-11: 心跳服务删除后刷新调度
**目标**: `HeartbeatApplicationService.DeleteHeartbeat`
**前置条件**: mock `scheduler.RefreshSchedule`
**预期**: 删除成功后调用 `RefreshSchedule`

### UT-12: executeHeartbeat 创建正确类型的需求
**目标**: `HeartbeatScheduler.executeHeartbeat`
**前置条件**: 某 heartbeat 的 `requirementType = "pr_review"`
**预期**: 创建的 `Requirement` 的 `RequirementType() == "pr_review"`

## 集成测试

### IT-1: SQLite Heartbeat 仓储 CRUD
**目标**: `SQLiteHeartbeatRepository`
**步骤**:
1. `Save` 一条新记录
2. `FindByID` 能读到，字段一致
3. `FindByProjectID` 能读到
4. `FindAllEnabled` 包含该记录（enabled=true）
5. `Delete` 后 `FindByID` 返回 nil

### IT-2: 数据迁移
**目标**: `MigrateHeartbeatToTable`
**前置条件**: 旧 `projects` 表有一条 `heartbeat_enabled=1, agent_code="scheduler", interval=60, md_content="test"` 的记录
**步骤**: 调用迁移函数
**预期**:
- `heartbeats` 表新增一条记录
- `project_id` 正确
- `agent_code="scheduler"`，`interval_minutes=60`，`md_content="test"`
- 再次调用迁移不重复插入

### IT-3: 数据迁移忽略未启用心跳的项目
**目标**: `MigrateHeartbeatToTable`
**前置条件**: 旧项目 `heartbeat_enabled=0`
**步骤**: 调用迁移函数
**预期**: `heartbeats` 表无对应记录

## E2E 测试

### E2E-1: 项目配置页管理心跳列表
**步骤**:
1. 进入项目需求页，创建一个项目
2. 打开项目配置，切换到"心跳管理" Tab
3. 点击"新增心跳"，填写：名称="PR检查"，间隔=15，Agent=scheduler，类型=pr_review，Prompt 使用默认模板
4. 保存
5. 列表中显示该心跳，状态为启用
6. 点击编辑，修改间隔为 30，保存
7. 列表中显示间隔为 30
8. 点击删除，确认后列表为空

**预期**: 以上各步骤均成功，数据持久化正确

### E2E-2: 旧项目迁移后心跳正常工作
**步骤**:
1. 用旧版本 CLI/API 创建一个启用心跳的项目（不经过新 Heartbeat API）
2. 重启服务器（触发迁移）
3. 进入前端项目配置页，查看心跳列表

**预期**:
- 列表中有一条"默认心跳"
- 配置与旧项目的心跳字段一致
- 调度器正常调度该心跳

### E2E-3: 心跳生成的需求类型正确
**步骤**:
1. 为项目创建一个类型为 `optimization` 的心跳
2. 手动触发或通过缩短间隔等待调度
3. 查看需求列表

**预期**:
- 新增一条需求，其 `requirement_type` 为 `optimization`
- 需求标题包含心跳标识 `[心跳]`
