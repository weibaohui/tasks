package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var requirementUpdateStateCmd = &cobra.Command{
	Use:   "update-state",
	Short: "更新需求状态",
	Long: `更新需求的 status 字段。通常用于将状态机执行结果同步到需求状态。

典型工作流:
  1. taskmanager requirement get-state --id <req-id>
  2. taskmanager statemachine execute --machine=<name> --from=<state> --trigger=<trigger>
  3. taskmanager requirement update-state --id <req-id> --status <new-state>`,
	Example: `  # 更新需求状态为 completed
  taskmanager requirement update-state --id req-abc-123 --status completed

  # 简写形式
  taskmanager requirement update-state -i req-abc-123 -s completed`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")
		status, _ := cmd.Flags().GetString("status")

		if id == "" {
			result := map[string]interface{}{
				"error":   true,
				"message": "--id is required",
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
			return
		}

		if status == "" {
			result := map[string]interface{}{
				"error":   true,
				"message": "--status is required",
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
			return
		}

		ctx := context.Background()
		c := client.New()

		// 先获取当前需求信息
		req, err := c.GetRequirement(ctx, id)
		if err != nil {
			result := map[string]interface{}{
				"error":         true,
				"requirement_id": id,
				"message":       fmt.Sprintf("获取需求失败: %v", err),
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
			return
		}

		oldStatus := req.Status

		// 更新需求状态
		updatedReq, err := c.UpdateRequirement(ctx, client.UpdateRequirementRequest{
			ID:     id,
			Status: &status,
		})
		if err != nil {
			result := map[string]interface{}{
				"error":         true,
				"requirement_id": id,
				"message":       fmt.Sprintf("更新状态失败: %v", err),
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
			return
		}

		// 返回成功结果
		result := map[string]interface{}{
			"success":         true,
			"requirement_id":  id,
			"old_status":      oldStatus,
			"new_status":      updatedReq.Status,
			"message":         fmt.Sprintf("状态已从 '%s' 更新为 '%s'", oldStatus, updatedReq.Status),
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func registerRequirementUpdateStateCommands() {
	requirementUpdateStateCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
	requirementUpdateStateCmd.Flags().StringP("status", "s", "", "新状态 (必填)")
	requirementUpdateStateCmd.MarkFlagRequired("id")
	requirementUpdateStateCmd.MarkFlagRequired("status")
}
