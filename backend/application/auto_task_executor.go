/**
 * AutoTaskExecutor - 自动任务执行器
 * 支持子任务分发和 Todo 列表管理
 */
package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/llm"
	"github.com/weibh/taskmanager/infrastructure/utils"
)

const MaxTaskDepth = 4

// inheritContextFromTask 从父任务继承上下文信息（agent_code, user_code, channel_code, session_key）
func inheritContextFromTask(parent *domain.Task, metadata map[string]interface{}) {
	if parent == nil || parent.Metadata() == nil {
		return
	}
	parentMeta := parent.Metadata()

	// 继承上下文字段
	if v, ok := parentMeta["agent_code"].(string); ok && v != "" {
		metadata["agent_code"] = v
	}
	if v, ok := parentMeta["user_code"].(string); ok && v != "" {
		metadata["user_code"] = v
	}
	if v, ok := parentMeta["channel_code"].(string); ok && v != "" {
		metadata["channel_code"] = v
	}
	if v, ok := parentMeta["session_key"].(string); ok && v != "" {
		metadata["session_key"] = v
	}
}

// TaskExecutionSummary 单个任务的执行摘要
type TaskExecutionSummary struct {
	TaskID      string `json:"task_id"`
	SpanID      string `json:"span_id"`
	Goal        string `json:"goal"`   // 目标是什么
	Result      string `json:"result"` // 结果是什么
	Stage       string `json:"stage"`
	CompletedAt int64  `json:"completed_at"`
	Status      string `json:"status"`
}

type AutoTaskExecutor struct {
	repo       domain.TaskRepository
	eventBus   interface{ Publish(domain.DomainEvent) }
	registry   *TaskRegistry
	workerPool interface{ Submit(*domain.Task) bool }
	llmLookup *taskLLMProvider
}

func NewAutoTaskExecutor(
	repo domain.TaskRepository,
	eventBus interface{ Publish(domain.DomainEvent) },
	registry *TaskRegistry,
	workerPool interface{ Submit(*domain.Task) bool },
) *AutoTaskExecutor {
	return &AutoTaskExecutor{
		repo:       repo,
		eventBus:   eventBus,
		registry:   registry,
		workerPool: workerPool,
	}
}

// SetRepositories 设置必要的仓库用于动态 LLM 查找
func (e *AutoTaskExecutor) SetRepositories(
	agentRepo domain.AgentRepository,
	providerRepo domain.LLMProviderRepository,
	channelRepo domain.ChannelRepository,
	factory domain.LLMProviderFactory,
) {
	e.llmLookup = newTaskLLMProvider(agentRepo, providerRepo, channelRepo, factory)
}

// SetLLMProvider 设置 LLM Provider (已废弃，使用 SetRepositories 代替)
func (e *AutoTaskExecutor) SetLLMProvider(provider llm.LLMProvider) {
	// 已废弃，保留此方法避免编译错误
}

// getLLMProviderForTask 根据任务元数据获取 LLM Provider
func (e *AutoTaskExecutor) getLLMProviderForTask(ctx context.Context, task *domain.Task) (llm.LLMProvider, error) {
	if e.llmLookup == nil {
		return nil, fmt.Errorf("LLM provider lookup not initialized")
	}
	return e.llmLookup.getProviderForTask(ctx, task)
}

