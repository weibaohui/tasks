# AI 自主驱动的任务闭环 — 详细设计

> 本文档描述「单一管家心跳 + 元需求 + 少量专用心跳 + 确定性/显式状态迁移」的完整设计，可与 `backend/domain/statemachine`、心跳触发链路对照实现。

---

## 目录

1. [目标与边界](#1-目标与边界)
   - [1.4 任务粒度原则（心跳与需求要足够简单）](#14-任务粒度原则心跳与需求要足够简单)
2. [概念与术语](#2-概念与术语)
3. [端到端流程](#3-端到端流程)
4. [架构与组件](#4-架构与组件)
   - [4.1 概念模型：船长、状态机与触发源](#41-概念模型船长状态机与触发源)
     - [关系总图（Mermaid）](#关系总图mermaid)
     - [术语与模块对照](#术语与模块对照)
     - [数据落在何处](#数据落在何处)
5. [GitHub 约定（唯一来源）](#5-github-约定唯一来源)
6. [状态迁移权威模型](#6-状态迁移权威模型)
7. [元需求数据模型](#7-元需求数据模型)
8. [状态机完整配置](#8-状态机完整配置)
9. [管家心跳](#9-管家心跳)
   - [9.0 管家运行上下文（Runbook）](#90-管家运行上下文runbook)
   - [9.0.8 项目航海日记：推荐数据结构](#908-项目航海日记推荐数据结构)
10. [专用心跳](#10-专用心跳)
11. [触发与参数传递](#11-触发与参数传递)
12. [Webhook 设计](#12-webhook-设计)
13. [后端接口与模块](#13-后端接口与模块)
14. [前端与可观测性](#14-前端与可观测性)
15. [错误处理与幂等](#15-错误处理与幂等)
16. [实施阶段与任务拆解](#16-实施阶段与任务拆解)
17. [测试策略](#17-测试策略)
18. [风险](#18-风险)

---

## 1. 目标与边界

### 1.1 目标

| 编号 | 目标 |
|------|------|
| G1 | 以 **GitHub Issue/PR** 为协作面，在系统内可追踪每条 Issue/PR 的**生命周期状态**。 |
| G2 | **单一调度入口（管家心跳）** 周期性整合扫描，避免多路定时心跳重复打 GitHub API。 |
| G3 | **专用心跳** 只做重活（实现、修 PR 等），由管家或状态机 Hook **按需触发**，并支持 **参数**（issue/pr 号）。 |
| G4 | 每一次状态迁移**可审计**：要么来自**服务端规则/Webhook**，要么来自 **Agent 显式调用** 迁移 API。 |
| G5 | 自动化中心可展示 **元需求看板** 与 **心跳执行轨迹**（含多类型心跳需求）。 |
| G6 | **管家心跳**具备跨轮次的 **运行上下文（Runbook）**：知道上一轮做了什么、结果如何、本轮焦点与阶段目标，避免「每次从零推理」。子任务仍保持短小（§1.4），连续性由 Runbook + 元需求状态机 + GitHub 真相共同提供。 |

### 1.2 非目标

- 不引入独立 BPMN/Workflow 引擎。
- 不要求与历史「8 路并行心跳」兼容；专用心跳数量可裁剪。
- 不在本文内规定具体 Agent 模型与工具链版本。

### 1.3 与代码库的硬约束

状态机配置解析与校验见 `backend/domain/statemachine/state_machine.go`：

- 必须包含状态 ID **`todo`** 与 **`completed`**（`completed` 须 `is_final: true`）。
- `todo` 至少有一条出向 `transitions`。
- **同一 `(from_state, trigger)` 组合在实现上取第一条匹配**（`FindTransition`），故 **禁止** 重复定义相同 `from` + `trigger` 指向不同 `to`。

下文 YAML 已全部按上述约束编排。

### 1.4 任务粒度原则（心跳与需求要足够简单）

复杂流程靠 **状态机与管家多次调度** 拼出来，**不靠**「一次心跳里塞满全流程」。类比现实世界里 **1+1=2** 那种清晰、可一眼判对错的粒度：每一次由心跳派生出的 **需求（Requirement）** 都应是 **独立、完整、短小** 的任务单元。

| 原则 | 含义 |
|------|------|
| **一事一任务** | 一次派发只做 **一件** 可验收的事：例如「对 Issue #42 发一条分析评论」「对 PR #7 修掉评审里指出的 A 点」「开一个关联 Issue 的 PR」——不要在同一需求里同时「扫全库 + 分析 + 实现 + 评审」。 |
| **时间有界** | 单次执行应有 **预期上限**（由团队定，例如轻量 5～15 分钟级、实现类 30～90 分钟级）；超时则失败可重试或拆子任务，而不是无限跑。 |
| **可验收** | 结束条件像算术一样明确：「评论已发」「PR 已开」「push 已完成」；避免「尽量做好」式描述。 |
| **复杂是编排的事** | 长链路 = 多个状态 × 多次心跳/多次需求；**禁止**用超长 Prompt 替代编排。 |
| **管家也要轻** | 管家一轮只做：扫一眼 → 处理 **1～2** 个事项 → 其余下轮；不把「整个项目季度规划」塞进一次管家任务。 |

**反例**：一个心跳 Prompt 要求 Agent「列出所有 open issue、逐个深度分析、再选三个写代码、再开 PR」——应拆成 **多次触发** 或 **多个专用心跳各带单 issue/pr 参数**。

**正例**：专用心跳 `github_implement` 仅带 `issue_number=42`，目标只有一个：在本轮可接受时间内完成该 Issue 对应实现并开 PR（若太大则先在 Issue 侧拆分子 Issue，再分别派发）。

**与 Runbook 的关系**：§1.4 要求单次执行「短」，**管家**仍须携带跨轮 **[§9.0 Runbook](#90-管家运行上下文runbook)**——那是编排用的记事本（上轮结论、阶段目标、本轮焦点），与「单轮动作范围小」不矛盾：管家每轮仍只处理 1～2 个事项，但**决策**应基于 Runbook + 元需求状态 + `gh` 现查。

---

## 2. 概念与术语

| 术语 | 说明 |
|------|------|
| **元需求（Meta Requirement）** | 一条 `Requirement`，`requirement_type` 为 `github_meta_issue` 或 `github_meta_pr`，用于绑定某 GitHub Issue/PR 与生命周期状态机。 |
| **项目需求（Work Requirement）** | 系统内可 **dispatch** 给 CodingAgent 的需求，由 Issue 在 label `lgtm` 后创建，承载实际开发；通过外键式字段或描述关联 Issue 元需求。 |
| **管家心跳** | 定时（+ 可选 Webhook 唤醒）执行，负责扫库、维护元需求、规则迁移、触发专用心跳；**每轮派发前注入 Runbook**（见 §9.0）。 |
| **Runbook（管家运行上下文）** | 按项目（或按管家心跳实例）持久化的摘要与指针：上一轮结论、当前焦点、阶段目标；**不是**把整段对话历史塞进 Prompt。 |
| **专用心跳** | 如 `github_implement`、`github_pr_fix`；**不以全库扫描为主路径**，由参数指定目标 Issue/PR。 |
| **确定性迁移** | 不依赖模型语义，由 `gh` JSON、Webhook payload、标签存在性等判定后直接 `TriggerTransition`。 |
| **显式迁移** | Agent 在任务结束时调用 `POST .../state-machines/.../transition` 或 CLI 等价物。 |

---

## 3. 端到端流程

### 3.1 主流程（文字）

1. 用户在 GitHub 创建 Issue。
2. 管家扫描 → 若无元需求则 **创建 `github_meta_issue`**，状态机从 `todo` 进入后续阶段（可由 AI 或人工在 GitHub 协作）。
3. Issue 经讨论后打上 **label `lgtm`** → **服务端规则**（或管家内规则）将元需求迁移到「可同步项目需求」状态 → 创建 **项目需求** 并关联 `meta_object_id`。
4. 管家或人工触发 **派发** 或 **专用心跳 `github_implement`**（带 `issue_number`）。
5. Agent 创建 PR（`Closes #n`）→ Webhook / 规则将 Issue 元需求迁到「已关联 PR」，PR 侧 **创建 `github_meta_pr`** 或使用已有追踪。
6. PR 走评审、修复、合并闸门；合并事件 → **确定性** 将 PR 元需求、`github_meta_issue` 置为 `completed`。

### 3.2 时序图（管家一轮）

```mermaid
sequenceDiagram
    participant Cron as 调度器
    participant HB as 管家心跳任务
    participant API as Taskmanager API
    participant GH as gh / GitHub
    participant SM as 状态机服务
    participant Sub as 专用心跳(可选)

    Cron->>HB: 触发心跳
    HB->>API: 创建心跳需求并派发
    Note over HB: Agent 执行 Prompt
    HB->>GH: issue list / pr list (json)
    GH-->>HB: 开放项列表
    HB->>API: 查询/创建元需求
    HB->>SM: 规则迁移(无模型)
    alt 需重活
        HB->>API: TriggerHeartbeatWithParams
        API->>Sub: 创建子任务并派发
    end
    HB->>API: 更新元需求 meta_last_*
```

### 3.3 对象关系（示意）

```mermaid
flowchart LR
    subgraph GitHub
        I[Issue]
        P[PR]
    end
    subgraph Taskmanager
        MI[元需求 github_meta_issue]
        MP[元需求 github_meta_pr]
        WR[项目需求 normal/...]
    end
    I -.->|追踪| MI
    P -.->|追踪| MP
    MI -->|LGTM 后创建关联| WR
    WR -->|产生| P
```

---

## 4. 架构与组件

```
┌─────────────────────────────────────────────────────────────┐
│                     管家心跳 github_project_manager          │
│   输入: 项目配置、gh 认证、上一轮 meta_last_*               │
│   输出: 元需求 CRUD、规则迁移、TriggerHeartbeatWithParams    │
└────────────────────────────┬────────────────────────────────┘
                             │
     ┌───────────────────────┼───────────────────────┐
     ▼                       ▼                       ▼
┌─────────┐           ┌───────────┐           ┌───────────────┐
│ GitHub  │           │ 状态机 SM  │           │ 专用心跳       │
│ gh CLI  │           │ Issue/PR  │           │ implement/fix │
└─────────┘           └───────────┘           └───────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ Webhook ingress   │
                    │ PR merge / sync    │
                    └─────────────────┘
```

### 4.1 概念模型：船长、状态机与触发源

本节把前文术语 **收束成一套可对外讲解的比喻**，与实现一一对应。

#### 船长（项目管家）

| 比喻 | 实现 |
|------|------|
| **船长** | **项目管家**（管家类心跳所派发的职责）：有 **航海日记**（Runbook 快照 + 航程条目 §9.0.8）、有 **阶段目标**（`phase_goal`）、知道 **现在走到哪**（快照 + 元需求状态机 + `gh` 现查）。 |
| **船长不亲自扛所有缆绳** | 船长负责 **调度、下发指令、按情况派发任务**；重活由 **专用心跳/项目需求** 执行（水手/小艇）。 |
| **船长的分身 / 大副执行本轮值班** | 每一轮 **管家心跳生成的那条需求** 由 Agent 执行：Agent 以 **船长职权**（Prompt + Runbook + 工具）代行判断——相当于 **船长助理代班**，但叙事上仍是「本轮船长决策」。 |
| **定时响铃** | **定时心跳** = 闹钟/助理提醒「该巡船了」→ 触发一次 **管家需求** → 由「船长视角」的 Agent 决定本轮具体做什么（仍受 §1.4 单轮范围约束）。 |

#### 状态机：一类事的「套路 / 航路图」

| 比喻 | 实现 |
|------|------|
| **一套套航路图** | **一类对象或一类需求类型绑定一套状态机**（如 Issue 元需求一条线、PR 元需求一条线、普通开发需求一条线）。 |
| **每票货按自己的图走** | 每个 **任务（Requirement）** 在创建时初始化对应状态机实例；执行过程按 **AIGuide + 显式/确定性迁移** 推进（§6）。 |
| **船长不背下整张海图的所有细节** | 船长 **调度**「哪条任务、是否触发子心跳」；**单条任务** 的细节推进由 **该任务的状态机** 约束。 |

#### 心跳、Webhook 与「谁决定出发」

| 触发方式 | 谁决定「何时动」 | 典型行为 |
|----------|------------------|----------|
| **船长主动** | **人 / 策略 / 管理台** 手动触发某心跳，或管家在轮值内 **TriggerHeartbeatWithParams** 创建子任务 | 灵活，用于补位与插队。 |
| **定时心跳** | **时钟固定**；时刻不由船长改（改的是 `interval` 配置） | 到点生成需求 → 往往对应 **管家一轮** 或某专用心跳；像 **助理定时请船长出来巡视**。 |
| **Webhook** | **外部系统（GitHub）固定**；事件类型与仓库配置绑定 | **规则化**：到达即迁移状态或触发绑定心跳；**不是**船长「想什么时候就什么时候」。 |

**小结**：**心跳**（定时或手动）与 **Webhook** 都是「产生需求 / 推进状态」的入口；**管家**是 **编排大脑**，其值班可由 **定时心跳** 规律唤醒，也可由 **Webhook** 与 **人工** 插单；**每条需求** 仍挂在各自 **状态机** 上执行。

#### 与 §1.4「小任务」的关系

- **船长一轮**、**专用心跳一条**：仍应是 **短、可验收** 的任务单元。  
- **复杂** 来自 **多轮值班 + 多条需求 + 多条状态机实例**，而非单条 Prompt 无限膨胀。

#### 关系总图（Mermaid）

**触发 → 需求 → 状态机 → 外部世界**

```mermaid
flowchart TB
    subgraph triggers["触发源（谁摇铃）"]
        T[定时调度器]
        M[人工 / API 手动触发心跳]
        W[GitHub Webhook]
    end

    subgraph entries["入口统一：心跳触发服务"]
        H[HeartbeatTriggerService]
    end

    subgraph work["任务单元 Requirement"]
        B[管家需求：本轮船长值班]
        S[专用心跳需求：小艇任务]
        O[普通项目需求：开发派发]
    end

    subgraph sm["一类事一套图"]
        SM1[状态机: Issue 元需求]
        SM2[状态机: PR 元需求]
        SM3[状态机: 开发需求…]
    end

    subgraph book["项目叙事"]
        R[Runbook 快照 + 航程条目]
    end

    T --> H
    M --> H
    W -->|规则绑定| H
    H --> B
    H --> S
    B --> R
    B -->|调度| S
    B -->|维护| SM1
    B -->|维护| SM2
    S --> SM2
    O --> SM3
```

**定时 vs 外部 vs 人的一条简图**

```mermaid
flowchart LR
    subgraph time["时间驱动"]
        CRON[interval] --> HB[心跳定义]
        HB --> REQ1[新需求]
    end

    subgraph ext["事件驱动"]
        GH[GitHub] --> WH[Webhook]
        WH -->|绑定| HB2[某心跳或仅状态迁移]
    end

    subgraph human["人驱动"]
        UI[控制台点「触发」] --> REQ2[新需求]
    end

    REQ1 --> CAP[本轮 Agent: 船长或水手角色由 Prompt 定]
    REQ2 --> CAP
    HB2 --> CAP
```

#### 术语与模块对照

| 口头说法 | 代码/数据中的落点 |
|----------|-------------------|
| 船长 | 管家类心跳配置 + 其派发的 **Requirement**（+ Prompt 中的职权描述） |
| 航海日记 | `project_runbook_snapshots` + `project_voyage_entries`（§9.0.8） |
| 航路图 / 一类事的套路 | `state_machines` 配置 + `project_state_machines` 绑定到 `requirement_type` |
| 这一票货走到哪 | `requirement_states` + `transition_logs` |
| 定时摇铃 | `HeartbeatScheduler` + `heartbeats.interval_minutes` |
| 助理代船长值班 | 执行管家需求的那次 **Agent 会话**（同一条 Requirement 的 `trace_id` / 对话） |
| 小艇出海 | `TriggerHeartbeatWithParams` 触发的专用心跳需求 |
| 外港发报 | `webhook_event_logs` + 处理逻辑里调用状态机或 `TriggerWithSource` |

#### 数据落在何处

| 想回答的问题 | 优先查 |
|--------------|--------|
| 项目整体走到哪一「阶段叙事」 | Runbook 快照 + 航程条目 |
| 某个 Issue/PR 在系统里什么状态 | 对应元需求 + `requirement_states` |
| 某次心跳为何失败 | 该需求的 `last_error` / `agent_runtime_result` |
| GitHub 上实际怎样 | `gh` 或 Webhook 落库 payload |
| 谁何时触发了哪次心跳 | 需求标题里 `[心跳][scheduler|manual|webhook]` + `acceptance_criteria` 内 `heartbeat_id` |

---

**专用心跳（推荐起步 2～3 个，可增删）**

| 逻辑名 | requirement_type 建议 | 职责 |
|--------|------------------------|------|
| GitHub 实现 | `github_implement` | 指定 Issue：clone、开发、`gh pr create` |
| GitHub PR 修复 | `github_pr_fix` | 指定 PR：checkout 分支、按评论修改、push |
| GitHub 轻量评审（可选） | `github_pr_review_light` | 指定 PR：发「需求评审通过」前序评论等 |

---

## 5. GitHub 约定（唯一来源）

实现代码、管家 Prompt、专用心跳 Prompt **仅允许引用下表**。

| 语义 | 机器可判定形式 | 人类/Agent 操作示例 |
|------|----------------|----------------------|
| Issue 可进入「同步项目需求 / 派发开发」 | Issue 含 **label `lgtm`**（小写） | `gh issue edit N --add-label lgtm` |
| PR 需求维度评审完成 | 任意评论正文包含 **`需求评审通过`** | `gh pr comment` |
| 合并前批准类评论 | 评论正文包含 **`/lgtm`**（与 label 区分） | 与内置场景、`gh pr comment` 一致 |
| PR 与 Issue 关联 | PR body 含 `Closes #N` 或 `Fixes #N` | `gh pr create` 模板 |

**禁止**：在文档或 Prompt 中写「给 Issue 打 /lgtm 标签」——`/lgtm` 在本约定中为 **PR 评论**，Issue 侧用 **label `lgtm`**。

---

## 6. 状态迁移权威模型

### 6.1 分类

| 类别 | 触发方 | 典型 trigger 名（示例） | 说明 |
|------|--------|-------------------------|------|
| A. Webhook | `GitHubWebhookHandler` 后处理 | `pr_merged`, `pr_synchronize` | Payload 解析后调用 SM |
| B. 管家规则 | 管家执行线程内，**无 LLM** | `lgtm_label_applied` | 对 `gh issue view --json labels` 判定 |
| C. Agent 显式 | 任务结束脚本/模型调用 API | `requirement_approved`, `dispatch_started` | 必须落库，禁止仅对话声明 |
| D. 人工 | 管理台 / CLI | `unblock_to_refining` | 可选 |

### 6.2 SuccessCriteria / AIGuide 的定位

- **指导 Agent**：说明本状态要做什么。
- **不**作为自动迁移引擎；迁移须经过 §6.1 的 A/B/C/D。

---

## 7. 元需求数据模型

### 7.1 需求类型枚举

| requirement_type | 用途 |
|------------------|------|
| `github_meta_issue` | 追踪 Issue |
| `github_meta_pr` | 追踪 PR |

若希望减少枚举，可合并为 `github_meta` + 列 `meta_object_type`。

### 7.2 表结构扩展（推荐）

在 `requirements` 表增加：

| 列名 | 类型 | 说明 |
|------|------|------|
| `meta_object_type` | TEXT | `issue` \| `pr` |
| `meta_object_id` | INTEGER | GitHub 序号 |
| `meta_object_url` | TEXT | 完整 URL |
| `meta_last_scanned_at` | INTEGER | Unix 时间戳，管家扫到即更新 |
| `meta_last_action_at` | INTEGER | 上次业务动作（创建 WR、触发子心跳等） |
| `meta_last_action_type` | TEXT | 短枚举：`scan` / `create_wr` / `trigger_child` / `rule_transition` |

**索引**：`(project_id, meta_object_type, meta_object_id)` 唯一，防止重复元需求。

### 7.3 项目需求与元需求关联

`requirements` 中项目需求（可派发）增加可选列：

| 列名 | 说明 |
|------|------|
| `source_meta_requirement_id` | 来自哪条 Issue 元需求（TEXT UUID） |

便于从 Issue 元需求跳到开发需求与 PR。

### 7.4 AcceptanceCriteria 扩展块（可选）

若暂不扩表，可在 `AcceptanceCriteria` 内嵌 JSON：

```json
{
  "github_meta": {
    "object_type": "issue",
    "object_id": 42,
    "object_url": "https://github.com/org/repo/issues/42"
  }
}
```

**不推荐长期依赖**：查询与唯一约束较弱，建议落列。

---

## 8. 状态机完整配置

以下 YAML 与 `statemachine.Config` 字段一致（`states` / `transitions`）。**状态 ID 含强制项 `todo`、`completed`**。

### 8.1 `github_issue_lifecycle`（Issue 元需求）

**说明**：`todo` 表示「新 Issue 已纳入追踪尚未深度处理」；业务上也可称「待分析」。`completed` 表示 Issue 侧生命周期结束（通常 PR 已合并）。

```yaml
name: github_issue_lifecycle
description: GitHub Issue 元需求生命周期（与 taskmanager 校验器兼容）
initial_state: todo

states:
  - id: todo
    name: 新 Issue / 待分析
    is_final: false
    ai_guide: |
      阅读 Issue，必要时在 GitHub Issue 评论中提问；不要修改业务代码。
      使用：gh issue view <n> --json title,body,labels,comments
    success_criteria: |
      已有一次有意义的互动（评论或标签变更）或已记录「无需操作」
    triggers:
      - trigger: start_analysis
        description: 开始分析/完善需求
      - trigger: mark_blocked
        description: 标记阻塞

  - id: refining
    name: 需求完善中
    is_final: false
    ai_guide: |
      结合仓库上下文分析可行性；在 Issue 下发表分析；可建议添加 label。
    triggers:
      - trigger: need_more_info
        description: 需要作者补充信息
      - trigger: ready_for_lgtm
        description: 需求已清晰，等待打 lgtm 标签
      - trigger: mark_blocked
        description: 阻塞

  - id: waiting_info
    name: 等待信息
    is_final: false
    ai_guide: |
      等待 Issue 作者回复；可定期 gh issue view 检查新评论。
    triggers:
      - trigger: info_received
        description: 已收到有效回复
      - trigger: timeout_human
        description: 超时转人工

  - id: waiting_lgtm
    name: 等待 label lgtm
    is_final: false
    ai_guide: |
      等待维护者给 Issue 打上 **label lgtm**（小写）。不要与 PR 评论 /lgtm 混淆。
    triggers:
      - trigger: lgtm_label_applied
        description: 已检测到 lgtm 标签（确定性）
      - trigger: scope_changed
        description: 需求范围变化，退回完善

  - id: sync_project_requirement
    name: 待同步项目需求
    is_final: false
    ai_guide: |
      已具备 lgtm：在系统内创建可派发的项目需求，并填写 source_meta_requirement_id。
    triggers:
      - trigger: project_requirement_created
        description: 项目需求已创建

  - id: waiting_dispatch
    name: 等待派发
    is_final: false
    ai_guide: |
      等待 dispatch 或触发 github_implement 专用心跳。
    triggers:
      - trigger: dispatch_started
        description: 已派发或已触发实现类心跳

  - id: implementing
    name: 开发中
    is_final: false
    ai_guide: |
      由专用心跳/Agent 在独立 workspace 执行；关注是否已开 PR 并关联 Issue。
    triggers:
      - trigger: pr_opened
        description: 已有关联 PR

  - id: linked_pr
    name: 已关联 PR
    is_final: false
    ai_guide: |
      后续进展以 PR 与 Webhook 为准；可轮询 gh pr list 关联 Issue。
    triggers:
      - trigger: pr_merged
        description: PR 已合并（确定性，推荐 Webhook）

  - id: blocked
    name: 阻塞
    is_final: false
    ai_guide: |
      记录原因；需要人工或解除条件后再迁移。
    triggers:
      - trigger: unblock_to_refining
        description: 回到完善
      - trigger: unblock_to_lgtm
        description: 回到等待 lgtm

  - id: completed
    name: 已完成
    is_final: true

transitions:
  - { from: todo, to: refining, trigger: start_analysis }
  - { from: todo, to: blocked, trigger: mark_blocked }

  - { from: refining, to: waiting_info, trigger: need_more_info }
  - { from: refining, to: waiting_lgtm, trigger: ready_for_lgtm }
  - { from: refining, to: blocked, trigger: mark_blocked }

  - { from: waiting_info, to: refining, trigger: info_received }
  - { from: waiting_info, to: blocked, trigger: timeout_human }

  - { from: waiting_lgtm, to: sync_project_requirement, trigger: lgtm_label_applied }
  - { from: waiting_lgtm, to: refining, trigger: scope_changed }

  - { from: sync_project_requirement, to: waiting_dispatch, trigger: project_requirement_created }

  - { from: waiting_dispatch, to: implementing, trigger: dispatch_started }

  - { from: implementing, to: linked_pr, trigger: pr_opened }

  - { from: linked_pr, to: completed, trigger: pr_merged }

  - { from: blocked, to: refining, trigger: unblock_to_refining }
  - { from: blocked, to: waiting_lgtm, trigger: unblock_to_lgtm }
```

**Hook 示例（可选）**：在 `lgtm_label_applied` **之后**可为 `sync_project_requirement` 配置 `trigger_heartbeat` 调用 `github_implement` 的 heartbeat_id（若策略为「一打 lgtm 自动开干」）。

### 8.2 `github_pr_lifecycle`（PR 元需求）

**说明**：从 `todo`（新 PR）到 `completed`；`closed` 用于关闭未合并。**每个 `(from, trigger)` 全局唯一。**

```yaml
name: github_pr_lifecycle
description: GitHub PR 元需求生命周期
initial_state: todo

states:
  - id: todo
    name: 新 PR
    is_final: false
    ai_guide: |
      gh pr view；判断是否需要需求评审。
    triggers:
      - trigger: begin_requirement_review
        description: 进入需求评审

  - id: requirement_reviewing
    name: 需求评审中
    is_final: false
    ai_guide: |
      关联 Issue；评论中写入结论；若通过须包含「需求评审通过」。
    triggers:
      - trigger: requirement_approved
        description: 需求评审通过（含关键字）
      - trigger: requirement_revision_requested
        description: 需求不清，需大改

  - id: code_reviewing
    name: 代码评审中
    is_final: false
    ai_guide: |
      gh pr diff；行评或总评；指出必须修改点。
    triggers:
      - trigger: code_review_requests_changes
        description: 代码需改
      - trigger: code_review_approved
        description: 代码评审通过，进入合并前检查

  - id: waiting_changes
    name: 等待作者修改
    is_final: false
    ai_guide: |
      等待新 commit；可触发 github_pr_fix。
    triggers:
      - trigger: changes_pushed
        description: 有新 push
      - trigger: assign_fix
        description: 指派专用心跳修复
      - trigger: give_up
        description: 放弃并关闭（少见）

  - id: fixing
    name: 修复中（可由专用心跳执行）
    is_final: false
    ai_guide: |
      在 PR 分支上修改并 push。
    triggers:
      - trigger: fix_pushed
        description: 修复已推送，回到代码评审

  - id: checking_merge_ready
    name: 合并前检查
    is_final: false
    ai_guide: |
      gh pr checks；无未解决评论；必要时发 /lgtm 评论（见 §5）。
    triggers:
      - trigger: merge_ready
        description: 检查通过
      - trigger: merge_blocked
        description: 仍有问题，退回等待修改

  - id: adding_doc
    name: 补充文档（可选阶段）
    is_final: false
    ai_guide: |
      按需补 README/changelog。
    triggers:
      - trigger: doc_done
        description: 文档处理完

  - id: adding_test
    name: 补充测试（可选阶段）
    is_final: false
    ai_guide: |
      按需补测试并跑通。
    triggers:
      - trigger: test_done
        description: 测试处理完

  - id: waiting_merge
    name: 等待合并
    is_final: false
    ai_guide: |
      由维护者合并；或策略允许 bot merge（默认关闭）。
    triggers:
      - trigger: merged
        description: 已合并

  - id: completed
    name: 已完成
    is_final: true

  - id: closed
    name: 已关闭（未合并）
    is_final: true

transitions:
  - { from: todo, to: requirement_reviewing, trigger: begin_requirement_review }

  - { from: requirement_reviewing, to: code_reviewing, trigger: requirement_approved }
  - { from: requirement_reviewing, to: waiting_changes, trigger: requirement_revision_requested }

  - { from: code_reviewing, to: waiting_changes, trigger: code_review_requests_changes }
  - { from: code_reviewing, to: checking_merge_ready, trigger: code_review_approved }

  - { from: waiting_changes, to: fixing, trigger: assign_fix }
  - { from: waiting_changes, to: code_reviewing, trigger: changes_pushed }
  - { from: waiting_changes, to: closed, trigger: give_up }

  - { from: fixing, to: code_reviewing, trigger: fix_pushed }

  - { from: checking_merge_ready, to: adding_doc, trigger: merge_ready }
  - { from: checking_merge_ready, to: waiting_changes, trigger: merge_blocked }

  - { from: adding_doc, to: adding_test, trigger: doc_done }

  - { from: adding_test, to: waiting_merge, trigger: test_done }

  - { from: waiting_merge, to: completed, trigger: merged }

  - { from: todo, to: closed, trigger: pr_closed }
  - { from: requirement_reviewing, to: closed, trigger: pr_closed }
  - { from: code_reviewing, to: closed, trigger: pr_closed }
  - { from: waiting_changes, to: closed, trigger: pr_closed }
  - { from: fixing, to: closed, trigger: pr_closed }
  - { from: checking_merge_ready, to: closed, trigger: pr_closed }
  - { from: adding_doc, to: closed, trigger: pr_closed }
  - { from: adding_test, to: closed, trigger: pr_closed }
  - { from: waiting_merge, to: closed, trigger: pr_closed }
```

**可选简化**：若团队不需要独立「文档/测试」阶段，可将 `checking_merge_ready` 直接 `merge_ready` → `waiting_merge`，删除 `adding_doc` / `adding_test` 两状态。

---

## 9. 管家心跳

单轮工作量与「一次只做少量事」的约束见 **[§1.4 任务粒度原则](#14-任务粒度原则心跳与需求要足够简单)**。跨轮 **记忆与目标** 由本节 **§9.0 Runbook** 承担。

### 9.0 管家运行上下文（Runbook）

#### 9.0.0 隐喻：航海日记（仅管家使用）

可以把 **Runbook** 理解成 **船长手里的航海日记**——**不是**某一次「放小艇干活」的任务单，而是 **整条船在时间长轴上** 的记录与计划：

| 角色 | 对应 | 日记里记什么 |
|------|------|----------------|
| **船长** | **管家心跳**（编排与决策） | 我们**要去哪**（阶段目标 `phase_goal`）、**现在在朝哪个方向走**（`current_focus`）、**上一段航程发生了什么**（`last_run_summary` / outcome） |
| **海图与灯塔** | GitHub（`gh`）、元需求状态机 | 客观事实：港口在哪、灯是否亮——**不以日记替代海图** |
| **单次放艇** | 专用心跳派生的小需求 | 只解决**这一趟**的具体活；**不**承担「整条航线叙事」 |

因此 Runbook 应是 **相对独立、专门服务于管家** 的一层数据：  
**独立**于每一条瞬时派发的 `Requirement`（那些像「单次航海任务报告」）；**独立**于 Issue/PR 元需求上的状态机（那是「每票货物的在途状态」）。没有这本日记，船长每轮上任都像第一次出海——只能现场看海图（`gh`），不知道**本船**上周决定先清哪条航线、进行到哪一步。

实现上仍用 §9.0.3 的字段与 §9.0.4～9.0.5 的注入/回写；本小节只固定 **产品隐喻与边界**，避免与「小任务需求」混淆。

#### 9.0.1 为什么必须加

当前实现路径（见 `HeartbeatTriggerService.TriggerWithSource`）是：**每次触发都新建一条 `Requirement`**，`description` 主要来自 `Heartbeat.RenderPrompt(project)`。因此若不额外注入：

- Agent **看不到**上一轮管家任务写了什么、判定结果如何；
- **「现在的任务」**容易与「GitHub 上已发生的事实」脱节，只能靠当次 `gh` 现查，**没有**项目侧的持续叙事（例如：我们阶段目标是在本迭代清掉带 `blocked` 的元需求）。

**结论**：需要在平台侧为管家维护一份 **持久化、结构化、体积可控** 的上下文，下称 **Runbook**。  
元需求状态机、Webhook、`gh` 提供 **事实**；Runbook 提供 **编排叙事与上轮结论**，三者一起支撑「持续运转」。

#### 9.0.2 与其它信息源的分工

| 信息源 | 提供什么 | 是否每轮现查 |
|--------|----------|--------------|
| `gh issue list` / `gh pr list` | Issue/PR **当前**开放项与标签 | 是（外部真相） |
| 元需求 + `requirement_states` / `transition_logs` | 每条 Issue/PR 在系统内的状态 | 可按需查询 |
| **Runbook** | 管家 **跨轮** 的摘要：上轮做了什么、结果、本轮意图、阶段目标 | 否，**读库注入** |
| 上一轮管家 `Requirement` 全文 | 历史细节 | 可选：只保留 **链接 + 一行摘要**，避免把长 Prompt 再灌一遍 |

#### 9.0.3 Runbook 建议字段（持久化）

存放粒度：**按 `project_id` 一条**；若同一项目有多个「管家类」心跳，可增加 **`heartbeat_id`** 联合主键。

| 字段 | 类型 | 含义 |
|------|------|------|
| `phase_goal` | TEXT | 当前阶段目标（一句话，人工或 Agent 定期更新） |
| `last_run_at` | INTEGER | 上次管家心跳触发时间 |
| `last_run_requirement_id` | TEXT | 上一轮产生的管家需求 ID（便于跳转审计） |
| `last_run_summary` | TEXT | 上轮执行摘要：处理了哪些 Issue/PR、成功/失败、是否已触发子心跳 |
| `last_run_outcome` | TEXT | 枚举建议：`success` / `partial` / `failed` / `noop` |
| `current_focus` | TEXT | 本轮建议焦点（例如：优先 Issue #42、或「无，维持扫尾」） |
| `next_planned` | TEXT(JSON) | 可选：队列下一批编号列表，供 Prompt 直接展示 |
| `rolling_notes` | TEXT | 可选：短滚动笔记（限制长度，例如 ≤2KB），超时由实现截断或摘要 |

实现可选：**扩 `projects` 一列 `automation_runbook_json`**，或独立表 `project_automation_runbook(project_id[, heartbeat_id], payload_json, updated_at)`。

#### 9.0.4 注入点（与现有代码对齐）

在 **`HeartbeatTriggerService`** 内，当 `heartbeat_id` 为管家（或 `requirement_type` 为管家类型）时，在 `RenderPrompt` 之后、`NewRequirement` 之前：

1. 加载 Runbook 记录；
2. 可选：查询「上一轮同心跳需求」的 `agent_runtime_result` 一行摘要（若已实现）；
3. 将下列块 **追加** 到本次需求的 `description`（或独立字段若后续扩展）：

```markdown
==================== 管家运行上下文（Runbook）====================
阶段目标：${runbook.phase_goal}
上次执行：${runbook.last_run_at} → 结果：${runbook.last_run_outcome}
上次摘要：${runbook.last_run_summary}
本轮建议焦点：${runbook.current_focus}
待下轮处理（若有）：${runbook.next_planned}
================================================================
```

变量名以实现为准；**禁止**把完整历史对话贴入。

#### 9.0.5 回写协议（每轮结束必须更新 Runbook）

否则下一轮仍为空。任选一种或组合：

| 方式 | 做法 |
|------|------|
| **A. Agent 结构化收尾** | 要求管家 Prompt 末尾输出固定 JSON 块（如 `<!-- RUNBOOK_UPDATE {...} -->`），派发通道或后处理任务解析并 `PATCH` Runbook |
| **B. 显式 API** | `PUT /api/v1/projects/:id/automation-runbook`，由 Agent 调用或人工修正 |
| **C. 服务端摘要** | 管家需求进入 `completed` 时，用 `agent_runtime_result` 截断写入 `last_run_summary`（质量依赖运行时） |

推荐 **A+B**：机器可解析 + 人可纠偏。

#### 9.0.6 与「小任务」原则的关系

- **Runbook** 解决 **连续性**（之前 / 现在 / 目标）；
- **§1.4** 仍约束 **单轮执行动作** 要少、要可验收；
- 子需求（专用心跳）继续是 **独立小任务**；管家是 **调度大脑**，Runbook 是它的 **记事本**，不是把大活塞进一条需求里。

#### 9.0.7 可选：追加审计表

`butler_run_log(id, project_id, heartbeat_id, requirement_id, summary, outcome, created_at)` 仅追加、不删，供排障与回放；Runbook 存「最新快照」，日志存「全历史」。

#### 9.0.8 项目航海日记：推荐数据结构（伴随项目一生）

**原则**：**一个项目一本日记**；日记由两部分组成——**当前页（快照）** 给管家每轮读入 Prompt；**按时间追加的条目** 记录从立项到归档的完整过程，可审计、可回放，不把全历史塞进单次 Prompt。

##### （1）两层模型

| 层 | 作用 | 读写特点 |
|----|------|----------|
| **快照 `ProjectRunbookSnapshot`** | 等价于「船长桌上翻开的那一页」：**现在要去哪、上一段结果、本轮焦点** | **读**：每轮管家触发前加载并注入；**写**：每轮结束或人工/API 更新 |
| **航程条目 `ProjectVoyageEntry`（追加日志）** | 等价于「日记本里从前到后每一篇记录」：里程碑、每一轮管家、阶段变更、重要人工批注 | **只追加**（append-only），原则上不删改；全生命周期 |

二者关系：每轮管家结束时，**先写一条 `ProjectVoyageEntry`**，再 **合并更新 `ProjectRunbookSnapshot`**，保证快照与最后一条条目一致（或快照由最后 N 条条目派生，实现二选一）。

##### （2）表 A：`project_runbook_snapshots`（每项目 0 或 1 行，推荐独立表便于迁移）

| 列名 | 类型 | 说明 |
|------|------|------|
| `project_id` | TEXT PK | 与 `projects.id` 1:1 |
| `updated_at` | INTEGER | 快照最后更新时间 |
| `version` | INTEGER | 乐观锁/迁移用，可选 |
| `phase_goal` | TEXT | 当前阶段目标（可人工改） |
| `current_focus` | TEXT | 本轮/近期焦点 |
| `last_run_at` | INTEGER | 上次管家触发时间 |
| `last_run_requirement_id` | TEXT | 上次管家需求 ID |
| `last_run_outcome` | TEXT | `success` / `partial` / `failed` / `noop` |
| `last_run_summary` | TEXT | 短摘要，供注入 |
| `next_planned` | TEXT | JSON 数组字符串，可选 |
| `rolling_notes` | TEXT | 限长滚动备注，可选 |
| `snapshot_json` | TEXT | **可选**：整快照冗余 JSON，便于扩展字段而不改表 |

项目创建时可 **惰性插入**（第一次管家跑之前）或 **建项目时插入空快照**。

##### （3）表 B：`project_voyage_entries`（航海日记正文，多条，伴随项目一生）

| 列名 | 类型 | 说明 |
|------|------|------|
| `id` | TEXT PK | ULID/UUID |
| `project_id` | TEXT FK | 索引 |
| `seq` | INTEGER | **项目内单调递增**（由触发器或应用层 `MAX(seq)+1`），保证顺序 |
| `occurred_at` | INTEGER | 事件发生时间 |
| `entry_type` | TEXT | 见下枚举 |
| `title` | TEXT | 一行标题，便于列表展示 |
| `summary` | TEXT | 短摘要（列表用） |
| `detail_json` | TEXT | 可选：结构化详情（关联 issue 号、子心跳 ID、迁移 trigger 等） |
| `heartbeat_id` | TEXT | 可选：哪条管家心跳产生 |
| `requirement_id` | TEXT | 可选：本轮管家需求 ID |
| `actor` | TEXT | `agent` / `system` / `user` / `webhook` |
| `created_at` | INTEGER | 写入时间 |

**`entry_type` 建议枚举**：

| 值 | 含义 |
|----|------|
| `project_started` | 项目创建或首次启用自动化（可选一条「启航」） |
| `phase_changed` | 阶段目标变更（人工或 Agent 更新 `phase_goal`） |
| `heartbeat_round` | 每一轮管家执行结束（**主条目**，与快照一一对应） |
| `child_heartbeat` | 触发了某专用心跳（记参数与结果摘要） |
| `webhook_milestone` | Webhook 驱动的里程碑（如 PR merged）可选记入，便于叙事连续 |
| `manual_note` | 人工在控制台写的批注 |
| `milestone` | 自由里程碑（如「v1 发布」） |

**索引**：`(project_id, seq)`、`(project_id, occurred_at DESC)`。

##### （4）与现有表的关系

- **`requirements`**：仍是「单次任务单」；管家每轮一条；**不**用需求行替代日记。
- **`transition_logs`**：仍是单条需求状态机迁移；**航海日记**是 **项目级** 叙事，二者可互链：`detail_json` 里可写 `requirement_id` / `transition_id`。
- **元需求**：管「每票 Issue/PR」；**航海日记**管「整条船」——层级不同。

##### （5）API 形态（建议）

| 操作 | 说明 |
|------|------|
| `GET /api/v1/projects/:id/runbook` | 返回当前快照（给前端「航海日记」首页卡片 + 给触发器注入） |
| `GET /api/v1/projects/:id/voyage-entries?cursor=&limit=` | 分页读日记条目（时间线 UI） |
| `POST /api/v1/projects/:id/voyage-entries` | 人工补记（`manual_note`） |
| `PUT /api/v1/projects/:id/runbook` | 更新快照（人工改 `phase_goal` 等）；可选同时 `INSERT` 一条 `phase_changed` 条目 |

管家每轮结束：服务端或解析 Agent 输出后 **upsert 快照 + insert voyage entry**。

##### （6）注入 Prompt 时

仍只用 **快照表** 拼 §9.0.4 的 Markdown 块；**不把** `project_voyage_entries` 全表拼进 Prompt。若需「最近脉络」，可 **最多取最近 K 条条目的 title+summary**（如 K=5）附加在快照块下方，并设总字数上限。

---

### 9.1 配置项

| 字段 | 建议值 |
|------|--------|
| 名称 | GitHub 项目管家 |
| 场景内编码 | `github_project_manager` |
| `interval_minutes` | 30（可配置） |
| `requirement_type` | `github_meta_manager`（新建枚举）或复用 `github_meta` |
| `agent_code` | 项目默认 Agent 或专用「调度型」Agent |

### 9.2 单轮算法（伪代码）

```
1. 加载 project（git URL、dispatch 渠道）
2. gh issue list --state open --json number,title,labels,updatedAt
3. gh pr list --state open --json number,title,updatedAt,headRefName
4. 对每条 Issue/PR：
   a. upsert 元需求（唯一索引）
   b. 若无状态机实例则 InitializeRequirementState
   c. 运行规则引擎 apply_issue_rules / apply_pr_rules（无 LLM）
5. 从「待处理集合」按优先级选 1～2 条：
   priority: blocked > 超时 > 24h 内有活动 > FIFO
6. 对选中项：
   a. 若需编码/大修 → TriggerHeartbeatWithParams
   b. 否则在 Prompt 指导下发评论 / 仅记录系统内执行摘要
7. 更新 meta_last_scanned_at / meta_last_action_*
```

### 9.3 Prompt 骨架（节选）

```markdown
你是 GitHub 项目管家。必须遵守约定：Issue 可开发条件 = **label lgtm**；PR 需求通过 = 评论含 **需求评审通过**；批准类评论用 **/lgtm**。

项目：${project.name}
仓库：${project.git_repo_url}

本回合步骤：
1. 执行 gh issue list / gh pr list（JSON），不得超过 50 条/类。
2. 对缺少元需求的 open 项：说明将调用 API 创建元需求（若你已接入 taskmanager CLI 则执行创建命令）。
3. 对已存在元需求：读取当前状态机状态（从 taskmanager 查询），不要臆造。
4. 仅对 1～2 个最高优先级事项采取行动；其余写入「下回合计划」到系统内执行摘要。
5. 禁止：merge PR、关闭 Issue，除非配置显式开启。
6. 需要专用心跳时：仅输出 heartbeat_id 与 issue_number/pr_number，由系统触发（若当前环境不支持，则记录待办）。

执行摘要请写在任务汇报中，GitHub 侧仅发必要评论。
```

---

## 10. 专用心跳

每个专用心跳对应的需求仍须满足 **[§1.4](#14-任务粒度原则心跳与需求要足够简单)**：目标单一、可验收、时间有界；若 Issue 过大，先在 GitHub 拆 Issue 再分别派发。

### 10.1 `github_implement`（示例）

**参数**：`issue_number`（必填），`base_branch`（可选）

**Prompt 要点**：`gh repo clone` / 现有 workspace 策略、`gh pr create`、PR body 含 `Closes #n`、跑测试与 lint。

**结束**：Agent 显式调用迁移 `pr_opened`（若已开 PR）或回报失败原因。

### 10.2 `github_pr_fix`

**参数**：`pr_number`

**Prompt 要点**：`gh pr checkout`、按 review 评论修改、push、在 PR 下简短回复。

### 10.3 定时策略

- 默认 **`interval_minutes` 极大（如 24h）或禁用调度**，仅 `manual` / `trigger_heartbeat` / Webhook 唤醒。

---

## 11. 触发与参数传递

### 11.1 `TriggerHeartbeatWithParams`

```go
// 语义示意
func (s *HeartbeatTriggerService) TriggerWithSourceAndParams(
    ctx context.Context,
    heartbeatID string,
    source HeartbeatTriggerSource,
    params map[string]interface{}, // issue_number, pr_number, ...
) (*domain.Requirement, error)
```

**持久化**：新建心跳任务需求时，将 `params` JSON 写入 `AcceptanceCriteria` 前缀块或独立列 `heartbeat_params_json`；`Heartbeat.RenderPrompt` 合并 `project` 与 `params` 模板变量。

### 11.2 状态机 Hook（`transition_executor`）

`Config` 键名：**`heartbeat_id`**（见 `executeTriggerHeartbeat`）。

可选扩展（需实现）：`heartbeat_params` map 合并进 Hook 上下文并写入触发链路。

---

## 12. Webhook 设计

### 12.1 订阅事件（建议）

| GitHub 事件 | 用途 |
|-------------|------|
| `pull_request` closed + merged=true | `merged` 迁移 PR 元需求；联动 Issue 元需求 `pr_merged` |
| `pull_request` synchronize | 可选：从 `waiting_changes` 触发 `changes_pushed` |
| `issues` labeled | 检测 `lgtm` → `lgtm_label_applied` |
| `issue_comment` / `pull_request_review` | 可选：检测「需求评审通过」关键字（注意限流与防抖） |

### 12.2 处理管线

1. 验签、解析 `repo`、PR/Issue 号。
2. 查找 `meta_object_id` 对应元需求（按项目 ID + 类型 + 序号）。
3. 调用 `StateMachineService.TriggerTransition` 使用 **§8 中精确 trigger 名**。
4. 失败重试策略：写 `webhook_event_logs`，支持人工重放。

---

## 13. 后端接口与模块

### 13.1 新增/扩展服务

| 模块 | 职责 |
|------|------|
| `MetaRequirementService` | `EnsureMetaForIssue`、`EnsureMetaForPR`、`GetByObject`、`ListByProject` |
| `GitHubRuleEngine`（包内） | `IssueHasLabel(n,"lgtm")`、`PROpenForIssue` 等纯函数 + `gh` 调用 |
| `HeartbeatTriggerService` | `TriggerWithSourceAndParams`；修复运行记录查询条件（见 §14） |

### 13.2 HTTP（示例）

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/heartbeats/:id/trigger` | body 可带 `params` |
| `GET` | `/api/v1/projects/:pid/meta-requirements` | 元需求列表 |
| `POST` | `/api/v1/projects/:pid/meta-requirements/sync` | 可选：手动全量对齐 GitHub |

（具体路径以 `router.go` 为准。）

---

## 14. 前端与可观测性

### 14.1 自动化中心 Tab

| Tab | 内容 |
|-----|------|
| 总览 | 保留；增加「元需求数量、阻塞数」卡片 |
| 元需求看板 | 表：类型、#号、标题、状态机状态、最后扫描、GitHub 链接 |
| 心跳实例 | 列表展示管家 + 专用心跳 |
| Webhook / 状态机 | 现有 |

### 14.2 元需求详情页

- 状态迁移时间线（读 transition log）
- 关联项目需求、PR 链接
- 关联心跳 Run（按 `heartbeat_id` / 标题 `[心跳]` 查询）

### 14.3 运行记录修复要点

- **不得**仅用 `requirement_type == heartbeat` 过滤。
- 条件：`title LIKE '[心跳]%'` **或** `acceptance_criteria LIKE '%heartbeat_id: <id>%'` **或** 显式 `origin_heartbeat` 字段（若后续添加）。

---

## 15. 错误处理与幂等

| 场景 | 策略 |
|------|------|
| 重复创建元需求 | DB 唯一索引 `(project_id, meta_object_type, meta_object_id)` 冲突则返回已有 |
| Webhook 重复投递 | 事件 ID 去重（已有 `webhook_event_logs` 可扩展） |
| `gh` 失败 | 管家记录 last_error，下轮重试；指数退避可选 |
| 专用心跳并发同一 PR | 由管家优先级串行或锁（项目+pr 维度） |

---

## 16. 实施阶段与任务拆解

### Phase P0 — 基础可跑

- [ ] DB migration：`meta_*` 列与唯一索引
- [ ] `MetaRequirementService` + 基础 API
- [ ] 状态机 YAML：`github_issue_lifecycle`、`github_pr_lifecycle` 入库并绑定项目需求类型
- [ ] `TriggerWithSourceAndParams` + Prompt 变量渲染
- [ ] 管家心跳 v1（Prompt + 规则引擎骨架）
- [ ] **航海日记**：`project_runbook_snapshots` + `project_voyage_entries`（§9.0.8）、触发时仅注入快照（§9.0.4）、每轮结束 upsert 快照并 append 条目、API（§9.0.8 第五节）
- [ ] 运行记录查询修正（§14.3）

### Phase P1 — 闭环加强

- [ ] Webhook：`pull_request` merged、`issues` labeled
- [ ] 专用心跳 `github_implement`、`github_pr_fix` 模板与场景注册
- [ ] `source_meta_requirement_id` 与从 Issue 创建项目需求流程

### Phase P2 — 体验与指标

- [ ] 元需求看板与详情页
- [ ] 可选：阻塞/耗时统计、告警

---

## 17. 测试策略

| 层级 | 内容 |
|------|------|
| 单元 | `GitHubRuleEngine` 对 mock JSON；状态机 `Validate()` 必须通过 |
| 集成 | Webhook payload → 期望 `TriggerTransition`；元需求唯一约束 |
| E2E | 沙盒仓库：建 Issue → label → 创建 WR → 模拟 merge → 元需求 `completed` |

---

## 18. 风险

| 风险 | 缓解 |
|------|------|
| GitHub API/限流 | 单管家、分页、冷却；Webhook 优先 |
| 模型绕过状态机 | §6 权威迁移 + 关键路径仅 Webhook/规则 |
| 状态机 YAML 与校验器演进 | 单测锁定 `ParseConfig` + `Validate` |

---

## 附录 A：与 `BuildGitHubDevWorkflowScenario` 的关系

当前内置场景中的多心跳条目可被 **替换或删减**：保留「管家 + 少量专用心跳」即可；`EnsureBuiltInScenarios` 可改为注册新场景 **`github_orchestrated_workflow`**，避免与旧 8 心跳并存重复扫描（若团队仍要旧场景，可在产品层标记为 deprecated）。

---

## 附录 B：文档修订记录

| 版本 | 说明 |
|------|------|
| 详细版 | 补全状态机 YAML、时序/对象图、表结构、Webhook、阶段任务、测试；状态 ID 对齐 `todo`/`completed` 校验器；PR 转移无重复 `(from,trigger)`。 |
