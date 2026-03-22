/**
 * TraceContext - 追踪上下文
 * 管理整个 Trace 的 Span 层级和级联取消
 */
package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/weibh/taskmanager/infrastructure/utils"
)

type Span struct {
	SpanID       string `json:"span_id"`
	ParentSpanID string `json:"parent_span_id,omitempty"`
	TaskID       string `json:"task_id"`
	Status       string `json:"status"`
	CreatedAt    int64  `json:"created_at"`
}

type TraceContext struct {
	TraceID    string `json:"trace_id"`
	RootTaskID string `json:"root_task_id"`
	mu         sync.RWMutex
	spans      map[string]*Span    `json:"spans"`     // spanID -> Span
	taskTree   map[string][]string `json:"task_tree"` // parentTaskID -> [childTaskIDs]
	cancelFunc context.CancelFunc  `json:"-"`
	rootCtx    context.Context     `json:"-"`
	logger     interface{ Info(string, ...interface{}) }
}

func NewTraceContext(traceID, rootTaskID string, logger interface{ Info(string, ...interface{}) }) *TraceContext {
	rootCtx, cancel := context.WithCancel(context.Background())
	return &TraceContext{
		TraceID:    traceID,
		RootTaskID: rootTaskID,
		spans:      make(map[string]*Span),
		taskTree:   make(map[string][]string),
		rootCtx:    rootCtx,
		cancelFunc: cancel,
		logger:     logger,
	}
}

func (tc *TraceContext) RootContext() context.Context {
	return tc.rootCtx
}

func (tc *TraceContext) GenerateSpanID(parentSpanID string) string {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	var spanID string
	if parentSpanID == "" {
		spanID = fmt.Sprintf("span-%s", utils.NewNanoIDGenerator(8).Generate())
	} else {
		spanID = fmt.Sprintf("%s-%s", parentSpanID, utils.NewNanoIDGenerator(4).Generate())
	}

	tc.spans[spanID] = &Span{
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
		TaskID:       "",
		Status:       "created",
		CreatedAt:    time.Now().UnixMilli(),
	}

	return spanID
}

func (tc *TraceContext) RegisterTask(taskID, parentTaskID, spanID, parentSpanID string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if span, ok := tc.spans[spanID]; ok {
		span.TaskID = taskID
		span.Status = "running"
	}

	if parentTaskID != "" {
		tc.taskTree[parentTaskID] = append(tc.taskTree[parentTaskID], taskID)
	}
}

func (tc *TraceContext) GetSpan(spanID string) *Span {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.spans[spanID]
}

func (tc *TraceContext) GetAllSpans() map[string]*Span {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	result := make(map[string]*Span)
	for k, v := range tc.spans {
		result[k] = v
	}
	return result
}

func (tc *TraceContext) GetTaskTree() map[string][]string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	result := make(map[string][]string)
	for k, v := range tc.taskTree {
		items := make([]string, len(v))
		copy(items, v)
		result[k] = items
	}
	return result
}

func (tc *TraceContext) GetChildTasks(parentTaskID string) []string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	children := tc.taskTree[parentTaskID]
	result := make([]string, len(children))
	copy(result, children)
	return result
}

func (tc *TraceContext) IsRootTask(taskID string) bool {
	return taskID == tc.RootTaskID
}

func (tc *TraceContext) GetTraceID() string {
	return tc.TraceID
}

func (tc *TraceContext) GetRootTaskID() string {
	return tc.RootTaskID
}

func (tc *TraceContext) CancelAll() {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	if tc.cancelFunc != nil {
		tc.cancelFunc()
	}
}

func (tc *TraceContext) CancelTask(taskID string) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	if tc.cancelFunc != nil && taskID == tc.RootTaskID {
		tc.cancelFunc()
	}
}

func (tc *TraceContext) MarkSpanCompleted(spanID string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if span, ok := tc.spans[spanID]; ok {
		span.Status = "completed"
	}
}

func (tc *TraceContext) MarkSpanFailed(spanID string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if span, ok := tc.spans[spanID]; ok {
		span.Status = "failed"
	}
}
