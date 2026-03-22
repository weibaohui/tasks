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
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/utils"
)

const MaxTaskDepth = 4

// TaskExecutionSummary 单个任务的执行摘要
type TaskExecutionSummary struct {
	TaskID      string `json:"task_id"`
	SpanID      string `json:"span_id"`
	Goal        string `json:"goal"`          // 目标是什么
	Result      string `json:"result"`        // 结果是什么
	Stage       string `json:"stage"`
	CompletedAt int64  `json:"completed_at"`
	Status      string `json:"status"`
}

type AutoTaskExecutor struct {
	repo       domain.TaskRepository
	eventBus   interface{ Publish(domain.DomainEvent) }
	registry   *TaskRegistry
	workerPool interface{ Submit(*domain.Task) bool }
	resultMu   sync.Mutex // 保护并发保存 execution_summaries
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
	time.Sleep(10 * time.Second)

	// 检查是否达到最大深度
	if currentDepth >= MaxTaskDepth {
		e.updateProgress(task, 100, "完成", "达到最大深度，直接完成")
		return e.finishTask(task)
	}

	// 90% 概率分发子任务，10% 概率直接完成
	subTaskIDs := make([]string, 0)
	hasSubTasks := false

	if rand.Float32() < 0.9 {
		hasSubTasks = true
		e.updateProgress(task, 10, "分发子任务", "开始创建子任务")

		subTasks := []struct {
			goal     string
			taskType domain.TaskType
		}{
			{"处理前50%数据", domain.TaskTypeDataProcessing},
			{"处理后50%数据", domain.TaskTypeFileOperation},
			{"验证处理结果", domain.TaskTypeAPICall},
		}

		idGen := utils.NewNanoIDGenerator(21)

		for _, st := range subTasks {
			subTaskID := idGen.Generate()
			subSpanID := fmt.Sprintf("%s-%s", spanID, idGen.Generate()[:4])

			subTask, err := domain.NewTask(
				domain.NewTaskID(subTaskID),
				domain.NewTraceID(traceID),
				domain.NewSpanID(subSpanID),
				func() *domain.TaskID { pid := domain.NewTaskID(taskID); return &pid }(),
				st.goal,
				"",
				st.taskType,
				map[string]interface{}{
					"goal":        st.goal,
					"parent_id":   taskID,
					"parent_span": spanID,
					"depth":       strconv.Itoa(currentDepth),
				},
				60000*time.Millisecond,
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

			e.executeSubTaskAsync(subTask)

			todoList.AddItem(subTaskID, st.goal, string(st.taskType), subSpanID, TodoStatusDistributed)
			subTaskIDs = append(subTaskIDs, subTaskID)

			if e.eventBus != nil {
				evt := domain.NewTodoSubTaskCreatedEvent(
					domain.NewTaskID(taskID),
					domain.NewTaskID(subTaskID),
					domain.NewTraceID(traceID),
					subTaskID,
					subSpanID,
					spanID,
					st.taskType,
					st.goal,
				)
				e.eventBus.Publish(evt)
			}

			log.Printf("[AutoExecutor] 创建子任务: %s, spanID: %s", subTaskID, subSpanID)
		}

		e.publishAndPersistTodoList(task, todoList)
	} else {
		hasSubTasks = false
		e.updateProgress(task, 50, "直接完成", "10%概率选择直接完成任务")
		time.Sleep(2 * time.Second)
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
				child, err := e.repo.FindByID(context.Background(), domain.NewTaskID(subTaskID))
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
	if task.ParentID() == nil && task.Metadata() != nil {
		if homework, ok := task.Metadata()["homework_submissions"]; ok {
			resultData["homework_submissions"] = homework
		}
	}

	result := domain.NewResult(resultData, "任务完成")
	task.Complete(result)
	e.updateProgress(task, 100, "完成", "任务执行完成")
	e.saveTaskPreservingMetadata(task)

	if err := e.submitHomeworkToRoot(task); err != nil {
		log.Printf("submit homework failed: %v", err)
	}

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

func (e *AutoTaskExecutor) submitHomeworkToRoot(task *domain.Task) error {
	if task.ParentID() == nil {
		return nil
	}

	rootTask := task
	for rootTask.ParentID() != nil {
		parent, err := e.repo.FindByID(context.Background(), *rootTask.ParentID())
		if err != nil {
			return err
		}
		rootTask = parent
	}

	if rootTask.Metadata() == nil {
		rootTask.SetMetadata(map[string]interface{}{})
	}

	submission := map[string]interface{}{
		"task_id":      task.ID().String(),
		"parent_id":    task.ParentID().String(),
		"trace_id":     task.TraceID().String(),
		"span_id":      task.SpanID().String(),
		"submitted_at": time.Now().UnixMilli(),
		"status":       task.Status().String(),
		"result":       nil,
	}
	if task.Result() != nil {
		submission["result"] = task.Result().ToMap()
	}

	raw, exists := rootTask.Metadata()["homework_submissions"]
	list := make([]map[string]interface{}, 0)
	if exists {
		if arr, ok := raw.([]interface{}); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					list = append(list, m)
				}
			}
		}
	}

	updated := false
	for i := range list {
		if id, ok := list[i]["task_id"].(string); ok && id == task.ID().String() {
			list[i] = submission
			updated = true
			break
		}
	}
	if !updated {
		list = append(list, submission)
	}

	rootTask.Metadata()["homework_submissions"] = list
	e.resultMu.Lock()
	err := e.repo.Save(context.Background(), rootTask)
	e.resultMu.Unlock()
	return err
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

func (e *AutoTaskExecutor) executeSubTaskAsync(task *domain.Task) {
	go func(t *domain.Task) {
		if err := e.ExecuteAutoTask(context.Background(), t); err != nil {
			log.Printf("sub-task execute failed: task=%s err=%v", t.ID().String(), err)
			_ = e.failTask(t, err)
		}
	}(task)
}
