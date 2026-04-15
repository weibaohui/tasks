package hooks

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/weibh/taskmanager/domain"
	"go.uber.org/zap"
)

// mockRequirementRepoForProgress 用于进度跟踪测试的轻量 mock
type mockRequirementRepoForProgress struct {
	requirements map[string]*domain.Requirement
}

func newMockRequirementRepoForProgress() *mockRequirementRepoForProgress {
	return &mockRequirementRepoForProgress{
		requirements: make(map[string]*domain.Requirement),
	}
}

func (m *mockRequirementRepoForProgress) Save(ctx context.Context, req *domain.Requirement) error {
	m.requirements[req.ID().String()] = req
	return nil
}

func (m *mockRequirementRepoForProgress) FindByID(ctx context.Context, id domain.RequirementID) (*domain.Requirement, error) {
	return m.requirements[id.String()], nil
}

func (m *mockRequirementRepoForProgress) FindByTraceID(ctx context.Context, traceID string) (*domain.Requirement, error) {
	for _, req := range m.requirements {
		if req.TraceID() == traceID {
			return req, nil
		}
	}
	return nil, nil
}

func (m *mockRequirementRepoForProgress) FindByProjectID(ctx context.Context, projectID domain.ProjectID) ([]*domain.Requirement, error) {
	return nil, nil
}

func (m *mockRequirementRepoForProgress) FindAll(ctx context.Context) ([]*domain.Requirement, error) {
	return nil, nil
}

func (m *mockRequirementRepoForProgress) Delete(ctx context.Context, id domain.RequirementID) error {
	delete(m.requirements, id.String())
	return nil
}

func (m *mockRequirementRepoForProgress) List(ctx context.Context, filter domain.RequirementListFilter) ([]*domain.Requirement, error) {
	return nil, nil
}

func (m *mockRequirementRepoForProgress) Count(ctx context.Context, filter domain.RequirementListFilter) (int, error) {
	return 0, nil
}

func (m *mockRequirementRepoForProgress) GetStatusStats(ctx context.Context, projectID *domain.ProjectID) ([]domain.StatusStat, error) {
	return nil, nil
}

func TestProgressTrackingHook_isProgressTool(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := NewProgressTrackingHook(nil, logger)

	tests := []struct {
		name     string
		toolName string
		want     bool
	}{
		{"todowrite lowercase", "todowrite", true},
		{"TodoWrite mixed case", "TodoWrite", true},
		{"TODOWRITE uppercase", "TODOWRITE", true},
		{"todo", "todo", true},
		{"todo_write", "todo_write", true},
		{"write", "write", false},
		{"edit", "edit", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := h.isProgressTool(tt.toolName); got != tt.want {
				t.Errorf("isProgressTool(%q) = %v, want %v", tt.toolName, got, tt.want)
			}
		})
	}
}

