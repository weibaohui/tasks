package application

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/domain/statemachine"
	channelBus "github.com/weibh/taskmanager/pkg/bus"
)

// HeartbeatScheduler 心跳调度器
type HeartbeatScheduler struct {
	cron                       *cron.Cron
	heartbeatRepo              domain.HeartbeatRepository
	projectRepo                domain.ProjectRepository
	agentRepo                  domain.AgentRepository
	requirementRepo            domain.RequirementRepository
	idGenerator                domain.IDGenerator
	inboundPublisher           interface {
		PublishInbound(msg *channelBus.InboundMessage)
	}
	requirementDispatchService *RequirementDispatchService
	stateMachineService        *StateMachineService
	rootCtx                    context.Context
	entries                    map[string]cron.EntryID
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
		// 检查分身是否存在（用于 IsStaleWithReplicaCheck）
		replicaExists := true
		if req.ReplicaAgentCode() != "" && s.agentRepo != nil {
			agent, err := s.agentRepo.FindByAgentCode(ctx, domain.NewAgentCode(req.ReplicaAgentCode()))
			if err == nil && agent == nil {
				replicaExists = false
			}
		}

		// 使用领域方法判断是否过期
		shouldCleanup, reason := req.IsStaleWithReplicaCheck(now, replicaExists)
		if !shouldCleanup {
			continue
		}

		staleCount++
		log.Printf("[HEARTBEAT] cleanup: found stale requirement %s (title: %s, reason: %s, updated: %s ago)",
			req.ID(), req.Title(), reason, now.Sub(req.UpdatedAt()).Round(time.Minute))

		// 清理分身（如果还存在）
		if replicaAgentCode := req.ReplicaAgentCode(); replicaAgentCode != "" {
			if s.agentRepo != nil {
				agent, err := s.agentRepo.FindByAgentCode(ctx, domain.NewAgentCode(replicaAgentCode))
				if err == nil && agent != nil {
					if err := s.agentRepo.Delete(ctx, agent.ID()); err != nil {
						log.Printf("[HEARTBEAT] cleanup: failed to delete replica agent %s: %v", replicaAgentCode, err)
					} else {
						log.Printf("[HEARTBEAT] cleanup: deleted replica agent %s", replicaAgentCode)
					}
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
	cronExpr := fmt.Sprintf("0 */%d * * * *", interval)

	heartbeatID := hb.ID().String()
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		s.executeHeartbeat(s.rootCtx, heartbeatID)
	})
	if err != nil {
		return err
	}
	s.entries[heartbeatID] = entryID
	return nil
}

// RefreshSchedule 刷新单条心跳的调度
func (s *HeartbeatScheduler) RefreshSchedule(ctx context.Context, heartbeatID string) error {
	if entryID, exists := s.entries[heartbeatID]; exists {
		s.cron.Remove(entryID)
		delete(s.entries, heartbeatID)
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
	hb, err := s.heartbeatRepo.FindByID(ctx, domain.NewHeartbeatID(heartbeatID))
	if err != nil || hb == nil {
		log.Printf("heartbeat: failed to find heartbeat %s: %v", heartbeatID, err)
		return
	}
	if !hb.Enabled() {
		return
	}

	project, err := s.projectRepo.FindByID(ctx, hb.ProjectID())
	if err != nil || project == nil {
		log.Printf("heartbeat: failed to find project for heartbeat %s: %v", heartbeatID, err)
		return
	}

	log.Printf("[HEARTBEAT] executing heartbeat %s for project %s", hb.Name(), project.Name())

	// 替换模板变量
	prompt := hb.RenderPrompt(project)

	// 确定需求类型
	reqType := hb.RequirementType()
	if reqType == "" {
		reqType = string(domain.RequirementTypeHeartbeat)
	}

	// 创建心跳需求
	requirement, err := domain.NewRequirement(
		domain.NewRequirementID(s.idGenerator.Generate()),
		project.ID(),
		fmt.Sprintf("[心跳] %s - %s", hb.Name(), time.Now().Format("2006-01-02 15:04")),
		prompt,
		"心跳自动生成",
		"",
	)
	if err != nil {
		log.Printf("heartbeat: failed to create requirement: %v", err)
		return
	}

	// 设置需求类型
	requirement.SetRequirementType(domain.RequirementType(reqType))

	// 保存需求
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		log.Printf("heartbeat: failed to save requirement: %v", err)
		return
	}

	// 初始化状态机状态（如果项目绑定了状态机）
	if s.stateMachineService != nil {
		psm, err := s.stateMachineService.GetProjectStateMachine(ctx, project.ID().String(), statemachine.RequirementType(reqType))
		if err == nil && psm != nil {
			rs, err := s.stateMachineService.InitializeRequirementState(ctx, requirement.ID().String(), psm.StateMachineID())
			if err != nil {
				log.Printf("[HEARTBEAT] failed to initialize requirement state: %v", err)
			} else {
				log.Printf("[HEARTBEAT] initialized requirement state for %s (state: %s)", requirement.ID(), rs.CurrentState)
			}
		}
	}

	log.Printf("[HEARTBEAT] created requirement %s for heartbeat %s, dispatching...", requirement.ID(), hb.Name())

	// 直接派发心跳需求
	if s.requirementDispatchService != nil {
		channelCode := project.DispatchChannelCode()
		sessionKey := project.DispatchSessionKey()
		if channelCode == "" || sessionKey == "" {
			log.Printf("heartbeat: project %s has no dispatch channel or session key configured", project.Name())
			return
		}
		result, err := s.requirementDispatchService.DispatchRequirement(ctx, DispatchRequirementCommand{
			RequirementID: requirement.ID(),
			AgentCode:     hb.AgentCode(),
			ChannelCode:   channelCode,
			SessionKey:    sessionKey,
		})
		if err != nil {
			log.Printf("heartbeat: failed to dispatch requirement %s: %v", requirement.ID(), err)
			return
		}
		log.Printf("[HEARTBEAT] dispatched requirement %s, task_id: %s, workspace: %s", requirement.ID(), result.TaskID, result.WorkspacePath)
	} else {
		log.Printf("heartbeat: requirementDispatchService not available, requirement %s created but not dispatched", requirement.ID())
	}
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
