
# AI 自主驱动的任务闭环设计文档

## 概述

本设计在现有状态机和心跳系统的基础上，实现 **AI 自主驱动的任务闭环**。核心思路是：

1. **基于 GitHub 进行任务管理**（Issue 作为任务来源，PR 作为产出物评审）
2. **创建 "项目管家" 心跳**（作为 AI 决策者，协调整个流程）
3. **为每个 Issue/PR 创建"元需求"**，绑定完整生命周期状态机
4. **让 AI 在状态机框架下自主决策**（利用现有 `AIGuide`、`SuccessCriteria`、`Triggers`）

**不引入新的强 Workflow 引擎**，完全利用现有架构实现 AI 自主闭环。编排复杂度集中在"管家心跳 + 元需求 + 状态机驱动"这一层，需要新写管家心跳的 Prompt 逻辑、元需求管理逻辑、状态机转换触发逻辑。

## 适用场景

这套系统支持多种任务类型：

| 场景 | Issue 内容 | 产出物 | PR 内容 |
|-----|----------|-------|---------|
| 代码开发 | "实现 XX 功能" | 代码 | 代码变更 |
| 专利撰写 | "撰写一个 XX 技术专利点" | 专利文档 | 专利章节草稿 |
| 剧本写作 | "写一个 XX 主题的剧本" | 剧本 | 剧本章节 |
| 调研报告 | "调研 XX 技术方向" | 调研报告 | 报告章节 |
| 文档编写 | "完善项目 XX 文档" | 文档 | 文档变更 |

---

## 核心流程设计

### Issue 作为需求来源的完整闭环

```
┌─────────────────────────────────────────────────────────────────┐
│          GitHub Issue 完整开发闭环流程                      │
└─────────────────────────────────────────────────────────────────┘

1. Issue 创建（一句话需求）
   │
   ▼
2. Issue 分析（AI + 人类通过评论完善需求）
   │
   ▼
3. 需求讨论与完善（多轮评论，AI 辅助分析）
   │
   ▼
4. 需求评审通过（打上 LGTM 标签）
   │
   ▼
5. 创建正式项目需求（将 Issue 转化为系统内的需求）
   │
   ▼
6. 需求派发（分配给 Agent 执行开发）
   │
   ▼
7. 开发实现 + PR 创建 + PR 评审 + 合并
   │
   ▼
8. 需求完成
```

### 关键控制点

| 步骤 | 关键动作 | 触发条件 | 说明 |
|-----|---------|---------|-----|
| **LGTM 标签** | 人工或 AI 打上 `/lgtm` 标签 | 需求已明确、完整、可开发 | **只有打上 LGTM 标签的 Issue 才能进入开发状态** |
| **创建项目需求** | 从 Issue 创建正式的项目需求 | Issue 有 LGTM 标签 + 状态机转换 | 将 GitHub Issue 转化为系统内可管理、可派发的需求 |
| **需求派发** | 将需求分配给 Agent 执行 | 项目需求已创建 + 状态机流转 | 只有系统内的项目需求才能派发给 Agent 开发 |

---

## 背景与现状分析

### 当前系统架构

| 组件 | 功能 | 实现状态 |
|-----|------|--------|
| **心跳系统** | 定时/手动/webhook 触发需求 | ✅ 完整实现 |
| **心跳场景** | 预设的心跳组合（如 GitHub 工作流） | ✅ 完整实现 |
| **状态机** | 管理需求状态流转 | ✅ 完整实现 |
| **AIGuide** | 状态内的 AI 操作指南 | ✅ 已设计 |
| **SuccessCriteria** | 成功判断标准 | ✅ 已设计 |
| **Triggers** | 状态转换触发器指南 | ✅ 已设计 |

### 当前内置 GitHub 工作流

`github_dev_workflow` 场景包含 8 个独立心跳项：

| 序号 | 心跳 | 间隔 | 功能 |
|-----|------|-----|-----|
| 1 | Issue 分析 | 180min | 分析新 Issue |
| 2 | LGTM 代码编写 | 120min | 为带 LGTM 标签的 Issue 写代码 |
| 3 | PR 需求评审 | 120min | 评审 PR 需求合理性 |
| 4 | PR 代码质量评审 | 180min | 评审 PR 代码质量 |
| 5 | PR 修改修复 | 240min | 修复 PR 中的问题 |
| 6 | PR 合并检查 | 120min | 检查 PR 是否可合并 |
| 7 | PR 文档补充 | 480min | 为 PR 补充文档 |
| 8 | PR 测试补充 | 480min | 为 PR 补充测试 |

### 存在的问题

