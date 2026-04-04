/**
 * TodoList 单元测试
 * 测试 TodoList 的核心功能、边界条件和并发安全性
 */
package application

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewTodoList 测试构造函数正确初始化所有字段
func TestNewTodoList(t *testing.T) {
	tests := []struct {
		name   string
		taskID string
	}{
		{"基本初始化", "task-1"},
		{"空任务ID", ""},
		{"复杂任务ID", "task-abc-123_xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			todoList := NewTodoList(tt.taskID)

			// 验证 TaskID
			if todoList.TaskID != tt.taskID {
				t.Errorf("期望 TaskID 为 '%s', 实际为 '%s'", tt.taskID, todoList.TaskID)
			}

			// 验证 Items 自动创建
			if todoList.Items == nil {
				t.Error("Items 应该自动创建，不应为 nil")
			}
			if len(todoList.Items) != 0 {
				t.Errorf("Items 初始长度应为 0, 实际为 %d", len(todoList.Items))
			}

			// 验证时间戳
			now := time.Now().UnixMilli()
			if todoList.CreatedAt == 0 {
				t.Error("CreatedAt 应该被设置")
			}
			if todoList.UpdatedAt == 0 {
				t.Error("UpdatedAt 应该被设置")
			}
			if todoList.CreatedAt != todoList.UpdatedAt {
				t.Error("创建时 CreatedAt 和 UpdatedAt 应该相等")
			}
			// 验证时间戳在合理范围内（创建前后1秒内）
			if todoList.CreatedAt > now || todoList.CreatedAt < now-1000 {
				t.Errorf("CreatedAt 应该在合理范围内，实际为 %d, 当前时间为 %d", todoList.CreatedAt, now)
			}
		})
	}
}

// TestTodoList_AddItem 测试添加子任务
func TestTodoList_AddItem(t *testing.T) {
	todoList := NewTodoList("task-1")

	tests := []struct {
		name       string
		subTaskID  string
		goal       string
		taskType   string
		spanID     string
		status     TodoStatus
	}{
		{
			name:       "添加第一个子任务",
			subTaskID:  "sub-1",
			goal:       "子任务目标1",
			taskType:   "thinking",
			spanID:     "span-1",
			status:     TodoStatusDistributed,
		},
		{
			name:       "添加第二个子任务",
			subTaskID:  "sub-2",
			goal:       "子任务目标2",
			taskType:   "coding",
			spanID:     "span-2",
			status:     TodoStatusRunning,
		},
		{
			name:       "添加空值子任务",
			subTaskID:  "",
			goal:       "",
			taskType:   "",
			spanID:     "",
			status:     TodoStatusDistributed,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeAdd := time.Now().UnixMilli()
			time.Sleep(10 * time.Millisecond) // 确保时间戳有变化

			todoList.AddItem(tt.subTaskID, tt.goal, tt.taskType, tt.spanID, tt.status)

			// 验证子任务数量
			if len(todoList.Items) != i+1 {
				t.Errorf("添加后 Items 长度应为 %d, 实际为 %d", i+1, len(todoList.Items))
			}

			// 验证子任务字段
			item := &todoList.Items[i]
			if item.SubTaskID != tt.subTaskID {
				t.Errorf("SubTaskID 应为 '%s', 实际为 '%s'", tt.subTaskID, item.SubTaskID)
			}
			if item.Goal != tt.goal {
				t.Errorf("Goal 应为 '%s', 实际为 '%s'", tt.goal, item.Goal)
			}
			if item.SubTaskType != tt.taskType {
				t.Errorf("SubTaskType 应为 '%s', 实际为 '%s'", tt.taskType, item.SubTaskType)
			}
			if item.SpanID != tt.spanID {
				t.Errorf("SpanID 应为 '%s', 实际为 '%s'", tt.spanID, item.SpanID)
			}
			if item.Status != tt.status {
				t.Errorf("Status 应为 '%s', 实际为 '%s'", tt.status, item.Status)
			}

			// 验证进度默认为0
			if item.Progress != 0 {
				t.Errorf("Progress 应为 0, 实际为 %d", item.Progress)
			}

			// 验证时间戳
			if item.CreatedAt == 0 {
				t.Error("CreatedAt 应该被设置")
			}
			if item.CompletedAt != nil {
				t.Error("新添加的子任务 CompletedAt 应为 nil")
			}

			// 验证 UpdatedAt 被更新
			if todoList.UpdatedAt <= beforeAdd {
				t.Error("AddItem 应该更新 UpdatedAt")
			}
		})
	}
}

