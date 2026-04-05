package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var stateMachineValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "验证状态转换是否允许（从A到B）",
	Long:  `验证从指定源状态到目标状态是否存在有效的转换规则。返回所有可达路径的触发器列表。`,
	Example: `  taskmanager statemachine validate --machine=dev-release --from=todo --to=completed
  taskmanager statemachine validate -m dev-release -f code_review -t code_merged`,
	Run: func(cmd *cobra.Command, args []string) {
		machineName, _ := cmd.Flags().GetString("machine")
		fromState, _ := cmd.Flags().GetString("from")
		toState, _ := cmd.Flags().GetString("to")

		if machineName == "" {
			printJSONError("必须指定 --machine 参数")
			return
		}
		if fromState == "" {
			printJSONError("必须指定 --from 参数（源状态）")
			return
		}
		if toState == "" {
			printJSONError("必须指定 --to 参数（目标状态）")
			return
		}

		ctx := context.Background()
		c := client.New()

		// 获取所有状态机，按名称匹配
		sms, err := c.ListStateMachines(ctx)
		if err != nil {
			printJSONError("获取状态机列表失败: %v", err)
			return
		}

		// 按名称查找状态机
		var found *client.StateMachine
		for i := range sms {
			if sms[i].Name == machineName {
				found = &sms[i]
				break
			}
		}

		if found == nil {
			printJSONError("状态机 '%s' 不存在", machineName)
			return
		}

		// 验证源状态和目标状态是否存在
		var fromStateExists, toStateExists bool
		for _, s := range found.Config.States {
			if s.ID == fromState {
				fromStateExists = true
			}
			if s.ID == toState {
				toStateExists = true
			}
		}

		// 查找从 fromState 到 toState 的所有转换
		type TransitionInfo struct {
			Trigger     string `json:"trigger"`
			Description string `json:"description,omitempty"`
		}
		var availableTransitions []TransitionInfo

		for _, t := range found.Config.Transitions {
			if t.From == fromState && t.To == toState {
				availableTransitions = append(availableTransitions, TransitionInfo{
					Trigger:     t.Trigger,
					Description: t.Description,
				})
			}
		}

		result := map[string]interface{}{
			"machine":             machineName,
			"from":                fromState,
			"to":                  toState,
			"from_state_exists":   fromStateExists,
			"to_state_exists":     toStateExists,
			"valid":               len(availableTransitions) > 0,
			"transitions":         availableTransitions,
			"transitions_count":   len(availableTransitions),
		}

		if len(availableTransitions) > 0 {
			result["message"] = fmt.Sprintf("从 '%s' 到 '%s' 有 %d 个可用转换",
				fromState, toState, len(availableTransitions))
		} else {
			if !fromStateExists {
				result["message"] = fmt.Sprintf("源状态 '%s' 不存在于状态机 '%s'", fromState, machineName)
			} else if !toStateExists {
				result["message"] = fmt.Sprintf("目标状态 '%s' 不存在于状态机 '%s'", toState, machineName)
			} else {
				result["message"] = fmt.Sprintf("从 '%s' 到 '%s' 没有直接的转换规则", fromState, toState)
				// 提供从源状态出发的所有可能转换
				var fromStateTransitions []map[string]string
				for _, t := range found.Config.Transitions {
					if t.From == fromState {
						fromStateTransitions = append(fromStateTransitions, map[string]string{
							"trigger": t.Trigger,
							"to":      t.To,
						})
					}
				}
				if len(fromStateTransitions) > 0 {
					result["available_from_transitions"] = fromStateTransitions
				}
			}
		}

		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	stateMachineValidateCmd.Flags().StringP("machine", "m", "", "状态机模板名称 (必填)")
	stateMachineValidateCmd.Flags().StringP("from", "f", "", "源状态ID (必填)")
	stateMachineValidateCmd.Flags().StringP("to", "t", "", "目标状态ID (必填)")
	stateMachineValidateCmd.MarkFlagRequired("machine")
	stateMachineValidateCmd.MarkFlagRequired("from")
	stateMachineValidateCmd.MarkFlagRequired("to")
}
