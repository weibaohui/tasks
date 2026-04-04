/**
 * QueryService 查询服务单元测试
 */
package application

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/weibh/taskmanager/domain"
)

// mockQueryTaskRepository 是用于 QueryService 测试的 mock 仓库
type mockQueryTaskRepository struct {
	mu     sync.Mutex
	tasks  map[string]*domain.Task
	nextID int
}

func newMockQueryTaskRepository() *mockQueryTaskRepository {
	return &mockQueryTaskRepository{
		tasks:  make(map[string]*domain.Task),
		nextID: 1,
	}
}

func (m *mockQueryTaskRepository) generateID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := "task-" + strconv.Itoa(m.nextID)
	m.nextID++
	return id
}

func (m *mockQueryTaskRepository) Save(ctx context.Context, task *domain.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[task.ID().String()] = task
	return nil
}

func (m *mockQueryTaskRepository) FindByID(ctx context.Context, id domain.TaskID) (*domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	task, ok := m.tasks[id.String()]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

func (m *mockQueryTaskRepository) FindAll(ctx context.Context) ([]*domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*domain.Task
	for _, task := range m.tasks {
		result = append(result, task)
	}
	return result, nil
}

func (m *mockQueryTaskRepository) FindByTraceID(ctx context.Context, traceID domain.TraceID) ([]*domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*domain.Task, 0)
	for _, task := range m.tasks {
		if task.TraceID().String() == traceID.String() {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *mockQueryTaskRepository) FindByParentID(ctx context.Context, parentID domain.TaskID) ([]*domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*domain.Task
	for _, task := range m.tasks {
		if task.ParentID() != nil && task.ParentID().String() == parentID.String() {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *mockQueryTaskRepository) FindByStatus(ctx context.Context, status domain.TaskStatus) ([]*domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*domain.Task
	for _, task := range m.tasks {
		if task.Status() == status {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *mockQueryTaskRepository) FindRunningTasks(ctx context.Context) ([]*domain.Task, error) {
	return m.FindByStatus(ctx, domain.TaskStatusRunning)
}

func (m *mockQueryTaskRepository) Delete(ctx context.Context, id domain.TaskID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tasks, id.String())
	return nil
}

func (m *mockQueryTaskRepository) Exists(ctx context.Context, id domain.TaskID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.tasks[id.String()]
	return ok, nil
}

// createQueryTestTask 创建测试任务辅助函数
func createQueryTestTask(repo *mockQueryTaskRepository, name, traceID, spanID string, parentID *domain.TaskID, depth int) (*domain.Task, error) {
	taskID := domain.NewTaskID(repo.generateID())
	traceIDObj := domain.NewTraceID(traceID)
	spanIDObj := domain.NewSpanID(spanID)

	task, err := domain.NewTask(
		taskID,
		traceIDObj,
		spanIDObj,
		parentID,
		name,
		"任务描述: "+name,
		domain.TaskTypeCustom,
		"任务要求: "+name,
		"验收标准: "+name,
		30*time.Minute,
		3,
		5,
	)
	if err != nil {
		return nil, err
	}

	// 设置深度和其他字段
	task.SetDepth(depth)
	task.SetTaskConclusion("任务结论: " + name)
	task.SetUserCode("USER001")
	task.SetAgentCode("AGENT001")
	task.SetChannelCode("CHANNEL001")
	task.SetSessionKey("SESSION001")
	task.SetTodoList("待办列表: " + name)
	task.SetAnalysis("分析结果: " + name)

	if parentID != nil {
		task.SetParentSpan("parent-span-" + parentID.String())
	}

	if err := repo.Save(context.Background(), task); err != nil {
		return nil, err
	}

	return task, nil
}

// createQueryTestTaskWithStatus 创建带状态的测试任务
func createQueryTestTaskWithStatus(repo *mockQueryTaskRepository, name, traceID, spanID string, parentID *domain.TaskID, depth int, status domain.TaskStatus) (*domain.Task, error) {
	task, err := createQueryTestTask(repo, name, traceID, spanID, parentID, depth)
	if err != nil {
		return nil, err
	}

	switch status {
	case domain.TaskStatusRunning:
		task.Start()
	case domain.TaskStatusCompleted:
		task.Start()
		task.SetTaskConclusion("已完成")
		task.Complete()
	case domain.TaskStatusFailed:
		task.Start()
		task.Fail(errors.New("测试失败"))
	case domain.TaskStatusCancelled:
		task.Cancel()
	}

	// 重新保存更新后的任务
	repo.Save(context.Background(), task)
	return task, nil
}

// TestNewQueryService 测试 QueryService 初始化
func TestNewQueryService(t *testing.T) {
	repo := newMockQueryTaskRepository()
	service := NewQueryService(repo)

	if service == nil {
		t.Fatal("NewQueryService 返回 nil")
	}

	if service.taskRepo != repo {
		t.Error("QueryService 的 taskRepo 应该与传入的 repo 相同")
	}
}

// TestQueryService_GetTask 测试获取任务详情
func TestQueryService_GetTask(t *testing.T) {
	ctx := context.Background()
	repo := newMockQueryTaskRepository()
	service := NewQueryService(repo)

	// 创建测试任务
	createdTask, err := createQueryTestTask(repo, "测试任务", "trace-001", "span-001", nil, 1)
	if err != nil {
		t.Fatalf("创建测试任务失败: %v", err)
	}

	// 测试正常获取任务
	t.Run("正常获取任务", func(t *testing.T) {
		dto, err := service.GetTask(ctx, createdTask.ID())
		if err != nil {
			t.Fatalf("获取任务失败: %v", err)
		}

		if dto == nil {
			t.Fatal("返回的 DTO 为 nil")
		}

		// 验证基本字段
		if dto.ID != createdTask.ID().String() {
			t.Errorf("期望 ID 为 %s, 实际为 %s", createdTask.ID().String(), dto.ID)
		}
		if dto.Name != "测试任务" {
			t.Errorf("期望 Name 为 '测试任务', 实际为 %s", dto.Name)
		}
		if dto.TraceID != "trace-001" {
			t.Errorf("期望 TraceID 为 'trace-001', 实际为 %s", dto.TraceID)
		}
		if dto.SpanID != "span-001" {
			t.Errorf("期望 SpanID 为 'span-001', 实际为 %s", dto.SpanID)
		}
		if dto.Type != domain.TaskTypeCustom.String() {
			t.Errorf("期望 Type 为 'custom', 实际为 %s", dto.Type)
		}
		if dto.Status != domain.TaskStatusPending.String() {
			t.Errorf("期望 Status 为 'pending', 实际为 %s", dto.Status)
		}
		if dto.Depth != 1 {
			t.Errorf("期望 Depth 为 1, 实际为 %d", dto.Depth)
		}
	})

	// 测试任务不存在时的错误处理
	t.Run("任务不存在时的错误处理", func(t *testing.T) {
		nonExistentID := domain.NewTaskID("non-existent-id")
		dto, err := service.GetTask(ctx, nonExistentID)

		if err == nil {
			t.Error("期望返回错误，但实际没有错误")
		}

		if !errors.Is(err, ErrTaskNotFound) {
			t.Errorf("期望返回 ErrTaskNotFound, 实际返回 %v", err)
		}

		if dto != nil {
			t.Error("期望返回的 DTO 为 nil")
		}
	})

	// 测试 DTO 字段正确性验证（包含可选字段）
	t.Run("DTO 字段正确性验证", func(t *testing.T) {
		// 创建一个带有更多字段的任务
		parentID := domain.NewTaskID("parent-001")
		task, err := createQueryTestTask(repo, "子任务", "trace-002", "span-002", &parentID, 2)
		if err != nil {
			t.Fatalf("创建子任务失败: %v", err)
		}

		// 启动任务以设置 StartedAt
		task.Start()
		repo.Save(ctx, task)

		dto, err := service.GetTask(ctx, task.ID())
		if err != nil {
			t.Fatalf("获取任务失败: %v", err)
		}

		// 验证 ParentID
		if dto.ParentID == nil {
			t.Error("期望 ParentID 不为 nil")
		} else if *dto.ParentID != "parent-001" {
			t.Errorf("期望 ParentID 为 'parent-001', 实际为 %s", *dto.ParentID)
		}

		// 验证 StartedAt
		if dto.StartedAt == nil {
			t.Error("期望 StartedAt 不为 nil")
		}

		// 验证 FinishedAt 为 nil（任务未完成）
		if dto.FinishedAt != nil {
			t.Error("期望 FinishedAt 为 nil")
		}

		// 验证扩展字段
		if dto.AcceptanceCriteria != "验收标准: 子任务" {
			t.Errorf("期望 AcceptanceCriteria 为 '验收标准: 子任务', 实际为 %s", dto.AcceptanceCriteria)
		}
		if dto.TaskRequirement != "任务要求: 子任务" {
			t.Errorf("期望 TaskRequirement 为 '任务要求: 子任务', 实际为 %s", dto.TaskRequirement)
		}
		if dto.UserCode != "USER001" {
			t.Errorf("期望 UserCode 为 'USER001', 实际为 %s", dto.UserCode)
		}
		if dto.AgentCode != "AGENT001" {
			t.Errorf("期望 AgentCode 为 'AGENT001', 实际为 %s", dto.AgentCode)
		}
		if dto.ChannelCode != "CHANNEL001" {
			t.Errorf("期望 ChannelCode 为 'CHANNEL001', 实际为 %s", dto.ChannelCode)
		}
		if dto.SessionKey != "SESSION001" {
			t.Errorf("期望 SessionKey 为 'SESSION001', 实际为 %s", dto.SessionKey)
		}
	})

	// 测试 FinishedAt 字段（已完成任务）
	t.Run("已完成任务的 FinishedAt 字段", func(t *testing.T) {
		task, err := createQueryTestTask(repo, "已完成任务", "trace-003", "span-003", nil, 1)
		if err != nil {
			t.Fatalf("创建任务失败: %v", err)
		}

		task.Start()
		task.SetTaskConclusion("任务已完成")
		task.Complete()
		repo.Save(ctx, task)

		dto, err := service.GetTask(ctx, task.ID())
		if err != nil {
			t.Fatalf("获取任务失败: %v", err)
		}

		// 验证 StartedAt 和 FinishedAt 都不为 nil
		if dto.StartedAt == nil {
			t.Error("期望 StartedAt 不为 nil")
		}
		if dto.FinishedAt == nil {
			t.Error("期望 FinishedAt 不为 nil")
		}
		if dto.Status != domain.TaskStatusCompleted.String() {
			t.Errorf("期望 Status 为 'completed', 实际为 %s", dto.Status)
		}
	})
}

// TestQueryService_ListAllTasks 测试获取所有任务
func TestQueryService_ListAllTasks(t *testing.T) {
	ctx := context.Background()
	repo := newMockQueryTaskRepository()
	service := NewQueryService(repo)

	// 测试空列表返回
	t.Run("空列表返回", func(t *testing.T) {
		result, err := service.ListAllTasks(ctx)
		if err != nil {
			t.Fatalf("获取任务列表失败: %v", err)
		}

		if result == nil {
			t.Fatal("返回的结果为 nil")
		}

		if len(result.Tasks) != 0 {
			t.Errorf("期望任务列表长度为 0, 实际为 %d", len(result.Tasks))
		}

		if result.Total != 0 {
			t.Errorf("期望 Total 为 0, 实际为 %d", result.Total)
		}
	})

	// 测试正常获取所有任务
	t.Run("正常获取所有任务", func(t *testing.T) {
		// 创建多个任务
		task1, _ := createQueryTestTask(repo, "任务1", "trace-001", "span-001", nil, 1)
		task2, _ := createQueryTestTask(repo, "任务2", "trace-002", "span-002", nil, 1)
		task3, _ := createQueryTestTask(repo, "任务3", "trace-003", "span-003", nil, 1)

		result, err := service.ListAllTasks(ctx)
		if err != nil {
			t.Fatalf("获取任务列表失败: %v", err)
		}

		if result.Total != 3 {
			t.Errorf("期望 Total 为 3, 实际为 %d", result.Total)
		}

		if len(result.Tasks) != 3 {
			t.Errorf("期望任务列表长度为 3, 实际为 %d", len(result.Tasks))
		}

		// 验证任务ID是否都在结果中
		ids := make(map[string]bool)
		for _, dto := range result.Tasks {
			ids[dto.ID] = true
		}

		if !ids[task1.ID().String()] {
			t.Error("任务1 不在结果中")
		}
		if !ids[task2.ID().String()] {
			t.Error("任务2 不在结果中")
		}
		if !ids[task3.ID().String()] {
			t.Error("任务3 不在结果中")
		}
	})

	// 测试 DTO 转换正确性
	t.Run("DTO 转换正确性", func(t *testing.T) {
		// 创建一个带有各种状态的任务
		repo2 := newMockQueryTaskRepository()
		service2 := NewQueryService(repo2)

		// 创建不同状态的任务
		createQueryTestTaskWithStatus(repo2, "待处理任务", "trace-101", "span-101", nil, 1, domain.TaskStatusPending)
		createQueryTestTaskWithStatus(repo2, "运行中任务", "trace-102", "span-102", nil, 1, domain.TaskStatusRunning)
		createQueryTestTaskWithStatus(repo2, "已完成任务", "trace-103", "span-103", nil, 1, domain.TaskStatusCompleted)
		createQueryTestTaskWithStatus(repo2, "失败任务", "trace-104", "span-104", nil, 1, domain.TaskStatusFailed)
		createQueryTestTaskWithStatus(repo2, "已取消任务", "trace-105", "span-105", nil, 1, domain.TaskStatusCancelled)

		result, err := service2.ListAllTasks(ctx)
		if err != nil {
			t.Fatalf("获取任务列表失败: %v", err)
		}

		if result.Total != 5 {
			t.Errorf("期望 Total 为 5, 实际为 %d", result.Total)
		}

		// 验证每个任务的状态转换正确
		statusCount := make(map[string]int)
		for _, dto := range result.Tasks {
			statusCount[dto.Status]++
		}

		if statusCount["pending"] != 1 {
			t.Errorf("期望 pending 状态任务数为 1, 实际为 %d", statusCount["pending"])
		}
		if statusCount["running"] != 1 {
			t.Errorf("期望 running 状态任务数为 1, 实际为 %d", statusCount["running"])
		}
		if statusCount["completed"] != 1 {
			t.Errorf("期望 completed 状态任务数为 1, 实际为 %d", statusCount["completed"])
		}
		if statusCount["failed"] != 1 {
			t.Errorf("期望 failed 状态任务数为 1, 实际为 %d", statusCount["failed"])
		}
		if statusCount["cancelled"] != 1 {
			t.Errorf("期望 cancelled 状态任务数为 1, 实际为 %d", statusCount["cancelled"])
		}
	})
}

// TestQueryService_ListTasksByTrace 测试按 TraceID 查询任务
func TestQueryService_ListTasksByTrace(t *testing.T) {
	ctx := context.Background()
	repo := newMockQueryTaskRepository()
	service := NewQueryService(repo)

	// 测试空结果返回
	t.Run("空结果返回", func(t *testing.T) {
		result, err := service.ListTasksByTrace(ctx, domain.NewTraceID("non-existent-trace"))
		if err != nil {
			t.Fatalf("查询任务失败: %v", err)
		}

		if result == nil {
			t.Fatal("返回的结果为 nil")
		}

		if len(result.Tasks) != 0 {
			t.Errorf("期望任务列表长度为 0, 实际为 %d", len(result.Tasks))
		}

		if result.Total != 0 {
			t.Errorf("期望 Total 为 0, 实际为 %d", result.Total)
		}
	})

	// 测试按 TraceID 查询任务
	t.Run("按 TraceID 查询任务", func(t *testing.T) {
		// 创建属于不同 trace 的任务
		createQueryTestTask(repo, "任务A1", "trace-A", "span-A1", nil, 1)
		createQueryTestTask(repo, "任务A2", "trace-A", "span-A2", nil, 1)
		createQueryTestTask(repo, "任务A3", "trace-A", "span-A3", nil, 1)
		createQueryTestTask(repo, "任务B1", "trace-B", "span-B1", nil, 1)
		createQueryTestTask(repo, "任务B2", "trace-B", "span-B2", nil, 1)

		// 查询 trace-A 的任务
		resultA, err := service.ListTasksByTrace(ctx, domain.NewTraceID("trace-A"))
		if err != nil {
			t.Fatalf("查询 trace-A 任务失败: %v", err)
		}

		if resultA.Total != 3 {
			t.Errorf("期望 trace-A 的 Total 为 3, 实际为 %d", resultA.Total)
		}

		for _, dto := range resultA.Tasks {
			if dto.TraceID != "trace-A" {
				t.Errorf("期望所有任务的 TraceID 为 'trace-A', 实际为 %s", dto.TraceID)
			}
		}

		// 查询 trace-B 的任务
		resultB, err := service.ListTasksByTrace(ctx, domain.NewTraceID("trace-B"))
		if err != nil {
			t.Fatalf("查询 trace-B 任务失败: %v", err)
		}

		if resultB.Total != 2 {
			t.Errorf("期望 trace-B 的 Total 为 2, 实际为 %d", resultB.Total)
		}

		// 查询不存在的 trace
		resultC, err := service.ListTasksByTrace(ctx, domain.NewTraceID("trace-C"))
		if err != nil {
			t.Fatalf("查询 trace-C 任务失败: %v", err)
		}

		if resultC.Total != 0 {
			t.Errorf("期望 trace-C 的 Total 为 0, 实际为 %d", resultC.Total)
		}
	})

	// 测试 DTO 列表正确性
	t.Run("DTO 列表正确性", func(t *testing.T) {
		repo2 := newMockQueryTaskRepository()
		service2 := NewQueryService(repo2)

		// 创建带有不同字段的任务
		parentTask, _ := createQueryTestTask(repo2, "父任务", "trace-parent", "span-parent", nil, 1)
		parentID := parentTask.ID()
		createQueryTestTask(repo2, "子任务1", "trace-parent", "span-child1", &parentID, 2)
		createQueryTestTask(repo2, "子任务2", "trace-parent", "span-child2", &parentID, 2)

		result, err := service2.ListTasksByTrace(ctx, domain.NewTraceID("trace-parent"))
		if err != nil {
			t.Fatalf("查询任务失败: %v", err)
		}

		if result.Total != 3 {
			t.Errorf("期望 Total 为 3, 实际为 %d", result.Total)
		}

		// 验证 DTO 字段映射
		for _, dto := range result.Tasks {
			if dto.TraceID != "trace-parent" {
				t.Errorf("期望 TraceID 为 'trace-parent', 实际为 %s", dto.TraceID)
			}

			// 验证 Depth 字段
			if dto.Name == "父任务" && dto.Depth != 1 {
				t.Errorf("期望父任务 Depth 为 1, 实际为 %d", dto.Depth)
			}
			if dto.Name == "子任务1" && dto.Depth != 2 {
				t.Errorf("期望子任务1 Depth 为 2, 实际为 %d", dto.Depth)
			}
		}
	})
}

// TestQueryService_GetTaskTree 测试获取任务树
func TestQueryService_GetTaskTree(t *testing.T) {
	ctx := context.Background()
	repo := newMockQueryTaskRepository()
	service := NewQueryService(repo)

	// 测试空任务列表返回
	t.Run("空任务列表返回", func(t *testing.T) {
		tree, err := service.GetTaskTree(ctx, domain.NewTraceID("empty-trace"))
		if err != nil {
			t.Fatalf("获取任务树失败: %v", err)
		}

		if tree == nil {
			// 返回 nil 也是可接受的
			return
		}

		if len(tree) != 0 {
			t.Errorf("期望树根节点数为 0, 实际为 %d", len(tree))
		}
	})

	// 测试构建单层任务树（无子任务）
	t.Run("构建单层任务树（无子任务）", func(t *testing.T) {
		repo2 := newMockQueryTaskRepository()
		service2 := NewQueryService(repo2)

		// 创建没有子任务的任务
		createQueryTestTask(repo2, "独立任务1", "trace-single", "span-1", nil, 1)
		createQueryTestTask(repo2, "独立任务2", "trace-single", "span-2", nil, 1)

		tree, err := service2.GetTaskTree(ctx, domain.NewTraceID("trace-single"))
		if err != nil {
			t.Fatalf("获取任务树失败: %v", err)
		}

		if len(tree) != 2 {
			t.Errorf("期望树根节点数为 2, 实际为 %d", len(tree))
		}

		// 验证每个根节点没有子节点
		for _, node := range tree {
			if node.Children != nil && len(node.Children) > 0 {
				t.Errorf("期望节点 %s 没有子节点, 实际有 %d 个子节点", node.Task.Name, len(node.Children))
			}
		}
	})

	// 测试构建多层嵌套任务树（带子任务）
	t.Run("构建多层嵌套任务树（带子任务）", func(t *testing.T) {
		repo3 := newMockQueryTaskRepository()
		service3 := NewQueryService(repo3)

		// 创建三层任务树
		// 根任务
		rootTask, _ := createQueryTestTask(repo3, "根任务", "trace-tree", "span-root", nil, 1)
		rootID := rootTask.ID()

		// 第一层子任务
		child1, _ := createQueryTestTask(repo3, "子任务1", "trace-tree", "span-child1", &rootID, 2)
		child2, _ := createQueryTestTask(repo3, "子任务2", "trace-tree", "span-child2", &rootID, 2)
		child1ID := child1.ID()
		child2ID := child2.ID()

		// 第二层子任务
		createQueryTestTask(repo3, "孙任务1-1", "trace-tree", "span-grand1-1", &child1ID, 3)
		createQueryTestTask(repo3, "孙任务1-2", "trace-tree", "span-grand1-2", &child1ID, 3)
		createQueryTestTask(repo3, "孙任务2-1", "trace-tree", "span-grand2-1", &child2ID, 3)

		tree, err := service3.GetTaskTree(ctx, domain.NewTraceID("trace-tree"))
		if err != nil {
			t.Fatalf("获取任务树失败: %v", err)
		}

		// 应该只有一个根节点
		if len(tree) != 1 {
			t.Fatalf("期望树根节点数为 1, 实际为 %d", len(tree))
		}

		rootNode := tree[0]

		// 验证根任务
		if rootNode.Task.Name != "根任务" {
			t.Errorf("期望根任务名称为 '根任务', 实际为 %s", rootNode.Task.Name)
		}

		// 验证第一层子任务
		if len(rootNode.Children) != 2 {
			t.Errorf("期望根任务有 2 个子任务, 实际有 %d 个", len(rootNode.Children))
		}

		// 验证 DTO 转换递归正确性
		// 查找子任务1和子任务2
		var child1Node, child2Node *TaskTreeNodeDTO
		for _, child := range rootNode.Children {
			if child.Task.Name == "子任务1" {
				child1Node = child
			}
			if child.Task.Name == "子任务2" {
				child2Node = child
			}
		}

		if child1Node == nil {
			t.Fatal("未找到子任务1节点")
		}
		if child2Node == nil {
			t.Fatal("未找到子任务2节点")
		}

		// 验证子任务1有2个孙任务
		if len(child1Node.Children) != 2 {
			t.Errorf("期望子任务1有 2 个孙任务, 实际有 %d 个", len(child1Node.Children))
		}

		// 验证子任务2有1个孙任务
		if len(child2Node.Children) != 1 {
			t.Errorf("期望子任务2有 1 个孙任务, 实际有 %d 个", len(child2Node.Children))
		}

		// 验证孙任务的父级 ID 正确
		for _, grandChild := range child1Node.Children {
			if grandChild.Task.ParentID == nil {
				t.Error("期望孙任务的 ParentID 不为 nil")
			} else if *grandChild.Task.ParentID != child1.ID().String() {
				t.Errorf("期望孙任务的 ParentID 为 %s, 实际为 %s", child1.ID().String(), *grandChild.Task.ParentID)
			}
		}
	})
}

// TestTaskTreeBuilder 测试 taskTreeBuilder
func TestTaskTreeBuilder(t *testing.T) {
	ctx := context.Background()
	repo := newMockQueryTaskRepository()
	builder := newTaskTreeBuilder(repo)

	// 测试 Build 方法 - 空列表
	t.Run("Build 方法 - 空列表", func(t *testing.T) {
		nodes, err := builder.Build(ctx, domain.NewTraceID("empty-trace"))
		if err != nil {
			t.Fatalf("构建任务树失败: %v", err)
		}

		// 代码在空列表时可能返回 nil 或空切片，两者都是可接受的
		if nodes != nil && len(nodes) != 0 {
			t.Errorf("期望节点数为 0, 实际为 %d", len(nodes))
		}
	})

	// 测试 buildNode 递归方法
	t.Run("buildNode 递归方法", func(t *testing.T) {
		repo2 := newMockQueryTaskRepository()
		builder2 := newTaskTreeBuilder(repo2)

		// 创建任务链：根 -> 子 -> 孙
		rootTask, _ := createQueryTestTask(repo2, "根节点", "trace-chain", "span-root", nil, 1)
		rootID := rootTask.ID()
		childTask, _ := createQueryTestTask(repo2, "子节点", "trace-chain", "span-child", &rootID, 2)
		childID := childTask.ID()
		grandChildTask, _ := createQueryTestTask(repo2, "孙节点", "trace-chain", "span-grand", &childID, 3)

		// 手动构建 taskMap
		taskMap := map[domain.TaskID]*domain.Task{
			rootTask.ID():      rootTask,
			childTask.ID():     childTask,
			grandChildTask.ID(): grandChildTask,
		}

		// 构建根节点
		rootNode := builder2.buildNode(rootTask, taskMap)

		if rootNode.Task.Name() != "根节点" {
			t.Errorf("期望根节点名称为 '根节点', 实际为 %s", rootNode.Task.Name())
		}

		if len(rootNode.Children) != 1 {
			t.Fatalf("期望根节点有 1 个子节点, 实际有 %d 个", len(rootNode.Children))
		}

		childNode := rootNode.Children[0]
		if childNode.Task.Name() != "子节点" {
			t.Errorf("期望子节点名称为 '子节点', 实际为 %s", childNode.Task.Name())
		}

		if len(childNode.Children) != 1 {
			t.Fatalf("期望子节点有 1 个子节点, 实际有 %d 个", len(childNode.Children))
		}

		grandChildNode := childNode.Children[0]
		if grandChildNode.Task.Name() != "孙节点" {
			t.Errorf("期望孙节点名称为 '孙节点', 实际为 %s", grandChildNode.Task.Name())
		}

		// 孙节点应该没有子节点
		if len(grandChildNode.Children) != 0 {
			t.Errorf("期望孙节点有 0 个子节点, 实际有 %d 个", len(grandChildNode.Children))
		}
	})
}

// TestToGetTaskDTO 测试 toGetTaskDTO 函数
func TestToGetTaskDTO(t *testing.T) {
	repo := newMockQueryTaskRepository()

	t.Run("基本字段转换", func(t *testing.T) {
		task, _ := createQueryTestTask(repo, "测试任务", "trace-001", "span-001", nil, 1)

		dto := toGetTaskDTO(task)

		if dto.ID != task.ID().String() {
			t.Errorf("期望 ID 为 %s, 实际为 %s", task.ID().String(), dto.ID)
		}
		if dto.Name != "测试任务" {
			t.Errorf("期望 Name 为 '测试任务', 实际为 %s", dto.Name)
		}
		if dto.TraceID != "trace-001" {
			t.Errorf("期望 TraceID 为 'trace-001', 实际为 %s", dto.TraceID)
		}
		if dto.Type != "custom" {
			t.Errorf("期望 Type 为 'custom', 实际为 %s", dto.Type)
		}
		if dto.Status != "pending" {
			t.Errorf("期望 Status 为 'pending', 实际为 %s", dto.Status)
		}
	})

	t.Run("ParentID 为 nil 的情况", func(t *testing.T) {
		task, _ := createQueryTestTask(repo, "无父任务", "trace-002", "span-002", nil, 1)

		dto := toGetTaskDTO(task)

		if dto.ParentID != nil {
			t.Error("期望 ParentID 为 nil")
		}
	})

	t.Run("ParentID 不为 nil 的情况", func(t *testing.T) {
		parentID := domain.NewTaskID("parent-001")
		task, _ := createQueryTestTask(repo, "有父任务", "trace-003", "span-003", &parentID, 2)

		dto := toGetTaskDTO(task)

		if dto.ParentID == nil {
			t.Fatal("期望 ParentID 不为 nil")
		}
		if *dto.ParentID != "parent-001" {
			t.Errorf("期望 ParentID 为 'parent-001', 实际为 %s", *dto.ParentID)
		}
	})

	t.Run("Error 字段转换", func(t *testing.T) {
		task, _ := createQueryTestTask(repo, "失败任务", "trace-004", "span-004", nil, 1)
		task.Start()
		task.Fail(errors.New("测试错误信息"))

		dto := toGetTaskDTO(task)

		if dto.Error != "测试错误信息" {
			t.Errorf("期望 Error 为 '测试错误信息', 实际为 %s", dto.Error)
		}
	})

	t.Run("扩展字段转换", func(t *testing.T) {
		task, _ := createQueryTestTask(repo, "完整字段任务", "trace-005", "span-005", nil, 1)

		dto := toGetTaskDTO(task)

		if dto.AcceptanceCriteria != "验收标准: 完整字段任务" {
			t.Errorf("期望 AcceptanceCriteria 为 '验收标准: 完整字段任务', 实际为 %s", dto.AcceptanceCriteria)
		}
		if dto.TaskRequirement != "任务要求: 完整字段任务" {
			t.Errorf("期望 TaskRequirement 为 '任务要求: 完整字段任务', 实际为 %s", dto.TaskRequirement)
		}
		if dto.UserCode != "USER001" {
			t.Errorf("期望 UserCode 为 'USER001', 实际为 %s", dto.UserCode)
		}
		if dto.AgentCode != "AGENT001" {
			t.Errorf("期望 AgentCode 为 'AGENT001', 实际为 %s", dto.AgentCode)
		}
		if dto.ChannelCode != "CHANNEL001" {
			t.Errorf("期望 ChannelCode 为 'CHANNEL001', 实际为 %s", dto.ChannelCode)
		}
		if dto.SessionKey != "SESSION001" {
			t.Errorf("期望 SessionKey 为 'SESSION001', 实际为 %s", dto.SessionKey)
		}
		if dto.TodoList != "待办列表: 完整字段任务" {
			t.Errorf("期望 TodoList 为 '待办列表: 完整字段任务', 实际为 %s", dto.TodoList)
		}
		if dto.Analysis != "分析结果: 完整字段任务" {
			t.Errorf("期望 Analysis 为 '分析结果: 完整字段任务', 实际为 %s", dto.Analysis)
		}
	})

	t.Run("Progress DTO 转换", func(t *testing.T) {
		task, _ := createQueryTestTask(repo, "进度任务", "trace-006", "span-006", nil, 1)
		task.UpdateProgress(50)

		dto := toGetTaskDTO(task)

		if dto.Progress.Value != 50 {
			t.Errorf("期望 Progress.Value 为 50, 实际为 %d", dto.Progress.Value)
		}
		if dto.Progress.UpdatedAt == 0 {
			t.Error("期望 Progress.UpdatedAt 不为 0")
		}
	})
}

// TestToTaskTreeDTOs 测试 toTaskTreeDTOs 函数
func TestToTaskTreeDTOs(t *testing.T) {
	repo := newMockQueryTaskRepository()

	t.Run("nil 输入", func(t *testing.T) {
		result := toTaskTreeDTOs(nil)
		if result != nil {
			t.Error("期望 nil 输入返回 nil")
		}
	})

	t.Run("空切片输入", func(t *testing.T) {
		nodes := []*taskTreeNode{}
		result := toTaskTreeDTOs(nodes)

		if len(result) != 0 {
			t.Errorf("期望返回空切片, 实际长度为 %d", len(result))
		}
	})

	t.Run("单层节点转换", func(t *testing.T) {
		task1, _ := createQueryTestTask(repo, "任务1", "trace-001", "span-001", nil, 1)
		task2, _ := createQueryTestTask(repo, "任务2", "trace-002", "span-002", nil, 1)

		nodes := []*taskTreeNode{
			{Task: task1, Children: nil},
			{Task: task2, Children: nil},
		}

		result := toTaskTreeDTOs(nodes)

		if len(result) != 2 {
			t.Errorf("期望返回 2 个节点, 实际为 %d", len(result))
		}

		if result[0].Task.Name != "任务1" {
			t.Errorf("期望第一个节点任务名称为 '任务1', 实际为 %s", result[0].Task.Name)
		}
		if result[1].Task.Name != "任务2" {
			t.Errorf("期望第二个节点任务名称为 '任务2', 实际为 %s", result[1].Task.Name)
		}
	})

	t.Run("多层嵌套节点转换", func(t *testing.T) {
		task1, _ := createQueryTestTask(repo, "父任务", "trace-003", "span-003", nil, 1)
		parentID := task1.ID()
		task2, _ := createQueryTestTask(repo, "子任务", "trace-004", "span-004", &parentID, 2)

		nodes := []*taskTreeNode{
			{
				Task: task1,
				Children: []*taskTreeNode{
					{Task: task2, Children: nil},
				},
			},
		}

		result := toTaskTreeDTOs(nodes)

		if len(result) != 1 {
			t.Fatalf("期望返回 1 个根节点, 实际为 %d", len(result))
		}

		if result[0].Task.Name != "父任务" {
			t.Errorf("期望根节点任务名称为 '父任务', 实际为 %s", result[0].Task.Name)
		}

		if len(result[0].Children) != 1 {
			t.Fatalf("期望根节点有 1 个子节点, 实际为 %d", len(result[0].Children))
		}

		if result[0].Children[0].Task.Name != "子任务" {
			t.Errorf("期望子节点任务名称为 '子任务', 实际为 %s", result[0].Children[0].Task.Name)
		}
	})
}
