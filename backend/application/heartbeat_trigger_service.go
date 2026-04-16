package application

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/domain/statemachine"
	channelBus "github.com/weibh/taskmanager/pkg/bus"
)

// HeartbeatTriggerService 心跳触发服务
// 负责执行单次心跳的完整流程：创建需求、初始化状态机、派发需求
type HeartbeatTriggerService struct {
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
}

// NewHeartbeatTriggerService 创建心跳触发服务
func NewHeartbeatTriggerService(
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
) *HeartbeatTriggerService {
	return &HeartbeatTriggerService{
		heartbeatRepo:              heartbeatRepo,
		projectRepo:                projectRepo,
		agentRepo:                  agentRepo,
		requirementRepo:            requirementRepo,
		idGenerator:                idGenerator,
		inboundPublisher:           inboundPublisher,
		requirementDispatchService: requirementDispatchService,
		stateMachineService:        stateMachineService,
	}
}

// Trigger 触发指定心跳的执行
func (s *HeartbeatTriggerService) Trigger(ctx context.Context, heartbeatID string) error {
	hb, err := s.heartbeatRepo.FindByID(ctx, domain.NewHeartbeatID(heartbeatID))
	if err != nil || hb == nil {
		return fmt.Errorf("failed to find heartbeat %s: %w", heartbeatID, err)
	}
	if !hb.Enabled() {
		return fmt.Errorf("heartbeat %s is disabled", heartbeatID)
	}

	project, err := s.projectRepo.FindByID(ctx, hb.ProjectID())
	if err != nil || project == nil {
		return fmt.Errorf("failed to find project for heartbeat %s: %w", heartbeatID, err)
	}

	log.Printf("[HEARTBEAT] executing heartbeat %s for project %s", hb.Name(), project.Name())

	// 解析 Agent：优先使用心跳指定的 agent，若不存在则回退到项目默认 agent
	agentCode := hb.AgentCode()
	if agentCode != "" && s.agentRepo != nil {
		baseAgent, _ := s.agentRepo.FindByAgentCode(ctx, domain.NewAgentCode(agentCode))
		if baseAgent == nil {
			fallback := project.DefaultAgentCode()
			if fallback != "" {
				log.Printf("[HEARTBEAT] agent %s not found for heartbeat %s, falling back to project default %s", agentCode, hb.Name(), fallback)
				agentCode = fallback
			}
		}
	}

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
		return fmt.Errorf("failed to create requirement: %w", err)
	}

	// 设置需求类型
	requirement.SetRequirementType(domain.RequirementType(reqType))

	// 保存需求
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return fmt.Errorf("failed to save requirement: %w", err)
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
			return fmt.Errorf("project has no dispatch channel or session key configured")
		}
		result, err := s.requirementDispatchService.DispatchRequirement(ctx, DispatchRequirementCommand{
			RequirementID: requirement.ID(),
			AgentCode:     agentCode,
			ChannelCode:   channelCode,
			SessionKey:    sessionKey,
		})
		if err != nil {
			return fmt.Errorf("failed to dispatch requirement %s: %w", requirement.ID(), err)
		}
		log.Printf("[HEARTBEAT] dispatched requirement %s, task_id: %s, workspace: %s", requirement.ID(), result.TaskID, result.WorkspacePath)
	} else {
		log.Printf("heartbeat: requirementDispatchService not available, requirement %s created but not dispatched", requirement.ID())
		return fmt.Errorf("requirement dispatch service not available")
	}

	return nil
}
