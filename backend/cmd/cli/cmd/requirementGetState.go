package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var requirementGetStateCmd = &cobra.Command{
	Use:   "get-state",
	Short: "获取需求当前状态",
	Long:  `获取需求在状态机中的当前状态。如果需求尚未初始化状态机，则返回错误。`,
	Example: `  taskmanager requirement get-state --id <requirement-id>
  taskmanager requirement get-state -i req-abc-123`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")

		if id == "" {
			fmt.Print(`{"error":"--id is required"}`)
			return
		}

		ctx := context.Background()
		c := client.New()

		// 获取需求状态机状态
		state, err := c.GetRequirementState(ctx, id)
		if err != nil {
			// 可能是需求不存在或未初始化状态机
			result := map[string]interface{}{
				"error":         true,
				"requirement_id": id,
				"message":       fmt.Sprintf("获取需求状态失败: %v", err),
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
			return
		}

		// 同时获取需求基本信息（包含 status 字段）
		req, err := c.GetRequirement(ctx, id)
		if err != nil {
			// 只返回状态机状态
			result := map[string]interface{}{
				"requirement_id":   id,
				"state_machine_id": state.StateMachineID,
				"current_state":    state.CurrentState,
				"state_name":       state.CurrentStateName,
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
			return
		}

		// 返回完整状态信息
		result := map[string]interface{}{
			"requirement_id":   id,
			"state_machine_id": state.StateMachineID,
			"current_state":    state.CurrentState,
			"state_name":       state.CurrentStateName,
			"requirement_status": req.Status,
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	requirementGetStateCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
	requirementGetStateCmd.MarkFlagRequired("id")
}
