package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/weibh/taskmanager/domain"
)

// mcpClientAdapter 适配 mcp-go client.Client 到 domain.MCPClient 接口
type mcpClientAdapter struct {
	inner *client.Client
}

// Ensure mcpClientAdapter implements domain.MCPClient
var _ domain.MCPClient = (*mcpClientAdapter)(nil)

func (a *mcpClientAdapter) Start(ctx context.Context) error {
	return a.inner.Start(ctx)
}

func (a *mcpClientAdapter) Initialize(ctx context.Context) error {
	_, err := a.inner.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "taskmanager-mcp-client", Version: "1.0.0"},
			Capabilities:    mcp.ClientCapabilities{},
		},
	})
	return err
}

func (a *mcpClientAdapter) ListTools(ctx context.Context) ([]domain.MCPToolInfo, error) {
	res, err := a.inner.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, err
	}
	tools := make([]domain.MCPToolInfo, 0, len(res.Tools))
	for _, t := range res.Tools {
		var schema map[string]interface{}
		if t.InputSchema.Properties != nil {
			b, err := json.Marshal(t.InputSchema)
			if err == nil {
				_ = json.Unmarshal(b, &schema)
			}
		}
		tools = append(tools, domain.MCPToolInfo{
			Name:        t.Name,
			Description: t.Description,
			Schema:      schema,
		})
	}
	return tools, nil
}

func (a *mcpClientAdapter) CallTool(ctx context.Context, toolName string, params map[string]interface{}) (domain.MCPToolResult, error) {
	res, err := a.inner.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: params,
		},
	})
	if err != nil {
		return domain.MCPToolResult{}, err
	}
	content := ""
	if len(res.Content) > 0 {
		if text, ok := res.Content[0].(mcp.TextContent); ok {
			content = text.Text
		} else {
			b, _ := json.Marshal(res.Content)
			content = string(b)
		}
	}
	return domain.MCPToolResult{
		Content: content,
		IsError: res.IsError,
	}, nil
}

func (a *mcpClientAdapter) Close() error {
	return a.inner.Close()
}

// MCPClientFactoryImpl 实现 domain.MCPClientFactory
type MCPClientFactoryImpl struct{}

// Ensure MCPClientFactoryImpl implements domain.MCPClientFactory
var _ domain.MCPClientFactory = (*MCPClientFactoryImpl)(nil)

func (f *MCPClientFactoryImpl) CreateClient(server *domain.MCPServer) (domain.MCPClient, error) {
	var c *client.Client
	var err error
	switch server.TransportType() {
	case domain.MCPTransportSTDIO:
		env := []string{}
		for k, v := range server.EnvVars() {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		c, err = client.NewStdioMCPClient(server.Command(), env, server.Args()...)
	case domain.MCPTransportHTTP:
		c, err = client.NewStreamableHttpClient(server.URL())
	case domain.MCPTransportSSE:
		c, err = client.NewSSEMCPClient(server.URL())
	default:
		return nil, fmt.Errorf("不支持的传输类型: %s", server.TransportType())
	}
	if err != nil {
		return nil, err
	}
	return &mcpClientAdapter{inner: c}, nil
}
