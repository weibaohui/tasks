package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteAdminCmd = &cobra.Command{
	Use:   "delete-admin",
	Short: "删除默认管理员用户（已废弃，请在 server 端执行）",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("此命令已废弃。请在 server 端执行:")
		fmt.Println("  cd backend && go run cmd/server/main.go delete-admin")
		fmt.Println("")
		fmt.Println("或者先启动 server，然后使用 Web UI 管理用户。")
	},
}
