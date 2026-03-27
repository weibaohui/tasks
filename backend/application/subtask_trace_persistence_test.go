/**
 * 子任务 Trace 关联关系持久化测试
 * 使用 Mock ID Generator 验证数据库中的 trace 链路关系
 */
package application

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/persistence"
	"github.com/weibh/taskmanager/infrastructure/trace"
)

func TestSubTaskTraceChain_DBIntegration(t *testing.T) {
	// 使用 Mock ID Generator 生成确定性的 ID
	mockGen := newMockIDGenerator()
	trace.SetIDGenerator(mockGen)
	defer trace.ResetIDGenerator()

	// 创建内存数据库
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("创建数据库失败: %v", err)
	}
	defer db.Close()

	if err := persistence.InitSchema(db); err != nil {
		t.Fatalf("初始化 Schema 失败: %v", err)
	}

	repo := persistence.NewSQLiteTaskRepository(db)
	ctx := context.Background()

	// 1. 创建根任务
	resetMockCounter()
	ctx, traceID, rootSpanID := trace.StartTrace(ctx)
	t.Logf("根任务 - traceID: %s, spanID: %s", traceID, rootSpanID)

	rootTask := createTaskWithMock(ctx, "root-task", nil, t)
	if err := repo.Save(ctx, rootTask); err != nil {
		t.Fatalf("保存根任务失败: %v", err)
	}

	// 2. 创建子任务
	resetMockCounter()
	ctx, subSpanID := trace.StartSpan(ctx)
	subParentSpanID := trace.GetParentSpanID(ctx)
	t.Logf("子任务 - traceID: %s, spanID: %s, parentSpanID: %s",
		trace.GetTraceID(ctx), subSpanID, subParentSpanID)

	subTask := createTaskWithMock(ctx, "sub-task", ptrTaskID(rootTask.ID()), t)
	if err := repo.Save(ctx, subTask); err != nil {
		t.Fatalf("保存子任务失败: %v", err)
	}

	// 3. 创建子子任务
	resetMockCounter()
	ctx, subSubSpanID := trace.StartSpan(ctx)
	subSubParentSpanID := trace.GetParentSpanID(ctx)
	t.Logf("子子任务 - traceID: %s, spanID: %s, parentSpanID: %s",
		trace.GetTraceID(ctx), subSubSpanID, subSubParentSpanID)

	subSubTask := createTaskWithMock(ctx, "sub-sub-task", ptrTaskID(subTask.ID()), t)
	if err := repo.Save(ctx, subSubTask); err != nil {
		t.Fatalf("保存子子任务失败: %v", err)
	}

	// 4. 从数据库重新读取，验证关系
	t.Log("\n=== 从数据库重新读取验证 ===")

	// 读取根任务
	savedRoot, err := repo.FindByID(ctx, rootTask.ID())
	if err != nil {
		t.Fatalf("读取根任务失败: %v", err)
	}
	t.Logf("根任务 DB - traceID: %s, spanID: %s, parentSpan: '%s'",
		savedRoot.TraceID().String(), savedRoot.SpanID().String(), savedRoot.ParentSpan())

	// 读取子任务
	savedSub, err := repo.FindByID(ctx, subTask.ID())
	if err != nil {
		t.Fatalf("读取子任务失败: %v", err)
	}
	t.Logf("子任务 DB - traceID: %s, spanID: %s, parentSpan: %s",
		savedSub.TraceID().String(), savedSub.SpanID().String(), savedSub.ParentSpan())

	// 读取子子任务
	savedSubSub, err := repo.FindByID(ctx, subSubTask.ID())
	if err != nil {
		t.Fatalf("读取子子任务失败: %v", err)
	}
	t.Logf("子子任务 DB - traceID: %s, spanID: %s, parentSpan: %s",
		savedSubSub.TraceID().String(), savedSubSub.SpanID().String(), savedSubSub.ParentSpan())

	// 5. 验证关系
	t.Log("\n=== 验证关系 ===")

	// 验证 traceID 一致
	if savedRoot.TraceID().String() != savedSub.TraceID().String() {
		t.Errorf("根任务和子任务的 traceID 不一致: %s vs %s",
			savedRoot.TraceID().String(), savedSub.TraceID().String())
	}
	if savedSub.TraceID().String() != savedSubSub.TraceID().String() {
		t.Errorf("子任务和子子任务的 traceID 不一致: %s vs %s",
			savedSub.TraceID().String(), savedSubSub.TraceID().String())
	}

	// 验证 spanID 各层级不同
	if savedRoot.SpanID().String() == savedSub.SpanID().String() {
		t.Error("根任务和子任务的 spanID 不应该相同")
	}
	if savedSub.SpanID().String() == savedSubSub.SpanID().String() {
		t.Error("子任务和子子任务的 spanID 不应该相同")
	}

	// 验证 parentSpan 关系
	if savedSub.ParentSpan() != savedRoot.SpanID().String() {
		t.Errorf("子任务的 parentSpan '%s' != 根任务的 spanID '%s'",
			savedSub.ParentSpan(), savedRoot.SpanID().String())
	}
	if savedSubSub.ParentSpan() != savedSub.SpanID().String() {
		t.Errorf("子子任务的 parentSpan '%s' != 子任务的 spanID '%s'",
			savedSubSub.ParentSpan(), savedSub.SpanID().String())
	}

	t.Log("\n=== 验证通过 ===")
}

