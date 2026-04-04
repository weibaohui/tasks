package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var stateMachineListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出状态机",
	Example: `  taskmanager statemachine list -p <project-id>`,
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project-id")

		if projectID == "" {
			printJSONError("项目 ID 不能为空")
			return
		}

		ctx := context.Background()
		c := client.New()

		sms, err := c.ListStateMachines(ctx, projectID)
		if err != nil {
			printJSONError("列出状态机失败: %v", err)
			return
		}

		type StateMachineInfo struct {
			ID          string `json:"id"`
			ProjectID   string `json:"project_id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			InitialState string `json:"initial_state"`
			StatesCount int    `json:"states_count"`
			TransitionsCount int `json:"transitions_count"`
			CreatedAt   int64  `json:"created_at"`
		}

		items := make([]StateMachineInfo, 0, len(sms))
		for _, sm := range sms {
			items = append(items, StateMachineInfo{
				ID:          sm.ID,
				ProjectID:   sm.ProjectID,
				Name:        sm.Name,
				Description: sm.Description,
				InitialState: sm.Config.InitialState,
				StatesCount: len(sm.Config.States),
				TransitionsCount: len(sm.Config.Transitions),
				CreatedAt:   sm.CreatedAt,
			})
		}

		jsonBytes, _ := json.Marshal(items)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	stateMachineListCmd.Flags().StringP("project-id", "p", "", "项目 ID (必填)")
	stateMachineListCmd.MarkFlagRequired("project-id")
}