# DDD & Golang 审计优化列表

## Phase 1: 错误处理修复 (P0)
- [x] 1.1 requirement_dispatch_service.go 中 7 处 `_ = s.requirementRepo.Save()` 改为错误日志
- [x] 1.2 state_machine_service.go 中 `_ = s.requirementRepo.Save()` 改为错误日志
- [x] 1.3 requirement_service.go 中 `_ = s.requirementRepo.Delete()` 改为错误日志
- [x] 1.4 添加对应的单元测试验证错误处理

## Phase 2: Domain 层优化 (P0/P1)
- [x] 2.1 domain/requirement.go 魔法字符串常量化 ("preparing", "coding" 等)
- [x] 2.2 domain/agent.go UpdateConfig 13参数改为配置结构体 AgentConfigUpdate
- [x] 2.3 domain/mcp.go UpdateProfile 7参数改为配置结构体 MCPProfileUpdate
- [x] 2.4 添加对应的单元测试

## Phase 3: 大文件和大函数拆分 (P1)
- [x] 3.1 agent_service.go (502行) 拆分为 agent_service.go (214行) + agent_service_update.go (283行)
- [x] 3.2 UpdateAgent/PatchAgent 提取共享 applyProfileUpdate 辅助函数消除重复
- [x] 3.3 添加对应的单元测试

## Phase 4: 命名规范统一 (P2)
- [x] 4.1 UpdateAgent/PatchAgent config 合并逻辑统一为 agentConfigPatch 接口模式
- [x] 4.2 添加对应的单元测试
