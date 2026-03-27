/**
 * 子任务 Trace 关联关系集成测试
 */
package application

import (
	"context"
	"testing"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/trace"
)

func TestSubTaskTraceChain(t *testing.T) {
	// 模拟 processor.Process 中的初始设置
	ctx := context.Background()

	// 1. 模拟根任务创建时的 trace 上下文（StartTrace）
	ctx, traceID, rootSpanID := trace.StartTrace(ctx)

	t.Logf("根任务 - traceID: %s, spanID: %s", traceID, rootSpanID)

	// 2. 模拟 CreateTaskTool.Execute 创建根任务
	rootTask := createTaskForTest(ctx, "root-task", t)
	if rootTask.TraceID().String() != traceID {
		t.Errorf("根任务 traceID 期望 %s, 实际 %s", traceID, rootTask.TraceID().String())
	}
	if rootTask.SpanID().String() != rootSpanID {
		t.Errorf("根任务 spanID 期望 %s, 实际 %s", rootSpanID, rootTask.SpanID().String())
	}

	// 3. 模拟 LLM 生成子任务，调用 StartSpan 创建子任务上下文
	subCtx, subSpanID := trace.StartSpan(ctx)
	subParentSpanID := trace.GetParentSpanID(subCtx)

	t.Logf("子任务 - traceID: %s, spanID: %s, parentSpanID: %s",
		trace.GetTraceID(subCtx), subSpanID, subParentSpanID)

	// 验证子任务的 parentSpanID 等于根任务的 spanID
	if subParentSpanID != rootSpanID {
		t.Errorf("子任务 parentSpanID %s != 根任务 spanID %s", subParentSpanID, rootSpanID)
	}

	// 4. 模拟 CreateTaskTool.Execute 创建子任务
	subTask := createTaskForTest(subCtx, "sub-task", t)
	if subTask.TraceID().String() != traceID {
		t.Errorf("子任务 traceID 期望 %s, 实际 %s", traceID, subTask.TraceID().String())
	}
	if subTask.SpanID().String() != subSpanID {
		t.Errorf("子任务 spanID 期望 %s, 实际 %s", subSpanID, subTask.SpanID().String())
	}
	if subTask.ParentSpan() != rootSpanID {
		t.Errorf("子任务 ParentSpan 期望 %s, 实际 %s", rootSpanID, subTask.ParentSpan())
	}

	// 5. 模拟子子任务（更深的层级）
	ctx, subSubSpanID := trace.StartSpan(subCtx)
	subSubParentSpanID := trace.GetParentSpanID(ctx)

	t.Logf("子子任务 - traceID: %s, spanID: %s, parentSpanID: %s",
		trace.GetTraceID(ctx), subSubSpanID, subSubParentSpanID)

	// 验证子子任务的 parentSpanID 等于子任务的 spanID
	if subSubParentSpanID != subSpanID {
		t.Errorf("子子任务 parentSpanID %s != 子任务 spanID %s", subSubParentSpanID, subSpanID)
	}
}

func TestSubTaskTraceChain_ThreeLevels(t *testing.T) {
	ctx := context.Background()

	// Level 1: 根任务
	ctx, traceID, level1SpanID := trace.StartTrace(ctx)

	// Level 2: 子任务
	ctx, level2SpanID := trace.StartSpan(ctx)

	// Level 3: 子子任务
	ctx, level3SpanID := trace.StartSpan(ctx)

	// 验证 traceID 在所有层级保持一致
	currentTraceID := trace.GetTraceID(ctx)
	if currentTraceID != traceID {
		t.Errorf("Level 3 traceID 期望 %s, 实际 %s", traceID, currentTraceID)
	}

	// 验证各层级 spanID 都不同
	allSpanIDs := map[string]bool{
		level1SpanID: true,
		level2SpanID: true,
		level3SpanID: true,
	}
	if len(allSpanIDs) != 3 {
		t.Error("三个层级的 spanID 应该都不同")
	}

	// 验证 ctx 中的 spanID 是当前层级的
	if currentSpanID := trace.MustGetSpanID(ctx); currentSpanID != level3SpanID {
		t.Errorf("Level 3 spanID 期望 %s, 实际 %s", level3SpanID, currentSpanID)
	}
}

func TestTraceContext_ExtractFromCtx(t *testing.T) {
	// 模拟 CreateTaskTool.Execute 的 ctx 提取逻辑
	ctx := context.Background()
	ctx, traceID, spanID := trace.StartTrace(ctx)

	// 模拟 CreateTaskTool 从 ctx 提取
	extractedTraceID := trace.GetTraceID(ctx)
	extractedSpanID := trace.MustGetSpanID(ctx)

	if extractedTraceID != traceID {
		t.Errorf("提取的 traceID 期望 %s, 实际 %s", traceID, extractedTraceID)
	}
	if extractedSpanID != spanID {
		t.Errorf("提取的 spanID 期望 %s, 实际 %s", spanID, extractedSpanID)
	}

	// 验证 parentSpanID 为空（根任务没有父）
	parentSpanID := trace.GetParentSpanID(ctx)
	if parentSpanID != "" {
		t.Errorf("根任务 parentSpanID 应为空, 实际 %s", parentSpanID)
	}
}

func TestTraceContext_ExtractParentSpanID(t *testing.T) {
	// 模拟子任务从 ctx 提取 parentSpanID
	ctx := context.Background()
	ctx, traceID, parentSpanID := trace.StartTrace(ctx)

	// 创建子任务上下文
	ctx, childSpanID := trace.StartSpan(ctx)

	// 模拟 CreateTaskTool 从 ctx 提取
	extractedTraceID := trace.GetTraceID(ctx)
	extractedSpanID := trace.MustGetSpanID(ctx)
	extractedParentSpanID := trace.GetParentSpanID(ctx)

	if extractedTraceID != traceID {
		t.Errorf("提取的 traceID 期望 %s, 实际 %s", traceID, extractedTraceID)
	}
	if extractedSpanID != childSpanID {
		t.Errorf("提取的 spanID 期望 %s, 实际 %s", childSpanID, extractedSpanID)
	}
	if extractedParentSpanID != parentSpanID {
		t.Errorf("提取的 parentSpanID 期望 %s, 实际 %s", parentSpanID, extractedParentSpanID)
	}
}

// createTaskForTest 创建测试用任务（简化版本）
func createTaskForTest(ctx context.Context, name string, t *testing.T) *domain.Task {
	traceIDStr := trace.GetTraceID(ctx)
	spanIDStr := trace.MustGetSpanID(ctx)
	parentSpanID := trace.GetParentSpanID(ctx)

	taskID := domain.NewTaskID(name + "-id")
	traceID := domain.NewTraceID(traceIDStr)
	spanID := domain.NewSpanID(spanIDStr)

	task, err := domain.NewTask(
		taskID,
		traceID,
		spanID,
		nil, // parentID
		name,
		"",
		domain.TaskTypeAgent,
		"目标",
		"验收标准",
		60000,
		0,
		0,
	)
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}

	// 设置 parentSpan（模拟从 ctx 提取）
	if parentSpanID != "" {
		task.SetParentSpan(parentSpanID)
	}

	return task
}