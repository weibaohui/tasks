package application

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/domain/statemachine"
	channelBus "github.com/weibh/taskmanager/pkg/bus"
)

// HeartbeatTriggerSource 定义心跳触发来源。
type HeartbeatTriggerSource string

const (
	HeartbeatTriggerSourceManual    HeartbeatTriggerSource = "manual"
	HeartbeatTriggerSourceScheduler HeartbeatTriggerSource = "scheduler"
	HeartbeatTriggerSourceWebhook   HeartbeatTriggerSource = "webhook"
)

// HeartbeatRunRecord 描述一次心跳触发执行记录。
type HeartbeatRunRecord struct {
	RequirementID string `json:"requirement_id"`
	HeartbeatID   string `json:"heartbeat_id"`
	HeartbeatName string `json:"heartbeat_name"`
	ProjectID     string `json:"project_id"`
	TriggerSource string `json:"trigger_source"`
	Status        string `json:"status"`
	Title         string `json:"title"`
	LastError     string `json:"last_error"`
	ErrorCategory string `json:"error_category"`
	CreatedAt     int64  `json:"created_at"`
}

// HeartbeatRunPage 描述分页返回的心跳运行记录。
type HeartbeatRunPage struct {
	Data   []HeartbeatRunRecord `json:"data"`
	Total  int                  `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
}

// HeartbeatTriggerService 心跳触发服务
// 负责执行单次心跳的完整流程：创建需求、初始化状态机、派发需求
type HeartbeatTriggerService struct {
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
	_, err := s.TriggerWithSource(ctx, heartbeatID, HeartbeatTriggerSourceManual, "")
	return err
}

// TriggerWithSource 按指定触发来源执行心跳，并返回创建的需求。
// sourceID 用于标识触发来源的具体ID，例如 webhook 触发时为 WebhookEventLogID
func (s *HeartbeatTriggerService) TriggerWithSource(ctx context.Context, heartbeatID string, source HeartbeatTriggerSource, sourceID string) (*domain.Requirement, error) {
	hb, err := s.heartbeatRepo.FindByID(ctx, domain.NewHeartbeatID(heartbeatID))
	if err != nil {
		return nil, fmt.Errorf("failed to find heartbeat %s: %w", heartbeatID, err)
	}
	if hb == nil {
		return nil, fmt.Errorf("failed to find heartbeat %s: not found", heartbeatID)
	}
	if !hb.Enabled() {
		return nil, fmt.Errorf("heartbeat %s is disabled", heartbeatID)
	}

	project, err := s.projectRepo.FindByID(ctx, hb.ProjectID())
	if err != nil {
		return nil, fmt.Errorf("failed to find project for heartbeat %s: %w", heartbeatID, err)
	}
	if project == nil {
		return nil, fmt.Errorf("failed to find project for heartbeat %s: not found", heartbeatID)
	}

	triggerSource := strings.TrimSpace(string(source))
	if triggerSource == "" {
		triggerSource = string(HeartbeatTriggerSourceManual)
	}
	log.Printf("[HEARTBEAT] 执行心跳，来源=%s，心跳=%s，项目=%s", triggerSource, hb.Name(), project.Name())

	// 解析 Agent：优先使用心跳指定的 agent，若不存在则回退到项目默认 agent
	agentCode := hb.AgentCode()
	if agentCode == "" {
		// 心跳未指定 agentCode，使用项目默认
		agentCode = project.DefaultAgentCode()
		if agentCode == "" {
			log.Printf("[HEARTBEAT] 心跳和项目均未配置可用Agent，无法派发，心跳=%s，项目=%s", hb.Name(), project.Name())
			return nil, fmt.Errorf("heartbeat has no agent code and project has no default agent")
		}
		log.Printf("[HEARTBEAT] 心跳使用项目默认Agent，心跳=%s，agent=%s", hb.Name(), agentCode)
	} else if s.agentRepo != nil {
		// 心跳指定了 agentCode，验证是否存在
		baseAgent, _ := s.agentRepo.FindByAgentCode(ctx, domain.NewAgentCode(agentCode))
		if baseAgent == nil {
			fallback := project.DefaultAgentCode()
			if fallback != "" {
				log.Printf("[HEARTBEAT] 心跳Agent不存在，回退项目默认Agent，心跳=%s，原Agent=%s，回退Agent=%s", hb.Name(), agentCode, fallback)
				agentCode = fallback
			} else {
				// 为了兼容历史行为：未找到 agent 时仅记录告警，后续交由派发阶段兜底处理。
				log.Printf("[HEARTBEAT] 心跳Agent不存在且项目无默认Agent，保持原Agent尝试派发，心跳=%s，agent=%s", hb.Name(), agentCode)
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
		fmt.Sprintf("[心跳][%s] %s - %s", triggerSource, hb.Name(), time.Now().Format("2006-01-02 15:04")),
		prompt,
		"",
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create requirement: %w", err)
	}

	// 设置需求类型
	requirement.SetRequirementType(domain.RequirementType(reqType))

	// 设置触发来源和心跳信息
	requirement.SetSource(string(source), sourceID)
	requirement.SetHeartbeatInfo(hb.ID().String(), hb.Name())

	// 保存需求
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, fmt.Errorf("failed to save requirement: %w", err)
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

	log.Printf("[HEARTBEAT] 已创建心跳需求，需求ID=%s，心跳=%s，准备派发", requirement.ID(), hb.Name())

	// 直接派发心跳需求
	if s.requirementDispatchService != nil {
		channelCode := project.DispatchChannelCode()
		sessionKey := project.DispatchSessionKey()
		if channelCode == "" || sessionKey == "" {
			log.Printf("[HEARTBEAT] 项目缺少派发渠道或会话配置，项目=%s", project.Name())
			s.markRequirementDispatchFailed(ctx, requirement, "dispatch_config_missing: project has no dispatch channel or session key configured")
			return nil, fmt.Errorf("project has no dispatch channel or session key configured")
		}
		result, err := s.requirementDispatchService.DispatchRequirement(ctx, DispatchRequirementCommand{
			RequirementID: requirement.ID(),
			AgentCode:     agentCode,
			ChannelCode:   channelCode,
			SessionKey:    sessionKey,
		})
		if err != nil {
			s.markRequirementDispatchFailed(ctx, requirement, fmt.Sprintf("dispatch_failed: %v", err))
			return nil, fmt.Errorf("failed to dispatch requirement %s: %w", requirement.ID(), err)
		}
		log.Printf("[HEARTBEAT] 派发成功，需求ID=%s，任务ID=%s，工作区=%s", requirement.ID(), result.TaskID, result.WorkspacePath)
	} else {
		log.Printf("[HEARTBEAT] 派发服务不可用，需求已创建但未派发，需求ID=%s", requirement.ID())
		s.markRequirementDispatchFailed(ctx, requirement, "dispatch_service_unavailable: requirement dispatch service not available")
		return nil, fmt.Errorf("requirement dispatch service not available")
	}

	return requirement, nil
}

// ListRunsByHeartbeat 查询指定心跳的最近执行记录。
func (s *HeartbeatTriggerService) ListRunsByHeartbeat(ctx context.Context, heartbeatID string, limit int) ([]HeartbeatRunRecord, error) {
	hb, err := s.heartbeatRepo.FindByID(ctx, domain.NewHeartbeatID(heartbeatID))
	if err != nil {
		return nil, fmt.Errorf("failed to find heartbeat %s: %w", heartbeatID, err)
	}
	if hb == nil {
		return nil, fmt.Errorf("heartbeat %s not found", heartbeatID)
	}
	if limit <= 0 {
		limit = 20
	}
	filter := domain.RequirementListFilter{
		ProjectID: ptrProjectID(hb.ProjectID()),
		SortBy:    "created_at",
		Order:     "desc",
		Limit:     limit * 3,
		Offset:    0,
	}
	reqs, err := s.requirementRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	records := make([]HeartbeatRunRecord, 0, limit)
	for _, req := range reqs {
		if !isRequirementFromHeartbeat(req, hb) {
			continue
		}
		records = append(records, HeartbeatRunRecord{
			RequirementID: req.ID().String(),
			HeartbeatID:   hb.ID().String(),
			HeartbeatName: hb.Name(),
			ProjectID:     hb.ProjectID().String(),
			TriggerSource: req.SourceType(),
			Status:        string(req.Status()),
			Title:         req.Title(),
			LastError:     req.LastError(),
			ErrorCategory: classifyHeartbeatRunError(req.LastError()),
			CreatedAt:     req.CreatedAt().UnixMilli(),
		})
		if len(records) >= limit {
			break
		}
	}
	return records, nil
}

// ListRunsByProject 分页查询项目级心跳执行记录，避免逐心跳查询带来的 N+1 问题。
func (s *HeartbeatTriggerService) ListRunsByProject(ctx context.Context, projectID string, limit, offset int, statuses []string) (*HeartbeatRunPage, error) {
	pid := domain.NewProjectID(projectID)
	project, err := s.projectRepo.FindByID(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to find project %s: %w", projectID, err)
	}
	if project == nil {
		return nil, fmt.Errorf("project %s not found", projectID)
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	filter := domain.RequirementListFilter{
		ProjectID:       &pid,
		RequirementType: string(domain.RequirementTypeHeartbeat),
		Statuses:        statuses,
		SortBy:          "created_at",
		Order:           "DESC",
		Limit:           limit,
		Offset:          offset,
	}
	total, err := s.requirementRepo.Count(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count project heartbeat runs: %w", err)
	}
	reqs, err := s.requirementRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list project heartbeat runs: %w", err)
	}
	records := make([]HeartbeatRunRecord, 0, len(reqs))
	for _, req := range reqs {
		records = append(records, HeartbeatRunRecord{
			RequirementID: req.ID().String(),
			HeartbeatID:   req.HeartbeatID(),
			HeartbeatName: req.HeartbeatName(),
			ProjectID:     projectID,
			TriggerSource: req.SourceType(),
			Status:        string(req.Status()),
			Title:         req.Title(),
			LastError:     req.LastError(),
			ErrorCategory: classifyHeartbeatRunError(req.LastError()),
			CreatedAt:     req.CreatedAt().UnixMilli(),
		})
	}
	return &HeartbeatRunPage{
		Data:   records,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// markRequirementDispatchFailed 在派发失败时把需求标记为 failed，避免形成“无状态需求”。
func (s *HeartbeatTriggerService) markRequirementDispatchFailed(ctx context.Context, requirement *domain.Requirement, reason string) {
	if requirement == nil {
		return
	}
	requirement.MarkFailed(reason)
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		log.Printf("[HEARTBEAT] 保存派发失败状态失败，需求ID=%s，错误=%v", requirement.ID(), err)
	}
}

// ptrProjectID 返回项目ID指针，便于构建过滤条件。
func ptrProjectID(projectID domain.ProjectID) *domain.ProjectID {
	return &projectID
}

// isRequirementFromHeartbeat 判断需求是否由指定心跳触发产生。
func isRequirementFromHeartbeat(req *domain.Requirement, hb *domain.Heartbeat) bool {
	return req.HeartbeatID() == hb.ID().String()
}

// classifyHeartbeatRunError 对执行错误进行粗分类，方便前端快速过滤排障。
func classifyHeartbeatRunError(lastError string) string {
	lower := strings.ToLower(strings.TrimSpace(lastError))
	if lower == "" {
		return "none"
	}
	if strings.Contains(lower, "dispatch_config_missing") || strings.Contains(lower, "channel") || strings.Contains(lower, "session key") {
		return "dispatch_config"
	}
	if strings.Contains(lower, "dispatch_service_unavailable") || strings.Contains(lower, "not available") {
		return "dispatch_service"
	}
	if strings.Contains(lower, "dispatch_failed") {
		return "dispatch_runtime"
	}
	if strings.Contains(lower, "agent") {
		return "agent"
	}
	if strings.Contains(lower, "project") {
		return "project"
	}
	return "other"
}
