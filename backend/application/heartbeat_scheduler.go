package application

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/weibh/taskmanager/domain"
	channelBus "github.com/weibh/taskmanager/pkg/bus"
)

// HeartbeatScheduler 心跳调度器
type HeartbeatScheduler struct {
	cron             *cron.Cron
	heartbeatRepo    domain.HeartbeatRepository
	projectRepo      domain.ProjectRepository
	agentRepo        domain.AgentRepository
	requirementRepo  domain.RequirementRepository
	idGenerator      domain.IDGenerator
	inboundPublisher interface {
		PublishInbound(msg *channelBus.InboundMessage)
	}
	requirementDispatchService *RequirementDispatchService
	stateMachineService        *StateMachineService
	triggerService             *HeartbeatTriggerService
	rootCtx                    context.Context
	entries                    map[string]cron.EntryID
	entriesMu                  sync.RWMutex
}

// NewHeartbeatScheduler 创建心跳调度器
func NewHeartbeatScheduler(
	heartbeatRepo domain.HeartbeatRepository,
	projectRepo domain.ProjectRepository,
	agentRepo domain.AgentRepository,
	requirementRepo domain.RequirementRepository,
	idGenerator domain.IDGenerator,
	inboundPublisher interface {
		PublishInbound(msg *channelBus.InboundMessage)
	},
	requirementDispatchService *RequirementDispatchService,
	stateMachineService *StateMachineService,
) *HeartbeatScheduler {
	triggerService := NewHeartbeatTriggerService(
		heartbeatRepo,
		projectRepo,
		agentRepo,
		requirementRepo,
		idGenerator,
		inboundPublisher,
		requirementDispatchService,
		stateMachineService,
	)
	return &HeartbeatScheduler{
		cron:                       cron.New(cron.WithSeconds()),
		heartbeatRepo:              heartbeatRepo,
		projectRepo:                projectRepo,
		agentRepo:                  agentRepo,
		requirementRepo:            requirementRepo,
		idGenerator:                idGenerator,
		inboundPublisher:           inboundPublisher,
		requirementDispatchService: requirementDispatchService,
		stateMachineService:        stateMachineService,
		triggerService:             triggerService,
		entries:                    make(map[string]cron.EntryID),
	}
}

// NewHeartbeatSchedulerWithTriggerService 使用外部提供的触发服务创建调度器
func NewHeartbeatSchedulerWithTriggerService(
	heartbeatRepo domain.HeartbeatRepository,
	projectRepo domain.ProjectRepository,
	agentRepo domain.AgentRepository,
	requirementRepo domain.RequirementRepository,
	idGenerator domain.IDGenerator,
	inboundPublisher interface {
		PublishInbound(msg *channelBus.InboundMessage)
	},
	requirementDispatchService *RequirementDispatchService,
	stateMachineService *StateMachineService,
	triggerService *HeartbeatTriggerService,
) *HeartbeatScheduler {
	return &HeartbeatScheduler{
		cron:                       cron.New(cron.WithSeconds()),
		heartbeatRepo:              heartbeatRepo,
		projectRepo:                projectRepo,
		agentRepo:                  agentRepo,
		requirementRepo:            requirementRepo,
		idGenerator:                idGenerator,
		inboundPublisher:           inboundPublisher,
		requirementDispatchService: requirementDispatchService,
		stateMachineService:        stateMachineService,
		triggerService:             triggerService,
		entries:                    make(map[string]cron.EntryID),
	}
}

// Start 启动调度器
func (s *HeartbeatScheduler) Start(ctx context.Context) error {
	s.rootCtx = ctx

	// 启动时清理过期的需求（服务器被kill时可能留下）
	s.cleanupStaleRequirements(ctx)

	// 加载所有启用心跳
	heartbeats, err := s.heartbeatRepo.FindAllEnabled(ctx)
	if err != nil {
		return fmt.Errorf("failed to load heartbeats: %w", err)
	}

	log.Printf("[HEARTBEAT] found %d enabled heartbeats", len(heartbeats))
	for _, hb := range heartbeats {
		if err := s.scheduleHeartbeat(hb); err != nil {
			log.Printf("failed to schedule heartbeat %s: %v", hb.ID(), err)
		} else {
			log.Printf("heartbeat scheduled: %s (project: %s, interval: %d minutes)",
				hb.Name(), hb.ProjectID(), hb.IntervalMinutes())
		}
	}

	s.cron.Start()
	log.Printf("heartbeat scheduler started")
	return nil
}

