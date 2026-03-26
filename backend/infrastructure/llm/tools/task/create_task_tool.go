/**
 * CreateTaskTool - 创建任务的 LLM 工具
 * 允许 Agent 通过工具调用创建新任务
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

// CreateTaskTool 任务创建工具
type CreateTaskTool struct {
	taskService *application.TaskApplicationService
	idGenerator domain.IDGenerator
	// 上下文信息
	agentCode   string
	userCode    string
	channelCode string
	sessionKey  string
	traceID     string
	spanID      string
}

// NewCreateTaskTool 创建任务创建工具
func NewCreateTaskTool(
	taskService *application.TaskApplicationService,
	idGenerator domain.IDGenerator,
	agentCode, userCode, channelCode, sessionKey, traceID, spanID string,
) *CreateTaskTool {
	return &CreateTaskTool{
		taskService: taskService,
		idGenerator: idGenerator,
		agentCode:   agentCode,
		userCode:    userCode,
		channelCode: channelCode,
		sessionKey:  sessionKey,
		traceID:    traceID,
		spanID:     spanID,
	}
}

var _ llm.Tool = (*CreateTaskTool)(nil)

// Name 返回工具名称
func (t *CreateTaskTool) Name() string {
	return "create_task"
}

// Description 返回工具描述
func (t *CreateTaskTool) Description() string {
	return `创建一个新任务。
参数 name: 任务名称（必填）
参数 description: 任务描述（可选）
参数 task_type: 任务类型（可选），可选值: agent, coding, custom，默认 custom
参数 timeout_ms: 超时时间毫秒数（可选），默认 60000
参数 priority: 优先级（可选），默认 0
参数 parent_id: 父任务 ID（可选），用于创建子任务

示例：create_task(name="测试任务", description="执行测试", task_type="agent", timeout_ms=30000)`
}

// Parameters 返回参数 schema
func (t *CreateTaskTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "任务名称（必填）"
			},
			"description": {
				"type": "string",
				"description": "任务描述（可选）"
			},
			"task_type": {
				"type": "string",
				"description": "任务类型（可选），可选值: agent, coding, custom",
				"enum": ["agent", "coding", "custom"]
			},
			"timeout_ms": {
				"type": "integer",
				"description": "超时时间毫秒数（可选），默认 60000"
			},
			"priority": {
				"type": "integer",
				"description": "优先级（可选），默认 0"
			},
			"parent_id": {
				"type": "string",
				"description": "父任务 ID（可选），用于创建子任务"
			}
		},
		"required": ["name"]
	}`)
}

// Execute 执行工具
func (t *CreateTaskTool) Execute(ctx context.Context, input json.RawMessage) (*llm.ToolResult, error) {
	var args struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		TaskType    string  `json:"task_type"`
		TimeoutMs   int64   `json:"timeout_ms"`
		Priority    int     `json:"priority"`
		ParentID    string  `json:"parent_id"`
	}

	if err := json.Unmarshal(input, &args); err != nil {
		return &llm.ToolResult{
			Output: fmt.Sprintf(`{"success": false, "error": "解析参数失败: %v"}`, err),
			Error:  "",
		}, nil
	}

	// 验证必填参数
	if args.Name == "" {
		return &llm.ToolResult{
			Output: `{"success": false, "error": "name 不能为空"}`,
			Error:  "",
		}, nil
	}

	// 确定任务类型
	taskType := domain.TaskTypeCustom
	if args.TaskType != "" {
		switch args.TaskType {
		case "agent":
			taskType = domain.TaskTypeAgent
		case "coding":
			taskType = domain.TaskTypeCoding
		case "custom":
			taskType = domain.TaskTypeCustom
		default:
			return &llm.ToolResult{
				Output: fmt.Sprintf(`{"success": false, "error": "无效的 task_type: %s"}`, args.TaskType),
				Error:  "",
			}, nil
		}
	}

	// 确定超时时间
	timeout := int64(60000) // 默认 60 秒
	if args.TimeoutMs > 0 {
		timeout = args.TimeoutMs
	}

	// 确定优先级
	priority := 0
	if args.Priority != 0 {
		priority = args.Priority
	}

	// 构建元数据
	metadata := map[string]interface{}{
		"source":    "agent_tool",
		"tool":      "create_task",
		"createdAt": time.Now().Format(time.RFC3339),
	}

	// 添加上下文信息用于后续任务执行
	if t.agentCode != "" {
		metadata["agent_code"] = t.agentCode
	}
	if t.userCode != "" {
		metadata["user_code"] = t.userCode
	}
	if t.channelCode != "" {
		metadata["channel_code"] = t.channelCode
	}
	if t.sessionKey != "" {
		metadata["session_key"] = t.sessionKey
	}

	// 构建创建命令
	cmd := application.CreateTaskCommand{
		Name:        args.Name,
		Description: args.Description,
		Type:        taskType,
		Metadata:    metadata,
		Timeout:     timeout,
		MaxRetries:  0,
		Priority:    priority,
	}

	// 设置 TraceID 和 SpanID（继承自当前会话）
	if t.traceID != "" {
		traceID := domain.NewTraceID(t.traceID)
		cmd.TraceID = &traceID
	}
	if t.spanID != "" {
		spanID := domain.NewSpanID(t.spanID)
		cmd.SpanID = &spanID
	}

	// 处理父任务 ID
	if args.ParentID != "" {
		parentID := domain.NewTaskID(args.ParentID)
		cmd.ParentID = &parentID
	}

	// 创建任务
	task, err := t.taskService.CreateTask(ctx, cmd)
	if err != nil {
		return &llm.ToolResult{
			Output: fmt.Sprintf(`{"success": false, "error": "创建任务失败: %v"}`, err),
			Error:  "",
		}, nil
	}

	// 构建成功响应
	result := map[string]interface{}{
		"success":    true,
		"task_id":    task.ID().String(),
		"name":       task.Name(),
		"status":     task.Status().String(),
		"created_at": task.CreatedAt().Format(time.RFC3339),
	}

	resultJSON, _ := json.Marshal(result)
	return &llm.ToolResult{
		Output: string(resultJSON),
		Error:  "",
	}, nil
}
