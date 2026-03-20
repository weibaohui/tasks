/**
 * HTTP Handler 端到端测试
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
	"github.com/weibh/taskmanager/infrastructure/bus"
)

type mockIDGenerator struct {
	count int
}

func (m *mockIDGenerator) Generate() string {
	m.count++
	return "id-" + string(rune('0'+m.count))
}

type mockTaskRepository struct {
	tasks map[string]*domain.Task
}

func newMockTaskRepository() *mockTaskRepository {
	return &mockTaskRepository{
		tasks: make(map[string]*domain.Task),
	}
}

func (m *mockTaskRepository) Save(ctx context.Context, task *domain.Task) error {
	m.tasks[task.ID().String()] = task
	return nil
}

func (m *mockTaskRepository) FindByID(ctx context.Context, id domain.TaskID) (*domain.Task, error) {
	task, ok := m.tasks[id.String()]
	if !ok {
		return nil, application.ErrTaskNotFound
	}
	return task, nil
}

func (m *mockTaskRepository) FindByTraceID(ctx context.Context, traceID domain.TraceID) ([]*domain.Task, error) {
	var result []*domain.Task
	for _, task := range m.tasks {
		if task.TraceID().String() == traceID.String() {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *mockTaskRepository) FindByParentID(ctx context.Context, parentID domain.TaskID) ([]*domain.Task, error) {
	return nil, nil
}

func (m *mockTaskRepository) FindByStatus(ctx context.Context, status domain.TaskStatus) ([]*domain.Task, error) {
	return nil, nil
}

func (m *mockTaskRepository) FindRunningTasks(ctx context.Context) ([]*domain.Task, error) {
	return nil, nil
}

func (m *mockTaskRepository) Delete(ctx context.Context, id domain.TaskID) error {
	delete(m.tasks, id.String())
	return nil
}

func (m *mockTaskRepository) Exists(ctx context.Context, id domain.TaskID) (bool, error) {
	_, ok := m.tasks[id.String()]
	return ok, nil
}

func setupTestHandler() (*TaskHandler, *http.ServeMux) {
	repo := newMockTaskRepository()
	idGen := &mockIDGenerator{}
	eventBus := bus.NewEventBus()

	taskService := application.NewTaskApplicationService(repo, idGen, eventBus, nil)
	queryService := application.NewQueryService(repo)

	handler := NewTaskHandler(taskService, queryService)
	mux := SetupRoutes(handler)

	return handler, mux
}

func TestCreateTask(t *testing.T) {
	_, mux := setupTestHandler()

	body := `{
		"name": "测试任务",
		"description": "任务描述",
		"type": "data_processing",
		"timeout": 60000,
		"max_retries": 3,
		"priority": 5
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusCreated, w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["id"] == nil || resp["id"] == "" {
		t.Error("响应中应该包含任务 ID")
	}

	if resp["trace_id"] == nil || resp["trace_id"] == "" {
		t.Error("响应中应该包含 trace_id")
	}
}

func TestCreateTask_InvalidBody(t *testing.T) {
	_, mux := setupTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateTask_EmptyName(t *testing.T) {
	_, mux := setupTestHandler()

	body := `{
		"name": "",
		"type": "data_processing"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetTask(t *testing.T) {
	_, mux := setupTestHandler()

	createBody := `{"name": "测试任务", "type": "data_processing", "timeout": 60000}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	taskID := createResp["id"].(string)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?id="+taskID, nil)
	getW := httptest.NewRecorder()

	mux.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, getW.Code)
	}

	var taskResp map[string]interface{}
	json.Unmarshal(getW.Body.Bytes(), &taskResp)

	if taskResp["name"] != "测试任务" {
		t.Errorf("期望任务名称为 '测试任务', 实际为 '%v'", taskResp["name"])
	}
}

func TestGetTask_NotFound(t *testing.T) {
	_, mux := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?id=non-existent", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusNotFound, w.Code)
	}
}

func TestGetTask_MissingID(t *testing.T) {
	_, mux := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestCancelTask(t *testing.T) {
	_, mux := setupTestHandler()

	createBody := `{"name": "测试任务", "type": "data_processing", "timeout": 60000}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	taskID := createResp["id"].(string)

	cancelReq := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+taskID+"/cancel", nil)
	cancelW := httptest.NewRecorder()

	mux.ServeHTTP(cancelW, cancelReq)

	if cancelW.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, cancelW.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?id="+taskID, nil)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getReq)

	var taskResp map[string]interface{}
	json.Unmarshal(getW.Body.Bytes(), &taskResp)

	if taskResp["status"] != "cancelled" {
		t.Errorf("期望任务状态为 'cancelled', 实际为 '%v'", taskResp["status"])
	}
}

func TestCancelTask_NotFound(t *testing.T) {
	_, mux := setupTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/non-existent/cancel", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusNotFound, w.Code)
	}
}

func TestListTasksByTrace(t *testing.T) {
	_, mux := setupTestHandler()

	createBody := `{"name": "测试任务1", "type": "data_processing", "timeout": 60000}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	traceID := createResp["trace_id"].(string)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/trace/"+traceID, nil)
	listW := httptest.NewRecorder()

	mux.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, listW.Code)
	}

	var listResp map[string]interface{}
	json.Unmarshal(listW.Body.Bytes(), &listResp)

	tasks := listResp["tasks"].([]interface{})
	if len(tasks) < 1 {
		t.Error("期望至少有一个任务")
	}
}

func TestGetTaskTree(t *testing.T) {
	_, mux := setupTestHandler()

	createBody := `{"name": "父任务", "type": "data_processing", "timeout": 60000}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	traceID := createResp["trace_id"].(string)

	treeReq := httptest.NewRequest(http.MethodGet, "/api/v1/traces/"+traceID+"/tree", nil)
	treeW := httptest.NewRecorder()

	mux.ServeHTTP(treeW, treeReq)

	if treeW.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, treeW.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	_, mux := setupTestHandler()

	req := httptest.NewRequest(http.MethodPut, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusMethodNotAllowed, w.Code)
	}
}
