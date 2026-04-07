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
	"github.com/weibh/taskmanager/domain/state_machine"
	"github.com/weibh/taskmanager/infrastructure/config"
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
	requirementRepo  domain.RequirementRepository
	projectRepo      domain.ProjectRepository
	agentRepo        domain.AgentRepository
	stateMachineRepo state_machine.Repository
	taskService      interface{} // TaskApplicationService - no longer used
	sessionService   *SessionApplicationService
	idGenerator      domain.IDGenerator
	inboundPublisher interface {
		PublishInbound(msg *channelBus.InboundMessage)
	}
	replicaCleanupSvc domain.ReplicaCleanupService
}

func NewRequirementDispatchService(
	requirementRepo domain.RequirementRepository,
	projectRepo domain.ProjectRepository,
	agentRepo domain.AgentRepository,
	taskService interface{}, // TaskApplicationService - no longer used
	sessionService *SessionApplicationService,
	idGenerator domain.IDGenerator,
	replicaCleanupSvc domain.ReplicaCleanupService,
	stateMachineRepo state_machine.Repository,
) *RequirementDispatchService {
	return &RequirementDispatchService{
		requirementRepo:   requirementRepo,
		projectRepo:       projectRepo,
		agentRepo:         agentRepo,
		taskService:       taskService,
		sessionService:    sessionService,
		idGenerator:       idGenerator,
		replicaCleanupSvc: replicaCleanupSvc,
		stateMachineRepo:  stateMachineRepo,
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

	workspacePath := filepath.Join(requirementWorkspaceRoot(requirement), requirement.ProjectID().String(), requirement.ID().String())
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		return nil, err
	}
	replicaAgent, err := s.createReplicaAgent(ctx, baseAgent, requirement, workspacePath)
	if err != nil {
		requirement.SetWorkspacePath(workspacePath)
		requirement.SetClaudeRuntimeError(err.Error())
		_ = s.requirementRepo.Save(ctx, requirement)
		_ = os.RemoveAll(workspacePath)
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
		_ = s.requirementRepo.Save(ctx, requirement)
		_ = os.RemoveAll(workspacePath)
		return nil, err
	}
	if s.inboundPublisher == nil {
		requirement.SetClaudeRuntimeError(ErrInboundPublisherNotConfigured.Error())
		_ = s.requirementRepo.Save(ctx, requirement)
		_ = os.RemoveAll(workspacePath)
		return nil, ErrInboundPublisherNotConfigured
	}
	if err := s.ensureDispatchSession(ctx, cmd, replicaAgent, requirement, project); err != nil {
		requirement.SetClaudeRuntimeError(err.Error())
		_ = s.requirementRepo.Save(ctx, requirement)
		_ = os.RemoveAll(workspacePath)
		return nil, err
	}

	// 获取状态机信息
	stateMachineName := s.getProjectStateMachineName(ctx, project.ID().String(), requirement.RequirementType())

	// 获取当前状态机状态和 AI Guide
	currentState, aiGuide := s.getStateMachineGuide(ctx, project.ID().String(), requirement.RequirementType())

	// 获取完整的状态机配置（用于注入触发器表）
	smConfig := s.getStateMachineConfig(ctx, project.ID().String(), requirement.RequirementType())

	// 记录状态转换日志
	if s.stateMachineRepo != nil && currentState != "" {
		fromStatus := string(requirement.Status())
		log := state_machine.NewTransitionLog(
			requirement.ID().String(),
			fromStatus,
			currentState,
			"dispatch",
			"system",
			"派发需求",
		)
		_ = s.stateMachineRepo.SaveTransitionLog(ctx, log)

		// 保存/更新 RequirementState
		s.saveRequirementState(ctx, requirement, currentState)
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

func (s *RequirementDispatchService) getProjectStateMachineName(ctx context.Context, projectID string, reqType domain.RequirementType) string {
	if s.stateMachineRepo == nil {
		return ""
	}

	psm, err := s.stateMachineRepo.GetProjectStateMachine(ctx, projectID, state_machine.RequirementType(reqType))
	if err != nil {
		return ""
	}

	snap := psm.ToSnapshot()
	sm, err := s.stateMachineRepo.GetStateMachine(ctx, snap.StateMachineID)
	if err != nil {
		return ""
	}

	return sm.Name
}

// getStateMachineGuide 获取当前状态机状态和 AI Guide
// 返回当前状态 ID 和 AI Guide 信息
func (s *RequirementDispatchService) getStateMachineGuide(ctx context.Context, projectID string, reqType domain.RequirementType) (string, map[string]interface{}) {
	if s.stateMachineRepo == nil {
		return "", nil
	}

	// 获取项目状态机映射
	psm, err := s.stateMachineRepo.GetProjectStateMachine(ctx, projectID, state_machine.RequirementType(reqType))
	if err != nil {
		return "", nil
	}

	snap := psm.ToSnapshot()

	// 获取状态机配置
	sm, err := s.stateMachineRepo.GetStateMachine(ctx, snap.StateMachineID)
	if err != nil {
		return "", nil
	}

	// 获取需求当前状态（从 RequirementState）
	reqState, err := s.stateMachineRepo.GetRequirementState(ctx, projectID+"_"+string(reqType))
	if err != nil {
		// 如果没有 RequirementState，返回初始状态
		return sm.Config.InitialState, sm.Config.GetStateAIGuide(sm.Config.InitialState)
	}

	// 返回当前状态和 AI Guide
	return reqState.CurrentState, sm.Config.GetStateAIGuide(reqState.CurrentState)
}

// saveRequirementState 保存需求状态到状态机
func (s *RequirementDispatchService) saveRequirementState(ctx context.Context, requirement *domain.Requirement, currentState string) {
	// 获取项目状态机映射
	psm, err := s.stateMachineRepo.GetProjectStateMachine(ctx, requirement.ProjectID().String(), state_machine.RequirementType(requirement.RequirementType()))
	if err != nil {
		return
	}

	snap := psm.ToSnapshot()

	// 获取状态机配置
	sm, err := s.stateMachineRepo.GetStateMachine(ctx, snap.StateMachineID)
	if err != nil {
		return
	}

	// 获取状态信息
	stateInfo := sm.Config.GetState(currentState)
	if stateInfo == nil {
		return
	}

	// 创建或更新 RequirementState
	rs := state_machine.NewRequirementState(requirement.ID().String(), sm.ID, currentState, stateInfo.Name)
	_ = s.stateMachineRepo.SaveRequirementState(ctx, rs)
}

// getStateMachineConfig 获取完整的状态机配置
func (s *RequirementDispatchService) getStateMachineConfig(ctx context.Context, projectID string, reqType domain.RequirementType) *state_machine.Config {
	if s.stateMachineRepo == nil {
		return nil
	}

	// 获取项目状态机映射
	psm, err := s.stateMachineRepo.GetProjectStateMachine(ctx, projectID, state_machine.RequirementType(reqType))
	if err != nil {
		return nil
	}

	snap := psm.ToSnapshot()

	// 获取状态机
	sm, err := s.stateMachineRepo.GetStateMachine(ctx, snap.StateMachineID)
	if err != nil {
		return nil
	}

	return sm.Config
}

func workspaceRootPath() string {
	return config.GetAgentAIWorkSpaceRoot()
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

func buildRequirementDispatchPrompt(requirement *domain.Requirement, project *domain.Project, workspacePath string, stateMachineName string, currentState string, aiGuide map[string]interface{}, smConfig *state_machine.Config) string {
	isHeartbeat := requirement.RequirementType() == domain.RequirementTypeHeartbeat
	requirementType := "普通需求"
	if isHeartbeat {
		requirementType = "心跳需求"
	}

	// 构建状态机使用指南和 AI 指南
	stateMachineGuide := buildStateMachineGuide(stateMachineName, smConfig)
	stateAIGuide := buildStateAIGuide(aiGuide)

	// 心跳需求：调度员角色，不直接修改代码
	var prompt string
	if isHeartbeat {
		prompt = fmt.Sprintf(`你是当前项目的调度员心跳（Heartbeat Agent），职责是 orchestrate 任务而非直接修改代码。

【需求元信息】
- 需求ID：%s
- 需求类型：%s
- 当前状态：%s
- 状态机当前状态：%s
- 需求标题：%s
- 需求描述：%s
- 项目ID：%s
- 项目名称：%s
- 关联状态机：%s

【验收标准】
%s

【项目信息】
- 仓库地址：%s
- 默认分支：%s

【工作目录】
- 当前工作目录：%s

%s

%s

【执行流程 - 心跳任务】
你是调度员，负责编排任务而非直接修改代码。请按照以下心跳任务定义执行：

%s

【调度员核心守则】
1. **严禁**直接修改任何源代码文件
2. **严禁**执行 git commit、git push、创建 PR
3. 所有代码改动必须通过 "taskmanager requirement create" 创建新需求，交由 CodingAgent 完成
4. 根据工作结果，使用状态机命令更新需求状态
5. 工作完成后，输出本次心跳的执行结果摘要
`, requirement.ID().String(), requirementType, requirement.Status(), firstNonEmpty(currentState, "未初始化"), requirement.Title(), firstNonEmpty(requirement.Description(), "无"),
			project.ID().String(), project.Name(), firstNonEmpty(stateMachineName, "未配置"),
			firstNonEmpty(requirement.AcceptanceCriteria(), "完成调度工作"),
			project.GitRepoURL(), project.DefaultBranch(), workspacePath,
			stateAIGuide,
			stateMachineGuide,
			project.HeartbeatMDContent())
	} else {
		initSteps := project.InitSteps()
		initStepsText := "无"
		if len(initSteps) > 0 {
			initStepsText = strings.Join(initSteps, "\n")
		}
		prompt = fmt.Sprintf(`你是当前需求的 CodingAgent 分身，请直接使用 Claude Code 在当前工作目录完成开发。

【执行契约 - 严格遵守】
当前阶段：%s
下一步动作：执行状态转换进入下一阶段

执行规则：
1. 必须执行命令，不输出解释性长文本
2. 每个 PHASE 完成后必须执行 ON_PHASE_COMPLETE 命令
3. 命令失败自动修复重试（最多3次）
4. 优先执行操作，不做过多分析

【需求元信息】
- 需求ID：%s
- 需求类型：%s
- 需求标题：%s
- 需求描述：%s
- 项目ID：%s
- 项目名称：%s

【验收标准】
%s

【项目信息】
- 仓库地址：%s
- 默认分支：%s
- 初始化步骤：
%s

【工作目录】
- 当前工作目录：%s
- 要求：必须在该目录内完成代码操作

%s

【工作循环 - 持续执行】
1. 规划当前阶段任务
2. 执行命令
3. 验证执行结果
4. 若失败 → 自动修复 → 重试
5. 进入下一 PHASE

【禁止行为】
- 修改与需求无关的文件
- 跳过测试或验证步骤
- 提交未通过验证的代码
- 创建额外分支

【失败处理】
若命令失败：
1. 分析错误输出
2. 自动修复
3. 重试最多3次
4. 仍失败则输出阻塞原因并停止


【执行流水线 - PIPELINE】
%s

【状态机附录 - 调试参考】
%s
`, firstNonEmpty(currentState, "未初始化"),
			requirement.ID().String(), requirementType, requirement.Title(), firstNonEmpty(requirement.Description(), "无"),
			project.ID().String(), project.Name(),
			firstNonEmpty(requirement.AcceptanceCriteria(), "完成需求并通过验证"),
			project.GitRepoURL(), project.DefaultBranch(), initStepsText, workspacePath,
			stateAIGuide,
			buildExecutionPipeline(smConfig, requirement.ID().String(), project.GitRepoURL(), project.DefaultBranch()),
			stateMachineGuide)
	}
	return prompt
}

// buildStateMachineGuide 构建状态机使用指南
func buildStateMachineGuide(stateMachineName string, smConfig *state_machine.Config) string {
	if stateMachineName == "" {
		return `【状态机使用指南】
当前需求未配置状态机。如需使用状态机管理工作流，请联系管理员配置。`
	}

	guide := fmt.Sprintf(`【状态机使用指南】
本需求关联的状态机名称：%s

你可以使用以下命令管理需求状态：

【推荐】一键状态转换（自动同步状态机+需求状态）：
   taskmanager requirement transition --id <需求ID> --trigger=<触发器>

【分步操作】如需更细粒度控制：

1. 查看当前需求状态：
   taskmanager requirement get-state --id <需求ID>

2. 查看当前状态可用触发器：
   taskmanager statemachine triggers --machine=<状态机名称> --from=<当前状态>

3. 验证状态转换是否合法：
   taskmanager statemachine validate --machine=<状态机名称> --from=<当前状态> --to=<目标状态>

4. 执行状态转换并同步需求状态：
   taskmanager requirement transition --id <需求ID> --trigger=<触发器>
`,
		stateMachineName)

	// 如果有状态机配置，注入完整触发器表
	if smConfig != nil {
		triggerTable := buildTriggerTable(smConfig)
		if triggerTable != "" {
			guide += fmt.Sprintf(`【状态机触发器表】
%s

`, triggerTable)
		}
	}

	return guide
}

// buildTriggerTable 构建状态机的完整触发器表（Markdown格式）
// 包含所有状态和从该状态出发可用的所有触发器
func buildTriggerTable(smConfig *state_machine.Config) string {
	if smConfig == nil || len(smConfig.States) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("| 源状态 | 触发器 | 目标状态 | 说明 |\n")
	sb.WriteString("|--------|--------|----------|------|\n")

	// 按状态分组，显示每个状态可用的触发器
	for _, state := range smConfig.States {
		if state.IsFinal {
			// 终态不显示触发器（不会有从终态出发的转换）
			continue
		}

		triggers := smConfig.GetAvailableTriggers(state.ID)
		if len(triggers) == 0 {
			// 没有触发器的状态显示为 -
			sb.WriteString(fmt.Sprintf("| %s | - | - | 无可用触发器 |\n", state.Name))
			continue
		}

		// 每个触发器一行
		for i, trigger := range triggers {
			// 查找对应的转换获取目标状态
			transition := findTransitionByTrigger(smConfig, state.ID, trigger.Trigger)
			toState := ""
			description := trigger.Description
			if transition != nil {
				toState = getStateName(smConfig, transition.ToState)
			}

			// 第一行显示状态名，后续行留空
			fromState := state.Name
			if i > 0 {
				fromState = ""
			}

			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				fromState, trigger.Trigger, toState, description))
		}
	}

	return sb.String()
}

// findTransitionByTrigger 根据触发器查找转换
func findTransitionByTrigger(smConfig *state_machine.Config, fromState, trigger string) *state_machine.Transition {
	for i := range smConfig.Transitions {
		t := &smConfig.Transitions[i]
		if t.FromState == fromState && t.Trigger == trigger {
			return t
		}
	}
	return nil
}

// getStateName 根据状态ID获取状态名称
func getStateName(smConfig *state_machine.Config, stateID string) string {
	for _, s := range smConfig.States {
		if s.ID == stateID {
			return s.Name
		}
	}
	return stateID
}

// buildExecutionPipeline 根据状态机配置动态生成执行流水线
// 使用 PHASE 结构和 ON_PHASE_COMPLETE Hook
func buildExecutionPipeline(smConfig *state_machine.Config, requirementID, gitRepo, branch string) string {
	if smConfig == nil || len(smConfig.Transitions) == 0 {
		return `PHASE 1 — EXECUTE
执行需求开发工作
ON_PHASE_COMPLETE:
taskmanager requirement transition --id ` + requirementID + ` --trigger=completed
`
	}

	// 构建状态链
	stateChain := buildStateChain(smConfig)
	if len(stateChain) == 0 {
		return `PHASE 1 — EXECUTE
执行需求开发工作
ON_PHASE_COMPLETE:
taskmanager requirement transition --id ` + requirementID + ` --trigger=completed
`
	}

	var sb strings.Builder
	phaseNum := 1

	// PHASE 0: 初始化环境（总是第一阶段）
	sb.WriteString(`PHASE 0 — INIT
执行：
`)
	sb.WriteString(fmt.Sprintf("git clone %s . && git checkout %s\n", gitRepo, branch))
	sb.WriteString(fmt.Sprintf("git checkout %s && git pull\n", branch))
	sb.WriteString(`ON_PHASE_COMPLETE:
`)
	sb.WriteString(fmt.Sprintf("taskmanager requirement transition --id %s --trigger=%s\n\n",
		requirementID, getFirstTrigger(smConfig)))

	// 遍历状态链生成 PHASE
	for i := 0; i < len(stateChain)-1; i++ {
		currentStateID := stateChain[i]
		nextStateID := stateChain[i+1]

		currentState := smConfig.GetState(currentStateID)
		if currentState == nil {
			continue
		}

		// 跳过终态
		if currentState.IsFinal {
			continue
		}

		// 查找转换获取触发器
		trigger := ""
		for _, t := range smConfig.Transitions {
			if t.FromState == currentStateID && t.ToState == nextStateID {
				trigger = t.Trigger
				break
			}
		}

		// PHASE 头部
		phaseName := strings.ToUpper(strings.ReplaceAll(currentState.Name, " ", "_"))
		sb.WriteString(fmt.Sprintf("PHASE %d — %s\n", phaseNum, phaseName))

		// 添加 AI 指南中的执行指令（提取命令，排除状态转换命令）
		aiGuide := strings.TrimSpace(currentState.AIGuide)
		if aiGuide != "" {
			guideLines := strings.Split(aiGuide, "\n")
			for _, line := range guideLines {
				trimmed := strings.TrimSpace(line)
				// 跳过包含"命令："、taskmanager、## 标题、**标记的行
				if strings.Contains(trimmed, "命令：") ||
					strings.HasPrefix(trimmed, "taskmanager") ||
					strings.HasPrefix(trimmed, "## ") ||
					strings.HasPrefix(trimmed, "**") ||
					trimmed == "" {
					continue
				}
				sb.WriteString(trimmed + "\n")
			}
		}

		// 如果是第一个业务 PHASE，添加环境初始化提示
		if phaseNum == 1 {
			sb.WriteString("\n执行环境初始化（已在 PHASE 0 完成）\n")
		}

		// ON_PHASE_COMPLETE Hook
		if trigger != "" {
			sb.WriteString("\nON_PHASE_COMPLETE:\n")
			sb.WriteString(fmt.Sprintf("taskmanager requirement transition --id %s --trigger=%s\n",
				requirementID, trigger))
		}

		sb.WriteString("\n")
		phaseNum++
	}

	// 如果没有生成任何 PHASE
	if phaseNum == 1 {
		return `PHASE 1 — EXECUTE
执行需求开发工作
ON_PHASE_COMPLETE:
taskmanager requirement transition --id ` + requirementID + ` --trigger=completed
`
	}

	return sb.String()
}

// getFirstTrigger 获取状态机的第一个触发器（从 initial_state 出发）
func getFirstTrigger(smConfig *state_machine.Config) string {
	if smConfig == nil {
		return "start"
	}
	for _, t := range smConfig.Transitions {
		if t.FromState == smConfig.InitialState {
			return t.Trigger
		}
	}
	return "start"
}

// buildStateChain 从状态机构建状态链
// 根据 transitions 构建从 initial_state 开始的完整状态路径
func buildStateChain(smConfig *state_machine.Config) []string {
	if smConfig == nil {
		return nil
	}

	// 构建转换映射：from_state -> to_state
	transitionMap := make(map[string]string)
	for _, t := range smConfig.Transitions {
		transitionMap[t.FromState] = t.ToState
	}

	// 从 initial_state 开始，沿着转换链构建状态数组
	var chain []string
	currentState := smConfig.InitialState

	for i := 0; i < len(smConfig.States)+1; i++ { // 防止无限循环
		chain = append(chain, currentState)
		nextState, exists := transitionMap[currentState]
		if !exists {
			break
		}
		// 防止循环
		if contains(chain, nextState) {
			break
		}
		currentState = nextState
	}

	return chain
}

// buildStateAIGuide 构建状态的 AI 指南
func buildStateAIGuide(aiGuide map[string]interface{}) string {
	if aiGuide == nil {
		return "【当前状态执行指南】\n暂无详细的 AI 执行指南。请根据需求描述和验收标准自行判断当前阶段的工作内容。"
	}

	var result strings.Builder
	result.WriteString("【当前状态执行指南】\n")

	// 状态名称
	if name, ok := aiGuide["name"].(string); ok && name != "" {
		result.WriteString(fmt.Sprintf("当前阶段：%s\n\n", name))
	}

	// AI Guide
	if guide, ok := aiGuide["ai_guide"].(string); ok && guide != "" {
		result.WriteString(guide)
		result.WriteString("\n\n")
	}

	// 自动初始化
	if autoInit, ok := aiGuide["auto_init"].(string); ok && autoInit != "" {
		result.WriteString("【自动初始化命令】\n")
		result.WriteString("进入此状态时，将自动执行以下命令：\n")
		result.WriteString("```bash\n")
		result.WriteString(autoInit)
		result.WriteString("\n```\n\n")
	}

	// 成功判断标准
	if success, ok := aiGuide["success_criteria"].(string); ok && success != "" {
		result.WriteString("【成功判断标准】\n")
		result.WriteString(success)
		result.WriteString("\n\n")
	}

	// 失败判断标准
	if failure, ok := aiGuide["failure_criteria"].(string); ok && failure != "" {
		result.WriteString("【失败判断标准】\n")
		result.WriteString(failure)
		result.WriteString("\n\n")
	}

	// 可用触发器
	if triggers, ok := aiGuide["triggers"].([]interface{}); ok && len(triggers) > 0 {
		result.WriteString("【可用状态转换】\n")
		result.WriteString("根据工作结果，选择合适的触发器执行转换：\n\n")
		for _, t := range triggers {
			if trigger, ok := t.(map[string]interface{}); ok {
				triggerName := ""
				description := ""
				condition := ""

				if n, ok := trigger["trigger"].(string); ok {
					triggerName = n
				}
				if d, ok := trigger["description"].(string); ok {
					description = d
				}
				if c, ok := trigger["condition"].(string); ok {
					condition = c
				}

				result.WriteString(fmt.Sprintf("• %s", triggerName))
				if description != "" {
					result.WriteString(fmt.Sprintf(" - %s", description))
				}
				result.WriteString("\n")
				if condition != "" {
					result.WriteString(fmt.Sprintf("  触发条件：%s\n", condition))
				}
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}
