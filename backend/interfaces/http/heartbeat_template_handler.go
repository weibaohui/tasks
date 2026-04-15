package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type HeartbeatTemplateHandler struct {
	service *application.HeartbeatTemplateApplicationService
}

func NewHeartbeatTemplateHandler(service *application.HeartbeatTemplateApplicationService) *HeartbeatTemplateHandler {
	return &HeartbeatTemplateHandler{service: service}
}

type CreateHeartbeatTemplateRequest struct {
	Name            string `json:"name" binding:"required"`
	MDContent       string `json:"md_content"`
	RequirementType string `json:"requirement_type"`
}

func (h *HeartbeatTemplateHandler) ListTemplates(c *gin.Context) {
	templates, err := h.service.ListHeartbeatTemplates(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(templates))
	for _, t := range templates {
		resp = append(resp, heartbeatTemplateToMap(t))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HeartbeatTemplateHandler) CreateTemplate(c *gin.Context) {
	var req CreateHeartbeatTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	t, err := h.service.CreateHeartbeatTemplate(c.Request.Context(), application.CreateHeartbeatTemplateCommand{
		Name:            req.Name,
		MDContent:       req.MDContent,
		RequirementType: req.RequirementType,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, heartbeatTemplateToMap(t))
}

func (h *HeartbeatTemplateHandler) DeleteTemplate(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteHeartbeatTemplate(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

func heartbeatTemplateToMap(t *domain.HeartbeatTemplate) map[string]interface{} {
	return map[string]interface{}{
		"id":               t.ID().String(),
		"name":             t.Name(),
		"md_content":       t.MDContent(),
		"requirement_type": t.RequirementType(),
		"created_at":       t.CreatedAt().UnixMilli(),
		"updated_at":       t.UpdatedAt().UnixMilli(),
	}
}
