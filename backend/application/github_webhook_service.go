package application

import (
	"context"
	"fmt"
	"log"

	"github.com/weibh/taskmanager/domain"
)

// GitHubWebhookService 处理 GitHub webhook 事件
type GitHubWebhookService struct {
	configRepo     domain.GitHubWebhookConfigRepository
	eventLogRepo   domain.WebhookEventLogRepository
	bindingRepo    domain.WebhookHeartbeatBindingRepository
	heartbeatRepo  domain.HeartbeatRepository
	triggerService *HeartbeatTriggerService
	idGenerator    domain.IDGenerator
}

// NewGitHubWebhookService 创建 GitHub webhook 服务
func NewGitHubWebhookService(
	configRepo domain.GitHubWebhookConfigRepository,
	eventLogRepo domain.WebhookEventLogRepository,
	bindingRepo domain.WebhookHeartbeatBindingRepository,
	heartbeatRepo domain.HeartbeatRepository,
	triggerService *HeartbeatTriggerService,
	idGenerator domain.IDGenerator,
) *GitHubWebhookService {
	return &GitHubWebhookService{
		configRepo:     configRepo,
		eventLogRepo:   eventLogRepo,
		bindingRepo:    bindingRepo,
		heartbeatRepo:  heartbeatRepo,
		triggerService: triggerService,
		idGenerator:    idGenerator,
	}
}

// HandleWebhookEvent 处理收到的 webhook 事件
func (s *GitHubWebhookService) HandleWebhookEvent(ctx context.Context, configID, projectID, eventType, payload string) error {
	log.Printf("[WEBHOOK] received event %s for project %s", eventType, projectID)

	// 1. 创建事件日志
	eventLog, err := domain.NewWebhookEventLog(
		domain.NewWebhookEventLogID(s.idGenerator.Generate()),
		domain.NewProjectID(projectID),
		eventType,
		payload,
	)
	if err != nil {
		return err
	}

	// 保存事件日志
	if err := s.eventLogRepo.Save(ctx, eventLog); err != nil {
		log.Printf("[WEBHOOK] failed to save event log: %v", err)
	}

	// 2. 根据 eventType 查找绑定的 heartbeat 并触发
	bindings, err := s.bindingRepo.FindByConfigIDAndEventType(
		ctx,
		domain.NewGitHubWebhookConfigID(configID),
		eventType,
	)
	if err != nil {
		eventLog.SetFailed(err.Error())
		s.eventLogRepo.Save(ctx, eventLog)
		return err
	}

	if len(bindings) == 0 {
		log.Printf("[WEBHOOK] no bindings found for event %s in project %s", eventType, projectID)
		eventLog.SetFailed("no bindings found")
		s.eventLogRepo.Save(ctx, eventLog)
		return nil
	}

	// 3. 触发每个绑定的心跳
	for _, binding := range bindings {
		if !binding.Enabled() {
			continue
		}
		heartbeatID := binding.HeartbeatID().String()
		log.Printf("[WEBHOOK] triggering heartbeat %s for event %s", heartbeatID, eventType)

		if err := s.triggerService.Trigger(ctx, heartbeatID); err != nil {
			log.Printf("[WEBHOOK] failed to trigger heartbeat %s: %v", heartbeatID, err)
			continue
		}
		eventLog.SetProcessed(heartbeatID)
	}

	// 更新事件日志状态
	return s.eventLogRepo.Save(ctx, eventLog)
}

// CreateConfig 创建 webhook 配置
func (s *GitHubWebhookService) CreateConfig(ctx context.Context, projectID, repo string) (*domain.GitHubWebhookConfig, error) {
	config, err := domain.NewGitHubWebhookConfig(
		domain.NewGitHubWebhookConfigID(s.idGenerator.Generate()),
		domain.NewProjectID(projectID),
		repo,
	)
	if err != nil {
		return nil, err
	}
	if err := s.configRepo.Save(ctx, config); err != nil {
		return nil, err
	}
	return config, nil
}

// GetConfig 获取 webhook 配置（通过 config ID）
func (s *GitHubWebhookService) GetConfig(ctx context.Context, configID string) (*domain.GitHubWebhookConfig, error) {
	return s.configRepo.FindByID(ctx, domain.NewGitHubWebhookConfigID(configID))
}

