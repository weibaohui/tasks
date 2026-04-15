package application

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/domain/statemachine"
)

var (
	ErrRequirementNotFound = errors.New("requirement not found")
)

type CreateRequirementCommand struct {
	ProjectID          domain.ProjectID
	Title              string
	Description        string
	AcceptanceCriteria string
	TempWorkspaceRoot  string
	RequirementType    string // 可选，默认为 "normal"
}

type UpdateRequirementCommand struct {
	ID                 domain.RequirementID
	Title              *string
	Description        *string
	AcceptanceCriteria *string
	TempWorkspaceRoot  *string
	RequirementType    *string
}

type UpdateRequirementStatusCommand struct {
	ID        domain.RequirementID
	NewStatus string
}

type ReportRequirementPRCommand struct {
	ID domain.RequirementID
}

type RedispatchRequirementCommand struct {
	ID domain.RequirementID
}

type RequirementApplicationService struct {
	requirementRepo   domain.RequirementRepository
	projectRepo       domain.ProjectRepository
	idGenerator       domain.IDGenerator
	replicaCleanupSvc domain.ReplicaCleanupService
	stateMachineRepo  statemachine.Repository
}

func NewRequirementApplicationService(
	requirementRepo domain.RequirementRepository,
	projectRepo domain.ProjectRepository,
	idGenerator domain.IDGenerator,
	replicaCleanupSvc domain.ReplicaCleanupService,
	stateMachineRepo statemachine.Repository,
) *RequirementApplicationService {
	return &RequirementApplicationService{
		requirementRepo:   requirementRepo,
		projectRepo:       projectRepo,
		idGenerator:       idGenerator,
		replicaCleanupSvc: replicaCleanupSvc,
		stateMachineRepo:  stateMachineRepo,
	}
}

// recordTransitionIfNeeded 如果状态发生变化，记录转换日志
func (s *RequirementApplicationService) recordTransitionIfNeeded(ctx context.Context, requirement *domain.Requirement, trigger, triggeredBy, remark string) {
	if s.stateMachineRepo == nil {
		return
	}
	previousStatus := requirement.PreviousStatus()
	currentStatus := requirement.Status()
	if previousStatus != "" && previousStatus != currentStatus {
		logEntry := statemachine.NewTransitionLog(
			requirement.ID().String(),
			string(previousStatus),
			string(currentStatus),
			trigger,
			triggeredBy,
			remark,
		)
		if errSave := s.stateMachineRepo.SaveTransitionLog(ctx, logEntry); errSave != nil {
			log.Printf("stateMachineRepo.SaveTransitionLog failed: %v", errSave)
		}
	}
}

func (s *RequirementApplicationService) CreateRequirement(ctx context.Context, cmd CreateRequirementCommand) (*domain.Requirement, error) {
	project, err := s.projectRepo.FindByID(ctx, cmd.ProjectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, ErrProjectNotFound
	}
	requirement, err := domain.NewRequirement(
		domain.NewRequirementID(s.idGenerator.Generate()),
		cmd.ProjectID,
		cmd.Title,
		cmd.Description,
		cmd.AcceptanceCriteria,
		cmd.TempWorkspaceRoot,
	)
	if err != nil {
		return nil, err
	}
	// 设置需求类型（默认为 normal）
	if cmd.RequirementType != "" {
		requirement.SetRequirementType(domain.RequirementType(cmd.RequirementType))
	}
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}
	return requirement, nil
}

func (s *RequirementApplicationService) GetRequirement(ctx context.Context, id domain.RequirementID) (*domain.Requirement, error) {
	requirement, err := s.requirementRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if requirement == nil {
		return nil, ErrRequirementNotFound
	}
	return requirement, nil
}

func (s *RequirementApplicationService) ListRequirements(ctx context.Context, projectID *domain.ProjectID) ([]*domain.Requirement, error) {
	if projectID != nil {
		return s.requirementRepo.FindByProjectID(ctx, *projectID)
	}
	return s.requirementRepo.FindAll(ctx)
}

// ListRequirementsQuery 分页查询需求参数
type ListRequirementsQuery struct {
	ProjectID *domain.ProjectID
	Statuses  []string
	Limit     int
	Offset    int
}

