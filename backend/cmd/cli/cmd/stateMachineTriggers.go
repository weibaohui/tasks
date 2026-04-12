package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var stateMachineTriggersCmd = &cobra.Command{
	Use:   "triggers",
	Short: "查询指定状态的可用触发器",
	Long:  `输入状态机模板名称和当前状态，返回该状态下所有可用的触发器列表。`,
	Example: `  taskmanager statemachine triggers --machine=dev-release --from=code_review`,
	Run: func(cmd *cobra.Command, args []string) {
		machineName, _ := cmd.Flags().GetString("machine")
		currentState, _ := cmd.Flags().GetString("from")

		if machineName == "" {
			printJSONError("必须指定 --machine 参数")
			return
		}
		if currentState == "" {
			printJSONError("必须指定 --from 参数")
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

		type TriggerInfo struct {
			Trigger     string `json:"trigger"`
			ToState     string `json:"to_state"`
			Description string `json:"description,omitempty"`
		}

		var triggers []TriggerInfo
		for _, t := range found.Config.Transitions {
			if t.From == currentState {
				triggers = append(triggers, TriggerInfo{
					Trigger:     t.Trigger,
					ToState:     t.To,
					Description: t.Description,
				})
			}
		}

		result := map[string]interface{}{
			"machine":       machineName,
			"current_state": currentState,
			"triggers":      triggers,
			"count":         len(triggers),
		}

		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func registerStateMachineTriggersCommands() {
	stateMachineTriggersCmd.Flags().StringP("machine", "m", "", "状态机模板名称 (必填)")
	stateMachineTriggersCmd.Flags().StringP("from", "f", "", "当前状态ID (必填)")
	stateMachineTriggersCmd.MarkFlagRequired("machine")
	stateMachineTriggersCmd.MarkFlagRequired("from")
}
