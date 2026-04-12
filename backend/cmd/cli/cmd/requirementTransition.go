package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var requirementTransitionCmd = &cobra.Command{
	Use:   "transition",
	Short: "执行需求状态转换（封装状态机+需求更新）",
	Long: `通过状态机执行转换，并同步更新需求状态。

这是一个封装命令，内部执行：
1. 调用状态机执行转换计算
2. 更新需求状态为转换后的目标状态

注意：需求必须已初始化状态机关联。`,
	Example: `  # 执行状态转换：从当前状态 + 触发器
  taskmanager requirement transition --id req-123 --trigger complete

  # 带元数据（传递给 hook 使用）
  taskmanager requirement transition --id req-123 --trigger complete --metadata '{"operator":"ai-agent"}'

  # 典型工作流
  taskmanager requirement transition --id req-abc-123 --trigger finish --metadata '{"result":"success"}'`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")
		trigger, _ := cmd.Flags().GetString("trigger")
		triggeredBy, _ := cmd.Flags().GetString("triggered-by")
		remark, _ := cmd.Flags().GetString("remark")
		metadataJSON, _ := cmd.Flags().GetString("metadata")

		if id == "" {
			result := map[string]interface{}{
				"error":   true,
				"message": "--id is required",
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
			return
		}

		if trigger == "" {
			result := map[string]interface{}{
				"error":   true,
				"message": "--trigger is required",
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
			return
		}

		// 解析 metadata
		var metadata map[string]interface{}
		if metadataJSON != "" {
			if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
				result := map[string]interface{}{
					"error":   true,
					"message": fmt.Sprintf("解析 metadata 失败: %v", err),
				}
				jsonBytes, _ := json.Marshal(result)
				fmt.Print(string(jsonBytes))
				return
			}
		}

		ctx := context.Background()
		c := client.New()

		// Step 1: 获取需求当前状态（用于返回信息）
		stateInfo, err := c.GetRequirementState(ctx, id)
		if err != nil {
			result := map[string]interface{}{
				"error":          true,
				"requirement_id": id,
				"message":        fmt.Sprintf("获取需求状态失败: %v", err),
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
			return
		}
		fromState := stateInfo.CurrentState

		// Step 2: 执行状态机转换
		// 这会更新 RequirementState，并返回新的状态
		err = c.TriggerTransition(ctx, id, trigger, triggeredBy, remark, metadata)
		if err != nil {
			result := map[string]interface{}{
				"error":          true,
				"requirement_id": id,
				"from_state":     fromState,
				"trigger":        trigger,
				"message":        fmt.Sprintf("状态机转换失败: %v", err),
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
			return
		}

		// Step 3: 获取转换后的新状态
		newStateInfo, err := c.GetRequirementState(ctx, id)
		if err != nil {
			result := map[string]interface{}{
				"error":          true,
				"requirement_id": id,
				"message":        fmt.Sprintf("获取新状态失败: %v", err),
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
			return
		}
		toState := newStateInfo.CurrentState

		// Step 4: 同步更新需求业务实体的 status 字段
		_, err = c.UpdateRequirement(ctx, client.UpdateRequirementRequest{
			ID:     id,
			Status: &toState,
		})
		if err != nil {
			result := map[string]interface{}{
				"error":          true,
				"requirement_id": id,
				"from_state":     fromState,
				"to_state":       toState,
				"trigger":        trigger,
				"message":        fmt.Sprintf("更新需求状态失败: %v", err),
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
			return
		}

		// 返回成功结果
		result := map[string]interface{}{
			"success":          true,
			"requirement_id":   id,
			"from_state":       fromState,
			"to_state":         toState,
			"to_state_name":    newStateInfo.CurrentStateName,
			"trigger":          trigger,
			"message":          fmt.Sprintf("状态已从 '%s' 转换为 '%s'", fromState, toState),
		}
		if metadata != nil {
			result["metadata"] = metadata
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func registerRequirementTransitionCommands() {
	requirementTransitionCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
	requirementTransitionCmd.Flags().StringP("trigger", "t", "", "触发器名称 (必填)")
	requirementTransitionCmd.Flags().StringP("triggered-by", "b", "cli", "触发者标识")
	requirementTransitionCmd.Flags().StringP("remark", "r", "", "转换备注")
	requirementTransitionCmd.Flags().StringP("metadata", "d", "", "自定义元数据JSON字符串（传递给hook）")
	requirementTransitionCmd.MarkFlagRequired("id")
	requirementTransitionCmd.MarkFlagRequired("trigger")
}
