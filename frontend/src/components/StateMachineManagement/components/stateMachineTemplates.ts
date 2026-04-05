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
];
