package application

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/weibh/taskmanager/domain"
)

// ---------- mocks ----------

type mockMCPIDGen struct {
	count int
}

func (m *mockMCPIDGen) Generate() string {
	m.count++
	return "mcp-id-" + strconv.Itoa(m.count)
}

type mockMCPServerRepo struct {
	servers            map[string]*domain.MCPServer
	checkCodeExistsErr error
	createErr          error
	getByIDErr         error
	updateErr          error
	deleteErr          error
	listErr            error
}

func newMockMCPServerRepo() *mockMCPServerRepo {
	return &mockMCPServerRepo{servers: make(map[string]*domain.MCPServer)}
}

func (m *mockMCPServerRepo) Create(ctx context.Context, server *domain.MCPServer) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.servers[server.ID().String()] = server
	return nil
}

func (m *mockMCPServerRepo) Update(ctx context.Context, server *domain.MCPServer) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.servers[server.ID().String()] = server
	return nil
}

func (m *mockMCPServerRepo) Delete(ctx context.Context, id domain.MCPServerID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.servers, id.String())
	return nil
}

func (m *mockMCPServerRepo) GetByID(ctx context.Context, id domain.MCPServerID) (*domain.MCPServer, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	return m.servers[id.String()], nil
}

func (m *mockMCPServerRepo) GetByCode(ctx context.Context, code string) (*domain.MCPServer, error) {
	for _, s := range m.servers {
		if s.Code() == code {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockMCPServerRepo) List(ctx context.Context) ([]*domain.MCPServer, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*domain.MCPServer
	for _, s := range m.servers {
		result = append(result, s)
	}
	return result, nil
}

func (m *mockMCPServerRepo) ListByStatus(ctx context.Context, status string) ([]*domain.MCPServer, error) {
	var result []*domain.MCPServer
	for _, s := range m.servers {
		if s.Status() == status {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockMCPServerRepo) CheckCodeExists(ctx context.Context, code string) (bool, error) {
	if m.checkCodeExistsErr != nil {
		return false, m.checkCodeExistsErr
	}
	for _, s := range m.servers {
		if s.Code() == code {
			return true, nil
		}
	}
	return false, nil
}

type mockMCPToolRepo struct {
	tools              map[string]*domain.MCPToolModel
	serverTools        map[string][]*domain.MCPToolModel
	deleteByServerIDErr error
	createErr          error
	listByServerIDErr  error
}

func newMockMCPToolRepo() *mockMCPToolRepo {
	return &mockMCPToolRepo{
		tools:       make(map[string]*domain.MCPToolModel),
		serverTools: make(map[string][]*domain.MCPToolModel),
	}
}

func (m *mockMCPToolRepo) Create(ctx context.Context, tool *domain.MCPToolModel) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.tools[tool.ID] = tool
	m.serverTools[tool.MCPServerID.String()] = append(m.serverTools[tool.MCPServerID.String()], tool)
	return nil
}

func (m *mockMCPToolRepo) DeleteByServerID(ctx context.Context, serverID domain.MCPServerID) error {
	if m.deleteByServerIDErr != nil {
		return m.deleteByServerIDErr
	}
	delete(m.serverTools, serverID.String())
	for k, v := range m.tools {
		if v.MCPServerID.String() == serverID.String() {
			delete(m.tools, k)
		}
	}
	return nil
}

func (m *mockMCPToolRepo) ListByServerID(ctx context.Context, serverID domain.MCPServerID) ([]*domain.MCPToolModel, error) {
	if m.listByServerIDErr != nil {
		return nil, m.listByServerIDErr
	}
	return m.serverTools[serverID.String()], nil
}

type mockAgentMCPBindingRepo struct {
	bindings          map[string]*domain.AgentMCPBinding
	getByIDErr        error
	getByAgentIDErr   error
	createErr         error
	updateErr         error
	deleteErr         error
	checkExistsErr    error
}

func newMockAgentMCPBindingRepo() *mockAgentMCPBindingRepo {
	return &mockAgentMCPBindingRepo{bindings: make(map[string]*domain.AgentMCPBinding)}
}

func (m *mockAgentMCPBindingRepo) Create(ctx context.Context, binding *domain.AgentMCPBinding) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.bindings[binding.ID().String()] = binding
	return nil
}

func (m *mockAgentMCPBindingRepo) Update(ctx context.Context, binding *domain.AgentMCPBinding) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.bindings[binding.ID().String()] = binding
	return nil
}

func (m *mockAgentMCPBindingRepo) Delete(ctx context.Context, id domain.AgentMCPBindingID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.bindings, id.String())
	return nil
}

