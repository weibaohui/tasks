package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/weibh/taskmanager/domain"
)

// WebhookForwardManager 管理 GitHub webhook 的创建和删除
// 使用 gh api 直接操作 GitHub webhook
type WebhookForwardManager struct {
	forwarders map[string]*forwardInfo // key is configID
	mu         sync.RWMutex
	serverURL  string // public URL for webhook
}

type forwardInfo struct {
	webhookID  int64
	webhookURL string
	started    time.Time
}

func NewWebhookForwardManager(serverURL string) *WebhookForwardManager {
	return &WebhookForwardManager{
		forwarders: make(map[string]*forwardInfo),
		serverURL:  strings.TrimSuffix(serverURL, "/api/v1"),
	}
}

// normalizeRepo converts full URL to short format (owner/repo)
func normalizeRepo(repo string) string {
	if strings.HasPrefix(repo, "https://github.com/") {
		repo = strings.TrimPrefix(repo, "https://github.com/")
	}
	return strings.TrimSuffix(repo, ".git")
}

// StartForwarder 创建 GitHub webhook 并返回 webhook ID
func (m *WebhookForwardManager) StartForwarder(ctx context.Context, configID, projectID, repo string) error {
	repoPath := normalizeRepo(repo)
	// 使用语义化 URL: /api/v1/webhook/repos/{owner}/{repo}
	webhookURL := fmt.Sprintf("%s/api/v1/webhook/repos/%s", m.serverURL, repoPath)

	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已经存在
	if existing, ok := m.forwarders[configID]; ok && existing.webhookID > 0 {
		log.Printf("[WEBHOOK-FORWARD] webhook already exists for config %s (webhookID=%d)", configID, existing.webhookID)
		return nil
	}

	// 先检查是否已有同名 webhook
	existingID, err := m.findExistingWebhook(repoPath)
	if err != nil {
		log.Printf("[WEBHOOK-FORWARD] failed to check existing webhooks: %v", err)
	}

	if existingID > 0 {
		// 更新现有 webhook 的 URL
		if err := m.updateWebhookURL(repoPath, existingID, webhookURL); err != nil {
			log.Printf("[WEBHOOK-FORWARD] failed to update webhook: %v", err)
		} else {
			log.Printf("[WEBHOOK-FORWARD] updated existing webhook %d for repo %s", existingID, repoPath)
		}
		m.forwarders[configID] = &forwardInfo{
			webhookID:  existingID,
			webhookURL: webhookURL,
			started:    time.Now(),
		}
		return nil
	}

	// 创建新 webhook
	webhookID, err := m.createWebhook(repoPath, webhookURL)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	m.forwarders[configID] = &forwardInfo{
		webhookID:  webhookID,
		webhookURL: webhookURL,
		started:    time.Now(),
	}

	log.Printf("[WEBHOOK-FORWARD] created webhook %d for config %s (repo=%s, url=%s)", webhookID, configID, repoPath, webhookURL)
	return nil
}

// StopForwarder 删除 GitHub webhook
func (m *WebhookForwardManager) StopForwarder(configID, projectID, repo string) error {
	repoPath := normalizeRepo(repo)

	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.forwarders[configID]
	if !ok || info.webhookID == 0 {
		log.Printf("[WEBHOOK-FORWARD] no webhook found for config %s", configID)
		return nil
	}

	// 删除 webhook
	if err := m.deleteWebhook(repoPath, info.webhookID); err != nil {
		log.Printf("[WEBHOOK-FORWARD] failed to delete webhook %d: %v", info.webhookID, err)
		return err
	}

	log.Printf("[WEBHOOK-FORWARD] deleted webhook %d for config %s", info.webhookID, configID)
	delete(m.forwarders, configID)
	return nil
}

// GetStatus 返回 forwarder 状态
func (m *WebhookForwardManager) GetStatus(configID, projectID, repo string) (running bool, webhookURL string, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.forwarders[configID]
	if !ok || info.webhookID == 0 {
		return false, "", nil
	}
	return true, info.webhookURL, nil
}

// IsRunning 检查是否在运行
func (m *WebhookForwardManager) IsRunning(configID, projectID, repo string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.forwarders[configID]
	return ok && info.webhookID > 0
}

// RestoreForwarders 恢复所有启用的 webhook
func (m *WebhookForwardManager) RestoreForwarders(ctx context.Context, configs []*domain.GitHubWebhookConfig) {
	for _, config := range configs {
		if !config.Enabled() {
			continue
		}
		configID := config.ID().String()
		projectID := config.ProjectID().String()
		repo := config.Repo()

		log.Printf("[WEBHOOK-FORWARD] restoring webhook for config %s (repo=%s)", configID, repo)
		if err := m.StartForwarder(ctx, configID, projectID, repo); err != nil {
			log.Printf("[WEBHOOK-FORWARD] failed to restore webhook for config %s: %v", configID, err)
		}
	}
}

// findExistingWebhook 查找是否已存在 webhook
func (m *WebhookForwardManager) findExistingWebhook(repo string) (int64, error) {
	cmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s/hooks", repo), "--jq", "[.[] | select(.name == \"web\")] | .[0].id")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return 0, nil
	}

	idStr := strings.TrimSpace(out.String())
	if idStr == "" || idStr == "null" {
		return 0, nil
	}

	var id int64
	fmt.Sscanf(idStr, "%d", &id)
	return id, nil
}

// createWebhook 创建 GitHub webhook
func (m *WebhookForwardManager) createWebhook(repo, url string) (int64, error) {
	// 构建 JSON payload
	payload := map[string]interface{}{
		"name":   "web",
		"active": true,
		"events": []string{"*"},
		"config": map[string]interface{}{
			"url":         url,
			"content_type": "json",
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal payload: %w", err)
	}

	cmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s/hooks", repo), "-X", "POST", "--input", "-")
	cmd.Stdin = bytes.NewReader(payloadBytes)

	var out bytes.Buffer
	cmd.Stdout = &out

	err = cmd.Run()
	if err != nil {
		return 0, fmt.Errorf("failed to create webhook: %w", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if id, ok := response["id"].(float64); ok {
		return int64(id), nil
	}

	return 0, fmt.Errorf("webhook id not found in response")
}

// updateWebhookURL 更新 webhook 的 URL
func (m *WebhookForwardManager) updateWebhookURL(repo string, webhookID int64, newURL string) error {
	payload := map[string]interface{}{
		"config": map[string]interface{}{
			"url": newURL,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	cmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s/hooks/%d", repo, webhookID), "-X", "PATCH", "--input", "-")
	cmd.Stdin = bytes.NewReader(payloadBytes)

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}

	return nil
}

// deleteWebhook 删除 GitHub webhook
func (m *WebhookForwardManager) deleteWebhook(repo string, webhookID int64) error {
	cmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s/hooks/%d", repo, webhookID), "-X", "DELETE")

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	return nil
}