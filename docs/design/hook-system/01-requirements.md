# Hook 系统需求文档

## 1. 项目概述

### 1.1 项目名称
**LLM Agent Hook System** - 可扩展的 LLM Agent Hook 拦截系统

### 1.2 项目目标
构建一个**可扩展、分层、事件驱动**的 Hook 系统，实现对 LLM Agent 执行过程中所有关键节点的拦截、扩展和定制。

### 1.3 背景与动机

当前 LLM Agent 系统（如 Claude Code）在执行过程中有众多关键节点需要拦截：
- LLM 调用（prompt 生成、模型调用、响应解析）
- 工具执行（工具选择、参数验证、执行结果处理）
- 消息处理（消息接收/发送、错误处理）
- 技能调用（Skill 加载、调用、卸载）
- MCP 协议（MCP 请求/响应、服务器管理）
- Prompt 管理（模板加载、渲染、合并）
- 会话管理（会话创建、恢复、保存）

Claude Code 官方 Hook 仅有 8 个，远不能满足复杂业务场景需求。

## 2. 功能需求

### 2.1 核心功能

#### 2.1.1 Hook 分类体系

| 类别 | 事件数 | 描述 |
|------|--------|------|
| Lifecycle Hooks | 5 | 系统生命周期管理 |
| LLM Hooks | 8 | LLM 调用全流程拦截 |
| Tool Hooks | 10 | 工具执行全流程拦截 |
| Message Hooks | 5 | 消息处理拦截 |
| Skill Hooks | 6 | 技能系统拦截 |
| MCP Hooks | 6 | MCP 协议拦截 |
| Prompt Hooks | 6 | Prompt 管理拦截 |
| Session Hooks | 5 | 会话管理拦截 |
| **总计** | **51** | **8 大类** |

#### 2.1.2 Hook 基础能力

| 功能 | 描述 |
|------|------|
| 注册/注销 | 支持动态注册和注销 Hook |
| 启用/禁用 | 支持单独启用/禁用某个 Hook |
| 优先级 | 支持按优先级排序执行（0-100，越小越先） |
| 同步/异步 | 支持同步和异步两种执行模式 |
| 错误处理 | 支持继续执行/停止执行两种策略 |

#### 2.1.3 Hook 拦截能力

| 能力 | 描述 |
|------|------|
| Pre 拦截 | 在事件发生前拦截，可修改输入参数 |
| Post 拦截 | 在事件发生后拦截，可修改输出结果 |
| 替代执行 | 完全替代原有逻辑 |
| 观察模式 | 仅观察，不干预 |

### 2.2 用户场景

#### 2.2.1 日志与监控
- 记录所有 LLM 调用的 prompt 和响应
- 监控工具执行时间、成功率
- 统计各环节耗时

#### 2.2.2 安全与合规
- 敏感信息脱敏
- Prompt 注入检测
- 工具调用权限控制

#### 2.2.3 扩展与定制
- 自定义 Prompt 模板
- 添加新的工具
- 修改 LLM 响应

#### 2.2.4 缓存与优化
- LLM 响应缓存
- 工具结果缓存
- 限流控制

### 2.3 非功能需求

#### 2.3.1 性能
- Hook 执行开销 < 5ms
- 支持至少 100 个 Hook 同时注册

#### 2.3.2 可靠性
- 单个 Hook 错误不能影响其他 Hook
- 支持错误恢复

#### 2.3.3 可维护性
- Hook 接口简洁明了
- 文档完整清晰

## 3. 详细事件列表

### 3.1 Lifecycle Hooks (5)

| 事件名称 | 触发时机 | 输入参数 | 返回值 |
|---------|---------|---------|--------|
| `OnInitialize` | 系统初始化时 | config | error |
| `OnShutdown` | 系统关闭时 | - | error |
| `OnStart` | 会话开始时 | session_id | error |
| `OnStop` | 会话结束时 | session_id, reason | error |
| `OnError` | 错误发生时 | error, context | error |

### 3.2 LLM Hooks (8)

| 事件名称 | 触发时机 | 输入参数 | 返回值 |
|---------|---------|---------|--------|
| `PreLLMCall` | LLM 调用前 | prompt, model, params | ModifiedPrompt, error |
| `PostLLMCall` | LLM 调用后 | prompt, response, usage | ModifiedResponse, error |
| `PrePromptGeneration` | Prompt 生成前 | template, vars | ModifiedTemplate, error |
| `PostPromptGeneration` | Prompt 生成后 | final_prompt | ModifiedPrompt, error |
| `PreParseResponse` | 响应解析前 | raw_response | ModifiedRaw, error |
| `PostParseResponse` | 响应解析后 | parsed_response | ModifiedResponse, error |
| `OnLLMRetry` | LLM 重试时 | attempt, error | RetryConfig, error |
| `OnLLMTimeout` | LLM 超时时 | timeout, prompt | NewTimeout, error |

### 3.3 Tool Hooks (10)

