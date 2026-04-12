package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var stateMachineGetCmd = &cobra.Command{
	Use:   "get",
	Short: "获取状态机模板详情",
	Long:  `获取指定状态机模板的完整规则定义（状态列表+流转规则）。`,
	Example: `  taskmanager statemachine get --machine=dev-release`,
	Run: func(cmd *cobra.Command, args []string) {
		machineName, _ := cmd.Flags().GetString("machine")
		if machineName == "" {
			printJSONError("必须指定 --machine 参数")
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

		// 构造纯规则输出（无ID、无业务数据）
		type StateInfo struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			IsFinal bool   `json:"is_final"`
		}

		type TransitionInfo struct {
			From        string `json:"from"`
			To          string `json:"to"`
			Trigger     string `json:"trigger"`
			Description string `json:"description,omitempty"`
		}

		type MachineTemplate struct {
			Name         string           `json:"name"`
			Description  string           `json:"description"`
			InitialState string           `json:"initial_state"`
			States       []StateInfo      `json:"states"`
			Transitions  []TransitionInfo `json:"transitions"`
		}

		result := MachineTemplate{
			Name:         found.Name,
			Description:  found.Description,
			InitialState: found.Config.InitialState,
		}

		for _, s := range found.Config.States {
			result.States = append(result.States, StateInfo{
				ID:      s.ID,
				Name:    s.Name,
				IsFinal: s.IsFinal,
			})
		}

		for _, t := range found.Config.Transitions {
			result.Transitions = append(result.Transitions, TransitionInfo{
				From:        t.From,
				To:          t.To,
				Trigger:     t.Trigger,
				Description: t.Description,
			})
		}

		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func registerStateMachineGetCommands() {
	stateMachineGetCmd.Flags().StringP("machine", "m", "", "状态机模板名称 (必填)")
	stateMachineGetCmd.MarkFlagRequired("machine")
}
