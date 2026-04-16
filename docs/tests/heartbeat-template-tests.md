# 心跳模板管理测试文档

## 后端测试

### Domain 层

- **TestNewHeartbeatTemplate**
  - 正常创建：name/mdContent/requirementType 均有效时返回实体。
  - 空名称：返回错误。
  - 空格名称：返回错误。
  - 空 requirementType：使用默认值 `heartbeat`。

- **TestHeartbeatTemplateUpdate**
  - 更新后字段正确。
  - 更新时 name 为空返回错误。

- **TestHeartbeatTemplateSnapshotRoundTrip**
  - `ToSnapshot` 与 `FromSnapshot` 互逆。

### Application 层

- **TestHeartbeatTemplateService_Create**
  - 正常创建后可在列表中查询到。

- **TestHeartbeatTemplateService_List**
  - 返回按创建时间排序的模板列表。

- **TestHeartbeatTemplateService_Delete**
  - 删除成功后再次查询返回空。

### Infrastructure 层

- **TestSQLiteHeartbeatTemplateRepository_SaveAndFind**
  - 保存后按 ID 查询字段一致。

- **TestSQLiteHeartbeatTemplateRepository_FindAll**
  - 多条记录时返回全部。

- **TestSQLiteHeartbeatTemplateRepository_Delete**
  - 删除后查询返回 nil。

## 前端测试

### 编译验证

- `pnpm run build` 无 TypeScript 编译错误。

### E2E / 手工验证清单

1. 打开项目配置 → 心跳管理 → 新增心跳。
2. 验证"选择模板"下拉框加载了预置的 3 个模板。
3. 选择"处理 PR"模板，验证 `md_content` 和 `requirement_type` 自动填充为 `pr_review`。
4. 修改 `md_content`，点击"保存为模板"，输入"处理 PR V2"，提交成功。
5. 再次打开新增心跳，验证下拉框中出现"处理 PR V2"。
6. 删除一个模板（如有删除入口），验证列表刷新。
7. 查看 Pet Shop 项目的心跳列表，验证有 3 条独立心跳。