func (m *mockAgentMCPBindingRepo) DeleteByAgentAndMCPServer(ctx context.Context, agentID domain.AgentID, serverID domain.MCPServerID) error {
	for k, v := range m.bindings {
		if v.AgentID().String() == agentID.String() && v.MCPServerID().String() == serverID.String() {
			delete(m.bindings, k)
		}
	}
	return nil
}

func (m *mockAgentMCPBindingRepo) GetByID(ctx context.Context, id domain.AgentMCPBindingID) (*domain.AgentMCPBinding, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	return m.bindings[id.String()], nil
}

func (m *mockAgentMCPBindingRepo) GetByAgentID(ctx context.Context, agentID domain.AgentID) ([]*domain.AgentMCPBinding, error) {
	if m.getByAgentIDErr != nil {
		return nil, m.getByAgentIDErr
	}
	var result []*domain.AgentMCPBinding
	for _, b := range m.bindings {
		if b.AgentID().String() == agentID.String() {
			result = append(result, b)
		}
	}
	return result, nil
}

func (m *mockAgentMCPBindingRepo) CheckExists(ctx context.Context, agentID domain.AgentID, serverID domain.MCPServerID) (bool, error) {
	if m.checkExistsErr != nil {
		return false, m.checkExistsErr
	}
	for _, b := range m.bindings {
		if b.AgentID().String() == agentID.String() && b.MCPServerID().String() == serverID.String() {
			return true, nil
		}
	}
	return false, nil
}

type mockMCPToolLogRepo struct{}

func (m *mockMCPToolLogRepo) Create(ctx context.Context, log *domain.MCPToolLog) error { return nil }
func (m *mockMCPToolLogRepo) ListByServerID(ctx context.Context, serverID domain.MCPServerID, limit int) ([]*domain.MCPToolLog, error) {
	return nil, nil
}

func setupTestMCPService() *MCPApplicationService {
	return NewMCPApplicationService(
		newMockMCPServerRepo(),
		newMockAgentRepo(),
		newMockAgentMCPBindingRepo(),
		newMockMCPToolRepo(),
		&mockMCPToolLogRepo{},
		&mockAgentIDGen{},
	)
}

func setupTestMCPServiceWithAgent() (*MCPApplicationService, *domain.Agent) {
	ctx := context.Background()
	agentRepo := newMockAgentRepo()
	idGen := &mockAgentIDGen{}
	agentSvc := NewAgentApplicationService(agentRepo, idGen)
	agent, _ := agentSvc.CreateAgent(ctx, CreateAgentCommand{
		UserCode: "usr_001",
		Name:     "TestAgent",
	})
	svc := NewMCPApplicationService(
		newMockMCPServerRepo(),
		agentRepo,
		newMockAgentMCPBindingRepo(),
		newMockMCPToolRepo(),
		&mockMCPToolLogRepo{},
		idGen,
	)
	return svc, agent
}

// ---------- tests: CreateServer ----------