// TestTodoList_GetItem 测试获取指定子任务
func TestTodoList_GetItem(t *testing.T) {
	todoList := NewTodoList("task-1")
	todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)
	todoList.AddItem("sub-2", "目标2", "coding", "span-2", TodoStatusRunning)
	todoList.AddItem("sub-3", "目标3", "testing", "span-3", TodoStatusCompleted)

	tests := []struct {
		name          string
		subTaskID     string
		expectFound   bool
		expectedGoal  string
		expectedStatus TodoStatus
	}{
		{"获取存在的子任务1", "sub-1", true, "目标1", TodoStatusDistributed},
		{"获取存在的子任务2", "sub-2", true, "目标2", TodoStatusRunning},
		{"获取存在的子任务3", "sub-3", true, "目标3", TodoStatusCompleted},
		{"获取不存在的子任务", "sub-999", false, "", ""},
		{"获取空ID子任务", "", false, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := todoList.GetItem(tt.subTaskID)

			if tt.expectFound {
				if item == nil {
					t.Fatalf("应该找到 subTaskID 为 '%s' 的子任务", tt.subTaskID)
				}
				if item.SubTaskID != tt.subTaskID {
					t.Errorf("SubTaskID 应为 '%s', 实际为 '%s'", tt.subTaskID, item.SubTaskID)
				}
				if item.Goal != tt.expectedGoal {
					t.Errorf("Goal 应为 '%s', 实际为 '%s'", tt.expectedGoal, item.Goal)
				}
				if item.Status != tt.expectedStatus {
					t.Errorf("Status 应为 '%s', 实际为 '%s'", tt.expectedStatus, item.Status)
				}
			} else {
				if item != nil {
					t.Errorf("不应该找到 subTaskID 为 '%s' 的子任务", tt.subTaskID)
				}
			}
		})
	}
}

// TestTodoList_GetAllItems 测试获取所有子任务
func TestTodoList_GetAllItems(t *testing.T) {
	t.Run("空列表返回副本", func(t *testing.T) {
		todoList := NewTodoList("task-1")
		items := todoList.GetAllItems()

		if items == nil {
			t.Error("空列表应该返回空切片而非 nil")
		}
		if len(items) != 0 {
			t.Errorf("空列表返回的切片长度应为 0, 实际为 %d", len(items))
		}
	})

	t.Run("返回副本而非原始切片", func(t *testing.T) {
		todoList := NewTodoList("task-1")
		todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)
		todoList.AddItem("sub-2", "目标2", "coding", "span-2", TodoStatusRunning)

		items := todoList.GetAllItems()

		// 验证返回的是副本
		if len(items) != 2 {
			t.Errorf("返回的切片长度应为 2, 实际为 %d", len(items))
		}

		// 修改返回的副本，验证不影响原始数据
		items[0].Goal = "被修改的目标"
		items = append(items, TodoItem{SubTaskID: "sub-3", Goal: "新增项"})

		// 验证原始数据未被修改
		original := todoList.GetItem("sub-1")
		if original == nil || original.Goal != "目标1" {
			t.Error("修改返回的副本不应该影响原始数据")
		}

		// 验证原始列表长度未变
		if todoList.TotalCount() != 2 {
			t.Errorf("原始列表长度应为 2, 实际为 %d", todoList.TotalCount())
		}
	})

	t.Run("多次调用返回独立副本", func(t *testing.T) {
		todoList := NewTodoList("task-1")
		todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)

		items1 := todoList.GetAllItems()
		items2 := todoList.GetAllItems()

		// 修改第一个副本
		items1[0].Goal = "修改后"

		// 验证第二个副本未受影响
		if items2[0].Goal != "目标1" {
			t.Error("不同调用返回的副本应该是独立的")
		}
	})
}

