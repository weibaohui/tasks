/**
 * WorkerPool 工作池
 * 负责管理工作线程和任务队列
 */
package application

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"github.com/weibh/taskmanager/domain"
	"go.uber.org/zap"
)

// TaskItem 任务项，用于优先级队列
type TaskItem struct {
	task     *domain.Task
	priority int
	index    int
}

// PriorityQueue 优先级队列
type PriorityQueue []*TaskItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].priority > pq[j].priority
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*TaskItem)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*pq = old[0 : n-1]
	return item
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// WorkerPool 工作池
type WorkerPool struct {
	workers    int
	taskQueue  *PriorityQueue
	mu         sync.Mutex
	cond       *sync.Cond
	wg         sync.WaitGroup
	logger     *zap.Logger
	closed     bool
	executeFn  func(ctx context.Context, task *domain.Task)
}

// NewWorkerPool 创建工作池
func NewWorkerPool(workers int, logger *zap.Logger) *WorkerPool {
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)

	wp := &WorkerPool{
		workers:   workers,
		taskQueue: &pq,
		logger:    logger,
		closed:    false,
	}

	heap.Init(wp.taskQueue)
	wp.cond = sync.NewCond(&wp.mu)

	return wp
}

// SetExecuteFunc 设置任务执行函数
func (wp *WorkerPool) SetExecuteFunc(fn func(ctx context.Context, task *domain.Task)) {
	wp.executeFn = fn
}

// Submit 提交任务
func (wp *WorkerPool) Submit(task *domain.Task) bool {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.closed {
		return false
	}

	item := &TaskItem{
		task:     task,
		priority: task.Priority(),
	}

	heap.Push(wp.taskQueue, item)
	wp.cond.Signal()

	return true
}

// Start 启动工作池
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
	wp.logger.Info("WorkerPool 已启动", zap.Int("workers", wp.workers))
}

// worker 工作协程
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	wp.mu.Lock()
	for !wp.closed {
		if wp.taskQueue.Len() == 0 {
			wp.cond.Wait()
			continue
		}

		item := heap.Pop(wp.taskQueue).(*TaskItem)
		task := item.task

		wp.mu.Unlock()

		// 创建任务上下文
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// 执行任务
		if wp.executeFn != nil {
			wp.logger.Info("开始执行任务", zap.String("taskID", task.ID().String()), zap.Int("worker", id))
			wp.executeFn(ctx, task)
		}

		wp.mu.Lock()
	}
	wp.mu.Unlock()

	wp.logger.Info("Worker 退出", zap.Int("worker", id))
}

// GracefulShutdown 优雅关闭
func (wp *WorkerPool) GracefulShutdown(timeout time.Duration) {
	wp.mu.Lock()
	wp.closed = true
	wp.cond.Broadcast()
	wp.mu.Unlock()

	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		wp.logger.Info("WorkerPool 优雅关闭完成")
	case <-time.After(timeout):
		wp.logger.Warn("WorkerPool 关闭超时")
	}
}
