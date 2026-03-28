/**
 * TaskExecutor 任务执行器
 * 根据任务类型执行对应的处理逻辑
 */
package application

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type TaskHandler func(ctx context.Context, task *domain.Task, repo domain.TaskRepository) error

type TaskExecutor struct {
	handlers map[domain.TaskType]TaskHandler
}

func NewTaskExecutor() *TaskExecutor {
	return &TaskExecutor{
		handlers: make(map[domain.TaskType]TaskHandler),
	}
}

func (e *TaskExecutor) RegisterHandler(taskType domain.TaskType, handler TaskHandler) {
	e.handlers[taskType] = handler
}

func (e *TaskExecutor) Execute(ctx context.Context, task *domain.Task, repo domain.TaskRepository) error {
	handler, ok := e.handlers[task.Type()]
	if !ok {
		handler = e.defaultHandler
	}
	return handler(ctx, task, repo)
}

func (e *TaskExecutor) defaultHandler(ctx context.Context, task *domain.Task, repo domain.TaskRepository) error {
	progressTotal := 100

	for i := 0; i <= progressTotal; i += 10 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			task.UpdateProgress(i)
			repo.Save(ctx, task)
			time.Sleep(time.Duration(100+rand.Intn(200)) * time.Millisecond)
		}
	}

	return task.Complete()
}

func DataProcessingHandler(ctx context.Context, task *domain.Task, repo domain.TaskRepository) error {
	iterations := 10

	for i := 1; i <= iterations; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			progress := (i * 100) / iterations
			task.UpdateProgress(progress)
			repo.Save(ctx, task)
			time.Sleep(time.Duration(200+rand.Intn(300)) * time.Millisecond)
		}
	}

	return task.Complete()
}

func FileOperationHandler(ctx context.Context, task *domain.Task, repo domain.TaskRepository) error {
	stages := []string{"读取文件", "处理数据", "写入结果", "验证完整性", "清理临时文件"}
	for i := range stages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			progress := ((i + 1) * 100) / len(stages)
			task.UpdateProgress(progress)
			repo.Save(ctx, task)
			time.Sleep(time.Duration(300+rand.Intn(200)) * time.Millisecond)
		}
	}

	return task.Complete()
}

func APICallHandler(ctx context.Context, task *domain.Task, repo domain.TaskRepository) error {
	stages := []string{"构建请求", "发送请求", "等待响应", "处理响应", "完成"}
	for i := range stages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			progress := ((i + 1) * 100) / len(stages)
			task.UpdateProgress(progress)
			repo.Save(ctx, task)
			time.Sleep(time.Duration(200+rand.Intn(150)) * time.Millisecond)
		}
	}

	return task.Complete()
}

func CustomHandler(ctx context.Context, task *domain.Task, repo domain.TaskRepository) error {
	task.UpdateProgress(10)
	repo.Save(ctx, task)
	time.Sleep(300 * time.Millisecond)

	task.UpdateProgress(50)
	repo.Save(ctx, task)
	time.Sleep(500 * time.Millisecond)

	task.UpdateProgress(90)
	repo.Save(ctx, task)
	time.Sleep(200 * time.Millisecond)

	return task.Complete()
}

func SimulatedLongRunningHandler(ctx context.Context, task *domain.Task, repo domain.TaskRepository) error {
	totalSteps := 20
	for i := 1; i <= totalSteps; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			progress := (i * 100) / totalSteps
			task.UpdateProgress(progress)
			repo.Save(ctx, task)
			time.Sleep(time.Duration(500+rand.Intn(300)) * time.Millisecond)
		}
	}

	_ = fmt.Sprintf("模拟任务完成，共执行 %d 步", totalSteps)
	return task.Complete()
}