// UpdateConfigRepo 更新 webhook 配置的 repo
func (s *GitHubWebhookService) UpdateConfigRepo(ctx context.Context, configID, repo string) error {
	config, err := s.configRepo.FindByID(ctx, domain.NewGitHubWebhookConfigID(configID))
	if err != nil || config == nil {
		return err
	}
	if err := config.UpdateRepo(repo); err != nil {
		return err
	}
	return s.configRepo.Save(ctx, config)
}

// SetConfigEnabled 设置配置启用状态
func (s *GitHubWebhookService) SetConfigEnabled(ctx context.Context, configID string, enabled bool) error {
	config, err := s.configRepo.FindByID(ctx, domain.NewGitHubWebhookConfigID(configID))
	if err != nil || config == nil {
		return err
	}
	config.SetEnabled(enabled)
	return s.configRepo.Save(ctx, config)
}

// SaveConfig 保存 webhook 配置
func (s *GitHubWebhookService) SaveConfig(ctx context.Context, config *domain.GitHubWebhookConfig) error {
	return s.configRepo.Save(ctx, config)
}

// ListConfigs 列出所有 webhook 配置
func (s *GitHubWebhookService) ListConfigs(ctx context.Context) ([]*domain.GitHubWebhookConfig, error) {
	return s.configRepo.FindAll(ctx)
}

// DeleteConfig 删除 webhook 配置
func (s *GitHubWebhookService) DeleteConfig(ctx context.Context, configID string) error {
	return s.configRepo.Delete(ctx, domain.NewGitHubWebhookConfigID(configID))
}

// ListEventLogs 列出项目的事件日志（分页）
func (s *GitHubWebhookService) ListEventLogs(ctx context.Context, projectID string, limit, offset int) ([]*domain.WebhookEventLog, int, error) {
	logs, err := s.eventLogRepo.FindByProjectID(ctx, domain.NewProjectID(projectID), limit, offset)
	if err != nil {
		return nil, 0, err
	}
	count, err := s.eventLogRepo.CountByProjectID(ctx, domain.NewProjectID(projectID))
	if err != nil {
		return nil, 0, err
	}
	return logs, count, nil
}

// ClearEventLogs 清空项目的事件日志
func (s *GitHubWebhookService) ClearEventLogs(ctx context.Context, projectID string) error {
	return s.eventLogRepo.DeleteByProjectID(ctx, domain.NewProjectID(projectID))
}

// CreateBinding 创建心跳绑定
func (s *GitHubWebhookService) CreateBinding(ctx context.Context, projectID, configID, eventType, heartbeatID string) (*domain.WebhookHeartbeatBinding, error) {
	// 验证心跳是否存在
	hb, err := s.heartbeatRepo.FindByID(ctx, domain.NewHeartbeatID(heartbeatID))
	if err != nil {
		return nil, fmt.Errorf("failed to find heartbeat: %w", err)
	}
	if hb == nil {
		return nil, fmt.Errorf("heartbeat %s not found", heartbeatID)
	}

	binding, err := domain.NewWebhookHeartbeatBinding(
		domain.NewWebhookHeartbeatBindingID(s.idGenerator.Generate()),
		domain.NewProjectID(projectID),
		domain.NewGitHubWebhookConfigID(configID),
		eventType,
		domain.NewHeartbeatID(heartbeatID),
	)
	if err != nil {
		return nil, err
	}
	if err := s.bindingRepo.Save(ctx, binding); err != nil {
		return nil, err
	}
	return binding, nil
}

// ListBindings 列出配置的所有绑定
func (s *GitHubWebhookService) ListBindings(ctx context.Context, configID string) ([]*domain.WebhookHeartbeatBinding, error) {
	return s.bindingRepo.FindByConfigID(ctx, domain.NewGitHubWebhookConfigID(configID))
}

// DeleteBinding 删除心跳绑定
func (s *GitHubWebhookService) DeleteBinding(ctx context.Context, bindingID string) error {
	return s.bindingRepo.Delete(ctx, domain.NewWebhookHeartbeatBindingID(bindingID))
}

// ListHeartbeats 列出项目的所有心跳
func (s *GitHubWebhookService) ListHeartbeats(ctx context.Context, projectID string) ([]*domain.Heartbeat, error) {
	return s.heartbeatRepo.FindByProjectID(ctx, domain.NewProjectID(projectID))
}
