package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/weibh/taskmanager/domain"
	channelBus "github.com/weibh/taskmanager/pkg/bus"
)

// TriggerAgentExecutor 触发 Agent 动作执行器
type TriggerAgentExecutor struct {
	agentRepo       domain.AgentRepository
	requirementRepo domain.RequirementRepository
	projectRepo     domain.ProjectRepository
	idGenerator     domain.IDGenerator
	publisher       MessagePublisher
}

// MessagePublisher 消息发布接口
type MessagePublisher interface {
	PublishInbound(msg *channelBus.InboundMessage)
}

// NewTriggerAgentExecutor 创建执行器
func NewTriggerAgentExecutor(
	agentRepo domain.AgentRepository,
	requirementRepo domain.RequirementRepository,
	projectRepo domain.ProjectRepository,
	idGenerator domain.IDGenerator,
	publisher MessagePublisher,
) *TriggerAgentExecutor {
	return &TriggerAgentExecutor{
		agentRepo:       agentRepo,
		requirementRepo: requirementRepo,
		projectRepo:     projectRepo,
		idGenerator:     idGenerator,
		publisher:       publisher,
	}
}

// Supports 返回支持的动作类型
func (e *TriggerAgentExecutor) Supports(actionType string) bool {
	return actionType == "coding_agent"
}

