package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var stateMachineListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有状态机模板",
	Long:  `列出系统中所有可用的通用状态机模板定义。`,
	Example: `  taskmanager statemachine list`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		c := client.New()

		sms, err := c.ListStateMachines(ctx)
		if err != nil {
			printJSONError("列出状态机失败: %v", err)
			return
		}

		type StateMachineInfo struct {
			Name             string `json:"name"`
			Description      string `json:"description"`
			InitialState     string `json:"initial_state"`
			StatesCount      int    `json:"states_count"`
			TransitionsCount int    `json:"transitions_count"`
		}

		items := make([]StateMachineInfo, 0, len(sms))
		for _, sm := range sms {
			items = append(items, StateMachineInfo{
				Name:             sm.Name,
				Description:      sm.Description,
				InitialState:     sm.Config.InitialState,
				StatesCount:      len(sm.Config.States),
				TransitionsCount: len(sm.Config.Transitions),
			})
		}

		jsonBytes, _ := json.Marshal(items)
		fmt.Print(string(jsonBytes))
	},
}
