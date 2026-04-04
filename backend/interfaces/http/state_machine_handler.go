package http

import (
	"encoding/json"
	"net/http"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain/state_machine"
)

// StateMachineHandler 状态机 HTTP 处理
type StateMachineHandler struct {
	service *application.StateMachineService
}

// NewStateMachineHandler 创建 handler
func NewStateMachineHandler(service *application.StateMachineService) *StateMachineHandler {
	return &StateMachineHandler{service: service}
}

// CreateStateMachineRequest 创建状态机请求
type CreateStateMachineRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Config      string `json:"config"` // YAML 内容
}

// BindTypeRequest 绑定类型请求
type BindTypeRequest struct {
	RequirementType string `json:"requirement_type"`
}

// TriggerTransitionRequest 触发转换请求
type TriggerTransitionRequest struct {
	Trigger     string `json:"trigger"`
	TriggeredBy string `json:"triggered_by"`
	Remark      string `json:"remark"`
}

// ListStateMachines 列出状态机
func (h *StateMachineHandler) ListStateMachines(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "project_id is required")
		return
	}

	sms, err := h.service.ListStateMachines(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, sms)
}

// CreateStateMachine 创建状态机
func (h *StateMachineHandler) CreateStateMachine(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "project_id is required")
		return
	}

	var req CreateStateMachineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Config == "" {
		writeError(w, http.StatusBadRequest, "name and config are required")
		return
	}

	sm, err := h.service.CreateStateMachine(r.Context(), projectID, req.Name, req.Description, req.Config)
	if err != nil {
		if smErr, ok := err.(*state_machine.StateMachineError); ok {
			writeError(w, http.StatusBadRequest, smErr.Message)
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, sm)
}

// GetStateMachine 获取状态机
func (h *StateMachineHandler) GetStateMachine(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	sm, err := h.service.GetStateMachine(r.Context(), id)
	if err != nil {
		if _, ok := err.(*state_machine.StateMachineError); ok {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, sm)
}

// DeleteStateMachine 删除状态机
func (h *StateMachineHandler) DeleteStateMachine(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.service.DeleteStateMachine(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateStateMachine 更新状态机
func (h *StateMachineHandler) UpdateStateMachine(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req CreateStateMachineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sm, err := h.service.CreateStateMachine(r.Context(), id, req.Name, req.Description, req.Config)
	if err != nil {
		if smErr, ok := err.(*state_machine.StateMachineError); ok {
			writeError(w, http.StatusBadRequest, smErr.Message)
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, sm)
}

// BindType 绑定类型
func (h *StateMachineHandler) BindType(w http.ResponseWriter, r *http.Request) {
	stateMachineID := r.PathValue("id")
	if stateMachineID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req BindTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RequirementType == "" {
		writeError(w, http.StatusBadRequest, "requirement_type is required")
		return
	}

	if err := h.service.BindType(r.Context(), stateMachineID, req.RequirementType); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// UnbindType 解绑类型
func (h *StateMachineHandler) UnbindType(w http.ResponseWriter, r *http.Request) {
	stateMachineID := r.PathValue("id")
	requirementType := r.PathValue("type")
	if stateMachineID == "" || requirementType == "" {
		writeError(w, http.StatusBadRequest, "id and type are required")
		return
	}

	if err := h.service.UnbindType(r.Context(), stateMachineID, requirementType); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TriggerTransition 触发转换
func (h *StateMachineHandler) TriggerTransition(w http.ResponseWriter, r *http.Request) {
	requirementID := r.PathValue("requirement_id")
	if requirementID == "" {
		writeError(w, http.StatusBadRequest, "requirement_id is required")
		return
	}

	var req TriggerTransitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Trigger == "" {
		writeError(w, http.StatusBadRequest, "trigger is required")
		return
	}

	if req.TriggeredBy == "" {
		req.TriggeredBy = "api"
	}

	rs, err := h.service.TriggerTransition(r.Context(), requirementID, req.Trigger, req.TriggeredBy, req.Remark)
	if err != nil {
		if smErr, ok := err.(*state_machine.StateMachineError); ok {
			switch smErr.Code {
			case "TRANSITION_NOT_FOUND", "STATE_NOT_FOUND":
				writeError(w, http.StatusBadRequest, smErr.Message)
			default:
				writeError(w, http.StatusInternalServerError, smErr.Message)
			}
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, rs)
}

// GetRequirementState 获取需求状态
func (h *StateMachineHandler) GetRequirementState(w http.ResponseWriter, r *http.Request) {
	requirementID := r.PathValue("requirement_id")
	if requirementID == "" {
		writeError(w, http.StatusBadRequest, "requirement_id is required")
		return
	}

	rs, err := h.service.GetRequirementState(r.Context(), requirementID)
	if err != nil {
		if _, ok := err.(*state_machine.StateMachineError); ok {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, rs)
}

// GetTransitionHistory 获取转换历史
func (h *StateMachineHandler) GetTransitionHistory(w http.ResponseWriter, r *http.Request) {
	requirementID := r.PathValue("requirement_id")
	if requirementID == "" {
		writeError(w, http.StatusBadRequest, "requirement_id is required")
		return
	}

	logs, err := h.service.GetTransitionHistory(r.Context(), requirementID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, logs)
}

// GetProjectStateSummary 获取项目状态统计
func (h *StateMachineHandler) GetProjectStateSummary(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "project_id is required")
		return
	}

	summary, err := h.service.GetProjectStateSummary(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, summary)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(HTTPError{Code: code, Message: message})
}