| 问题 | 说明 |
|-----|------|
| **心跳各自为战** | 8 个心跳独立定时触发，无统一协调 |
| **状态与执行解耦** | 状态机只管理单个需求，不追踪 Issue/PR 全局流程 |
| **AI 角色单一** | AI 只是被动执行 Prompt，不做决策 |
| **重复扫描** | 每个心跳都独立扫描 GitHub，造成 API 浪费 |
| **时序不可控** | 心跳按各自定时触发，可能出现顺序错乱 |

---

## 设计原则

| 原则 | 说明 |
|-----|------|
| 充分利用现有架构 | 不引入新的强 Workflow 引擎 |
| AI 自主决策 | AI 根据状态和指南决定下一步 |
| 渐进式演进 | 分阶段实施，风险可控 |

---

## 核心设计方案

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                  GitHub 项目管家心跳（决策者）               │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  1. 扫描项目状态                                      │  │
│  │     - 获取 open Issues                               │  │
│  │     - 获取 open PRs                                  │  │
│  │     - 检查每个 Issue/PR 的元需求状态机                │  │
│  └──────────────┬────────────────────────────────────────┘  │
│                 │                                           │
│  ┌──────────────▼────────────────────────────────────────┐  │
│  │  2. AI 自主决策（基于状态机）                         │  │
│  │     - 根据 AIGuide                                    │  │
│  │     - 根据 SuccessCriteria                            │  │
│  │     - 根据 Triggers                                   │  │
│  └──────────────┬────────────────────────────────────────┘  │
│                 │                                           │
│  ┌──────────────▼────────────────────────────────────────┐  │
│  │  3. 执行动作                                           │  │
│  │     选项 A: 自己直接执行（简单任务）                   │  │
│  │     选项 B: 触发相应的专用心跳（复杂任务）             │  │
│  │     选项 C: 更新 Issue/PR 元需求的状态机               │  │
│  └────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│           Issue/PR 元需求 + 生命周期状态机                    │
│  ┌──────────────────┐    ┌──────────────────┐             │
│  │ Issue 元需求     │    │ PR 元需求        │             │
│  │  状态机         │    │  状态机          │             │
│  └──────────────────┘    └──────────────────┘             │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              8 个专用心跳（执行工具）                         │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐             │
│  │ Issue  │ │ LGTM   │ │ PR     │ │ ...    │             │
│  │ 分析   │ │ 代码   │ │ 评审   │ │        │             │
│  └────────┘ └────────┘ └────────┘ └────────┘             │
└─────────────────────────────────────────────────────────────┘
```

---

## 详细设计

### 1. Issue/PR 元需求（Meta Requirement）

#### 1.1 需求类型扩展

新增两种特殊需求类型：

| 需求类型 | 说明 |
|---------|------|
| `github_meta_issue` | GitHub Issue 元需求（追踪完整生命周期） |
| `github_meta_pr` | GitHub PR 元需求（追踪完整生命周期） |

#### 1.2 元需求的数据结构

在 `Requirement` 实体中扩展字段（或在 `AcceptanceCriteria` 中结构化存储）：

```go
// 在 Requirement 的 AcceptanceCriteria 中结构化存储元数据
type GitHubMetaRequirementData struct {
    ObjectType     string `json:"object_type"`      // "issue" 或 "pr"
    ObjectID       int    `json:"object_id"`        // GitHub issue/pr number
    ObjectURL      string `json:"object_url"`       // GitHub 链接
    LastScannedAt  int64  `json:"last_scanned_at"`  // 上次扫描时间
    LastActionAt   int64  `json:"last_action_at"`   // 上次执行动作时间
    LastActionType string `json:"last_action_type"` // 上次动作类型
}
```

### 2. Issue/PR 生命周期状态机设计

#### 2.1 GitHub Issue 生命周期状态机

```yaml
name: github_issue_lifecycle
description: GitHub Issue 完整生命周期管理（从需求来源到派发开发）

initial_state: new_issue

