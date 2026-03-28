/**
 * TaskSummarizer - 任务总结器
 * 订阅 TaskPendingSummary 事件，当父任务所有子任务完成时，生成总结
 */
package application

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/bus"
	"github.com/weibh/taskmanager/infrastructure/llm"
)

// TaskSummarizer 任务总结器
type TaskSummarizer struct {
	repo      domain.TaskRepository
	executor  *AutoTaskExecutor
	eventBus  *bus.EventBus
}

// NewTaskSummarizer 创建任务总结器
func NewTaskSummarizer(
	repo domain.TaskRepository,
	executor *AutoTaskExecutor,
	eventBus *bus.EventBus,
) *TaskSummarizer {
	return &TaskSummarizer{
		repo:      repo,
		executor:  executor,
		eventBus:  eventBus,
	}
}

// Start 启动总结器，订阅事件
func (s *TaskSummarizer) Start() {
	s.eventBus.Subscribe("TaskPendingSummary", s.HandlePendingSummary)
	log.Println("[TaskSummarizer] 已订阅 TaskPendingSummary 事件")
}

// HandlePendingSummary 处理待总结事件
func (s *TaskSummarizer) HandlePendingSummary(event domain.DomainEvent) {
	pendingEvent, ok := event.(*domain.TaskPendingSummaryEvent)
	if !ok {
		log.Printf("[TaskSummarizer] 事件类型错误: %T", event)
		return
	}

	task := pendingEvent.Task()
	ctx := context.Background()

	// 重新加载任务确保最新状态
	task, err := s.repo.FindByID(ctx, task.ID())
	if err != nil {
		log.Printf("[TaskSummarizer] 重新加载任务失败: %v", err)
		return
	}

	// 再次检查状态，确保是 PendingSummary
	if task.Status() != domain.TaskStatusPendingSummary {
		log.Printf("[TaskSummarizer] 任务状态不是 PendingSummary，跳过: taskID=%s, status=%s", task.ID(), task.Status())
		return
	}

	log.Printf("[TaskSummarizer] 开始处理任务总结: taskID=%s", task.ID())

	// 解析 subtask_records
	pairs, err := domain.ParseTaskResultPairs(task.SubtaskRecords())
	if err != nil {
		log.Printf("[TaskSummarizer] 解析 subtask_records 失败: %v", err)
		s.failTask(task, fmt.Errorf("解析 subtask_records 失败: %w", err))
		return
	}

	if len(pairs) == 0 {
		log.Printf("[TaskSummarizer] subtask_records 为空，直接完成")
		s.completeTask(task, "无子任务结果")
		return
	}

	// 获取 LLM provider
	provider, err := s.executor.llmLookup.getProviderForTask(ctx, task)
	if err != nil {
		log.Printf("[TaskSummarizer] 获取 LLM Provider 失败: %v", err)
		s.failTask(task, fmt.Errorf("获取 LLM Provider 失败: %w", err))
		return
	}

	// 生成总结
	summary, err := s.generateSummary(ctx, task, pairs, provider)
	if err != nil {
		log.Printf("[TaskSummarizer] 生成总结失败: %v", err)
		s.failTask(task, fmt.Errorf("生成总结失败: %w", err))
		return
	}

	// 完成任务
	s.completeTask(task, summary)
}

// generateSummary 调用 LLM 生成总结
func (s *TaskSummarizer) generateSummary(ctx context.Context, task *domain.Task, pairs []domain.TaskResultPair, provider llm.LLMProvider) (string, error) {
	var sb strings.Builder

	sb.WriteString("## 任务总结\n\n")
	sb.WriteString("### 任务要求\n")
	sb.WriteString(task.TaskRequirement())
	sb.WriteString("\n\n")

	if task.AcceptanceCriteria() != "" {
		sb.WriteString("### 验收标准\n")
		sb.WriteString(task.AcceptanceCriteria())
		sb.WriteString("\n\n")
	}

	sb.WriteString("### 子任务完成情况\n")
	for i, pair := range pairs {
		sb.WriteString(fmt.Sprintf("#### %d. %s\n", i+1, pair.TaskName))
		sb.WriteString(fmt.Sprintf("- 要求: %s\n", pair.TaskRequirement))
		if pair.AcceptanceCriteria != "" {
			sb.WriteString(fmt.Sprintf("- 验收标准: %s\n", pair.AcceptanceCriteria))
		}
		sb.WriteString(fmt.Sprintf("- 结论: %s\n", pair.TaskConclusion))
		sb.WriteString(fmt.Sprintf("- 状态: %s\n", pair.Status))
		sb.WriteString("\n")
	}

	sb.WriteString("\n### 综合分析\n")
	sb.WriteString("请根据以上子任务完成情况，生成综合总结。")

	prompt := sb.String()

	// 调用 LLM 生成总结
	summary, err := provider.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("LLM 调用失败: %w", err)
	}

	return summary, nil
}

// completeTask 完成总结，设置结论并调用 Complete
func (s *TaskSummarizer) completeTask(task *domain.Task, conclusion string) {
	ctx := context.Background()

	task.SetTaskConclusion(conclusion)

	if err := task.Complete(); err != nil {
		log.Printf("[TaskSummarizer] 完成任务失败: %v", err)
		return
	}

	if err := s.repo.Save(ctx, task); err != nil {
		log.Printf("[TaskSummarizer] 保存任务失败: %v", err)
		return
	}

	log.Printf("[TaskSummarizer] 任务总结完成: taskID=%s", task.ID())
}

// failTask 总结失败，标记任务失败
func (s *TaskSummarizer) failTask(task *domain.Task, taskErr error) {
	ctx := context.Background()

	task.Fail(taskErr)

	if err := s.repo.Save(ctx, task); err != nil {
		log.Printf("[TaskSummarizer] 保存失败任务失败: %v", err)
		return
	}

	log.Printf("[TaskSummarizer] 任务总结失败: taskID=%s, err=%v", task.ID(), taskErr)
}
