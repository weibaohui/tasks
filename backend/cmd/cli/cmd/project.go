package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "项目管理",
	Long:  `列出项目、管理心跳设置`,
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有项目",
	Example: `  taskmanager project list`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		c := client.New()

		projects, err := c.ListProjects(ctx)
		if err != nil {
			fmt.Printf("列出项目失败: %v\n", err)
			return
		}

		fmt.Printf("\n项目列表 (共 %d 个):\n", len(projects))
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("%-20s %s\n", "ID", "名称")
		fmt.Println("--------------------------------------------------------------------------------")
		for _, project := range projects {
			idStr := project.ID
			if len(idStr) > 16 {
				idStr = idStr[:16] + "..."
			}
			fmt.Printf("%-20s %s\n", idStr, project.Name)
		}
		fmt.Println()
	},
}

// project heartbeat subcommand
var projectHeartbeatCmd = &cobra.Command{
	Use:   "heartbeat",
	Short: "心跳设置",
	Long:  `管理心跳设置 (enable/disable/set-interval/status)`,
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
		fmt.Printf("%-20s %-10s %-10s %s\n", "项目ID", "心跳", "间隔(分钟)", "项目名称")
		fmt.Println("--------------------------------------------------------------------------------")
		for _, project := range projects {
			status := "关闭"
			if project.HeartbeatEnabled {
				status = "开启"
			}
			idStr := project.ID
			if len(idStr) > 16 {
				idStr = idStr[:16] + "..."
			}
			fmt.Printf("%-20s %-10s %-10d %s\n",
				idStr,
				status,
				project.HeartbeatIntervalMinutes,
				project.Name)
		}
		fmt.Println()
	},
}

var heartbeatEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "开启心跳",
	Example: `  taskmanager project heartbeat enable <project_id>`,
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

		interval := project.HeartbeatIntervalMinutes
		if interval < 1 {
			interval = 30
		}

		updatedProject, err := c.UpdateProjectHeartbeat(ctx, projectID, true, interval, project.HeartbeatMDContent, project.AgentCode)
		if err != nil {
			fmt.Printf("保存项目失败: %v\n", err)
			return
		}

		fmt.Printf("心跳已开启，项目: %s，间隔: %d 分钟\n", updatedProject.Name, updatedProject.HeartbeatIntervalMinutes)
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
		minutes := 0
		fmt.Sscanf(args[1], "%d", &minutes)
		if minutes < 1 {
			fmt.Println("错误: minutes 必须大于 0")
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

func init() {
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectHeartbeatCmd)

	projectHeartbeatCmd.AddCommand(heartbeatStatusCmd)
	projectHeartbeatCmd.AddCommand(heartbeatEnableCmd)
	projectHeartbeatCmd.AddCommand(heartbeatDisableCmd)
	projectHeartbeatCmd.AddCommand(heartbeatSetIntervalCmd)
}
