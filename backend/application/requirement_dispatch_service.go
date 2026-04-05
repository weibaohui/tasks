package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/weibh/taskmanager/domain"
	channelBus "github.com/weibh/taskmanager/pkg/bus"
)

var (
	ErrBaseAgentNotFound             = errors.New("base agent not found")
	ErrInboundPublisherNotConfigured = errors.New("inbound publisher is not configured")
	ErrInvalidSessionKey             = errors.New("invalid session key")
)

type DispatchRequirementCommand struct {
	RequirementID  domain.RequirementID
	AgentCode     string
	ChannelCode   string
	SessionKey    string
}

type DispatchRequirementResult struct {
	RequirementID   string `json:"requirement_id"`
	Status          string `json:"status"`
	WorkspacePath   string `json:"workspace_path"`
	ReplicaAgentCode string `json:"replica_agent_code"`
	TaskID          string `json:"task_id"`
}

type RequirementDispatchService struct {
	requirementRepo      domain.RequirementRepository
	projectRepo          domain.ProjectRepository
	agentRepo            domain.AgentRepository
	taskService          interface{} // TaskApplicationService - no longer used
	sessionService       *SessionApplicationService
	idGenerator          domain.IDGenerator
	inboundPublisher     interface {
		PublishInbound(msg *channelBus.InboundMessage)
	}
	replicaAgentManager *domain.ReplicaAgentManager
}

func NewRequirementDispatchService(
	requirementRepo domain.RequirementRepository,
	projectRepo domain.ProjectRepository,
	agentRepo domain.AgentRepository,
	taskService interface{}, // TaskApplicationService - no longer used
	sessionService *SessionApplicationService,
	idGenerator domain.IDGenerator,
	replicaAgentManager *domain.ReplicaAgentManager,
) *RequirementDispatchService {
	return &RequirementDispatchService{
		requirementRepo:      requirementRepo,
		projectRepo:         projectRepo,
		agentRepo:           agentRepo,
		taskService:         taskService,
		sessionService:      sessionService,
		idGenerator:         idGenerator,
		replicaAgentManager: replicaAgentManager,
	}
}

func (s *RequirementDispatchService) SetInboundPublisher(publisher interface {
	PublishInbound(msg *channelBus.InboundMessage)
}) {
	s.inboundPublisher = publisher
}

func (s *RequirementDispatchService) DispatchRequirement(ctx context.Context, cmd DispatchRequirementCommand) (*DispatchRequirementResult, error) {
	requirement, err := s.requirementRepo.FindByID(ctx, cmd.RequirementID)
	if err != nil {
		return nil, err
	}
	if requirement == nil {
		return nil, ErrRequirementNotFound
	}

	// 设置分身管理器
	requirement.SetReplicaAgentManager(s.replicaAgentManager)
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
	if err := requirement.StartDispatch(cmd.AgentCode); err != nil {
		return nil, err
	}
	requirement.SetDispatchSessionKey(cmd.SessionKey)
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}
	workspacePath := filepath.Join(requirementWorkspaceRoot(requirement), requirement.ProjectID().String(), requirement.ID().String())
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		return nil, err
	}
	replicaAgent, err := s.createReplicaAgent(ctx, baseAgent, requirement, workspacePath)
	if err != nil {
		requirement.MarkFailed(err.Error())
		_ = s.requirementRepo.Save(ctx, requirement)
		_ = os.RemoveAll(workspacePath)
		return nil, err
	}
	if err := requirement.MarkCoding(workspacePath, replicaAgent.AgentCode().String()); err != nil {
		return nil, err
	}
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}
	channelType, chatID, err := parseSessionKey(cmd.SessionKey)
	if err != nil {
		requirement.MarkFailed(err.Error())
		_ = s.requirementRepo.Save(ctx, requirement)
		_ = os.RemoveAll(workspacePath)
		return nil, err
	}
	if s.inboundPublisher == nil {
		requirement.MarkFailed(ErrInboundPublisherNotConfigured.Error())
		_ = s.requirementRepo.Save(ctx, requirement)
		_ = os.RemoveAll(workspacePath)
		return nil, ErrInboundPublisherNotConfigured
	}
	if err := s.ensureDispatchSession(ctx, cmd, replicaAgent, requirement, project); err != nil {
		requirement.MarkFailed(err.Error())
		_ = s.requirementRepo.Save(ctx, requirement)
		_ = os.RemoveAll(workspacePath)
		return nil, err
	}
	dispatchPrompt := buildRequirementDispatchPrompt(requirement, project, workspacePath)

	// 保存 Claude Runtime 执行提示词
	requirement.SetClaudeRuntimePrompt(dispatchPrompt)
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}

	s.inboundPublisher.PublishInbound(&channelBus.InboundMessage{
		Channel:   channelType,
		SenderID:  "requirement_dispatch",
		ChatID:    chatID,
		Content:   dispatchPrompt,
		Timestamp: time.Now(),
		Media:     []string{},
		Metadata: map[string]any{
			"agent_code":      replicaAgent.AgentCode().String(),
			"user_code":       replicaAgent.UserCode(),
			"channel_code":    cmd.ChannelCode,
			"requirement_id":  requirement.ID().String(),
			"project_id":      project.ID().String(),
			"dispatch_source": "requirement",
		},
	})
	dispatchID := "dispatch_" + s.idGenerator.Generate()
	return &DispatchRequirementResult{
		RequirementID:   requirement.ID().String(),
		Status:          string(requirement.Status()),
		WorkspacePath:   requirement.WorkspacePath(),
		ReplicaAgentCode: requirement.ReplicaAgentCode(),
		TaskID:          dispatchID,
	}, nil
}

