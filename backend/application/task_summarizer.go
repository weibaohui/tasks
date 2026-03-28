/**
 * TaskSummarizer - 任务总结器
 * 订阅 TaskPendingSummary 事件，从下到上逐层生成任务总结
 * 同一 traceId 的事件串行处理，不同 traceId 可并行
 */
package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/bus"
	"github.com/weibh/taskmanager/infrastructure/llm"
)

// TaskSummarizer 任务总结器
type TaskSummarizer struct {
	repo             domain.TaskRepository
	executor         *AutoTaskExecutor
	eventBus         *bus.EventBus
	providerResolver func(context.Context, *domain.Task) (llm.LLMProvider, error)

	// 按 traceId 分组的事件 channel
	traceChannels map[string]chan *domain.TaskPendingSummaryEvent
	traceStopCh  chan string // 停止某个 traceId 的处理

	mu sync.RWMutex
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
		providerResolver: func(ctx context.Context, task *domain.Task) (llm.LLMProvider, error) {
			if executor == nil || executor.llmLookup == nil {
				return nil, fmt.Errorf("LLM provider resolver 未初始化")
			}
			return executor.llmLookup.getProviderForTask(ctx, task)
		},
		traceChannels: make(map[string]chan *domain.TaskPendingSummaryEvent),
		traceStopCh:   make(chan string, 10),
	}
}

// Start 启动总结器，订阅事件
func (s *TaskSummarizer) Start() {
	// 订阅事件，根据 traceId 分发到不同的 channel
	s.eventBus.Subscribe("TaskPendingSummary", func(event domain.DomainEvent) {
		if pendingEvent, ok := event.(*domain.TaskPendingSummaryEvent); ok {
			s.dispatchByTraceId(pendingEvent)
		}
	})

	log.Println("[TaskSummarizer] 已启动，按 traceId 分组串行处理事件")
}

// dispatchByTraceId 根据 traceId 分发事件到对应 channel
func (s *TaskSummarizer) dispatchByTraceId(event *domain.TaskPendingSummaryEvent) {
	task := event.Task()
	traceId := task.TraceID().String()

	s.mu.Lock()
	ch, exists := s.traceChannels[traceId]
	if !exists {
		// 创建新的 channel 和 goroutine
		ch = make(chan *domain.TaskPendingSummaryEvent, 100)
		s.traceChannels[traceId] = ch
		go s.processTraceEvents(traceId, ch)
		log.Printf("[TaskSummarizer] 为 traceId=%s 启动处理协程", traceId)
	}
	s.mu.Unlock()

	// 发送到对应 traceId 的 channel
	ch <- event
}

// processTraceEvents 处理单个 traceId 的所有事件，串行阻塞
func (s *TaskSummarizer) processTraceEvents(traceId string, ch chan *domain.TaskPendingSummaryEvent) {
	for event := range ch {
		s.handlePendingSummary(event)
	}

	// channel 关闭，清理资源
	s.mu.Lock()
	delete(s.traceChannels, traceId)
	s.mu.Unlock()

	log.Printf("[TaskSummarizer] traceId=%s 的处理协程退出", traceId)
}

