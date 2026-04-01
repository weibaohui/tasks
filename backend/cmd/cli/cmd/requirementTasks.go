package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/domain"
)

var requirementTasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "查看需求的子任务",
	Example: `  taskmanager requirement tasks --id <requirement_id>`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")

		if id == "" {
			fmt.Print(`{"error":"--id is required"}`)
			return
		}

		requirementRepo, _, _, _, reqCleanup := getRequirementRepos()
		defer reqCleanup()

		taskRepo, taskCleanup := getTaskRepo()
		defer taskCleanup()

		ctx := context.Background()

		// 查找需求
		requirement, err := requirementRepo.FindByID(ctx, domain.NewRequirementID(id))
		if err != nil {
			fmt.Printf(`{"error":"find requirement failed: %v"}`, err)
			return
		}
		if requirement == nil {
			fmt.Printf(`{"error":"requirement not found: %s"}`, id)
			return
		}

		// 尝试通过 session_key 查找相关任务
		sessionKey := requirement.DispatchSessionKey()
		if sessionKey == "" {
			fmt.Printf(`{"requirement_id":"%s","message":"not dispatched yet"}`, id)
			return
		}

		// 查找所有任务，然后筛选
		allTasks, err := taskRepo.FindAll(ctx)
		if err != nil {
			fmt.Printf(`{"error":"find tasks failed: %v"}`, err)
			return
		}

		// 筛选与该需求相关的任务（通过 session_key 匹配）
		var relatedTasks []*domain.Task
		for _, task := range allTasks {
			if task.SessionKey() == sessionKey {
				relatedTasks = append(relatedTasks, task)
			}
		}

		if len(relatedTasks) == 0 {
			fmt.Printf(`{"requirement_id":"%s","tasks":[]}`, id)
			return
		}

		// 构建 JSON 输出
		type TaskInfo struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Status   string `json:"status"`
			Progress int    `json:"progress"`
			ParentID string `json:"parent_id,omitempty"`
			Type     string `json:"type"`
		}

		tasks := make([]TaskInfo, 0, len(relatedTasks))
		for _, task := range relatedTasks {
			var parentID string
			if task.ParentID() != nil {
				parentID = task.ParentID().String()
			}
			tasks = append(tasks, TaskInfo{
				ID:       task.ID().String(),
				Name:     task.Name(),
				Status:   task.Status().String(),
				Progress: task.Progress().Value(),
				ParentID: parentID,
				Type:     task.Type().String(),
			})
		}

		result := map[string]interface{}{
			"requirement_id": id,
			"session_key":    sessionKey,
			"tasks":          tasks,
		}

		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	requirementTasksCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
}