// TestTodoList_UpdateProgress 测试更新进度和状态流转
func TestTodoList_UpdateProgress(t *testing.T) {
	t.Run("更新进度触发状态转换 distributed -> running", func(t *testing.T) {
		todoList := NewTodoList("task-1")
		todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)

		// 初始状态
		item := todoList.GetItem("sub-1")
		if item.Status != TodoStatusDistributed {
			t.Errorf("初始状态应为 'distributed', 实际为 '%s'", item.Status)
		}

		// 更新进度为 1，应触发状态转换
		todoList.UpdateProgress("sub-1", 1)
		item = todoList.GetItem("sub-1")
		if item.Status != TodoStatusRunning {
			t.Errorf("进度>0后状态应为 'running', 实际为 '%s'", item.Status)
		}
		if item.Progress != 1 {
			t.Errorf("Progress 应为 1, 实际为 %d", item.Progress)
		}
	})

	t.Run("更新进度不触发状态转换", func(t *testing.T) {
		todoList := NewTodoList("task-1")
		todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusRunning)

		todoList.UpdateProgress("sub-1", 50)
		item := todoList.GetItem("sub-1")
		if item.Status != TodoStatusRunning {
			t.Errorf("状态应保持 'running', 实际为 '%s'", item.Status)
		}
		if item.Progress != 50 {
			t.Errorf("Progress 应为 50, 实际为 %d", item.Progress)
		}
	})

	t.Run("更新不存在的子任务", func(t *testing.T) {
		todoList := NewTodoList("task-1")
		todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)

		beforeUpdate := todoList.UpdatedAt
		time.Sleep(10 * time.Millisecond)

		// 更新不存在的子任务，不应 panic
		todoList.UpdateProgress("sub-999", 50)

		// 验证现有子任务未受影响
		item := todoList.GetItem("sub-1")
		if item.Progress != 0 {
			t.Errorf("现有子任务进度不应被修改，实际为 %d", item.Progress)
		}

		// UpdatedAt 仍会被更新
		if todoList.UpdatedAt <= beforeUpdate {
			t.Error("UpdateProgress 应该更新 UpdatedAt")
		}
	})

	t.Run("各种进度值", func(t *testing.T) {
		todoList := NewTodoList("task-1")
		todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)

		progressValues := []int{0, 1, 50, 99, 100, -10, 150}
		for _, progress := range progressValues {
			todoList.UpdateProgress("sub-1", progress)
			item := todoList.GetItem("sub-1")
			if item.Progress != progress {
				t.Errorf("Progress 应为 %d, 实际为 %d", progress, item.Progress)
			}
		}
	})
}

// TestTodoList_MarkCompleted 测试标记完成
func TestTodoList_MarkCompleted(t *testing.T) {
	t.Run("标记完成设置状态和进度", func(t *testing.T) {
		todoList := NewTodoList("task-1")
		todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusRunning)

		todoList.MarkCompleted("sub-1")

		item := todoList.GetItem("sub-1")
		if item.Status != TodoStatusCompleted {
			t.Errorf("状态应为 'completed', 实际为 '%s'", item.Status)
		}
		if item.Progress != 100 {
			t.Errorf("进度应为 100, 实际为 %d", item.Progress)
		}
		if item.CompletedAt == nil {
			t.Error("CompletedAt 不应为 nil")
		}
	})

	t.Run("标记不存在的子任务完成", func(t *testing.T) {
		todoList := NewTodoList("task-1")
		todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)

		// 不应 panic
		todoList.MarkCompleted("sub-999")

		// 验证现有子任务未受影响
		item := todoList.GetItem("sub-1")
		if item.Status != TodoStatusDistributed {
			t.Errorf("现有子任务状态不应被修改")
		}
	})
}

