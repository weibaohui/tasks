package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var stateMachineHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "获取需求转换历史",
	Example: `  taskmanager statemachine history <requirement-id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("需求 ID 不能为空")
			return
		}

		requirementID := args[0]

		ctx := context.Background()
		c := client.New()

		history, err := c.GetTransitionHistory(ctx, requirementID)
		if err != nil {
			printJSONError("获取转换历史失败: %v", err)
			return
		}

		jsonBytes, _ := json.Marshal(history)
		fmt.Print(string(jsonBytes))
	},
}