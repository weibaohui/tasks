package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var createAdminCmd = &cobra.Command{
	Use:   "create-admin",
	Short: "创建默认管理员用户（已废弃，请在 server 端执行）",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("此命令已废弃。请在 server 端执行:")
		fmt.Println("  cd backend && go run cmd/server/main.go create-admin")
		fmt.Println("")
		fmt.Println("或者先启动 server，然后使用 Web UI 创建用户。")
	},
}