// Execute 执行触发 Agent 动作
func (e *TriggerAgentExecutor) Execute(
	ctx context.Context,
	config *domain.RequirementHookConfig,
	req *domain.Requirement,
	change *domain.StateChange,
) (*domain.ActionResult, error) {
	fmt.Printf("[DEBUG] TriggerAgentExecutor.Execute: config=%s, actionType=%s, requirement=%s\n", config.Name, config.ActionType, req.ID())

	// 1. 解析配置
	var actionConfig domain.TriggerAgentActionConfig
	if err := json.Unmarshal([]byte(config.ActionConfig), &actionConfig); err != nil {
		return nil, fmt.Errorf("invalid action config: %w", err)
	}

	fmt.Printf("[DEBUG] TriggerAgentExecutor: promptTemplate=%s\n", actionConfig.PromptTemplate)

	// 2. 获取项目配置，使用项目默认 Agent
	project, err := e.projectRepo.FindByID(ctx, req.ProjectID())
	if err != nil {
		return nil, fmt.Errorf("failed to find project: %w", err)
	}
	if project == nil {
		return nil, fmt.Errorf("project not found: %s", req.ProjectID())
	}

	// 使用项目默认 agent_code
	agentCode := project.AgentCode()
	if agentCode == "" {
		return nil, fmt.Errorf("project %s has no default agent configured", project.Name())
	}

	baseAgent, err := e.agentRepo.FindByAgentCode(ctx, domain.NewAgentCode(agentCode))
	if err != nil {
		return nil, fmt.Errorf("failed to find base agent: %w", err)
	}
	if baseAgent == nil {
		return nil, fmt.Errorf("base agent not found: %s", agentCode)
	}

	// 3. 创建工作目录
	workspacePath := e.renderWorkspace(actionConfig.WorkspaceTemplate, req)
	if err := mkdirAll(workspacePath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// 4. 创建分身 Agent
	replicaAgent, err := e.createReplicaAgent(ctx, baseAgent, req, workspacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create replica agent: %w", err)
	}

	// 5. 更新需求的 replica_agent_code，确保后续清理能正确追踪新分身
	req.SetReplicaAgentCode(replicaAgent.AgentCode().String())
	req.SetWorkspacePath(workspacePath)
	if e.requirementRepo != nil {
		if err := e.requirementRepo.Save(ctx, req); err != nil {
			return nil, fmt.Errorf("failed to save requirement replica info: %w", err)
		}
	}

	// 6. 构建 Prompt
	prompt := e.renderPrompt(actionConfig.PromptTemplate, req, change)

	// 7. 解析 session_key 获取 channel 和 chat_id
	// session_key 格式: "feishu:ou_xxx" 或 "internal:xxx"
	sessionKey := req.DispatchSessionKey()
	var channel, chatID string
	if sessionKey != "" {
		parts := strings.SplitN(sessionKey, ":", 2)
		if len(parts) == 2 {
			channel = parts[0]
			chatID = parts[1]
		} else {
			channel = "internal"
			chatID = sessionKey
		}
	} else {
		channel = "internal"
		chatID = replicaAgent.ID().String()
	}

	// 8. 发送任务消息
	e.publisher.PublishInbound(&channelBus.InboundMessage{
		Channel:   channel,
		SenderID:  "hook_system",
		ChatID:    chatID,
		Content:   prompt,
		Timestamp: time.Now(),
		Media:     []string{},
		Metadata: map[string]any{
			"agent_code":     replicaAgent.AgentCode().String(),
			"user_code":      replicaAgent.UserCode(),
			"requirement_id": req.ID().String(),
			"hook_config_id": config.ID,
			"dispatch_source": "hook_trigger",
		},
	})

	return &domain.ActionResult{
		Success: true,
		Output:  fmt.Sprintf("triggered agent %s, replica %s", baseAgent.ID().String(), replicaAgent.ID().String()),
	}, nil
}

func (e *TriggerAgentExecutor) createReplicaAgent(ctx context.Context, baseAgent *domain.Agent, req *domain.Requirement, workspacePath string) (*domain.Agent, error) {
	snap := baseAgent.ToSnapshot()
	now := time.Now()
	snap.ID = domain.NewAgentID(e.idGenerator.Generate())
	snap.AgentCode = domain.NewAgentCode("agt_" + e.idGenerator.Generate())
	snap.Name = fmt.Sprintf("%s-hook-%s", baseAgent.Name(), req.ID().String())
	snap.IsDefault = false
	snap.IsActive = true
	snap.AgentType = domain.AgentTypeCoding
	snap.ShadowFrom = baseAgent.AgentCode().String() // 记录分身来源
	snap.CreatedAt = now
	snap.UpdatedAt = now

	if snap.ClaudeCodeConfig == nil {
		snap.ClaudeCodeConfig = domain.DefaultClaudeCodeConfig()
	} else {
		cfg := *snap.ClaudeCodeConfig
		snap.ClaudeCodeConfig = &cfg
	}
	snap.ClaudeCodeConfig.Cwd = workspacePath

	continueConversation := false
	forkSession := true
	snap.ClaudeCodeConfig.ContinueConversation = &continueConversation
	snap.ClaudeCodeConfig.ForkSession = &forkSession

	replica := &domain.Agent{}
	replica.FromSnapshot(snap)
	if err := e.agentRepo.Save(ctx, replica); err != nil {
		return nil, err
	}
	return replica, nil
}

func (e *TriggerAgentExecutor) renderPrompt(template string, req *domain.Requirement, change *domain.StateChange) string {
	result := template

	// 替换需求相关变量
	result = strings.ReplaceAll(result, "${requirement.id}", req.ID().String())
	result = strings.ReplaceAll(result, "${requirement.title}", req.Title())
	result = strings.ReplaceAll(result, "${requirement.description}", req.Description())
	result = strings.ReplaceAll(result, "${requirement.acceptance_criteria}", req.AcceptanceCriteria())
	result = strings.ReplaceAll(result, "${requirement.temp_workspace_root}", req.TempWorkspaceRoot())

	// 替换项目相关变量（需要获取项目信息，这里暂时用占位符）
	result = strings.ReplaceAll(result, "${project.id}", req.ProjectID().String())
	result = strings.ReplaceAll(result, "${project.name}", "")

	// 替换工作目录变量
	result = strings.ReplaceAll(result, "${workspace.path}", req.WorkspacePath())

	// 替换 Agent 相关变量
	result = strings.ReplaceAll(result, "${agent.id}", req.ReplicaAgentCode())

	// 替换状态变更相关变量
	result = strings.ReplaceAll(result, "${change.trigger}", change.Trigger)
	result = strings.ReplaceAll(result, "${change.reason}", change.Reason)
	result = strings.ReplaceAll(result, "${change.from_status}", string(change.FromStatus))
	result = strings.ReplaceAll(result, "${change.to_status}", string(change.ToStatus))

	return result
}

func (e *TriggerAgentExecutor) renderWorkspace(template string, req *domain.Requirement) string {
	if template == "" {
		if req.WorkspacePath() != "" {
			return req.WorkspacePath()
		}
		// 兜底：使用默认工作目录（与需求派发时保持一致）
		return fmt.Sprintf("/tmp/ai-devops/%s/%s", req.ProjectID().String(), req.ID().String())
	}

	result := template
	result = strings.ReplaceAll(result, "${requirement.id}", req.ID().String())
	result = strings.ReplaceAll(result, "${project.id}", req.ProjectID().String())
	return result
}

func mkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}