func TestMCPService_CreateServer(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	server, err := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_test",
		Name:          "测试 MCP 服务器",
		Description:   "desc",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "node",
		Args:          []string{"server.js"},
		EnvVars:       map[string]string{"KEY": "VAL"},
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if server.Code() != "mcp_test" {
		t.Errorf("期望 code 为 'mcp_test', 实际为 '%s'", server.Code())
	}
	if server.Name() != "测试 MCP 服务器" {
		t.Errorf("期望 name 为 '测试 MCP 服务器', 实际为 '%s'", server.Name())
	}
	if server.TransportType() != domain.MCPTransportSTDIO {
		t.Errorf("期望 transport 为 'stdio', 实际为 '%s'", server.TransportType())
	}
	if server.Command() != "node" {
		t.Errorf("期望 command 为 'node', 实际为 '%s'", server.Command())
	}
}

func TestMCPService_CreateServer_CodeExists(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	_, err := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_dup",
		Name:          "First",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "cmd",
	})
	if err != nil {
		t.Fatalf("第一次创建应该成功: %v", err)
	}

	_, err = svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_dup",
		Name:          "Second",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "cmd",
	})
	if err == nil {
		t.Fatal("期望重复 code 返回错误")
	}
}

func TestMCPService_CreateServer_InvalidParams(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	_, err := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_bad",
		Name:          "",
		TransportType: domain.MCPTransportSTDIO,
	})
	if err == nil {
		t.Fatal("期望空 name 返回错误")
	}
}

// ---------- tests: GetServer / ListServers ----------

func TestMCPService_GetServer(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	created, _ := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_get",
		Name:          "GetServer",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "cmd",
	})

	server, err := svc.GetServer(ctx, created.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if server.Name() != "GetServer" {
		t.Errorf("期望 name 为 'GetServer', 实际为 '%s'", server.Name())
	}
}

func TestMCPService_ListServers(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	svc.CreateServer(ctx, CreateMCPServerCommand{Code: "mcp_1", Name: "S1", TransportType: domain.MCPTransportSTDIO, Command: "cmd"})
	svc.CreateServer(ctx, CreateMCPServerCommand{Code: "mcp_2", Name: "S2", TransportType: domain.MCPTransportHTTP, URL: "http://a"})
	svc.CreateServer(ctx, CreateMCPServerCommand{Code: "mcp_3", Name: "S3", TransportType: domain.MCPTransportSSE, URL: "http://b"})

	servers, err := svc.ListServers(ctx)
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if len(servers) != 3 {
		t.Errorf("期望 3 个 servers, 实际为 %d", len(servers))
	}
}

// ---------- tests: UpdateServer ----------

func TestMCPService_UpdateServer(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	created, _ := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_upd",
		Name:          "Original",
		Description:   "OldDesc",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "old_cmd",
		URL:           "",
	})

	newName := "Updated"
	newDesc := "NewDesc"
	newTransport := domain.MCPTransportHTTP
	newCommand := "new_cmd"
	newURL := "http://localhost"
	newArgs := []string{"--port", "8080"}
	newEnv := map[string]string{"FOO": "BAR"}

	updated, err := svc.UpdateServer(ctx, UpdateMCPServerCommand{
		ID:            created.ID(),
		Name:          &newName,
		Description:   &newDesc,
		TransportType: &newTransport,
		Command:       &newCommand,
		Args:          &newArgs,
		URL:           &newURL,
		EnvVars:       &newEnv,
	})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if updated.Name() != "Updated" {
		t.Errorf("期望 name 为 'Updated', 实际为 '%s'", updated.Name())
	}
	if updated.Description() != "NewDesc" {
		t.Errorf("期望 description 为 'NewDesc', 实际为 '%s'", updated.Description())
	}
	if updated.TransportType() != domain.MCPTransportHTTP {
		t.Errorf("期望 transport 为 'http', 实际为 '%s'", updated.TransportType())
	}
	if updated.Command() != "new_cmd" {
		t.Errorf("期望 command 为 'new_cmd', 实际为 '%s'", updated.Command())
	}
	if updated.URL() != "http://localhost" {
		t.Errorf("期望 url 为 'http://localhost', 实际为 '%s'", updated.URL())
	}
}

