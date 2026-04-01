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

// AddCommands 注册所有子命令
func AddCommands() {
	rootCmd.AddCommand(createAdminCmd)
	rootCmd.AddCommand(deleteAdminCmd)
	rootCmd.AddCommand(requirementCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(projectCmd)
	rootCmd.AddCommand(configCmd)
}

// RegisterFlagErrorFunc 自定义错误处理
func RegisterFlagErrorFunc() {
	cobra.EnableCommandSorting = false
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return err
	})
}
