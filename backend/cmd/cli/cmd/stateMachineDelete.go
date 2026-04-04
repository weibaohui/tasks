package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var stateMachineDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "删除状态机",
	Example: `  taskmanager statemachine delete <state-machine-id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("状态机 ID 不能为空")
			return
		}

		id := args[0]

		ctx := context.Background()
		c := client.New()

		err := c.DeleteStateMachine(ctx, id)
		if err != nil {
			printJSONError("删除状态机失败: %v", err)
			return
		}

		fmt.Print(`{"message":"删除成功"}`)
	},
}