func TestMCPService_UpdateServer_NotFound(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	newName := "Updated"
	_, err := svc.UpdateServer(ctx, UpdateMCPServerCommand{
		ID:   domain.NewMCPServerID("non-existent"),
		Name: &newName,
	})
	if err == nil || err.Error() != "MCP 服务器不存在" {
		t.Errorf("期望 'MCP 服务器不存在' 错误, 实际为 %v", err)
	}
}

// ---------- tests: DeleteServer ----------

func TestMCPService_DeleteServer(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	created, _ := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_del",
		Name:          "ToDelete",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "cmd",
	})

	// pre-create a tool for this server
	svc.mcpToolRepo.(*mockMCPToolRepo).Create(ctx, &domain.MCPToolModel{
		ID:          "tool-1",
		MCPServerID: created.ID(),
		Name:        "tool1",
	})

	err := svc.DeleteServer(ctx, created.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	_, err = svc.GetServer(ctx, created.ID())
	if err != nil {
		t.Fatalf("获取时出错了: %v", err)
	}
	server, _ := svc.GetServer(ctx, created.ID())
	if server != nil {
		t.Error("删除后服务器应该不存在")
	}
}

func TestMCPService_DeleteServer_DeleteToolsError(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	created, _ := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_del_err",
		Name:          "ToDelete",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "cmd",
	})

	svc.mcpToolRepo.(*mockMCPToolRepo).deleteByServerIDErr = errors.New("db error")

	err := svc.DeleteServer(ctx, created.ID())
	if err == nil {
		t.Fatal("期望 delete tools 错误返回")
	}
}

// ---------- tests: TestServer ----------

func TestMCPService_TestServer_NotFound(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	err := svc.TestServer(ctx, domain.NewMCPServerID("not-found"))
	if err == nil || err.Error() != "MCP 服务器不存在" {
		t.Errorf("期望 'MCP 服务器不存在' 错误, 实际为 %v", err)
	}
}

func TestMCPService_TestServer_UnsupportedTransport(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	// 手动创建 server 并注入不正确的 transport
	server, _ := domain.NewMCPServer(domain.NewMCPServerID("mcp-test-1"), "mcp_tt", "TT", domain.MCPTransportSTDIO)
	server.UpdateProfile("TT", "", "bad_transport", "", "", nil, nil)
	svc.mcpServerRepo.(*mockMCPServerRepo).servers["mcp-test-1"] = server

	err := svc.TestServer(ctx, domain.NewMCPServerID("mcp-test-1"))
	if err == nil {
		t.Fatal("期望不支持的 transport 返回错误")
	}
	if server.Status() != "error" {
		t.Errorf("期望 status 为 'error', 实际为 '%s'", server.Status())
	}
}

// ---------- tests: RefreshCapabilities ----------

func TestMCPService_RefreshCapabilities_NotFound(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	err := svc.RefreshCapabilities(ctx, domain.NewMCPServerID("not-found"))
	if err == nil || err.Error() != "MCP 服务器不存在" {
		t.Errorf("期望 'MCP 服务器不存在' 错误, 实际为 %v", err)
	}
}

func TestMCPService_RefreshCapabilities_UnsupportedTransport(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	server, _ := domain.NewMCPServer(domain.NewMCPServerID("mcp-ref-1"), "mcp_ref", "Ref", domain.MCPTransportSTDIO)
	server.UpdateProfile("Ref", "", "bad_transport", "", "", nil, nil)
	svc.mcpServerRepo.(*mockMCPServerRepo).servers["mcp-ref-1"] = server

	err := svc.RefreshCapabilities(ctx, domain.NewMCPServerID("mcp-ref-1"))
	if err == nil {
		t.Fatal("期望不支持的 transport 返回错误")
	}
	if server.Status() != "error" {
		t.Errorf("期望 status 为 'error', 实际为 '%s'", server.Status())
	}
}

// ---------- tests: ListTools ----------

