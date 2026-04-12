package cmd

import (
	"github.com/spf13/cobra"
)

var requirementTypeCmd = &cobra.Command{
	Use:   "requirement-type",
	Short: "需求类型管理",
	Long:  `查询和创建需求类型`,
}

func registerRequirementTypeCommands() {
	requirementTypeCmd.AddCommand(requirementTypeListCmd)
	requirementTypeCmd.AddCommand(requirementTypeCreateCmd)
}