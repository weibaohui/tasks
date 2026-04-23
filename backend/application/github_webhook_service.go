package application

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/weibh/taskmanager/domain"
)

// GitHubWebhookService 处理 GitHub webhook 事件
type GitHubWebhookService struct {
	configRepo              domain.GitHubWebhookConfigRepository
	eventLogRepo            domain.WebhookEventLogRepository
	bindingRepo             domain.WebhookHeartbeatBindingRepository
	heartbeatRepo           domain.HeartbeatRepository
	triggeredHeartbeatRepo  domain.WebhookEventTriggeredHeartbeatRepository
	triggerService          *HeartbeatTriggerService
	idGenerator             domain.IDGenerator
}

// NewGitHubWebhookService 创建 GitHub webhook 服务
func NewGitHubWebhookService(
	configRepo domain.GitHubWebhookConfigRepository,
	eventLogRepo domain.WebhookEventLogRepository,
	bindingRepo domain.WebhookHeartbeatBindingRepository,
	heartbeatRepo domain.HeartbeatRepository,
	triggeredHeartbeatRepo domain.WebhookEventTriggeredHeartbeatRepository,
	triggerService *HeartbeatTriggerService,
	idGenerator domain.IDGenerator,
) *GitHubWebhookService {
	return &GitHubWebhookService{
		configRepo:             configRepo,
		eventLogRepo:           eventLogRepo,
		bindingRepo:            bindingRepo,
		heartbeatRepo:          heartbeatRepo,
		triggeredHeartbeatRepo:  triggeredHeartbeatRepo,
		triggerService:          triggerService,
		idGenerator:            idGenerator,
	}
}

// HandleWebhookEvent 处理收到的 webhook 事件
func (s *GitHubWebhookService) HandleWebhookEvent(ctx context.Context, configID, projectID, eventType, method, headers, payload string) error {
	log.Printf("[WEBHOOK] received event %s for project %s", eventType, projectID)

	// 1. 创建事件日志
	eventLog, err := domain.NewWebhookEventLog(
		domain.NewWebhookEventLogID(s.idGenerator.Generate()),
		domain.NewProjectID(projectID),
		eventType,
		method,
		headers,
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

	// 3. 触发每个绑定的心跳，并记录触发的心跳
	hasSuccess := false
	var capturedHeartbeatID string
	var capturedRequirementID string

	for _, binding := range bindings {
		if !binding.Enabled() {
			continue
		}
		heartbeatID := binding.HeartbeatID().String()
		delayMinutes := binding.DelayMinutes()
		log.Printf("[WEBHOOK] triggering heartbeat %s for event %s (delay: %d min)", heartbeatID, eventType, delayMinutes)

		trigger := func() {
			bgCtx := context.Background()
			requirement, err := s.triggerService.TriggerWithSource(bgCtx, heartbeatID, HeartbeatTriggerSourceWebhook)
			if err != nil {
				log.Printf("[WEBHOOK] failed to trigger heartbeat %s: %v", heartbeatID, err)
				return
			}

			requirementID := ""
			if requirement != nil {
				requirementID = requirement.ID().String()
			}

			// 记录第一个成功的触发（用于事件日志的 trigger_heartbeat_id 字段）
			if capturedHeartbeatID == "" {
				capturedHeartbeatID = heartbeatID
				capturedRequirementID = requirementID
			}

			triggered, err := domain.NewWebhookEventTriggeredHeartbeat(
				domain.NewWebhookEventTriggeredHeartbeatID(s.idGenerator.Generate()),
				eventLog.ID(),
				binding.HeartbeatID(),
				requirementID,
			)
			if err != nil {
				log.Printf("[WEBHOOK] failed to create triggered heartbeat record: %v", err)
			} else if err := s.triggeredHeartbeatRepo.Save(bgCtx, triggered); err != nil {
				log.Printf("[WEBHOOK] failed to save triggered heartbeat: %v", err)
			}
		}

		if delayMinutes > 0 {
			log.Printf("[WEBHOOK] heartbeat %s will trigger after %d minutes delay", heartbeatID, delayMinutes)
			go func(delay int) {
				time.Sleep(time.Duration(delay) * time.Minute)
				trigger()
			}(delayMinutes)
			// 延迟触发已成功调度
			hasSuccess = true
		} else {
			trigger()
			hasSuccess = true
		}
	}

	// 4. 更新事件日志状态
	if hasSuccess {
		if capturedHeartbeatID != "" {
			eventLog.SetProcessed(capturedHeartbeatID, capturedRequirementID)
		} else {
			eventLog.SetStatus(domain.WebhookEventStatusProcessed)
		}
	} else {
		eventLog.SetFailed("all heartbeat triggers failed")
	}
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

// ListEventLogsWithTriggeredHeartbeats 列出项目的事件日志（包含触发的心跳列表）
func (s *GitHubWebhookService) ListEventLogsWithTriggeredHeartbeats(ctx context.Context, projectID string, limit, offset int) ([]*domain.WebhookEventLog, []*domain.WebhookEventTriggeredHeartbeat, int, error) {
	logs, count, err := s.ListEventLogs(ctx, projectID, limit, offset)
	if err != nil {
		return nil, nil, 0, err
	}

	// 收集所有事件日志 ID
	eventLogIDs := make([]domain.WebhookEventLogID, len(logs))
	for i, log := range logs {
		eventLogIDs[i] = log.ID()
	}

	// 获取所有触发的心跳记录
	allTriggered := make([]*domain.WebhookEventTriggeredHeartbeat, 0)
	triggeredByEventLog := make(map[domain.WebhookEventLogID][]*domain.WebhookEventTriggeredHeartbeat)

	for _, eventLogID := range eventLogIDs {
		triggered, err := s.triggeredHeartbeatRepo.FindByEventLogID(ctx, eventLogID)
		if err != nil {
			log.Printf("[WEBHOOK] failed to find triggered heartbeats for event %s: %v", eventLogID.String(), err)
			continue
		}
		triggeredByEventLog[eventLogID] = triggered
		allTriggered = append(allTriggered, triggered...)
	}

	return logs, allTriggered, count, nil
}

// ClearEventLogs 清空项目的事件日志
func (s *GitHubWebhookService) ClearEventLogs(ctx context.Context, projectID string) error {
	return s.eventLogRepo.DeleteByProjectID(ctx, domain.NewProjectID(projectID))
}

// CreateBinding 创建心跳绑定
func (s *GitHubWebhookService) CreateBinding(ctx context.Context, projectID, configID, eventType, heartbeatID string, delayMinutes int) (*domain.WebhookHeartbeatBinding, error) {
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
		delayMinutes,
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
