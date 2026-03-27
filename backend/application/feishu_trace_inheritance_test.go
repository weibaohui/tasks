/**
 * 飞书对话创建任务时 Trace 继承测试
 * 模拟从飞书消息创建任务时 traceID/spanID/parentSpanID 的继承关系
 */
package application

import (
	"context"
	"testing"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/trace"
)

func TestCreateTaskFromFeishuConversation(t *testing.T) {
	// 模拟飞书消息携带的 trace 信息
	feishuTraceID := "feishu-trace-abc123"
	feishuSpanID := "feishu-span-xyz789"

	// 1. 模拟 MessageProcessor.Process 设置 context（飞书消息进入时）
	ctx := context.Background()
	ctx = trace.WithTraceID(ctx, feishuTraceID)
	ctx = trace.WithSpanID(ctx, feishuSpanID)
	ctx = trace.WithSessionInfo(ctx, "feishu:chat-123", "feishu")
	ctx = trace.WithChannelCode(ctx, "channel-code-001")
	ctx = trace.WithUserCode(ctx, "user-001")
	ctx = trace.WithAgentCode(ctx, "agent-001")

	t.Logf("飞书消息上下文 - traceID: %s, spanID: %s", feishuTraceID, feishuSpanID)

	// 2. 模拟 LLM 调用 CreateTaskTool 创建任务
	// CreateTaskTool.Execute 从 ctx 中提取 trace 信息
	traceIDStr := trace.GetTraceID(ctx)
	spanIDStr := trace.MustGetSpanID(ctx)
	parentSpanID := trace.GetParentSpanID(ctx)

	t.Logf("CreateTaskTool 提取 - traceID: %s, spanID: %s, parentSpanID: '%s'",
		traceIDStr, spanIDStr, parentSpanID)

	// 3. 创建根任务（继承飞书会话的 trace）
	rootTask := createTaskFromCtx(ctx, "feishu-task-root", nil, t)

	t.Logf("根任务 - traceID: %s, spanID: %s, parentSpan: '%s'",
		rootTask.TraceID().String(), rootTask.SpanID().String(), rootTask.ParentSpan())

	// 验证根任务继承飞书会话的 trace
	if rootTask.TraceID().String() != feishuTraceID {
		t.Errorf("根任务 traceID 应继承飞书会话 %s, 实际为 %s", feishuTraceID, rootTask.TraceID().String())
	}
	if rootTask.SpanID().String() != spanIDStr {
		t.Errorf("根任务 spanID 应为 %s, 实际为 %s", spanIDStr, rootTask.SpanID().String())
	}
	if rootTask.ParentSpan() != "" {
		t.Errorf("根任务 parentSpan 应为空, 实际为 '%s'", rootTask.ParentSpan())
	}

	// 4. 模拟 LLM 生成子任务，调用 StartSpan 创建新的 span
	ctx, subSpanID := trace.StartSpan(ctx)
	subParentSpanID := trace.GetParentSpanID(ctx)

	t.Logf("子任务上下文 - traceID: %s, spanID: %s, parentSpanID: %s",
		trace.GetTraceID(ctx), subSpanID, subParentSpanID)

	// 验证子任务的 parentSpanID 等于根任务的 spanID
	if subParentSpanID != rootTask.SpanID().String() {
		t.Errorf("子任务 parentSpanID %s != 根任务 spanID %s", subParentSpanID, rootTask.SpanID().String())
	}

	// 5. 创建子任务
	subTask := createTaskFromCtx(ctx, "feishu-task-sub", ptrTaskID(rootTask.ID()), t)

	t.Logf("子任务 - traceID: %s, spanID: %s, parentSpan: '%s'",
		subTask.TraceID().String(), subTask.SpanID().String(), subTask.ParentSpan())

	// 验证子任务
	if subTask.TraceID().String() != feishuTraceID {
		t.Errorf("子任务 traceID 应为 %s, 实际为 %s", feishuTraceID, subTask.TraceID().String())
	}
	if subTask.SpanID().String() != subSpanID {
		t.Errorf("子任务 spanID 应为 %s, 实际为 %s", subSpanID, subTask.SpanID().String())
	}
	if subTask.ParentSpan() != rootTask.SpanID().String() {
		t.Errorf("子任务 parentSpan '%s' != 根任务 spanID '%s'", subTask.ParentSpan(), rootTask.SpanID().String())
	}
}

