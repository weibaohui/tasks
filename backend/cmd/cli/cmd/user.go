package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "用户管理",
	Long:  `列出、创建、更新和删除用户`,
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出用户",
	Example: `  taskmanager user list`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		c := client.New()

		users, err := c.ListUsers(ctx)
		if err != nil {
			printJSONError("list users failed: %v", err)
			return
		}

		type UserInfo struct {
			ID         string `json:"id"`
			UserCode   string `json:"user_code"`
			Username   string `json:"username"`
			IsAdmin    bool   `json:"is_admin"`
			IsActive   bool   `json:"is_active"`
			Email      string `json:"email,omitempty"`
			Department string `json:"department,omitempty"`
		}

		items := make([]UserInfo, 0, len(users))
		for _, u := range users {
			items = append(items, UserInfo{
				ID:         u.ID,
				UserCode:   u.UserCode,
				Username:   u.Username,
				IsAdmin:    u.IsAdmin,
				IsActive:   u.IsActive,
				Email:      u.Email,
				Department: u.Department,
			})
		}

		jsonBytes, _ := json.Marshal(items)
		fmt.Print(string(jsonBytes))
	},
}

var userCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建用户",
	Example: `  taskmanager user create --user-code <code> --username <name> --password <pwd>
  taskmanager user create -c user1 -n "User One" -p secret123`,
	Run: func(cmd *cobra.Command, args []string) {
		userCode, _ := cmd.Flags().GetString("user-code")
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		isAdmin, _ := cmd.Flags().GetBool("admin")
		isActive, _ := cmd.Flags().GetBool("active")
		email, _ := cmd.Flags().GetString("email")
		department, _ := cmd.Flags().GetString("department")

		if userCode == "" || username == "" || password == "" {
			printJSONError("--user-code, --username and --password are required")
			return
		}

		ctx := context.Background()
		c := client.New()

		u, err := c.CreateUser(ctx, client.CreateUserAPIRequest{
			UserCode:   userCode,
			Username:   username,
			Password:   password,
			IsAdmin:    isAdmin,
			IsActive:   isActive,
			Email:      email,
			Department: department,
		})
		if err != nil {
			printJSONError("create user failed: %v", err)
			return
		}

		result := map[string]string{
			"id":         u.ID,
			"user_code":  u.UserCode,
			"username":   u.Username,
			"message":    "created",
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

var userUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新用户",
	Example: `  taskmanager user update <id> --username <name>
  taskmanager user update <id> --admin --active`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("user id is required")
			return
		}
		id := args[0]

		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		isAdmin, _ := cmd.Flags().GetBool("admin")
		isActive, _ := cmd.Flags().GetBool("active")
		email, _ := cmd.Flags().GetString("email")
		department, _ := cmd.Flags().GetString("department")

		req := client.UpdateUserAPIRequest{}
		if cmd.Flags().Changed("username") {
			req.Username = &username
		}
		if cmd.Flags().Changed("password") && password != "" {
			req.Password = &password
		}
		if cmd.Flags().Changed("admin") {
			req.IsAdmin = &isAdmin
		}
		if cmd.Flags().Changed("active") {
			req.IsActive = &isActive
		}
		if cmd.Flags().Changed("email") {
			req.Email = &email
		}
		if cmd.Flags().Changed("department") {
			req.Department = &department
		}

		ctx := context.Background()
		c := client.New()

		u, err := c.UpdateUser(ctx, id, req)
		if err != nil {
			printJSONError("update user failed: %v", err)
			return
		}

		result := map[string]string{
			"id":         u.ID,
			"user_code":  u.UserCode,
			"username":   u.Username,
			"message":    "updated",
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "删除用户",
	Example: `  taskmanager user delete <id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("user id is required")
			return
		}
		id := args[0]

		ctx := context.Background()
		c := client.New()

		if err := c.DeleteUser(ctx, id); err != nil {
			printJSONError("delete user failed: %v", err)
			return
		}

		result := map[string]string{"message": "deleted"}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userCreateCmd)
	userCmd.AddCommand(userUpdateCmd)
	userCmd.AddCommand(userDeleteCmd)

	userCreateCmd.Flags().StringP("user-code", "c", "", "用户代码 (必填)")
	userCreateCmd.Flags().StringP("username", "n", "", "用户名 (必填)")
	userCreateCmd.Flags().StringP("password", "p", "", "密码 (必填)")
	userCreateCmd.Flags().Bool("admin", false, "设为管理员")
	userCreateCmd.Flags().Bool("active", true, "启用用户")
	userCreateCmd.Flags().String("email", "", "邮箱")
	userCreateCmd.Flags().String("department", "", "部门")
	userUpdateCmd.Flags().String("username", "", "用户名")
	userUpdateCmd.Flags().String("password", "", "密码 (留空则不更新)")
	userUpdateCmd.Flags().Bool("admin", false, "设为管理员")
	userUpdateCmd.Flags().Bool("active", false, "启用用户")
	userUpdateCmd.Flags().String("email", "", "邮箱")
	userUpdateCmd.Flags().String("department", "", "部门")
}