func (e *AutoTaskExecutor) ExecuteAutoTask(ctx context.Context, task *domain.Task) error {
	taskID := task.ID().String()
	traceID := task.TraceID().String()
	spanID := task.SpanID().String()

	// 从 metadata 获取当前深度
	currentDepth := 1
	if task.Metadata() != nil {
		if depthStr, ok := task.Metadata()["depth"].(string); ok {
			if depth, err := strconv.Atoi(depthStr); err == nil {
				currentDepth = depth + 1
			}
		}
	}

	log.Printf("[AutoExecutor] 执行任务: %s, spanID: %s, depth: %d/%d", taskID, spanID, currentDepth, MaxTaskDepth)

	todoList := NewTodoList(taskID)

	e.updateProgress(task, 0, "初始化", fmt.Sprintf("开始执行任务，深度 %d", currentDepth))
	time.Sleep(AgentInitDelay)

	// 检查是否达到最大深度
	if currentDepth >= MaxTaskDepth {
		e.updateProgress(task, 100, "完成", "达到最大深度，直接完成")
		return e.finishTask(task)
	}

	// 使用 LLM 动态生成子任务
	subTaskIDs := make([]string, 0)
	hasSubTasks := false
	isAgentTask := task.Type() == domain.TaskTypeAgent

	// 动态获取 LLM Provider
	llmProvider, err := e.getLLMProviderForTask(ctx, task)
	if err != nil {
		log.Printf("[AutoExecutor] 获取 LLM Provider 失败: %v", err)
		if isAgentTask {
			e.updateProgress(task, 5, "LLM 未配置", "Agent 模式未配置 LLM，任务终止")
			return e.failTask(task, fmt.Errorf("Agent 模式未配置 LLM: %w", err))
		}
		hasSubTasks = false
	} else if llmProvider != nil {
		e.updateProgress(task, 5, "LLM 规划中", "正在调用 LLM 生成子任务...")

		plan, err := llmProvider.GenerateSubTasks(
			ctx,
			task.Name(),
			task.Description(),
			currentDepth,
			MaxTaskDepth,
		)
		if err != nil {
			log.Printf("[AutoExecutor] LLM 生成子任务失败: %v", err)
			if isAgentTask {
				e.updateProgress(task, 10, "LLM 规划失败", "Agent 模式调用 LLM 失败，任务终止")
				return e.failTask(task, fmt.Errorf("Agent 模式调用 LLM 失败: %w", err))
			}
			log.Printf("[AutoExecutor] 非 Agent 任务回退到默认子任务")
			hasSubTasks = false
		} else if len(plan.SubTasks) > 0 {
			hasSubTasks = true
			e.updateProgress(task, 10, "分发子任务", fmt.Sprintf("LLM 生成了 %d 个子任务", len(plan.SubTasks)))

			idGen := utils.NewNanoIDGenerator(21)

			for _, st := range plan.SubTasks {
				subTaskID := idGen.Generate()
				subSpanID := fmt.Sprintf("%s-%s", spanID, idGen.Generate()[:4])

				taskType := parseTaskType(st.TaskType)
				if isAgentTask {
					taskType = domain.TaskTypeAgent
				}

				// 构建子任务 metadata 并继承父任务上下文
				subTaskMeta := map[string]interface{}{
					"goal":        st.Goal,
					"parent_id":   taskID,
					"parent_span": spanID,
					"depth":       strconv.Itoa(currentDepth),
					"llm_reason":  plan.Reason,
				}
				inheritContextFromTask(task, subTaskMeta)

				subTask, err := domain.NewTask(
					domain.NewTaskID(subTaskID),
					domain.NewTraceID(traceID),
					domain.NewSpanID(subSpanID),
					func() *domain.TaskID { pid := domain.NewTaskID(taskID); return &pid }(),
					st.Goal,
					"",
					taskType,
					subTaskMeta,
					DefaultTaskTimeout,
					0,
					0,
				)
				if err != nil {
					log.Printf("Failed to create sub-task: %v", err)
					continue
				}

				subTask.Start()
				if err := e.repo.Save(context.Background(), subTask); err != nil {
					log.Printf("Failed to save sub-task: %v", err)
					continue
				}

				e.executeSubTaskAsync(ctx, subTask)

				todoList.AddItem(subTaskID, st.Goal, taskType.String(), subSpanID, TodoStatusDistributed)
				subTaskIDs = append(subTaskIDs, subTaskID)

				if e.eventBus != nil {
					evt := domain.NewTodoSubTaskCreatedEvent(
						domain.NewTaskID(taskID),
						domain.NewTaskID(subTaskID),
						domain.NewTraceID(traceID),
						subTaskID,
						subSpanID,
						spanID,
						taskType,
						st.Goal,
					)
					e.eventBus.Publish(evt)
				}

				log.Printf("[AutoExecutor] 创建子任务(LLM): %s, spanID: %s, type: %s", subTaskID, subSpanID, taskType.String())
			}

			e.publishAndPersistTodoList(task, todoList)
		} else if isAgentTask {
			log.Printf("[AutoExecutor] Agent 模式下 LLM 未返回可用子任务")
			e.updateProgress(task, 10, "LLM 规划为空", "Agent 模式要求 LLM 返回子任务，任务终止")
			return e.failTask(task, errors.New("Agent 模式下 LLM 未生成子任务"))
		}
	}

	// 如果没有 LLM Provider 或 LLM 返回空，使用简单的随机子任务
	if !hasSubTasks {
		e.updateProgress(task, 10, "分发子任务", "使用默认子任务")
		time.Sleep(DefaultSubTaskDelay)

		subTasks := []struct {
			goal     string
			taskType domain.TaskType
		}{
			{"处理前50%数据", domain.TaskTypeCustom},
			{"处理后50%数据", domain.TaskTypeCustom},
			{"验证处理结果", domain.TaskTypeCustom},
		}

		idGen := utils.NewNanoIDGenerator(21)

		for _, st := range subTasks {
			subTaskID := idGen.Generate()
			subSpanID := fmt.Sprintf("%s-%s", spanID, idGen.Generate()[:4])
			taskType := st.taskType
			if task.Type() == domain.TaskTypeAgent {
				taskType = domain.TaskTypeAgent
			}

			// 构建子任务 metadata 并继承父任务上下文
			subTaskMeta := map[string]interface{}{
				"goal":        st.goal,
				"parent_id":   taskID,
				"parent_span": spanID,
				"depth":       strconv.Itoa(currentDepth),
			}
			inheritContextFromTask(task, subTaskMeta)

			subTask, err := domain.NewTask(
				domain.NewTaskID(subTaskID),
				domain.NewTraceID(traceID),
				domain.NewSpanID(subSpanID),
				func() *domain.TaskID { pid := domain.NewTaskID(taskID); return &pid }(),
				st.goal,
				"",
				taskType,
				subTaskMeta,
				DefaultTaskTimeout,
				0,
				0,
			)
			if err != nil {
				log.Printf("Failed to create sub-task: %v", err)
				continue
			}

			subTask.Start()
			if err := e.repo.Save(context.Background(), subTask); err != nil {
				log.Printf("Failed to save sub-task: %v", err)
				continue
			}

			e.executeSubTaskAsync(ctx, subTask)

			todoList.AddItem(subTaskID, st.goal, taskType.String(), subSpanID, TodoStatusDistributed)
			subTaskIDs = append(subTaskIDs, subTaskID)

			if e.eventBus != nil {
				evt := domain.NewTodoSubTaskCreatedEvent(
					domain.NewTaskID(taskID),
					domain.NewTaskID(subTaskID),
					domain.NewTraceID(traceID),
					subTaskID,
					subSpanID,
					spanID,
					taskType,
					st.goal,
				)
				e.eventBus.Publish(evt)
			}

			log.Printf("[AutoExecutor] 创建子任务: %s, spanID: %s, type: %s", subTaskID, subSpanID, taskType.String())
		}

		hasSubTasks = true
		e.publishAndPersistTodoList(task, todoList)
	}

	// 最终检查：只有所有子任务都完成，父任务才能完成
	if hasSubTasks {
		allCompleted, err := e.waitChildrenDone(ctx, task, todoList, subTaskIDs)
		if err != nil {
			return err
		}
		if !allCompleted {
			return e.failTask(task, errors.New("存在未完成子任务"))
		}
	}

	// 最终检查：只有所有子任务都完成，父任务才能完成
	if len(subTaskIDs) > 0 {
		allCompleted, err := e.waitChildrenDone(ctx, task, todoList, subTaskIDs)
		if err != nil {
			return err
		}
		if !allCompleted {
			return e.failTask(task, errors.New("存在未完成子任务"))
		}
	}

	return e.finishTask(task)
}

