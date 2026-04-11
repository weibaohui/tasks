package http

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain/statemachine"
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

// TriggerTransitionRequest 触发转换请求
type TriggerTransitionRequest struct {
	Trigger     string                 `json:"trigger"`
	TriggeredBy string                 `json:"triggered_by"`
	Remark      string                 `json:"remark"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ListStateMachines 列出状态机
func (h *StateMachineHandler) ListStateMachines(c *gin.Context) {
	sms, err := h.service.ListStateMachines(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, sms)
}

// CreateStateMachine 创建状态机
func (h *StateMachineHandler) CreateStateMachine(c *gin.Context) {
	var req CreateStateMachineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request body"})
		return
	}

	if req.Name == "" || req.Config == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "name and config are required"})
		return
	}

	sm, err := h.service.CreateStateMachine(c.Request.Context(), req.Name, req.Description, req.Config)
	if err != nil {
		if smErr, ok := err.(*statemachine.StateMachineError); ok {
			c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: smErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, sm)
}

// GetStateMachine 获取状态机
func (h *StateMachineHandler) GetStateMachine(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}

	sm, err := h.service.GetStateMachine(c.Request.Context(), id)
	if err != nil {
		if _, ok := err.(*statemachine.StateMachineError); ok {
			c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, sm)
}

// DeleteStateMachine 删除状态机
func (h *StateMachineHandler) DeleteStateMachine(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}

	if err := h.service.DeleteStateMachine(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateStateMachine 更新状态机
func (h *StateMachineHandler) UpdateStateMachine(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}

	var req CreateStateMachineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request body"})
		return
	}

	sm, err := h.service.UpdateStateMachine(c.Request.Context(), id, req.Name, req.Description, req.Config)
	if err != nil {
		if smErr, ok := err.(*statemachine.StateMachineError); ok {
			c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: smErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, sm)
}

// TriggerTransition 触发转换
func (h *StateMachineHandler) TriggerTransition(c *gin.Context) {
	requirementID := c.Param("requirement_id")
	if requirementID == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "requirement_id is required"})
		return
	}

	var req TriggerTransitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request body"})
		return
	}

	if req.Trigger == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "trigger is required"})
		return
	}

	if req.TriggeredBy == "" {
		req.TriggeredBy = "api"
	}

	// 将 metadata 存入 context
	ctx := c.Request.Context()
	if req.Metadata != nil {
		ctx = statemachine.WithMetadata(ctx, req.Metadata)
	}

	rs, err := h.service.TriggerTransition(ctx, requirementID, req.Trigger, req.TriggeredBy, req.Remark)
	if err != nil {
		if smErr, ok := err.(*statemachine.StateMachineError); ok {
			switch smErr.Code {
			case "TRANSITION_NOT_FOUND", "STATE_NOT_FOUND":
				c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: smErr.Message})
			default:
				c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: smErr.Message})
			}
			return
		}
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, rs)
}

// GetRequirementState 获取需求状态
func (h *StateMachineHandler) GetRequirementState(c *gin.Context) {
	requirementID := c.Param("requirement_id")
	if requirementID == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "requirement_id is required"})
		return
	}

	rs, err := h.service.GetRequirementState(c.Request.Context(), requirementID)
	if err != nil {
		if _, ok := err.(*statemachine.StateMachineError); ok {
			c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, rs)
}

// InitializeRequirementStateRequest 初始化需求状态请求
type InitializeRequirementStateRequest struct {
	StateMachineID string `json:"state_machine_id"`
}

// InitializeRequirementState 初始化需求状态
func (h *StateMachineHandler) InitializeRequirementState(c *gin.Context) {
	requirementID := c.Param("requirement_id")
	if requirementID == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "requirement_id is required"})
		return
	}

	var req InitializeRequirementStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request body"})
		return
	}

	if req.StateMachineID == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "state_machine_id is required"})
		return
	}

	rs, err := h.service.InitializeRequirementState(c.Request.Context(), requirementID, req.StateMachineID)
	if err != nil {
		if smErr, ok := err.(*statemachine.StateMachineError); ok {
			c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: smErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, rs)
}

// GetTransitionHistory 获取转换历史
func (h *StateMachineHandler) GetTransitionHistory(c *gin.Context) {
	requirementID := c.Param("requirement_id")
	if requirementID == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "requirement_id is required"})
		return
	}

	logs, err := h.service.GetTransitionHistory(c.Request.Context(), requirementID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// GetStateSummary 获取状态统计
func (h *StateMachineHandler) GetStateSummary(c *gin.Context) {
	summary, err := h.service.GetStateSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// writeJSON 辅助函数，用于直接写 JSON（兼容性保留）
func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// writeError 辅助函数，用于写错误响应（兼容性保留）
func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(HTTPError{Code: code, Message: message})
}
