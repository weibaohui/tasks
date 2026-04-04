package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var stateMachineTransitionCmd = &cobra.Command{
	Use:   "transition",
	Short: "触发状态转换",
	Example: `  taskmanager statemachine transition <requirement-id> -t start
  taskmanager statemachine transition <requirement-id> -t complete -b "user" -r "备注"`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("需求 ID 不能为空")
			return
		}

		requirementID := args[0]
		trigger, _ := cmd.Flags().GetString("trigger")
		triggeredBy, _ := cmd.Flags().GetString("by")
		remark, _ := cmd.Flags().GetString("remark")

		if trigger == "" {
			printJSONError("触发器不能为空")
			return
		}

		ctx := context.Background()
		c := client.New()

		// 先获取需求当前状态
		state, err := c.GetRequirementState(ctx, requirementID)
		if err != nil {
			printJSONError("获取需求状态失败: %v", err)
			return
		}

		// 触发转换
		err = c.TriggerTransition(ctx, requirementID, trigger, triggeredBy, remark)
		if err != nil {
			printJSONError("触发状态转换失败: %v", err)
			return
		}

		// 获取新的状态
		newState, err := c.GetRequirementState(ctx, requirementID)
		if err != nil {
			printJSONError("触发成功但获取新状态失败: %v", err)
			return
		}

		result := map[string]interface{}{
			"requirement_id": requirementID,
			"from_state":    state.CurrentState,
			"to_state":      newState.CurrentState,
			"trigger":       trigger,
			"triggered_by":  triggeredBy,
		}

		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	stateMachineTransitionCmd.Flags().StringP("trigger", "t", "", "触发器名称 (必填)")
	stateMachineTransitionCmd.Flags().StringP("by", "b", "cli", "触发者")
	stateMachineTransitionCmd.Flags().StringP("remark", "r", "", "备注")
	stateMachineTransitionCmd.MarkFlagRequired("trigger")
}