func TestMCPService_ListTools(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	server, _ := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_tools",
		Name:          "ToolServer",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "cmd",
	})

	svc.mcpToolRepo.(*mockMCPToolRepo).Create(ctx, &domain.MCPToolModel{
		ID:          "t1",
		MCPServerID: server.ID(),
		Name:        "toolA",
	})
	svc.mcpToolRepo.(*mockMCPToolRepo).Create(ctx, &domain.MCPToolModel{
		ID:          "t2",
		MCPServerID: server.ID(),
		Name:        "toolB",
	})

	tools, err := svc.ListTools(ctx, server.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if len(tools) != 2 {
		t.Errorf("期望 2 个 tools, 实际为 %d", len(tools))
	}
}

// ---------- tests: Agent Binding ----------

func TestMCPService_CreateAgentBinding(t *testing.T) {
	svc, agent := setupTestMCPServiceWithAgent()
	ctx := context.Background()

	server, _ := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_bind",
		Name:          "BindServer",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "cmd",
	})

	binding, err := svc.CreateAgentBinding(ctx, CreateAgentMCPBindingCommand{
		AgentID:      agent.ID(),
		MCPServerID:  server.ID(),
		EnabledTools: []string{"tool1", "tool2"},
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if binding.AgentID().String() != agent.ID().String() {
		t.Errorf("期望 agent id 匹配")
	}
	if binding.MCPServerID().String() != server.ID().String() {
		t.Errorf("期望 server id 匹配")
	}
	if len(binding.EnabledTools()) != 2 || binding.EnabledTools()[0] != "tool1" {
		t.Errorf("期望 enabled tools 为 ['tool1', 'tool2'], 实际为 %v", binding.EnabledTools())
	}
	if !binding.IsActive() {
		t.Error("新绑定应该默认激活")
	}
}

func TestMCPService_CreateAgentBinding_AgentNotFound(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	server, _ := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_bind2",
		Name:          "BindServer",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "cmd",
	})

	_, err := svc.CreateAgentBinding(ctx, CreateAgentMCPBindingCommand{
		AgentID:     domain.NewAgentID("not-exist"),
		MCPServerID: server.ID(),
	})
	if err == nil || err.Error() != "Agent 不存在" {
		t.Errorf("期望 'Agent 不存在' 错误, 实际为 %v", err)
	}
}

func TestMCPService_CreateAgentBinding_AlreadyExists(t *testing.T) {
	svc, agent := setupTestMCPServiceWithAgent()
	ctx := context.Background()

	server, _ := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_bind3",
		Name:          "BindServer",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "cmd",
	})

	_, err := svc.CreateAgentBinding(ctx, CreateAgentMCPBindingCommand{
		AgentID:     agent.ID(),
		MCPServerID: server.ID(),
	})
	if err != nil {
		t.Fatalf("第一次绑定应成功: %v", err)
	}

	_, err = svc.CreateAgentBinding(ctx, CreateAgentMCPBindingCommand{
		AgentID:     agent.ID(),
		MCPServerID: server.ID(),
	})
	if err == nil || err.Error() != "Agent 已绑定该 MCP 服务器" {
		t.Errorf("期望重复绑定错误, 实际为 %v", err)
	}
}

func TestMCPService_UpdateAgentBinding(t *testing.T) {
	svc, agent := setupTestMCPServiceWithAgent()
	ctx := context.Background()

	server, _ := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_updbind",
		Name:          "BindServer",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "cmd",
	})

	binding, _ := svc.CreateAgentBinding(ctx, CreateAgentMCPBindingCommand{
		AgentID:     agent.ID(),
		MCPServerID: server.ID(),
	})

	newTools := []string{"tool3"}
	isActive := false
	autoLoad := true

	updated, err := svc.UpdateAgentBinding(ctx, UpdateAgentMCPBindingCommand{
		ID:           binding.ID(),
		EnabledTools: &newTools,
		IsActive:     &isActive,
		AutoLoad:     &autoLoad,
	})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if updated.IsActive() {
		t.Error("期望更新后 binding 为非激活")
	}
	if !updated.AutoLoad() {
		t.Error("期望更新后 binding autoLoad 为 true")
	}
	if len(updated.EnabledTools()) != 1 || updated.EnabledTools()[0] != "tool3" {
		t.Errorf("期望 enabled tools 为 ['tool3'], 实际为 %v", updated.EnabledTools())
	}
}