// handlePendingSummary 处理待总结事件
func (s *TaskSummarizer) handlePendingSummary(pendingEvent *domain.TaskPendingSummaryEvent) {
	task := pendingEvent.Task()
	ctx := context.Background()

	// 使用 sync.Once 确保 Done channel 只被关闭一次
	var once sync.Once
	closeDone := func() {
		if pendingEvent.Done != nil {
			once.Do(func() {
				close(pendingEvent.Done)
			})
		}
	}

	// 重新加载任务确保最新状态
	task, err := s.repo.FindByID(ctx, task.ID())
	if err != nil {
		log.Printf("[TaskSummarizer] 重新加载任务失败: %v", err)
		closeDone()
		return
	}

	// 再次检查状态，确保是 PendingSummary
	if task.Status() != domain.TaskStatusPendingSummary {
		log.Printf("[TaskSummarizer] 任务状态不是 PendingSummary，跳过: taskID=%s, status=%s", task.ID(), task.Status())
		closeDone()
		return
	}

	log.Printf("[TaskSummarizer] 开始处理任务总结: taskID=%s, traceID=%s",
		task.ID(), task.TraceID())

	hasSubTasks := task.TodoList() != ""
	if hasSubTasks {
		if err := s.collectChildResults(ctx, task); err != nil {
			log.Printf("[TaskSummarizer] 收集子任务成对文档失败: %v", err)
			s.failTask(task, fmt.Errorf("收集子任务成对文档失败: %w", err))
			closeDone()
			return
		}
	}

	// 重新加载任务（collectChildResults 可能更新了内容）
	task, _ = s.repo.FindByID(ctx, task.ID())

	// 生成总结
	var summary string
	if hasSubTasks {
		pairs, err := domain.ParseTaskResultPairs(task.SubtaskRecords())
		if err != nil {
			log.Printf("[TaskSummarizer] 解析子任务成对文档失败: %v", err)
			s.failTask(task, fmt.Errorf("解析子任务成对文档失败: %w", err))
			closeDone()
			return
		}
		if len(pairs) == 0 {
			s.failTask(task, fmt.Errorf("存在子任务但未收集到可总结结果"))
			closeDone()
			return
		}

		provider, err := s.providerResolver(ctx, task)
		if err != nil {
			log.Printf("[TaskSummarizer] 获取 LLM Provider 失败: %v", err)
			s.failTask(task, fmt.Errorf("获取 LLM Provider 失败: %w", err))
			closeDone()
			return
		}

		summary, err = s.generateSummary(ctx, task, pairs, provider)
		if err != nil {
			log.Printf("[TaskSummarizer] 生成总结失败: %v", err)
			s.failTask(task, fmt.Errorf("生成总结失败: %w", err))
			closeDone()
			return
		}
	} else {
		summary = task.TaskConclusion()
		if summary == "" {
			summary = "任务完成"
		}
	}

	// 完成总结
	s.completeTaskAndNotifyParent(ctx, task, summary)

	// 处理完成，关闭 Done channel
	closeDone()
}

// collectChildResults 收集所有子任务的成对文档到当前任务的 subtask_records
func (s *TaskSummarizer) collectChildResults(ctx context.Context, task *domain.Task) error {
	todoListStr := task.TodoList()
	if todoListStr == "" {
		return nil
	}

	var todoList TodoList
	if err := json.Unmarshal([]byte(todoListStr), &todoList); err != nil {
		return fmt.Errorf("解析 todoList 失败: %w", err)
	}

	log.Printf("[TaskSummarizer] 收集子任务成对文档: taskID=%s, 子任务数=%d", task.ID(), len(todoList.Items))

	pairs := make([]domain.TaskResultPair, 0, len(todoList.Items))
	for _, item := range todoList.Items {
		subTask, err := s.repo.FindByID(ctx, domain.NewTaskID(item.SubTaskID))
		if err != nil || subTask == nil {
			continue
		}

		// 子任务必须已完成
		if subTask.Status() != domain.TaskStatusCompleted {
			log.Printf("[TaskSummarizer] 子任务未完成，跳过: subTaskID=%s, status=%s",
				item.SubTaskID, subTask.Status())
			continue
		}

		completedAt := subTask.FinishedAt()
		if completedAt == nil {
			now := time.Now()
			completedAt = &now
		}
		pair := domain.TaskResultPair{
			TaskID:             subTask.ID().String(),
			TaskName:           subTask.Name(),
			TaskRequirement:    subTask.TaskRequirement(),
			AcceptanceCriteria: subTask.AcceptanceCriteria(),
			TaskConclusion:     subTask.TaskConclusion(),
			CompletedAt:        completedAt,
			Status:             subTask.Status(),
		}
		pairs = append(pairs, pair)
		log.Printf("[TaskSummarizer] 收集子任务成对文档: subTaskID=%s, conclusion='%.50s'",
			item.SubTaskID, subTask.TaskConclusion())
	}

	records, err := buildTaskResultRecords(pairs)
	if err != nil {
		return fmt.Errorf("构建子任务成对文档失败: %w", err)
	}

	task.SetSubtaskRecords(records)
	if err := s.repo.Save(ctx, task); err != nil {
		return fmt.Errorf("保存子任务成对文档失败: %w", err)
	}

	log.Printf("[TaskSummarizer] subtask_records 更新完成: taskID=%s, len=%d",
		task.ID(), len(task.SubtaskRecords()))
	return nil
}

// completeTaskAndNotifyParent 完成总结并通知父任务
func (s *TaskSummarizer) completeTaskAndNotifyParent(ctx context.Context, task *domain.Task, summary string) {
	// 设置总结
	task.SetTaskConclusion(summary)

	// 完成当前任务
	if err := task.Complete(); err != nil {
		log.Printf("[TaskSummarizer] 完成任务失败: %v", err)
		return
	}
	s.repo.Save(ctx, task)
	log.Printf("[TaskSummarizer] 任务完成: taskID=%s", task.ID())

	// 通知父任务收集子任务结果
	parentID := task.ParentID()
	if parentID != nil {
		s.notifyParentToCollect(ctx, task, parentID)
	}
}

