package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var hookListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出 Hook 配置",
	Example: `  taskmanager hook list
  taskmanager hook list -p <project-id>`,
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project-id")

		ctx := context.Background()
		c := client.New()

		hooks, err := c.ListHookConfigs(ctx, projectID)
		if err != nil {
			printJSONError("列出 Hook 配置失败: %v", err)
			return
		}

		type HookInfo struct {
			ID           string `json:"id"`
			ProjectID   string `json:"project_id"`
			Name        string `json:"name"`
			TriggerPoint string `json:"trigger_point"`
			ActionType  string `json:"action_type"`
			Enabled     bool   `json:"enabled"`
			Priority    int    `json:"priority"`
		}

		items := make([]HookInfo, 0, len(hooks))
		for _, h := range hooks {
			items = append(items, HookInfo{
				ID:           h.ID,
				ProjectID:   h.ProjectID,
				Name:        h.Name,
				TriggerPoint: h.TriggerPoint,
				ActionType:  h.ActionType,
				Enabled:     h.Enabled,
				Priority:    h.Priority,
			})
		}

		jsonBytes, _ := json.Marshal(items)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	hookListCmd.Flags().StringP("project-id", "p", "", "项目 ID (可选)")
}