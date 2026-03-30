package http

import (
	"encoding/json"
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
}

type UpdateRequirementRequest struct {
	ID                 string `json:"id"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	TempWorkspaceRoot  string `json:"temp_workspace_root"`
}

type DispatchRequirementRequest struct {
	RequirementID string `json:"requirement_id"`
	AgentID       string `json:"agent_id"`
	ChannelCode   string `json:"channel_code"`
	SessionKey    string `json:"session_key"`
}

type ReportRequirementPRRequest struct {
	RequirementID string `json:"requirement_id"`
	PRURL         string `json:"pr_url"`
	BranchName    string `json:"branch_name"`
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
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(requirementToMap(requirement))
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
	_ = json.NewEncoder(w).Encode(requirementToMap(requirement))
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
		resp = append(resp, requirementToMap(requirement))
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
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(requirementToMap(requirement))
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
		AgentID:       domain.NewAgentID(req.AgentID),
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
		ID:         domain.NewRequirementID(req.RequirementID),
		PRURL:      req.PRURL,
		BranchName: req.BranchName,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(requirementToMap(requirement))
}

func requirementToMap(requirement *domain.Requirement) map[string]interface{} {
	startedAt := interface{}(nil)
	if requirement.StartedAt() != nil {
		startedAt = requirement.StartedAt().UnixMilli()
	}
	completedAt := interface{}(nil)
	if requirement.CompletedAt() != nil {
		completedAt = requirement.CompletedAt().UnixMilli()
	}
	return map[string]interface{}{
		"id":                  requirement.ID().String(),
		"project_id":          requirement.ProjectID().String(),
		"title":               requirement.Title(),
		"description":         requirement.Description(),
		"acceptance_criteria": requirement.AcceptanceCriteria(),
		"temp_workspace_root": requirement.TempWorkspaceRoot(),
		"status":              requirement.Status(),
		"dev_state":           requirement.DevState(),
		"assignee_agent_id":   requirement.AssigneeAgentID(),
		"replica_agent_id":    requirement.ReplicaAgentID(),
		"workspace_path":      requirement.WorkspacePath(),
		"branch_name":         requirement.BranchName(),
		"pr_url":              requirement.PRURL(),
		"last_error":          requirement.LastError(),
		"started_at":          startedAt,
		"completed_at":        completedAt,
		"created_at":          requirement.CreatedAt().UnixMilli(),
		"updated_at":          requirement.UpdatedAt().UnixMilli(),
	}
}
