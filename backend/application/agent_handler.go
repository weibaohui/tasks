/**
 * AgentHandler - Agent 模式任务处理器
 * 使用 LLM 动态生成子任务并执行
 */
package application

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/llm"
	"github.com/weibh/taskmanager/infrastructure/utils"
)

const MaxAgentTaskDepth = 4

// AgentHandlerFunc Agent 模式任务处理函数 (实现 TaskHandler 接口)
func AgentHandlerFunc(ctx context.Context, task *domain.Task, repo domain.TaskRepository) error {
	taskID := task.ID().String()
	_ = task.TraceID().String() // traceID 可用于后续扩展
	spanID := task.SpanID().String()

	log.Printf("[AgentHandler] 执行 Agent 任务: %s, spanID: %s", taskID, spanID)

	// 获取当前深度
	currentDepth := 1
	if task.Metadata() != nil {
		if depthStr, ok := task.Metadata()["depth"].(string); ok {
			if depth, err := strconv.Atoi(depthStr); err == nil {
				currentDepth = depth + 1
			}
		}
	}

	// 更新进度
	updateAgentProgress(task, repo, 0, "初始化", fmt.Sprintf("Agent 模式启动，深度 %d/%d", currentDepth, MaxAgentTaskDepth))
	time.Sleep(AgentInitDelay)

	// 检查是否达到最大深度
	if currentDepth >= MaxAgentTaskDepth {
		updateAgentProgress(task, repo, 100, "完成", "达到最大深度，直接完成")
		return finishAgentTask(task, repo)
	}

	// Agent 模式下，直接完成（子任务生成由 AutoTaskExecutor 的 LLM 提供）
	updateAgentProgress(task, repo, 100, "完成", "Agent 任务完成")
	return finishAgentTask(task, repo)
}

// parseTaskType 解析任务类型字符串
func parseTaskType(typeStr string) domain.TaskType {
	switch typeStr {
	case "agent":
		return domain.TaskTypeAgent
	case "coding":
		return domain.TaskTypeCoding
	case "custom":
		return domain.TaskTypeCustom
	default:
		return domain.TaskTypeCustom
	}
}

func updateAgentProgress(task *domain.Task, repo domain.TaskRepository, progress int, stage, detail string) {
	task.UpdateProgress(100, progress, stage, detail)
	saveAgentTaskPreservingMetadata(task, repo)
}

func saveAgentTaskPreservingMetadata(task *domain.Task, repo domain.TaskRepository) {
	current, err := repo.FindByID(context.Background(), task.ID())
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
	repo.Save(context.Background(), task)
}

func finishAgentTask(task *domain.Task, repo domain.TaskRepository) error {
	resultData := map[string]interface{}{
		"completed_at": time.Now().UnixMilli(),
		"handler":      "agent",
	}
	result := domain.NewResult(resultData, "Agent 任务完成")
	task.Complete(result)
	updateAgentProgress(task, repo, 100, "完成", "Agent 任务执行完成")
	return nil
}

// CreateSubTasksFromLLM 根据 LLM 响应创建子任务
func CreateSubTasksFromLLM(
	ctx context.Context,
	task *domain.Task,
	repo domain.TaskRepository,
	plan *llm.SubTaskPlan,
) ([]string, error) {
	taskID := task.ID().String()
	traceID := task.TraceID().String()
	spanID := task.SpanID().String()

	subTaskIDs := make([]string, 0)
	idGen := utils.NewNanoIDGenerator(21)

	todoList := NewTodoList(taskID)

	for _, st := range plan.SubTasks {
		subTaskID := idGen.Generate()
		subSpanID := fmt.Sprintf("%s-%s", spanID, idGen.Generate()[:4])

		taskType := domain.TaskTypeAgent

		subTask, err := domain.NewTask(
			domain.NewTaskID(subTaskID),
			domain.NewTraceID(traceID),
			domain.NewSpanID(subSpanID),
			func() *domain.TaskID { pid := domain.NewTaskID(taskID); return &pid }(),
			st.Goal,
			fmt.Sprintf("LLM 生成的子任务: %s", st.Goal),
			taskType,
			map[string]interface{}{
				"goal":        st.Goal,
				"parent_id":   taskID,
				"parent_span": spanID,
				"depth":       strconv.Itoa(getCurrentDepth(task)),
				"llm_reason":  plan.Reason,
			},
			DefaultTaskTimeout,
			0,
			0,
		)
		if err != nil {
			log.Printf("[AgentHandler] 创建子任务失败: %v", err)
			continue
		}

		subTask.Start()
		if err := repo.Save(context.Background(), subTask); err != nil {
			log.Printf("[AgentHandler] 保存子任务失败: %v", err)
			continue
		}

		todoList.AddItem(subTaskID, st.Goal, taskType.String(), subSpanID, TodoStatusDistributed)
		subTaskIDs = append(subTaskIDs, subTaskID)

		log.Printf("[AgentHandler] 创建子任务: %s, spanID: %s, type: %s", subTaskID, subSpanID, taskType.String())
	}

	// 持久化 todo list
	if task.Metadata() == nil {
		task.SetMetadata(map[string]interface{}{})
	}
	task.Metadata()["todo_list"] = todoList.ToJSON()
	saveAgentTaskPreservingMetadata(task, repo)

	return subTaskIDs, nil
}

func getCurrentDepth(task *domain.Task) int {
	depth := 1
	if task.Metadata() != nil {
		if depthStr, ok := task.Metadata()["depth"].(string); ok {
			if d, err := strconv.Atoi(depthStr); err == nil {
				depth = d + 1
			}
		}
	}
	return depth
}
