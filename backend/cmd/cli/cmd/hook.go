package cmd

import (
	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Hook 管理",
	Long:  `查看和管理 Hook 配置`,
}

func init() {
	hookCmd.AddCommand(hookListCmd)
}