states:
  - id: new_issue
    name: 新 Issue
    is_final: false
    ai_guide: |
      这是一个新创建的 Issue，作为需求来源。
      
      你的任务：
      1. 仔细阅读 Issue 描述
      2. 理解用户的需求或问题
      3. 判断是否需要更多信息
      
      可用动作：
      - 如果需要更多信息：在 Issue 下评论提问
      - 如果需求明确：进行分析并添加相应标签
      
      提示：使用 gh issue view &lt;number&gt; 查看详情。
    success_criteria: |
      Issue 已经被处理过（至少有一条你的评论或已添加标签）
    failure_criteria: |
      无法理解 Issue 内容或无法访问仓库
    triggers:
      - trigger: start_requirement_refinement
        description: 开始需求完善
        condition: Issue 尚未被分析过

  - id: requirement_refining
    name: 需求完善中
    is_final: false
    ai_guide: |
      Issue 正在需求完善阶段。
      
      你的任务：
      1. 克隆仓库（如果还没有）
      2. 结合代码库深入理解 Issue
      3. 分析问题根因或需求可行性
      4. 在 Issue 下发布分析结论
      5. 添加相应标签（如 "bug", "enhancement", "help-wanted"）
      
      这个阶段是人机协作的：
      - AI 可以主动分析并提问，帮助完善需求
      - 人类可以通过评论补充需求细节
      - 通过多轮讨论，让需求逐渐清晰、完整、可执行
      
      如果需求明确且可行，可以添加 "ready-for-dev" 标签。
      如果需要更多信息，在 Issue 下提问并回到 "等待信息" 状态。
    success_criteria: |
      已在 Issue 下发布分析评论，并且已添加相关标签
    failure_criteria: |
      无法克隆仓库或分析过程出错
    triggers:
      - trigger: need_more_info
        description: 需要更多信息，进入等待状态
        condition: 已在 Issue 下提问
      - trigger: ready_for_lgtm
        description: 需求已明确，可以申请 LGTM 标签
        condition: 已添加 "ready-for-dev" 标签，并且需求已充分讨论

  - id: waiting_for_info
    name: 等待信息
    is_final: false
    ai_guide: |
      正在等待 Issue 作者提供更多信息...
      
      定期检查：
      - Issue 是否有新评论
      - 作者是否回答了问题
      
      如果有足够信息，可以回到需求完善状态。
    success_criteria: |
      Issue 作者已回复并提供了足够信息
    triggers:
      - trigger: info_received
        description: 收到所需信息
        condition: Issue 有新的相关评论
      - trigger: timeout_ask_human
        description: 超时，提醒人工介入
        condition: 等待超过 7 天无回复

  - id: waiting_for_lgtm
    name: 等待 LGTM
    is_final: false
    ai_guide: |
      需求已基本明确，正在等待 LGTM（Looks Good To Me）标签。
      
      这个阶段的关键点：
      - LGTM 标签表示需求已通过评审，可以开始开发
      - 只有打上 LGTM 标签后，才能进入下一阶段
      
      定期检查：
      - Issue 是否被添加了 "LGTM" 标签
      - Issue 是否有其他重要更新
      
      可以做的动作：
      - 总结当前需求状态，评论询问是否可以打 LGTM
      - 如果需求有变化，回到需求完善阶段
    success_criteria: |
      Issue 已被标记 "LGTM" 标签
    triggers:
      - trigger: lgtm_added
        description: LGTM 标签已添加
        condition: Issue 有 "LGTM" 标签
      - trigger: go_back_refining
        description: 需求有变化，回到完善阶段
        condition: 有新的评论表明需求不完整

  - id: create_project_requirement
    name: 创建项目需求
    is_final: false
    ai_guide: |
      LGTM 标签已获得！现在需要将 GitHub Issue 转化为系统内的项目需求。
      
      这个阶段的关键动作：
      1. 将 Issue 内容同步到系统内，创建正式的项目需求
      2. 将 Issue 的元需求（Meta Requirement）与项目需求关联
      3. 设置需求类型、优先级等属性
      4. 关联状态机（使用项目的需求状态机）
      
      完成后，需求就可以进入派发流程了。
    success_criteria: |
      成功创建项目需求，并且与 Issue 元需求关联
    failure_criteria: |
      创建项目需求失败
    triggers:
      - trigger: project_requirement_created
        description: 项目需求已创建
        condition: 系统内的项目需求已创建

  - id: waiting_for_dispatch
    name: 等待派发
    is_final: false
    ai_guide: |
      项目需求已创建，正在等待派发。
      
      这个阶段：
      - 需求已在系统内管理
      - 可以被分配给 Agent
      - 可以设置优先级、指派负责人等
      
      等待触发派发的条件：
      - 有可用的 Agent
      - 项目配置了派发渠道
      - 手动或自动触发派发
    success_criteria: |
      需求已成功派发给 Agent
    triggers:
      - trigger: requirement_dispatched
        description: 需求已派发
        condition: 需求已分配给 Agent 并且开始执行

  - id: implementing
    name: 开发中
    is_final: false
    ai_guide: |
      需求已派发，Agent 正在开发...
      
      关注：
      1. Agent 是否创建了 feature 分支
      2. Agent 是否按需求实现了代码
      3. 测试是否通过
      4. 是否创建了 PR 并关联 Issue（使用 Closes #&lt;number&gt;）
      
      这个阶段的实际执行由专门的代码编写心跳完成。
    success_criteria: |
      已创建 PR 并在描述中关联了此 Issue
    failure_criteria: |
      代码实现失败或测试无法通过
    triggers:
      - trigger: pr_created
        description: PR 已创建
        condition: 已关联的 PR 存在

  - id: linked_to_pr
    name: 已关联 PR
    is_final: false
    ai_guide: |
      Issue 已关联 PR，后续主要在 PR 那边进行。
      
      关注 PR 的状态变化：
      - PR 是否被合并
      - PR 是否被关闭
      
      如果 PR 合并，此 Issue 可标记完成。
    success_criteria: |
      关联的 PR 已合并
    triggers:
      - trigger: pr_merged
        description: PR 已合并
        condition: 关联 PR 状态为 merged

  - id: completed
    name: 已完成
    is_final: true

  - id: blocked
    name: 阻塞
    is_final: false
    ai_guide: |
      此 Issue 遇到阻塞，需要人工介入。
      
      记录阻塞原因，并在 Issue 下评论说明情况。
    triggers:
      - trigger: unblock
        description: 解除阻塞
        condition: 问题已解决

