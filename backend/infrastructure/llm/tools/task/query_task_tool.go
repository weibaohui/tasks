/**
 * QueryTaskTool - 查询任务状态的 LLM 工具
 * 允许 Agent 查询已创建任务的状态和结果
 */
package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/llm"
)

// QueryTaskTool 任务查询工具
type QueryTaskTool struct {
	taskService *application.TaskApplicationService
}

// NewQueryTaskTool 创建任务查询工具
func NewQueryTaskTool(taskService *application.TaskApplicationService) *QueryTaskTool {
	return &QueryTaskTool{
		taskService: taskService,
	}
}

var _ llm.Tool = (*QueryTaskTool)(nil)

// Name 返回工具名称
func (t *QueryTaskTool) Name() string {
	return "query_task"
}

// Description 返回工具描述
func (t *QueryTaskTool) Description() string {
	return `查询任务状态和结果。
参数 task_id: 任务 ID（必填）

返回任务的详细信息包括：状态、进度、子任务列表、执行结果或错误信息。

示例：query_task(task_id="xxx")`
}

// Parameters 返回参数 schema
func (t *QueryTaskTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"task_id": {
				"type": "string",
				"description": "任务 ID（必填）"
			}
		},
		"required": ["task_id"]
	}`)
}

// Execute 执行工具
func (t *QueryTaskTool) Execute(ctx context.Context, input json.RawMessage) (*llm.ToolResult, error) {
	var args struct {
		TaskID string `json:"task_id"`
	}

	if err := json.Unmarshal(input, &args); err != nil {
		return &llm.ToolResult{
			Output: fmt.Sprintf(`{"success": false, "error": "解析参数失败: %v"}`, err),
			Error:  "",
		}, nil
	}

	// 验证必填参数
	if args.TaskID == "" {
		return &llm.ToolResult{
			Output: `{"success": false, "error": "task_id 不能为空"}`,
			Error:  "",
		}, nil
	}

	// 获取任务
	taskID := domain.NewTaskID(args.TaskID)
	task, err := t.taskService.GetTask(ctx, taskID)
	if err != nil {
		return &llm.ToolResult{
			Output: fmt.Sprintf(`{"success": false, "error": "获取任务失败: %v"}`, err),
			Error:  "",
		}, nil
	}

	if task == nil {
		return &llm.ToolResult{
			Output: fmt.Sprintf(`{"success": false, "task_id": "%s", "error": "任务不存在"}`, args.TaskID),
			Error:  "",
		}, nil
	}

	// 构建任务信息
	taskInfo := map[string]interface{}{
		"success":    true,
		"task_id":    task.ID().String(),
		"name":       task.Name(),
		"status":     task.Status().String(),
		"type":       task.Type().String(),
		"progress":   task.Progress().Percentage(),
		"stage":      task.Progress().Stage(),
		"detail":     task.Progress().Detail(),
		"created_at": task.CreatedAt().Format(time.RFC3339),
	}

	// 添加开始和结束时间
	if start := task.StartedAt(); start != nil {
		taskInfo["started_at"] = start.Format(time.RFC3339)
	}
	if end := task.FinishedAt(); end != nil {
		taskInfo["finished_at"] = end.Format(time.RFC3339)
	}

	// 添加结果或错误
	if task.Status() == domain.TaskStatusCompleted {
		if res := task.Result(); res != nil {
			// 安全地序列化 data，避免指针等无法 JSON 序列化的问题
			var dataJSON interface{}
			if res.Data() != nil {
				if dataBytes, err := json.Marshal(res.Data()); err == nil {
					json.Unmarshal(dataBytes, &dataJSON)
				} else {
					dataJSON = fmt.Sprintf("%v", res.Data())
				}
			}
			taskInfo["result"] = map[string]interface{}{
				"message": res.Message(),
				"data":    dataJSON,
			}
		}
	} else if task.Status() == domain.TaskStatusFailed {
		if err := task.Error(); err != nil {
			taskInfo["error"] = err.Error()
		}
	}

	// 获取子任务
	children, err := t.taskService.GetChildTasks(ctx, taskID)
	if err == nil && len(children) > 0 {
		subTasks := make([]map[string]interface{}, 0, len(children))
		for _, child := range children {
			subTasks = append(subTasks, map[string]interface{}{
				"task_id": child.ID().String(),
				"name":    child.Name(),
				"status":  child.Status().String(),
				"progress": child.Progress().Percentage(),
				"stage":   child.Progress().Stage(),
			})
		}
		taskInfo["sub_tasks"] = subTasks
		taskInfo["sub_tasks_count"] = len(subTasks)
	}

	// 添加执行摘要（使用独立字段）
	if summary := task.ExecutionSummary(); summary != nil {
		taskInfo["execution_summary"] = summary
	}
	// 添加 todo_list（使用独立字段）
	if todoList := task.TodoList(); todoList != "" {
		taskInfo["todo_list"] = todoList
	}
	// 添加 Agent 模式的分析结果（使用独立字段）
	if analysis := task.Analysis(); analysis != "" {
		taskInfo["analysis"] = analysis
	}

	// 添加 task_conclusion（任务结论）
	if taskConclusion := task.TaskConclusion(); taskConclusion != "" {
		taskInfo["task_conclusion"] = taskConclusion
	}

	resultJSON, _ := json.Marshal(taskInfo)
	return &llm.ToolResult{
		Output: string(resultJSON),
		Error:  "",
	}, nil
}
