/**
 * 状态机模板配置
 */

export interface StateMachineTemplate {
  id: string;
  name: string;
  description: string;
  yaml: string;
}

// 预定义的状态机模板
export const stateMachineTemplates: StateMachineTemplate[] = [
  {
    id: 'simple-workflow',
    name: '简化开发流程',
    description: '最小化的软件开发流程：提交 → 审查 → 构建 → 测试 → 完成',
    yaml: `name: simple_workflow
description: 简化版开发流程

initial_state: submitted

states:
  - id: submitted
    name: 已提交
    is_final: false
  - id: in_review
    name: 审查中
    is_final: false
  - id: building
    name: 构建中
    is_final: false
  - id: testing
    name: 测试中
    is_final: false
  - id: completed
    name: 已完成
    is_final: true

transitions:
  - from: submitted
    to: in_review
    trigger: submit_review
    description: 提交审查
    hooks:
      - name: 通知审查者
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: in_review
    to: building
    trigger: approve
    description: 审查通过
    hooks:
      - name: 触发构建
        type: command
        config:
          command: echo "Building {{requirement_id}}..."
        timeout: 30
        retry: 0

  - from: in_review
    to: submitted
    trigger: reject
    description: 审查拒绝
    hooks:
      - name: 通知开发者
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: building
    to: testing
    trigger: build_success
    description: 构建成功
    hooks:
      - name: 触发测试
        type: command
        config:
          command: echo "Testing {{requirement_id}}..."
        timeout: 30
        retry: 0

  - from: building
    to: submitted
    trigger: build_failed
    description: 构建失败
    hooks:
      - name: 通知失败
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: testing
    to: completed
    trigger: test_pass
    description: 测试通过
    hooks:
      - name: 发送完成通知
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: testing
    to: building
    trigger: test_failed
    description: 测试失败
    hooks:
      - name: 通知测试失败
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1`,
  },
  {
    id: 'full-workflow',
    name: '完整开发流程',
    description: '完整的软件开发流程：提交 → 审查 → 构建 → 测试 → 预发布 → 生产 → 完成',
    yaml: `name: full_development_workflow
description: 完整的软件开发流程

initial_state: code_commit

states:
  - id: code_commit
    name: 代码已提交
    is_final: false
  - id: code_review
    name: 代码审查中
    is_final: false
  - id: build
    name: 构建中
    is_final: false
  - id: testing
    name: 测试中
    is_final: false
  - id: staging
    name: 预发布环境
    is_final: false
  - id: production
    name: 生产环境
    is_final: false
  - id: completed
    name: 已完成
    is_final: true

transitions:
  - from: code_commit
    to: code_review
    trigger: submit_review
    description: 提交代码审查
    hooks:
      - name: 发送审查通知
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: code_review
    to: build
    trigger: approve
    description: 审查通过
    hooks:
      - name: 触发CI构建
        type: command
        config:
          command: echo "Building {{requirement_id}}..."
        timeout: 300
        retry: 0

  - from: code_review
    to: code_commit
    trigger: reject
    description: 审查拒绝
    hooks:
      - name: 发送拒绝通知
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: build
    to: testing
    trigger: build_success
    description: 构建成功
    hooks:
      - name: 触发测试
        type: command
        config:
          command: echo "Testing {{requirement_id}}..."
        timeout: 600
        retry: 0

  - from: build
    to: code_commit
    trigger: build_failed
    description: 构建失败
    hooks:
      - name: 发送构建失败通知
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: testing
    to: staging
    trigger: test_pass
    description: 测试通过
    hooks:
      - name: 部署到预发布
        type: command
        config:
          command: echo "Deploying to staging..."
        timeout: 300
        retry: 2

  - from: testing
    to: build
    trigger: test_failed
    description: 测试失败
    hooks:
      - name: 发送测试失败通知
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: staging
    to: production
    trigger: promote_production
    description: 上线生产
    hooks:
      - name: 执行生产部署
        type: command
        config:
          command: echo "Deploying to production..."
        timeout: 600
        retry: 3

  - from: staging
    to: code_commit
    trigger: rollback
    description: 回滚
    hooks:
      - name: 发送回滚通知
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: production
    to: completed
    trigger: confirm_production
    description: 确认上线完成
    hooks:
      - name: 发送完成通知
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1`,
  },
  {
    id: 'basic',
    name: '基础流程',
    description: '最简单的三状态流程：待处理 → 进行中 → 已完成',
    yaml: `name: basic_workflow
description: 基础流程

initial_state: pending

states:
  - id: pending
    name: 待处理
    is_final: false
  - id: in_progress
    name: 进行中
    is_final: false
  - id: completed
    name: 已完成
    is_final: true

transitions:
  - from: pending
    to: in_progress
    trigger: start
    description: 开始处理
    hooks:
      - name: 发送开始通知
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: in_progress
    to: completed
    trigger: complete
    description: 完成处理
    hooks:
      - name: 发送完成通知
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1`,
  },
  {
    id: 'heartbeat',
    name: '心跳任务流程',
    description: '简单心跳任务流程：待处理 → 处理中 → 已完成',
    yaml: `name: heartbeat_workflow
description: 心跳任务专用流程

initial_state: todo

states:
  - id: todo
    name: 待处理
    is_final: false
    ai_guide: |
      立即执行转换到 processing 状态，防止重复调度。
      使用命令: taskmanager requirement transition {{requirement_id}} --to-state=processing

  - id: processing
    name: 处理中
    is_final: false
    ai_guide: |
      执行心跳任务逻辑，完成后标记为完成。
      使用命令: taskmanager requirement transition {{requirement_id}} --to-state=completed

  - id: completed
    name: 已完成
    is_final: true

transitions:
  - from: todo
    to: processing
    trigger: start
    description: 开始处理

  - from: processing
    to: completed
    trigger: complete
    description: 处理完成`,
  },
  {
    id: 'requirement-execution',
    name: '需求执行流程',
    description: '需求执行流程：理解 → 分析 → 实现 → 提交PR（以PR提交为结束）',
    yaml: `name: requirement_execution
description: 需求执行流程 - 理解 → 分析 → 实现 → 提交PR

initial_state: understanding

states:
  - id: understanding
    name: 理解需求
    is_final: false
    ai_guide: |
      ## 任务：理解需求内容

      执行步骤：
      1. taskmanager requirement get <requirement_id> 获取需求详情
      2. 仔细阅读 description、background、acceptance 验收标准
      3. 确认：任务目标、验收标准、工作分支

      理解完毕后 → 转换到 analyzing

  - id: analyzing
    name: 分析实现方案
    is_final: false
    ai_guide: |
      ## 任务：分析实现方案

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
      ## 任务：实现代码

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
      ## 任务：提交 PR

      执行步骤：
      1. git checkout -b <branch_name>
         - 分支命名：feat/<requirement_id>-<简短描述>
      2. git add <files>
      3. git commit -m "<描述做了什么>"
      4. git push -u origin <branch_name>
      5. gh pr create --title "<标题>" --body "<描述>"

      PR 创建成功 → 转换到 completed

  - id: completed
    name: PR已提交
    is_final: true
    ai_guide: |
      ## 任务完成

      PR 已提交，状态机退出。

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
    description: 'PR审查流程：获取PR → 审查 → 决策 → 执行（lgtm/合并/创建修复需求）',
    yaml: `name: pr_review_execution
description: PR审查执行流程 - 获取 → 审查 → 决策 → 执行

initial_state: pr_fetching

states:
  - id: pr_fetching
    name: 获取PR信息
    is_final: false
    ai_guide: |
      ## 任务：获取 PR 信息

      执行步骤：
      1. gh pr list --state open --mergeable non-conflicting --json number,title,author,body,url
      2. 对每个 PR 获取详情：gh pr view <PR_NUMBER> --json title,body,state,url,author,headRefName,baseRefName
      3. 查看 PR 描述和变更内容

      理解 PR 的背景和目的后 → 转换到 reviewing

  - id: reviewing
    name: 审查代码
    is_final: false
    ai_guide: |
      ## 任务：审查 PR 代码

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
      ## 任务：决定下一步行动

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
      ## 任务：写入 /lgtm 评论

      执行步骤：
      gh pr comment <PR_NUMBER> --body "/lgtm"

      评论成功后 → 转换到 deciding（再次检查是否可以直接合并）

  - id: merging
    name: 合并PR
    is_final: false
    ai_guide: |
      ## 任务：合并 PR

      执行步骤：
      gh pr merge <PR_NUMBER> --squash --delete-branch

      选项：
      - --squash：压缩合并
      - --delete-branch：合并后删除源分支

      合并成功后 → 转换到 completed

  - id: creating_fix_requirement
    name: 创建修复需求
    is_final: false
    ai_guide: |
      ## 任务：创建修复需求

      执行步骤：
      使用 taskmanager requirement create 创建修复需求：
      - project-id: 当前项目ID
      - title: [修复] <修复标题>
      - description: 包含 PR 链接、评论内容摘要、修复要求
      - acceptance: 修复完成后 PR 可以合并

      **重要**：不合并此 PR，创建需求让其他 AI 修复

      创建需求后 → 转换到 completed

  - id: completed
    name: PR处理完成
    is_final: true
    ai_guide: |
      ## PR 处理完成

      状态机退出。

transitions:
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
    description: '优化点分析流程：选择方向 → 分析 → 创建优化需求',
    yaml: `name: optimization_execution
description: 优化点分析流程 - 选择方向 → 分析 → 创建需求

initial_state: selecting_direction

states:
  - id: selecting_direction
    name: 选择优化方向
    is_final: false
    ai_guide: |
      ## 任务：选择本次优化方向

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
      ## 任务：深入分析优化点

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
      ## 任务：创建优化需求

      执行步骤：
      使用 taskmanager requirement create 创建优化需求：
      - project-id: 当前项目ID
      - title: [优化] <优化标题>
      - description: 包含背景分析、优化方案、验收标准
      - acceptance: 优化完成并验证通过

      创建需求后 → 转换到 completed

  - id: completed
    name: 优化点分析完成
    is_final: true
    ai_guide: |
      ## 优化点分析完成

      状态机退出。

transitions:
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