// TestTodoList_MarkFailed 测试标记失败
func TestTodoList_MarkFailed(t *testing.T) {
	t.Run("标记失败设置状态", func(t *testing.T) {
		todoList := NewTodoList("task-1")
		todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusRunning)

		todoList.MarkFailed("sub-1")

		item := todoList.GetItem("sub-1")
		if item.Status != TodoStatusFailed {
			t.Errorf("状态应为 'failed', 实际为 '%s'", item.Status)
		}
		if item.CompletedAt == nil {
			t.Error("CompletedAt 不应为 nil")
		}
	})

	t.Run("标记不存在的子任务失败", func(t *testing.T) {
		todoList := NewTodoList("task-1")
		todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)

		// 不应 panic
		todoList.MarkFailed("sub-999")

		item := todoList.GetItem("sub-1")
		if item.Status != TodoStatusDistributed {
			t.Errorf("现有子任务状态不应被修改")
		}
	})
}

// TestTodoList_MarkCancelled 测试标记取消
func TestTodoList_MarkCancelled(t *testing.T) {
	t.Run("标记取消设置状态", func(t *testing.T) {
		todoList := NewTodoList("task-1")
		todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)

		todoList.MarkCancelled("sub-1")

		item := todoList.GetItem("sub-1")
		if item.Status != TodoStatusCancelled {
			t.Errorf("状态应为 'cancelled', 实际为 '%s'", item.Status)
		}
		if item.CompletedAt == nil {
			t.Error("CompletedAt 不应为 nil")
		}
	})

	t.Run("标记不存在的子任务取消", func(t *testing.T) {
		todoList := NewTodoList("task-1")
		todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)

		// 不应 panic
		todoList.MarkCancelled("sub-999")

		item := todoList.GetItem("sub-1")
		if item.Status != TodoStatusDistributed {
			t.Errorf("现有子任务状态不应被修改")
		}
	})
}

// TestTodoList_AllCompleted 测试全部完成检查
func TestTodoList_AllCompleted(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(tl *TodoList)
		expectedResult bool
	}{
		{
			name:           "空列表返回 false",
			setupFunc:      func(tl *TodoList) {},
			expectedResult: false,
		},
		{
			name: "单个未完成返回 false",
			setupFunc: func(tl *TodoList) {
				tl.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)
			},
			expectedResult: false,
		},
		{
			name: "单个已完成返回 true",
			setupFunc: func(tl *TodoList) {
				tl.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)
				tl.MarkCompleted("sub-1")
			},
			expectedResult: true,
		},
		{
			name: "多个全部完成返回 true",
			setupFunc: func(tl *TodoList) {
				tl.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)
				tl.AddItem("sub-2", "目标2", "coding", "span-2", TodoStatusDistributed)
				tl.MarkCompleted("sub-1")
				tl.MarkCompleted("sub-2")
			},
			expectedResult: true,
		},
		{
			name: "多个部分完成返回 false",
			setupFunc: func(tl *TodoList) {
				tl.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)
				tl.AddItem("sub-2", "目标2", "coding", "span-2", TodoStatusDistributed)
				tl.MarkCompleted("sub-1")
				// sub-2 未完成
			},
			expectedResult: false,
		},
		{
			name: "包含失败返回 false",
			setupFunc: func(tl *TodoList) {
				tl.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)
				tl.AddItem("sub-2", "目标2", "coding", "span-2", TodoStatusDistributed)
				tl.MarkCompleted("sub-1")
				tl.MarkFailed("sub-2")
			},
			expectedResult: false,
		},
		{
			name: "包含取消返回 false",
			setupFunc: func(tl *TodoList) {
				tl.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)
				tl.AddItem("sub-2", "目标2", "coding", "span-2", TodoStatusDistributed)
				tl.MarkCompleted("sub-1")
				tl.MarkCancelled("sub-2")
			},
			expectedResult: false,
		},
		{
			name: "running 状态返回 false",
			setupFunc: func(tl *TodoList) {
				tl.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusRunning)
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			todoList := NewTodoList("task-1")
			tt.setupFunc(todoList)

			result := todoList.AllCompleted()
			if result != tt.expectedResult {
				t.Errorf("AllCompleted() 应返回 %v, 实际为 %v", tt.expectedResult, result)
			}
		})
	}
}

