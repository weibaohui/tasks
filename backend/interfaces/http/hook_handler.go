package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type HookHandler struct {
	configRepo domain.RequirementHookConfigRepository
	logRepo   domain.RequirementHookActionLogRepository
	idGen     IDGenerator
}

type IDGenerator interface {
	Generate() string
}

func NewHookHandler(
	configRepo domain.RequirementHookConfigRepository,
	logRepo domain.RequirementHookActionLogRepository,
	idGen IDGenerator,
) *HookHandler {
	return &HookHandler{
		configRepo: configRepo,
		logRepo:   logRepo,
		idGen:     idGen,
	}
}

type CreateHookConfigRequest struct {
	ProjectID    string `json:"project_id"`
	Name         string `json:"name"`
	TriggerPoint string `json:"trigger_point"`
	ActionType  string `json:"action_type"`
	ActionConfig string `json:"action_config"`
	Enabled     bool   `json:"enabled"`
	Priority    int    `json:"priority"`
}

type UpdateHookConfigRequest struct {
	ID           string  `json:"id"`
	ProjectID    *string `json:"project_id"`
	Name         *string `json:"name"`
	TriggerPoint *string `json:"trigger_point"`
	ActionType   *string `json:"action_type"`
	ActionConfig *string `json:"action_config"`
	Enabled      *bool   `json:"enabled"`
	Priority     *int    `json:"priority"`
}

// CreateHookConfig 创建 Hook 配置
func (h *HookHandler) CreateHookConfig(w http.ResponseWriter, r *http.Request) {
	var req CreateHookConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	now := time.Now()
	config := &domain.RequirementHookConfig{
		ID:           h.idGen.Generate(),
		ProjectID:    req.ProjectID,
		Name:         req.Name,
		TriggerPoint: req.TriggerPoint,
		ActionType:  req.ActionType,
		ActionConfig: req.ActionConfig,
		Enabled:     req.Enabled,
		Priority:    req.Priority,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.configRepo.Save(r.Context(), config); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(hookConfigToMap(config))
}

// GetHookConfig 获取单个 Hook 配置
func (h *HookHandler) GetHookConfig(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}

	config, err := h.configRepo.FindByID(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	if config == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusNotFound, Message: "not found"})
		return
	}

	_ = json.NewEncoder(w).Encode(hookConfigToMap(config))
}

// ListHookConfigs 获取所有 Hook 配置
func (h *HookHandler) ListHookConfigs(w http.ResponseWriter, r *http.Request) {
	triggerPoint := r.URL.Query().Get("trigger_point")
	projectID := r.URL.Query().Get("project_id")

	var configs []*domain.RequirementHookConfig
	var err error

	if projectID != "" {
		configs, err = h.configRepo.FindByProjectID(r.Context(), projectID)
	} else if triggerPoint != "" {
		configs, err = h.configRepo.FindByTriggerPoint(r.Context(), triggerPoint)
	} else {
		// 获取所有配置（需要添加 FindAll 方法到接口，暂时用 FindByTriggerPoint 遍历）
		triggerPoints := []string{"start_dispatch", "mark_coding", "claude_code_finished", "mark_failed", "mark_pr_opened"}
		for _, tp := range triggerPoints {
			tpConfigs, tpErr := h.configRepo.FindByTriggerPoint(r.Context(), tp)
			if tpErr != nil {
				err = tpErr
				break
			}
			configs = append(configs, tpConfigs...)
		}
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	resp := make([]map[string]interface{}, 0, len(configs))
	for _, config := range configs {
		resp = append(resp, hookConfigToMap(config))
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// UpdateHookConfig 更新 Hook 配置
func (h *HookHandler) UpdateHookConfig(w http.ResponseWriter, r *http.Request) {
	var req UpdateHookConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	existing, err := h.configRepo.FindByID(r.Context(), req.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	if existing == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusNotFound, Message: "not found"})
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.ProjectID != nil {
		existing.ProjectID = *req.ProjectID
	}
	if req.TriggerPoint != nil {
		existing.TriggerPoint = *req.TriggerPoint
	}
	if req.ActionType != nil {
		existing.ActionType = *req.ActionType
	}
	if req.ActionConfig != nil {
		existing.ActionConfig = *req.ActionConfig
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		existing.Priority = *req.Priority
	}
	existing.UpdatedAt = time.Now()

	if err := h.configRepo.Save(r.Context(), existing); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(hookConfigToMap(existing))
}

// DeleteHookConfig 删除 Hook 配置
func (h *HookHandler) DeleteHookConfig(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}

	if err := h.configRepo.Delete(r.Context(), id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

// EnableHookConfig 启用 Hook 配置
func (h *HookHandler) EnableHookConfig(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}

	config, err := h.configRepo.FindByID(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	if config == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusNotFound, Message: "not found"})
		return
	}

	config.Enabled = true
	config.UpdatedAt = time.Now()

	if err := h.configRepo.Save(r.Context(), config); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(hookConfigToMap(config))
}

// DisableHookConfig 禁用 Hook 配置
func (h *HookHandler) DisableHookConfig(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}

	config, err := h.configRepo.FindByID(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	if config == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusNotFound, Message: "not found"})
		return
	}

	config.Enabled = false
	config.UpdatedAt = time.Now()

	if err := h.configRepo.Save(r.Context(), config); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(hookConfigToMap(config))
}

// ListHookLogs 获取 Hook 执行日志
func (h *HookHandler) ListHookLogs(w http.ResponseWriter, r *http.Request) {
	requirementID := r.URL.Query().Get("requirement_id")
	hookConfigID := r.URL.Query().Get("hook_config_id")

	var logs []*domain.RequirementHookActionLog
	var err error

	if requirementID != "" {
		logs, err = h.logRepo.FindByRequirementID(r.Context(), requirementID)
	} else if hookConfigID != "" {
		logs, err = h.logRepo.FindByHookConfigID(r.Context(), hookConfigID, 100)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "requirement_id or hook_config_id is required"})
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	resp := make([]map[string]interface{}, 0, len(logs))
	for _, log := range logs {
		resp = append(resp, hookActionLogToMap(log))
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func hookConfigToMap(config *domain.RequirementHookConfig) map[string]interface{} {
	return map[string]interface{}{
		"id":            config.ID,
		"name":          config.Name,
		"trigger_point": config.TriggerPoint,
		"action_type":   config.ActionType,
		"action_config": config.ActionConfig,
		"enabled":       config.Enabled,
		"priority":      config.Priority,
		"created_at":    config.CreatedAt.UnixMilli(),
		"updated_at":    config.UpdatedAt.UnixMilli(),
	}
}

func hookActionLogToMap(log *domain.RequirementHookActionLog) map[string]interface{} {
	result := map[string]interface{}{
		"id":             log.ID,
		"hook_config_id":  log.HookConfigID,
		"requirement_id": log.RequirementID,
		"trigger_point":   log.TriggerPoint,
		"action_type":    log.ActionType,
		"status":         log.Status,
		"input_context":  log.InputContext,
		"result":         log.Result,
		"error":          log.Error,
		"started_at":      log.StartedAt.UnixMilli(),
	}

	if log.CompletedAt != nil {
		result["completed_at"] = log.CompletedAt.UnixMilli()
	}

	return result
}
