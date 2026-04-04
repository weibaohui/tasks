# 状态机设计文档

## 一、背景

现有需求状态使用固定枚举（todo/preparing/coding/pr_opened/failed/completed/done），无法满足不同项目、不同需求类型的差异化流程管理。需要一套通用状态机机制，支持：

- 按项目独立定义状态机
- 按需求类型绑定不同状态机
- 状态转换时触发 Hook
- 完整的审计日志

## 二、核心概念

| 概念 | 说明 |
|------|------|
| **StateMachine** | 状态机定义，按项目独立，一个项目可有多个 |
| **RequirementType** | 需求类型（normal/heartbeat），每个类型绑定一个状态机 |
| **RequirementState** | 每个需求有自己的状态记录（requirement_id, state_machine_id, current_state） |
| **Transition** | 转换规则（from + to + trigger） |
| **TransitionHook** | 转换钩子，独立于现有 Hook 系统 |
| **TransitionLog** | 审计日志 |

## 三、数据库设计

### 3.1 state_machines（状态机定义）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uuid | 主键 |
| project_id | uuid | 所属项目 |
| name | string | 状态机名称 |
| description | string | 描述 |
| config | jsonb | YAML 解析后的配置 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

### 3.2 state_machine_type_bindings（类型绑定）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uuid | 主键 |
| state_machine_id | uuid | 状态机 ID |
| requirement_type | string | 需求类型（normal/heartbeat） |
| created_at | timestamp | 创建时间 |

### 3.3 requirement_states（需求状态记录）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uuid | 主键 |
| requirement_id | uuid | 需求 ID |
| state_machine_id | uuid | 绑定的状态机 ID |
| current_state | string | 当前状态名称 |
| current_state_id | string | 当前状态 ID |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

### 3.4 transition_definitions（转换规则）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uuid | 主键 |
| state_machine_id | uuid | 所属状态机 |
| from_state | string | 源状态 ID |
| from_state_name | string | 源状态名称 |
| to_state | string | 目标状态 ID |
| to_state_name | string | 目标状态名称 |
| trigger | string | 触发器名称 |
| description | string | 描述 |

### 3.5 transition_hooks（转换钩子）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uuid | 主键 |
| transition_id | uuid | 所属转换规则 |
| name | string | Hook 名称 |
| type | string | hook 类型（如 webhook） |
| config | jsonb | hook 配置 |
| retry_count | int | 重试次数 |
| timeout_seconds | int | 超时秒数 |

### 3.6 transition_logs（审计日志）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uuid | 主键 |
| requirement_id | uuid | 需求 ID |
| from_state | string | 源状态 |
| to_state | string | 目标状态 |
| trigger | string | 触发器 |
| triggered_by | string | 触发者（user/agent/api） |
| remark | string | 备注 |
| result | string | 结果（success/failed） |
| error_message | string | 错误信息 |
| created_at | timestamp | 时间 |

## 四、YAML 配置格式

### 4.1 普通需求状态机示例

```yaml
name: normal_requirement_flow
description: 普通需求流程
initial_state: created

states:
  - id: created
    name: 已创建
    is_final: false
  - id: in_progress
    name: 进行中
    is_final: false
  - id: review
    name: 评审中
    is_final: false
  - id: completed
    name: 已完成
    is_final: true
  - id: cancelled
    name: 已取消
    is_final: true

transitions:
  - from: created
    to: in_progress
    trigger: start
    description: 开始处理

  - from: in_progress
    to: review
    trigger: submit_review
    description: 提交评审
    hooks:
      - name: 通知评审人
        type: webhook
        config:
          url: "http://notify-service/review"
          method: POST
          body: "{{.requirement}}"
        retry: 1
        timeout: 30

  - from: review
    to: completed
    trigger: approve
    description: 评审通过

  - from: review
    to: in_progress
    trigger: reject
    description: 评审驳回

  - from: in_progress
    to: cancelled
    trigger: cancel
    description: 取消

  - from: created
    to: cancelled
    trigger: cancel
    description: 取消
```

### 4.2 心跳需求状态机示例

```yaml
name: heartbeat_flow
description: 心跳任务流程
initial_state: active

states:
  - id: active
    name: 活跃
    is_final: false
  - id: stopped
    name: 已停止
    is_final: true

transitions:
  - from: active
    to: stopped
    trigger: stop
    description: 停止心跳

  - from: stopped
    to: active
    trigger: restart
    description: 重启心跳
```

## 五、API 设计

### 5.1 状态机管理

```
GET    /projects/{project_id}/state-machines
       → 列出项目下所有状态机

POST   /projects/{project_id}/state-machines
       Body: { name, description, config }  # config 为 YAML 内容
       → 创建状态机

GET    /projects/{project_id}/state-machines/{id}
       → 获取状态机详情

PUT    /projects/{project_id}/state-machines/{id}
       Body: { name, description, config }
       → 更新状态机

DELETE /projects/{project_id}/state-machines/{id}
       → 删除状态机（不检查是否被使用）

POST   /projects/{project_id}/state-machines/{id}/bind
       Body: { requirement_type }
       → 绑定需求类型

DELETE /projects/{project_id}/state-machines/{id}/bind/{type}
       → 解绑需求类型
```

### 5.2 状态转换

```
POST   /requirements/{requirement_id}/transitions
       Body: { trigger, triggered_by, remark }
       → 触发转换
       → 成功返回新状态
       → 失败返回错误

GET    /requirements/{requirement_id}/state
       → 获取当前状态

GET    /requirements/{requirement_id}/transitions/history
       → 获取转换历史
```

### 5.3 状态查询（前端用）

```
GET    /projects/{project_id}/requirements/states/summary
       → 获取项目下所有需求的状态统计
       → 返回: { "in_progress": 5, "completed": 10, ... }
```

## 六、流程说明

### 6.1 创建需求

```
创建 Requirement
  → 根据 requirement_type 查找绑定状态机
  → 创建 RequirementState（初始状态 from config.initial_state）
  → 记录 TransitionLog（result: success）
```

### 6.2 触发转换

```
接收请求 (trigger, triggered_by, remark)
  → 查找 RequirementState
  → 校验 (current_state, trigger) 是否有对应 Transition
  → 更新 RequirementState.current_state
  → 异步执行 TransitionHooks（goroutine）
    → 失败重试1次
    → 仍失败记录 warning 到 TransitionLog.error_message
  → 记录 TransitionLog（异步执行结果不阻塞）
```

### 6.3 终态

处于 `is_final: true` 的状态不可作为 Transition 的 `from`。

## 七、目录结构

```
backend/
├── domain/
│   └── state_machine/
│       ├── state_machine.go          # 状态机聚合根
│       ├── requirement_state.go     # 需求状态实体
│       ├── transition.go            # 转换规则值对象
│       ├── transition_hook.go       # 转换钩子值对象
│       └── transition_log.go       # 审计日志实体
├── application/
│   └── state_machine_service.go     # 应用服务
├── infrastructure/
│   └── state_machine/
│       ├── persistence/
│       │   ├── sqlite_state_machine_repo.go
│       │   ├── sqlite_requirement_state_repo.go
│       │   └── sqlite_transition_log_repo.go
│       └── transition_executor.go  # 转换执行器（异步执行 hooks）
└── interfaces/
    └── http/
        ├── state_machine_handler.go
        └── router.go
```
