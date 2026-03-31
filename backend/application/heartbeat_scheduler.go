package application

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/weibh/taskmanager/domain"
	channelBus "github.com/weibh/taskmanager/pkg/bus"
)

// HeartbeatScheduler 心跳调度器
type HeartbeatScheduler struct {
	cron                     *cron.Cron
	projectRepo              domain.ProjectRepository
	agentRepo                domain.AgentRepository
	requirementRepo          domain.RequirementRepository
	idGenerator              domain.IDGenerator
	inboundPublisher         interface {
		PublishInbound(msg *channelBus.InboundMessage)
	}
	requirementDispatchService *RequirementDispatchService
}

// NewHeartbeatScheduler 创建心跳调度器
func NewHeartbeatScheduler(
	projectRepo domain.ProjectRepository,
	agentRepo domain.AgentRepository,
	requirementRepo domain.RequirementRepository,
	idGenerator domain.IDGenerator,
	inboundPublisher interface {
		PublishInbound(msg *channelBus.InboundMessage)
	},
	requirementDispatchService *RequirementDispatchService,
) *HeartbeatScheduler {
	return &HeartbeatScheduler{
		cron:                      cron.New(cron.WithSeconds()),
		projectRepo:               projectRepo,
		agentRepo:                 agentRepo,
		requirementRepo:           requirementRepo,
		idGenerator:               idGenerator,
		inboundPublisher:          inboundPublisher,
		requirementDispatchService: requirementDispatchService,
	}
}

// Start 启动调度器
func (s *HeartbeatScheduler) Start(ctx context.Context) error {
	// 加载所有启用心跳的项目
	projects, err := s.projectRepo.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	for _, project := range projects {
		if project.HeartbeatEnabled() && project.HeartbeatAgentID() != "" {
			if err := s.scheduleProject(project); err != nil {
				log.Printf("failed to schedule heartbeat for project %s: %v", project.ID(), err)
			} else {
				log.Printf("heartbeat scheduled for project %s (interval: %d minutes)",
					project.Name(), project.HeartbeatIntervalMinutes())
			}
		}
	}

	s.cron.Start()
	log.Printf("heartbeat scheduler started with %d projects", len(projects))
	return nil
}

// Stop 停止调度器
func (s *HeartbeatScheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Printf("heartbeat scheduler stopped")
}

// scheduleProject 为项目调度心跳
func (s *HeartbeatScheduler) scheduleProject(project *domain.Project) error {
	// 将分钟转换为 cron 表达式（每N分钟执行一次）
	interval := project.HeartbeatIntervalMinutes()
	if interval < 1 {
		interval = 60
	}
	cronExpr := fmt.Sprintf("0 */%d * * * *", interval) // 每N分钟的 cron 表达式

	projectID := project.ID()
	_, err := s.cron.AddFunc(cronExpr, func() {
		s.executeHeartbeat(projectID.String())
	})
	return err
}

// executeHeartbeat 执行心跳
func (s *HeartbeatScheduler) executeHeartbeat(projectID string) {
	ctx := context.Background()

	project, err := s.projectRepo.FindByID(ctx, domain.NewProjectID(projectID))
	if err != nil || project == nil {
		log.Printf("heartbeat: failed to find project %s: %v", projectID, err)
		return
	}

	if !project.HeartbeatEnabled() {
		return
	}

	log.Printf("[HEARTBEAT] executing heartbeat for project %s", project.Name())

	// 查找 agent
	agent, err := s.agentRepo.FindByID(ctx, domain.NewAgentID(project.HeartbeatAgentID()))
	if err != nil || agent == nil {
		log.Printf("heartbeat: failed to find agent %s: %v", project.HeartbeatAgentID(), err)
		return
	}

	// 替换模板变量
	prompt := s.renderTemplate(project.HeartbeatMDContent(), project)

	// 创建心跳需求
	requirement, err := domain.NewRequirement(
		domain.NewRequirementID(s.idGenerator.Generate()),
		project.ID(),
		fmt.Sprintf("[心跳] %s - %s", project.Name(), time.Now().Format("2006-01-02 15:04")),
		prompt,
		"心跳自动生成",
		"",
	)
	if err != nil {
		log.Printf("heartbeat: failed to create requirement: %v", err)
		return
	}

	// 保存需求
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		log.Printf("heartbeat: failed to save requirement: %v", err)
		return
	}

	log.Printf("[HEARTBEAT] created requirement %s for project %s, dispatching...", requirement.ID(), project.Name())

	// 直接派发心跳需求
	if s.requirementDispatchService != nil {
		// 使用项目配置的默认 session_key 或者心跳专用渠道
		sessionKey := "feishu:ou_df798fe15d056000143691af8c1cdb55"
		result, err := s.requirementDispatchService.DispatchRequirement(ctx, DispatchRequirementCommand{
			RequirementID: requirement.ID(),
			AgentID:       domain.NewAgentID(project.HeartbeatAgentID()),
			ChannelCode:   "feishu",
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

// renderTemplate 渲染模板变量
func (s *HeartbeatScheduler) renderTemplate(template string, project *domain.Project) string {
	result := template
	result = strings.ReplaceAll(result, "${project.id}", project.ID().String())
	result = strings.ReplaceAll(result, "${project.name}", project.Name())
	result = strings.ReplaceAll(result, "${project.git_repo_url}", project.GitRepoURL())
	result = strings.ReplaceAll(result, "${project.default_branch}", project.DefaultBranch())
	result = strings.ReplaceAll(result, "${timestamp}", time.Now().Format("2006-01-02 15:04:05"))
	return result
}

// UpdateProjectHeartbeat 更新项目心跳配置
func (s *HeartbeatScheduler) UpdateProjectHeartbeat(ctx context.Context, projectID string, enabled bool, intervalMinutes int, mdContent, agentID string) error {
	project, err := s.projectRepo.FindByID(ctx, domain.NewProjectID(projectID))
	if err != nil || project == nil {
		return fmt.Errorf("project not found")
	}

	project.UpdateHeartbeatConfig(enabled, intervalMinutes, mdContent, agentID)
	return s.projectRepo.Save(ctx, project)
}

// RefreshSchedule 刷新调度（当项目心跳配置变更时调用）
func (s *HeartbeatScheduler) RefreshSchedule(ctx context.Context) error {
	// 停止所有现有任务
	s.cron.Stop()

	// 重新加载并调度
	projects, err := s.projectRepo.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	for _, project := range projects {
		if project.HeartbeatEnabled() && project.HeartbeatAgentID() != "" {
			if err := s.scheduleProject(project); err != nil {
				log.Printf("failed to schedule heartbeat for project %s: %v", project.ID(), err)
			}
		}
	}

	s.cron.Start()
	return nil
}
