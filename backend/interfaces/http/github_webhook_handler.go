package http

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/config"
)

type GitHubWebhookHandler struct {
	webhookService *application.GitHubWebhookService
	webhookGitHub  *application.WebhookGitHubManager
	webhookAMC     *application.WebhookAMCManager
	projectRepo    domain.ProjectRepository
	authHandler    *AuthHandler
}

func NewGitHubWebhookHandler(webhookService *application.GitHubWebhookService, webhookGitHub *application.WebhookGitHubManager, webhookAMC *application.WebhookAMCManager, projectRepo domain.ProjectRepository, authHandler *AuthHandler) *GitHubWebhookHandler {
	return &GitHubWebhookHandler{
		webhookService: webhookService,
		webhookGitHub:  webhookGitHub,
		webhookAMC:     webhookAMC,
		projectRepo:    projectRepo,
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
	ProjectID    string `json:"project_id" binding:"required"`
	ConfigID     string `json:"config_id" binding:"required"`
	EventType    string `json:"event_type" binding:"required"`
	HeartbeatID  string `json:"heartbeat_id" binding:"required"`
	DelayMinutes int    `json:"delay_minutes"`
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

// EnableWebhook 启用 webhook（根据平台类型创建 GitHub 或 AMC webhook）
func (h *GitHubWebhookHandler) EnableWebhook(c *gin.Context) {
	id := c.Param("id")
	config, err := h.webhookService.GetConfig(c.Request.Context(), id)
	if err != nil || config == nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "config not found"})
		return
	}

	// 获取项目信息，判断平台类型
	project, err := h.projectRepo.FindByID(c.Request.Context(), config.ProjectID())
	if err != nil || project == nil {
		log.Printf("[WEBHOOK] EnableWebhook: project lookup failed, err=%v, project=%v, projectID=%s", err, project, config.ProjectID())
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "project not found"})
		return
	}

	platformType := project.PlatformType()
	log.Printf("[WEBHOOK] EnableWebhook: configID=%s, projectID=%s, repo=%s, gitURL=%s, platform=%s, webhookAMC=%v",
		config.ID(), config.ProjectID(), config.Repo(), project.GitRepoURL(), platformType, h.webhookAMC)
	var webhookURL string

	if platformType == domain.PlatformTypeAtomGit && h.webhookAMC != nil {
		// AtomGit 平台，使用 AMC 创建 webhook
		webhookURL, err = h.webhookAMC.CreateWebhook(c.Request.Context(), config.ID().String(), config.ProjectID().String(), config.Repo())
	} else {
		// GitHub 平台，使用 gh 创建 webhook
		log.Printf("[WEBHOOK] EnableWebhook: using GitHub path for repo=%s", config.Repo())
		webhookURL, err = h.webhookGitHub.CreateWebhook(c.Request.Context(), config.ID().String(), config.ProjectID().String(), config.Repo())
	}

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

