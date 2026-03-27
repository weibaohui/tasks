/**
 * WorkerPool 单元测试
 */
package application

import (
	"container/heap"
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/weibh/taskmanager/domain"
	"go.uber.org/zap"
)

func TestWorkerPool_Submit(t *testing.T) {
	logger := zap.NewNop()
	wp := NewWorkerPool(2, logger)

	task, _ := domain.NewTask(
		domain.NewTaskID("task-1"),
		domain.NewTraceID("trace-1"),
		domain.NewSpanID("span-1"),
		nil,
		"测试任务1",
		"",
		domain.TaskTypeCustom,
		"测试目标",
		"测试验收标准",
		nil,
		60*time.Second,
		5,
		0,
	)

	// 提交前队列为空
	if wp.taskQueue.Len() != 0 {
		t.Errorf("期望队列长度为0, 实际为 %d", wp.taskQueue.Len())
	}

	// 提交任务
	ok := wp.Submit(task)
	if !ok {
		t.Error("Submit应返回true")
	}

	// 提交后队列有1个任务
	if wp.taskQueue.Len() != 1 {
		t.Errorf("期望队列长度为1, 实际为 %d", wp.taskQueue.Len())
	}
}

func TestWorkerPool_SubmitAfterClose(t *testing.T) {
	logger := zap.NewNop()
	wp := NewWorkerPool(2, logger)

	// 关闭工作池
	wp.mu.Lock()
	wp.closed = true
	wp.mu.Unlock()

	task, _ := domain.NewTask(
		domain.NewTaskID("task-1"),
		domain.NewTraceID("trace-1"),
		domain.NewSpanID("span-1"),
		nil,
		"测试任务1",
		"",
		domain.TaskTypeCustom,
		"测试目标",
		"测试验收标准",
		nil,
		60*time.Second,
		5,
		0,
	)

	ok := wp.Submit(task)
	if ok {
		t.Error("关闭后Submit应返回false")
	}
}

func TestWorkerPool_PriorityQueue(t *testing.T) {
	logger := zap.NewNop()
	wp := NewWorkerPool(1, logger)

	// 创建不同优先级的任务
	// NewTask 签名: ..., maxRetries int, priority int
	task1, _ := domain.NewTask(
		domain.NewTaskID("task-1"),
		domain.NewTraceID("trace-1"),
		domain.NewSpanID("span-1"),
		nil,
		"任务1",
		"",
		domain.TaskTypeCustom,
		"测试目标",
		"测试验收标准",
		nil,
		60*time.Second,
		0,   // maxRetries
		1,   // 优先级1
	)

	task2, _ := domain.NewTask(
		domain.NewTaskID("task-2"),
		domain.NewTraceID("trace-1"),
		domain.NewSpanID("span-1"),
		nil,
		"任务2",
		"",
		domain.TaskTypeCustom,
		"测试目标",
		"测试验收标准",
		nil,
		60*time.Second,
		0,   // maxRetries
		10,  // 优先级10（更高）
	)

	task3, _ := domain.NewTask(
		domain.NewTaskID("task-3"),
		domain.NewTraceID("trace-1"),
		domain.NewSpanID("span-1"),
		nil,
		"任务3",
		"",
		domain.TaskTypeCustom,
		"测试目标",
		"测试验收标准",
		nil,
		60*time.Second,
		0,   // maxRetries
		5,   // 优先级5
	)

	// 验证任务优先级是否正确设置
	if task1.Priority() != 1 {
		t.Errorf("task1 优先级期望1, 实际 %d", task1.Priority())
	}
	if task2.Priority() != 10 {
		t.Errorf("task2 优先级期望10, 实际 %d", task2.Priority())
	}
	if task3.Priority() != 5 {
		t.Errorf("task3 优先级期望5, 实际 %d", task3.Priority())
	}

	// 按优先级从低到高提交
	wp.Submit(task1) // 优先级1
	wp.Submit(task2) // 优先级10
	wp.Submit(task3) // 优先级5

	// 验证队列长度
	if wp.taskQueue.Len() != 3 {
		t.Fatalf("期望3个任务, 实际为 %d", wp.taskQueue.Len())
	}

	// 通过 heap.Pop 验证优先级顺序
	// heap.Pop 返回优先级最高的任务
	wp.mu.Lock()
	item1 := heap.Pop(wp.taskQueue).(*TaskItem)
	item2 := heap.Pop(wp.taskQueue).(*TaskItem)
	item3 := heap.Pop(wp.taskQueue).(*TaskItem)
	wp.mu.Unlock()

	// 验证 pop 顺序（高优先级先出）
	if item1.priority != 10 {
		t.Errorf("第一次Pop期望优先级10, 实际为 %d", item1.priority)
	}

	if item2.priority != 5 {
		t.Errorf("第二次Pop期望优先级5, 实际为 %d", item2.priority)
	}

	if item3.priority != 1 {
		t.Errorf("第三次Pop期望优先级1, 实际为 %d", item3.priority)
	}
}

func TestWorkerPool_TaskExecution(t *testing.T) {
	logger := zap.NewNop()
	wp := NewWorkerPool(1, logger)

	var executedTaskID string
	var wg sync.WaitGroup
	wg.Add(1)

	wp.SetExecuteFunc(func(ctx context.Context, task *domain.Task) {
		executedTaskID = task.ID().String()
		task.Start()
		wg.Done()
	})

	task, _ := domain.NewTask(
		domain.NewTaskID("task-exec-001"),
		domain.NewTraceID("trace-1"),
		domain.NewSpanID("span-1"),
		nil,
		"执行测试任务",
		"",
		domain.TaskTypeCustom,
		"测试目标",
		"测试验收标准",
		nil,
		60*time.Second,
		5,
		0,
	)

	wp.Start()
	wp.Submit(task)

	// 等待任务执行完成
	wg.Wait()

	if executedTaskID != "task-exec-001" {
		t.Errorf("期望执行任务ID为 task-exec-001, 实际为 %s", executedTaskID)
	}

	// 队列应该为空
	if wp.taskQueue.Len() != 0 {
		t.Errorf("期望队列长度为0, 实际为 %d", wp.taskQueue.Len())
	}
}

