package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var stateMachineExecuteCmd = &cobra.Command{
	Use:   "execute",
	Short: "执行通用状态转换（纯计算，无业务绑定）",
	Long: `执行状态转换计算：输入状态机模板、当前状态、触发器，返回目标状态。

纯通用引擎：不管理业务实例ID，不做数据存储，只负责规则计算。
业务层自行管理实例ID和状态存储。

可通过 --metadata 注入自定义元数据，这些元数据会在状态转换时传递给 hook 执行上下文。`,
	Example: `  # 执行转换：从 build 状态 + 触发器 build_success
  taskmanager statemachine execute --machine=dev-release --from=build --trigger=build_success

  # 执行转换并注入元数据（供 hook 使用）
  taskmanager statemachine execute --machine=dev-release --from=build --trigger=build_success --metadata '{"requirement_id":"req-123","operator":"zhangsan"}'

  # 返回目标状态（业务层自行保存）
  # 输出: testing`,
	Run: func(cmd *cobra.Command, args []string) {
		machineName, _ := cmd.Flags().GetString("machine")
		currentState, _ := cmd.Flags().GetString("from")
		trigger, _ := cmd.Flags().GetString("trigger")
		metadataJSON, _ := cmd.Flags().GetString("metadata")

		if machineName == "" {
			printJSONError("必须指定 --machine 参数")
			return
		}
		if currentState == "" {
			printJSONError("必须指定 --from 参数")
			return
		}
		if trigger == "" {
			printJSONError("必须指定 --trigger 参数")
			return
		}

		// 解析 metadata
		var metadata map[string]interface{}
		if metadataJSON != "" {
			if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
				printJSONError("解析 metadata 失败: %v", err)
				return
			}
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

		// 查找匹配的转换
		var toState string
		var transitionFound bool
		var description string
		for _, t := range found.Config.Transitions {
			if t.From == currentState && t.Trigger == trigger {
				toState = t.To
				transitionFound = true
				description = t.Description
				break
			}
		}

		if !transitionFound {
			// 返回错误信息，包含可用触发器
			var availableTriggers []string
			for _, t := range found.Config.Transitions {
				if t.From == currentState {
					availableTriggers = append(availableTriggers, t.Trigger)
				}
			}

			result := map[string]interface{}{
				"error":     true,
				"message":   fmt.Sprintf("状态 '%s' 不支持触发器 '%s'", currentState, trigger),
				"machine":   machineName,
				"state":     currentState,
				"trigger":   trigger,
			}
			if len(availableTriggers) > 0 {
				result["available_triggers"] = availableTriggers
			}
			jsonBytes, _ := json.Marshal(result)
			fmt.Print(string(jsonBytes))
			return
		}

		// 成功返回目标状态（极简输出，业务层可直接使用）
		result := map[string]interface{}{
			"success":     true,
			"machine":     machineName,
			"from_state":  currentState,
			"to_state":    toState,
			"trigger":     trigger,
			"description": description,
		}

		// 如果提供了 metadata，包含在输出中
		if metadata != nil {
			result["metadata"] = metadata
		}

		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func registerStateMachineExecuteCommands() {
	stateMachineExecuteCmd.Flags().StringP("machine", "m", "", "状态机模板名称 (必填)")
	stateMachineExecuteCmd.Flags().StringP("from", "f", "", "当前状态ID (必填)")
	stateMachineExecuteCmd.Flags().StringP("trigger", "t", "", "触发器名称 (必填)")
	stateMachineExecuteCmd.Flags().StringP("metadata", "d", "", "自定义元数据JSON字符串（供hook使用）")
	stateMachineExecuteCmd.MarkFlagRequired("machine")
	stateMachineExecuteCmd.MarkFlagRequired("from")
	stateMachineExecuteCmd.MarkFlagRequired("trigger")
}
