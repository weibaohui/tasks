package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var deleteAdminCmd = &cobra.Command{
	Use:   "delete-admin",
	Short: "删除默认管理员用户",
	Run: func(cmd *cobra.Command, args []string) {
		userRepo, _, cleanup := getUserRepos()
		defer cleanup()

		ctx := context.Background()

		// 查找用户
		existingUser, err := userRepo.FindByUsername(ctx, defaultAdminUsername)
		if err != nil {
			fmt.Printf("查找用户失败: %v\n", err)
			return
		}
		if existingUser == nil {
			fmt.Printf("管理员用户不存在: %s\n", defaultAdminUsername)
			return
		}

		// 删除用户
		if err := userRepo.Delete(ctx, existingUser.ID()); err != nil {
			fmt.Printf("删除管理员用户失败: %v\n", err)
			return
		}

		fmt.Printf("管理员用户已删除: %s\n", defaultAdminUsername)
	},
}
