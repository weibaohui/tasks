# 状态机 AI 指南设计文档

## 背景

当前状态机只定义了状态流转规则，AI 不知道在每个状态应该做什么、如何判断完成、如何决定下一步。需要增强状态机，让每个状态都包含 AI 操作指南。

## 目标

1. **明确指导**：AI 进入每个状态时，知道该执行什么任务
2. **判断标准**：AI 知道如何判断成功/失败
3. **触发器选择**：AI 知道在什么条件下应该触发哪个转换
4. **自动初始化**：进入状态时自动执行准备命令（可选）

## 设计

### 增强的数据结构

```go
// State 状态节点（增强版）
type State struct {
    ID       string `json:"id" yaml:"id"`
    Name     string `json:"name" yaml:"name"`
    IsFinal  bool   `json:"is_final" yaml:"is_final"`
    
    // === 新增字段 ===
    
    // AIGuide AI操作指南（Markdown格式）
    // 说明当前阶段应该做什么、执行步骤、注意事项
    AIGuide string `json:"ai_guide,omitempty" yaml:"ai_guide,omitempty"`
    
    // AutoInit 自动初始化命令（可选）
    // 进入此状态时自动执行的 shell 命令
    // 如果执行失败，自动触发 fail 转换
    AutoInit string `json:"auto_init,omitempty" yaml:"auto_init,omitempty"`
    
    // SuccessCriteria 成功判断标准
    // AI 根据此标准判断任务是否成功完成
    SuccessCriteria string `json:"success_criteria,omitempty" yaml:"success_criteria,omitempty"`
    
    // FailureCriteria 失败判断标准
    // AI 根据此标准判断任务是否失败
    FailureCriteria string `json:"failure_criteria,omitempty" yaml:"failure_criteria,omitempty"`
    
    // Triggers 可用的触发器说明
    // 告诉 AI 在什么条件下应该触发哪个转换
    Triggers []StateTriggerGuide `json:"triggers,omitempty" yaml:"triggers,omitempty"`
}

// StateTriggerGuide 状态内触发器指南
type StateTriggerGuide struct {
    Trigger     string `json:"trigger" yaml:"trigger"`
    Description string `json:"description" yaml:"description"`
    Condition   string `json:"condition" yaml:"condition"`
}
```

### 完整示例配置

```yaml
name: dev_workflow
description: 开发任务工作流

initial_state: preparing

states:
  - id: preparing
    name: 准备中
    ai_guide: |
      ## 当前阶段任务
      
      你需要准备开发环境：
      1. 检查工作目录是否为空
      2. 如果为空，克隆代码仓库
      3. 检出正确的分支
      4. 执行项目初始化（安装依赖等）
      
      ## 验收标准
      
      - 工作目录存在且包含 `.git` 目录
      - `git status` 能正常执行
      - 项目依赖已安装（如 package.json 中的依赖）
      
      ## 失败处理
      
      如果克隆失败或分支不存在，使用 `fail` 触发器。
    
    auto_init: |
      #!/bin/bash
      set -e
      if [ ! -d ".git" ]; then
        echo "克隆代码仓库..."
        git clone {{git_repo_url}} . || exit 1
      fi
      git checkout {{default_branch}} || exit 1
      git pull origin {{default_branch}} || exit 1
      echo "环境准备完成"
    
    triggers:
      - trigger: ready
        description: 环境准备就绪，可以开始编码
        condition: 工作目录已初始化，git状态正常
      
      - trigger: fail
        description: 环境准备失败
        condition: 克隆失败、分支不存在或初始化脚本执行失败

  - id: coding
    name: 编码中
    ai_guide: |
      ## 当前阶段任务
      
      根据需求描述实现功能：
      
      1. **理解需求**
         - 仔细阅读需求标题和描述
         - 确认验收标准
      
      2. **分析代码**
         - 查看项目结构
         - 找到需要修改的文件
         - 理解现有代码逻辑
      
      3. **实现功能**
         - 编写代码实现需求
         - 遵循项目代码规范
         - 添加必要的注释
      
      4. **本地验证**
         - 运行测试（如果有）
         - 手动验证功能
         - 检查边界情况
      
      ## 成功判断标准
      
      以下**全部满足**时，使用 `complete` 触发器：
      - 代码实现完整，满足需求描述
      - 所有验收标准已满足
      - 测试通过（如有测试命令）
      - 代码风格符合项目规范
      
      ## 部分完成判断标准
      
      以下情况时，使用 `partial` 触发器：
      - 核心功能已实现，但存在非阻塞性问题
      - 需要人工 Review 确认
      - 有遗留问题需要后续处理
      
      ## 失败判断标准
      
      以下情况时，使用 `fail` 触发器：
      - 尝试多种方案仍无法实现需求
      - 遇到技术障碍无法解决
      - 发现需求本身存在问题无法继续
      
      ## 可用命令
      
      - 查看需求详情：`taskmanager requirement get --id {{requirement_id}}`
      - 查看项目信息：`taskmanager project get --id {{project_id}}`
      - 执行状态转换：`taskmanager requirement transition --id {{requirement_id}} --trigger <trigger>`
    
    triggers:
      - trigger: complete
        description: 代码实现完成且测试通过
        condition: 验收标准全部满足，测试通过
      
      - trigger: partial
        description: 部分完成，需要人工Review
        condition: 核心功能完成但存在遗留问题
      
      - trigger: fail
        description: 无法实现或遇到技术障碍
        condition: 尝试多种方案仍无法解决
      
      - trigger: block
        description: 需要等待外部依赖
        condition: 依赖其他需求或等待外部资源

  - id: reviewing
    name: 代码审查
    ai_guide: |
      ## 当前阶段任务
      
      审查已提交的代码：
      1. 查看代码变更
      2. 检查是否满足需求
      3. 检查代码质量
      4. 决定是否通过
      
      ## 注意
      
      此状态通常由人工或专门的 Review Agent 处理。
      
    triggers:
      - trigger: approve
        description: 审查通过，可以合并
        condition: 代码质量合格，满足需求
      
      - trigger: reject
        description: 审查不通过，需要修改
        condition: 存在问题需要修复

  - id: completed
    name: 已完成
    is_final: true
    ai_guide: |
      ## 任务完成
      
      需求已实现并通过审查。无需进一步操作。

  - id: failed
    name: 已失败
    is_final: true
    ai_guide: |
      ## 任务失败
      
      需求无法实现。建议：
      1. 查看失败原因
      2. 人工评估是否可以重新派发
      3. 或调整需求后重试

transitions:
  - from: preparing
    to: coding
    trigger: ready
    
  - from: preparing
    to: failed
    trigger: fail
    
  - from: coding
    to: reviewing
    trigger: complete
    
  - from: coding
    to: reviewing
    trigger: partial
    
  - from: coding
    to: failed
    trigger: fail
    
  - from: coding
    to: coding
    trigger: block
    
  - from: reviewing
    to: completed
    trigger: approve
    
  - from: reviewing
    to: coding
    trigger: reject
```