// TestTodoList_CompletedCount 测试已完成数量统计
func TestTodoList_CompletedCount(t *testing.T) {
	todoList := NewTodoList("task-1")

	// 空列表
	if todoList.CompletedCount() != 0 {
		t.Errorf("空列表已完成数量应为 0, 实际为 %d", todoList.CompletedCount())
	}

	// 添加子任务
	todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)
	todoList.AddItem("sub-2", "目标2", "coding", "span-2", TodoStatusRunning)
	todoList.AddItem("sub-3", "目标3", "testing", "span-3", TodoStatusDistributed)
	todoList.AddItem("sub-4", "目标4", "deploy", "span-4", TodoStatusDistributed)

	// 标记部分完成
	todoList.MarkCompleted("sub-1")
	todoList.MarkCompleted("sub-3")

	// 验证统计
	if todoList.CompletedCount() != 2 {
		t.Errorf("已完成数量应为 2, 实际为 %d", todoList.CompletedCount())
	}

	// 标记更多完成
	todoList.MarkCompleted("sub-2")

	if todoList.CompletedCount() != 3 {
		t.Errorf("已完成数量应为 3, 实际为 %d", todoList.CompletedCount())
	}

	// 失败的不计入已完成
	todoList.MarkFailed("sub-4")
	if todoList.CompletedCount() != 3 {
		t.Errorf("失败不计入已完成，数量应为 3, 实际为 %d", todoList.CompletedCount())
	}
}

// TestTodoList_TotalCount 测试总数量
func TestTodoList_TotalCount(t *testing.T) {
	todoList := NewTodoList("task-1")

	// 空列表
	if todoList.TotalCount() != 0 {
		t.Errorf("空列表总数量应为 0, 实际为 %d", todoList.TotalCount())
	}

	// 添加子任务
	todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)
	if todoList.TotalCount() != 1 {
		t.Errorf("总数量应为 1, 实际为 %d", todoList.TotalCount())
	}

	todoList.AddItem("sub-2", "目标2", "coding", "span-2", TodoStatusDistributed)
	if todoList.TotalCount() != 2 {
		t.Errorf("总数量应为 2, 实际为 %d", todoList.TotalCount())
	}

	todoList.AddItem("sub-3", "目标3", "testing", "span-3", TodoStatusDistributed)
	if todoList.TotalCount() != 3 {
		t.Errorf("总数量应为 3, 实际为 %d", todoList.TotalCount())
	}
}

// TestTodoList_ToJSON 测试 JSON 序列化
func TestTodoList_ToJSON(t *testing.T) {
	t.Run("序列化空列表", func(t *testing.T) {
		todoList := NewTodoList("task-1")
		jsonStr := todoList.ToJSON()

		// 验证是有效的 JSON
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			t.Errorf("ToJSON 返回的不是有效 JSON: %v", err)
		}

		// 验证包含 TaskID
		if result["task_id"] != "task-1" {
			t.Errorf("JSON 应包含 task_id='task-1', 实际为 %v", result["task_id"])
		}

		// 验证包含空 items 数组
		items, ok := result["items"].([]interface{})
		if !ok || len(items) != 0 {
			t.Error("JSON 应包含空的 items 数组")
		}
	})

	t.Run("序列化包含子任务的列表", func(t *testing.T) {
		todoList := NewTodoList("task-1")
		todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)
		todoList.UpdateProgress("sub-1", 50)

		jsonStr := todoList.ToJSON()

		// 验证是有效的 JSON
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			t.Errorf("ToJSON 返回的不是有效 JSON: %v", err)
		}

		// 验证基本字段
		if result["task_id"] != "task-1" {
			t.Errorf("JSON 应包含 task_id='task-1'")
		}

		// 验证 items
		items, ok := result["items"].([]interface{})
		if !ok || len(items) != 1 {
			t.Fatalf("JSON 应包含 1 个 item")
		}

		item := items[0].(map[string]interface{})
		if item["sub_task_id"] != "sub-1" {
			t.Errorf("item 应包含 sub_task_id='sub-1'")
		}
		if item["goal"] != "目标1" {
			t.Errorf("item 应包含 goal='目标1'")
		}
		if item["status"] != "running" { // 进度>0后状态变为 running
			t.Errorf("item 应包含 status='running'")
		}
		if progress, ok := item["progress"].(float64); !ok || progress != 50 {
			t.Errorf("item 应包含 progress=50, 实际为 %v", item["progress"])
		}
	})

	t.Run("序列化包含完成时间的列表", func(t *testing.T) {
		todoList := NewTodoList("task-1")
		todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)
		todoList.MarkCompleted("sub-1")

		jsonStr := todoList.ToJSON()

		// 验证 completed_at 存在
		if !strings.Contains(jsonStr, "completed_at") {
			t.Error("JSON 应包含 completed_at 字段")
		}
	})

	t.Run("验证 mu 字段不被序列化", func(t *testing.T) {
		todoList := NewTodoList("task-1")
		jsonStr := todoList.ToJSON()

		// 验证 JSON 中不包含 mu 字段
		if strings.Contains(jsonStr, "mu") {
			t.Error("JSON 不应包含 mu 字段")
		}
	})
}

