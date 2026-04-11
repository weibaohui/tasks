package application

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type CreateMCPServerCommand struct {
	Code          string
	Name          string
	Description   string
	TransportType domain.MCPTransportType
	Command       string
	Args          []string
	URL           string
	EnvVars       map[string]string
}

type UpdateMCPServerCommand struct {
	ID            domain.MCPServerID
	Name          *string
	Description   *string
	TransportType *domain.MCPTransportType
	Command       *string
	Args          *[]string
	URL           *string
	EnvVars       *map[string]string
}

type CreateAgentMCPBindingCommand struct {
	AgentID      domain.AgentID
	MCPServerID  domain.MCPServerID
	EnabledTools []string
	IsActive     *bool
	AutoLoad     *bool
}

type UpdateAgentMCPBindingCommand struct {
	ID           domain.AgentMCPBindingID
	EnabledTools *[]string
	IsActive     *bool
	AutoLoad     *bool
}

type MCPApplicationService struct {
	mcpServerRepo  domain.MCPServerRepository
	agentRepo      domain.AgentRepository
	bindingRepo    domain.AgentMCPBindingRepository
	mcpToolRepo    domain.MCPToolRepository
	mcpToolLogRepo domain.MCPToolLogRepository
	idGen          domain.IDGenerator
	clientFactory  domain.MCPClientFactory
}

func NewMCPApplicationService(
	mcpServerRepo domain.MCPServerRepository,
	agentRepo domain.AgentRepository,
	bindingRepo domain.AgentMCPBindingRepository,
	mcpToolRepo domain.MCPToolRepository,
	mcpToolLogRepo domain.MCPToolLogRepository,
	idGen domain.IDGenerator,
	clientFactory domain.MCPClientFactory,
) *MCPApplicationService {
	return &MCPApplicationService{
		mcpServerRepo:  mcpServerRepo,
		agentRepo:      agentRepo,
		bindingRepo:    bindingRepo,
		mcpToolRepo:    mcpToolRepo,
		mcpToolLogRepo: mcpToolLogRepo,
		idGen:          idGen,
		clientFactory:  clientFactory,
	}
}

// MCP Server
func (s *MCPApplicationService) CreateServer(ctx context.Context, cmd CreateMCPServerCommand) (*domain.MCPServer, error) {
	exists, err := s.mcpServerRepo.CheckCodeExists(ctx, cmd.Code)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("MCP 服务器编码已存在: %s", cmd.Code)
	}
	server, err := domain.NewMCPServer(domain.NewMCPServerID(s.idGen.Generate()), cmd.Code, cmd.Name, cmd.TransportType)
	if err != nil {
		return nil, err
	}
	server.UpdateProfile(cmd.Name, cmd.Description, cmd.TransportType, cmd.Command, cmd.URL, cmd.Args, cmd.EnvVars)
	if err := s.mcpServerRepo.Create(ctx, server); err != nil {
		return nil, err
	}
	return server, nil
}

func (s *MCPApplicationService) GetServer(ctx context.Context, id domain.MCPServerID) (*domain.MCPServer, error) {
	return s.mcpServerRepo.GetByID(ctx, id)
}

func (s *MCPApplicationService) ListServers(ctx context.Context) ([]*domain.MCPServer, error) {
	return s.mcpServerRepo.List(ctx)
}

func (s *MCPApplicationService) UpdateServer(ctx context.Context, cmd UpdateMCPServerCommand) (*domain.MCPServer, error) {
	server, err := s.mcpServerRepo.GetByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, errors.New("MCP 服务器不存在")
	}
	name := server.Name()
	desc := server.Description()
	trans := server.TransportType()
	command := server.Command()
	args := server.Args()
	url := server.URL()
	env := server.EnvVars()
	if cmd.Name != nil {
		name = *cmd.Name
	}
	if cmd.Description != nil {
		desc = *cmd.Description
	}
	if cmd.TransportType != nil {
		trans = *cmd.TransportType
	}
	if cmd.Command != nil {
		command = *cmd.Command
	}
	if cmd.Args != nil {
		args = *cmd.Args
	}
	if cmd.URL != nil {
		url = *cmd.URL
	}
	if cmd.EnvVars != nil {
		env = *cmd.EnvVars
	}
	server.UpdateProfile(name, desc, trans, command, url, args, env)
	if err := s.mcpServerRepo.Update(ctx, server); err != nil {
		return nil, err
	}
	return server, nil
}

