package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/application"
)

const (
	defaultAdminUsername = "admin"
	defaultAdminPassword = "admin123"
)

var createAdminCmd = &cobra.Command{
	Use:   "create-admin",
	Short: "创建默认管理员用户 (admin/admin123)",
	Run: func(cmd *cobra.Command, args []string) {
		userRepo, idGen, cleanup := getUserRepos()
		defer cleanup()

		ctx := context.Background()

		// 检查是否已存在
		existingUser, err := userRepo.FindByUsername(ctx, defaultAdminUsername)
		if err != nil {
			fmt.Printf("检查用户失败: %v\n", err)
			return
		}
		if existingUser != nil {
			fmt.Printf("管理员用户已存在: %s\n", defaultAdminUsername)
			return
		}

		// 使用 application 层创建用户
		userService := application.NewUserApplicationService(userRepo, idGen)
		user, err := userService.CreateUser(ctx, application.CreateUserCommand{
			Username:    defaultAdminUsername,
			DisplayName: "系统管理员",
			Email:       "admin@local.dev",
			Password:    defaultAdminPassword,
		})
		if err != nil {
			fmt.Printf("创建管理员用户失败: %v\n", err)
			return
		}

		fmt.Printf("管理员用户创建成功！\n")
		fmt.Printf("用户名: %s\n", user.Username())
		fmt.Printf("用户码: %s\n", user.UserCode().String())
		fmt.Printf("初始密码: %s (请登录后立即修改)\n", defaultAdminPassword)
	},
}
