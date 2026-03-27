/**
 * 应用服务单元测试
 */
package application

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/bus"
	"go.uber.org/zap"
)

type mockIDGenerator struct {
	mu    sync.Mutex
	count int
}

func (m *mockIDGenerator) Generate() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.count++
	return "id-" + strconv.Itoa(m.count)
}

type mockTaskRepository struct {
	mu    sync.Mutex
	tasks map[string]*domain.Task
}

func newMockTaskRepository() *mockTaskRepository {
	return &mockTaskRepository{
		tasks: make(map[string]*domain.Task),
	}
}

func (m *mockTaskRepository) Save(ctx context.Context, task *domain.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[task.ID().String()] = task
	return nil
}

func (m *mockTaskRepository) FindByID(ctx context.Context, id domain.TaskID) (*domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	task, ok := m.tasks[id.String()]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

func (m *mockTaskRepository) FindAll(ctx context.Context) ([]*domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*domain.Task
	for _, task := range m.tasks {
		result = append(result, task)
	}
	return result, nil
}

func (m *mockTaskRepository) FindByTraceID(ctx context.Context, traceID domain.TraceID) ([]*domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
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
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tasks, id.String())
	return nil
}

func (m *mockTaskRepository) Exists(ctx context.Context, id domain.TaskID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.tasks[id.String()]
	return ok, nil
}

func TestTaskApplicationService_CreateTask(t *testing.T) {
	repo := newMockTaskRepository()
	idGen := &mockIDGenerator{}
	eventBus := bus.NewEventBus()
	logger, _ := zap.NewDevelopment()

	service := NewTaskApplicationService(repo, idGen, eventBus, logger)

	cmd := CreateTaskCommand{
		Name:               "测试任务",
		TaskRequirement:    "测试目标",
		AcceptanceCriteria: "测试验收标准",
		Description:        "任务描述",
		Type:              domain.TaskTypeCustom,
		Timeout:           60000,
		MaxRetries:        3,
		Priority:          5,
	}

	task, err := service.CreateTask(context.Background(), cmd)
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}

	if task.Name() != "测试任务" {
		t.Errorf("期望任务名称为 '测试任务', 实际为 '%s'", task.Name())
	}

	if task.Status() != domain.TaskStatusPending {
		t.Errorf("期望状态为 Pending, 实际为 %v", task.Status())
	}

	if task.ID().String() == "" {
		t.Error("期望任务ID不为空")
	}
}

func TestTaskApplicationService_CreateTask_WithParent(t *testing.T) {
	repo := newMockTaskRepository()
	idGen := &mockIDGenerator{}
	eventBus := bus.NewEventBus()

	logger, _ := zap.NewDevelopment()
	service := NewTaskApplicationService(repo, idGen, eventBus, logger)

	parentCmd := CreateTaskCommand{
		Name:               "父任务",
		TaskRequirement:    "父任务目标",
		AcceptanceCriteria: "父任务验收标准",
		Type:               domain.TaskTypeCustom,
		Timeout:            60000,
	}

	parent, _ := service.CreateTask(context.Background(), parentCmd)
	parentID := parent.ID()

	cmd := CreateTaskCommand{
		Name:               "子任务",
		TaskRequirement:    "子任务目标",
		AcceptanceCriteria: "子任务验收标准",
		Type:               domain.TaskTypeCustom,
		Timeout:            30000,
		ParentID:           &parentID,
	}

	child, err := service.CreateTask(context.Background(), cmd)
	if err != nil {
		t.Fatalf("创建子任务失败: %v", err)
	}

	if child.TraceID().String() != parent.TraceID().String() {
		t.Error("子任务应该继承父任务的 TraceID")
	}
}

func TestTaskApplicationService_StartTask(t *testing.T) {
	repo := newMockTaskRepository()
	idGen := &mockIDGenerator{}
	eventBus := bus.NewEventBus()

	logger, _ := zap.NewDevelopment()
	service := NewTaskApplicationService(repo, idGen, eventBus, logger)

	task, _ := service.CreateTask(context.Background(), CreateTaskCommand{
		Name:               "测试任务",
		TaskRequirement:    "测试目标",
		AcceptanceCriteria: "测试验收标准",
		Type:               domain.TaskTypeCustom,
		Timeout:            60000,
	})

	err := service.StartTask(context.Background(), task.ID())
	if err != nil {
		t.Fatalf("启动任务失败: %v", err)
	}

	updatedTask, _ := repo.FindByID(context.Background(), task.ID())
	if updatedTask.Status() != domain.TaskStatusRunning {
		t.Errorf("期望状态为 Running, 实际为 %v", updatedTask.Status())
	}
}

