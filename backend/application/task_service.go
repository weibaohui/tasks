/**
 * TaskApplicationService 任务应用服务
 * 协调领域对象完成用例，不包含业务规则
 */
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/bus"
	"go.uber.org/zap"
)

// 错误定义
var (
	ErrTaskNotFound = errors.New("task not found")
)

// CreateTaskCommand 创建任务命令
type CreateTaskCommand struct {
	Name        string
	Description string
	Type        domain.TaskType
	Metadata    map[string]interface{}
	Timeout     int64
	MaxRetries  int
	Priority    int
	ParentID    *domain.TaskID
}

// TaskApplicationService 任务应用服务
type TaskApplicationService struct {
	taskRepo    domain.TaskRepository
	idGenerator domain.IDGenerator
	eventBus    *bus.EventBus
	taskRuntime *TaskRuntime
	logger      *zap.Logger
}

// NewTaskApplicationService 创建任务应用服务
func NewTaskApplicationService(
	taskRepo domain.TaskRepository,
	idGenerator domain.IDGenerator,
	eventBus *bus.EventBus,
	logger *zap.Logger,
) *TaskApplicationService {
	return &TaskApplicationService{
		taskRepo:    taskRepo,
		idGenerator: idGenerator,
		eventBus:    eventBus,
		taskRuntime: NewTaskRuntime(),
		logger:      logger,
	}
}

// CreateTask 创建任务用例
func (s *TaskApplicationService) CreateTask(ctx context.Context, cmd CreateTaskCommand) (*domain.Task, error) {
	// 1. 生成ID
	taskID := domain.NewTaskID(s.idGenerator.Generate())
	spanID := domain.NewSpanID(s.idGenerator.Generate())

	// 2. 确定TraceID
	var traceID domain.TraceID
	if cmd.ParentID != nil {
		parent, err := s.taskRepo.FindByID(ctx, *cmd.ParentID)
		if err != nil {
			return nil, fmt.Errorf("parent task not found: %w", err)
		}
		traceID = parent.TraceID()
	} else {
		traceID = domain.NewTraceID(s.idGenerator.Generate())
	}

	// 3. 创建领域实体
	timeout := time.Duration(cmd.Timeout) * time.Millisecond
	task, err := domain.NewTask(
		taskID,
		traceID,
		spanID,
		cmd.ParentID,
		cmd.Name,
		cmd.Description,
		cmd.Type,
		cmd.Metadata,
		timeout,
		cmd.MaxRetries,
		cmd.Priority,
	)
	if err != nil {
		return nil, err
	}

	// 4. 持久化
	if err := s.taskRepo.Save(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	// 5. 发布领域事件
	for _, event := range task.PopEvents() {
		s.eventBus.Publish(event)
	}

	s.logger.Info("任务创建成功",
		zap.String("taskID", taskID.String()),
		zap.String("traceID", traceID.String()))

	return task, nil
}

// StartTask 启动任务
func (s *TaskApplicationService) StartTask(ctx context.Context, taskID domain.TaskID) error {
	// 1. 获取任务
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return ErrTaskNotFound
	}

	// 2. 启动任务
	if err := task.Start(); err != nil {
		return err
	}

	// 3. 持久化
	if err := s.taskRepo.Save(ctx, task); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	// 4. 发布领域事件
	for _, event := range task.PopEvents() {
		s.eventBus.Publish(event)
	}

	s.logger.Info("任务启动成功", zap.String("taskID", taskID.String()))

	return nil
}

// CancelTask 取消任务
func (s *TaskApplicationService) CancelTask(ctx context.Context, taskID domain.TaskID) error {
	// 1. 获取任务
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return ErrTaskNotFound
	}

	// 2. 取消任务
	if err := task.Cancel(); err != nil {
		return err
	}

	// 3. 持久化
	if err := s.taskRepo.Save(ctx, task); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	// 4. 发布领域事件
	for _, event := range task.PopEvents() {
		s.eventBus.Publish(event)
	}

	s.logger.Info("任务取消成功", zap.String("taskID", taskID.String()))

	return nil
}

// CompleteTask 完成任务
func (s *TaskApplicationService) CompleteTask(ctx context.Context, taskID domain.TaskID, result domain.Result) error {
	// 1. 获取任务
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return ErrTaskNotFound
	}

	// 2. 完成任务
	if err := task.Complete(result); err != nil {
		return err
	}

	// 3. 持久化
	if err := s.taskRepo.Save(ctx, task); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	// 4. 发布领域事件
	for _, event := range task.PopEvents() {
		s.eventBus.Publish(event)
	}

	s.logger.Info("任务完成", zap.String("taskID", taskID.String()))

	return nil
}

// FailTask 标记任务失败
func (s *TaskApplicationService) FailTask(ctx context.Context, taskID domain.TaskID, taskErr error) error {
	// 1. 获取任务
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return ErrTaskNotFound
	}

	// 2. 标记失败
	if err := task.Fail(taskErr); err != nil {
		return err
	}

	// 3. 持久化
	if err := s.taskRepo.Save(ctx, task); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	// 4. 发布领域事件
	for _, event := range task.PopEvents() {
		s.eventBus.Publish(event)
	}

	s.logger.Info("任务失败", zap.String("taskID", taskID.String()), zap.Error(taskErr))

	return nil
}

// UpdateProgress 更新任务进度
func (s *TaskApplicationService) UpdateProgress(ctx context.Context, taskID domain.TaskID, total, current int, stage, detail string) error {
	// 1. 获取任务
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return ErrTaskNotFound
	}

	// 2. 更新进度
	task.UpdateProgress(total, current, stage, detail)

	// 3. 持久化
	if err := s.taskRepo.Save(ctx, task); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	// 4. 发布领域事件
	for _, event := range task.PopEvents() {
		s.eventBus.Publish(event)
	}

	return nil
}
