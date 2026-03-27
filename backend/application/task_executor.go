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
	taskName := task.Name()

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

	result := domain.NewResult(map[string]interface{}{
		"task_name": taskName,
		"executed":  true,
		"timestamp": time.Now().Unix(),
	}, "执行完成")
	return task.Complete(result)
}

func DataProcessingHandler(ctx context.Context, task *domain.Task, repo domain.TaskRepository) error {
	taskName := task.Name()
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

	result := domain.NewResult(map[string]interface{}{
		"task_name": taskName,
		"type":      "custom",
		"processed": iterations,
		"timestamp": time.Now().Unix(),
	}, fmt.Sprintf("成功处理 %d 批数据", iterations))
	return task.Complete(result)
}

func FileOperationHandler(ctx context.Context, task *domain.Task, repo domain.TaskRepository) error {
	taskName := task.Name()
	fileCount := 5

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

	files := make([]string, fileCount)
	for i := 0; i < fileCount; i++ {
		files[i] = fmt.Sprintf("file_%d.dat", i+1)
	}

	result := domain.NewResult(map[string]interface{}{
		"task_name":  taskName,
		"type":       "custom",
		"files":      files,
		"file_count": fileCount,
		"timestamp":  time.Now().Unix(),
	}, fmt.Sprintf("成功处理 %d 个文件", fileCount))
	return task.Complete(result)
}

func APICallHandler(ctx context.Context, task *domain.Task, repo domain.TaskRepository) error {
	taskName := task.Name()
	url := ""
	method := "GET"

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

	resultData := map[string]interface{}{
		"task_name": taskName,
		"type":      "custom",
		"method":    method,
		"url":       url,
		"status":    "success",
		"code":      200,
		"timestamp": time.Now().Unix(),
	}
	result := domain.NewResult(resultData, fmt.Sprintf("%s 请求成功", method))
	return task.Complete(result)
}

func CustomHandler(ctx context.Context, task *domain.Task, repo domain.TaskRepository) error {
	taskName := task.Name()
	command := ""

	task.UpdateProgress(10)
	repo.Save(ctx, task)
	time.Sleep(300 * time.Millisecond)

	task.UpdateProgress(50)
	repo.Save(ctx, task)
	time.Sleep(500 * time.Millisecond)

	task.UpdateProgress(90)
	repo.Save(ctx, task)
	time.Sleep(200 * time.Millisecond)

	result := domain.NewResult(map[string]interface{}{
		"task_name": taskName,
		"type":      "custom",
		"command":   command,
		"timestamp": time.Now().Unix(),
	}, "自定义任务执行完成")
	return task.Complete(result)
}

func SimulatedLongRunningHandler(ctx context.Context, task *domain.Task, repo domain.TaskRepository) error {
	taskName := task.Name()
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

	result := domain.NewResult(map[string]interface{}{
		"task_name": taskName,
		"type":      "simulated",
		"steps":     totalSteps,
		"timestamp": time.Now().Unix(),
	}, fmt.Sprintf("模拟任务完成，共执行 %d 步", totalSteps))
	return task.Complete(result)
}