func (s *RequirementDispatchService) ensureDispatchSession(
	ctx context.Context,
	cmd DispatchRequirementCommand,
	replicaAgent *domain.Agent,
	requirement *domain.Requirement,
	project *domain.Project,
) error {
	if s.sessionService == nil {
		return nil
	}
	existingMetadata, err := s.sessionService.GetSessionMetadata(ctx, cmd.SessionKey)
	if err != nil && !errors.Is(err, ErrSessionNotFound) {
		return err
	}
	mergedMetadata := map[string]interface{}{}
	for key, value := range existingMetadata {
		mergedMetadata[key] = value
	}
	mergedMetadata["dispatch_source"] = "requirement"
	mergedMetadata["requirement_id"] = requirement.ID().String()
	mergedMetadata["project_id"] = project.ID().String()
	mergedMetadata["channel_code"] = cmd.ChannelCode
	mergedMetadata["agent_code"] = replicaAgent.AgentCode().String()
	mergedMetadata["user_code"] = replicaAgent.UserCode()
	if errors.Is(err, ErrSessionNotFound) {
		_, createErr := s.sessionService.CreateSession(ctx, CreateSessionCommand{
			UserCode:    replicaAgent.UserCode(),
			ChannelCode: cmd.ChannelCode,
			AgentCode:   replicaAgent.AgentCode().String(),
			SessionKey:  cmd.SessionKey,
			ExternalID:  cmd.SessionKey,
			Metadata:    mergedMetadata,
		})
		return createErr
	}
	return s.sessionService.UpdateSessionMetadata(ctx, UpdateSessionMetadataCommand{
		SessionKey: cmd.SessionKey,
		Metadata:   mergedMetadata,
	})
}

func (s *RequirementDispatchService) createReplicaAgent(ctx context.Context, baseAgent *domain.Agent, requirement *domain.Requirement, workspacePath string) (*domain.Agent, error) {
	snap := baseAgent.ToSnapshot()
	now := time.Now()
	snap.ID = domain.NewAgentID(s.idGenerator.Generate())
	snap.AgentCode = domain.NewAgentCode("agt_" + s.idGenerator.Generate())
	snap.Name = fmt.Sprintf("%s-replica-%s", baseAgent.Name(), requirement.ID().String())
	snap.IsDefault = false
	snap.IsActive = true
	snap.AgentType = domain.AgentTypeCoding
	snap.CreatedAt = now
	snap.UpdatedAt = now
	if snap.ClaudeCodeConfig == nil {
		snap.ClaudeCodeConfig = domain.DefaultClaudeCodeConfig()
	} else {
		cfg := *snap.ClaudeCodeConfig
		snap.ClaudeCodeConfig = &cfg
	}
	snap.ClaudeCodeConfig.Cwd = resolveReplicaAgentCwd(requirement, workspacePath)
	continueConversation := false
	forkSession := true
	snap.ClaudeCodeConfig.ContinueConversation = &continueConversation
	snap.ClaudeCodeConfig.ForkSession = &forkSession
	replica := &domain.Agent{}
	replica.FromSnapshot(snap)
	if err := s.agentRepo.Save(ctx, replica); err != nil {
		return nil, err
	}
	return replica, nil
}