// DisableWebhook 停用 webhook（根据平台类型删除 GitHub 或 AMC webhook）
func (h *GitHubWebhookHandler) DisableWebhook(c *gin.Context) {
	id := c.Param("id")
	config, err := h.webhookService.GetConfig(c.Request.Context(), id)
	if err != nil || config == nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "config not found"})
		return
	}

	// 获取项目信息，判断平台类型
	project, err := h.projectRepo.FindByID(c.Request.Context(), config.ProjectID())
	if err != nil || project == nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "project not found"})
		return
	}

	platformType := project.PlatformType()

	if platformType == domain.PlatformTypeAtomGit && h.webhookAMC != nil {
		// AtomGit 平台，使用 AMC 删除 webhook
		if err := h.webhookAMC.DeleteWebhook(c.Request.Context(), config.ID().String(), config.ProjectID().String(), config.Repo()); err != nil {
			log.Printf("[WEBHOOK] failed to delete AMC webhook: %v", err)
		}
	} else {
		// GitHub 平台，使用 gh 删除 webhook
		if err := h.webhookGitHub.DeleteWebhook(c.Request.Context(), config.ID().String(), config.ProjectID().String(), config.Repo()); err != nil {
			log.Printf("[WEBHOOK] failed to delete webhook: %v", err)
		}
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

	// 获取项目信息，判断平台类型
	project, err := h.projectRepo.FindByID(c.Request.Context(), config.ProjectID())
	if err != nil || project == nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "project not found"})
		return
	}

	platformType := project.PlatformType()
	var exists bool

	if platformType == domain.PlatformTypeAtomGit && h.webhookAMC != nil {
		// AtomGit 平台，使用 AMC 检查 webhook
		exists, _ = h.webhookAMC.CheckWebhookExists(c.Request.Context(), config.Repo())
	} else {
		// GitHub 平台，使用 gh 检查 webhook
		exists, _ = h.webhookGitHub.CheckWebhookExists(c.Request.Context(), config.Repo())
	}

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

	// 获取项目信息，判断平台类型
	project, err := h.projectRepo.FindByID(c.Request.Context(), config.ProjectID())
	if err != nil || project == nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "project not found"})
		return
	}

	platformType := project.PlatformType()
	var needsUpdate bool
	var currentURL string

	if platformType == domain.PlatformTypeAtomGit && h.webhookAMC != nil {
		// AtomGit 平台，使用 AMC 检查 webhook
		needsUpdate, currentURL, err = h.webhookAMC.CheckAndUpdateWebhook(c.Request.Context(), config.Repo())
	} else {
		// GitHub 平台，使用 gh 检查 webhook
		needsUpdate, currentURL, err = h.webhookGitHub.CheckAndUpdateWebhook(c.Request.Context(), config.Repo())
	}

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

	// 获取项目信息，判断平台类型
	project, err := h.projectRepo.FindByID(c.Request.Context(), config.ProjectID())
	if err != nil || project == nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "project not found"})
		return
	}

	platformType := project.PlatformType()
	repoPath := application.NormalizeRepo(config.Repo())
	var needsUpdate bool
	var currentURL string

	if platformType == domain.PlatformTypeAtomGit && h.webhookAMC != nil {
		// AtomGit 平台，使用 AMC 检查 webhook
		needsUpdate, currentURL, err = h.webhookAMC.CheckAndUpdateWebhook(c.Request.Context(), config.Repo())
	} else {
		// GitHub 平台，使用 gh 检查 webhook
		needsUpdate, currentURL, err = h.webhookGitHub.CheckAndUpdateWebhook(c.Request.Context(), config.Repo())
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "failed to check webhook: " + err.Error()})
		return
	}

	if !needsUpdate {
		c.JSON(http.StatusOK, gin.H{
			"message":     "webhook URL is up to date",
			"webhook_url": config.WebhookURL(),
		})
		return
	}

	// 需要更新，先获取 webhook ID
	var webhookID interface{}
	if platformType == domain.PlatformTypeAtomGit && h.webhookAMC != nil {
		webhookID, err = h.webhookAMC.FindExistingWebhook(repoPath)
	} else {
		webhookID, err = h.webhookGitHub.FindExistingWebhook(repoPath)
	}

	if err != nil || webhookID == nil || webhookID == "" || webhookID == 0 {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "webhook not found, please enable again"})
		return
	}

	// 更新 webhook URL
	newURL := config.WebhookURL()
	if platformType == domain.PlatformTypeAtomGit && h.webhookAMC != nil {
		err = h.webhookAMC.UpdateWebhookURL(repoPath, webhookID.(string), newURL)
	} else {
		err = h.webhookGitHub.UpdateWebhookURL(repoPath, webhookID.(int64), newURL)
	}

	if err != nil {
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
		"old_url":     currentURL,
		"webhook_url": newURL,
	})
}

