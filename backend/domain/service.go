/**
 * 领域服务
 * 包含 TaskTreeBuilder 等领域服务
 */
package domain

import (
	"context"
)

// TaskTreeBuilder 任务树构建器
type TaskTreeBuilder struct {
	taskRepo TaskRepository
}

func NewTaskTreeBuilder(taskRepo TaskRepository) *TaskTreeBuilder {
	return &TaskTreeBuilder{taskRepo: taskRepo}
}

// TaskTreeNode 任务树节点
type TaskTreeNode struct {
	Task     *Task
	Children []*TaskTreeNode
}

// Build 构建任务树
func (b *TaskTreeBuilder) Build(ctx context.Context, traceID TraceID) ([]*TaskTreeNode, error) {
	// 查找所有任务
	tasks, err := b.taskRepo.FindByTraceID(ctx, traceID)
	if err != nil {
		return nil, err
	}

	// 构建任务映射
	taskMap := make(map[TaskID]*Task)
	for _, task := range tasks {
		taskMap[task.ID()] = task
	}

	// 构建树
	var roots []*TaskTreeNode
	for _, task := range tasks {
		if task.ParentID() == nil {
			roots = append(roots, b.buildNode(task, taskMap))
		}
	}

	return roots, nil
}

func (b *TaskTreeBuilder) buildNode(task *Task, taskMap map[TaskID]*Task) *TaskTreeNode {
	node := &TaskTreeNode{
		Task:     task,
		Children: nil,
	}

	// 查找子任务
	for _, t := range taskMap {
		if t.ParentID() != nil && t.ParentID().Equals(task.ID()) {
			node.Children = append(node.Children, b.buildNode(t, taskMap))
		}
	}

	return node
}