func TestMCPService_UpdateAgentBinding_NotFound(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	newTools := []string{"tool3"}
	_, err := svc.UpdateAgentBinding(ctx, UpdateAgentMCPBindingCommand{
		ID:           domain.NewAgentMCPBindingID("not-found"),
		EnabledTools: &newTools,
	})
	if err == nil || err.Error() != "绑定不存在" {
		t.Errorf("期望 '绑定不存在' 错误, 实际为 %v", err)
	}
}

func TestMCPService_DeleteAgentBinding(t *testing.T) {
	svc, agent := setupTestMCPServiceWithAgent()
	ctx := context.Background()

	server, _ := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_delbind",
		Name:          "BindServer",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "cmd",
	})

	binding, _ := svc.CreateAgentBinding(ctx, CreateAgentMCPBindingCommand{
		AgentID:     agent.ID(),
		MCPServerID: server.ID(),
	})

	err := svc.DeleteAgentBinding(ctx, binding.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	bindings, _ := svc.ListAgentBindings(ctx, agent.ID())
	if len(bindings) != 0 {
		t.Errorf("期望删除后 bindings 为空, 实际为 %d", len(bindings))
	}
}

// ---------- tests: GetAgentMCPTools ----------

func TestMCPService_GetAgentMCPTools(t *testing.T) {
	svc, agent := setupTestMCPServiceWithAgent()
	ctx := context.Background()

	server, _ := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_toolserver",
		Name:          "ToolServer",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "cmd",
	})
	// set server active and capabilities
	server.SetStatus("active", "")
	server.SetCapabilities([]domain.MCPTool{
		{Name: "toolA", Description: "descA"},
		{Name: "toolB", Description: "descB"},
		{Name: "toolC", Description: "descC"},
	})
	svc.mcpServerRepo.(*mockMCPServerRepo).servers[server.ID().String()] = server

	// binding with subset of tools enabled
	_, _ = svc.CreateAgentBinding(ctx, CreateAgentMCPBindingCommand{
		AgentID:      agent.ID(),
		MCPServerID:  server.ID(),
		EnabledTools: []string{"toolA", "toolC"},
		IsActive:     boolPtr(true),
	})

	tools, err := svc.GetAgentMCPTools(ctx, agent.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if len(tools) != 2 {
		t.Errorf("期望 2 个 tools, 实际为 %d", len(tools))
	}
	for _, tool := range tools {
		if tool.Name != "toolA" && tool.Name != "toolC" {
			t.Errorf("不应该包含 tool %s", tool.Name)
		}
	}
}

func TestMCPService_GetAgentMCPTools_InactiveBinding(t *testing.T) {
	svc, agent := setupTestMCPServiceWithAgent()
	ctx := context.Background()

	server, _ := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_inactive",
		Name:          "Inactive",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "cmd",
	})
	server.SetStatus("active", "")
	server.SetCapabilities([]domain.MCPTool{{Name: "toolA"}})
	svc.mcpServerRepo.(*mockMCPServerRepo).servers[server.ID().String()] = server

	isActive := false
	_, _ = svc.CreateAgentBinding(ctx, CreateAgentMCPBindingCommand{
		AgentID:     agent.ID(),
		MCPServerID: server.ID(),
		IsActive:    &isActive,
	})

	tools, err := svc.GetAgentMCPTools(ctx, agent.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("期望 inactive binding 返回 0 个 tools, 实际为 %d", len(tools))
	}
}

