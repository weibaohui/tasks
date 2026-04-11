package channel

import (
	"context"
	"fmt"
	"strings"
	"github.com/weibh/taskmanager/domain"
)

func (p *MessageProcessor) buildPrompt(ctx context.Context, session *Session, userInput string, agent *domain.Agent) string {
	var sb strings.Builder

	// 添加 Agent 人格信息
	if agent != nil {
		appendIfNotEmpty := func(content string) {
			if strings.TrimSpace(content) != "" {
				sb.WriteString(content)
				sb.WriteString("\n\n")
			}
		}
		appendIfNotEmpty(agent.IdentityContent())
		appendIfNotEmpty(agent.SoulContent())
		appendIfNotEmpty(agent.AgentsContent())
		appendIfNotEmpty(agent.ToolsContent())
		appendIfNotEmpty(agent.UserContent())
	}

	// 添加 MCP Server 列表（如果有绑定）
	if agent != nil && p.mcpService != nil {
		mcpInfo := p.getAgentMCPServers(ctx, agent)
		if mcpInfo != "" {
			sb.WriteString(mcpInfo)
			sb.WriteString("\n")
		}
	}

	// 添加对话历史
	messages := session.Messages()
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			sb.WriteString(fmt.Sprintf("用户: %s\n", msg.Content))
		case "assistant":
			sb.WriteString(fmt.Sprintf("助手: %s\n", msg.Content))
		case "system":
			sb.WriteString(fmt.Sprintf("系统: %s\n", msg.Content))
		}
	}

	// 添加当前用户输入
	sb.WriteString(fmt.Sprintf("用户: %s\n助手:", userInput))

	return sb.String()
}

// getAgentMCPServers 获取 Agent 绑定的 MCP Server 列表，生成提示词
func (p *MessageProcessor) getAgentMCPServers(ctx context.Context, agent *domain.Agent) string {
	if agent == nil {
		return ""
	}

	bindings, err := p.mcpService.ListAgentBindings(ctx, agent.ID())
	if err != nil || len(bindings) == 0 {
		return ""
	}

	var servers []string
	for _, binding := range bindings {
		if !binding.IsActive() {
			continue
		}
		server, err := p.mcpService.GetServer(ctx, binding.MCPServerID())
		if err != nil || server == nil {
			continue
		}
		if server.Status() != "active" {
			continue
		}
		desc := server.Description()
		if desc == "" {
			desc = "无描述"
		}
		servers = append(servers, fmt.Sprintf("- **%s** (%s): %s", server.Code(), server.Name(), desc))
	}

	if len(servers) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## 可用的 MCP Servers\n")
	sb.WriteString("你可以使用 `use_mcp` 工具加载以下 MCP Server 的工具:\n\n")
	for _, s := range servers {
		sb.WriteString(s)
		sb.WriteString("\n")
	}
	sb.WriteString("\n使用示例: use_mcp(server_code=\"服务器编码\", action=\"load\")")

	return sb.String()
}

// buildAgentToolsRegistry 为 Agent 构建工具注册表
// 包括 Bash（按 agent.ToolsList 配置）、MCP（按 agent 绑定）、Skills（按 agent.SkillsList 配置）
// 如果各项配置都为空，则不注册任何工具
