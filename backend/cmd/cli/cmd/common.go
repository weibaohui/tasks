package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// printJSONError 安全地输出 JSON 格式的错误信息
func printJSONError(format string, args ...interface{}) {
	errResult := map[string]string{
		"error": fmt.Sprintf(format, args...),
	}
	jsonBytes, _ := json.Marshal(errResult)
	fmt.Print(string(jsonBytes))
}

// AddCommands 显式注册所有子命令和 flags
func AddCommands() {
	rootCmd.AddCommand(serverCmd)
	registerServerCommands()
	rootCmd.AddCommand(tunnelCmd)
	registerTunnelCommands()
	rootCmd.AddCommand(createAdminCmd)
	registerCreateAdminCommands()
	rootCmd.AddCommand(deleteAdminCmd)
	registerDeleteAdminCommands()
	rootCmd.AddCommand(authCmd)
	registerAuthCommands()
	rootCmd.AddCommand(requirementCmd)
	registerRequirementCommands()
	registerRequirementCreateCommands()
	registerRequirementUpdateCommands()
	registerRequirementDispatchCommands()
	registerRequirementRedispatchCommands()
	registerRequirementCompleteCommands()
	registerRequirementListCommands()
	registerRequirementGetCommands()
	registerRequirementReviewCommands()
	registerRequirementResetCommands()
	registerRequirementGetStateCommands()
	registerRequirementUpdateStateCommands()
	registerRequirementTransitionCommands()
	rootCmd.AddCommand(requirementTypeCmd)
	registerRequirementTypeCommands()
	registerRequirementTypeCreateCommands()
	registerRequirementTypeListCommands()
	rootCmd.AddCommand(agentCmd)
	registerAgentCommands()
	rootCmd.AddCommand(projectCmd)
	registerProjectCommands()
	rootCmd.AddCommand(configCmd)
	registerConfigCommands()
	rootCmd.AddCommand(stateMachineCmd)
	registerStateMachineCommands()
	registerStateMachineExecuteCommands()
	registerStateMachineGetCommands()
	registerStateMachineGuideCommands()
	registerStateMachineTriggersCommands()
	registerStateMachineValidateCommands()
	rootCmd.AddCommand(stateCmd)
	registerStateCommands()
	rootCmd.AddCommand(hookCmd)
	registerHookCommands()
	registerHookListCommands()
	rootCmd.AddCommand(channelCmd)
	registerChannelCommands()
	rootCmd.AddCommand(providerCmd)
	registerProviderCommands()
	rootCmd.AddCommand(mcpCmd)
	registerMcpCommands()
	rootCmd.AddCommand(sessionCmd)
	registerSessionCommands()
	rootCmd.AddCommand(userCmd)
	registerUserCommands()
	rootCmd.AddCommand(skillCmd)
	registerSkillCommands()
}

// RegisterFlagErrorFunc 自定义错误处理
func RegisterFlagErrorFunc() {
	cobra.EnableCommandSorting = false
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return err
	})
}
