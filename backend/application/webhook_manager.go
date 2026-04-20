package application

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/platform"
)

// WebhookManager 管理 Webhook 的创建和删除（支持多平台）
type WebhookManager struct {
	mu         sync.RWMutex
	serverURL  string
	providers  map[domain.PlatformType]domain.PlatformProvider
	defaultPlatform domain.PlatformType
}

// NewWebhookManager 创建 WebhookManager
func NewWebhookManager(serverURL string) *WebhookManager {
	m := &WebhookManager{
		serverURL:  strings.TrimSuffix(serverURL, "/api/v1"),
		providers:  make(map[domain.PlatformType]domain.PlatformProvider),
		defaultPlatform: domain.PlatformTypeGitHub,
	}

	// 注册默认支持的平台
	m.providers[domain.PlatformTypeGitHub] = platform.MustNewProvider(domain.PlatformTypeGitHub)
	m.providers[domain.PlatformTypeAtomGit] = platform.MustNewProvider(domain.PlatformTypeAtomGit)

	return m
}

// UpdateServerURL 更新 server URL（tunnel 地址变更时调用）
func (m *WebhookManager) UpdateServerURL(serverURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	newURL := strings.TrimSuffix(serverURL, "/api/v1")
	if newURL == m.serverURL {
		return
	}
	log.Printf("[WEBHOOK] server URL updated: %s -> %s", m.serverURL, newURL)
	m.serverURL = newURL
}

// SnapshotServerURL 快照当前 server URL（批量更新时使用）
func (m *WebhookManager) SnapshotServerURL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.serverURL
}

// GetProvider 获取指定平台的 Provider
func (m *WebhookManager) GetProvider(platformType domain.PlatformType) (domain.PlatformProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, ok := m.providers[platformType]
	if !ok {
		return nil, fmt.Errorf("unsupported platform: %s", platformType)
	}
	return provider, nil
}

// GetDefaultProvider 获取默认平台的 Provider
func (m *WebhookManager) GetDefaultProvider() domain.PlatformProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.providers[m.defaultPlatform]
}

// SetDefaultPlatform 设置默认平台
func (m *WebhookManager) SetDefaultPlatform(platformType domain.PlatformType) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultPlatform = platformType
}

// BuildWebhookURL 构建 webhook URL
func (m *WebhookManager) BuildWebhookURL(repo string, platformType domain.PlatformType) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	provider := m.providers[platformType]
	repoPath := provider.NormalizeRepo(repo)
	return fmt.Sprintf("%s/api/v1/webhook/repos/%s", m.serverURL, repoPath)
}

// CreateWebhook 创建 webhook
func (m *WebhookManager) CreateWebhook(ctx context.Context, configID, projectID, repo string, platformType domain.PlatformType) (string, error) {
	provider, err := m.GetProvider(platformType)
	if err != nil {
		return "", err
	}

	repoPath := provider.NormalizeRepo(repo)
	webhookURL := m.BuildWebhookURL(repo, platformType)

	// 先检查是否已有 webhook
	existingID, err := provider.FindExistingWebhook(ctx, repoPath)
	if err != nil {
		log.Printf("[WEBHOOK] failed to check existing webhooks: %v", err)
	}

	if existingID > 0 {
		// 更新现有 webhook 的 URL
		if err := provider.UpdateWebhookURL(ctx, repoPath, existingID, webhookURL); err != nil {
			log.Printf("[WEBHOOK] failed to update webhook: %v", err)
			return "", err
		}
		log.Printf("[WEBHOOK] updated existing webhook %d for repo %s", existingID, repoPath)
		return webhookURL, nil
	}

	// 创建新 webhook
	webhookID, err := provider.CreateWebhook(ctx, repoPath, webhookURL)
	if err != nil {
		return "", fmt.Errorf("failed to create webhook: %w", err)
	}

	log.Printf("[WEBHOOK] created webhook %d for config %s (repo=%s, url=%s)", webhookID, configID, repoPath, webhookURL)
	return webhookURL, nil
}

// DeleteWebhook 删除 webhook
func (m *WebhookManager) DeleteWebhook(ctx context.Context, configID, projectID, repo string, platformType domain.PlatformType) error {
	provider, err := m.GetProvider(platformType)
	if err != nil {
		return err
	}

	repoPath := provider.NormalizeRepo(repo)

	// 查找现有的 webhook
	webhookID, err := provider.FindExistingWebhook(ctx, repoPath)
	if err != nil {
		return err
	}
	if webhookID == 0 {
		log.Printf("[WEBHOOK] no webhook found for repo %s", repoPath)
		return nil
	}

	// 删除 webhook
	if err := provider.DeleteWebhook(ctx, repoPath, webhookID); err != nil {
		log.Printf("[WEBHOOK] failed to delete webhook %d: %v", webhookID, err)
		return err
	}

	log.Printf("[WEBHOOK] deleted webhook %d for config %s", webhookID, configID)
	return nil
}

// CheckWebhookExists 检查 webhook 是否存在
func (m *WebhookManager) CheckWebhookExists(ctx context.Context, repo string, platformType domain.PlatformType) (bool, error) {
	provider, err := m.GetProvider(platformType)
	if err != nil {
		return false, err
	}

	repoPath := provider.NormalizeRepo(repo)
	webhookID, err := provider.FindExistingWebhook(ctx, repoPath)
	if err != nil {
		return false, err
	}
	return webhookID > 0, nil
}

// CheckAndUpdateWebhook 检查 webhook URL 是否需要更新，如果需要则更新
// 返回：(需要更新, 当前URL, 错误)
func (m *WebhookManager) CheckAndUpdateWebhook(ctx context.Context, repo string, platformType domain.PlatformType) (bool, string, error) {
	provider, err := m.GetProvider(platformType)
	if err != nil {
		return false, "", err
	}

	repoPath := provider.NormalizeRepo(repo)
	expectedURL := m.BuildWebhookURL(repo, platformType)

	webhookID, err := provider.FindExistingWebhook(ctx, repoPath)
	if err != nil {
		return false, "", err
	}
	if webhookID == 0 {
		// 没有 webhook，需要创建
		return true, "", nil
	}

	// 获取当前 webhook URL
	currentURL, err := provider.GetWebhookURL(ctx, repoPath, webhookID)
	if err != nil {
		return false, "", err
	}

	// 比较 URL 是否一致
	if currentURL != expectedURL {
		log.Printf("[WEBHOOK] webhook URL mismatch for %s: current=%s, expected=%s, will update",
			repoPath, currentURL, expectedURL)
		return true, currentURL, nil
	}

	return false, currentURL, nil
}

// UpdateWebhookURL 更新 webhook URL
func (m *WebhookManager) UpdateWebhookURL(ctx context.Context, repo string, platformType domain.PlatformType) error {
	provider, err := m.GetProvider(platformType)
	if err != nil {
		return err
	}

	repoPath := provider.NormalizeRepo(repo)
	expectedURL := m.BuildWebhookURL(repo, platformType)

	webhookID, err := provider.FindExistingWebhook(ctx, repoPath)
	if err != nil {
		return err
	}
	if webhookID == 0 {
		return fmt.Errorf("webhook not found for repo %s", repoPath)
	}

	return provider.UpdateWebhookURL(ctx, repoPath, webhookID, expectedURL)
}
