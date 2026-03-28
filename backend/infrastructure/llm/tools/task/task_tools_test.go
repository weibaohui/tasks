/**
 * Task Tools 单元测试
 */
package task

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/llm"
)

// mockIDGenerator - 用于测试的 ID 生成器模拟
type mockIDGenerator struct {
	generateFn func() string
}

func (m *mockIDGenerator) Generate() string {
	if m.generateFn != nil {
		return m.generateFn()
	}
	return "mock-id"
}

// mockTask - 用于测试的 Task 模拟
type mockTask struct {
	id         domain.TaskID
	name       string
	status     domain.TaskStatus
	taskType   domain.TaskType
	progress   domain.Progress
	execErr    error
	createdAt  time.Time
	startedAt  *time.Time
	finishedAt *time.Time
}

func (m *mockTask) ID() domain.TaskID         { return m.id }
func (m *mockTask) Name() string              { return m.name }
func (m *mockTask) Status() domain.TaskStatus { return m.status }
func (m *mockTask) Type() domain.TaskType     { return m.taskType }
func (m *mockTask) Progress() domain.Progress { return m.progress }
func (m *mockTask) Error() error              { return m.execErr }
func (m *mockTask) StartedAt() *time.Time     { return m.startedAt }
func (m *mockTask) FinishedAt() *time.Time    { return m.finishedAt }
func (m *mockTask) CreatedAt() time.Time      { return m.createdAt }

// CreateTaskTool tests

func TestCreateTaskTool_Name(t *testing.T) {
	tool := NewCreateTaskTool(nil, nil, "", "", "", "")
	if tool.Name() != "create_task" {
		t.Errorf("期望名称为 create_task, 实际为 %s", tool.Name())
	}
}

func TestCreateTaskTool_Description(t *testing.T) {
	tool := NewCreateTaskTool(nil, nil, "", "", "", "")
	desc := tool.Description()
	if desc == "" {
		t.Error("期望有描述")
	}
}

func TestCreateTaskTool_Parameters(t *testing.T) {
	tool := NewCreateTaskTool(nil, nil, "", "", "", "")
	params := tool.Parameters()
	if params == nil {
		t.Fatal("期望有参数")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(params, &parsed); err != nil {
		t.Fatalf("参数应该是有效的 JSON: %v", err)
	}

	if parsed["type"] != "object" {
		t.Errorf("期望 type 为 object, 实际为 %v", parsed["type"])
	}

	props, ok := parsed["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("期望有 properties")
	}

	if _, ok := props["name"]; !ok {
		t.Error("期望有 name 参数")
	}

	// 检查 required 字段
	required, ok := parsed["required"].([]interface{})
	if !ok {
		t.Fatal("期望有 required 字段")
	}

	found := false
	for _, r := range required {
		if r.(string) == "name" {
			found = true
			break
		}
	}
	if !found {
		t.Error("name 应该在 required 中")
	}
}

func TestCreateTaskTool_ImplementsToolInterface(t *testing.T) {
	tool := NewCreateTaskTool(nil, nil, "", "", "", "")
	var _ llm.Tool = tool
}

func TestCreateTaskTool_Execute_EmptyName(t *testing.T) {
	tool := NewCreateTaskTool(nil, nil, "", "", "", "")

	result, err := tool.Execute(context.Background(), []byte(`{"name": ""}`))
	if err != nil {
		t.Fatalf("Execute 不应返回 error: %v", err)
	}

	// 错误信息现在在 Output 中（JSON 格式）
	if result.Output == "" {
		t.Error("期望有输出（包含错误信息的 JSON）")
	}

	// Error 字段应为空
	if result.Error != "" {
		t.Error("期望 Error 为空，错误信息在 Output 中")
	}
}

func TestCreateTaskTool_Execute_InvalidJSON(t *testing.T) {
	tool := NewCreateTaskTool(nil, nil, "", "", "", "")

	result, err := tool.Execute(context.Background(), []byte(`invalid json`))
	if err != nil {
		t.Fatalf("Execute 不应返回 error: %v", err)
	}

	// 错误信息现在在 Output 中
	if result.Output == "" {
		t.Error("期望有输出（包含错误信息的 JSON）")
	}
}

func TestCreateTaskTool_Execute_InvalidTaskType(t *testing.T) {
	tool := NewCreateTaskTool(nil, nil, "", "", "", "")

	result, err := tool.Execute(context.Background(), []byte(`{"name": "test", "task_type": "invalid"}`))
	if err != nil {
		t.Fatalf("Execute 不应返回 error: %v", err)
	}

	// 错误信息现在在 Output 中
	if result.Output == "" {
		t.Error("期望有输出（包含错误信息的 JSON）")
	}
}

// QueryTaskTool tests

func TestQueryTaskTool_Name(t *testing.T) {
	tool := NewQueryTaskTool(nil)
	if tool.Name() != "query_task" {
		t.Errorf("期望名称为 query_task, 实际为 %s", tool.Name())
	}
}

func TestQueryTaskTool_Description(t *testing.T) {
	tool := NewQueryTaskTool(nil)
	desc := tool.Description()
	if desc == "" {
		t.Error("期望有描述")
	}
}

func TestQueryTaskTool_Parameters(t *testing.T) {
	tool := NewQueryTaskTool(nil)
	params := tool.Parameters()
	if params == nil {
		t.Fatal("期望有参数")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(params, &parsed); err != nil {
		t.Fatalf("参数应该是有效的 JSON: %v", err)
	}

	if parsed["type"] != "object" {
		t.Errorf("期望 type 为 object, 实际为 %v", parsed["type"])
	}

	props, ok := parsed["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("期望有 properties")
	}

	if _, ok := props["task_id"]; !ok {
		t.Error("期望有 task_id 参数")
	}

	// 检查 required 字段
	required, ok := parsed["required"].([]interface{})
	if !ok {
		t.Fatal("期望有 required 字段")
	}

	found := false
	for _, r := range required {
		if r.(string) == "task_id" {
			found = true
			break
		}
	}
	if !found {
		t.Error("task_id 应该在 required 中")
	}
}

func TestQueryTaskTool_ImplementsToolInterface(t *testing.T) {
	tool := NewQueryTaskTool(nil)
	var _ llm.Tool = tool
}

func TestQueryTaskTool_Execute_EmptyTaskID(t *testing.T) {
	tool := NewQueryTaskTool(nil)

	result, err := tool.Execute(context.Background(), []byte(`{"task_id": ""}`))
	if err != nil {
		t.Fatalf("Execute 不应返回 error: %v", err)
	}

	// 错误信息现在在 Output 中
	if result.Output == "" {
		t.Error("期望有输出（包含错误信息的 JSON）")
	}

	// Error 字段应为空
	if result.Error != "" {
		t.Error("期望 Error 为空，错误信息在 Output 中")
	}
}

func TestQueryTaskTool_Execute_InvalidJSON(t *testing.T) {
	tool := NewQueryTaskTool(nil)

	result, err := tool.Execute(context.Background(), []byte(`invalid json`))
	if err != nil {
		t.Fatalf("Execute 不应返回 error: %v", err)
	}

	// 错误信息现在在 Output 中
	if result.Output == "" {
		t.Error("期望有输出（包含错误信息的 JSON）")
	}
}
