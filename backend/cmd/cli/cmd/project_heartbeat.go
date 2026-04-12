package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

// ==================== Heartbeat Subcommands ====================

var projectHeartbeatCmd = &cobra.Command{
	Use:   "heartbeat",
	Short: "心跳配置管理",
	Long:  `管理项目的心跳配置，包括启用、关闭、设置间隔等`,
}

var heartbeatStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看心跳状态",
	Example: `  taskmanager project heartbeat status
  taskmanager project heartbeat status <project_id>`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		c := client.New()

		var projects []client.Project
		var err error

		if len(args) >= 1 {
			project, err := c.GetProject(ctx, args[0])
			if err != nil {
				fmt.Printf("查找项目失败: %v\n", err)
				return
			}
			projects = []client.Project{*project}
		} else {
			projects, err = c.ListProjects(ctx)
			if err != nil {
				fmt.Printf("列出项目失败: %v\n", err)
				return
			}
		}

		fmt.Println("\n项目心跳状态:")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("%-20s %-10s %-10s %-15s %s\n", "项目ID", "心跳", "间隔(分钟)", "Agent编码", "项目名称")
		fmt.Println("--------------------------------------------------------------------------------")
		for _, project := range projects {
			idStr := project.ID
			if len(idStr) > 16 {
				idStr = idStr[:16] + "..."
			}
			status := "关闭"
			if project.HeartbeatEnabled {
				status = "开启"
			}
			agentCode := project.AgentCode
			if agentCode == "" {
				agentCode = "-"
			}
			fmt.Printf("%-20s %-10s %-10d %-15s %s\n",
				idStr,
				status,
				project.HeartbeatIntervalMinutes,
				agentCode,
				project.Name)
		}
		fmt.Println()
	},
}

var heartbeatEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "开启心跳",
	Example: `  taskmanager project heartbeat enable <project_id> [--interval <minutes>]`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("错误: 缺少 project_id 参数")
			cmd.Usage()
			return
		}
		projectID := args[0]

		interval, _ := cmd.Flags().GetInt("interval")
		mdContent, _ := cmd.Flags().GetString("md-content")

		ctx := context.Background()
		c := client.New()

		project, err := c.GetProject(ctx, projectID)
		if err != nil {
			fmt.Printf("查找项目失败: %v\n", err)
			return
		}

		if interval < 1 {
			interval = project.HeartbeatIntervalMinutes
		}
		if interval < 1 {
			interval = 30
		}
		if mdContent == "" {
			mdContent = project.HeartbeatMDContent
		}
		if mdContent == "" {
			mdContent = "# 心跳报告\n\n## 任务状态\n\n## 需要关注的问题"
		}

		updatedProject, err := c.UpdateProjectHeartbeat(ctx, projectID, true, interval, mdContent, project.AgentCode)
		if err != nil {
			fmt.Printf("保存项目失败: %v\n", err)
			return
		}

		fmt.Printf("心跳已开启!\n")
		fmt.Printf("  项目: %s\n", updatedProject.Name)
		fmt.Printf("  间隔: %d 分钟\n", updatedProject.HeartbeatIntervalMinutes)
	},
}

var heartbeatDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "关闭心跳",
	Example: `  taskmanager project heartbeat disable <project_id>`,
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

		updatedProject, err := c.UpdateProjectHeartbeat(ctx, projectID, false, project.HeartbeatIntervalMinutes, project.HeartbeatMDContent, project.AgentCode)
		if err != nil {
			fmt.Printf("保存项目失败: %v\n", err)
			return
		}

		fmt.Printf("心跳已关闭，项目: %s\n", updatedProject.Name)
	},
}

var heartbeatSetIntervalCmd = &cobra.Command{
	Use:   "set-interval",
	Short: "设置心跳间隔",
	Example: `  taskmanager project heartbeat set-interval <project_id> <minutes>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			fmt.Println("错误: 缺少参数")
			fmt.Println("用法: taskmanager project heartbeat set-interval <project_id> <minutes>")
			return
		}
		projectID := args[0]
		minutes, err := strconv.Atoi(args[1])
		if err != nil || minutes < 1 {
			fmt.Println("错误: minutes 必须是大于 0 的整数")
			return
		}

		ctx := context.Background()
		c := client.New()

		project, err := c.GetProject(ctx, projectID)
		if err != nil {
			fmt.Printf("查找项目失败: %v\n", err)
			return
		}

		updatedProject, err := c.UpdateProjectHeartbeat(ctx, projectID, project.HeartbeatEnabled, minutes, project.HeartbeatMDContent, project.AgentCode)
		if err != nil {
			fmt.Printf("保存项目失败: %v\n", err)
			return
		}

		fmt.Printf("心跳间隔已设置为 %d 分钟，项目: %s\n", minutes, updatedProject.Name)
	},
}

var heartbeatSetMdCmd = &cobra.Command{
	Use:   "set-template",
	Short: "设置心跳模板内容",
	Example: `  taskmanager project heartbeat set-template <project_id> --content "<markdown>"`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("错误: 缺少 project_id 参数")
			cmd.Usage()
			return
		}
		projectID := args[0]

		mdContent, _ := cmd.Flags().GetString("content")
		if mdContent == "" {
			fmt.Println("错误: --content 参数必填")
			cmd.Usage()
			return
		}

		ctx := context.Background()
		c := client.New()

		project, err := c.GetProject(ctx, projectID)
		if err != nil {
			fmt.Printf("查找项目失败: %v\n", err)
			return
		}

		updatedProject, err := c.UpdateProjectHeartbeat(ctx, projectID, project.HeartbeatEnabled, project.HeartbeatIntervalMinutes, mdContent, project.AgentCode)
		if err != nil {
			fmt.Printf("保存项目失败: %v\n", err)
			return
		}

		fmt.Printf("心跳模板已更新，项目: %s\n", updatedProject.Name)
	},
}

var heartbeatSetAgentCmd = &cobra.Command{
	Use:   "set-agent",
	Short: "设置心跳使用的Agent",
	Example: `  taskmanager project heartbeat set-agent <project_id> --agent-code <code>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("错误: 缺少 project_id 参数")
			cmd.Usage()
			return
		}
		projectID := args[0]

		agentCode, _ := cmd.Flags().GetString("agent-code")
		if agentCode == "" {
			fmt.Println("错误: --agent-code 参数必填")
			cmd.Usage()
			return
		}

		ctx := context.Background()
		c := client.New()

		project, err := c.GetProject(ctx, projectID)
		if err != nil {
			fmt.Printf("查找项目失败: %v\n", err)
			return
		}

		updatedProject, err := c.UpdateProjectHeartbeat(ctx, projectID, project.HeartbeatEnabled, project.HeartbeatIntervalMinutes, project.HeartbeatMDContent, agentCode)
		if err != nil {
			fmt.Printf("保存项目失败: %v\n", err)
			return
		}

		fmt.Printf("心跳Agent已设置为: %s，项目: %s\n", agentCode, updatedProject.Name)
	},
}