func TestTraceChain_BuildTree(t *testing.T) {
	mockGen := newMockIDGenerator()
	trace.SetIDGenerator(mockGen)
	defer trace.ResetIDGenerator()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("创建数据库失败: %v", err)
	}
	defer db.Close()

	if err := persistence.InitSchema(db); err != nil {
		t.Fatalf("初始化 Schema 失败: %v", err)
	}

	repo := persistence.NewSQLiteTaskRepository(db)
	ctx := context.Background()

	// 创建三级任务链
	resetMockCounter()
	ctx, traceID, _ := trace.StartTrace(ctx)
	rootTask := createTaskWithMock(ctx, "root", nil, t)
	repo.Save(ctx, rootTask)

	resetMockCounter()
	ctx, _ = trace.StartSpan(ctx)
	subTask := createTaskWithMock(ctx, "sub", ptrTaskID(rootTask.ID()), t)
	repo.Save(ctx, subTask)

	resetMockCounter()
	ctx, _ = trace.StartSpan(ctx)
	subSubTask := createTaskWithMock(ctx, "sub-sub", ptrTaskID(subTask.ID()), t)
	repo.Save(ctx, subSubTask)

	// 按 traceID 查询所有任务
	tasks, err := repo.FindByTraceID(ctx, domain.NewTraceID(traceID))
	if err != nil {
		t.Fatalf("按 traceID 查询失败: %v", err)
	}

	t.Logf("找到 %d 个任务", len(tasks))

	// 构建树结构
	var root *taskNode
	levelMap := make(map[string]*taskNode)

	for _, task := range tasks {
		level := 0
		if task.ParentSpan() != "" {
			// 找到父任务
			for _, p := range tasks {
				if p.SpanID().String() == task.ParentSpan() {
					if parentNode, ok := levelMap[p.ID().String()]; ok {
						level = parentNode.level + 1
					}
					break
				}
			}
		}
		node := &taskNode{task: task, level: level}
		levelMap[task.ID().String()] = node
		if level == 0 {
			root = node
		}
	}

	// 打印树结构
	t.Log("任务树结构:")
	printTaskTree(t, root, tasks)

	// 验证树结构
	if root == nil {
		t.Fatal("未找到根任务")
	}
	if root.level != 0 {
		t.Errorf("根任务 level 应该是 0, 实际为 %d", root.level)
	}
}

type taskNode struct {
	task  *domain.Task
	level int
}

func printTaskTree(t *testing.T, node *taskNode, allTasks []*domain.Task) {
	if node == nil {
		return
	}
	prefix := ""
	for i := 0; i < node.level; i++ {
		prefix += "  "
	}
	t.Logf("%s[%s] span=%s parentSpan=%s",
		prefix, node.task.Name(), node.task.SpanID().String(), node.task.ParentSpan())

	// 找到子任务
	for _, child := range allTasks {
		if child.ParentSpan() == node.task.SpanID().String() {
			printTaskTree(t, &taskNode{task: child, level: node.level + 1}, allTasks)
		}
	}
}

// ptrTaskID returns pointer to TaskID
func ptrTaskID(id domain.TaskID) *domain.TaskID {
	return &id
}

// mockIDGeneratorImpl 模拟 ID 生成器
type mockIDGeneratorImpl struct {
	counter uint64
}

func newMockIDGenerator() *mockIDGeneratorImpl {
	return &mockIDGeneratorImpl{counter: 0}
}

func (g *mockIDGeneratorImpl) Generate() string {
	return g.NewSpanID()
}

func (g *mockIDGeneratorImpl) NewTraceID() string {
	g.counter++
	return "trace-" + u64toa(g.counter)
}

func (g *mockIDGeneratorImpl) NewSpanID() string {
	g.counter++
	return "span-" + u64toa(g.counter)
}

func u64toa(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

var mockCounter uint64

func resetMockCounter() {
	mockCounter = 0
}

// createTaskWithMock 使用 mock ID 创建任务
func createTaskWithMock(ctx context.Context, name string, parentID *domain.TaskID, t *testing.T) *domain.Task {
	traceIDStr := trace.GetTraceID(ctx)
	spanIDStr := trace.MustGetSpanID(ctx)
	parentSpanID := trace.GetParentSpanID(ctx)

	taskIDStr := "task-" + name
	taskID := domain.NewTaskID(taskIDStr)
	td := domain.NewTraceID(traceIDStr)
	spanID := domain.NewSpanID(spanIDStr)

	task, err := domain.NewTask(
		taskID,
		td,
		spanID,
		parentID,
		name,
		"",
		domain.TaskTypeAgent,
		"目标: "+name,
		"验收标准: "+name,
		60000,
		0,
		0,
	)
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}

	if parentSpanID != "" {
		task.SetParentSpan(parentSpanID)
	}

	return task
}