func TestMCPService_GetAgentMCPTools_InactiveServer(t *testing.T) {
	svc, agent := setupTestMCPServiceWithAgent()
	ctx := context.Background()

	server, _ := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_srv_inactive",
		Name:          "InactiveServer",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "cmd",
	})
	server.SetStatus("inactive", "")
	server.SetCapabilities([]domain.MCPTool{{Name: "toolA"}})
	svc.mcpServerRepo.(*mockMCPServerRepo).servers[server.ID().String()] = server

	_, _ = svc.CreateAgentBinding(ctx, CreateAgentMCPBindingCommand{
		AgentID:     agent.ID(),
		MCPServerID: server.ID(),
		IsActive:    boolPtr(true),
	})

	tools, err := svc.GetAgentMCPTools(ctx, agent.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("期望 inactive server 返回 0 个 tools, 实际为 %d", len(tools))
	}
}

func TestMCPService_GetAgentMCPTools_NilEnabledTools(t *testing.T) {
	svc, agent := setupTestMCPServiceWithAgent()
	ctx := context.Background()

	server, _ := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_alltools",
		Name:          "AllTools",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "cmd",
	})
	server.SetStatus("active", "")
	server.SetCapabilities([]domain.MCPTool{
		{Name: "toolA"},
		{Name: "toolB"},
	})
	svc.mcpServerRepo.(*mockMCPServerRepo).servers[server.ID().String()] = server

	// binding with nil EnabledTools means all enabled
	_, _ = svc.CreateAgentBinding(ctx, CreateAgentMCPBindingCommand{
		AgentID:     agent.ID(),
		MCPServerID: server.ID(),
		IsActive:    boolPtr(true),
	})

	tools, err := svc.GetAgentMCPTools(ctx, agent.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if len(tools) != 2 {
		t.Errorf("期望返回全部 2 个 tools, 实际为 %d", len(tools))
	}
}

// ---------- table-driven tests for ListAgentBindings ----------

