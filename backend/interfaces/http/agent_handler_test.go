/**
 * Agent Handler 测试
 */
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type mockAgentRepository struct {
	agents map[string]*domain.Agent
}

func newMockAgentRepository() *mockAgentRepository {
	return &mockAgentRepository{
		agents: make(map[string]*domain.Agent),
	}
}

func (m *mockAgentRepository) Save(ctx context.Context, agent *domain.Agent) error {
	m.agents[agent.ID().String()] = agent
	return nil
}

func (m *mockAgentRepository) FindByID(ctx context.Context, id domain.AgentID) (*domain.Agent, error) {
	agent, ok := m.agents[id.String()]
	if !ok {
		return nil, nil
	}
	return agent, nil
}

func (m *mockAgentRepository) FindAll(ctx context.Context) ([]*domain.Agent, error) {
	var result []*domain.Agent
	for _, agent := range m.agents {
		result = append(result, agent)
	}
	return result, nil
}

func (m *mockAgentRepository) FindByUserCode(ctx context.Context, userCode string) ([]*domain.Agent, error) {
	var result []*domain.Agent
	for _, agent := range m.agents {
		if agent.UserCode() == userCode {
			result = append(result, agent)
		}
	}
	return result, nil
}

func (m *mockAgentRepository) FindByAgentCode(ctx context.Context, code domain.AgentCode) (*domain.Agent, error) {
	for _, agent := range m.agents {
		if agent.AgentCode().String() == code.String() {
			return agent, nil
		}
	}
	return nil, nil
}

func (m *mockAgentRepository) Delete(ctx context.Context, id domain.AgentID) error {
	delete(m.agents, id.String())
	return nil
}

type mockAgentIDGenerator struct {
	count int
}

func (m *mockAgentIDGenerator) Generate() string {
	m.count++
	return "agent-id-" + string(rune('0'+m.count))
}

func setupTestAgentHandler() (*AgentHandler, *http.ServeMux) {
	repo := newMockAgentRepository()
	idGen := &mockAgentIDGenerator{}

	agentService := application.NewAgentApplicationService(repo, idGen)
	handler := NewAgentHandler(agentService)
	mux := SetupAgentRoutes(handler)

	return handler, mux
}

