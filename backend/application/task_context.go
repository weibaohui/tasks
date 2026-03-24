/**
 * TaskContext - 任务执行上下文
 * 管理单个任务的执行状态和回调
 */
package application

import (
	"context"
	"sync"

	"github.com/weibh/taskmanager/domain"
)

type SubTaskResult struct {
	SubTaskID    string
	SpanID       string
	ParentSpanID string
	TaskType     domain.TaskType
	Goal         string
}

type TaskContext struct {
	TaskID       string
	Goal         string
	TaskType     domain.TaskType
	SpanID       string
	ParentSpanID string

	TraceContext *TraceContext
	TodoList     *TodoList

	CancelFunc context.CancelFunc

	OnProgress func(progress int)
	OnComplete func()
	OnSubTask  func(goal string, taskType domain.TaskType) *SubTaskResult

	mu sync.RWMutex
}

func NewTaskContext(taskID string, taskType domain.TaskType, goal, spanID, parentSpanID string, tc *TraceContext) *TaskContext {
	return &TaskContext{
		TaskID:       taskID,
		TaskType:     taskType,
		Goal:         goal,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
		TraceContext: tc,
		TodoList:     NewTodoList(taskID),
	}
}

func (tc *TaskContext) SetCancelFunc(cancel context.CancelFunc) {
	tc.CancelFunc = cancel
}

func (tc *TaskContext) SetOnProgress(fn func(progress int)) {
	tc.OnProgress = fn
}

func (tc *TaskContext) SetOnComplete(fn func()) {
	tc.OnComplete = fn
}

func (tc *TaskContext) SetOnSubTask(fn func(goal string, taskType domain.TaskType) *SubTaskResult) {
	tc.OnSubTask = fn
}

func (tc *TaskContext) ReportProgress(progress int) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	if tc.OnProgress != nil {
		tc.OnProgress(progress)
	}
}

func (tc *TaskContext) Complete() {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	if tc.OnComplete != nil {
		tc.OnComplete()
	}
}

func (tc *TaskContext) SpawnSubTask(goal string, taskType domain.TaskType) *SubTaskResult {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	if tc.OnSubTask != nil {
		return tc.OnSubTask(goal, taskType)
	}
	return nil
}

func (tc *TaskContext) Cancel() {
	if tc.CancelFunc != nil {
		tc.CancelFunc()
	}
}

func (tc *TaskContext) UpdateTodoProgress(subTaskID string, progress int) {
	tc.TodoList.UpdateProgress(subTaskID, progress)
}

func (tc *TaskContext) UpdateTodoCompleted(subTaskID string) {
	tc.TodoList.MarkCompleted(subTaskID)
}

func (tc *TaskContext) UpdateTodoFailed(subTaskID string) {
	tc.TodoList.MarkFailed(subTaskID)
}

func (tc *TaskContext) AddTodoItem(subTaskID, goal, taskType, spanID string) {
	tc.TodoList.AddItem(subTaskID, goal, taskType, spanID, TodoStatusDistributed)
}

func (tc *TaskContext) AllSubTasksCompleted() bool {
	return tc.TodoList.AllCompleted()
}