func TestMCPService_ListAgentBindings(t *testing.T) {
	svc, agent := setupTestMCPServiceWithAgent()
	ctx := context.Background()

	server1, _ := svc.CreateServer(ctx, CreateMCPServerCommand{Code: "mcp_bind1", Name: "S1", TransportType: domain.MCPTransportSTDIO, Command: "cmd"})
	server2, _ := svc.CreateServer(ctx, CreateMCPServerCommand{Code: "mcp_bind2", Name: "S2", TransportType: domain.MCPTransportSTDIO, Command: "cmd"})

	_, _ = svc.CreateAgentBinding(ctx, CreateAgentMCPBindingCommand{AgentID: agent.ID(), MCPServerID: server1.ID()})
	_, _ = svc.CreateAgentBinding(ctx, CreateAgentMCPBindingCommand{AgentID: agent.ID(), MCPServerID: server2.ID()})

	bindings, err := svc.ListAgentBindings(ctx, agent.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if len(bindings) != 2 {
		t.Errorf("期望 2 个 bindings, 实际为 %d", len(bindings))
	}
}


// ---------- additional tests for TestServer / RefreshCapabilities / ExecuteTool coverage ----------

func TestMCPService_TestServer_SSEStartFail(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	server, _ := domain.NewMCPServer(domain.NewMCPServerID("mcp-sse-start"), "mcp_sse_start", "SSEStart", domain.MCPTransportSSE)
	server.UpdateProfile("SSEStart", "", domain.MCPTransportSSE, "", "", nil, nil)
	svc.mcpServerRepo.(*mockMCPServerRepo).servers["mcp-sse-start"] = server

	err := svc.TestServer(ctx, domain.NewMCPServerID("mcp-sse-start"))
	if err == nil {
		t.Fatal("期望 SSE Start 失败返回错误")
	}
	if server.Status() != "error" {
		t.Errorf("期望 status 为 'error', 实际为 '%s'", server.Status())
	}
}

func TestMCPService_TestServer_HTTPInitFail(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	server, _ := domain.NewMCPServer(domain.NewMCPServerID("mcp-http-init"), "mcp_http_init", "HTTPInit", domain.MCPTransportHTTP)
	server.UpdateProfile("HTTPInit", "", domain.MCPTransportHTTP, "", "", nil, nil)
	svc.mcpServerRepo.(*mockMCPServerRepo).servers["mcp-http-init"] = server

	err := svc.TestServer(ctx, domain.NewMCPServerID("mcp-http-init"))
	if err == nil {
		t.Fatal("期望 HTTP Initialize 失败返回错误")
	}
	if server.Status() != "error" {
		t.Errorf("期望 status 为 'error', 实际为 '%s'", server.Status())
	}
}

func TestMCPService_RefreshCapabilities_SSEStartFail(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	server, _ := domain.NewMCPServer(domain.NewMCPServerID("mcp-ref-sse"), "mcp_ref_sse", "RefSSE", domain.MCPTransportSSE)
	server.UpdateProfile("RefSSE", "", domain.MCPTransportSSE, "", "", nil, nil)
	svc.mcpServerRepo.(*mockMCPServerRepo).servers["mcp-ref-sse"] = server

	err := svc.RefreshCapabilities(ctx, domain.NewMCPServerID("mcp-ref-sse"))
	if err == nil {
		t.Fatal("期望 SSE Start 失败返回错误")
	}
	if server.Status() != "error" {
		t.Errorf("期望 status 为 'error', 实际为 '%s'", server.Status())
	}
}

func TestMCPService_RefreshCapabilities_HTTPInitFail(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	server, _ := domain.NewMCPServer(domain.NewMCPServerID("mcp-ref-http"), "mcp_ref_http", "RefHTTP", domain.MCPTransportHTTP)
	server.UpdateProfile("RefHTTP", "", domain.MCPTransportHTTP, "", "", nil, nil)
	svc.mcpServerRepo.(*mockMCPServerRepo).servers["mcp-ref-http"] = server

	err := svc.RefreshCapabilities(ctx, domain.NewMCPServerID("mcp-ref-http"))
	if err == nil {
		t.Fatal("期望 HTTP Initialize 失败返回错误")
	}
	if server.Status() != "error" {
		t.Errorf("期望 status 为 'error', 实际为 '%s'", server.Status())
	}
}

func TestMCPService_ExecuteTool_ServerNotFound(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	_, err := svc.ExecuteTool(ctx, domain.NewMCPServerID("not-found"), "tool1", nil)
	if err == nil || err.Error() != "MCP 服务器不可用" {
		t.Errorf("期望 'MCP 服务器不可用' 错误, 实际为 %v", err)
	}
}

func TestMCPService_ExecuteTool_ServerInactive(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	server, _ := svc.CreateServer(ctx, CreateMCPServerCommand{
		Code:          "mcp_exec_inactive",
		Name:          "ExecInactive",
		TransportType: domain.MCPTransportSTDIO,
		Command:       "cmd",
	})

	_, err := svc.ExecuteTool(ctx, server.ID(), "tool1", nil)
	if err == nil || err.Error() != "MCP 服务器不可用" {
		t.Errorf("期望 'MCP 服务器不可用' 错误, 实际为 %v", err)
	}
}

func TestMCPService_ExecuteTool_UnsupportedTransport(t *testing.T) {
	svc := setupTestMCPService()
	ctx := context.Background()

	server, _ := domain.NewMCPServer(domain.NewMCPServerID("mcp-exec"), "mcp_exec", "Exec", domain.MCPTransportSTDIO)
	server.UpdateProfile("Exec", "", "bad_transport", "", "", nil, nil)
	server.SetStatus("active", "")
	svc.mcpServerRepo.(*mockMCPServerRepo).servers["mcp-exec"] = server

	_, err := svc.ExecuteTool(ctx, domain.NewMCPServerID("mcp-exec"), "tool1", nil)
	if err == nil {
		t.Fatal("期望不支持的 transport 返回错误")
	}
}