// TestTodoList_Concurrent_AddItem 测试并发添加子任务
func TestTodoList_Concurrent_AddItem(t *testing.T) {
	todoList := NewTodoList("task-1")

	const numGoroutines = 100
	const itemsPerGoroutine = 50

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				subTaskID := "sub-" + strconv.Itoa(index) + "-" + strconv.Itoa(j)
				todoList.AddItem(subTaskID, "目标", "thinking", "span", TodoStatusDistributed)
			}
		}(i)
	}
	wg.Wait()

	expectedCount := numGoroutines * itemsPerGoroutine
	if todoList.TotalCount() != expectedCount {
		t.Errorf("期望有 %d 个子任务, 实际有 %d 个", expectedCount, todoList.TotalCount())
	}
}

// TestTodoList_Concurrent_MixedOperations 测试并发混合操作
func TestTodoList_Concurrent_MixedOperations(t *testing.T) {
	todoList := NewTodoList("task-1")

	// 先添加一些初始子任务
	for i := 0; i < 20; i++ {
		todoList.AddItem("sub-"+strconv.Itoa(i), "目标", "thinking", "span-"+strconv.Itoa(i), TodoStatusDistributed)
	}

	const numGoroutines = 50
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	var addCounter int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				switch j % 10 {
				case 0, 1:
					// 添加新子任务
					subTaskID := "new-sub-" + strconv.Itoa(index) + "-" + strconv.Itoa(j)
					todoList.AddItem(subTaskID, "新目标", "coding", "span-new", TodoStatusDistributed)
					atomic.AddInt64(&addCounter, 1)
				case 2, 3:
					// 更新进度
					subTaskID := "sub-" + strconv.Itoa(j%20)
					todoList.UpdateProgress(subTaskID, j%100)
				case 4:
					// 标记完成
					subTaskID := "sub-" + strconv.Itoa(j%20)
					todoList.MarkCompleted(subTaskID)
				case 5:
					// 标记失败
					subTaskID := "sub-" + strconv.Itoa(j%20)
					todoList.MarkFailed(subTaskID)
				case 6:
					// 标记取消
					subTaskID := "sub-" + strconv.Itoa(j%20)
					todoList.MarkCancelled(subTaskID)
				case 7:
					// 获取子任务
					subTaskID := "sub-" + strconv.Itoa(j%20)
					_ = todoList.GetItem(subTaskID)
				case 8:
					// 获取所有子任务
					_ = todoList.GetAllItems()
				case 9:
					// 统计操作
					_ = todoList.AllCompleted()
					_ = todoList.CompletedCount()
					_ = todoList.TotalCount()
				}
			}
		}(i)
	}
	wg.Wait()

	// 验证子任务数量
	expectedCount := 20 + int(addCounter)
	if todoList.TotalCount() != expectedCount {
		t.Errorf("期望有 %d 个子任务, 实际有 %d 个", expectedCount, todoList.TotalCount())
	}

}

