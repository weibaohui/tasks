package domain

import "context"

// MCPToolService provides MCP server and tool operations for infrastructure tools.
type MCPToolService interface {
	ListServers(ctx context.Context) ([]*MCPServer, error)
	ListTools(ctx context.Context, id MCPServerID) ([]*MCPToolModel, error)
	ExecuteTool(ctx context.Context, serverID MCPServerID, toolName string, params map[string]interface{}) (string, error)
}