func (s *MCPApplicationService) DeleteServer(ctx context.Context, id domain.MCPServerID) error {
	// 清理工具
	if err := s.mcpToolRepo.DeleteByServerID(ctx, id); err != nil {
		return fmt.Errorf("failed to delete tools: %w", err)
	}
	return s.mcpServerRepo.Delete(ctx, id)
}

// Test connection
func (s *MCPApplicationService) TestServer(ctx context.Context, id domain.MCPServerID) error {
	server, err := s.mcpServerRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if server == nil {
		return errors.New("MCP 服务器不存在")
	}
	cli, err := s.clientFactory.CreateClient(server)
	if err != nil {
		server.SetStatus("error", fmt.Sprintf("创建客户端失败: %v", err))
		if updateErr := s.mcpServerRepo.Update(ctx, server); updateErr != nil {
			log.Printf("failed to update server status: %v", updateErr)
		}
		return err
	}
	defer cli.Close()
	ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := cli.Start(ctx2); err != nil {
		server.SetStatus("error", fmt.Sprintf("启动失败: %v", err))
		if updateErr := s.mcpServerRepo.Update(ctx, server); updateErr != nil {
			log.Printf("failed to update server status: %v", updateErr)
		}
		return err
	}
	err = cli.Initialize(ctx2)
	if err != nil {
		server.SetStatus("error", fmt.Sprintf("初始化失败: %v", err))
		if updateErr := s.mcpServerRepo.Update(ctx, server); updateErr != nil {
			log.Printf("failed to update server status: %v", updateErr)
		}
		return err
	}
	server.SetStatus("active", "")
	return s.mcpServerRepo.Update(ctx, server)
}

// Refresh tools
func (s *MCPApplicationService) RefreshCapabilities(ctx context.Context, id domain.MCPServerID) error {
	server, err := s.mcpServerRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if server == nil {
		return errors.New("MCP 服务器不存在")
	}
	cli, err := s.clientFactory.CreateClient(server)
	if err != nil {
		server.SetStatus("error", fmt.Sprintf("创建客户端失败: %v", err))
		if updateErr := s.mcpServerRepo.Update(ctx, server); updateErr != nil {
			log.Printf("failed to update server status: %v", updateErr)
		}
		return err
	}
	defer cli.Close()
	ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := cli.Start(ctx2); err != nil {
		server.SetStatus("error", fmt.Sprintf("启动失败: %v", err))
		if updateErr := s.mcpServerRepo.Update(ctx, server); updateErr != nil {
			log.Printf("failed to update server status: %v", updateErr)
		}
		return err
	}
	err = cli.Initialize(ctx2)
	if err != nil {
		server.SetStatus("error", fmt.Sprintf("初始化失败: %v", err))
		if updateErr := s.mcpServerRepo.Update(ctx, server); updateErr != nil {
			log.Printf("failed to update server status: %v", updateErr)
		}
		return err
	}
	tools, err := cli.ListTools(ctx2)
	if err != nil {
		server.SetStatus("error", fmt.Sprintf("获取工具列表失败: %v", err))
		if updateErr := s.mcpServerRepo.Update(ctx, server); updateErr != nil {
			log.Printf("failed to update server status: %v", updateErr)
		}
		return err
	}
	// clear and save
	if err := s.mcpToolRepo.DeleteByServerID(ctx, server.ID()); err != nil {
		return err
	}
	capabilities := make([]domain.MCPTool, 0, len(tools))
	for _, t := range tools {
		var schema map[string]interface{}
		if t.Schema != nil {
			schema = t.Schema
		}
		capabilities = append(capabilities, domain.MCPTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		})
		toolModel := &domain.MCPToolModel{
			ID:          s.idGen.Generate(),
			MCPServerID: server.ID(),
			Name:        t.Name,
			Description: t.Description,
			InputSchema: domain.EncodeAny(schema),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := s.mcpToolRepo.Create(ctx, toolModel); err != nil {
			return fmt.Errorf("保存工具失败: %w", err)
		}
	}
	server.SetCapabilities(capabilities)
	server.SetStatus("active", "")
	return s.mcpServerRepo.Update(ctx, server)
}

