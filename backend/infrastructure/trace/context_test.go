/**
 * Trace Context 单元测试
 */
package trace

import (
	"context"
	"testing"
)

func TestStartTrace(t *testing.T) {
	ctx := context.Background()

	ctx, traceID, spanID := StartTrace(ctx)

	if traceID == "" {
		t.Error("期望 traceID 不为空")
	}

	if spanID == "" {
		t.Error("期望 spanID 不为空")
	}

	// 验证 ctx 中存储了正确的值
	if gotTraceID := GetTraceID(ctx); gotTraceID != traceID {
		t.Errorf("期望 traceID 为 %s, 实际为 %s", traceID, gotTraceID)
	}

	if gotSpanID := MustGetSpanID(ctx); gotSpanID != spanID {
		t.Errorf("期望 spanID 为 %s, 实际为 %s", spanID, gotSpanID)
	}
}

func TestStartSpan(t *testing.T) {
	ctx := context.Background()

	// 1. 先创建 trace 和初始 span
	ctx, traceID, parentSpanID := StartTrace(ctx)

	// 2. 从初始 span 启动新的 span（模拟子任务）
	childCtx, childSpanID := StartSpan(ctx)

	// 验证新的 spanID 不等于 parentSpanID
	if childSpanID == parentSpanID {
		t.Error("子 spanID 不应等于父 spanID")
	}

	// 验证 traceID 保持不变
	if gotTraceID := GetTraceID(childCtx); gotTraceID != traceID {
		t.Errorf("期望 traceID 为 %s, 实际为 %s", traceID, gotTraceID)
	}

	// 验证子 spanID 已更新
	if gotChildSpanID := MustGetSpanID(childCtx); gotChildSpanID != childSpanID {
		t.Errorf("期望 child spanID 为 %s, 实际为 %s", childSpanID, gotChildSpanID)
	}

	// 验证 parentSpanID 已设置
	if gotParentSpanID := GetParentSpanID(childCtx); gotParentSpanID != parentSpanID {
		t.Errorf("期望 parentSpanID 为 %s, 实际为 %s", parentSpanID, gotParentSpanID)
	}
}

func TestStartSpan_ThreeLevels(t *testing.T) {
	ctx := context.Background()

	// Level 1: 根任务
	ctx, traceID, level1SpanID := StartTrace(ctx)

	// Level 2: 子任务
	ctx, level2SpanID := StartSpan(ctx)

	// Level 3: 子子任务
	ctx, level3SpanID := StartSpan(ctx)

	// 验证 traceID 在所有层级保持一致
	if gotTraceID := GetTraceID(ctx); gotTraceID != traceID {
		t.Errorf("期望 traceID 为 %s, 实际为 %s", traceID, gotTraceID)
	}

	// 验证各层级 spanID 都不同
	if level1SpanID == level2SpanID {
		t.Error("level1 和 level2 spanID 不应相等")
	}
	if level2SpanID == level3SpanID {
		t.Error("level2 和 level3 spanID 不应相等")
	}
	if level1SpanID == level3SpanID {
		t.Error("level1 和 level3 spanID 不应相等")
	}

	// 验证 ctx 中的 spanID 是最新的
	if currentSpanID := MustGetSpanID(ctx); currentSpanID != level3SpanID {
		t.Errorf("期望当前 spanID 为 %s, 实际为 %s", level3SpanID, currentSpanID)
	}
}

func TestGetParentSpanID_NoParent(t *testing.T) {
	ctx := context.Background()

	// 在没有 StartSpan 的情况下，parentSpanID 应该为空
	if parentSpanID := GetParentSpanID(ctx); parentSpanID != "" {
		t.Errorf("期望 parentSpanID 为空, 实际为 %s", parentSpanID)
	}
}

func TestMustGetSpanID_Empty(t *testing.T) {
	ctx := context.Background()

	// 在没有设置 spanID 的情况下，MustGetSpanID 应该返回空字符串
	if spanID := MustGetSpanID(ctx); spanID != "" {
		t.Errorf("期望 spanID 为空, 实际为 %s", spanID)
	}
}

func TestGetTraceID_GeneratesNewIfMissing(t *testing.T) {
	ctx := context.Background()

	// GetTraceID 在没有设置时会生成新的（但不会存储回 ctx）
	traceID := GetTraceID(ctx)

	if traceID == "" {
		t.Error("期望 GetTraceID 返回非空值")
	}

	// 用 WithTraceID 存储后，再获取应该得到相同的值
	ctx = WithTraceID(ctx, traceID)
	if gotTraceID := GetTraceID(ctx); gotTraceID != traceID {
		t.Errorf("期望 GetTraceID 返回存储的值 %s, 实际为 %s", traceID, gotTraceID)
	}
}

func TestWithTraceID_Override(t *testing.T) {
	ctx := context.Background()

	ctx, originalTraceID, _ := StartTrace(ctx)

	// 用新的 traceID 覆盖
	ctx = WithTraceID(ctx, "new-trace-id")

	newTraceID := GetTraceID(ctx)
	if newTraceID != "new-trace-id" {
		t.Errorf("期望 traceID 为 new-trace-id, 实际为 %s", newTraceID)
	}

	// 原始的 traceID 应该被覆盖
	if newTraceID == originalTraceID {
		t.Error("traceID 应该被新值覆盖")
	}
}