func (e *AutoTaskExecutor) updateProgress(task *domain.Task, progress int, stage, detail string) {
	task.UpdateProgress(100, progress, stage, detail)
	e.saveTaskPreservingMetadata(task)

	// 当任务完成（100% progress）时，收集执行结果
	if progress == 100 {
		e.collectTaskResult(task, stage, detail)
	}

	if e.eventBus != nil {
		evt := domain.NewTaskProgressUpdatedEvent(task, task.Progress())
		e.eventBus.Publish(evt)
	}
}

// collectTaskResult 收集任务执行结果到自身
func (e *AutoTaskExecutor) collectTaskResult(task *domain.Task, stage, detail string) {
	summary := map[string]interface{}{
		"task_id":      task.ID().String(),
		"span_id":      task.SpanID().String(),
		"goal":         task.Name(),
		"result":       detail,
		"stage":        stage,
		"completed_at": time.Now().UnixMilli(),
		"status":       task.Status().String(),
	}

	if task.Metadata() == nil {
		task.SetMetadata(map[string]interface{}{})
	}
	task.Metadata()["execution_summary"] = summary

	if err := e.repo.Save(context.Background(), task); err != nil {
		log.Printf("[AutoExecutor] collectTaskResult: save failed, err=%v", err)
	}
}

