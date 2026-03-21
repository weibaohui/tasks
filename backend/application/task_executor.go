/**
 * TaskExecutor 任务执行器
 * 根据任务类型执行对应的处理逻辑
 */
package application

import (
	"context"
	"encoding/json"
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
	metadata := task.Metadata()
	taskName := task.Name()

	progressTotal := 100
	if v, ok := metadata["progress_total"].(float64); ok {
		progressTotal = int(v)
	}

	for i := 0; i <= progressTotal; i += 10 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			task.UpdateProgress(progressTotal, i, "执行中", fmt.Sprintf("已完成 %d%%", i))
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
	metadata := task.Metadata()
	taskName := task.Name()
	iterations := 10
	if v, ok := metadata["iterations"].(float64); ok {
		iterations = int(v)
	}

	for i := 1; i <= iterations; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			progress := (i * 100) / iterations
			task.UpdateProgress(100, progress, "数据处理中", fmt.Sprintf("处理第 %d/%d 批", i, iterations))
			repo.Save(ctx, task)
			time.Sleep(time.Duration(200+rand.Intn(300)) * time.Millisecond)
		}
	}

	result := domain.NewResult(map[string]interface{}{
		"task_name": taskName,
		"type":      "data_processing",
		"processed": iterations,
		"timestamp": time.Now().Unix(),
	}, fmt.Sprintf("成功处理 %d 批数据", iterations))
	return task.Complete(result)
}

func FileOperationHandler(ctx context.Context, task *domain.Task, repo domain.TaskRepository) error {
	metadata := task.Metadata()
	taskName := task.Name()
	fileCount := 5
	if v, ok := metadata["file_count"].(float64); ok {
		fileCount = int(v)
	}

	stages := []string{"读取文件", "处理数据", "写入结果", "验证完整性", "清理临时文件"}
	for i, stage := range stages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			progress := ((i + 1) * 100) / len(stages)
			task.UpdateProgress(100, progress, stage, fmt.Sprintf("正在%s (%d/%d)", stage, i+1, len(stages)))
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
		"type":       "file_operation",
		"files":      files,
		"file_count": fileCount,
		"timestamp":  time.Now().Unix(),
	}, fmt.Sprintf("成功处理 %d 个文件", fileCount))
	return task.Complete(result)
}

func APICallHandler(ctx context.Context, task *domain.Task, repo domain.TaskRepository) error {
	metadata := task.Metadata()
	taskName := task.Name()
	url := ""
	if v, ok := metadata["url"].(string); ok {
		url = v
	}
	method := "GET"
	if v, ok := metadata["method"].(string); ok {
		method = v
	}

	stages := []string{"构建请求", "发送请求", "等待响应", "处理响应", "完成"}
	for i, stage := range stages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			progress := ((i + 1) * 100) / len(stages)
			detail := fmt.Sprintf("正在%s", stage)
			if stage == "发送请求" && url != "" {
				detail = fmt.Sprintf("发送 %s 请求到 %s", method, url)
			}
			task.UpdateProgress(100, progress, stage, detail)
			repo.Save(ctx, task)
			time.Sleep(time.Duration(200+rand.Intn(150)) * time.Millisecond)
		}
	}

	resultData := map[string]interface{}{
		"task_name": taskName,
		"type":      "api_call",
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
	metadata := task.Metadata()
	taskName := task.Name()
	command := ""
	if v, ok := metadata["command"].(string); ok {
		command = v
	}

	task.UpdateProgress(100, 10, "准备执行", "初始化自定义任务")
	repo.Save(ctx, task)
	time.Sleep(300 * time.Millisecond)

	task.UpdateProgress(100, 50, "执行中", fmt.Sprintf("执行命令: %s", command))
	repo.Save(ctx, task)
	time.Sleep(500 * time.Millisecond)

	task.UpdateProgress(100, 90, "完成", "自定义任务执行完成")
	repo.Save(ctx, task)
	time.Sleep(200 * time.Millisecond)

	metadataJSON, _ := json.Marshal(metadata)
	result := domain.NewResult(map[string]interface{}{
		"task_name": taskName,
		"type":      "custom",
		"command":   command,
		"metadata":  string(metadataJSON),
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
			_ = (i * 100) / totalSteps
			task.UpdateProgress(totalSteps, i, "模拟任务执行中", fmt.Sprintf("步骤 %d/%d", i, totalSteps))
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