transitions:
  - from: new_issue
    to: requirement_refining
    trigger: start_requirement_refinement
    
  - from: requirement_refining
    to: waiting_for_info
    trigger: need_more_info
    
  - from: requirement_refining
    to: waiting_for_lgtm
    trigger: ready_for_lgtm
    
  - from: waiting_for_info
    to: requirement_refining
    trigger: info_received
    
  - from: waiting_for_info
    to: blocked
    trigger: timeout_ask_human
    
  - from: waiting_for_lgtm
    to: create_project_requirement
    trigger: lgtm_added
    
  - from: waiting_for_lgtm
    to: requirement_refining
    trigger: go_back_refining
    
  - from: create_project_requirement
    to: waiting_for_dispatch
    trigger: project_requirement_created
    
  - from: waiting_for_dispatch
    to: implementing
    trigger: requirement_dispatched
    
  - from: implementing
    to: linked_to_pr
    trigger: pr_created
    
  - from: linked_to_pr
    to: completed
    trigger: pr_merged
    
  - from: blocked
    to: requirement_refining
    trigger: unblock
    
  - from: blocked
    to: waiting_for_lgtm
    trigger: unblock
```

#### 2.2 GitHub PR 生命周期状态机

```yaml
name: github_pr_lifecycle
description: GitHub PR 完整生命周期管理

initial_state: new_pr

states:
  - id: new_pr
    name: 新 PR
    is_final: false
    ai_guide: |
      这是一个新创建的 PR。
      
      你的任务：
      1. 查看 PR 描述和变更
      2. 判断是否需要进行需求评审
    success_criteria: |
      PR 已被初步查看
    triggers:
      - trigger: start_requirement_review
        description: 开始需求评审
        condition: PR 需要需求评审

  - id: requirement_reviewing
    name: 需求评审中
    is_final: false
    ai_guide: |
      正在进行需求评审...
      
      你的任务：
      1. 查看 PR 描述
      2. 如果 PR 有关联 Issue，查看 Issue 内容
      3. 判断需求是否合理、明确
      4. 在 PR 下发布需求评审评论
      5. 如果通过，评论中包含"需求评审通过"字样
    success_criteria: |
      已发布需求评审评论
    triggers:
      - trigger: requirement_approved
        description: 需求评审通过
        condition: 评论中包含"需求评审通过"
      - trigger: need_revision
        description: 需要修改
        condition: 需求不明确或不合理

  - id: code_reviewing
    name: 代码评审中
    is_final: false
    ai_guide: |
      正在进行代码质量评审...
      
      你的任务：
      1. 查看 PR 完整变更
      2. 从代码质量、潜在 bug、安全漏洞、性能问题等角度评审
      3. 在 PR 下发布评审意见（可以是行评论或总体评论）
      4. 如果发现问题，指明需要修改的地方
      5. 如果代码质量良好，给出正面反馈
    success_criteria: |
      已发布代码评审评论
    triggers:
      - trigger: code_review_complete
        description: 代码评审完成
        condition: 已发布评审意见

  - id: waiting_for_changes
    name: 等待修改
    is_final: false
    ai_guide: |
      等待 PR 作者根据评审意见进行修改...
      
      定期检查：
      - PR 是否有新的 commit
      - PR 是否有新的评论
    success_criteria: |
      PR 有新的 commit 或回复
    triggers:
      - trigger: changes_made
        description: 已做出修改
        condition: PR 有新的 commit

  - id: fixing
    name: 修改中
    is_final: false
    ai_guide: |
      正在根据评审意见修改 PR...
      
      你的任务：
      1. 阅读所有评审意见
      2. 判断哪些是可执行的合理建议
      3. checkout PR 分支
      4. 按建议修改代码
      5. 运行测试确保通过
      6. commit 并 push 更新
      7. 在 PR 下评论说明已修复的内容
    success_criteria: |
      已根据评审意见完成修改并 push
    failure_criteria: |
      修改失败或测试无法通过
    triggers:
      - trigger: fix_complete
        description: 修改完成
        condition: 已 push 修改

  - id: checking_merge_ready
    name: 检查合并条件
    is_final: false
    ai_guide: |
      正在检查 PR 是否满足合并条件...
      
      检查项：
      1. CI 是否通过（gh pr checks）
      2. 是否有至少一条批准评论（如 "/lgtm"）
      3. 是否有未解决的评审意见
      4. 是否有代码冲突
      
      如果所有条件满足，在 PR 下评论 "/lgtm" 表示批准合并。
    success_criteria: |
      已完成合并条件检查
    triggers:
      - trigger: merge_ready
        description: 满足合并条件
        condition: 所有检查项通过且已评论 "/lgtm"
      - trigger: not_ready
        description: 不满足合并条件
        condition: 还有未满足的条件

  - id: adding_doc
    name: 补充文档
    is_final: false
    ai_guide: |
      正在为 PR 补充文档...
      
      你的任务：
      1. 查看 PR 变更
      2. 判断是否需要更新 README、API 文档、变更日志等
      3. 如需要，在 PR 分支上补充文档
      4. commit 并 push
      5. 在 PR 下评论说明更新内容
    success_criteria: |
      文档补充完成（或判断不需要补充）
    triggers:
      - trigger: doc_complete
        description: 文档完成
        condition: 已处理文档补充

  - id: adding_test
    name: 补充测试
    is_final: false
    ai_guide: |
      正在为 PR 补充测试...
      
      你的任务：
      1. 查看 PR 变更
      2. 识别新增/修改的功能点
      3. 判断是否需要补充单元测试、集成测试
      4. 如需要，编写并补充相关测试
      5. 运行测试确保通过
      6. commit 并 push
      7. 在 PR 下评论说明补充的测试
    success_criteria: |
      测试补充完成（或判断不需要补充）
    triggers:
      - trigger: test_complete
        description: 测试完成
        condition: 已处理测试补充

  - id: waiting_for_merge
    name: 等待合并
    is_final: false
    ai_guide: |
      PR 已就绪，等待人工合并...
      
      定期检查：
      - PR 是否被合并
      - PR 是否有新的评论
    success_criteria: |
      PR 已被合并
    triggers:
      - trigger: merged
        description: PR 已合并
        condition: PR 状态为 merged

  - id: completed
    name: 已完成
    is_final: true

  - id: closed
    name: 已关闭
    is_final: true

