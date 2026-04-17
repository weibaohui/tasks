package http

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type GitHubWebhookHandler struct {
	webhookService *application.GitHubWebhookService
	forwardManager *application.WebhookForwardManager
	authHandler    *AuthHandler
}

func NewGitHubWebhookHandler(webhookService *application.GitHubWebhookService, forwardManager *application.WebhookForwardManager, authHandler *AuthHandler) *GitHubWebhookHandler {
	return &GitHubWebhookHandler{
		webhookService: webhookService,
		forwardManager: forwardManager,
		authHandler:    authHandler,
	}
}

// Authorize 验证请求权限
func (h *GitHubWebhookHandler) Authorize(r *http.Request) (*domain.User, error) {
	if h.authHandler == nil {
		return nil, errors.New("auth handler not available")
	}
	return h.authHandler.Authorize(r)
}

// CreateConfigRequest 创建 webhook 配置请求
type CreateConfigRequest struct {
	ProjectID string `json:"project_id" binding:"required"`
	Repo      string `json:"repo" binding:"required"`
}

// UpdateConfigRequest 更新配置请求
type UpdateConfigRequest struct {
	Repo string `json:"repo" binding:"required"`
}

// CreateWebhookBindingRequest 创建心跳绑定请求
type CreateWebhookBindingRequest struct {
	ProjectID   string `json:"project_id" binding:"required"`
	ConfigID    string `json:"config_id" binding:"required"`
	EventType   string `json:"event_type" binding:"required"`
	HeartbeatID string `json:"heartbeat_id" binding:"required"`
}

// ListConfigs 列出所有 webhook 配置
func (h *GitHubWebhookHandler) ListConfigs(c *gin.Context) {
	configs, err := h.webhookService.ListConfigs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(configs))
	for _, config := range configs {
		running, webhookURL, _ := h.forwardManager.GetStatus(config.ID().String(), config.ProjectID().String(), config.Repo())
		resp = append(resp, configToMap(config, running, webhookURL))
	}
	c.JSON(http.StatusOK, resp)
}

// CreateConfig 创建 webhook 配置
func (h *GitHubWebhookHandler) CreateConfig(c *gin.Context) {
	var req CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	config, err := h.webhookService.CreateConfig(c.Request.Context(), req.ProjectID, req.Repo)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, configToMap(config, false, ""))
}

// UpdateConfig 更新 webhook 配置
func (h *GitHubWebhookHandler) UpdateConfig(c *gin.Context) {
	id := c.Param("id")
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	if err := h.webhookService.UpdateConfigRepo(c.Request.Context(), id, req.Repo); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// DeleteConfig 删除 webhook 配置
func (h *GitHubWebhookHandler) DeleteConfig(c *gin.Context) {
	id := c.Param("id")
	config, err := h.webhookService.GetConfig(c.Request.Context(), id)
	if err != nil || config == nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "config not found"})
		return
	}
	// 先删除 GitHub webhook
	h.forwardManager.StopForwarder(config.ID().String(), config.ProjectID().String(), config.Repo())
	if err := h.webhookService.DeleteConfig(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// EnableWebhook 启用 webhook（创建 GitHub webhook）
func (h *GitHubWebhookHandler) EnableWebhook(c *gin.Context) {
	id := c.Param("id")
	config, err := h.webhookService.GetConfig(c.Request.Context(), id)
	if err != nil || config == nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "config not found"})
		return
	}

	// 创建 GitHub webhook（使用 config_id 路径区分项目）
	if err := h.forwardManager.StartForwarder(c.Request.Context(), config.ID().String(), config.ProjectID().String(), config.Repo()); err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "failed to start webhook: " + err.Error()})
		return
	}

	// 更新配置状态
	config.SetEnabled(true)
	if err := h.webhookService.SetConfigEnabled(c.Request.Context(), id, true); err != nil {
		log.Printf("[WEBHOOK] failed to update config enabled status: %v", err)
	}

	running, webhookURL, _ := h.forwardManager.GetStatus(config.ID().String(), config.ProjectID().String(), config.Repo())
	c.JSON(http.StatusOK, configToMap(config, running, webhookURL))
}

// DisableWebhook 停用 webhook（删除 GitHub webhook）
func (h *GitHubWebhookHandler) DisableWebhook(c *gin.Context) {
	id := c.Param("id")
	config, err := h.webhookService.GetConfig(c.Request.Context(), id)
	if err != nil || config == nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "config not found"})
		return
	}

	// 删除 GitHub webhook
	if err := h.forwardManager.StopForwarder(config.ID().String(), config.ProjectID().String(), config.Repo()); err != nil {
		log.Printf("[WEBHOOK] failed to stop webhook: %v", err)
	}

	// 更新配置状态
	config.SetEnabled(false)
	if err := h.webhookService.SetConfigEnabled(c.Request.Context(), id, false); err != nil {
		log.Printf("[WEBHOOK] failed to update config enabled status: %v", err)
	}

	running, webhookURL, _ := h.forwardManager.GetStatus(config.ID().String(), config.ProjectID().String(), config.Repo())
	c.JSON(http.StatusOK, configToMap(config, running, webhookURL))
}