func TestWorkerPool_ConcurrentSubmit(t *testing.T) {
	logger := zap.NewNop()
	wp := NewWorkerPool(4, logger)

	var submittedCount int32
	var executedCount int32
	var wg sync.WaitGroup

	wp.SetExecuteFunc(func(ctx context.Context, task *domain.Task) {
		atomic.AddInt32(&executedCount, 1)
		task.Start()
	})

	wp.Start()

	// 并发提交多个任务
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			task, _ := domain.NewTask(
				domain.NewTaskID("task-concurrent-"+string(rune('a'+id))),
				domain.NewTraceID("trace-1"),
				domain.NewSpanID("span-1"),
				nil,
				"并发任务",
				"",
				domain.TaskTypeCustom,
		"测试目标",
		"测试验收标准",
				nil,
				60*time.Second,
				id%10,
				0,
			)
			if wp.Submit(task) {
				atomic.AddInt32(&submittedCount, 1)
			}
		}(i)
	}

	wg.Wait()

	// 等待任务执行
	time.Sleep(500 * time.Millisecond)

	if atomic.LoadInt32(&submittedCount) != 20 {
		t.Errorf("期望提交20个任务, 实际为 %d", atomic.LoadInt32(&submittedCount))
	}

	if atomic.LoadInt32(&executedCount) == 0 {
		t.Error("期望至少执行一些任务")
	}
}

func TestWorkerPool_GracefulShutdown(t *testing.T) {
	logger := zap.NewNop()
	wp := NewWorkerPool(2, logger)

	var executedCount int32

	wp.SetExecuteFunc(func(ctx context.Context, task *domain.Task) {
		atomic.AddInt32(&executedCount, 1)
		task.Start()
		// 模拟耗时操作
		time.Sleep(100 * time.Millisecond)
	})

	wp.Start()

	// 提交任务
	for i := 0; i < 5; i++ {
		task, _ := domain.NewTask(
			domain.NewTaskID("task-shutdown-"+strconv.Itoa(i)),
			domain.NewTraceID("trace-1"),
			domain.NewSpanID("span-1"),
			nil,
			"关闭测试任务",
			"",
			domain.TaskTypeCustom,
		"测试目标",
		"测试验收标准",
			nil,
			60*time.Second,
			5,
			0,
		)
		wp.Submit(task)
	}

	// 等待一些任务开始执行
	time.Sleep(50 * time.Millisecond)

	// 优雅关闭
	wp.GracefulShutdown(2 * time.Second)

	// 工作池应该已关闭
	wp.mu.Lock()
	closed := wp.closed
	wp.mu.Unlock()

	if !closed {
		t.Error("期望工作池已关闭")
	}
}

func TestWorkerPool_MultipleWorkers(t *testing.T) {
	logger := zap.NewNop()
	wp := NewWorkerPool(3, logger)

	var executedTasks []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	wp.SetExecuteFunc(func(ctx context.Context, task *domain.Task) {
		mu.Lock()
		executedTasks = append(executedTasks, task.ID().String())
		mu.Unlock()
		task.Start()
		wg.Done()
	})

	wp.Start()

	// 提交6个任务
	for i := 0; i < 6; i++ {
		wg.Add(1)
		task, _ := domain.NewTask(
			domain.NewTaskID("task-multi-"+string(rune('a'+i))),
			domain.NewTraceID("trace-1"),
			domain.NewSpanID("span-1"),
			nil,
			"多Worker测试",
			"",
			domain.TaskTypeCustom,
		"测试目标",
		"测试验收标准",
			nil,
			60*time.Second,
			5,
			0,
		)
		wp.Submit(task)
	}

	wg.Wait()

	// 验证所有任务都被执行
	if len(executedTasks) != 6 {
		t.Errorf("期望执行6个任务, 实际为 %d", len(executedTasks))
	}

	wp.GracefulShutdown(1 * time.Second)
}

func TestWorkerPool_EmptyQueueBlocking(t *testing.T) {
	logger := zap.NewNop()
	wp := NewWorkerPool(1, logger)

	var workerStarted int32
	var wg sync.WaitGroup
	wg.Add(1)

	wp.SetExecuteFunc(func(ctx context.Context, task *domain.Task) {
		atomic.AddInt32(&workerStarted, 1)
		task.Start()
		wg.Done()
	})

	wp.Start()

	// 不提交任何任务，等待一下
	time.Sleep(50 * time.Millisecond)

	// 验证 worker 没有启动任务执行（因为没有任务）
	if atomic.LoadInt32(&workerStarted) != 0 {
		t.Error("没有任务时worker不应执行任务")
	}

	// 现在提交任务
	task, _ := domain.NewTask(
		domain.NewTaskID("task-late"),
		domain.NewTraceID("trace-1"),
		domain.NewSpanID("span-1"),
		nil,
		"延迟提交",
		"",
		domain.TaskTypeCustom,
		"测试目标",
		"测试验收标准",
		nil,
		60*time.Second,
		5,
		0,
	)
	wp.Submit(task)

	wg.Wait()

	if atomic.LoadInt32(&workerStarted) != 1 {
		t.Error("期望worker执行了任务")
	}

	wp.GracefulShutdown(1 * time.Second)
}