// cleanupStaleRequirements 清理过期的需求
// 当服务器异常关闭时，可能会留下处于 in_progress|coding 状态的需求
// 这些需求的分身可能没有被正确清理，需要在启动时检查并清理
func (s *HeartbeatScheduler) cleanupStaleRequirements(ctx context.Context) {
	if s.requirementRepo == nil {
		return
	}

	requirements, err := s.requirementRepo.FindAll(ctx)
	if err != nil {
		log.Printf("[HEARTBEAT] cleanup: failed to find requirements: %v", err)
		return
	}

	now := time.Now()
	staleCount := 0
	for _, req := range requirements {
		replicaAgentCode := req.ReplicaAgentCode()
		if replicaAgentCode == "" {
			continue
		}

		// 检查分身是否存在
		replicaExists := false
		if s.agentRepo != nil {
			agent, err := s.agentRepo.FindByAgentCode(ctx, domain.NewAgentCode(replicaAgentCode))
			if err != nil {
				log.Printf("[HEARTBEAT] cleanup: failed to find replica agent %s: %v", replicaAgentCode, err)
			} else if agent != nil {
				replicaExists = true
			}
		}

		// 分身已不存在，或者超过 30 分钟未更新，直接清理
		shouldCleanup := false
		reason := ""
		if !replicaExists {
			shouldCleanup = true
			reason = "replica agent missing"
		} else if now.Sub(req.UpdatedAt()) > domain.StaleThreshold {
			shouldCleanup = true
			reason = "timeout - no update for 30+ minutes"
		}
		if !shouldCleanup {
			continue
		}

		staleCount++
		log.Printf("[HEARTBEAT] cleanup: found stale requirement %s (title: %s, reason: %s, updated: %s ago)",
			req.ID(), req.Title(), reason, now.Sub(req.UpdatedAt()).Round(time.Minute))

		// 清理分身（如果还存在）
		if replicaExists && s.agentRepo != nil {
			agent, _ := s.agentRepo.FindByAgentCode(ctx, domain.NewAgentCode(replicaAgentCode))
			if agent != nil {
				if err := s.agentRepo.Delete(ctx, agent.ID()); err != nil {
					log.Printf("[HEARTBEAT] cleanup: failed to delete replica agent %s: %v", replicaAgentCode, err)
				} else {
					log.Printf("[HEARTBEAT] cleanup: deleted replica agent %s", replicaAgentCode)
				}
			}
		}

		// 标记为失败
		req.MarkFailed("cleanup: " + reason)
		if err := s.requirementRepo.Save(ctx, req); err != nil {
			log.Printf("[HEARTBEAT] cleanup: failed to save requirement %s: %v", req.ID(), err)
		} else {
			log.Printf("[HEARTBEAT] cleanup: marked requirement %s as failed", req.ID())
		}
	}

	if staleCount > 0 {
		log.Printf("[HEARTBEAT] cleanup: cleaned up %d stale requirements", staleCount)
	}
}

// Stop 停止调度器
func (s *HeartbeatScheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Printf("heartbeat scheduler stopped")
}

// scheduleHeartbeat 为心跳调度任务
func (s *HeartbeatScheduler) scheduleHeartbeat(hb *domain.Heartbeat) error {
	interval := hb.IntervalMinutes()
	if interval < 1 {
		interval = 60
	}
	cronExpr := fmt.Sprintf("@every %dm", interval)

	heartbeatID := hb.ID().String()
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		s.executeHeartbeat(s.rootCtx, heartbeatID)
	})
	if err != nil {
		return err
	}
	s.entriesMu.Lock()
	s.entries[heartbeatID] = entryID
	s.entriesMu.Unlock()
	return nil
}

// RefreshSchedule 刷新单条心跳的调度
func (s *HeartbeatScheduler) RefreshSchedule(ctx context.Context, heartbeatID string) error {
	s.entriesMu.RLock()
	entryID, exists := s.entries[heartbeatID]
	s.entriesMu.RUnlock()
	if exists {
		s.cron.Remove(entryID)
		s.entriesMu.Lock()
		delete(s.entries, heartbeatID)
		s.entriesMu.Unlock()
	}

	hb, err := s.heartbeatRepo.FindByID(ctx, domain.NewHeartbeatID(heartbeatID))
	if err != nil {
		return fmt.Errorf("failed to find heartbeat %s: %w", heartbeatID, err)
	}
	if hb != nil && hb.Enabled() {
		return s.scheduleHeartbeat(hb)
	}
	return nil
}

// executeHeartbeat 执行心跳
func (s *HeartbeatScheduler) executeHeartbeat(ctx context.Context, heartbeatID string) {
	if _, err := s.triggerService.TriggerWithSource(ctx, heartbeatID, HeartbeatTriggerSourceScheduler); err != nil {
		log.Printf("heartbeat: failed to execute heartbeat %s: %v", heartbeatID, err)
	}
}

// TriggerService 获取心跳触发服务，供外部调用
func (s *HeartbeatScheduler) TriggerService() *HeartbeatTriggerService {
	return s.triggerService
}

// renderTemplate 渲染模板变量（保留兼容）
func (s *HeartbeatScheduler) renderTemplate(template string, project *domain.Project) string {
	result := template
	result = strings.ReplaceAll(result, "${project.id}", project.ID().String())
	result = strings.ReplaceAll(result, "${project.name}", project.Name())
	result = strings.ReplaceAll(result, "${project.git_repo_url}", project.GitRepoURL())
	result = strings.ReplaceAll(result, "${project.default_branch}", project.DefaultBranch())
	result = strings.ReplaceAll(result, "${timestamp}", time.Now().Format("2006-01-02 15:04:05"))
	return result
}
