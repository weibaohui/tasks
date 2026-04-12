package cmd

import (
	"github.com/spf13/cobra"
)

var stateMachineCmd = &cobra.Command{
	Use:   "statemachine",
	Short: "状态机管理",
	Long:  `创建、管理状态机和触发状态转换`,
}

func registerStateMachineCommands() {
	stateMachineCmd.AddCommand(stateMachineListCmd)
	stateMachineCmd.AddCommand(stateMachineGetCmd)
	stateMachineCmd.AddCommand(stateMachineTriggersCmd)
	stateMachineCmd.AddCommand(stateMachineExecuteCmd)
	stateMachineCmd.AddCommand(stateMachineValidateCmd)
	stateMachineCmd.AddCommand(stateMachineGuideCmd)
}
