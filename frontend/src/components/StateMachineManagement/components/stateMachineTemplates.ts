/**
 * 状态机模板配置
 */

export interface StateMachineTemplate {
  id: string;
  name: string;
  description: string;
  yaml: string;
}

/**
 * 状态机模板说明：
 *
 * 标准状态定义：
 * - todo（待处理）：初始状态，表示任务刚创建还未开始处理
 * - processing（处理中）：任务正在处理中，避免 todo 状态被重复进入
 * - completed（已完成）：任务完成，终止状态
 *
 * 状态转换规则：
 * 1. todo → processing：任务开始处理时立即转换，防止重复调度
 * 2. processing → completed：任务处理完成后转换
 *
 * 注意：processing 状态应该是唯一的，不应该有从 processing 回到 todo 的转换
 */

// 预定义的状态机模板
export const stateMachineTemplates: StateMachineTemplate[] = [
  {
    id: 'basic-universal',
    name: '通用基础流程',
    description: '标准三状态流程：todo(待处理) → processing(处理中) → completed(已完成)，适用于所有场景',
    yaml: `name: basic_universal
description: |
  通用基础流程 - 标准三状态模型
  状态：todo → processing → completed
  特点：处理时立即进入 processing，避免 todo 重复进入

initial_state: todo

states:
  - id: todo
    name: 待处理
    is_final: false
    ai_guide: |
      ## 状态：待处理

      这是任务的初始状态。当任务被创建或刚被派发时，处于此状态。

      **重要**：一旦开始处理，必须立即转换到 processing 状态！
      执行命令：taskmanager requirement transition {{requirement_id}} --to-state=processing

      **禁止**：不要在 processing 或 completed 状态后再转回 todo，这会导致重复处理。

  - id: processing
    name: 处理中
    is_final: false
    ai_guide: |
      ## 状态：处理中

      任务正在处理中。此时应该执行业务逻辑。

      执行步骤（根据实际需求调整）：
      1. 获取任务详情
      2. 执行相关操作
      3. 完成后转换到 completed

      **注意**：此状态不应被重复进入。完成后直接到 completed。

  - id: completed
    name: 已完成
    is_final: true
    ai_guide: |
      ## 状态：已完成

      任务已成功完成。状态机退出。

transitions:
  - from: todo
    to: processing
    trigger: start
    description: 开始处理（必须立即执行，防止重复调度）

  - from: processing
    to: completed
    trigger: complete
    description: 处理完成`,
  },
  {
    id: 'simple-workflow',
    name: '简化开发流程',
    description: '最小化的软件开发流程：todo → processing → review → completed',
    yaml: `name: simple_workflow
description: |
  简化版开发流程
  标准三状态：todo(待处理) → processing(处理中) → completed(已完成)

initial_state: todo

states:
  - id: todo
    name: 待处理
    is_final: false
    ai_guide: |
      ## 状态：待处理
      任务初始状态。立即转换到 processing 开始处理。
      命令：taskmanager requirement transition {{requirement_id}} --to-state=processing

  - id: processing
    name: 处理中
    is_final: false
    ai_guide: |
      ## 状态：处理中
      执行编码工作，完成后转换到 completed。
      命令：taskmanager requirement transition {{requirement_id}} --to-state=completed

  - id: review
    name: 审查中
    is_final: false
    ai_guide: |
      ## 状态：审查中
      代码审查阶段。

  - id: completed
    name: 已完成
    is_final: true
    ai_guide: |
      ## 状态：已完成
      任务完成，状态机退出。

transitions:
  - from: todo
    to: processing
    trigger: start
    description: 开始处理（必须立即执行）

  - from: processing
    to: review
    trigger: submit_review
    description: 提交审查

  - from: processing
    to: completed
    trigger: complete
    description: 处理完成（无需审查时）

  - from: review
    to: completed
    trigger: approve
    description: 审查通过

  - from: review
    to: processing
    trigger: reject
    description: 审查拒绝，需修复`,
  },
  {
    id: 'full-workflow',
    name: '完整开发流程',
    description: '完整的软件开发流程：todo → processing(含多阶段) → completed',
    yaml: `name: full_development_workflow
description: |
  完整的软件开发流程 - 标准三状态模型
  状态：todo → processing(含子阶段) → completed
  processing 子阶段：code_commit → code_review → build → testing → staging → production

initial_state: todo

states:
  - id: todo
    name: 待处理
    is_final: false
    ai_guide: |
      ## 状态：待处理
      **重要**：必须立即转换到 processing 状态，防止重复调度！
      命令：taskmanager requirement transition {{requirement_id}} --to-state=processing

  - id: processing
    name: 处理中
    is_final: false
    ai_guide: |
      ## 状态：处理中
      进入开发流程阶段。

  - id: code_commit
    name: 代码已提交
    is_final: false
    ai_guide: |
      ## 子状态：代码已提交
      等待提交代码审查。

  - id: code_review
    name: 代码审查中
    is_final: false
    ai_guide: |
      ## 子状态：代码审查中
      等待审查通过或拒绝。

  - id: build
    name: 构建中
    is_final: false
    ai_guide: |
      ## 子状态：构建中
      CI 构建执行中。

  - id: testing
    name: 测试中
    is_final: false
    ai_guide: |
      ## 子状态：测试中
      测试执行中。

  - id: staging
    name: 预发布环境
    is_final: false
    ai_guide: |
      ## 子状态：预发布环境
      部署到预发布环境验证。

  - id: production
    name: 生产环境
    is_final: false
    ai_guide: |
      ## 子状态：生产环境
      部署到生产环境。

  - id: completed
    name: 已完成
    is_final: true
    ai_guide: |
      ## 状态：已完成
      开发流程完成，状态机退出。

transitions:
  - from: todo
    to: processing
    trigger: start
    description: 开始处理

  - from: processing
    to: code_commit
    trigger: start_development
    description: 开始开发

  - from: code_commit
    to: code_review
    trigger: submit_review
    description: 提交代码审查

  - from: code_review
    to: build
    trigger: approve
    description: 审查通过

  - from: code_review
    to: code_commit
    trigger: reject
    description: 审查拒绝

  - from: build
    to: testing
    trigger: build_success
    description: 构建成功

  - from: build
    to: code_commit
    trigger: build_failed
    description: 构建失败

  - from: testing
    to: staging
    trigger: test_pass
    description: 测试通过

  - from: testing
    to: build
    trigger: test_failed
    description: 测试失败

  - from: staging
    to: production
    trigger: promote_production
    description: 上线生产

  - from: staging
    to: code_commit
    trigger: rollback
    description: 回滚

  - from: production
    to: completed
    trigger: confirm_production
    description: 确认上线完成`,
  },
  {
    id: 'basic',
    name: '基础流程（标准三状态）',
    description: '标准三状态流程：todo(待处理) → processing(处理中) → completed(已完成)，最简流程',
    yaml: `name: basic_workflow
description: |
  基础流程 - 标准三状态模型
  状态：todo → processing → completed

initial_state: todo

states:
  - id: todo
    name: 待处理
    is_final: false
    ai_guide: |
      ## 状态：待处理
      任务初始状态。立即转换到 processing 开始处理。
      命令：taskmanager requirement transition {{requirement_id}} --to-state=processing

  - id: processing
    name: 处理中
    is_final: false
    ai_guide: |
      ## 状态：处理中
      执行任务。完成后转换到 completed。
      命令：taskmanager requirement transition {{requirement_id}} --to-state=completed

  - id: completed
    name: 已完成
    is_final: true
    ai_guide: |
      ## 状态：已完成
      任务完成，状态机退出。

transitions:
  - from: todo
    to: processing
    trigger: start
    description: 开始处理（必须立即执行）

  - from: processing
    to: completed
    trigger: complete
    description: 处理完成`,
  },
  {
    id: 'heartbeat',
    name: '心跳任务流程',
    description: '心跳任务专用流程：todo(待处理) → processing(处理中) → completed(已完成)',
    yaml: `name: heartbeat_workflow
description: |
  心跳任务专用流程 - 标准三状态模型
  状态：todo → processing → completed
  特点：立即从 todo 转换到 processing，防止重复调度

initial_state: todo

states:
  - id: todo
    name: 待处理
    is_final: false
    ai_guide: |
      ## 状态：待处理
      **重要**：必须立即转换到 processing 状态，防止重复调度！
      命令：taskmanager requirement transition {{requirement_id}} --to-state=processing

  - id: processing
    name: 处理中
    is_final: false
    ai_guide: |
      ## 状态：处理中
      执行心跳任务逻辑：
      1. 检查任务状态和环境
      2. 执行心跳检测或监控
      3. 完成后转换到 completed
      命令：taskmanager requirement transition {{requirement_id}} --to-state=completed

  - id: completed
    name: 已完成
    is_final: true
    ai_guide: |
      ## 状态：已完成
      心跳任务执行完成，状态机退出。

transitions:
  - from: todo
    to: processing
    trigger: start
    description: 开始处理（必须立即执行，防止重复调度）

  - from: processing
    to: completed
    trigger: complete
    description: 处理完成`,
  },
  {
    id: 'requirement-execution',
    name: '需求执行流程',
    description: '需求执行流程：todo(待处理) → processing(分析实现) → completed(完成)',
    yaml: `name: requirement_execution
description: |
  需求执行流程 - 标准三状态模型
  状态：todo → processing(含子阶段) → completed
  processing 子阶段：understanding → analyzing → implementing → submitting

initial_state: todo

states:
  - id: todo
    name: 待处理
    is_final: false
    ai_guide: |
      ## 状态：待处理
      **重要**：必须立即转换到 processing 状态，防止重复调度！
      命令：taskmanager requirement transition {{requirement_id}} --to-state=processing

  - id: processing
    name: 处理中
    is_final: false
    ai_guide: |
      ## 状态：处理中
      进入处理阶段，按子状态执行：understanding → analyzing → implementing → submitting
      完成后转换到 completed

  - id: understanding
    name: 理解需求
    is_final: false
    ai_guide: |
      ## 子状态：理解需求
      执行步骤：
      1. taskmanager requirement get <requirement_id> 获取需求详情
      2. 仔细阅读 description、background、acceptance 验收标准
      3. 确认：任务目标、验收标准、工作分支

      理解完毕后 → 转换到 analyzing

  - id: analyzing
    name: 分析实现方案
    is_final: false
    ai_guide: |
      ## 子状态：分析实现方案
      执行步骤：
      1. 使用 vexor/grep 搜索相关代码
      2. 理解现有代码结构
      3. 确定修改的文件列表
      4. 制定实现步骤

      **架构约束**：
      - domain 层不引用其他层
      - infrastructure 不引用 interfaces
      - 应用服务是贫血模型（不含业务逻辑）

      分析完毕后 → 转换到 implementing

  - id: implementing
    name: 编写代码
    is_final: false
    ai_guide: |
      ## 子状态：编写代码
      执行步骤：
      1. 按 acceptance 标准逐项实现
      2. 遵循项目代码风格
      3. 运行 go vet ./... 确保无警告
      4. 编写必要的单元测试

      **约束**：
      - 单文件不超过 300 行
      - 不修改与需求无关的代码
      - 先写测试再写实现（推荐）

      代码完成后 → 转换到 submitting

  - id: submitting
    name: 提交PR
    is_final: false
    ai_guide: |
      ## 子状态：提交 PR
      执行步骤：
      1. git checkout -b <branch_name>
         - 分支命名：feat/<requirement_id>-<简短描述>
      2. git add <files>
      3. git commit -m "<描述做了什么>"
      4. git push -u origin <branch_name>
      5. gh pr create --title "<标题>" --body "<描述>"

      PR 创建成功 → 转换到 completed

  - id: completed
    name: 已完成
    is_final: true
    ai_guide: |
      ## 状态：已完成
      任务完成，状态机退出。

transitions:
  - from: understanding
    to: analyzing
    trigger: understood
    description: 需求理解完毕

  - from: analyzing
    to: implementing
    trigger: analyzed
    description: 分析完毕

  - from: implementing
    to: submitting
    trigger: implemented
    description: 代码编写完毕

  - from: submitting
    to: completed
    trigger: pr_submitted
    description: PR已提交`,
  },
  {
    id: 'pr-review-execution',
    name: 'PR审查流程',
    description: 'PR审查流程：todo → processing(含审查子阶段) → completed',
    yaml: `name: pr_review_execution
description: |
  PR审查执行流程 - 标准三状态模型
  状态：todo → processing(含子阶段) → completed
  processing 子阶段：pr_fetching → reviewing → deciding → (commenting/merging/creating_fix_requirement)

initial_state: todo

states:
  - id: todo
    name: 待处理
    is_final: false
    ai_guide: |
      ## 状态：待处理
      **重要**：必须立即转换到 processing 状态，防止重复调度！
      命令：taskmanager requirement transition {{requirement_id}} --to-state=processing

  - id: processing
    name: 处理中
    is_final: false
    ai_guide: |
      ## 状态：处理中
      进入 PR 审查阶段，按子状态执行。

  - id: pr_fetching
    name: 获取PR信息
    is_final: false
    ai_guide: |
      ## 子状态：获取 PR 信息
      执行步骤：
      1. gh pr list --state open --mergeable non-conflicting --json number,title,author,body,url
      2. 对每个 PR 获取详情：gh pr view <PR_NUMBER> --json title,body,state,url,author,headRefName,baseRefName
      3. 查看 PR 描述和变更内容

      理解 PR 的背景和目的后 → 转换到 reviewing

  - id: reviewing
    name: 审查代码
    is_final: false
    ai_guide: |
      ## 子状态：审查 PR 代码
      执行步骤：
      1. 查看变更文件：gh pr diff <PR_NUMBER>
      2. 检查代码质量和风格
      3. 验证是否遵循项目规范
      4. 检查 CI 状态：gh pr checks <PR_NUMBER>
      5. 查看评论：gh pr view <PR_NUMBER> --json comments

      **判断标准**：
      - CI 是否全部通过？
      - 评论是否都已解决？
      - 代码是否有明显问题？

      审查完毕 → 转换到 deciding

  - id: deciding
    name: 做出决策
    is_final: false
    ai_guide: |
      ## 子状态：决定下一步行动
      根据 reviewing 阶段的审查结果，做出决策：

      **情况1 - 可以评论 lgtm**：
      - 所有评论已解决、CI 通过、代码审查通过
      → 转换到 commenting

      **情况2 - 需要修复**：
      - 有未解决的评论、CI 失败、代码有问题
      → 转换到 creating_fix_requirement

      **情况3 - 可以直接合并**：
      - 已有 /lgtm 评论
      → 转换到 merging

  - id: commenting
    name: 写入lgtm评论
    is_final: false
    ai_guide: |
      ## 子状态：写入 /lgtm 评论
      执行步骤：gh pr comment <PR_NUMBER> --body "/lgtm"
      评论成功后 → 转换到 deciding（再次检查是否可以直接合并）

  - id: merging
    name: 合并PR
    is_final: false
    ai_guide: |
      ## 子状态：合并 PR
      执行步骤：gh pr merge <PR_NUMBER> --squash --delete-branch
      合并成功后 → 转换到 completed

  - id: creating_fix_requirement
    name: 创建修复需求
    is_final: false
    ai_guide: |
      ## 子状态：创建修复需求
      执行步骤：
      使用 taskmanager requirement create 创建修复需求：
      - project-id: 当前项目ID
      - title: [修复] <修复标题>
      - description: 包含 PR 链接、评论内容摘要、修复要求
      - acceptance: 修复完成后 PR 可以合并

      **重要**：不合并此 PR，创建需求让其他 AI 修复
      创建需求后 → 转换到 completed

  - id: completed
    name: 已完成
    is_final: true
    ai_guide: |
      ## 状态：已完成
      PR 处理完成，状态机退出。

transitions:
  - from: todo
    to: processing
    trigger: start
    description: 开始处理

  - from: processing
    to: pr_fetching
    trigger: start_review
    description: 开始获取PR信息

  - from: pr_fetching
    to: reviewing
    trigger: pr_fetched
    description: PR信息已获取

  - from: reviewing
    to: deciding
    trigger: review_completed
    description: 审查完毕

  - from: deciding
    to: commenting
    trigger: need_lgtm
    description: 需要lgtm

  - from: deciding
    to: merging
    trigger: can_merge
    description: 可以合并

  - from: deciding
    to: creating_fix_requirement
    trigger: needs_fix
    description: 需要修复

  - from: commenting
    to: deciding
    trigger: lgtm_posted
    description: lgtm已评论

  - from: merging
    to: completed
    trigger: merged
    description: 已合并

  - from: creating_fix_requirement
    to: completed
    trigger: requirement_created
    description: 修复需求已创建`,
  },
  {
    id: 'optimization-execution',
    name: '优化点分析流程',
    description: '优化点分析流程：todo → processing(含分析子阶段) → completed',
    yaml: `name: optimization_execution
description: |
  优化点分析流程 - 标准三状态模型
  状态：todo → processing(含子阶段) → completed
  processing 子阶段：selecting_direction → analyzing → creating_requirement

initial_state: todo

states:
  - id: todo
    name: 待处理
    is_final: false
    ai_guide: |
      ## 状态：待处理
      **重要**：必须立即转换到 processing 状态，防止重复调度！
      命令：taskmanager requirement transition {{requirement_id}} --to-state=processing

  - id: processing
    name: 处理中
    is_final: false
    ai_guide: |
      ## 状态：处理中
      进入优化点分析阶段，按子状态执行。

  - id: selecting_direction
    name: 选择优化方向
    is_final: false
    ai_guide: |
      ## 子状态：选择优化方向
      从以下三个方向中选择一个：

      **方向1 - Go 最佳实践**：
      检查代码中是否有：
      - 性能问题（不必要的内存分配、重复计算）
      - 错误处理不当
      - 架构违反 DDD 原则
      → 使用 vexor 搜索相关代码模式

      **方向2 - 测试覆盖**：
      检查关键模块是否缺少测试：
      - domain 层核心逻辑
      - application 层服务
      - 边界条件处理
      → 使用 go test -cover 检查覆盖率

      **方向3 - 功能优化**：
      搜索可优化点：
      - 重复代码（可以提取公共函数）
      - 硬编码（可以配置化）
      - 缺失的功能（根据用户场景判断）
      → 使用 grep/vexor 分析

      选择方向后 → 转换到 analyzing

  - id: analyzing
    name: 深入分析
    is_final: false
    ai_guide: |
      ## 子状态：深入分析优化点
      执行步骤：
      1. 定位相关代码文件和代码段
      2. 分析问题的具体原因
      3. 评估优化后的收益
      4. 确定具体的优化方案

      **重要约束**：
      - 优化点要具体、可执行
      - 避免过度优化（简单问题不要复杂化）
      - 优先优化影响大的点

      分析完毕后 → 转换到 creating_requirement

  - id: creating_requirement
    name: 创建优化需求
    is_final: false
    ai_guide: |
      ## 子状态：创建优化需求
      执行步骤：
      使用 taskmanager requirement create 创建优化需求：
      - project-id: 当前项目ID
      - title: [优化] <优化标题>
      - description: 包含背景分析、优化方案、验收标准
      - acceptance: 优化完成并验证通过

      创建需求后 → 转换到 completed

  - id: completed
    name: 已完成
    is_final: true
    ai_guide: |
      ## 状态：已完成
      优化点分析完成，状态机退出。

transitions:
  - from: todo
    to: processing
    trigger: start
    description: 开始处理

  - from: processing
    to: selecting_direction
    trigger: start_analysis
    description: 开始选择优化方向

  - from: selecting_direction
    to: analyzing
    trigger: direction_selected
    description: 优化方向已选定

  - from: analyzing
    to: creating_requirement
    trigger: analysis_completed
    description: 分析完毕

  - from: creating_requirement
    to: completed
    trigger: requirement_created
    description: 优化需求已创建`,
  },
];
