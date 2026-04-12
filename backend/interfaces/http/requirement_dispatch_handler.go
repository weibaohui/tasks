package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type DispatchRequirementRequest struct {
	RequirementID string `json:"requirement_id"`
	AgentCode     string `json:"agent_code"`
	ChannelCode   string `json:"channel_code"`
	SessionKey    string `json:"session_key"`
}

type ReportRequirementPRRequest struct {
	RequirementID string `json:"requirement_id"`
}

func (h *RequirementHandler) DispatchRequirement(c *gin.Context) {
	var req DispatchRequirementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	result, err := h.dispatchService.DispatchRequirement(c.Request.Context(), application.DispatchRequirementCommand{
		RequirementID: domain.NewRequirementID(req.RequirementID),
		AgentCode:     req.AgentCode,
		ChannelCode:   req.ChannelCode,
		SessionKey:    req.SessionKey,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *RequirementHandler) ReportRequirementPROpened(c *gin.Context) {
	var req ReportRequirementPRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	requirement, err := h.requirementService.ReportRequirementPROpened(c.Request.Context(), application.ReportRequirementPRCommand{
		ID: domain.NewRequirementID(req.RequirementID),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.requirementToMap(requirement))
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

func (h *RequirementHandler) RedispatchRequirement(c *gin.Context) {
	var req RedispatchRequirementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	requirement, err := h.requirementService.RedispatchRequirement(c.Request.Context(), application.RedispatchRequirementCommand{
		ID: domain.NewRequirementID(req.RequirementID),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.requirementToMap(requirement))
}

func (h *RequirementHandler) CopyAndDispatchRequirement(c *gin.Context) {
	var req RedispatchRequirementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	requirement, err := h.requirementService.CopyAndDispatchRequirement(c.Request.Context(), application.CopyAndDispatchRequirementCommand{
		ID: domain.NewRequirementID(req.RequirementID),
	}, h.dispatchService)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.requirementToMap(requirement))
}