func TestTaskApplicationService_StartTask_NotFound(t *testing.T) {
	repo := newMockTaskRepository()
	idGen := &mockIDGenerator{}
	eventBus := bus.NewEventBus()

	logger, _ := zap.NewDevelopment()
	service := NewTaskApplicationService(repo, idGen, eventBus, logger)

	err := service.StartTask(context.Background(), domain.NewTaskID("non-existent"))
	if !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("期望返回 ErrTaskNotFound, 实际返回 %v", err)
	}
}

func TestTaskApplicationService_CancelTask(t *testing.T) {
	repo := newMockTaskRepository()
	idGen := &mockIDGenerator{}
	eventBus := bus.NewEventBus()

	logger, _ := zap.NewDevelopment()
	service := NewTaskApplicationService(repo, idGen, eventBus, logger)

	task, _ := service.CreateTask(context.Background(), CreateTaskCommand{
		Name:               "测试任务",
		TaskRequirement:    "测试目标",
		AcceptanceCriteria: "测试验收标准",
		Type:               domain.TaskTypeCustom,
		Timeout:            60000,
	})

	err := service.CancelTask(context.Background(), task.ID())
	if err != nil {
		t.Fatalf("取消任务失败: %v", err)
	}

	updatedTask, _ := repo.FindByID(context.Background(), task.ID())
	if updatedTask.Status() != domain.TaskStatusCancelled {
		t.Errorf("期望状态为 Cancelled, 实际为 %v", updatedTask.Status())
	}
}

func TestTaskApplicationService_CompleteTask(t *testing.T) {
	repo := newMockTaskRepository()
	idGen := &mockIDGenerator{}
	eventBus := bus.NewEventBus()

	logger, _ := zap.NewDevelopment()
	service := NewTaskApplicationService(repo, idGen, eventBus, logger)

	task, _ := service.CreateTask(context.Background(), CreateTaskCommand{
		Name:               "测试任务",
		TaskRequirement:    "测试目标",
		AcceptanceCriteria: "测试验收标准",
		Type:               domain.TaskTypeCustom,
		Timeout:            60000,
	})

	service.StartTask(context.Background(), task.ID())
	task.SetTaskConclusion("测试结论") // 需要设置结论才能完成任务

	result := domain.NewResult(map[string]interface{}{"status": "ok"}, "完成")
	err := service.CompleteTask(context.Background(), task.ID(), result)
	if err != nil {
		t.Fatalf("完成任务失败: %v", err)
	}

	updatedTask, _ := repo.FindByID(context.Background(), task.ID())
	if updatedTask.Status() != domain.TaskStatusCompleted {
		t.Errorf("期望状态为 Completed, 实际为 %v", updatedTask.Status())
	}
}

func TestTaskApplicationService_UpdateProgress(t *testing.T) {
	repo := newMockTaskRepository()
	idGen := &mockIDGenerator{}
	eventBus := bus.NewEventBus()

	logger, _ := zap.NewDevelopment()
	service := NewTaskApplicationService(repo, idGen, eventBus, logger)

	task, _ := service.CreateTask(context.Background(), CreateTaskCommand{
		Name:               "测试任务",
		TaskRequirement:    "测试目标",
		AcceptanceCriteria: "测试验收标准",
		Type:               domain.TaskTypeCustom,
		Timeout:            60000,
	})

	service.StartTask(context.Background(), task.ID())

	err := service.UpdateProgress(context.Background(), task.ID(), 50)
	if err != nil {
		t.Fatalf("更新进度失败: %v", err)
	}

	updatedTask, _ := repo.FindByID(context.Background(), task.ID())
	progress := updatedTask.Progress()
	if progress.Value() != 50 {
		t.Errorf("期望当前进度为 50, 实际为 %d", progress.Value())
	}
}

func TestTaskApplicationService_FailTask(t *testing.T) {
	repo := newMockTaskRepository()
	idGen := &mockIDGenerator{}
	eventBus := bus.NewEventBus()

	logger, _ := zap.NewDevelopment()
	service := NewTaskApplicationService(repo, idGen, eventBus, logger)

	task, _ := service.CreateTask(context.Background(), CreateTaskCommand{
		Name:               "测试任务",
		TaskRequirement:    "测试目标",
		AcceptanceCriteria: "测试验收标准",
		Type:               domain.TaskTypeCustom,
		Timeout:            60000,
	})

	service.StartTask(context.Background(), task.ID())

	taskErr := errors.New("任务失败")
	err := service.FailTask(context.Background(), task.ID(), taskErr)
	if err != nil {
		t.Fatalf("标记任务失败: %v", err)
	}

	updatedTask, _ := repo.FindByID(context.Background(), task.ID())
	if updatedTask.Status() != domain.TaskStatusFailed {
		t.Errorf("期望状态为 Failed, 实际为 %v", updatedTask.Status())
	}
}
