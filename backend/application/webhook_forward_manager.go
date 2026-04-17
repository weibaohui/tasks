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
)

// WebhookGitHubManager 管理 GitHub webhook 的创建和删除
type WebhookGitHubManager struct {
	mu        sync.RWMutex
	serverURL string // public URL for webhook
}

// NewWebhookGitHubManager 创建 WebhookGitHubManager
func NewWebhookGitHubManager(serverURL string) *WebhookGitHubManager {
	return &WebhookGitHubManager{
		serverURL: strings.TrimSuffix(serverURL, "/api/v1"),
	}
}

// normalizeRepo converts full URL to short format (owner/repo)
func normalizeRepo(repo string) string {
	if strings.HasPrefix(repo, "https://github.com/") {
		repo = strings.TrimPrefix(repo, "https://github.com/")
	}
	return strings.TrimSuffix(repo, ".git")
}

// BuildWebhookURL 构建 webhook URL
func (m *WebhookGitHubManager) BuildWebhookURL(repo string) string {
	repoPath := normalizeRepo(repo)
	return fmt.Sprintf("%s/api/v1/webhook/repos/%s", m.serverURL, repoPath)
}

// CreateWebhook 创建 GitHub webhook
func (m *WebhookGitHubManager) CreateWebhook(ctx context.Context, configID, projectID, repo string) (string, error) {
	repoPath := normalizeRepo(repo)
	webhookURL := m.BuildWebhookURL(repo)

	// 先检查是否已有同名 webhook
	existingID, err := m.findExistingWebhook(repoPath)
	if err != nil {
		log.Printf("[WEBHOOK] failed to check existing webhooks: %v", err)
	}

	if existingID > 0 {
		// 更新现有 webhook 的 URL
		if err := m.updateWebhookURL(repoPath, existingID, webhookURL); err != nil {
			log.Printf("[WEBHOOK] failed to update webhook: %v", err)
			return "", err
		}
		log.Printf("[WEBHOOK] updated existing webhook %d for repo %s", existingID, repoPath)
		return webhookURL, nil
	}

	// 创建新 webhook
	webhookID, err := m.createWebhook(repoPath, webhookURL)
	if err != nil {
		return "", fmt.Errorf("failed to create webhook: %w", err)
	}

	log.Printf("[WEBHOOK] created webhook %d for config %s (repo=%s, url=%s)", webhookID, configID, repoPath, webhookURL)
	return webhookURL, nil
}

// DeleteWebhook 删除 GitHub webhook
func (m *WebhookGitHubManager) DeleteWebhook(ctx context.Context, configID, projectID, repo string) error {
	repoPath := normalizeRepo(repo)

	// 查找现有的 webhook
	webhookID, err := m.findExistingWebhook(repoPath)
	if err != nil {
		return err
	}
	if webhookID == 0 {
		log.Printf("[WEBHOOK] no webhook found for repo %s", repoPath)
		return nil
	}

	// 删除 webhook
	if err := m.deleteWebhook(repoPath, webhookID); err != nil {
		log.Printf("[WEBHOOK] failed to delete webhook %d: %v", webhookID, err)
		return err
	}

	log.Printf("[WEBHOOK] deleted webhook %d for config %s", webhookID, configID)
	return nil
}

// CheckWebhookExists 检查 webhook 是否存在
func (m *WebhookGitHubManager) CheckWebhookExists(ctx context.Context, repo string) (bool, error) {
	repoPath := normalizeRepo(repo)
	webhookID, err := m.findExistingWebhook(repoPath)
	if err != nil {
		return false, err
	}
	return webhookID > 0, nil
}

// findExistingWebhook 查找是否已存在 webhook
func (m *WebhookGitHubManager) findExistingWebhook(repo string) (int64, error) {
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
func (m *WebhookGitHubManager) createWebhook(repo, url string) (int64, error) {
	payload := map[string]interface{}{
		"name":   "web",
		"active": true,
		"events": []string{"*"},
		"config": map[string]interface{}{
			"url":          url,
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
func (m *WebhookGitHubManager) updateWebhookURL(repo string, webhookID int64, newURL string) error {
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
func (m *WebhookGitHubManager) deleteWebhook(repo string, webhookID int64) error {
	cmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s/hooks/%d", repo, webhookID), "-X", "DELETE")

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	return nil
}
