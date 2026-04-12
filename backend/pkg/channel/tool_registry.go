package channel

import (
	"context"
	"strings"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/llm/tools"
	"github.com/weibh/taskmanager/infrastructure/llm/tools/mcp"
	skilltools "github.com/weibh/taskmanager/infrastructure/llm/tools/skill"
)

func (p *MessageProcessor) buildAgentToolsRegistry(ctx context.Context, agent *domain.Agent, contextParams map[string]string) *domain.ToolRegistry {
	if agent == nil {
		return nil
	}

	registry := domain.NewToolRegistry()
	registered := false

	// 1. 注册 Bash 工具（如果 agent.ToolsList 包含 "bash"）
	agentTools := agent.ToolsList()
	for _, t := range agentTools {
		if t == "bash" {
			registry.Register(tools.NewBashTool())
			registered = true
			break
		}
	}

	// 2. 注册 MCP 工具（如果 agent 有 MCP 绑定）
	if p.mcpService != nil {
		bindings, err := p.mcpService.ListAgentBindings(ctx, agent.ID())
		if err == nil && len(bindings) > 0 {
			// 检查是否有任何启用的绑定
			hasActiveBinding := false
			for _, b := range bindings {
				if b.IsActive() {
					hasActiveBinding = true
					break
				}
			}
			if hasActiveBinding {
				registry.Register(mcp.NewUseMCPTool(p.mcpService))
				registry.Register(mcp.NewCallMCPTool(p.mcpService))
				registered = true
			}
		}
	}

	// 3. 注册 Skills 工具（如果 agent.SkillsList 非空）
	if p.skillsLoader != nil {
		skills := p.skillsLoader.ListSkills()
		agentSkills := agent.SkillsList()
		if len(agentSkills) > 0 && len(skills) > 0 {
			enabledSkills := make(map[string]bool)
			for _, s := range agentSkills {
				enabledSkills[s] = true
			}

			skillToolsRegistry := tools.NewSkillToolsAdapterRegistry(p.skillsLoader)
			// 使用 GetToolsForSkills 避免重复发现技能（复用已获取的 skills 列表）
			for _, t := range skillToolsRegistry.GetToolsForSkills(skills) {
				toolName := t.Name()
				if strings.HasPrefix(toolName, "skill__") {
					skillName := strings.TrimPrefix(toolName, "skill__")
					if enabledSkills[skillName] {
						registry.Register(t)
						registered = true
					}
				}
			}

			// 注册 use_skill 工具（用于动态加载技能）
			registry.Register(skilltools.NewUseSkillTool(p.skillsLoader))
			registered = true
		}
	}

	// 4. 注册任务工具（已禁用 - Task 模块已移除）
	// if p.taskService != nil {
	// 	registry.Register(tasktools.NewCreateTaskTool(
	// 		p.taskService,
	// 		p.idGenerator,
	// 		contextParams["agentCode"],
	// 		contextParams["userCode"],
	// 		contextParams["channelCode"],
	// 		contextParams["sessionKey"],
	// 	))
	// 	registry.Register(tasktools.NewQueryTaskTool(p.taskService))
	// 	registered = true
	// }

	// 如果没有注册任何工具，返回 nil
	if !registered {
		return nil
	}

	return registry
}

