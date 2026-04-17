package http

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type GitHubWebhookHandler struct {
	webhookService   *application.GitHubWebhookService
	webhookGitHub   *application.WebhookGitHubManager
	authHandler     *AuthHandler
}

func NewGitHubWebhookHandler(webhookService *application.GitHubWebhookService, webhookGitHub *application.WebhookGitHubManager, authHandler *AuthHandler) *GitHubWebhookHandler {
	return &GitHubWebhookHandler{
		webhookService:   webhookService,
		webhookGitHub:   webhookGitHub,
		authHandler:     authHandler,
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
		resp = append(resp, configToMap(config))
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
	c.JSON(http.StatusCreated, configToMap(config))
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
	if config.Enabled() {
		h.webhookGitHub.DeleteWebhook(c.Request.Context(), config.ID().String(), config.ProjectID().String(), config.Repo())
	}
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

	// 创建 GitHub webhook 并获取 URL
	webhookURL, err := h.webhookGitHub.CreateWebhook(c.Request.Context(), config.ID().String(), config.ProjectID().String(), config.Repo())
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "failed to create webhook: " + err.Error()})
		return
	}

	// 更新配置状态和 webhook URL
	config.SetEnabled(true)
	config.SetWebhookURL(webhookURL)
	if err := h.webhookService.SaveConfig(c.Request.Context(), config); err != nil {
		log.Printf("[WEBHOOK] failed to save config: %v", err)
	}

	c.JSON(http.StatusOK, configToMap(config))
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
	if err := h.webhookGitHub.DeleteWebhook(c.Request.Context(), config.ID().String(), config.ProjectID().String(), config.Repo()); err != nil {
		log.Printf("[WEBHOOK] failed to delete webhook: %v", err)
	}

	// 更新配置状态，清除 webhook URL
	config.SetEnabled(false)
	config.SetWebhookURL("")
	if err := h.webhookService.SaveConfig(c.Request.Context(), config); err != nil {
		log.Printf("[WEBHOOK] failed to save config: %v", err)
	}

	c.JSON(http.StatusOK, configToMap(config))
}

// GetWebhookStatus 获取 webhook 状态
func (h *GitHubWebhookHandler) GetWebhookStatus(c *gin.Context) {
	id := c.Param("id")
	config, err := h.webhookService.GetConfig(c.Request.Context(), id)
	if err != nil || config == nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "config not found"})
		return
	}

	// 检查 GitHub 上 webhook 是否存在
	exists, _ := h.webhookGitHub.CheckWebhookExists(c.Request.Context(), config.Repo())
	c.JSON(http.StatusOK, gin.H{
		"enabled":     config.Enabled(),
		"webhook_url": config.WebhookURL(),
		"exists":      exists,
	})
}

// CheckWebhookURL 检查 webhook URL 是否需要更新
func (h *GitHubWebhookHandler) CheckWebhookURL(c *gin.Context) {
	id := c.Param("id")
	config, err := h.webhookService.GetConfig(c.Request.Context(), id)
	if err != nil || config == nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "config not found"})
		return
	}

	if !config.Enabled() {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "webhook is not enabled"})
		return
	}

	needsUpdate, currentURL, err := h.webhookGitHub.CheckAndUpdateWebhook(c.Request.Context(), config.Repo())
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "failed to check webhook: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"needs_update": needsUpdate,
		"current_url":  currentURL,
		"expected_url": config.WebhookURL(),
	})
}

// UpdateWebhookURL 更新 webhook URL（如果需要）
func (h *GitHubWebhookHandler) UpdateWebhookURL(c *gin.Context) {
	id := c.Param("id")
	config, err := h.webhookService.GetConfig(c.Request.Context(), id)
	if err != nil || config == nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "config not found"})
		return
	}

	if !config.Enabled() {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "webhook is not enabled"})
		return
	}

	// 检查是否需要更新
	needsUpdate, currentURL, err := h.webhookGitHub.CheckAndUpdateWebhook(c.Request.Context(), config.Repo())
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "failed to check webhook: " + err.Error()})
		return
	}

	if !needsUpdate {
		c.JSON(http.StatusOK, gin.H{
			"message":   "webhook URL is up to date",
			"webhook_url": config.WebhookURL(),
		})
		return
	}

	// 需要更新，先获取 webhook ID
	repoPath := config.Repo()
	if strings.HasPrefix(repoPath, "https://github.com/") {
		repoPath = strings.TrimPrefix(repoPath, "https://github.com/")
	}
	repoPath = strings.TrimSuffix(repoPath, ".git")

	webhookID, err := h.webhookGitHub.FindExistingWebhook(repoPath)
	if err != nil || webhookID == 0 {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "webhook not found, please enable again"})
		return
	}

	// 更新 webhook URL
	newURL := config.WebhookURL()
	if err := h.webhookGitHub.UpdateWebhookURL(repoPath, webhookID, newURL); err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "failed to update webhook: " + err.Error()})
		return
	}

	// 更新数据库中的 webhook_url
	config.SetWebhookURL(newURL)
	if err := h.webhookService.SaveConfig(c.Request.Context(), config); err != nil {
		log.Printf("[WEBHOOK] failed to save config: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "webhook URL updated",
		"old_url":    currentURL,
		"webhook_url": newURL,
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

func configToMap(config *domain.GitHubWebhookConfig) map[string]interface{} {
	return map[string]interface{}{
		"id":          config.ID().String(),
		"project_id":  config.ProjectID().String(),
		"repo":        config.Repo(),
		"enabled":     config.Enabled(),
		"webhook_url": config.WebhookURL(),
		"created_at":  config.CreatedAt().UnixMilli(),
		"updated_at":  config.UpdatedAt().UnixMilli(),
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