## CLI 接口

### 新增命令

```bash
# 获取指定状态的 AI 指南
taskmanager statemachine guide --machine=dev_workflow --state=coding

# 输出示例：
# {
#   "state": "coding",
#   "name": "编码中",
#   "ai_guide": "## 当前阶段任务...",
#   "auto_init": null,
#   "success_criteria": "验收标准全部满足...",
#   "failure_criteria": "尝试多种方案仍无法实现...",
#   "triggers": [
#     {"trigger": "complete", "description": "...", "condition": "..."},
#     {"trigger": "partial", "description": "...", "condition": "..."},
#     {"trigger": "fail", "description": "...", "condition": "..."}
#   ]
# }
```

## 分身 Agent 提示词集成

在 `buildRequirementDispatchPrompt` 中，注入当前状态的完整指南：

```go
func buildRequirementDispatchPrompt(requirement *Requirement, project *Project, 
    workspacePath string, stateMachineName string, currentState string) string {
    
    // ... 现有代码 ...
    
    // 获取当前状态的 AI 指南
    stateGuide := getStateAIGuide(stateMachineName, currentState)
    
    prompt := fmt.Sprintf(`
【需求元信息】
...

%s  ← 注入状态机使用指南

【当前状态执行指南】
%s  ← 注入当前状态的 ai_guide

【判断标准】
成功：%s  ← 注入 success_criteria
失败：%s  ← 注入 failure_criteria

【可用状态转换】
%s  ← 注入 triggers 列表

【执行流程】
请按以下顺序执行：
1. 如果存在 AutoInit 脚本，先执行它
2. 按照【当前状态执行指南】完成任务
3. 根据【判断标准】决定任务结果
4. 选择合适的触发器执行状态转换：
   taskmanager requirement transition --id %s --trigger <trigger>
`, ...)
}
```

## 实现计划

### Phase 1: 数据结构扩展
- [ ] 扩展 `State` 结构体，添加 AI 相关字段
- [ ] 扩展 `Transition` 结构体（如有需要）
- [ ] 更新 YAML 解析和校验逻辑

### Phase 2: CLI 支持
- [ ] 新增 `statemachine guide` 子命令
- [ ] 更新 `statemachine validate` 支持新字段校验

### Phase 3: 提示词集成
- [ ] 修改 `buildRequirementDispatchPrompt`，注入状态指南
- [ ] 支持 `auto_init` 自动执行（可选）

### Phase 4: 前端支持
- [ ] 状态机编辑器支持 AI Guide 编辑
- [ ] 可视化触发器配置

## 向后兼容性

所有新增字段都是 `omitempty`，现有状态机配置无需修改即可继续工作。没有 AI Guide 的状态保持现有行为（AI 根据通用提示词自行判断）。