// ListRequirementsPaginated 分页查询需求列表
func (s *RequirementApplicationService) ListRequirementsPaginated(ctx context.Context, query ListRequirementsQuery) ([]*domain.Requirement, int, error) {
	filter := domain.RequirementListFilter{
		ProjectID: query.ProjectID,
		Statuses:  query.Statuses,
		Limit:     query.Limit,
		Offset:    query.Offset,
	}
	items, err := s.requirementRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.requirementRepo.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *RequirementApplicationService) UpdateRequirement(ctx context.Context, cmd UpdateRequirementCommand) (*domain.Requirement, error) {
	requirement, err := s.requirementRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if requirement == nil {
		return nil, ErrRequirementNotFound
	}

	if cmd.Title != nil || cmd.Description != nil || cmd.AcceptanceCriteria != nil || cmd.TempWorkspaceRoot != nil || cmd.RequirementType != nil {
		title := requirement.Title()
		description := requirement.Description()
		acceptanceCriteria := requirement.AcceptanceCriteria()
		tempWorkspaceRoot := requirement.TempWorkspaceRoot()

		if cmd.Title != nil {
			title = *cmd.Title
		}
		if cmd.Description != nil {
			description = *cmd.Description
		}
		if cmd.AcceptanceCriteria != nil {
			acceptanceCriteria = *cmd.AcceptanceCriteria
		}
		if cmd.TempWorkspaceRoot != nil {
			tempWorkspaceRoot = *cmd.TempWorkspaceRoot
		}

		if err := requirement.UpdateContent(title, description, acceptanceCriteria, tempWorkspaceRoot); err != nil {
			return nil, err
		}
	}

	// 更新需求类型
	if cmd.RequirementType != nil {
		requirement.SetRequirementType(domain.RequirementType(*cmd.RequirementType))
	}
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}
	return requirement, nil
}

// UpdateRequirementStatus 直接更新需求状态（用于修复异常状态）
func (s *RequirementApplicationService) UpdateRequirementStatus(ctx context.Context, cmd UpdateRequirementStatusCommand) (*domain.Requirement, error) {
	requirement, err := s.requirementRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if requirement == nil {
		return nil, ErrRequirementNotFound
	}

	// 记录状态转换日志
	fromStatus := requirement.Status()
	toStatus := cmd.NewStatus
	if string(fromStatus) != toStatus && s.stateMachineRepo != nil {
		logEntry := statemachine.NewTransitionLog(
			requirement.ID().String(),
			string(fromStatus),
			toStatus,
			"manual_update",
			"system",
			"手动修改状态",
		)
		if errSave := s.stateMachineRepo.SaveTransitionLog(ctx, logEntry); errSave != nil {
			log.Printf("stateMachineRepo.SaveTransitionLog failed: %v", errSave)
		}
	}

	// 使用 SyncStatusFromStateMachine 直接设置状态
	requirement.SyncStatusFromStateMachine(toStatus)

	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}
	return requirement, nil
}

// GetRequirementTransitionHistory 获取需求的状态转换历史
func (s *RequirementApplicationService) GetRequirementTransitionHistory(ctx context.Context, requirementID domain.RequirementID) ([]*statemachine.TransitionLog, error) {
	if s.stateMachineRepo == nil {
		return nil, nil
	}
	return s.stateMachineRepo.ListTransitionLogs(ctx, requirementID.String())
}

// GetStatusStats 获取所有状态的统计数据（动态从数据库提取）
func (s *RequirementApplicationService) GetStatusStats(ctx context.Context, projectID *domain.ProjectID) ([]domain.StatusStat, error) {
	return s.requirementRepo.GetStatusStats(ctx, projectID)
}

func (s *RequirementApplicationService) ReportRequirementPROpened(ctx context.Context, cmd ReportRequirementPRCommand) (*domain.Requirement, error) {
	fmt.Printf("[DEBUG] ReportRequirementPROpened CALLED: id=%s\n", cmd.ID)
	requirement, err := s.requirementRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if requirement == nil {
		return nil, ErrRequirementNotFound
	}

	// 先清理分身和工作区（应用层职责）
	if s.replicaCleanupSvc != nil {
		if err := s.replicaCleanupSvc.CleanupReplica(ctx, requirement.ReplicaAgentCode(), requirement.WorkspacePath()); err != nil {
			log.Printf("failed to cleanup replica for requirement %s: %v", requirement.ID().String(), err)
		}
	}

	requirement.MarkPROpened()
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}
	return requirement, nil
}