func TestProgressTrackingHook_extractTodos_standardArray(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := NewProgressTrackingHook(nil, logger)

	input := map[string]interface{}{
		"todos": []interface{}{
			map[string]interface{}{
				"content":  "任务一",
				"status":   "completed",
				"priority": "high",
			},
			map[string]interface{}{
				"content":  "任务二",
				"status":   "in_progress",
				"priority": "medium",
			},
		},
	}

	items, ok := h.extractTodos(input)
	if !ok {
		t.Fatal("expected ok")
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Content != "任务一" || items[0].Status != "completed" || items[0].Priority != "high" {
		t.Errorf("unexpected first item: %+v", items[0])
	}
	if items[1].Content != "任务二" || items[1].Status != "in_progress" || items[1].Priority != "medium" {
		t.Errorf("unexpected second item: %+v", items[1])
	}
}

func TestProgressTrackingHook_extractTodos_stringifiedJSON(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := NewProgressTrackingHook(nil, logger)

	// 模拟某些 Agent 将 todos 作为 JSON 字符串传入
	stringified := `[{"content":"检查项目运行状态和最新PR状态","status":"in_progress","priority":"high"},{"content":"分析GitHub最新提交和PR活动","status":"pending","priority":"high"}]`
	input := map[string]interface{}{
		"todos": stringified,
	}

	items, ok := h.extractTodos(input)
	if !ok {
		t.Fatal("expected ok for stringified JSON")
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Content != "检查项目运行状态和最新PR状态" || items[0].Status != "in_progress" {
		t.Errorf("unexpected first item: %+v", items[0])
	}
	if items[1].Content != "分析GitHub最新提交和PR活动" || items[1].Status != "pending" {
		t.Errorf("unexpected second item: %+v", items[1])
	}
}

func TestProgressTrackingHook_extractTodos_activeFormPriority(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := NewProgressTrackingHook(nil, logger)

	input := map[string]interface{}{
		"todos": []interface{}{
			map[string]interface{}{
				"activeForm": "Creating test file",
				"content":    "Create test file",
				"status":     "completed",
				"priority":   "high",
			},
			map[string]interface{}{
				"content":  "只有 content",
				"status":   "pending",
				"priority": "low",
			},
		},
	}

	items, ok := h.extractTodos(input)
	if !ok {
		t.Fatal("expected ok")
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	// activeForm 应该优先于 content
	if items[0].Content != "Creating test file" {
		t.Errorf("expected activeForm to take priority, got %q", items[0].Content)
	}
	if items[1].Content != "只有 content" {
		t.Errorf("expected content fallback, got %q", items[1].Content)
	}
}

func TestProgressTrackingHook_extractTodos_fallbackFields(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := NewProgressTrackingHook(nil, logger)

	input := map[string]interface{}{
		"todos": []interface{}{
			map[string]interface{}{
				"task":   "任务名称",
				"status": "done",
			},
			map[string]interface{}{
				"title":  "标题",
				"status": "completed",
			},
		},
	}

	items, ok := h.extractTodos(input)
	if !ok {
		t.Fatal("expected ok")
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Content != "任务名称" {
		t.Errorf("expected task fallback, got %q", items[0].Content)
	}
	if items[1].Content != "标题" {
		t.Errorf("expected title fallback, got %q", items[1].Content)
	}
}

func TestProgressTrackingHook_extractTodos_invalidInputs(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := NewProgressTrackingHook(nil, logger)

	tests := []struct {
		name  string
		input map[string]interface{}
	}{
		{"nil input", nil},
		{"missing todos", map[string]interface{}{"other": "value"}},
		{"todos as number", map[string]interface{}{"todos": 123}},
		{"todos as invalid string", map[string]interface{}{"todos": "not json"}},
		{"empty todos array", map[string]interface{}{"todos": []interface{}{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, ok := h.extractTodos(tt.input)
			if ok {
				t.Errorf("expected not ok, got items=%v", items)
			}
		})
	}
}

func TestProgressTrackingHook_PreToolCall_updatesRequirement(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := newMockRequirementRepoForProgress()
	h := NewProgressTrackingHook(repo, logger)

	req, _ := domain.NewRequirement(
		domain.NewRequirementID("req-001"),
		domain.NewProjectID("proj-001"),
		"测试需求",
		"描述",
		"验收标准",
		"/tmp/workspace",
	)
	req.SetTraceID("trace-123")
	_ = repo.Save(context.Background(), req)

	ctx := domain.NewHookContext(context.Background())
	callCtx := &domain.ToolCallContext{
		ToolName:  "todowrite",
		ToolInput: map[string]interface{}{
			"todos": []interface{}{
				map[string]interface{}{
					"content":  "步骤1",
					"status":   "completed",
					"priority": "high",
				},
				map[string]interface{}{
					"content":  "步骤2",
					"status":   "in_progress",
					"priority": "medium",
				},
			},
		},
		TraceID: "trace-123",
	}

	result, err := h.PreToolCall(ctx, callCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != callCtx {
		t.Error("expected same callCtx returned")
	}

	updated, err := repo.FindByID(context.Background(), domain.NewRequirementID("req-001"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	progressJSON := updated.ProgressData()
	if progressJSON == "" {
		t.Fatal("expected progress_data to be updated")
	}

	var data domain.ProgressData
	if err := json.Unmarshal([]byte(progressJSON), &data); err != nil {
		t.Fatalf("failed to unmarshal progress data: %v", err)
	}
	if len(data.Items) != 2 {
		t.Fatalf("expected 2 progress items, got %d", len(data.Items))
	}
	if data.Percent != 50 {
		t.Errorf("expected percent 50, got %d", data.Percent)
	}
	if data.Items[0].Content != "步骤1" || data.Items[0].Status != "completed" {
		t.Errorf("unexpected first progress item: %+v", data.Items[0])
	}
}

func TestProgressTrackingHook_PreToolCall_noTraceID(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := newMockRequirementRepoForProgress()
	h := NewProgressTrackingHook(repo, logger)

	ctx := domain.NewHookContext(context.Background())
	callCtx := &domain.ToolCallContext{
		ToolName:  "todowrite",
		ToolInput: map[string]interface{}{
			"todos": []interface{}{
				map[string]interface{}{
					"content": "步骤1",
					"status":  "completed",
				},
			},
		},
		TraceID: "",
	}

	// 设置 metadata 中的 trace_id
	ctx.SetMetadata("trace_id", "trace-missing")

	_, err := h.PreToolCall(ctx, callCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 由于没有对应 trace_id 的需求，不应该 panic，只是静默跳过
}

func TestProgressTrackingHook_PreToolCall_nonProgressTool(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := newMockRequirementRepoForProgress()
	h := NewProgressTrackingHook(repo, logger)

	ctx := domain.NewHookContext(context.Background())
	callCtx := &domain.ToolCallContext{
		ToolName:  "write",
		ToolInput: map[string]interface{}{"content": "hello"},
		TraceID:   "trace-123",
	}

	result, err := h.PreToolCall(ctx, callCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != callCtx {
		t.Error("expected same callCtx returned")
	}
}