func workspaceRootPath() string {
	if p := os.Getenv("AI_DEVOPS_WORKSPACE_ROOT"); p != "" {
		return p
	}
	return "/tmp/ai-devops"
}

func requirementWorkspaceRoot(requirement *domain.Requirement) string {
	if requirement != nil && requirement.TempWorkspaceRoot() != "" {
		return requirement.TempWorkspaceRoot()
	}
	return workspaceRootPath()
}

func resolveReplicaAgentCwd(requirement *domain.Requirement, workspacePath string) string {
	if requirement != nil {
		if tempWorkspace := strings.TrimSpace(requirement.TempWorkspaceRoot()); tempWorkspace != "" {
			return tempWorkspace
		}
	}
	return workspacePath
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func parseSessionKey(sessionKey string) (string, string, error) {
	trimmed := strings.TrimSpace(sessionKey)
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 {
		return "", "", ErrInvalidSessionKey
	}
	channelType := strings.TrimSpace(parts[0])
	chatID := strings.TrimSpace(parts[1])
	if channelType == "" || chatID == "" {
		return "", "", ErrInvalidSessionKey
	}
	return channelType, chatID, nil
}

func buildRequirementDispatchPrompt(requirement *domain.Requirement, project *domain.Project, workspacePath string) string {
	isHeartbeat := requirement.RequirementType() == domain.RequirementTypeHeartbeat

	// 心跳需求：调度员角色，不直接修改代码
	var prompt string
	if isHeartbeat {
		prompt = fmt.Sprintf(`你是当前项目的调度员心跳（Heartbeat Agent），职责是 orchestrate 任务而非直接写代码。

【需求信息】
- 需求ID：%s
- 标题：%s
- 描述：%s

【验收标准】
%s

【项目信息】
- 项目ID：%s
- 项目名称：%s
- 仓库地址：%s
- 默认分支：%s

【工作目录】
- 当前工作目录：%s

请按以下顺序执行：
1. 如果工作目录为空，先克隆代码仓库：git clone %s . && git checkout %s
2. 如果仓库已存在，先拉取最新代码：git checkout %s && git pull
3. 基于需求与验收标准，使用 taskmanager / gh 等 CLI 工具完成调度工作
4. **严禁**修改任何源代码、**严禁**执行 git commit、**严禁**执行 git push、**严禁**创建 PR
5. 所有需要代码改动的事项，必须使用 taskmanager requirement create 生成新需求，让其他 CodingAgent 完成
6. 工作完成后，直接输出本次心跳的执行结果摘要
`, requirement.ID().String(), requirement.Title(), firstNonEmpty(requirement.Description(), "无"),
			firstNonEmpty(requirement.AcceptanceCriteria(), "完成调度工作"),
			project.ID().String(), project.Name(), project.GitRepoURL(), project.DefaultBranch(), workspacePath,
			project.GitRepoURL(), project.DefaultBranch(), project.DefaultBranch())
	} else {
		initSteps := project.InitSteps()
		initStepsText := "无"
		if len(initSteps) > 0 {
			initStepsText = strings.Join(initSteps, "\n")
		}
		prompt = fmt.Sprintf(`你是当前需求的 CodingAgent 分身，请直接使用 Claude Code 在当前工作目录完成开发。

【需求信息】
- 需求ID：%s
- 标题：%s
- 描述：%s

【验收标准】
%s

【项目信息】
- 项目ID：%s
- 项目名称：%s
- 仓库地址：%s
- 默认分支：%s
- 初始化步骤：
%s

【工作目录】
- 当前工作目录：%s
- 要求：必须在该目录内完成代码操作

请按以下顺序执行：
1. 如果工作目录为空，先克隆代码仓库：git clone %s . && git checkout %s
2. 如果仓库已存在，先拉取最新代码：git checkout %s && git pull
3. 严格执行初始化步骤
4. 基于需求与验收标准完成实现
5. 运行必要的校验命令
6. 提交代码：git add . && git commit -m "feat: 完成需求 %s"
7. 推送代码：git push origin feature/%s
8. 创建 PR 或输出 PR 信息
`, requirement.ID().String(), requirement.Title(), firstNonEmpty(requirement.Description(), "无"),
			firstNonEmpty(requirement.AcceptanceCriteria(), "完成需求并通过验证"),
			project.ID().String(), project.Name(), project.GitRepoURL(), project.DefaultBranch(), initStepsText, workspacePath,
			project.GitRepoURL(), project.DefaultBranch(), project.DefaultBranch(),
			requirement.ID().String(), requirement.ID().String())
	}
	return prompt
}