transitions:
  - from: new_pr
    to: requirement_reviewing
    trigger: start_requirement_review
    
  - from: requirement_reviewing
    to: code_reviewing
    trigger: requirement_approved
    
  - from: requirement_reviewing
    to: waiting_for_changes
    trigger: need_revision
    
  - from: code_reviewing
    to: waiting_for_changes
    trigger: code_review_complete
    
  - from: waiting_for_changes
    to: fixing
    trigger: changes_made
    
  - from: fixing
    to: code_reviewing
    trigger: fix_complete
    
  - from: code_reviewing
    to: checking_merge_ready
    trigger: code_review_complete
    
  - from: checking_merge_ready
    to: adding_doc
    trigger: merge_ready
    
  - from: checking_merge_ready
    to: waiting_for_changes
    trigger: not_ready
    
  - from: adding_doc
    to: adding_test
    trigger: doc_complete
    
  - from: adding_test
    to: waiting_for_merge
    trigger: test_complete
    
  - from: waiting_for_merge
    to: completed
    trigger: merged
    
  - from: new_pr
    to: closed
    trigger: closed
  - from: requirement_reviewing
    to: closed
    trigger: closed
  - from: code_reviewing
    to: closed
    trigger: closed
```

### 3. GitHub 项目管家心跳设计

#### 3.1 心跳基本信息

| 属性 | 值 |
|-----|---|
| **名称** | GitHub 项目管家 |
| **编码** | `github_project_manager` |
| **间隔** | 30 分钟（可配置） |
| **需求类型** | `github_meta` |
| **Agent** | （使用项目默认 Agent 或指定） |

#### 3.2 心跳 Prompt 设计

```markdown
你是这个 GitHub 项目的智能项目经理。你的职责是：

1. 扫描项目当前状态（Issues、PRs）
2. 为每个需要追踪的 Issue/PR 创建"元需求"并绑定状态机
3. 判断哪些事项需要处理
4. 根据状态机指南自主决策下一步
5. 执行相应的动作（自己执行或触发专用心跳）

==================== 项目信息 ====================
项目名称: ${project.name}
仓库: ${project.git_repo_url}
默认分支: ${project.default_branch}

==================== 当前时间 ====================
${current_time}

==================== 执行步骤 ====================

第一步：扫描项目状态
========================
1. 获取所有 open Issues
   - 命令: gh issue list --repo ${repo_owner}/${repo_name} --state open --limit 50
   - 对每个 Issue，记录：number、title、labels、created_at、updated_at

2. 获取所有 open PRs
   - 命令: gh pr list --repo ${repo_owner}/${repo_name} --state open --limit 50
   - 对每个 PR，记录：number、title、labels、created_at、updated_at、author