func TestCreateTaskFromFeishuConversation_MultiLevelSubTasks(t *testing.T) {
	// 简化测试：只测试两级（根任务 -> 子任务）
	// 这样更容易调试问题

	feishuTraceID := "feishu-trace-multi"
	feishuSpanID := "feishu-span-init"

	ctx := context.Background()
	ctx = trace.WithTraceID(ctx, feishuTraceID)
	ctx = trace.WithSpanID(ctx, feishuSpanID)

	t.Logf("飞书会话初始 - traceID: %s, spanID: %s", feishuTraceID, feishuSpanID)

	// 创建根任务
	rootTask := createTaskFromCtx(ctx, "root", nil, t)
	t.Logf("根任务 - spanID: %s, parentSpan: '%s'", rootTask.SpanID().String(), rootTask.ParentSpan())

	// 验证根任务
	if rootTask.TraceID().String() != feishuTraceID {
		t.Errorf("根任务 traceID 不一致")
	}
	if rootTask.ParentSpan() != "" {
		t.Errorf("根任务 parentSpan 应为空")
	}

	// 第一层子任务
	ctx, level1SpanID := trace.StartSpan(ctx)
	level1ParentSpan := trace.GetParentSpanID(ctx)
	t.Logf("创建 level-1 - newSpanID: %s, parentSpanID: %s (期望根任务span: %s)",
		level1SpanID, level1ParentSpan, rootTask.SpanID().String())

	if level1ParentSpan != rootTask.SpanID().String() {
		t.Errorf("level-1 parentSpanID 不正确: 期望 %s, 实际 %s",
			rootTask.SpanID().String(), level1ParentSpan)
	}

	level1Task := createTaskFromCtx(ctx, "level-1", ptrTaskID(rootTask.ID()), t)
	t.Logf("level-1 任务 - spanID: %s, parentSpan: '%s'",
		level1Task.SpanID().String(), level1Task.ParentSpan())

	if level1Task.ParentSpan() != rootTask.SpanID().String() {
		t.Errorf("level-1 task.ParentSpan() 不正确: 期望 %s, 实际 %s",
			rootTask.SpanID().String(), level1Task.ParentSpan())
	}

	// 第二层子任务
	ctx, level2SpanID := trace.StartSpan(ctx)
	level2ParentSpan := trace.GetParentSpanID(ctx)
	t.Logf("创建 level-2 - newSpanID: %s, parentSpanID: %s (期望level-1 span: %s)",
		level2SpanID, level2ParentSpan, level1Task.SpanID().String())

	if level2ParentSpan != level1Task.SpanID().String() {
		t.Errorf("level-2 parentSpanID 不正确: 期望 %s, 实际 %s",
			level1Task.SpanID().String(), level2ParentSpan)
	}

	level2Task := createTaskFromCtx(ctx, "level-2", ptrTaskID(level1Task.ID()), t)
	t.Logf("level-2 任务 - spanID: %s, parentSpan: '%s'",
		level2Task.SpanID().String(), level2Task.ParentSpan())

	if level2Task.ParentSpan() != level1Task.SpanID().String() {
		t.Errorf("level-2 task.ParentSpan() 不正确: 期望 %s, 实际 %s",
			level1Task.SpanID().String(), level2Task.ParentSpan())
	}

	t.Log("\n=== 多级继承关系验证通过 ===")
}

func TestTraceContext_FromFeishuMetadata(t *testing.T) {
	// 模拟飞书消息的 metadata 中携带的 trace 信息
	// 这是 Feishu 消息处理器如何从消息中提取 trace 信息的方式

	// 场景1: 新会话，没有历史 trace
	t.Run("新会话生成新Trace", func(t *testing.T) {
		ctx := context.Background()

		// 如果消息 metadata 中没有 traceID，则生成新的
		// 这模拟了第一消息进入时的处理
		if traceID := trace.GetTraceID(ctx); traceID == "" {
			ctx = trace.WithTraceID(ctx, trace.NewTraceID())
		}
		if spanID := trace.MustGetSpanID(ctx); spanID == "" {
			ctx = trace.WithSpanID(ctx, trace.NewSpanID())
		}

		traceIDStr := trace.GetTraceID(ctx)
		spanIDStr := trace.MustGetSpanID(ctx)

		t.Logf("新会话 - traceID: %s, spanID: %s", traceIDStr, spanIDStr)

		if traceIDStr == "" {
			t.Error("traceID 不应为空")
		}
		if spanIDStr == "" {
			t.Error("spanID 不应为空")
		}
	})

	// 场景2: 已有会话，从会话中继承 trace
	t.Run("已有会话继承Trace", func(t *testing.T) {
		existingTraceID := "existing-trace-12345"
		existingSpanID := "existing-span-67890"

		ctx := context.Background()
		ctx = trace.WithTraceID(ctx, existingTraceID)
		ctx = trace.WithSpanID(ctx, existingSpanID)

		// 模拟从会话历史中获取 trace
		traceIDStr := trace.GetTraceID(ctx)
		spanIDStr := trace.MustGetSpanID(ctx)

		t.Logf("已有会话 - traceID: %s, spanID: %s", traceIDStr, spanIDStr)

		if traceIDStr != existingTraceID {
			t.Errorf("traceID 应为 %s, 实际为 %s", existingTraceID, traceIDStr)
		}
		if spanIDStr != existingSpanID {
			t.Errorf("spanID 应为 %s, 实际为 %s", existingSpanID, spanIDStr)
		}
	})

	// 场景3: 创建子任务时，StartSpan 自动设置 parentSpanID
	t.Run("子任务自动继承ParentSpan", func(t *testing.T) {
		ctx := context.Background()
		ctx, traceID, parentSpanID := trace.StartTrace(ctx)

		t.Logf("父任务 - traceID: %s, spanID: %s", traceID, parentSpanID)

		// 创建子任务上下文
		ctx, childSpanID := trace.StartSpan(ctx)
		childParentSpanID := trace.GetParentSpanID(ctx)

		t.Logf("子任务 - traceID: %s, spanID: %s, parentSpanID: %s",
			trace.GetTraceID(ctx), childSpanID, childParentSpanID)

		if childParentSpanID != parentSpanID {
			t.Errorf("子任务 parentSpanID 应为 %s, 实际为 %s", parentSpanID, childParentSpanID)
		}
	})
}

// createTaskFromCtx 模拟 CreateTaskTool.Execute 从 ctx 创建任务
func createTaskFromCtx(ctx context.Context, name string, parentID *domain.TaskID, t *testing.T) *domain.Task {
	traceIDStr := trace.GetTraceID(ctx)
	spanIDStr := trace.MustGetSpanID(ctx)
	parentSpanID := trace.GetParentSpanID(ctx)

	taskID := domain.NewTaskID("task-" + name)
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
