package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/weibh/taskmanager/domain"
	"go.uber.org/zap"
)

// ProgressTrackingHook 从 todo/todowrite 等工具调用中提取进度并更新 requirement
type ProgressTrackingHook struct {
	*domain.BaseHook
	requirementRepo domain.RequirementRepository
	logger          *zap.Logger
}

// NewProgressTrackingHook 创建进度跟踪 Hook
func NewProgressTrackingHook(
	requirementRepo domain.RequirementRepository,
	logger *zap.Logger,
) *ProgressTrackingHook {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ProgressTrackingHook{
		BaseHook:        domain.NewBaseHook("progress_tracking", 40, domain.HookTypeTool),
		requirementRepo: requirementRepo,
		logger:          logger,
	}
}

// PreToolCall 工具调用前执行进度提取
func (h *ProgressTrackingHook) PreToolCall(ctx *domain.HookContext, callCtx *domain.ToolCallContext) (*domain.ToolCallContext, error) {
	if callCtx == nil || !h.isProgressTool(callCtx.ToolName) {
		return callCtx, nil
	}

	todos, ok := h.extractTodos(callCtx.ToolInput)
	if !ok || len(todos) == 0 {
		return callCtx, nil
	}

	traceID := callCtx.TraceID
	if traceID == "" && ctx != nil {
		traceID = ctx.GetMetadata("trace_id")
	}
	if traceID == "" {
		h.logger.Debug("ProgressTrackingHook: no trace_id found, skipping")
		return callCtx, nil
	}

	progressData := domain.NewProgressData()
	progressData.Items = todos
	progressData.CalculatePercent()

	if err := h.updateRequirementProgress(ctx.Context, traceID, progressData); err != nil {
		h.logger.Warn("ProgressTrackingHook: failed to update requirement progress",
			zap.String("trace_id", traceID),
			zap.Error(err))
	} else {
		h.logger.Info("ProgressTrackingHook: updated requirement progress",
			zap.String("trace_id", traceID),
			zap.Int("percent", progressData.Percent),
			zap.Int("total_items", len(todos)))
	}

	return callCtx, nil
}

// PostToolCall 工具完成后再次提取（某些工具可能在结果中返回更新后的状态）
func (h *ProgressTrackingHook) PostToolCall(ctx *domain.HookContext, callCtx *domain.ToolCallContext, result *domain.ToolExecutionResult) (*domain.ToolExecutionResult, error) {
	// todo 工具的进度主要在 PreToolCall 时从参数中提取
	// PostToolCall 这里暂时不需要额外处理，因为 todowrite 的参数已经包含完整状态
	return result, nil
}

// OnToolError 工具错误时不处理
func (h *ProgressTrackingHook) OnToolError(ctx *domain.HookContext, callCtx *domain.ToolCallContext, err error) (*domain.ToolExecutionResult, error) {
	return &domain.ToolExecutionResult{Success: false, Error: err}, nil
}

// isProgressTool 判断是否是进度相关的工具
func (h *ProgressTrackingHook) isProgressTool(toolName string) bool {
	lower := strings.ToLower(toolName)
	return lower == "todowrite" || lower == "todo" || lower == "todo_write"
}

// extractTodos 从工具输入中提取 todo 列表（兼容多种 Agent 格式）
func (h *ProgressTrackingHook) extractTodos(input map[string]interface{}) ([]domain.TodoItem, bool) {
	if input == nil {
		return nil, false
	}

	rawTodos, ok := input["todos"]
	if !ok {
		return nil, false
	}

	var todosSlice []interface{}
	switch v := rawTodos.(type) {
	case []interface{}:
		todosSlice = v
	case string:
		// 某些 Agent 将 todos 序列化为 JSON 字符串，需要二次解析
		if err := json.Unmarshal([]byte(v), &todosSlice); err != nil {
			return nil, false
		}
	default:
		return nil, false
	}

	var items []domain.TodoItem
	for _, raw := range todosSlice {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		content := ""
		status := ""
		priority := ""
		// 部分 Agent（如 OpenCode）使用 activeForm 作为显示文本，优先使用
		if v, ok := item["activeForm"].(string); ok && v != "" {
			content = v
		}
		if content == "" {
			if v, ok := item["content"].(string); ok {
				content = v
			}
		}
		if v, ok := item["status"].(string); ok {
			status = v
		}
		if v, ok := item["priority"].(string); ok {
			priority = v
		}
		// 兼容其他可能的字段名
		if content == "" {
			if v, ok := item["task"].(string); ok {
				content = v
			} else if v, ok := item["title"].(string); ok {
				content = v
			}
		}
		items = append(items, domain.TodoItem{
			Content:  content,
			Status:   status,
			Priority: priority,
		})
	}

	return items, len(items) > 0
}

// updateRequirementProgress 更新需求的进度数据
func (h *ProgressTrackingHook) updateRequirementProgress(ctx context.Context, traceID string, data *domain.ProgressData) error {
	if h.requirementRepo == nil {
		return fmt.Errorf("requirementRepo is nil")
	}

	requirement, err := h.requirementRepo.FindByTraceID(ctx, traceID)
	if err != nil {
		return fmt.Errorf("find requirement by trace_id failed: %w", err)
	}
	if requirement == nil {
		return fmt.Errorf("requirement not found for trace_id: %s", traceID)
	}

	requirement.SetProgressData(data)
	return h.requirementRepo.Save(ctx, requirement)
}

// extractTodosFromJSON 从 JSON 字符串解析 todo 列表（备用）
func extractTodosFromJSON(s string) ([]domain.TodoItem, bool) {
	var wrapper struct {
		Todos []domain.TodoItem `json:"todos"`
	}
	if err := json.Unmarshal([]byte(s), &wrapper); err != nil {
		return nil, false
	}
	return wrapper.Todos, len(wrapper.Todos) > 0
}