3. 检查现有元需求
   - 查看当前项目下类型为 "github_meta_issue" 和 "github_meta_pr" 的需求
   - 记录每个元需求关联的 object_id、当前状态机状态

第二步：识别需要处理的事项
============================
对每个扫描到的 Issue：
  - 如果没有对应的元需求 → 创建元需求 + 初始化状态机
  - 如果有元需求 → 检查状态机状态，判断是否需要推进

对每个扫描到的 PR：
  - 如果没有对应的元需求 → 创建元需求 + 初始化状态机
  - 如果有元需求 → 检查状态机状态，判断是否需要推进

第三步：优先级排序
====================
按以下优先级选择最需要处理的事项：
  1. 阻塞的事项（状态 = blocked）
  2. 即将超时的事项
  3. 有新活动的事项（最近 24 小时有更新）
  4. 较旧的事项（先到先处理）

第四步：AI 自主决策与执行
==========================
选择优先级最高的 1-2 个事项进行处理。

对每个事项，执行以下步骤：
  1. 查看该事项元需求的当前状态机状态
  2. 阅读状态的 AIGuide（AI 操作指南）
  3. 阅读 SuccessCriteria（成功判断标准）
  4. 查看可用的 Triggers（触发器）
  5. 根据以上信息，自主决定下一步该做什么
  
可用的执行方式：
  选项 A：自己直接执行（适合简单任务）
    - 使用 gh CLI 直接操作
    - 更新元需求的状态机
    - 在元需求评论区记录执行结果
  
  选项 B：触发专用心跳（适合复杂任务）
    - 触发相应的专用心跳（如 Issue 分析心跳、代码编写心跳等）
    - 将当前 Issue/PR 的上下文传递给心跳
    - 等待心跳执行完成（或异步继续）
    - 心跳完成后更新元需求状态机
  
  选项 C：仅更新状态机（状态已自然变化）
    - 如果 Issue/PR 状态已自然变化（如标签已添加、PR 已合并等）
    - 直接触发状态机转换

第五步：记录结果
================
对本次处理的每个事项，记录：
  - 处理的事项（Issue #N 或 PR #N）
  - 当前状态机状态
  - 执行的动作
  - 执行结果（成功/失败）
  - 下一步计划

==================== 约束条件 ====================
1. 每次心跳最多处理 2 个事项（避免过载）
2. 所有 GitHub 操作使用 gh CLI
3. 不要执行破坏性操作（如合并 PR、关闭 Issue 等）
4. 如果无法判断下一步，记录下来，不要盲目执行
5. 每个事项的处理都要在元需求的评论区留下记录

==================== 开始执行 ====================
现在开始执行！
```

### 4. 专用心跳改造

将现有 8 个心跳改造为"可被调用"的执行工具：

#### 4.1 改造原则

| 原则 | 说明 |
|-----|------|
| 可接受参数 | 接收特定 Issue/PR 编号，而不是扫描所有 |
| 聚焦单个任务 | 每次只处理一个指定的 Issue/PR |
| 结果可追踪 | 执行结果被管家心跳获取 |

#### 4.2 心跳参数传递机制

通过心跳的 `mdContent` 中的变量替换传递参数：

```markdown
# 这是一个参数化的心跳 Prompt

目标 Issue: #{{issue_number}}
目标 URL: {{issue_url}}

你的任务是专门处理这个 Issue...
```

### 5. 状态机与心跳的集成

#### 5.1 状态机转换触发心跳

利用现有的 `trigger_heartbeat` hook，在状态转换时自动触发相应心跳：

```yaml
transitions:
  - from: waiting_for_dev
    to: implementing
    trigger: lgtm_added
    description: LGTM 标签已添加，开始编码
    hooks:
      - name: 触发 LGTM 代码编写心跳
        type: trigger_heartbeat
        config:
          heartbeat_name: "lgtm_code_writing"
          issue_number: "{{object_id}}"
          project_id: "{{project_id}}"
        timeout: 300
        retry: 1
