package http

import (
	"encoding/json"
	"net/http"

	"github.com/weibh/taskmanager/application"
)

// ProjectStateMachineHandler 项目状态机 HTTP 处理
type ProjectStateMachineHandler struct {
	service *application.ProjectStateMachineApplicationService
}

// NewProjectStateMachineHandler 创建 handler
func NewProjectStateMachineHandler(service *application.ProjectStateMachineApplicationService) *ProjectStateMachineHandler {
	return &ProjectStateMachineHandler{service: service}
}

// SetProjectStateMachineRequest 设置项目状态机请求
type SetProjectStateMachineRequest struct {
	RequirementType string `json:"requirement_type"`
	StateMachineID  string `json:"state_machine_id"`
}

// ProjectStateMachineResponse 项目状态机响应
type ProjectStateMachineResponse struct {
	ID               string `json:"id"`
	RequirementType  string `json:"requirement_type"`
	StateMachineID   string `json:"state_machine_id"`
	StateMachineName string `json:"state_machine_name,omitempty"`
	CreatedAt        int64  `json:"created_at"`
	UpdatedAt        int64  `json:"updated_at"`
}

// ListProjectStateMachines 列出项目的所有状态机映射
func (h *ProjectStateMachineHandler) ListProjectStateMachines(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "project_id is required")
		return
	}

	mappings, err := h.service.GetProjectStateMachines(r.Context(), application.GetProjectStateMachinesQuery{
		ProjectID: projectID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, mappings)
}

// SetProjectStateMachine 设置项目状态机映射
func (h *ProjectStateMachineHandler) SetProjectStateMachine(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "project_id is required")
		return
	}

	var req SetProjectStateMachineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RequirementType == "" || req.StateMachineID == "" {
		writeError(w, http.StatusBadRequest, "requirement_type and state_machine_id are required")
		return
	}

	mapping, err := h.service.SetProjectStateMachine(r.Context(), application.SetProjectStateMachineCommand{
		ProjectID:       projectID,
		RequirementType: req.RequirementType,
		StateMachineID:  req.StateMachineID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, mapping)
}

// DeleteProjectStateMachine 删除项目状态机映射
func (h *ProjectStateMachineHandler) DeleteProjectStateMachine(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.service.DeleteProjectStateMachine(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetProjectStateMachineByType 获取指定类型的项目状态机映射
func (h *ProjectStateMachineHandler) GetProjectStateMachineByType(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "project_id is required")
		return
	}

	requirementType := r.PathValue("requirement_type")
	if requirementType == "" {
		writeError(w, http.StatusBadRequest, "requirement_type is required")
		return
	}

	mapping, err := h.service.GetProjectStateMachineByType(r.Context(), projectID, requirementType)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, mapping)
}

// GetAvailableRequirementTypes 获取可用的需求类型列表
func (h *ProjectStateMachineHandler) GetAvailableRequirementTypes(w http.ResponseWriter, r *http.Request) {
	types := h.service.GetAvailableRequirementTypes()
	writeJSON(w, map[string]interface{}{
		"types": types,
	})
}
