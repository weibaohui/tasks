package application

import (
	"log"
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/domain/statemachine"
	channelBus "github.com/weibh/taskmanager/pkg/bus"
)

var (
	ErrBaseAgentNotFound             = errors.New("base agent not found")
	ErrInboundPublisherNotConfigured = errors.New("inbound publisher is not configured")
	ErrInvalidSessionKey             = errors.New("invalid session key")
)

type DispatchRequirementCommand struct {
	RequirementID domain.RequirementID
	AgentCode     string
	ChannelCode   string
	SessionKey    string
}

type DispatchRequirementResult struct {
	RequirementID    string `json:"requirement_id"`
	Status           string `json:"status"`
	WorkspacePath    string `json:"workspace_path"`
	ReplicaAgentCode string `json:"replica_agent_code"`
	TaskID           string `json:"task_id"`
}

type RequirementDispatchService struct {
	requirementRepo   domain.RequirementRepository
	projectRepo       domain.ProjectRepository
	agentRepo         domain.AgentRepository
	stateMachineRepo  statemachine.Repository
	workspaceConfig   domain.WorkspaceConfigProvider
	workspaceManager  domain.WorkspaceManager
	sessionService    *SessionApplicationService
	idGenerator       domain.IDGenerator
	inboundPublisher  interface {
		PublishInbound(msg *channelBus.InboundMessage)
	}
	replicaCleanupSvc domain.ReplicaCleanupService
}

func NewRequirementDispatchService(
	requirementRepo domain.RequirementRepository,
	projectRepo domain.ProjectRepository,
	agentRepo domain.AgentRepository,
	sessionService *SessionApplicationService,
	idGenerator domain.IDGenerator,
	replicaCleanupSvc domain.ReplicaCleanupService,
	stateMachineRepo statemachine.Repository,
	workspaceConfig domain.WorkspaceConfigProvider,
	workspaceManager domain.WorkspaceManager,
	inboundPublisher interface {
		PublishInbound(msg *channelBus.InboundMessage)
	},
) *RequirementDispatchService {
	return &RequirementDispatchService{
		requirementRepo:   requirementRepo,
		projectRepo:       projectRepo,
		agentRepo:         agentRepo,
		sessionService:    sessionService,
		idGenerator:       idGenerator,
		replicaCleanupSvc: replicaCleanupSvc,
		stateMachineRepo:  stateMachineRepo,
		workspaceConfig:   workspaceConfig,
		workspaceManager:  workspaceManager,
		inboundPublisher:  inboundPublisher,
	}
}

