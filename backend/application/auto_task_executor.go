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
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/llm"
	"github.com/weibh/taskmanager/infrastructure/trace"
	"github.com/weibh/taskmanager/infrastructure/utils"
)

const MaxTaskDepth = 4

// inheritContextFromTask 从父任务继承上下文信息（agent_code, user_code, channel_code, session_key）
func inheritContextFromTask(parent *domain.Task, childTask *domain.Task) {
	if parent == nil {
		return
	}

	// 继承上下文字段到子任务
	if v := parent.AgentCode(); v != "" {
		childTask.SetAgentCode(v)
	}
	if v := parent.UserCode(); v != "" {
		childTask.SetUserCode(v)
	}
	if v := parent.ChannelCode(); v != "" {
		childTask.SetChannelCode(v)
	}
	if v := parent.SessionKey(); v != "" {
		childTask.SetSessionKey(v)
	}
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
	// 从 context 中获取 trace 信息，不再从 task 提取
	traceID := trace.GetTraceID(ctx)
	spanID := trace.MustGetSpanID(ctx)

	// 使用独立字段获取当前深度
	currentDepth := task.Depth() + 1

	log.Printf("[AutoExecutor] 执行任务: %s, spanID: %s, depth: %d/%d", taskID, spanID, currentDepth, MaxTaskDepth)

	todoList := NewTodoList(taskID)

	e.updateProgress(task, 0, "初始化", fmt.Sprintf("开始执行任务，深度 %d", currentDepth))
	time.Sleep(AgentInitDelay)

	// 使用 LLM 动态生成子任务
	subTaskIDs := make([]string, 0)
	hasSubTasks := false
	isAgentTask := task.Type() == domain.TaskTypeAgent

	// 动态获取 LLM Provider
	llmProvider, err := e.getLLMProviderForTask(ctx, task)

	// 检查是否达到最大深度
	if currentDepth >= MaxTaskDepth {
		// 达到最大深度时，仍然需要调用 LLM 获取最终结论
		if llmProvider != nil {
			e.updateProgress(task, 95, "获取最终结论", "达到最大深度，调用 LLM 获取结论")
			plan, llmErr := llmProvider.GenerateSubTasks(
				ctx,
				task.Name(),
				task.Description(),
				currentDepth,
				MaxTaskDepth,
			)
			if llmErr == nil && plan != nil && plan.Reason != "" {
				task.SetTaskConclusion(plan.Reason)
			}
		}
		e.updateProgress(task, 100, "完成", "任务执行完成")
		return e.finishTask(task)
	}
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
				// 使用 trace.StartSpan 从 context 自动获取新 spanID，parentSpanID 自动注入
				subCtx, subSpanID := trace.StartSpan(ctx)

				taskType := parseTaskType(st.TaskType)
				if isAgentTask {
					taskType = domain.TaskTypeAgent
				}

				// 子任务目标来自 LLM 生成的 Goal，验收标准来自规划原因
				taskRequirement := st.Goal
				acceptanceCriteria := fmt.Sprintf("完成目标: %s", st.Goal)
				if plan.Reason != "" {
					acceptanceCriteria = plan.Reason
				}

				subTask, err := domain.NewTask(
					domain.NewTaskID(subTaskID),
					domain.NewTraceID(trace.GetTraceID(subCtx)),
					domain.NewSpanID(subSpanID),
					func() *domain.TaskID { pid := domain.NewTaskID(taskID); return &pid }(),
					st.Goal,
					"",
					taskType,
					taskRequirement,
					acceptanceCriteria,
					DefaultTaskTimeout,
					0,
					0,
				)
				if err != nil {
					log.Printf("Failed to create sub-task: %v", err)
					continue
				}

				// 设置深度（parentSpan 已通过 StartSpan 自动注入 context）
				subTask.SetDepth(currentDepth)
				// 从 context 获取 parentSpanID 并设置
				subTask.SetParentSpan(trace.GetParentSpanID(subCtx))

				// 继承父任务上下文
				inheritContextFromTask(task, subTask)

				subTask.Start()
				if err := e.repo.Save(context.Background(), subTask); err != nil {
					log.Printf("Failed to save sub-task: %v", err)
					continue
				}

				e.executeSubTaskAsync(subCtx, subTask)

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
			log.Printf("[AutoExecutor] Agent 模式下 LLM 未返回子任务，任务直接完成")
			// Agent 模式下 LLM 认为不需要子任务是正常的，把 LLM 的分析结果作为结论
			if plan != nil && plan.Reason != "" {
				task.SetTaskConclusion(plan.Reason)
			}
			return e.finishTask(task)
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

			// 默认子任务：目标来自 st.goal，验收标准来自父任务的 acceptanceCriteria
			taskRequirement := st.goal
			acceptanceCriteria := task.AcceptanceCriteria()

			subTask, err := domain.NewTask(
				domain.NewTaskID(subTaskID),
				domain.NewTraceID(traceID),
				domain.NewSpanID(subSpanID),
				func() *domain.TaskID { pid := domain.NewTaskID(taskID); return &pid }(),
				st.goal,
				"",
				taskType,
				taskRequirement,
				acceptanceCriteria,
				DefaultTaskTimeout,
				0,
				0,
			)
			if err != nil {
				log.Printf("Failed to create sub-task: %v", err)
				continue
			}

			// 设置深度和父 span（独立字段）
			subTask.SetDepth(currentDepth)
			subTask.SetParentSpan(spanID)

			// 继承父任务上下文
			inheritContextFromTask(task, subTask)

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
	task.UpdateProgress(progress)
	e.saveTaskPreservingMetadata(task)

	if e.eventBus != nil {
		evt := domain.NewTaskProgressUpdatedEvent(task, task.Progress())
		e.eventBus.Publish(evt)
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
	// 使用独立字段存储 todo_list
	task.SetTodoList(todoList.ToJSON())
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

				progress := child.Progress().Value()
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

	// 获取任务自身的结论
	taskConclusion := task.TaskConclusion()

	// 收集子任务结果，提取每个子任务的 task_conclusion
	var allChildConclusions []string
	todoListStr := task.TodoList()
	if todoListStr != "" {
		var todoList TodoList
		if err := json.Unmarshal([]byte(todoListStr), &todoList); err == nil {
			subTaskResults := make([]map[string]interface{}, 0, len(todoList.Items))
			for _, item := range todoList.Items {
				// 获取子任务
				subTask, err := e.repo.FindByID(context.Background(), domain.NewTaskID(item.SubTaskID))
				if err == nil && subTask != nil {
					subResult := map[string]interface{}{
						"task_id":   item.SubTaskID,
						"goal":      item.Goal,
						"status":    string(item.Status),
						"progress":  item.Progress,
					}
					// 从子任务获取 task_conclusion
					if childConclusion := subTask.TaskConclusion(); childConclusion != "" {
						subResult["task_conclusion"] = childConclusion
						allChildConclusions = append(allChildConclusions, childConclusion)
					}
					subTaskResults = append(subTaskResults, subResult)
				}
			}
			if len(subTaskResults) > 0 {
				resultData["sub_tasks_results"] = subTaskResults
			}
		}
	}

	// 如果任务没有自己的结论，聚合子任务的结论
	if taskConclusion == "" && len(allChildConclusions) > 0 {
		combinedConclusion := ""
		for i, r := range allChildConclusions {
			if i > 0 {
				combinedConclusion += "\n\n"
			}
			combinedConclusion += r
		}
		taskConclusion = combinedConclusion
	}

	// 确保 task_conclusion 一定有值
	if taskConclusion == "" {
		taskConclusion = "任务完成"
	}
	// 必须先设置结论，Complete 会使用 taskConclusion 作为 result 的值
	task.SetTaskConclusion(taskConclusion)

	task.Complete()
	e.updateProgress(task, 100, "完成", "任务执行完成")
	e.saveTaskPreservingMetadata(task)

	if e.eventBus != nil {
		evt := domain.NewTaskCompletedEvent(task)
		e.eventBus.Publish(evt)
	}
	return nil
}

// updateParentWithChildResult 更新父任务的 result，将当前任务的 task_conclusion 追加到父任务的 sub_tasks_results 中
func (e *AutoTaskExecutor) updateParentWithChildResult(task *domain.Task) {
	parentID := task.ParentID()
	if parentID == nil {
		return
	}

	// 获取当前任务的 task_conclusion
	taskConclusion := task.TaskConclusion()
	if taskConclusion == "" {
		return
	}

	// 获取父任务
	parent, err := e.repo.FindByID(context.Background(), *parentID)
	if err != nil || parent == nil {
		return
	}

	// 更新父任务的保存
	e.repo.Save(context.Background(), parent)
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
