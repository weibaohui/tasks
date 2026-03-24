/**
 * TodoList - 子任务列表
 * 父任务维护的子任务追踪列表
 */
package application

import (
	"encoding/json"
	"sync"
	"time"
)

type TodoStatus string

const (
	TodoStatusDistributed TodoStatus = "distributed" // 已分发
	TodoStatusRunning     TodoStatus = "running"     // 执行中
	TodoStatusCompleted   TodoStatus = "completed"   // 已完成
	TodoStatusFailed      TodoStatus = "failed"      // 失败
	TodoStatusCancelled   TodoStatus = "cancelled"   // 被取消
)

type TodoItem struct {
	SubTaskID   string     `json:"sub_task_id"`
	SubTaskType string     `json:"sub_task_type"`
	Goal        string     `json:"goal"`
	Status      TodoStatus `json:"status"`
	Progress    int        `json:"progress"` // 0-100
	SpanID      string     `json:"span_id"`
	CreatedAt   int64      `json:"created_at"`
	CompletedAt *int64     `json:"completed_at,omitempty"`
}

type TodoList struct {
	TaskID    string     `json:"task_id"`
	Items     []TodoItem `json:"items"`
	CreatedAt int64      `json:"created_at"`
	UpdatedAt int64      `json:"updated_at"`
	mu        sync.RWMutex
}

func NewTodoList(taskID string) *TodoList {
	now := time.Now().UnixMilli()
	return &TodoList{
		TaskID:    taskID,
		Items:     make([]TodoItem, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (tl *TodoList) AddItem(subTaskID, goal, taskType, spanID string, status TodoStatus) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.Items = append(tl.Items, TodoItem{
		SubTaskID:   subTaskID,
		SubTaskType: taskType,
		Goal:        goal,
		Status:      status,
		Progress:    0,
		SpanID:      spanID,
		CreatedAt:   time.Now().UnixMilli(),
	})
	tl.UpdatedAt = time.Now().UnixMilli()
}

func (tl *TodoList) UpdateProgress(subTaskID string, progress int) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	for i := range tl.Items {
		if tl.Items[i].SubTaskID == subTaskID {
			tl.Items[i].Progress = progress
			if progress > 0 && tl.Items[i].Status == TodoStatusDistributed {
				tl.Items[i].Status = TodoStatusRunning
			}
			break
		}
	}
	tl.UpdatedAt = time.Now().UnixMilli()
}

func (tl *TodoList) MarkCompleted(subTaskID string) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	now := time.Now().UnixMilli()
	for i := range tl.Items {
		if tl.Items[i].SubTaskID == subTaskID {
			tl.Items[i].Status = TodoStatusCompleted
			tl.Items[i].Progress = 100
			tl.Items[i].CompletedAt = &now
			break
		}
	}
	tl.UpdatedAt = now
}

func (tl *TodoList) MarkFailed(subTaskID string) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	now := time.Now().UnixMilli()
	for i := range tl.Items {
		if tl.Items[i].SubTaskID == subTaskID {
			tl.Items[i].Status = TodoStatusFailed
			tl.Items[i].CompletedAt = &now
			break
		}
	}
	tl.UpdatedAt = now
}

func (tl *TodoList) MarkCancelled(subTaskID string) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	now := time.Now().UnixMilli()
	for i := range tl.Items {
		if tl.Items[i].SubTaskID == subTaskID {
			tl.Items[i].Status = TodoStatusCancelled
			tl.Items[i].CompletedAt = &now
			break
		}
	}
	tl.UpdatedAt = now
}

func (tl *TodoList) GetItem(subTaskID string) *TodoItem {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	for i := range tl.Items {
		if tl.Items[i].SubTaskID == subTaskID {
			return &tl.Items[i]
		}
	}
	return nil
}

func (tl *TodoList) GetAllItems() []TodoItem {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	result := make([]TodoItem, len(tl.Items))
	copy(result, tl.Items)
	return result
}

func (tl *TodoList) AllCompleted() bool {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	for i := range tl.Items {
		if tl.Items[i].Status != TodoStatusCompleted {
			return false
		}
	}
	return len(tl.Items) > 0
}

func (tl *TodoList) CompletedCount() int {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	count := 0
	for i := range tl.Items {
		if tl.Items[i].Status == TodoStatusCompleted {
			count++
		}
	}
	return count
}

func (tl *TodoList) TotalCount() int {
	tl.mu.RLock()
	defer tl.mu.RUnlock()
	return len(tl.Items)
}

func (tl *TodoList) ToJSON() string {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	data, _ := json.Marshal(tl)
	return string(data)
}
