package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var stateMachineBindCmd = &cobra.Command{
	Use:   "bind",
	Short: "绑定需求类型到状态机",
	Example: `  taskmanager statemachine bind <state-machine-id> -t heartbeat
  taskmanager statemachine bind <state-machine-id> -t normal`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("状态机 ID 不能为空")
			return
		}

		stateMachineID := args[0]
		requirementType, _ := cmd.Flags().GetString("type")

		if requirementType == "" {
			printJSONError("需求类型不能为空")
			return
		}

		ctx := context.Background()
		c := client.New()

		err := c.BindType(ctx, stateMachineID, requirementType)
		if err != nil {
			printJSONError("绑定失败: %v", err)
			return
		}

		fmt.Print(`{"message":"绑定成功"}`)
	},
}

func init() {
	stateMachineBindCmd.Flags().StringP("type", "t", "", "需求类型 (必填: heartbeat/normal)")
	stateMachineBindCmd.MarkFlagRequired("type")
}