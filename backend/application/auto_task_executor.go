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
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/utils"
)

const MaxTaskDepth = 3 // 最大任务深度，超过则直接完成

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

	log.Printf("[AutoExecutor] 开始执行任务: %s, traceID: %s, spanID: %s", taskID, traceID, spanID)

	todoList := NewTodoList(taskID)

	e.updateProgress(task, 0, "初始化", "开始执行任务")
	time.Sleep(10 * time.Second)

	// 100% 分发子任务（调试用）
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
	}

	e.publishTodoList(taskID, traceID, spanID, todoList)

	for i := 20; i <= 90; i += 10 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			e.updateProgress(task, i, "等待子任务执行", fmt.Sprintf("等待子任务完成... %d%%", i))
			time.Sleep(5 * time.Second)
		}
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

func (e *AutoTaskExecutor) HandleSubTaskCompleted(subTaskID, parentTaskID string) {
	parentTask, err := e.repo.FindByID(context.Background(), domain.NewTaskID(parentTaskID))
	if err != nil {
		log.Printf("Failed to find parent task: %v", err)
		return
	}

	todoListJSON, ok := parentTask.Metadata()["todo_list"]
	if !ok {
		return
	}

	todoJSON, ok := todoListJSON.(string)
	if !ok {
		return
	}

	var todoList *TodoList
	if err := json.Unmarshal([]byte(todoJSON), todoList); err != nil {
		log.Printf("Failed to unmarshal todo list: %v", err)
		return
	}

	todoList.MarkCompleted(subTaskID)

	updatedJSON, _ := json.Marshal(todoList)
	parentTask.Metadata()["todo_list"] = string(updatedJSON)
	e.repo.Save(context.Background(), parentTask)

	e.publishTodoList(parentTaskID, parentTask.TraceID().String(), parentTask.SpanID().String(), todoList)
}
