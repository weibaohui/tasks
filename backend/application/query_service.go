/**
 * QueryService 查询服务 (CQRS)
 * 专门负责查询，不包含业务逻辑
 */
package application

import (
	"context"

	"github.com/weibh/taskmanager/domain"
)

// QueryService 查询服务
type QueryService struct {
	taskRepo domain.TaskRepository
}

// NewQueryService 创建查询服务
func NewQueryService(taskRepo domain.TaskRepository) *QueryService {
	return &QueryService{taskRepo: taskRepo}
}

// GetTask 获取任务详情
func (s *QueryService) GetTask(ctx context.Context, taskID domain.TaskID) (*GetTaskDTO, error) {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return toGetTaskDTO(task), nil
}

// ListAllTasks 获取所有任务
func (s *QueryService) ListAllTasks(ctx context.Context) (*ListTasksDTO, error) {
	tasks, err := s.taskRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	taskDTOs := make([]*GetTaskDTO, len(tasks))
	for i, task := range tasks {
		taskDTOs[i] = toGetTaskDTO(task)
	}

	return &ListTasksDTO{Tasks: taskDTOs, Total: len(taskDTOs)}, nil
}

// ListTasksByTrace 获取任务列表
func (s *QueryService) ListTasksByTrace(ctx context.Context, traceID domain.TraceID) (*ListTasksDTO, error) {
	tasks, err := s.taskRepo.FindByTraceID(ctx, traceID)
	if err != nil {
		return nil, err
	}

	taskDTOs := make([]*GetTaskDTO, len(tasks))
	for i, task := range tasks {
		taskDTOs[i] = toGetTaskDTO(task)
	}

	return &ListTasksDTO{
		Tasks: taskDTOs,
		Total: len(taskDTOs),
	}, nil
}

// taskTreeBuilder 任务树构建器
type taskTreeBuilder struct {
	taskRepo domain.TaskRepository
}

func newTaskTreeBuilder(taskRepo domain.TaskRepository) *taskTreeBuilder {
	return &taskTreeBuilder{taskRepo: taskRepo}
}

// taskTreeNode 任务树节点
type taskTreeNode struct {
	Task     *domain.Task
	Children []*taskTreeNode
}

func (b *taskTreeBuilder) Build(ctx context.Context, traceID domain.TraceID) ([]*taskTreeNode, error) {
	tasks, err := b.taskRepo.FindByTraceID(ctx, traceID)
	if err != nil {
		return nil, err
	}

	taskMap := make(map[domain.TaskID]*domain.Task)
	for _, task := range tasks {
		taskMap[task.ID()] = task
	}

	var roots []*taskTreeNode
	for _, task := range tasks {
		if task.ParentID() == nil {
			roots = append(roots, b.buildNode(task, taskMap))
		}
	}

	return roots, nil
}

func (b *taskTreeBuilder) buildNode(task *domain.Task, taskMap map[domain.TaskID]*domain.Task) *taskTreeNode {
	node := &taskTreeNode{
		Task:     task,
		Children: nil,
	}

	for _, t := range taskMap {
		if t.ParentID() != nil && t.ParentID().Equals(task.ID()) {
			node.Children = append(node.Children, b.buildNode(t, taskMap))
		}
	}

	return node
}

// GetTaskTree 获取任务树
func (s *QueryService) GetTaskTree(ctx context.Context, traceID domain.TraceID) ([]*TaskTreeNodeDTO, error) {
	builder := newTaskTreeBuilder(s.taskRepo)
	nodes, err := builder.Build(ctx, traceID)
	if err != nil {
		return nil, err
	}

	return toTaskTreeDTOs(nodes), nil
}

// toGetTaskDTO 转换为 GetTaskDTO
func toGetTaskDTO(task *domain.Task) *GetTaskDTO {
	dto := &GetTaskDTO{
		ID:          task.ID().String(),
		TraceID:     task.TraceID().String(),
		SpanID:      task.SpanID().String(),
		Name:        task.Name(),
		Description: task.Description(),
		Type:        task.Type().String(),
		Status:      task.Status().String(),
		Metadata:    task.Metadata(),
		Timeout:     int64(task.Timeout()),
		MaxRetries:  task.MaxRetries(),
		Priority:    task.Priority(),
		CreatedAt:   task.CreatedAt().UnixMilli(),
	}

	if task.ParentID() != nil {
		parentID := task.ParentID().String()
		dto.ParentID = &parentID
	}

	if task.StartedAt() != nil {
		startedAt := task.StartedAt().UnixMilli()
		dto.StartedAt = &startedAt
	}

	if task.FinishedAt() != nil {
		finishedAt := task.FinishedAt().UnixMilli()
		dto.FinishedAt = &finishedAt
	}

	progress := task.Progress()
	dto.Progress = ProgressDTO{
		Total:      progress.Total(),
		Current:    progress.Current(),
		Percentage: progress.Percentage(),
		Stage:      progress.Stage(),
		Detail:     progress.Detail(),
		UpdatedAt:  progress.UpdatedAt().UnixMilli(),
	}

	if task.Result() != nil {
		dto.Result = &ResultDTO{
			Data:           task.Result().Data(),
			Message:        task.Result().Message(),
			TaskConclusion: task.TaskConclusion(),
		}
	}

	if task.Error() != nil {
		dto.Error = task.Error().Error()
	}

	return dto
}

// toTaskTreeDTOs 转换为任务树 DTO
func toTaskTreeDTOs(nodes []*taskTreeNode) []*TaskTreeNodeDTO {
	if nodes == nil {
		return nil
	}

	result := make([]*TaskTreeNodeDTO, len(nodes))
	for i, node := range nodes {
		result[i] = &TaskTreeNodeDTO{
			Task:     toGetTaskDTO(node.Task),
			Children: toTaskTreeDTOs(node.Children),
		}
	}

	return result
}
