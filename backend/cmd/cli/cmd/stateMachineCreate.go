package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var stateMachineCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建状态机",
	Example: `  taskmanager statemachine create -n "需求流程" -d "描述" -c '{"name":"flow","initial_state":"created","states":[{"id":"created","name":"已创建","is_final":false}],"transitions":[]}'`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		config, _ := cmd.Flags().GetString("config")

		if name == "" {
			printJSONError("状态机名称不能为空")
			return
		}
		if config == "" {
			printJSONError("状态机配置不能为空")
			return
		}

		ctx := context.Background()
		c := client.New()

		sm, err := c.CreateStateMachine(ctx, name, description, config)
		if err != nil {
			printJSONError("创建状态机失败: %v", err)
			return
		}

		jsonBytes, _ := json.Marshal(sm)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	stateMachineCreateCmd.Flags().StringP("name", "n", "", "状态机名称 (必填)")
	stateMachineCreateCmd.Flags().StringP("description", "d", "", "状态机描述")
	stateMachineCreateCmd.Flags().StringP("config", "c", "", "状态机配置 JSON (必填)")
	stateMachineCreateCmd.MarkFlagRequired("name")
	stateMachineCreateCmd.MarkFlagRequired("config")
}
