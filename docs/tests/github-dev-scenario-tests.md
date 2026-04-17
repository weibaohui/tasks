# GitHub 开发协作心跳场景测试文档

## 后端测试

### Domain 层

#### HeartbeatScenario
- **TestNewHeartbeatScenario**
  - 正常创建：code/name/items 均有效时返回实体。
  - 空 code：返回错误。
  - 空格 code：返回错误（trim 后为空）。
  - 空 name：返回错误。
  - items 为空切片：允许创建（虽然无实际心跳）。

- **TestHeartbeatScenario_ApplyToProject**
  - 将场景应用到项目，返回的心跳数量等于场景 items 数量。
  - 每个返回心跳的 projectID 与传入一致。
  - 返回心跳的 name 按格式为 `"{scenarioName} - {itemName}"`。
  - 返回心跳的 intervalMinutes/agentCode/requirementType/mdContent 与 item 定义一致。

- **TestHeartbeatScenario_Update**
  - 更新名称和 items 后字段正确。
  - 更新时 name 为空返回错误。

#### Project（扩展）
- **TestProject_SetHeartbeatScenarioCode**
  - 设置场景编码后 `HeartbeatScenarioCode()` 返回正确值。
  - `updatedAt` 被更新。

### Application 层

#### HeartbeatScenarioService
- **TestHeartbeatScenarioService_Create**
  - 正常创建后可通过 code 查询。
  - 创建重复 code 返回错误。

- **TestHeartbeatScenarioService_GetByCode**
  - 存在的 code 返回场景。
  - 不存在的 code 返回 nil 或错误。

- **TestHeartbeatScenarioService_List**
  - 返回所有场景列表（包含内置场景）。

- **TestHeartbeatScenarioService_ApplyScenarioToProject**
  - 项目应用场景成功后，项目下心跳数量等于场景 items 数量。
  - 再次应用同一场景：先清理旧场景心跳，再重新创建（幂等性）。
  - 应用到不存在的场景返回错误。
  - 应用到不存在的项目返回错误。

### Infrastructure 层

#### SQLiteHeartbeatScenarioRepository
- **TestSQLiteHeartbeatScenarioRepository_SaveAndFindByCode**
  - 保存后按 code 查询字段一致（含 JSON items 反序列化）。

- **TestSQLiteHeartbeatScenarioRepository_FindAll**
  - 多条记录时返回全部。

#### SQLiteProjectRepository（扩展）
- **TestSQLiteProjectRepository_SaveWithScenarioCode**
  - 保存包含 `heartbeat_scenario_code` 的项目后，查询返回该字段值正确。

### HTTP Handler 层

- **TestHeartbeatScenarioHandler_List**
  - GET `/api/v1/heartbeat-scenarios` 返回 200 和场景数组。

- **TestHeartbeatScenarioHandler_ApplyScenario**
  - POST `/api/v1/projects/:project_id/apply-scenario` 正确应用场景返回 200。
  - 缺少 `scenario_code` body 返回 400。
  - 不存在的 scenario_code 返回 404。

## 集成测试

### 内置场景初始化
- 服务器启动时，`HeartbeatScenarioService` 自动检查并创建 `github_dev_workflow` 内置场景。
- 验证内置场景的 code/name/items 数量与需求文档一致（8 个心跳）。

### 应用场景端到端
1. 创建一个测试项目。
2. 调用 `ApplyScenarioToProject` 应用 `github_dev_workflow`。
3. 查询项目心跳列表，验证存在 8 条心跳，且 enabled=true。
4. 验证调度器已注册这些心跳（可通过日志或调度器 entries 断言）。

## 前端测试

### 编译验证
- `pnpm run build` 无 TypeScript 编译错误。

### E2E / 手工验证清单

1. 进入项目配置页面，"心跳场景"下拉框展示"GitHub 开发协作工作流"。
2. 选择该场景，点击"应用"。
3. 进入项目心跳列表，验证出现 8 条心跳：
   - Issue 分析 (30 分钟)
   - LGTM 代码编写 (30 分钟)
   - PR 需求评审 (30 分钟)
   - PR 代码质量评审 (30 分钟)
   - PR 修改修复 (30 分钟)
   - PR 合并检查 (30 分钟)
   - PR 文档补充 (60 分钟)
   - PR 测试补充 (60 分钟)
4. 对其中一条心跳点击"禁用"，验证调度器停止触发该心跳。
5. 修改一条心跳的间隔为 60 分钟，验证保存成功且调度器重新注册。
6. 重新应用一次场景，验证旧场景心跳被清理并重新创建（或按设计的简化策略处理）。
7. 手动触发"Issue 分析"心跳，验证成功创建一个 requirement 并被派发。