```

#### 5.2 心跳执行结果反馈到状态机

心跳执行完成后，通过以下方式反馈到元需求的状态机：

1. 心跳在需求评论区记录执行结果
2. 管家心跳下次扫描时读取评论
3. 根据结果触发相应的状态机转换

---

## 数据模型变更

### 1. 数据库表变更（如需要）

#### 选项 A：在 Requirement 表中扩展字段（推荐）

```sql
-- 扩展 requirements 表，支持元需求
ALTER TABLE requirements ADD COLUMN meta_object_type TEXT;      -- "issue" 或 "pr"
ALTER TABLE requirements ADD COLUMN meta_object_id INTEGER;     -- GitHub issue/pr number
ALTER TABLE requirements ADD COLUMN meta_object_url TEXT;       -- GitHub 链接
ALTER TABLE requirements ADD COLUMN meta_last_scanned_at INTEGER;
ALTER TABLE requirements ADD COLUMN meta_last_action_at INTEGER;
ALTER TABLE requirements ADD COLUMN meta_last_action_type TEXT;
```

#### 选项 B：独立的元数据表

```sql
-- 新建 github_meta_objects 表
CREATE TABLE IF NOT EXISTS github_meta_objects (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    requirement_id TEXT NOT NULL,
    object_type TEXT NOT NULL,    -- "issue" 或 "pr"
    object_id INTEGER NOT NULL,   -- GitHub number
    object_url TEXT NOT NULL,
    state_machine_state TEXT NOT NULL,
    last_scanned_at INTEGER NOT NULL,
    last_action_at INTEGER,
    last_action_type TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (requirement_id) REFERENCES requirements(id)
);

CREATE INDEX IF NOT EXISTS idx_github_meta_project ON github_meta_objects(project_id);
CREATE INDEX IF NOT EXISTS idx_github_meta_object ON github_meta_objects(object_type, object_id);
```

### 2. 内置状态机配置

在系统中预置两个状态机：
- `github_issue_lifecycle` - Issue 生命周期
- `github_pr_lifecycle` - PR 生命周期

---

## 后端实现要点

### 1. 应用层变更

#### 1.1 HeartbeatTriggerService 增强

增强 `TriggerWithSource` 方法，支持传递参数给心跳：

```go
type TriggerHeartbeatOptions struct {
    ProjectID     string
    HeartbeatName string
    Parameters    map[string]interface{}  // 动态参数
}

func (s *HeartbeatTriggerService) TriggerHeartbeatWithParams(
    ctx context.Context,
    opts TriggerHeartbeatOptions,
) (*domain.Requirement, error)
```

#### 1.2 MetaRequirementService（新增）

新增服务管理 Issue/PR 元需求：

```go
type MetaRequirementService struct {
    requirementRepo domain.RequirementRepository
    projectRepo     domain.ProjectRepository
    stateMachineSvc *StateMachineService
    idGenerator     domain.IDGenerator
}

func (s *MetaRequirementService) CreateMetaRequirementForIssue(
    ctx context.Context,
    projectID string,
    issueNumber int,
    issueURL string,
    issueTitle string,
) (*domain.Requirement, error)

func (s *MetaRequirementService) CreateMetaRequirementForPR(
    ctx context.Context,
    projectID string,
    prNumber int,
    prURL string,
    prTitle string,
) (*domain.Requirement, error)

