package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/bus"
	"github.com/weibh/taskmanager/infrastructure/llm"
)

type fakeSummaryProvider struct {
	summary string
	prompt  string
}

func (f *fakeSummaryProvider) Generate(ctx context.Context, prompt string) (string, error) {
	f.prompt = prompt
	return f.summary, nil
}

func (f *fakeSummaryProvider) GenerateWithTools(ctx context.Context, prompt string, tools []*llm.ToolRegistry, maxIterations int) (string, []llm.ToolCall, error) {
	return "", nil, nil
}

func (f *fakeSummaryProvider) GenerateSubTasks(ctx context.Context, taskName string, taskDesc string, depth int, maxDepth int) (*llm.SubTaskPlan, error) {
	return &llm.SubTaskPlan{}, nil
}

func (f *fakeSummaryProvider) GetLastUsage() llm.Usage {
	return llm.Usage{}
}

func (f *fakeSummaryProvider) Name() string {
	return "fake-summary-provider"
}

func TestTaskSummarizer_HandlePendingSummary_WithChildrenAndEmptyRecords(t *testing.T) {
	ctx := context.Background()
	repo := newMockTaskRepository()
	eventBus := bus.NewEventBus()

	parent := mustCreateRunningTask(t, "task-parent", nil, "父任务", "整体验收")
	childA := mustCreateCompletedTask(t, "task-child-a", parent.ID(), "子任务A", "A结论")
	childB := mustCreateCompletedTask(t, "task-child-b", parent.ID(), "子任务B", "B结论")

	todo := NewTodoList(parent.ID().String())
	todo.AddItem(childA.ID().String(), childA.Name(), childA.Type().String(), "span-a", TodoStatusCompleted)
	todo.AddItem(childB.ID().String(), childB.Name(), childB.Type().String(), "span-b", TodoStatusCompleted)
	parent.SetTodoList(todo.ToJSON())
	if err := parent.PendingSummary(); err != nil {
		t.Fatalf("父任务进入 PendingSummary 失败: %v", err)
	}

	if err := repo.Save(ctx, parent); err != nil {
		t.Fatalf("保存父任务失败: %v", err)
	}
	if err := repo.Save(ctx, childA); err != nil {
		t.Fatalf("保存子任务 A 失败: %v", err)
	}
	if err := repo.Save(ctx, childB); err != nil {
		t.Fatalf("保存子任务 B 失败: %v", err)
	}

	provider := &fakeSummaryProvider{summary: "父任务综合总结"}
	summarizer := NewTaskSummarizer(repo, nil, eventBus)
	summarizer.providerResolver = func(ctx context.Context, task *domain.Task) (llm.LLMProvider, error) {
		return provider, nil
	}

	summarizer.handlePendingSummary(domain.NewTaskPendingSummaryEvent(parent))

	updatedParent, err := repo.FindByID(ctx, parent.ID())
	if err != nil {
		t.Fatalf("重新读取父任务失败: %v", err)
	}

	if updatedParent.Status() != domain.TaskStatusCompleted {
		t.Fatalf("期望父任务为 Completed，实际为 %s", updatedParent.Status())
	}
	if updatedParent.TaskConclusion() != "父任务综合总结" {
		t.Fatalf("期望父任务总结为 LLM 输出，实际为 %s", updatedParent.TaskConclusion())
	}

	pairs, err := domain.ParseTaskResultPairs(updatedParent.SubtaskRecords())
	if err != nil {
		t.Fatalf("解析父任务 subtask_records 失败: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("期望父任务收集 2 条子任务结果，实际为 %d", len(pairs))
	}
	if pairs[0].TaskID == "" || pairs[1].TaskID == "" {
		t.Fatalf("期望子任务成对文档包含 task_id 字段")
	}
	if !strings.Contains(provider.prompt, "A结论") || !strings.Contains(provider.prompt, "B结论") {
		t.Fatalf("期望总结 prompt 包含全部子任务结论，实际 prompt: %s", provider.prompt)
	}
}

func TestTaskSummarizer_NotifyParentToCollect_UpsertWithoutDuplicate(t *testing.T) {
	ctx := context.Background()
	repo := newMockTaskRepository()
	eventBus := bus.NewEventBus()

	parent := mustCreateRunningTask(t, "task-parent-upsert", nil, "父任务", "整体验收")
	child := mustCreateCompletedTask(t, "task-child-upsert", parent.ID(), "子任务", "首次结论")

	if err := repo.Save(ctx, parent); err != nil {
		t.Fatalf("保存父任务失败: %v", err)
	}
	if err := repo.Save(ctx, child); err != nil {
		t.Fatalf("保存子任务失败: %v", err)
	}

	summarizer := NewTaskSummarizer(repo, nil, eventBus)
	parentID := parent.ID()
	summarizer.notifyParentToCollect(ctx, child, &parentID)

	child.SetTaskConclusion("更新后结论")
	if err := repo.Save(ctx, child); err != nil {
		t.Fatalf("更新子任务失败: %v", err)
	}
	summarizer.notifyParentToCollect(ctx, child, &parentID)

	updatedParent, err := repo.FindByID(ctx, parent.ID())
	if err != nil {
		t.Fatalf("重新读取父任务失败: %v", err)
	}

	pairs, err := domain.ParseTaskResultPairs(updatedParent.SubtaskRecords())
	if err != nil {
		t.Fatalf("解析父任务 subtask_records 失败: %v", err)
	}
	if len(pairs) != 1 {
		t.Fatalf("期望父任务只保留 1 条子任务记录，实际为 %d", len(pairs))
	}
	if pairs[0].TaskConclusion != "更新后结论" {
		t.Fatalf("期望父任务记录为子任务最新结论，实际为 %s", pairs[0].TaskConclusion)
	}
}

func mustCreateRunningTask(t *testing.T, id string, parentID *domain.TaskID, name string, acceptance string) *domain.Task {
	t.Helper()
	task, err := domain.NewTask(
		domain.NewTaskID(id),
		domain.NewTraceID("trace-summary"),
		domain.NewSpanID("span-"+id),
		parentID,
		name,
		"",
		domain.TaskTypeCustom,
		name+"-要求",
		acceptance,
		time.Minute,
		0,
		0,
	)
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	if err := task.Start(); err != nil {
		t.Fatalf("启动任务失败: %v", err)
	}
	return task
}

func mustCreateCompletedTask(t *testing.T, id string, parentID domain.TaskID, name string, conclusion string) *domain.Task {
	t.Helper()
	pid := parentID
	task := mustCreateRunningTask(t, id, &pid, name, name+"-验收")
	task.SetTaskConclusion(conclusion)
	if err := task.Complete(); err != nil {
		t.Fatalf("完成任务失败: %v", err)
	}
	return task
}