// TestTodoList_Concurrent_ReadWrite 测试并发读写
func TestTodoList_Concurrent_ReadWrite(t *testing.T) {
	todoList := NewTodoList("task-1")

	const numGoroutines = 50

	var wg sync.WaitGroup

	// 写操作 goroutines
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				subTaskID := "sub-" + strconv.Itoa(index) + "-" + strconv.Itoa(j)
				todoList.AddItem(subTaskID, "目标", "thinking", "span", TodoStatusDistributed)
				todoList.UpdateProgress(subTaskID, j*2)
				if j%3 == 0 {
					todoList.MarkCompleted(subTaskID)
				}
			}
		}(i)
	}

	// 读操作 goroutines
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				subTaskID := "sub-" + strconv.Itoa(j%10)
				_ = todoList.GetItem(subTaskID)
				_ = todoList.GetAllItems()
				_ = todoList.AllCompleted()
				_ = todoList.CompletedCount()
				_ = todoList.TotalCount()
				_ = todoList.ToJSON()
			}
		}(i)
	}

	wg.Wait()

}

// TestTodoList_EdgeCases 测试边界条件
func TestTodoList_EdgeCases(t *testing.T) {
	t.Run("对空列表的各种操作", func(t *testing.T) {
		todoList := NewTodoList("task-1")

		// 获取不存在的子任务
		item := todoList.GetItem("non-existent")
		if item != nil {
			t.Error("空列表获取子任务应返回 nil")
		}

		// 更新不存在的子任务
		todoList.UpdateProgress("non-existent", 50) // 不应 panic

		// 标记不存在的子任务完成
		todoList.MarkCompleted("non-existent") // 不应 panic

		// 标记不存在的子任务失败
		todoList.MarkFailed("non-existent") // 不应 panic

		// 标记不存在的子任务取消
		todoList.MarkCancelled("non-existent") // 不应 panic

		// 统计检查
		if todoList.AllCompleted() {
			t.Error("空列表 AllCompleted 应返回 false")
		}
		if todoList.CompletedCount() != 0 {
			t.Error("空列表 CompletedCount 应为 0")
		}
		if todoList.TotalCount() != 0 {
			t.Error("空列表 TotalCount 应为 0")
		}

		// 序列化
		jsonStr := todoList.ToJSON()
		if jsonStr == "" {
			t.Error("空列表 ToJSON 不应返回空字符串")
		}
	})

	t.Run("重复添加相同ID的子任务", func(t *testing.T) {
		todoList := NewTodoList("task-1")

		// 添加相同ID的子任务两次
		todoList.AddItem("sub-1", "目标1", "thinking", "span-1", TodoStatusDistributed)
		todoList.AddItem("sub-1", "目标2", "coding", "span-2", TodoStatusRunning)

		// 应该有两个子任务（允许重复ID）
		if todoList.TotalCount() != 2 {
			t.Errorf("允许重复ID时总数量应为 2, 实际为 %d", todoList.TotalCount())
		}

		// GetItem 返回第一个匹配的
		item := todoList.GetItem("sub-1")
		if item == nil || item.Goal != "目标1" {
			t.Error("GetItem 应返回第一个匹配的子任务")
		}

		// UpdateProgress 更新第一个匹配的
		todoList.UpdateProgress("sub-1", 50)
		items := todoList.GetAllItems()
		if items[0].Progress != 50 {
			t.Error("UpdateProgress 应更新第一个匹配的子任务")
		}
		if items[1].Progress != 0 {
			t.Error("第二个子任务进度不应被修改")
		}
	})

	t.Run("特殊字符子任务ID", func(t *testing.T) {
		todoList := NewTodoList("task-1")

		specialIDs := []string{
			"sub-with-dash",
			"sub_with_underscore",
			"sub.with.dot",
			"sub:with:colon",
			"sub/with/slash",
			"sub with space",
			"sub\twith\ttab",
			"sub\nwith\nnewline",
			"中文子任务ID",
			"🔥emoji",
		}

		for i, id := range specialIDs {
			todoList.AddItem(id, "目标"+strconv.Itoa(i), "thinking", "span", TodoStatusDistributed)
		}

		if todoList.TotalCount() != len(specialIDs) {
			t.Errorf("总数量应为 %d, 实际为 %d", len(specialIDs), todoList.TotalCount())
		}

		// 验证可以获取所有特殊ID
		for _, id := range specialIDs {
			item := todoList.GetItem(id)
			if item == nil {
				t.Errorf("应该能找到 subTaskID 为 '%s' 的子任务", id)
			}
		}
	})
}
