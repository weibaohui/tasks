/**
 * CreateTaskTool - 创建任务的 LLM 工具
 * 允许 Agent 通过工具调用创建新任务
 */
package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/llm"
	"github.com/weibh/taskmanager/infrastructure/trace"
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
}

// NewCreateTaskTool 创建任务创建工具
func NewCreateTaskTool(
	taskService *application.TaskApplicationService,
	idGenerator domain.IDGenerator,
	agentCode, userCode, channelCode, sessionKey string,
) *CreateTaskTool {
	return &CreateTaskTool{
		taskService: taskService,
		idGenerator: idGenerator,
		agentCode:   agentCode,
		userCode:    userCode,
		channelCode: channelCode,
		sessionKey:  sessionKey,
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
参数 task_requirement: 任务目标/要求（必填）- 描述任务要达成的具体目标
参数 acceptance_criteria: 验收标准（必填）- 描述如何判断任务完成
参数 description: 任务描述（可选）
参数 task_type: 任务类型（可选），可选值: agent(智能体), coding(编码), custom(自定义)，默认 agent
参数 timeout_s: 超时时间秒数（可选），默认 60
参数 priority: 优先级（可选），默认 0
参数 parent_id: 父任务 ID（可选），用于创建子任务

示例：create_task(name="测试任务", task_requirement="执行单元测试", acceptance_criteria="所有测试通过", task_type="agent")`
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
			"task_requirement": {
				"type": "string",
				"description": "任务目标/要求（必填）- 描述任务要达成的具体目标"
			},
			"acceptance_criteria": {
				"type": "string",
				"description": "验收标准（必填）- 描述如何判断任务完成"
			},
			"description": {
				"type": "string",
				"description": "任务描述（可选）"
			},
			"task_type": {
				"type": "string",
				"description": "任务类型（可选），可选值: agent(智能体), coding(编码), custom(自定义)，默认 agent"
			},
			"timeout_s": {
				"type": "integer",
				"description": "超时时间秒数（可选），默认 60"
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
		"required": ["name", "task_requirement", "acceptance_criteria"]
	}`)
}

// Execute 执行工具
func (t *CreateTaskTool) Execute(ctx context.Context, input json.RawMessage) (*llm.ToolResult, error) {
	var args struct {
		Name               string `json:"name"`
		TaskRequirement    string `json:"task_requirement"`
		AcceptanceCriteria string `json:"acceptance_criteria"`
		Description        string `json:"description"`
		TaskType           string `json:"task_type"`
		TimeoutS           int64  `json:"timeout_s"`
		Priority           int    `json:"priority"`
		ParentID           string `json:"parent_id"`
	}

	if err := json.Unmarshal(input, &args); err != nil {
		return &llm.ToolResult{
			Output: fmt.Sprintf(`{"success": false, "error": "解析参数失败: %v"}`, err),
			Error:  "",
		}, nil
	}

	// 验证必填参数
	if strings.TrimSpace(args.Name) == "" {
		return &llm.ToolResult{
			Output: `{"success": false, "error": "缺少必填参数 name（任务名称）"}`,
			Error:  "",
		}, nil
	}

	// 验证必填参数
	if strings.TrimSpace(args.TaskRequirement) == "" {
		return &llm.ToolResult{
			Output: `{"success": false, "error": "缺少必填参数 task_requirement（任务目标）"}`,
			Error:  "",
		}, nil
	}

	// 验证必填参数
	if strings.TrimSpace(args.AcceptanceCriteria) == "" {
		return &llm.ToolResult{
			Output: `{"success": false, "error": "缺少必填参数 acceptance_criteria（验收标准）"}`,
			Error:  "",
		}, nil
	}

	// 确定任务类型，默认 agent（智能体）
	taskType := domain.TaskTypeAgent
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
	timeout := int64(60) // 默认 60 秒
	if args.TimeoutS > 0 {
		timeout = args.TimeoutS
	}

	// 确定优先级
	priority := 0
	if args.Priority != 0 {
		priority = args.Priority
	}

	// 构建创建命令
	cmd := application.CreateTaskCommand{
		Name:               args.Name,
		TaskRequirement:    args.TaskRequirement,
		AcceptanceCriteria: args.AcceptanceCriteria,
		Description:        args.Description,
		Type:               taskType,
		Timeout:            timeout,
		MaxRetries:         0,
		Priority:           priority,
	}

	// 添加上下文信息到命令（用于设置独立字段）
	if t.agentCode != "" {
		cmd.AgentCode = t.agentCode
	}
	if t.userCode != "" {
		cmd.UserCode = t.userCode
	}
	if t.channelCode != "" {
		cmd.ChannelCode = t.channelCode
	}
	if t.sessionKey != "" {
		cmd.SessionKey = t.sessionKey
	}

	// 设置 TraceID 和 SpanID（从 ctx 提取）
	traceIDStr := trace.GetTraceID(ctx)
	spanIDStr := trace.MustGetSpanID(ctx)
	if traceIDStr != "" {
		traceIDVal := domain.NewTraceID(traceIDStr)
		cmd.TraceID = &traceIDVal
	}
	if spanIDStr != "" {
		spanIDVal := domain.NewSpanID(spanIDStr)
		cmd.SpanID = &spanIDVal
	}

	// 设置 ParentSpanID（从 ctx 提取）- 用于 trace 链路
	// 优先从 HookContext metadata 获取（PreToolCall 设置），其次从 trace context 获取
	parentSpanID := trace.GetParentSpanID(ctx)
	if parentSpanID == "" {
		// 尝试从 HookContext metadata 获取
		if hc, ok := ctx.(*domain.HookContext); ok {
			parentSpanID = hc.GetMetadata("span_id")
		}
	}
	if parentSpanID != "" {
		cmd.ParentSpanID = parentSpanID
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

	// 启动任务并提交到工作池执行
	if err := t.taskService.StartTask(ctx, task.ID()); err != nil {
		// 启动失败也返回成功，因为任务已创建成功
		// 任务保持 Pending 状态，可以后续手动启动
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