// notifyParentToCollect 通知父任务收集子任务结果
func (s *TaskSummarizer) notifyParentToCollect(ctx context.Context, child *domain.Task, parentID *domain.TaskID) {
	parent, err := s.repo.FindByID(ctx, *parentID)
	if err != nil || parent == nil {
		log.Printf("[TaskSummarizer] 获取父任务失败: %v", err)
		return
	}

	// 将子任务的成对文档追加到父任务的 subtask_records
	childConclusion := child.TaskConclusion()
	if childConclusion == "" {
		childConclusion = "任务完成"
	}

	completedAt := time.Now()
	pair := domain.TaskResultPair{
		TaskID:             child.ID().String(),
		TaskName:           child.Name(),
		TaskRequirement:    child.TaskRequirement(),
		AcceptanceCriteria: child.AcceptanceCriteria(),
		TaskConclusion:     childConclusion,
		CompletedAt:        &completedAt,
		Status:             child.Status(),
	}

	newRecords, err := upsertTaskResultPair(parent.SubtaskRecords(), pair)
	if err != nil {
		log.Printf("[TaskSummarizer] 追加到父任务 subtask_records 失败: %v", err)
		return
	}

	parent.SetSubtaskRecords(newRecords)
	s.repo.Save(ctx, parent)

	log.Printf("[TaskSummarizer] 子任务结果已添加到父任务: childID=%s, parentID=%s, subtaskRecords len=%d",
		child.ID(), parent.ID(), len(newRecords))

	// 检查父任务是否所有子任务都完成了
	s.checkAndTriggerParentSummary(ctx, parent)
}

// checkAndTriggerParentSummary 检查父任务是否所有子任务都完成，如果是，触发总结
func (s *TaskSummarizer) checkAndTriggerParentSummary(ctx context.Context, parent *domain.Task) {
	todoListStr := parent.TodoList()
	if todoListStr == "" {
		return
	}

	var todoList TodoList
	if err := json.Unmarshal([]byte(todoListStr), &todoList); err != nil {
		return
	}

	// 检查所有子任务是否都完成了
	allCompleted := true
	for _, item := range todoList.Items {
		subTask, err := s.repo.FindByID(ctx, domain.NewTaskID(item.SubTaskID))
		if err != nil || subTask == nil {
			allCompleted = false
			break
		}
		if subTask.Status() != domain.TaskStatusCompleted {
			allCompleted = false
			break
		}
	}

	log.Printf("[TaskSummarizer] 检查父任务子任务完成状态: parentID=%s, allCompleted=%v", parent.ID(), allCompleted)

	if allCompleted && parent.Status() == domain.TaskStatusRunning {
		// 所有子任务都完成了，父任务进入 PendingSummary
		if err := parent.PendingSummary(); err != nil {
			log.Printf("[TaskSummarizer] 父任务进入 PendingSummary 失败: %v", err)
			return
		}
		s.repo.Save(ctx, parent)

		// 发布事件触发总结（发送到同一 traceId 的 channel，串行处理）
		evt := domain.NewTaskPendingSummaryEvent(parent)
		s.dispatchByTraceId(evt)

		log.Printf("[TaskSummarizer] 父任务进入 PendingSummary: parentID=%s", parent.ID())
	}
}

func upsertTaskResultPair(existingRecords string, pair domain.TaskResultPair) (string, error) {
	pairs, err := domain.ParseTaskResultPairs(existingRecords)
	if err != nil {
		return "", err
	}

	replaced := false
	for i := range pairs {
		if isSameTaskPair(pairs[i], pair) {
			pairs[i] = pair
			replaced = true
			break
		}
	}
	if !replaced {
		pairs = append(pairs, pair)
	}

	return buildTaskResultRecords(pairs)
}

func isSameTaskPair(left domain.TaskResultPair, right domain.TaskResultPair) bool {
	if left.TaskID != "" && right.TaskID != "" {
		return left.TaskID == right.TaskID
	}
	return left.TaskName == right.TaskName &&
		left.TaskRequirement == right.TaskRequirement &&
		left.AcceptanceCriteria == right.AcceptanceCriteria
}

func buildTaskResultRecords(pairs []domain.TaskResultPair) (string, error) {
	records := ""
	for _, pair := range pairs {
		var err error
		records, err = domain.AppendTaskResultPair(records, pair)
		if err != nil {
			return "", err
		}
	}
	return records, nil
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

	log.Printf("[TaskSummarizer] generateSummary result len=%d, content='%.200s'",
		len(summary), summary)

	return summary, nil
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