// UpdateAllWebhooksIfNeeded 检查并更新所有启用的 webhook URL（供 tunnel start 调用）
func (h *GitHubWebhookHandler) UpdateAllWebhooksIfNeeded() {
	// 获取最新的 public URL
	newPublicURL := config.GetPublicURL()
	if newPublicURL == "" {
		log.Printf("[WEBHOOK] no public URL available, skipping webhook update")
		return
	}

	// 记录新的 public URL 用于排查问题
	log.Printf("[WEBHOOK] updating webhook URLs with new public URL: %s", newPublicURL)

	// 确认加载成功后再更新 manager 中的 serverURL
	h.webhookGitHub.UpdateServerURL(newPublicURL)

	configs, err := h.webhookService.ListConfigs(context.Background())
	if err != nil {
		log.Printf("[WEBHOOK] failed to list configs for URL update: %v", err)
		return
	}

	for _, config := range configs {
		if !config.Enabled() {
			continue
		}

		repoPath := application.NormalizeRepo(config.Repo())

		needsUpdate, _, err := h.webhookGitHub.CheckAndUpdateWebhook(context.Background(), repoPath)
		if err != nil || !needsUpdate {
			continue
		}

		webhookID, err := h.webhookGitHub.FindExistingWebhook(repoPath)
		if err != nil || webhookID == 0 {
			log.Printf("[WEBHOOK] webhook not found for repo %s, skipping update", repoPath)
			continue
		}

		// 使用新的 public URL 构建的 webhook URL
		newWebhookURL := h.webhookGitHub.BuildWebhookURL(repoPath)
		if err := h.webhookGitHub.UpdateWebhookURL(repoPath, webhookID, newWebhookURL); err != nil {
			log.Printf("[WEBHOOK] failed to update webhook for repo %s: %v", repoPath, err)
			continue
		}

		// 更新数据库中的 webhook_url
		config.SetWebhookURL(newWebhookURL)
		if err := h.webhookService.SaveConfig(context.Background(), config); err != nil {
			log.Printf("[WEBHOOK] failed to save config for repo %s: %v", repoPath, err)
			continue
		}

		log.Printf("[WEBHOOK] webhook URL updated for repo %s: %s", repoPath, newWebhookURL)
	}
}

// ListEventLogs 列出事件日志（分页）
func (h *GitHubWebhookHandler) ListEventLogs(c *gin.Context) {
	configID := c.Param("id")
	config, err := h.webhookService.GetConfig(c.Request.Context(), configID)
	if err != nil || config == nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "config not found"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	logs, triggered, total, err := h.webhookService.ListEventLogsWithTriggeredHeartbeats(c.Request.Context(), config.ProjectID().String(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(logs))
	for _, log := range logs {
		resp = append(resp, eventLogToMap(log, triggered))
	}
	c.JSON(http.StatusOK, gin.H{
		"data":   resp,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ClearEventLogs 清空事件日志
func (h *GitHubWebhookHandler) ClearEventLogs(c *gin.Context) {
	configID := c.Param("id")
	config, err := h.webhookService.GetConfig(c.Request.Context(), configID)
	if err != nil || config == nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "config not found"})
		return
	}
	if err := h.webhookService.ClearEventLogs(c.Request.Context(), config.ProjectID().String()); err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
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
	binding, err := h.webhookService.CreateBinding(c.Request.Context(), req.ProjectID, req.ConfigID, req.EventType, req.HeartbeatID, req.DelayMinutes)
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

func eventLogToMap(log *domain.WebhookEventLog, allTriggered []*domain.WebhookEventTriggeredHeartbeat) map[string]interface{} {
	// 筛选出属于当前事件的心跳
	triggeredHeartbeats := make([]map[string]interface{}, 0)
	for _, t := range allTriggered {
		if t.WebhookEventLogID() == log.ID() {
			triggeredHeartbeats = append(triggeredHeartbeats, map[string]interface{}{
				"id":             t.ID().String(),
				"heartbeat_id":    t.HeartbeatID().String(),
				"requirement_id": t.RequirementID(),
				"triggered_at":    t.TriggeredAt().UnixMilli(),
			})
		}
	}
	return map[string]interface{}{
		"id":                      log.ID().String(),
		"project_id":              log.ProjectID().String(),
		"event_type":              log.EventType(),
		"payload":                 log.Payload(),
		"status":                  string(log.Status()),
		"trigger_heartbeat_id":    log.TriggerHeartbeatID(),
		"requirement_id":          log.RequirementID(),
		"error_message":           log.ErrorMessage(),
		"received_at":             log.ReceivedAt().UnixMilli(),
		"triggered_heartbeats":    triggeredHeartbeats,
	}
}

func webhookBindingToMap(binding *domain.WebhookHeartbeatBinding) map[string]interface{} {
	return map[string]interface{}{
		"id":                       binding.ID().String(),
		"project_id":               binding.ProjectID().String(),
		"github_webhook_config_id": binding.ConfigID().String(),
		"github_event_type":        binding.GitHubEventType(),
		"heartbeat_id":             binding.HeartbeatID().String(),
		"enabled":                  binding.Enabled(),
		"delay_minutes":            binding.DelayMinutes(),
		"created_at":               binding.CreatedAt().UnixMilli(),
	}
}
