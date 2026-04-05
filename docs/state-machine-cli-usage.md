# 状态机 CLI 使用指南

## 概述

状态机实现了一个 DevOps 开发流程，从代码提交到最终上线，经历多个环节的转换。

## 流程图

```
submitted → in_review → building → testing → completed
     ↓           ↓
  rejected    rejected (can go back to submitted)
```

## 快速开始

### 1. 创建状态机

```bash
taskmanager statemachine create \
  -n "开发流程" \
  -d "从代码提交到上线的完整流程" \
  -c '{
    "name": "开发流程",
    "initial_state": "submitted",
    "states": [
      {"id": "submitted", "name": "已提交", "is_final": false},
      {"id": "in_review", "name": "审查中", "is_final": false},
      {"id": "building", "name": "构建中", "is_final": false},
      {"id": "testing", "name": "测试中", "is_final": false},
      {"id": "completed", "name": "已完成", "is_final": true},
      {"id": "rejected", "name": "已拒绝", "is_final": false}
    ],
    "transitions": [
      {"from": "submitted", "to": "in_review", "trigger": "submit_review", "description": "提交审查"},
      {"from": "in_review", "to": "building", "trigger": "approve", "description": "审查通过"},
      {"from": "in_review", "to": "rejected", "trigger": "reject", "description": "审查拒绝"},
      {"from": "rejected", "to": "submitted", "trigger": "revise", "description": "修订后重新提交"},
      {"from": "building", "to": "testing", "trigger": "build_success", "description": "构建成功"},
      {"from": "building", "to": "rejected", "trigger": "build_failed", "description": "构建失败"},
      {"from": "testing", "to": "completed", "trigger": "test_pass", "description": "测试通过"},
      {"from": "testing", "to": "building", "trigger": "test_failed", "description": "测试失败"}
    ]
  }'
```

返回示例：
```json
{"id":"abc123","name":"开发流程","description":"从代码提交到上线的完整流程",...}
```

### 2. 初始化需求状态

将一个需求（requirement）绑定到状态机，开始流程：

```bash
# 初始化需求 req-001 到状态机 abc123
taskmanager statemachine init req-001 -s abc123
```

返回示例：
```json
{"requirement_id":"req-001","current_state":"submitted","current_state_name":"已提交",...}
```

### 3. 执行状态转换

使用 `transition` 命令，将需求从当前状态转换到下一个状态：

```bash
# 提交审查 (submitted → in_review)
taskmanager statemachine transition req-001 -t submit_review -b "developer" -r "代码完成提交审查"

# 审查通过 (in_review → building)
taskmanager statemachine transition req-001 -t approve -b "reviewer" -r "代码审查通过"

# 构建成功 (building → testing)
taskmanager statemachine transition req-001 -t build_success -b "ci_system"

# 测试通过 (testing → completed)
taskmanager statemachine transition req-001 -t test_pass -b "qa_engineer"
```

参数说明：
- `-t, --trigger`: 触发器名称（必填），对应状态机配置中的 trigger
- `-b, --by`: 触发者标识（可选，默认 "cli"）
- `-r, --remark`: 备注说明（可选）

### 4. 查看当前状态

```bash
taskmanager statemachine state req-001
```

返回：
```json
{"requirement_id":"req-001","current_state":"testing","current_state_name":"测试中",...}
```

### 5. 查看转换历史

```bash
taskmanager statemachine history req-001
```

返回：
```json
[
  {"from_state":"","to_state":"submitted","trigger":"init","triggered_by":"system",...},
  {"from_state":"submitted","to_state":"in_review","trigger":"submit_review","triggered_by":"developer",...},
  {"from_state":"in_review","to_state":"building","trigger":"approve","triggered_by":"reviewer",...},
  {"from_state":"building","to_state":"testing","trigger":"build_success","triggered_by":"ci_system",...}
]
```

### 6. 其他命令

```bash
# 列出所有状态机
taskmanager statemachine list

# 删除状态机
taskmanager statemachine delete <state-machine-id>
```

## 完整使用示例

```bash
#!/bin/bash
set -e

# 1. 创建状态机
SM=$(taskmanager statemachine create \
  -n "CI/CD流程" \
  -d "持续集成/部署流程" \
  -c '{
    "name":"CI/CD流程",
    "initial_state":"code_submitted",
    "states":[
      {"id":"code_submitted","name":"代码已提交","is_final":false},
      {"id":"code_review","name":"代码审查","is_final":false},
      {"id":"unit_test","name":"单元测试","is_final":false},
      {"id":"integration_test","name":"集成测试","is_final":false},
      {"id":"deploy_staging","name":"部署预发布","is_final":false},
      {"id":"deploy_production","name":"部署生产","is_final":true}
    ],
    "transitions":[
      {"from":"code_submitted","to":"code_review","trigger":"start_review"},
      {"from":"code_review","to":"unit_test","trigger":"review_passed"},
      {"from":"code_review","to":"code_submitted","trigger":"review_failed"},
      {"from":"unit_test","to":"integration_test","trigger":"unit_passed"},
      {"from":"unit_test","to":"code_submitted","trigger":"unit_failed"},
      {"from":"integration_test","to":"deploy_staging","trigger":"integration_passed"},
      {"from":"integration_test","to":"code_submitted","trigger":"integration_failed"},
      {"from":"deploy_staging","to":"deploy_production","trigger":"staging_verified"},
      {"from":"deploy_staging","to":"integration_test","trigger":"staging_failed"}
    ]
  }')

SM_ID=$(echo $SM | jq -r '.id')
echo "创建状态机: $SM_ID"

# 2. 初始化需求
taskmanager statemachine init REQ-001 -s $SM_ID

# 3. 执行流程
echo "开始代码审查..."
taskmanager statemachine transition REQ-001 -t start_review -b "dev"

echo "审查通过，运行单元测试..."
taskmanager statemachine transition REQ-001 -t review_passed -b "reviewer"

echo "单元测试通过，运行集成测试..."
taskmanager statemachine transition REQ-001 -t unit_passed -b "test_runner"

echo "集成测试通过，部署预发布..."
taskmanager statemachine transition REQ-001 -t integration_passed -b "ci"

echo "预发布验证通过，部署生产..."
taskmanager statemachine transition REQ-001 -t staging_verified -b "release_manager"

# 4. 查看最终状态
echo "最终状态:"
taskmanager statemachine state REQ-001 | jq .

echo "转换历史:"
taskmanager statemachine history REQ-001 | jq .
```

## 触发器参考

| 触发器 | 从状态 | 到状态 | 说明 |
|--------|--------|--------|------|
| `submit_review` | submitted | in_review | 提交审查 |
| `approve` | in_review | building | 审查通过 |
| `reject` | in_review | submitted | 审查拒绝（打回） |
| `build_success` | building | testing | 构建成功 |
| `build_failed` | building | submitted | 构建失败 |
| `test_pass` | testing | completed | 测试通过 |
| `test_failed` | building | testing | 测试失败 |

## 注意事项

1. **状态机 ID**：创建状态机后保存返回的 `id`，用于后续初始化和删除
2. **需求 ID**：可以是任意字符串，如 `REQ-001`、`feature-123`、`user-story-456`
3. **转换校验**：只能执行当前状态允许的转换，否则会报错
4. **元数据传递**：通过 `--metadata` 参数可传递额外上下文信息（用于 Hook 变量替换）

```bash
# 带元数据的转换（元数据会传递给 Hook）
taskmanager statemachine transition REQ-001 -t approve -b "reviewer" \
  --metadata '{"project_id":"proj-1","environment":"production"}'
```
