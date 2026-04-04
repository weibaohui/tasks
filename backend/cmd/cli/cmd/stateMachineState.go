package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var stateMachineStateCmd = &cobra.Command{
	Use:   "state",
	Short: "获取需求当前状态",
	Example: `  taskmanager statemachine state <requirement-id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("需求 ID 不能为空")
			return
		}

		requirementID := args[0]

		ctx := context.Background()
		c := client.New()

		state, err := c.GetRequirementState(ctx, requirementID)
		if err != nil {
			printJSONError("获取需求状态失败: %v", err)
			return
		}

		jsonBytes, _ := json.Marshal(state)
		fmt.Print(string(jsonBytes))
	},
}