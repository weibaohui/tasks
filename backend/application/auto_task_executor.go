/**
 * AutoTaskExecutor - 自动任务执行器
 * 支持子任务分发和 Todo 列表管理
 */
package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/utils"
)

const MaxTaskDepth = 4

type AutoTaskExecutor struct {
	repo       domain.TaskRepository
	eventBus   interface{ Publish(domain.DomainEvent) }
	registry   *TaskRegistry
	workerPool interface{ Submit(*domain.Task) bool }
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
	if rand.Float32() < 0.9 {
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

			if e.workerPool != nil {
				e.workerPool.Submit(subTask)
			}

			todoList.AddItem(subTaskID, st.goal, string(st.taskType), subSpanID, TodoStatusDistributed)

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

		e.publishTodoList(taskID, traceID, spanID, todoList)

		// 保存 TodoList 到 task metadata
		if task.Metadata() != nil {
			task.Metadata()["todo_list"] = todoList.ToJSON()
			e.repo.Save(context.Background(), task)
		}

		for i := 20; i <= 90; i += 10 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				e.updateProgress(task, i, "等待子任务执行", fmt.Sprintf("等待子任务完成... %d%%", i))
				time.Sleep(5 * time.Second)
			}
		}
	} else {
		e.updateProgress(task, 50, "直接完成", "10%概率选择直接完成任务")
		time.Sleep(2 * time.Second)
	}

	return e.finishTask(task)
}

func (e *AutoTaskExecutor) updateProgress(task *domain.Task, progress int, stage, detail string) {
	task.UpdateProgress(100, progress, stage, detail)
	e.repo.Save(context.Background(), task)

	if e.eventBus != nil {
		evt := domain.NewTaskProgressUpdatedEvent(task, task.Progress())
		e.eventBus.Publish(evt)
	}
}

func (e *AutoTaskExecutor) publishTodoList(taskID, traceID, spanID string, todoList *TodoList) {
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

func (e *AutoTaskExecutor) finishTask(task *domain.Task) error {
	result := domain.NewResult(map[string]interface{}{
		"completed_at": time.Now().UnixMilli(),
	}, "任务完成")
	task.Complete(result)
	e.repo.Save(context.Background(), task)

	if e.eventBus != nil {
		evt := domain.NewTaskCompletedEvent(task)
		e.eventBus.Publish(evt)
	}
	return nil
}
