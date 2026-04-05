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
	Short: "列出状态机的可用触发器",
	Example: `  taskmanager statemachine triggers <state-machine-id>
  taskmanager statemachine triggers <state-machine-id> -s submitted`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("状态机 ID 不能为空")
			return
		}

		id := args[0]
		currentState, _ := cmd.Flags().GetString("state")

		ctx := context.Background()
		c := client.New()

		sm, err := c.GetStateMachine(ctx, id)
		if err != nil {
			printJSONError("获取状态机详情失败: %v", err)
			return
		}

		type TriggerInfo struct {
			Trigger    string `json:"trigger"`
			FromState string `json:"from_state"`
			ToState   string `json:"to_state"`
			Description string `json:"description,omitempty"`
		}

		var triggers []TriggerInfo
		for _, t := range sm.Config.Transitions {
			if currentState == "" || t.From == currentState {
				triggers = append(triggers, TriggerInfo{
					Trigger:    t.Trigger,
					FromState: t.From,
					ToState:   t.To,
					Description: t.Description,
				})
			}
		}

		jsonBytes, _ := json.MarshalIndent(triggers, "", "  ")
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	stateMachineTriggersCmd.Flags().StringP("state", "s", "", "筛选指定状态的可用触发器")
}