func (s *RequirementDispatchService) DispatchRequirement(ctx context.Context, cmd DispatchRequirementCommand) (*DispatchRequirementResult, error) {
	requirement, err := s.requirementRepo.FindByID(ctx, cmd.RequirementID)
	if err != nil {
		return nil, err
	}
	if requirement == nil {
		return nil, ErrRequirementNotFound
	}

	project, err := s.projectRepo.FindByID(ctx, requirement.ProjectID())
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, ErrProjectNotFound
	}
	baseAgent, err := s.agentRepo.FindByAgentCode(ctx, domain.NewAgentCode(cmd.AgentCode))
	if err != nil {
		return nil, err
	}
	if baseAgent == nil {
		return nil, ErrBaseAgentNotFound
	}

	// 设置分配信息和 session key
	requirement.SetDispatchSessionKey(cmd.SessionKey)
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}

	workspacePath := filepath.Join(s.requirementWorkspaceRoot(requirement), requirement.ProjectID().String(), requirement.ID().String())
	if err := s.workspaceManager.CreateWorkspace(workspacePath); err != nil {
		return nil, err
	}
	replicaAgent, err := s.createReplicaAgent(ctx, baseAgent, requirement, workspacePath)
	if err != nil {
		requirement.SetWorkspacePath(workspacePath)
		requirement.SetClaudeRuntimeError(err.Error())
		if errSave := s.requirementRepo.Save(ctx, requirement); errSave != nil {
			log.Printf("requirementRepo.Save failed: %v", errSave)
		}
		_ = s.workspaceManager.RemoveWorkspace(workspacePath)
		return nil, err
	}

	// 设置工作空间信息和分身 Agent code
	requirement.SetWorkspacePath(workspacePath)
	requirement.SetReplicaAgentCode(replicaAgent.AgentCode().String())
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}

	channelType, chatID, err := parseSessionKey(cmd.SessionKey)
	if err != nil {
		requirement.SetClaudeRuntimeError(err.Error())
		if errSave := s.requirementRepo.Save(ctx, requirement); errSave != nil {
			log.Printf("requirementRepo.Save failed: %v", errSave)
		}
		_ = s.workspaceManager.RemoveWorkspace(workspacePath)
		return nil, err
	}
	if s.inboundPublisher == nil {
		requirement.SetClaudeRuntimeError(ErrInboundPublisherNotConfigured.Error())
		if errSave := s.requirementRepo.Save(ctx, requirement); errSave != nil {
			log.Printf("requirementRepo.Save failed: %v", errSave)
		}
		_ = s.workspaceManager.RemoveWorkspace(workspacePath)
		return nil, ErrInboundPublisherNotConfigured
	}
	if err := s.ensureDispatchSession(ctx, cmd, replicaAgent, requirement, project); err != nil {
		requirement.SetClaudeRuntimeError(err.Error())
		if errSave := s.requirementRepo.Save(ctx, requirement); errSave != nil {
			log.Printf("requirementRepo.Save failed: %v", errSave)
		}
		_ = s.workspaceManager.RemoveWorkspace(workspacePath)
		return nil, err
	}

	// 获取状态机信息
	stateMachineName := s.getProjectStateMachineName(ctx, project.ID().String(), requirement.RequirementType())

	// 获取当前状态机状态和 AI Guide
	currentState, aiGuide := s.getStateMachineGuide(ctx, project.ID().String(), requirement.ID().String(), requirement.RequirementType())

	// 获取完整的状态机配置（用于注入触发器表）
	smConfig := s.getStateMachineConfig(ctx, project.ID().String(), requirement.RequirementType())

	// 记录状态转换日志
	if s.stateMachineRepo != nil && currentState != "" {
		fromStatus := string(requirement.Status())
		logEntry := statemachine.NewTransitionLog(
			requirement.ID().String(),
			fromStatus,
			currentState,
			"dispatch",
			"system",
			"派发需求",
		)
		if errSave := s.stateMachineRepo.SaveTransitionLog(ctx, logEntry); errSave != nil {
			log.Printf("stateMachineRepo.SaveTransitionLog failed: %v", errSave)
		}

		// 保存/更新 RequirementState
		s.saveRequirementState(ctx, requirement, currentState)

		// 自动执行 todo → 第一个处理中状态转换（系统自动完成，不需要 AI 介入）
		if currentState == "todo" && smConfig != nil {
			autoTransition := smConfig.FindFirstTransitionFrom("todo")
			if autoTransition != nil {
				nextState := smConfig.GetState(autoTransition.ToState)
				if nextState != nil {
					reqState, _ := s.stateMachineRepo.GetRequirementState(ctx, requirement.ID().String())
					if reqState != nil {
						reqState.Transition(nextState.ID, nextState.Name)
						if errSave := s.stateMachineRepo.UpdateRequirementState(ctx, reqState); errSave != nil {
							log.Printf("stateMachineRepo.UpdateRequirementState failed: %v", errSave)
						}

						autoLog := statemachine.NewTransitionLog(
							requirement.ID().String(), "todo", nextState.ID,
							autoTransition.Trigger, "system", "派发时自动状态转换")
						if errSave := s.stateMachineRepo.SaveTransitionLog(ctx, autoLog); errSave != nil {
							log.Printf("stateMachineRepo.SaveTransitionLog failed: %v", errSave)
						}

						requirement.SyncStatusFromStateMachine(nextState.ID)
						if errSave := s.requirementRepo.Save(ctx, requirement); errSave != nil {
							log.Printf("requirementRepo.Save failed: %v", errSave)
						}
						currentState = nextState.ID
						aiGuide = smConfig.GetStateAIGuide(nextState.ID)
					}
				}
			}
		}
	}

	// 使用状态机的当前状态（可能已经初始化为 todo 或其他状态）
	requirement.SyncStatusFromStateMachine(currentState)

	dispatchPrompt := buildRequirementDispatchPrompt(requirement, project, workspacePath, stateMachineName, currentState, aiGuide, smConfig)

	// 保存 Claude Runtime 执行提示词
	requirement.SetClaudeRuntimePrompt(dispatchPrompt)
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}

	// 构建元数据，包含环境变量供hook使用
	reqMetadata := map[string]any{
		"agent_code":         replicaAgent.AgentCode().String(),
		"user_code":          replicaAgent.UserCode(),
		"channel_code":       cmd.ChannelCode,
		"requirement_id":     requirement.ID().String(),
		"project_id":         project.ID().String(),
		"dispatch_source":    "requirement",
		"state_machine_name": stateMachineName,
		"requirement_type":   string(requirement.RequirementType()),
		"requirement_status": string(requirement.Status()),
		"requirement_title":  requirement.Title(),
	}

	s.inboundPublisher.PublishInbound(&channelBus.InboundMessage{
		Channel:   channelType,
		SenderID:  "requirement_dispatch",
		ChatID:    chatID,
		Content:   dispatchPrompt,
		Timestamp: time.Now(),
		Media:     []string{},
		Metadata:  reqMetadata,
	})
	dispatchID := "dispatch_" + s.idGenerator.Generate()
	return &DispatchRequirementResult{
		RequirementID:    requirement.ID().String(),
		Status:           string(requirement.Status()),
		WorkspacePath:    requirement.WorkspacePath(),
		ReplicaAgentCode: requirement.ReplicaAgentCode(),
		TaskID:           dispatchID,
	}, nil
}
