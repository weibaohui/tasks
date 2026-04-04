package cmd

import (
	"github.com/spf13/cobra"
)

var stateMachineCmd = &cobra.Command{
	Use:   "statemachine",
	Short: "状态机管理",
	Long:  `创建、管理状态机和触发状态转换`,
}

func init() {
	stateMachineCmd.AddCommand(stateMachineCreateCmd)
	stateMachineCmd.AddCommand(stateMachineListCmd)
	stateMachineCmd.AddCommand(stateMachineDeleteCmd)
	stateMachineCmd.AddCommand(stateMachineTransitionCmd)
	stateMachineCmd.AddCommand(stateMachineStateCmd)
	stateMachineCmd.AddCommand(stateMachineBindCmd)
	stateMachineCmd.AddCommand(stateMachineHistoryCmd)
}