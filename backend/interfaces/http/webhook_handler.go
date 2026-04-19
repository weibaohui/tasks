package http

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type WebhookHandler struct {
	webhookService *application.GitHubWebhookService
}

func NewWebhookHandler(webhookService *application.GitHubWebhookService) *WebhookHandler {
	return &WebhookHandler{
		webhookService: webhookService,
	}
}

// HandleWebhook 处理 GitHub webhook 事件（无需认证）
func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
	// 读取原始 body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[WEBHOOK] failed to read body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}
	payload := string(body)

	// 获取事件类型
	eventType := c.GetHeader("X-GitHub-Event")
	if eventType == "" {
		eventType = "unknown"
	}

	// 解析 payload 获取 repo 信息来确定项目
	var payloadData map[string]interface{}
	if err := json.Unmarshal(body, &payloadData); err != nil {
		log.Printf("[WEBHOOK] failed to parse payload: %v", err)
	}

	// 从 payload 中提取 repository full_name
	var repo string
	if repoData, ok := payloadData["repository"].(map[string]interface{}); ok {
		if fullName, ok := repoData["full_name"].(string); ok {
			repo = fullName
		}
	}

	log.Printf("[WEBHOOK] received event %s from repo %s", eventType, repo)

	// 查找该 repo 对应的项目配置
	// 通过 repo 名称查找对应的 webhook 配置
	configs, err := h.webhookService.ListConfigs(c.Request.Context())
	if err != nil {
		log.Printf("[WEBHOOK] failed to list configs: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"status": "received",
			"event":  eventType,
			"repo":   repo,
		})
		return
	}

	// 提取 owner/repo 格式（支持完整 URL 或简短格式）
	repoPath := repo
	if strings.HasPrefix(repo, "https://github.com/") {
		repoPath = strings.TrimPrefix(repo, "https://github.com/")
	}
	repoPath = strings.TrimSuffix(repoPath, ".git")

	// 查找匹配的 repo 配置
	var matchedConfig *domain.GitHubWebhookConfig
	for _, config := range configs {
		configRepo := config.Repo()
		// 支持完整 URL 或简短格式的匹配
		if (configRepo == repoPath || configRepo == repo || configRepo == "https://github.com/"+repoPath) && config.Enabled() {
			matchedConfig = config
			break
		}
	}

	if matchedConfig == nil {
		log.Printf("[WEBHOOK] no enabled config found for repo %s", repo)
		c.JSON(http.StatusOK, gin.H{
			"status": "received",
			"event":  eventType,
			"repo":   repo,
		})
		return
	}

	// 处理事件
	if err := h.webhookService.HandleWebhookEvent(c.Request.Context(), matchedConfig.ID().String(), matchedConfig.ProjectID().String(), eventType, payload); err != nil {
		log.Printf("[WEBHOOK] failed to handle event: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "processed",
		"event":      eventType,
		"repo":       repo,
		"project_id": matchedConfig.ProjectID().String(),
	})
}

// HandleWebhookByRepo 处理指定 repo 的 webhook（使用语义化 URL）
func (h *WebhookHandler) HandleWebhookByRepo(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	if owner == "" || repo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "owner and repo are required"})
		return
	}
	repoName := owner + "/" + repo

	// 读取原始 body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[WEBHOOK] failed to read body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}
	payload := string(body)

	// 获取事件类型
	eventType := c.GetHeader("X-GitHub-Event")
	if eventType == "" {
		eventType = "unknown"
	}

	log.Printf("[WEBHOOK] received event %s for repo %s", eventType, repoName)

	// 通过 repo 名称查找配置
	configs, err := h.webhookService.ListConfigs(c.Request.Context())
	if err != nil {
		log.Printf("[WEBHOOK] failed to list configs: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"status": "received",
			"event":  eventType,
			"repo":   repoName,
		})
		return
	}

	// 查找匹配的 repo 配置
	var matchedConfig *domain.GitHubWebhookConfig
	for _, config := range configs {
		configRepo := application.NormalizeRepo(config.Repo())
		// 精确匹配 owner/repo 格式
		if configRepo == repoName {
			matchedConfig = config
			break
		}
	}

	if matchedConfig == nil {
		log.Printf("[WEBHOOK] no config found for repo %s", repoName)
		c.JSON(http.StatusOK, gin.H{
			"status":  "received",
			"event":   eventType,
			"repo":    repoName,
			"message": "config not found",
		})
		return
	}

	if !matchedConfig.Enabled() {
		log.Printf("[WEBHOOK] config for repo %s is disabled", repoName)
		c.JSON(http.StatusOK, gin.H{
			"status":  "received",
			"event":   eventType,
			"repo":    repoName,
			"message": "config disabled",
		})
		return
	}

	// 处理事件
	if err := h.webhookService.HandleWebhookEvent(c.Request.Context(), matchedConfig.ID().String(), matchedConfig.ProjectID().String(), eventType, payload); err != nil {
		log.Printf("[WEBHOOK] failed to handle event: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "processed",
		"event":      eventType,
		"repo":       repoName,
		"project_id": matchedConfig.ProjectID().String(),
	})
}