// GetForwarderStatus 获取 forwarder 运行状态
func (h *GitHubWebhookHandler) GetForwarderStatus(c *gin.Context) {
	id := c.Param("id")
	config, err := h.webhookService.GetConfig(c.Request.Context(), id)
	if err != nil || config == nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "config not found"})
		return
	}
	running, webhookURL, _ := h.forwardManager.GetStatus(config.ID().String(), config.ProjectID().String(), config.Repo())
	c.JSON(http.StatusOK, gin.H{
		"running":     running,
		"webhook_url": webhookURL,
	})
}

// ListEventLogs 列出事件日志
func (h *GitHubWebhookHandler) ListEventLogs(c *gin.Context) {
	configID := c.Param("id")
	config, err := h.webhookService.GetConfig(c.Request.Context(), configID)
	if err != nil || config == nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "config not found"})
		return
	}
	logs, err := h.webhookService.ListEventLogs(c.Request.Context(), config.ProjectID().String(), 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(logs))
	for _, log := range logs {
		resp = append(resp, eventLogToMap(log))
	}
	c.JSON(http.StatusOK, resp)
}

// ListBindings 列出心跳绑定
func (h *GitHubWebhookHandler) ListBindings(c *gin.Context) {
	configID := c.Param("id")
	bindings, err := h.webhookService.ListBindings(c.Request.Context(), configID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(bindings))
	for _, binding := range bindings {
		resp = append(resp, webhookBindingToMap(binding))
	}
	c.JSON(http.StatusOK, resp)
}

// CreateBinding 创建心跳绑定
func (h *GitHubWebhookHandler) CreateBinding(c *gin.Context) {
	var req CreateWebhookBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	binding, err := h.webhookService.CreateBinding(c.Request.Context(), req.ProjectID, req.ConfigID, req.EventType, req.HeartbeatID)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, webhookBindingToMap(binding))
}

// DeleteBinding 删除心跳绑定
func (h *GitHubWebhookHandler) DeleteBinding(c *gin.Context) {
	id := c.Param("id")
	if err := h.webhookService.DeleteBinding(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// ListHeartbeats 列出项目的所有心跳（用于选择绑定）
func (h *GitHubWebhookHandler) ListHeartbeats(c *gin.Context) {
	projectID := c.Query("project_id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "project_id is required"})
		return
	}
	heartbeats, err := h.webhookService.ListHeartbeats(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(heartbeats))
	for _, hb := range heartbeats {
		resp = append(resp, map[string]interface{}{
			"id":               hb.ID().String(),
			"project_id":       hb.ProjectID().String(),
			"name":             hb.Name(),
			"enabled":          hb.Enabled(),
			"interval_minutes": hb.IntervalMinutes(),
			"agent_code":       hb.AgentCode(),
			"requirement_type": hb.RequirementType(),
		})
	}
	c.JSON(http.StatusOK, resp)
}

func configToMap(config *domain.GitHubWebhookConfig, running bool, webhookURL string) map[string]interface{} {
	return map[string]interface{}{
		"id":            config.ID().String(),
		"project_id":    config.ProjectID().String(),
		"repo":          config.Repo(),
		"enabled":       config.Enabled(),
		"webhook_url":   webhookURL,
		"running":       running,
		"created_at":    config.CreatedAt().UnixMilli(),
		"updated_at":    config.UpdatedAt().UnixMilli(),
	}
}

func eventLogToMap(log *domain.WebhookEventLog) map[string]interface{} {
	return map[string]interface{}{
		"id":                  log.ID().String(),
		"project_id":         log.ProjectID().String(),
		"event_type":         log.EventType(),
		"status":             string(log.Status()),
		"trigger_heartbeat_id": log.TriggerHeartbeatID(),
		"error_message":      log.ErrorMessage(),
		"received_at":        log.ReceivedAt().UnixMilli(),
	}
}

func webhookBindingToMap(binding *domain.WebhookHeartbeatBinding) map[string]interface{} {
	return map[string]interface{}{
		"id":                   binding.ID().String(),
		"project_id":          binding.ProjectID().String(),
		"github_webhook_config_id": binding.ConfigID().String(),
		"github_event_type":    binding.GitHubEventType(),
		"heartbeat_id":        binding.HeartbeatID().String(),
		"enabled":             binding.Enabled(),
		"created_at":          binding.CreatedAt().UnixMilli(),
	}
}
