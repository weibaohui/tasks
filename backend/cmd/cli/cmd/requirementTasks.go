package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
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

		ctx := context.Background()
		c := client.New()

		// 获取需求详情以获取 session_key
		requirement, err := c.GetRequirement(ctx, id)
		if err != nil {
			fmt.Printf(`{"error":"find requirement failed: %v"}`, err)
			return
		}

		sessionKey := requirement.DispatchSessionKey
		if sessionKey == "" {
			fmt.Printf(`{"requirement_id":"%s","message":"not dispatched yet"}`, id)
			return
		}

		// 获取任务列表
		tasks, _, err := c.GetRequirementTasks(ctx, id)
		if err != nil {
			fmt.Printf(`{"error":"find tasks failed: %v"}`, err)
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

		taskInfos := make([]TaskInfo, 0, len(tasks))
		for _, task := range tasks {
			taskInfos = append(taskInfos, TaskInfo{
				ID:       task.ID,
				Name:     task.Name,
				Status:   task.Status,
				Progress: task.Progress,
				ParentID: task.ParentID,
				Type:     task.Type,
			})
		}

		result := map[string]interface{}{
			"requirement_id": id,
			"session_key":    sessionKey,
			"tasks":          taskInfos,
		}

		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	requirementTasksCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
}