func SetupAgentRoutes(handler *AgentHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/agents", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.ListAgents(w, r)
		case http.MethodPost:
			handler.CreateAgent(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v1/agents/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.GetAgent(w, r)
		case http.MethodPut:
			handler.UpdateAgent(w, r)
		case http.MethodDelete:
			handler.DeleteAgent(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	return mux
}

func TestCreateAgent(t *testing.T) {
	_, mux := setupTestAgentHandler()

	body := `{
		"user_code": "usr_001",
		"name": "测试Agent",
		"description": "测试描述",
		"model": "gpt-4"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusCreated, w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["id"] == nil || resp["id"] == "" {
		t.Error("响应中应该包含 id")
	}

	if resp["agent_code"] == nil || resp["agent_code"] == "" {
		t.Error("响应中应该包含 agent_code")
	}

	if resp["name"] != "测试Agent" {
		t.Errorf("期望 name 为 '测试Agent', 实际为 '%v'", resp["name"])
	}

	if resp["user_code"] != "usr_001" {
		t.Errorf("期望 user_code 为 'usr_001', 实际为 '%v'", resp["user_code"])
	}
}

func TestCreateAgent_InvalidJSON(t *testing.T) {
	_, mux := setupTestAgentHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateAgent_EmptyName(t *testing.T) {
	_, mux := setupTestAgentHandler()

	body := `{
		"user_code": "usr_001",
		"name": "",
		"model": "gpt-4"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestListAgents(t *testing.T) {
	_, mux := setupTestAgentHandler()

	// 先创建一个 agent
	createBody := `{"user_code": "usr_001", "name": "Agent1", "model": "gpt-4"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	// 列出所有 agents
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	listW := httptest.NewRecorder()

	mux.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, listW.Code)
	}

	var resp []map[string]interface{}
	json.Unmarshal(listW.Body.Bytes(), &resp)

	if len(resp) < 1 {
		t.Error("期望至少有一个 agent")
	}
}

func TestListAgents_FilterByUserCode(t *testing.T) {
	_, mux := setupTestAgentHandler()

	// 创建两个不同 user_code 的 agents
	createBody1 := `{"user_code": "usr_001", "name": "Agent1", "model": "gpt-4"}`
	createReq1 := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBufferString(createBody1))
	createReq1.Header.Set("Content-Type", "application/json")
	createW1 := httptest.NewRecorder()
	mux.ServeHTTP(createW1, createReq1)

	createBody2 := `{"user_code": "usr_002", "name": "Agent2", "model": "gpt-4"}`
	createReq2 := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBufferString(createBody2))
	createReq2.Header.Set("Content-Type", "application/json")
	createW2 := httptest.NewRecorder()
	mux.ServeHTTP(createW2, createReq2)

	// 按 user_code 过滤
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents?user_code=usr_001", nil)
	listW := httptest.NewRecorder()

	mux.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, listW.Code)
	}

	var resp []map[string]interface{}
	json.Unmarshal(listW.Body.Bytes(), &resp)

	for _, agent := range resp {
		if agent["user_code"] != "usr_001" {
			t.Errorf("期望所有 agents 的 user_code 为 'usr_001', 实际为 '%v'", agent["user_code"])
		}
	}
}

func TestGetAgent_ByID(t *testing.T) {
	_, mux := setupTestAgentHandler()

	// 先创建一个 agent
	createBody := `{"user_code": "usr_001", "name": "测试Agent", "model": "gpt-4"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	agentID := createResp["id"].(string)

	// 通过 ID 获取
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents/?id="+agentID, nil)
	getW := httptest.NewRecorder()

	mux.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, getW.Code)
	}

	var getResp map[string]interface{}
	json.Unmarshal(getW.Body.Bytes(), &getResp)

	if getResp["name"] != "测试Agent" {
		t.Errorf("期望 name 为 '测试Agent', 实际为 '%v'", getResp["name"])
	}
}

func TestGetAgent_ByCode(t *testing.T) {
	_, mux := setupTestAgentHandler()

	// 先创建一个 agent
	createBody := `{"user_code": "usr_001", "name": "测试Agent", "model": "gpt-4"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	agentCode := createResp["agent_code"].(string)

	// 通过 code 获取
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents/?code="+agentCode, nil)
	getW := httptest.NewRecorder()

	mux.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, getW.Code)
	}

	var getResp map[string]interface{}
	json.Unmarshal(getW.Body.Bytes(), &getResp)

	if getResp["name"] != "测试Agent" {
		t.Errorf("期望 name 为 '测试Agent', 实际为 '%v'", getResp["name"])
	}
}

func TestGetAgent_NotFound(t *testing.T) {
	_, mux := setupTestAgentHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/?id=non-existent", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusNotFound, w.Code)
	}
}

func TestGetAgent_MissingIDAndCode(t *testing.T) {
	_, mux := setupTestAgentHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestUpdateAgent(t *testing.T) {
	_, mux := setupTestAgentHandler()

	// 先创建一个 agent
	createBody := `{"user_code": "usr_001", "name": "原始名称", "model": "gpt-4"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	agentID := createResp["id"].(string)

	// 更新 agent
	updateBody := `{"name": "新名称", "description": "新描述"}`
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/agents/?id="+agentID, bytes.NewBufferString(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()

	mux.ServeHTTP(updateW, updateReq)

	if updateW.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, updateW.Code)
	}

	var updateResp map[string]interface{}
	json.Unmarshal(updateW.Body.Bytes(), &updateResp)

	if updateResp["name"] != "新名称" {
		t.Errorf("期望 name 为 '新名称', 实际为 '%v'", updateResp["name"])
	}
}

func TestUpdateAgent_NotFound(t *testing.T) {
	_, mux := setupTestAgentHandler()

	updateBody := `{"name": "新名称"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/?id=non-existent", bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestUpdateAgent_MissingID(t *testing.T) {
	_, mux := setupTestAgentHandler()

	updateBody := `{"name": "新名称"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/", bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestDeleteAgent(t *testing.T) {
	_, mux := setupTestAgentHandler()

	// 先创建一个 agent
	createBody := `{"user_code": "usr_001", "name": "待删除Agent", "model": "gpt-4"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	agentID := createResp["id"].(string)

	// 删除 agent
	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/?id="+agentID, nil)
	deleteW := httptest.NewRecorder()

	mux.ServeHTTP(deleteW, deleteReq)

	if deleteW.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, deleteW.Code)
	}

	// 验证已删除
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents/?id="+agentID, nil)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusNotFound {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusNotFound, getW.Code)
	}
}

func TestDeleteAgent_NotFound(t *testing.T) {
	_, mux := setupTestAgentHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/?id=non-existent", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestDeleteAgent_MissingID(t *testing.T) {
	_, mux := setupTestAgentHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestAgent_MethodNotAllowed(t *testing.T) {
	_, mux := setupTestAgentHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusMethodNotAllowed, w.Code)
	}
}