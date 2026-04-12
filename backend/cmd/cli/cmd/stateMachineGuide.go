package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var stateMachineGuideCmd = &cobra.Command{
	Use:   "guide",
	Short: "获取状态的 AI 执行指南",
	Long: `获取指定状态机中某个状态的完整 AI 指南信息。

包括：
- AI 操作指南（当前阶段应该做什么）
- 自动初始化命令（进入状态时自动执行）
- 成功/失败判断标准
- 可用的触发器及选择条件`,
	Example: `  # 获取指定状态的 AI 指南
  taskmanager statemachine guide --machine=dev_workflow --state=coding

  # 查看当前需求所在状态的指南
  CURRENT_STATE=$(taskmanager requirement get-state --id req-123 | jq -r '.current_state')
  taskmanager statemachine guide --machine=dev_workflow --state="$CURRENT_STATE"`,
	Run: func(cmd *cobra.Command, args []string) {
		machineName, _ := cmd.Flags().GetString("machine")
		stateID, _ := cmd.Flags().GetString("state")

		if machineName == "" {
			result := map[string]interface{}{
				"error":   true,
				"message": "必须指定 --machine 参数",
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
			return
		}

		if stateID == "" {
			result := map[string]interface{}{
				"error":   true,
				"message": "必须指定 --state 参数",
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
			return
		}

		ctx := context.Background()
		c := client.New()

		// 获取所有状态机
		sms, err := c.ListStateMachines(ctx)
		if err != nil {
			result := map[string]interface{}{
				"error":   true,
				"message": fmt.Sprintf("获取状态机列表失败: %v", err),
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
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
			result := map[string]interface{}{
				"error":   true,
				"message": fmt.Sprintf("状态机 '%s' 不存在", machineName),
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
			return
		}

		// 查找指定状态
		var stateFound *client.State
		for i := range found.Config.States {
			if found.Config.States[i].ID == stateID {
				stateFound = &found.Config.States[i]
				break
			}
		}

		if stateFound == nil {
			result := map[string]interface{}{
				"error":    true,
				"message":  fmt.Sprintf("状态 '%s' 不存在于状态机 '%s'", stateID, machineName),
				"machine":  machineName,
				"state":    stateID,
				"available_states": getStateIDs(found.Config.States),
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
			return
		}

		// 获取可用的触发器
		triggers := getTriggersForState(found.Config, stateID)

		// 构建响应
		result := map[string]interface{}{
			"machine":          machineName,
			"state":            stateFound.ID,
			"name":             stateFound.Name,
			"is_final":         stateFound.IsFinal,
			"ai_guide":         stateFound.AIGuide,
			"auto_init":        stateFound.AutoInit,
			"success_criteria": stateFound.SuccessCriteria,
			"failure_criteria": stateFound.FailureCriteria,
			"triggers":         triggers,
		}

		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

// getStateIDs 获取状态 ID 列表
func getStateIDs(states []client.State) []string {
	ids := make([]string, len(states))
	for i, s := range states {
		ids[i] = s.ID
	}
	return ids
}

// getTriggersForState 获取状态可用的触发器
func getTriggersForState(config client.StateMachineConfig, stateID string) []map[string]string {
	var triggers []map[string]string

	// 首先从状态的 triggers 字段获取
	for _, s := range config.States {
		if s.ID == stateID && len(s.Triggers) > 0 {
			for _, t := range s.Triggers {
				triggers = append(triggers, map[string]string{
					"trigger":     t.Trigger,
					"description": t.Description,
					"condition":   t.Condition,
				})
			}
			return triggers
		}
	}

	// 否则从 transitions 中推导
	for _, t := range config.Transitions {
		if t.From == stateID {
			triggers = append(triggers, map[string]string{
				"trigger":     t.Trigger,
				"description": t.Description,
				"condition":   "",
			})
		}
	}
	return triggers
}

func registerStateMachineGuideCommands() {
	stateMachineCmd.AddCommand(stateMachineGuideCmd)
	stateMachineGuideCmd.Flags().StringP("machine", "m", "", "状态机模板名称 (必填)")
	stateMachineGuideCmd.Flags().StringP("state", "s", "", "状态 ID (必填)")
	stateMachineGuideCmd.MarkFlagRequired("machine")
	stateMachineGuideCmd.MarkFlagRequired("state")
}