func (e *AutoTaskExecutor) publishTodoList(taskID, traceID string, todoList *TodoList) {
	todoListJSON, _ := json.Marshal(todoList)
	if e.eventBus != nil {
		evt := domain.NewTodoPublishedEvent(
			domain.NewTaskID(taskID),
			domain.NewTraceID(traceID),
			string(todoListJSON),
		)
		e.eventBus.Publish(evt)
	}
}

func (e *AutoTaskExecutor) publishAndPersistTodoList(task *domain.Task, todoList *TodoList) {
	if task.Metadata() == nil {
		task.SetMetadata(map[string]interface{}{})
	}
	task.Metadata()["todo_list"] = todoList.ToJSON()
	e.saveTaskPreservingMetadata(task)
	e.publishTodoList(task.ID().String(), task.TraceID().String(), todoList)
}

func (e *AutoTaskExecutor) waitChildrenDone(ctx context.Context, task *domain.Task, todoList *TodoList, subTaskIDs []string) (bool, error) {
	if len(subTaskIDs) == 0 {
		return true, nil
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-ticker.C:
			completed := 0
			failed := 0

			for _, subTaskID := range subTaskIDs {
				child, err := e.repo.FindByID(ctx, domain.NewTaskID(subTaskID))
				if err != nil {
					continue
				}

				progress := int(child.Progress().Percentage())
				todoList.UpdateProgress(subTaskID, progress)

				switch child.Status() {
				case domain.TaskStatusCompleted:
					todoList.MarkCompleted(subTaskID)
					completed++
				case domain.TaskStatusFailed:
					todoList.MarkFailed(subTaskID)
					failed++
				case domain.TaskStatusCancelled:
					todoList.MarkCancelled(subTaskID)
					failed++
				}
			}

			e.publishAndPersistTodoList(task, todoList)

			total := len(subTaskIDs)
			percentage := 10 + int(float64(completed+failed)/float64(total)*80)
			e.updateProgress(task, percentage, "等待子任务执行", fmt.Sprintf("子任务完成 %d/%d", completed, total))

			if completed+failed == total {
				return failed == 0, nil
			}
		}
	}
}

func (e *AutoTaskExecutor) finishTask(task *domain.Task) error {
	resultData := map[string]interface{}{
		"completed_at": time.Now().UnixMilli(),
	}

	result := domain.NewResult(resultData, "任务完成")
	task.Complete(result)
	e.updateProgress(task, 100, "完成", "任务执行完成")
	e.saveTaskPreservingMetadata(task)

	if e.eventBus != nil {
		evt := domain.NewTaskCompletedEvent(task)
		e.eventBus.Publish(evt)
	}
	return nil
}

func (e *AutoTaskExecutor) failTask(task *domain.Task, taskErr error) error {
	task.Fail(taskErr)
	e.saveTaskPreservingMetadata(task)

	if e.eventBus != nil {
		evt := domain.NewTaskFailedEvent(task)
		e.eventBus.Publish(evt)
	}
	return nil
}

func (e *AutoTaskExecutor) saveTaskPreservingMetadata(task *domain.Task) {
	current, err := e.repo.FindByID(context.Background(), task.ID())
	if err == nil && current.Metadata() != nil {
		if task.Metadata() == nil {
			task.SetMetadata(map[string]interface{}{})
		}
		for k, v := range current.Metadata() {
			if _, ok := task.Metadata()[k]; !ok {
				task.Metadata()[k] = v
			}
		}
	}
	e.repo.Save(context.Background(), task)
}

func (e *AutoTaskExecutor) executeSubTaskAsync(ctx context.Context, task *domain.Task) {
	go func(t *domain.Task) {
		if err := e.ExecuteAutoTask(ctx, t); err != nil {
			log.Printf("sub-task execute failed: task=%s err=%v", t.ID().String(), err)
			_ = e.failTask(t, err)
		}
	}(task)
}
