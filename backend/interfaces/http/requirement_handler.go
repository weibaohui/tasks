package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type RequirementHandler struct {
	requirementService *application.RequirementApplicationService
	dispatchService    *application.RequirementDispatchService
}

func NewRequirementHandler(requirementService *application.RequirementApplicationService, dispatchService *application.RequirementDispatchService) *RequirementHandler {
	return &RequirementHandler{
		requirementService: requirementService,
		dispatchService:    dispatchService,
	}
}

type CreateRequirementRequest struct {
	ProjectID          string `json:"project_id"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	TempWorkspaceRoot  string `json:"temp_workspace_root"`
	RequirementType    string `json:"requirement_type"`
}

type UpdateRequirementRequest struct {
	ID                 string  `json:"id"`
	Title              *string `json:"title"`
	Description        *string `json:"description"`
	AcceptanceCriteria *string `json:"acceptance_criteria"`
	TempWorkspaceRoot  *string `json:"temp_workspace_root"`
	RequirementType    *string `json:"requirement_type"`
}

type UpdateRequirementStatusRequest struct {
	ID        string `json:"id"`
	NewStatus string `json:"new_status"`
}

type DispatchRequirementRequest struct {
	RequirementID string `json:"requirement_id"`
	AgentCode     string `json:"agent_code"`
	ChannelCode   string `json:"channel_code"`
	SessionKey    string `json:"session_key"`
}

type ReportRequirementPRRequest struct {
	RequirementID string `json:"requirement_id"`
}

func (h *RequirementHandler) CreateRequirement(w http.ResponseWriter, r *http.Request) {
	var req CreateRequirementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	requirement, err := h.requirementService.CreateRequirement(r.Context(), application.CreateRequirementCommand{
		ProjectID:          domain.NewProjectID(req.ProjectID),
		Title:              req.Title,
		Description:        req.Description,
		AcceptanceCriteria: req.AcceptanceCriteria,
		TempWorkspaceRoot:  req.TempWorkspaceRoot,
		RequirementType:    req.RequirementType,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(h.requirementToMap(r, requirement))
}

func (h *RequirementHandler) GetRequirement(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	requirement, err := h.requirementService.GetRequirement(r.Context(), domain.NewRequirementID(id))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusNotFound, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(h.requirementToMap(r, requirement))
}

func (h *RequirementHandler) ListRequirements(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.URL.Query().Get("project_id")
	var projectID *domain.ProjectID
	if projectIDStr != "" {
		id := domain.NewProjectID(projectIDStr)
		projectID = &id
	}
	requirements, err := h.requirementService.ListRequirements(r.Context(), projectID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(requirements))
	for _, requirement := range requirements {
		resp = append(resp, h.requirementToMap(r, requirement))
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *RequirementHandler) UpdateRequirement(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequirementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	requirement, err := h.requirementService.UpdateRequirement(r.Context(), application.UpdateRequirementCommand{
		ID:                 domain.NewRequirementID(req.ID),
		Title:              req.Title,
		Description:        req.Description,
		AcceptanceCriteria: req.AcceptanceCriteria,
		TempWorkspaceRoot:  req.TempWorkspaceRoot,
		RequirementType:     req.RequirementType,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(h.requirementToMap(r, requirement))
}

func (h *RequirementHandler) UpdateRequirementStatus(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequirementStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	requirement, err := h.requirementService.UpdateRequirementStatus(r.Context(), application.UpdateRequirementStatusCommand{
		ID:        domain.NewRequirementID(req.ID),
		NewStatus: req.NewStatus,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(h.requirementToMap(r, requirement))
}

func (h *RequirementHandler) GetRequirementTransitionHistory(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	history, err := h.requirementService.GetRequirementTransitionHistory(r.Context(), domain.NewRequirementID(id))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	// 转换为响应格式
	resp := make([]map[string]interface{}, 0, len(history))
	for _, log := range history {
		resp = append(resp, map[string]interface{}{
			"id":             log.ID,
			"requirement_id":  log.RequirementID,
			"from_state":     log.FromState,
			"to_state":       log.ToState,
			"trigger":        log.Trigger,
			"triggered_by":   log.TriggeredBy,
			"remark":         log.Remark,
			"result":         log.Result,
			"error_message":  log.ErrorMessage,
			"created_at":     log.CreatedAt.UnixMilli(),
		})
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// GetStatusStats 获取状态统计数据（动态从数据库提取）
func (h *RequirementHandler) GetStatusStats(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.URL.Query().Get("project_id")
	var projectID *domain.ProjectID
	if projectIDStr != "" {
		id := domain.NewProjectID(projectIDStr)
		projectID = &id
	}

	stats, err := h.requirementService.GetStatusStats(r.Context(), projectID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(stats)
}

func (h *RequirementHandler) DispatchRequirement(w http.ResponseWriter, r *http.Request) {

	var req DispatchRequirementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	result, err := h.dispatchService.DispatchRequirement(r.Context(), application.DispatchRequirementCommand{
		RequirementID: domain.NewRequirementID(req.RequirementID),
		AgentCode:     req.AgentCode,
		ChannelCode:   req.ChannelCode,
		SessionKey:    req.SessionKey,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(result)
}

func (h *RequirementHandler) ReportRequirementPROpened(w http.ResponseWriter, r *http.Request) {
	var req ReportRequirementPRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	requirement, err := h.requirementService.ReportRequirementPROpened(r.Context(), application.ReportRequirementPRCommand{
		ID: domain.NewRequirementID(req.RequirementID),
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(h.requirementToMap(r, requirement))
}

type RedispatchRequirementRequest struct {
	RequirementID string `json:"requirement_id"`
}

type DeleteRequirementRequest struct {
	ID string `json:"id"`
}

type BatchDeleteRequirementsRequest struct {
	IDs []string `json:"ids"`
}

func (h *RequirementHandler) RedispatchRequirement(w http.ResponseWriter, r *http.Request) {
	var req RedispatchRequirementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	requirement, err := h.requirementService.RedispatchRequirement(r.Context(), application.RedispatchRequirementCommand{
		ID: domain.NewRequirementID(req.RequirementID),
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(h.requirementToMap(r, requirement))
}

// CopyAndDispatchRequirement 复制需求并派发新副本
func (h *RequirementHandler) CopyAndDispatchRequirement(w http.ResponseWriter, r *http.Request) {

	var req RedispatchRequirementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	requirement, err := h.requirementService.CopyAndDispatchRequirement(r.Context(), application.CopyAndDispatchRequirementCommand{
		ID: domain.NewRequirementID(req.RequirementID),
	}, h.dispatchService)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(h.requirementToMap(r, requirement))
}

func (h *RequirementHandler) DeleteRequirement(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	err := h.requirementService.DeleteRequirement(r.Context(), application.DeleteRequirementCommand{
		ID: domain.NewRequirementID(id),
	})
	if err != nil {
		h.handleDeleteError(w, r.Context(), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RequirementHandler) BatchDeleteRequirements(w http.ResponseWriter, r *http.Request) {
	var req BatchDeleteRequirementsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	if len(req.IDs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "ids is required"})
		return
	}
	ids := make([]domain.RequirementID, 0, len(req.IDs))
	for _, id := range req.IDs {
		ids = append(ids, domain.NewRequirementID(id))
	}
	err := h.requirementService.BatchDeleteRequirements(r.Context(), application.BatchDeleteRequirementsCommand{
		IDs: ids,
	})
	if err != nil {
		h.handleDeleteError(w, r.Context(), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RequirementHandler) requirementToMap(r *http.Request, requirement *domain.Requirement) map[string]interface{} {
	startedAt := interface{}(nil)
	if requirement.StartedAt() != nil {
		startedAt = requirement.StartedAt().UnixMilli()
	}
	completedAt := interface{}(nil)
	if requirement.CompletedAt() != nil {
		completedAt = requirement.CompletedAt().UnixMilli()
	}
	resp := map[string]interface{}{
		"id":                   requirement.ID().String(),
		"project_id":           requirement.ProjectID().String(),
		"title":                requirement.Title(),
		"description":          requirement.Description(),
		"acceptance_criteria":  requirement.AcceptanceCriteria(),
		"temp_workspace_root":  requirement.TempWorkspaceRoot(),
		"status":               requirement.Status(),
		"assignee_agent_code":  requirement.AssigneeAgentCode(),
		"replica_agent_code":   requirement.ReplicaAgentCode(),
		"dispatch_session_key": requirement.DispatchSessionKey(),
		"workspace_path":       requirement.WorkspacePath(),
		"last_error":           requirement.LastError(),
		"trace_id":             requirement.TraceID(),
		"prompt_tokens":        requirement.PromptTokens(),
		"completion_tokens":    requirement.CompletionTokens(),
		"total_tokens":         requirement.TotalTokens(),
		"started_at":           startedAt,
		"completed_at":         completedAt,
		"created_at":           requirement.CreatedAt().UnixMilli(),
		"updated_at":           requirement.UpdatedAt().UnixMilli(),
		"requirement_type":     requirement.RequirementType(),
	}
	resp["claude_runtime"] = h.getClaudeRuntimeByRequirement(r, requirement)
	return resp
}

func (h *RequirementHandler) getClaudeRuntimeByRequirement(r *http.Request, requirement *domain.Requirement) map[string]interface{} {
	result := make(map[string]interface{})

	// 先检查 nil，避免空指针
	if requirement == nil {
		return result
	}

	// 从 requirement 直接获取所有 Claude Runtime 状态字段（已持久化到数据库）
	result["prompt"] = requirement.ClaudeRuntimePrompt()
	result["result"] = requirement.ClaudeRuntimeResult()
	result["status"] = requirement.ClaudeRuntimeStatus()
	result["last_error"] = requirement.ClaudeRuntimeError()

	if startedAt := requirement.ClaudeRuntimeStartedAt(); startedAt != nil {
		result["started_at"] = startedAt.UnixMilli()
	} else {
		result["started_at"] = nil
	}

	if endedAt := requirement.ClaudeRuntimeEndedAt(); endedAt != nil {
		result["ended_at"] = endedAt.UnixMilli()
	} else {
		result["ended_at"] = nil
	}

	return result
}

// handleDeleteError 处理删除操作的错误，根据错误类型返回相应的状态码
// - ErrRequirementNotFound -> 404
// - 上下文取消/超时 -> 500
// - 其他内部错误 -> 500（不暴露原始错误信息）
func (h *RequirementHandler) handleDeleteError(w http.ResponseWriter, ctx context.Context, err error) {
	if errors.Is(err, application.ErrRequirementNotFound) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusNotFound, Message: "requirement not found"})
		return
	}

	// 检查上下文错误
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: "request timeout or cancelled"})
		return
	}

	// 其他内部错误，不暴露原始错误信息
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: "internal server error"})
}
