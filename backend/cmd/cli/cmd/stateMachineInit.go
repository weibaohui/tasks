package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var stateMachineInitCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化需求状态",
	Example: `  taskmanager statemachine init <requirement-id> -s <state-machine-id>
  taskmanager statemachine init req-001 -s 45e6ad81-610a-4880-9d35-0377604ed2d3`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("需求 ID 不能为空")
			return
		}

		requirementID := args[0]
		stateMachineID, _ := cmd.Flags().GetString("state-machine")

		if stateMachineID == "" {
			printJSONError("状态机 ID 不能为空，请使用 -s 参数指定")
			return
		}

		ctx := context.Background()
		c := client.New()

		rs, err := c.InitializeRequirementState(ctx, requirementID, stateMachineID)
		if err != nil {
			printJSONError("初始化需求状态失败: %v", err)
			return
		}

		jsonBytes, _ := json.Marshal(rs)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	stateMachineInitCmd.Flags().StringP("state-machine", "s", "", "状态机 ID (必填)")
	stateMachineInitCmd.MarkFlagRequired("state-machine")
}