| 事件名称 | 触发时机 | 输入参数 | 返回值 |
|---------|---------|---------|--------|
| `PreToolCall` | 工具调用前 | tool_name, tool_input | ModifiedInput, error |
| `PostToolCall` | 工具调用后 | tool_result | ModifiedResult, error |
| `OnToolError` | 工具执行错误 | error, tool_name | RetryResult, error |
| `PreToolValidation` | 工具参数验证前 | tool_name, params | ModifiedParams, error |
| `OnToolRegistered` | 工具注册时 | tool_definition | error |
| `OnToolUnregistered` | 工具注销时 | tool_name | error |
| `PreToolExecution` | 工具实际执行前 | tool_context | ModifiedContext, error |
| `PostToolExecution` | 工具执行完成后 | execution_result | ModifiedResult, error |
| `OnToolCacheHit` | 工具缓存命中 | cache_key, result | ModifiedResult, error |
| `OnToolRateLimit` | 工具限流时 | tool_name, retry_after | WaitDuration, error |

### 3.4 Message Hooks (5)

| 事件名称 | 触发时机 | 输入参数 | 返回值 |
|---------|---------|---------|--------|
| `PreMessageReceive` | 消息接收前 | raw_message | ModifiedMessage, error |
| `PostMessageReceive` | 消息接收后 | message | ModifiedMessage, error |
| `PreMessageSend` | 消息发送前 | message | ModifiedMessage, error |
| `PostMessageSend` | 消息发送后 | message | error |
| `OnMessageError` | 消息处理错误 | error, message | error |

### 3.5 Skill Hooks (6)

| 事件名称 | 触发时机 | 输入参数 | 返回值 |
|---------|---------|---------|--------|
| `PreSkillInvoke` | Skill 调用前 | skill_name, params | ModifiedParams, error |
| `PostSkillInvoke` | Skill 调用后 | skill_result | ModifiedResult, error |
| `OnSkillError` | Skill 执行错误 | error, skill_name | error |
| `OnSkillLoaded` | Skill 加载时 | skill_definition | error |
| `OnSkillUnloaded` | Skill 卸载时 | skill_name | error |
| `PreSkillExecution` | Skill 实际执行前 | execution_context | ModifiedContext, error |

### 3.6 MCP Hooks (6)

| 事件名称 | 触发时机 | 输入参数 | 返回值 |
|---------|---------|---------|--------|
| `PreMCPRequest` | MCP 请求前 | request_type, params | ModifiedParams, error |
| `PostMCPResponse` | MCP 响应后 | response | ModifiedResponse, error |
| `OnMCPError` | MCP 错误时 | error, request | error |
| `OnMCPServerStart` | MCP 服务器启动 | server_config | error |
| `OnMCPServerStop` | MCP 服务器停止 | server_name | error |
| `PreMCPStream` | MCP 流式请求前 | stream_data | ModifiedData, error |

### 3.7 Prompt Hooks (6)

| 事件名称 | 触发时机 | 输入参数 | 返回值 |
|---------|---------|---------|--------|
| `PrePromptRender` | Prompt 渲染前 | template, vars | ModifiedTemplate, error |
| `PostPromptRender` | Prompt 渲染后 | final_prompt | ModifiedPrompt, error |
| `PrePromptMerge` | Prompt 合并前 | prompts[] | ModifiedPrompts, error |
| `PostPromptMerge` | Prompt 合并后 | merged_prompt | ModifiedPrompt, error |
| `OnTemplateLoaded` | 模板加载时 | template_path | error |
| `OnTemplateError` | 模板错误时 | error, template | error |

### 3.8 Session Hooks (5)

| 事件名称 | 触发时机 | 输入参数 | 返回值 |
|---------|---------|---------|--------|
| `PreSessionCreate` | 会话创建前 | session_config | ModifiedConfig, error |
| `PostSessionCreate` | 会话创建后 | session | error |
| `OnSessionResume` | 会话恢复时 | session_id | error |
| `OnSessionExpired` | 会话过期时 | session_id | error |
| `OnSessionSave` | 会话保存时 | session_state | error |

## 4. 验收标准

### 4.1 功能验收

- [ ] 支持 8 大类 51 个 Hook 事件
- [ ] 支持 Hook 的注册、注销、启用、禁用
- [ ] 支持优先级排序
- [ ] 支持同步和异步执行
- [ ] Pre 拦截可修改输入参数
- [ ] Post 拦截可修改输出结果

### 4.2 性能验收

- [ ] 单个 Hook 执行开销 < 5ms
- [ ] 支持 100+ Hook 同时注册

### 4.3 可靠性验收

- [ ] 单个 Hook 错误不影响其他 Hook
- [ ] 支持错误策略配置

## 5. 术语表

| 术语 | 定义 |
|------|------|
| Hook | 拦截点在特定事件发生时被调用的回调函数 |
| Pre Hook | 在事件发生前执行的 Hook，可修改输入 |
| Post Hook | 在事件发生后执行的 Hook，可修改输出 |
| Hook Chain | 按优先级排序的 Hook 执行链 |
| Hook Registry | Hook 注册表，管理所有 Hook 的注册和注销 |
| Hook Executor | Hook 执行器，负责按顺序执行 Hook 链 |
