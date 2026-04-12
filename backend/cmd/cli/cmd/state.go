package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "状态统计和历史",
	Long:  `获取项目状态统计和需求转换历史`,
}

func registerStateCommands() {
	stateCmd.AddCommand(stateSummaryCmd)
	stateCmd.AddCommand(stateHistoryCmd)
}

// stateSummaryCmd 项目状态统计
var stateSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "获取项目状态统计",
	Example: `  taskmanager state summary <project-id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("项目 ID 不能为空")
			return
		}

		projectID := args[0]

		ctx := context.Background()
		c := client.New()

		summary, err := c.GetProjectStateSummary(ctx, projectID)
		if err != nil {
			printJSONError("获取状态统计失败: %v", err)
			return
		}

		jsonBytes, _ := json.Marshal(summary)
		fmt.Print(string(jsonBytes))
	},
}

// stateHistoryCmd 需求转换历史
var stateHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "获取需求转换历史",
	Example: `  taskmanager state history <requirement-id>`,
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