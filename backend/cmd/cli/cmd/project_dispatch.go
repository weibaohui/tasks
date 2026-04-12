package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

// ==================== Dispatch Subcommands ====================

var projectDispatchCmd = &cobra.Command{
	Use:   "dispatch",
	Short: "派发配置管理",
	Long:  `管理项目的派发配置，设置需求派发时的渠道和会话`,
}

var dispatchGetCmd = &cobra.Command{
	Use:   "get",
	Short: "获取派发配置",
	Example: `  taskmanager project dispatch get <project_id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("错误: 缺少 project_id 参数")
			cmd.Usage()
			return
		}
		projectID := args[0]

		ctx := context.Background()
		c := client.New()

		project, err := c.GetProject(ctx, projectID)
		if err != nil {
			fmt.Printf("查找项目失败: %v\n", err)
			return
		}

		fmt.Println("\n派发配置:")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("%-20s %s\n", "项目ID", project.ID)
		fmt.Printf("%-20s %s\n", "项目名称", project.Name)
		fmt.Printf("%-20s %s\n", "派发渠道编码", project.DispatchChannelCode)
		fmt.Printf("%-20s %s\n", "派发会话密钥", project.DispatchSessionKey)
		fmt.Println()
	},
}

var dispatchSetCmd = &cobra.Command{
	Use:   "set",
	Short: "设置派发配置",
	Example: `  taskmanager project dispatch set <project_id> --channel-code <code> --session-key <key>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("错误: 缺少 project_id 参数")
			cmd.Usage()
			return
		}
		projectID := args[0]

		channelCode, _ := cmd.Flags().GetString("channel-code")
		sessionKey, _ := cmd.Flags().GetString("session-key")

		if channelCode == "" && sessionKey == "" {
			fmt.Println("错误: 至少需要提供 --channel-code 或 --session-key 参数")
			cmd.Usage()
			return
		}

		ctx := context.Background()
		c := client.New()

		// 先获取现有配置
		existing, err := c.GetProject(ctx, projectID)
		if err != nil {
			fmt.Printf("查找项目失败: %v\n", err)
			return
		}

		req := client.UpdateProjectRequest{
			ID: projectID,
		}

		if channelCode != "" {
			req.DispatchChannelCode = &channelCode
		} else if existing.DispatchChannelCode != "" {
			req.DispatchChannelCode = &existing.DispatchChannelCode
		}

		if sessionKey != "" {
			req.DispatchSessionKey = &sessionKey
		} else if existing.DispatchSessionKey != "" {
			req.DispatchSessionKey = &existing.DispatchSessionKey
		}

		project, err := c.UpdateProject(ctx, req)
		if err != nil {
			fmt.Printf("保存派发配置失败: %v\n", err)
			return
		}

		fmt.Printf("派发配置已更新!\n")
		fmt.Printf("  项目: %s\n", project.Name)
		fmt.Printf("  渠道编码: %s\n", project.DispatchChannelCode)
		fmt.Printf("  会话密钥: %s\n", project.DispatchSessionKey)
	},
}

var dispatchClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "清除派发配置",
	Example: `  taskmanager project dispatch clear <project_id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("错误: 缺少 project_id 参数")
			cmd.Usage()
			return
		}
		projectID := args[0]

		ctx := context.Background()
		c := client.New()

		emptyChannel := ""
		emptySession := ""

		req := client.UpdateProjectRequest{
			ID:                  projectID,
			DispatchChannelCode: &emptyChannel,
			DispatchSessionKey:  &emptySession,
		}

		project, err := c.UpdateProject(ctx, req)
		if err != nil {
			fmt.Printf("清除派发配置失败: %v\n", err)
			return
		}

		fmt.Printf("派发配置已清除，项目: %s\n", project.Name)
	},
}