// RedispatchRequirement 重新派发需求（重置当前需求状态）
func (s *RequirementApplicationService) RedispatchRequirement(ctx context.Context, cmd RedispatchRequirementCommand) (*domain.Requirement, error) {
	requirement, err := s.requirementRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if requirement == nil {
		return nil, ErrRequirementNotFound
	}

	// 记录状态转换日志
	if s.stateMachineRepo != nil {
		fromStatus := string(requirement.Status())
		toStatus := "todo"
		if fromStatus != toStatus {
			logEntry := statemachine.NewTransitionLog(
				requirement.ID().String(),
				fromStatus,
				toStatus,
				"redispatch",
				"system",
				"重新派发",
			)
			if errSave := s.stateMachineRepo.SaveTransitionLog(ctx, logEntry); errSave != nil {
				log.Printf("stateMachineRepo.SaveTransitionLog failed: %v", errSave)
			}
		}
	}

	if err := requirement.Redispatch(); err != nil {
		return nil, err
	}
	if err := s.requirementRepo.Save(ctx, requirement); err != nil {
		return nil, err
	}
	return requirement, nil
}

// CopyAndDispatchRequirementCommand 复制并派发需求命令
type CopyAndDispatchRequirementCommand struct {
	ID domain.RequirementID
}

// DeleteRequirementCommand 删除需求命令
type DeleteRequirementCommand struct {
	ID domain.RequirementID
}

// BatchDeleteRequirementsCommand 批量删除需求命令
type BatchDeleteRequirementsCommand struct {
	IDs []domain.RequirementID
}

// CopyAndDispatchRequirement 复制需求并派发新副本
// 创建一个新需求，复制原需求内容，标题增加"[重新派发]"标记，然后派发
func (s *RequirementApplicationService) CopyAndDispatchRequirement(ctx context.Context, cmd CopyAndDispatchRequirementCommand, dispatchService *RequirementDispatchService) (*domain.Requirement, error) {
	// 1. 查找原需求
	originalReq, err := s.requirementRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if originalReq == nil {
		return nil, ErrRequirementNotFound
	}

	// 2. 查找项目获取派发配置
	project, err := s.projectRepo.FindByID(ctx, originalReq.ProjectID())
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, ErrProjectNotFound
	}

	// 3. 创建新需求（使用领域工厂方法）
	newID := domain.NewRequirementID(s.idGenerator.Generate())
	newReq, err := domain.NewRedispatchedRequirement(newID, originalReq)
	if err != nil {
		return nil, err
	}

	// 4. 保存新需求
	if err := s.requirementRepo.Save(ctx, newReq); err != nil {
		return nil, err
	}

	// 5. 派发新需求
	agentCode := originalReq.AssigneeAgentCode()
	if agentCode == "" {
		agentCode = originalReq.ReplicaAgentCode()
	}
	channelCode := project.DispatchChannelCode()
	sessionKey := project.DispatchSessionKey()

	if channelCode == "" {
		channelCode = "feishu"
	}

	_, err = dispatchService.DispatchRequirement(ctx, DispatchRequirementCommand{
		RequirementID: newReq.ID(),
		AgentCode:     agentCode,
		ChannelCode:   channelCode,
		SessionKey:    sessionKey,
	})
	if err != nil {
		// 派发失败，删除已保存的需求以保持一致性
		if errSave := s.requirementRepo.Delete(ctx, newReq.ID()); errSave != nil {
			log.Printf("requirementRepo.Delete failed: %v", errSave)
		}
		log.Printf("requirement creation failed, cleaned up %s: %v", newReq.ID().String(), err)
		return nil, err
	}

	// 6. 返回新需求（重新从数据库获取以获得派发后的状态）
	newReq, err = s.requirementRepo.FindByID(ctx, newReq.ID())
	if err != nil {
		return nil, err
	}

	return newReq, nil
}

// DeleteRequirement 删除需求
func (s *RequirementApplicationService) DeleteRequirement(ctx context.Context, cmd DeleteRequirementCommand) error {
	requirement, err := s.requirementRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return err
	}
	if requirement == nil {
		return ErrRequirementNotFound
	}
	return s.requirementRepo.Delete(ctx, cmd.ID)
}

// BatchDeleteRequirements 批量删除需求
func (s *RequirementApplicationService) BatchDeleteRequirements(ctx context.Context, cmd BatchDeleteRequirementsCommand) error {
	var lastErr error
	for _, id := range cmd.IDs {
		if err := s.DeleteRequirement(ctx, DeleteRequirementCommand{ID: id}); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
