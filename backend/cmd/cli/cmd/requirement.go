package cmd

import (
	"github.com/spf13/cobra"
)

var requirementCmd = &cobra.Command{
	Use:   "requirement",
	Short: "需求管理",
	Long:  `创建、更新、派发和查看需求`,
}

func registerRequirementCommands() {
	requirementCmd.AddCommand(requirementCreateCmd)
	requirementCmd.AddCommand(requirementUpdateCmd)
	requirementCmd.AddCommand(requirementDeleteCmd)
	requirementCmd.AddCommand(requirementDispatchCmd)
	requirementCmd.AddCommand(requirementCompleteCmd)
	requirementCmd.AddCommand(requirementListCmd)
	requirementCmd.AddCommand(requirementGetCmd)
	requirementCmd.AddCommand(requirementReviewCmd)
	requirementCmd.AddCommand(requirementResetCmd)
	requirementCmd.AddCommand(requirementGetStateCmd)
	requirementCmd.AddCommand(requirementUpdateStateCmd)
	requirementCmd.AddCommand(requirementTransitionCmd)
}