func (s *MetaRequirementService) ListMetaRequirements(
    ctx context.Context,
    projectID string,
) ([]*domain.Requirement, error)
```

### 2. 内置心跳场景更新

更新 `github_dev_workflow` 场景，新增"项目管家"心跳：

```go
func BuildGitHubDevWorkflowScenario(id string) *domain.HeartbeatScenario {
    items := []domain.HeartbeatScenarioItem{
        // 新增：项目管家心跳（优先级最高）
        {
            Name:            "GitHub 项目管家",
            IntervalMinutes: 30,
            RequirementType: "github_meta",
            AgentCode:       "",
            SortOrder:       0,
            MDContent:       "... 管家心跳 Prompt ...",
        },
        // ... 原有的 8 个心跳 ...
    }
    // ...
}
```

---

## 前端实现要点

### 1. 自动化中心增强

#### 1.1 新增"元需求看板"视图

在自动化中心新增 tab：

| Tab | 功能 |
|-----|------|
| **总览** | 现有总览 |
| **心跳实例** | 现有心跳管理 |
| **元需求看板** | 新增 - Issue/PR 元需求列表 |
| **Webhook 事件** | 现有 Webhook 管理 |
| **状态机配置** | 现有状态机管理 |

#### 1.2 元需求列表

展示内容：
- 类型（Issue/PR）图标
- 编号和标题
- 当前状态机状态
- 最后更新时间
- 关联链接（跳转到 GitHub）

#### 1.3 元需求详情页面

展示内容：
- 状态机时间线
- 执行历史记录
- 关联的心跳执行记录
- 直接跳转到 GitHub

---

## 实施路径

### Phase 1: 基础框架（优先级：P0）

**目标**：建立 Issue/PR 元需求和状态机基础

- [ ] 设计并创建 `github_issue_lifecycle` 状态机配置
- [ ] 设计并创建 `github_pr_lifecycle` 状态机配置
- [ ] 实现 `MetaRequirementService` 基础 CRUD
- [ ] 实现元需求创建逻辑
- [ ] 数据库表变更（如需要）
- [ ] 前端新增"元需求看板"基础视图

### Phase 2: 项目管家心跳（优先级：P0）

**目标**：实现管家心跳的基础扫描和决策能力

- [ ] 编写"GitHub 项目管家"心跳 Prompt
- [ ] 实现项目状态扫描逻辑（获取 Issues/PRs）
- [ ] 实现元需求自动创建逻辑
- [ ] 实现状态机初始化逻辑
- [ ] 将管家心跳加入内置场景
- [ ] 集成测试：验证管家心跳正常扫描和创建元需求

### Phase 3: 专用心跳改造（优先级：P1）

**目标**：让现有 8 个心跳被参数化调用

- [ ] 改造 Issue 分析心跳，支持指定 Issue 编号
- [ ] 改造 LGTM 代码编写心跳，支持指定 Issue 编号
- [ ] 改造 PR 需求评审心跳，支持指定 PR 编号
- [ ] 改造 PR 代码质量评审心跳，支持指定 PR 编号
- [ ] 改造 PR 修改修复心跳，支持指定 PR 编号
- [ ] 改造 PR 合并检查心跳，支持指定 PR 编号
- [ ] 改造 PR 文档补充心跳，支持指定 PR 编号
- [ ] 改造 PR 测试补充心跳，支持指定 PR 编号
- [ ] 实现心跳参数传递机制

### Phase 4: 状态机与心跳集成（优先级：P1）

**目标**：让状态机转换可以触发心跳，心跳结果可以反馈状态机

- [ ] 实现 `trigger_heartbeat` hook 的参数化
- [ ] 在关键状态转换中配置心跳触发
- [ ] 实现心跳执行结果到状态机的反馈机制
- [ ] 实现状态机的 AI Guide 动态更新
- [ ] 端到端测试：验证完整闭环

### Phase 5: 前端增强（优先级：P2）

**目标**：提供完整的元需求管理和可视化界面

- [ ] 完善元需求列表视图
- [ ] 实现元需求详情页面
- [ ] 实现状态机时间线可视化
- [ ] 实现元需求与 GitHub 的跳转
- [ ] 优化自动化中心的交互体验

### Phase 6: 可观测性增强（优先级：P2）

**目标**：提供闭环执行的可观测性

- [ ] 实现元需求执行历史记录
- [ ] 实现闭环执行指标统计
- [ ] 实现瓶颈分析
- [ ] 实现异常告警

---

## 风险与应对

| 风险 | 影响 | 概率 | 应对措施 |
|-----|------|-----|---------|
| AI 决策不可控 | 高 | 中 | 通过状态机的 AIGuide 和 SuccessCriteria 提供清晰框架；关键动作需要人工确认 |
| GitHub API 限流 | 中 | 高 | 增加冷却机制；合并扫描请求；缓存状态 |
| 状态机过于复杂 | 中 | 中 | 保持状态机简洁；提供默认配置 |
| 改造工作量大 | 中 | 中 | 分阶段实施；先验证核心流程 |

---

## 验收标准

### Phase 1 验收
- [ ] 可以创建 Issue 元需求并绑定状态机
- [ ] 可以创建 PR 元需求并绑定状态机
- [ ] 前端可以看到元需求列表

### Phase 2 验收
- [ ] 管家心跳可以正常扫描项目 Issues/PRs
- [ ] 管家心跳可以自动创建元需求
- [ ] 元需求状态机可以正常初始化

### Phase 3 验收
- [ ] 8 个专用心跳都可以接受参数调用
- [ ] 心跳可以聚焦处理单个指定 Issue/PR

### Phase 4 验收
- [ ] 状态机转换可以触发相应心跳
- [ ] 心跳执行结果可以反馈到状态机
- [ ] 可以演示完整的 Issue → PR → 合并闭环

### Phase 5 验收
- [ ] 前端元需求看板功能完整
- [ ] 状态机时间线可视化正常

### Phase 6 验收
- [ ] 查看闭环执行历史
- [ ] 显示执行统计指标

---

## 总结

本设计完全在现有架构基础上实现 AI 自主驱动的任务闭环，核心优势：

| 优势 | 说明 |
|-----|------|
| 不引入新组件 | 完全利用现有的状态机和心跳系统 |
| AI 真正自主 | AI 在状态机框架下自主决策下一步 |
| 保持灵活性 | 没有硬编码的工作流，AI 灵活调整 |
| 渐进式演进 | 分阶段实施，风险可控 |
| 支持多场景 | 基于 GitHub 支持代码开发、专利撰写、剧本写作、调研报告等多种任务 |

### 为什么选择 GitHub 作为基础设施

1. 统一的协作平台：Issue 作为任务入口，PR 作为产出物评审，评论作为协作沟通
2. 成熟的生态：标签、分支、合并等机制完善
3. 无需额外基础设施：降低成本和复杂度
4. 可扩展性强：通过不同的仓库、标签、工作流配置，支持各种场景

通过这个方案，AI 从"被动执行者"升级为"主动决策者"，真正实现任务闭环的自主运转！
