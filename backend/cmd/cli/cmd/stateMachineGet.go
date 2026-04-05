package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var stateMachineGetCmd = &cobra.Command{
	Use:   "get",
	Short: "获取状态机详情",
	Example: `  taskmanager statemachine get <state-machine-id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("状态机 ID 不能为空")
			return
		}

		id := args[0]

		ctx := context.Background()
		c := client.New()

		sm, err := c.GetStateMachine(ctx, id)
		if err != nil {
			printJSONError("获取状态机详情失败: %v", err)
			return
		}

		jsonBytes, _ := json.MarshalIndent(sm, "", "  ")
		fmt.Print(string(jsonBytes))
	},
}