func (s *MCPApplicationService) ListTools(ctx context.Context, id domain.MCPServerID) ([]*domain.MCPToolModel, error) {
	return s.mcpToolRepo.ListByServerID(ctx, id)
}

// Agent bindings
func (s *MCPApplicationService) ListAgentBindings(ctx context.Context, agentID domain.AgentID) ([]*domain.AgentMCPBinding, error) {
	return s.bindingRepo.GetByAgentID(ctx, agentID)
}

func (s *MCPApplicationService) CreateAgentBinding(ctx context.Context, cmd CreateAgentMCPBindingCommand) (*domain.AgentMCPBinding, error) {
	agent, err := s.agentRepo.FindByID(ctx, cmd.AgentID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, errors.New("Agent 不存在")
	}
	exists, err := s.bindingRepo.CheckExists(ctx, cmd.AgentID, cmd.MCPServerID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("Agent 已绑定该 MCP 服务器")
	}
	binding := domain.NewAgentMCPBinding(domain.NewAgentMCPBindingID(s.idGen.Generate()), cmd.AgentID, cmd.MCPServerID)
	if cmd.EnabledTools != nil {
		binding.SetEnabledTools(cmd.EnabledTools)
	}
	if cmd.IsActive != nil {
		binding.SetActive(*cmd.IsActive)
	}
	if cmd.AutoLoad != nil {
		binding.SetAutoLoad(*cmd.AutoLoad)
	}
	if err := s.bindingRepo.Create(ctx, binding); err != nil {
		return nil, err
	}
	return binding, nil
}

func (s *MCPApplicationService) UpdateAgentBinding(ctx context.Context, cmd UpdateAgentMCPBindingCommand) (*domain.AgentMCPBinding, error) {
	binding, err := s.bindingRepo.GetByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, errors.New("绑定不存在")
	}
	if cmd.EnabledTools != nil {
		binding.SetEnabledTools(*cmd.EnabledTools)
	}
	if cmd.IsActive != nil {
		binding.SetActive(*cmd.IsActive)
	}
	if cmd.AutoLoad != nil {
		binding.SetAutoLoad(*cmd.AutoLoad)
	}
	if err := s.bindingRepo.Update(ctx, binding); err != nil {
		return nil, err
	}
	return binding, nil
}

func (s *MCPApplicationService) DeleteAgentBinding(ctx context.Context, id domain.AgentMCPBindingID) error {
	return s.bindingRepo.Delete(ctx, id)
}

func (s *MCPApplicationService) GetAgentMCPTools(ctx context.Context, agentID domain.AgentID) ([]domain.MCPTool, error) {
	bindings, err := s.bindingRepo.GetByAgentID(ctx, agentID)
	if err != nil {
		return nil, err
	}
	var result []domain.MCPTool
	for _, b := range bindings {
		if !b.IsActive() {
			continue
		}
		server, err := s.mcpServerRepo.GetByID(ctx, b.MCPServerID())
		if err != nil || server == nil || server.Status() != "active" {
			continue
		}
		enabled := b.EnabledTools()
		for _, tool := range server.Capabilities() {
			if enabled == nil || contains(enabled, tool.Name) {
				result = append(result, tool)
			}
		}
	}
	return result, nil
}

func contains(arr []string, v string) bool {
	for _, s := range arr {
		if s == v {
			return true
		}
	}
	return false
}

// call tool – optional, used for debug
func (s *MCPApplicationService) ExecuteTool(ctx context.Context, serverID domain.MCPServerID, toolName string, params map[string]interface{}) (string, error) {
	server, err := s.mcpServerRepo.GetByID(ctx, serverID)
	if err != nil {
		return "", err
	}
	if server == nil || server.Status() != "active" {
		return "", fmt.Errorf("MCP 服务器不可用")
	}
	cli, err := s.clientFactory.CreateClient(server)
	if err != nil {
		return "", err
	}
	defer cli.Close()
	ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := cli.Start(ctx2); err != nil {
		return "", err
	}
	err = cli.Initialize(ctx2)
	if err != nil {
		return "", err
	}
	result, err := cli.CallTool(ctx2, toolName, params)
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

