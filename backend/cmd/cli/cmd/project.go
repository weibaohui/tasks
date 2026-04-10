package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "项目管理",
	Long:  `项目管理和配置，包括创建、更新、删除和各项配置`,
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
		fmt.Printf("%-20s %-10s %-15s %s\n", "ID", "心跳", "间隔(分钟)", "项目名称")
		fmt.Println("--------------------------------------------------------------------------------")
		for _, project := range projects {
			idStr := project.ID
			if len(idStr) > 16 {
				idStr = idStr[:16] + "..."
			}
			heartbeatStatus := "关闭"
			if project.HeartbeatEnabled {
				heartbeatStatus = "开启"
			}
			fmt.Printf("%-20s %-10s %-15d %s\n",
				idStr,
				heartbeatStatus,
				project.HeartbeatIntervalMinutes,
				project.Name)
		}
		fmt.Println()
	},
}

var projectGetCmd = &cobra.Command{
	Use:   "get",
	Short: "获取项目详情",
	Example: `  taskmanager project get <project_id>`,
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
			fmt.Printf("获取项目详情失败: %v\n", err)
			return
		}

		fmt.Println("\n项目详情:")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("%-20s %s\n", "ID", project.ID)
		fmt.Printf("%-20s %s\n", "名称", project.Name)
		fmt.Printf("%-20s %s\n", "Git仓库", project.GitRepoURL)
		fmt.Printf("%-20s %s\n", "默认分支", project.DefaultBranch)
		if len(project.InitSteps) > 0 {
			fmt.Printf("%-20s %s\n", "初始化步骤", strings.Join(project.InitSteps, "\n"+strings.Repeat(" ", 20)))
		}
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("%-20s %s\n", "心跳", boolToStr(project.HeartbeatEnabled))
		fmt.Printf("%-20s %d 分钟\n", "心跳间隔", project.HeartbeatIntervalMinutes)
		fmt.Printf("%-20s %s\n", "Agent编码", project.AgentCode)
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("%-20s %s\n", "派发渠道", project.DispatchChannelCode)
		fmt.Printf("%-20s %s\n", "派发会话密钥", project.DispatchSessionKey)
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("%-20s %d\n", "创建时间", project.CreatedAt)
		fmt.Printf("%-20s %d\n", "更新时间", project.UpdatedAt)
		fmt.Println()
	},
}

var projectCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建新项目",
	Example: `  taskmanager project create --name "我的项目" --git-repo-url "https://github.com/user/repo"`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		gitRepoURL, _ := cmd.Flags().GetString("git-repo-url")
		defaultBranch, _ := cmd.Flags().GetString("default-branch")
		initStepsStr, _ := cmd.Flags().GetString("init-steps")

		if name == "" {
			fmt.Println("错误: --name 参数必填")
			cmd.Usage()
			return
		}

		var initSteps []string
		if initStepsStr != "" {
			for _, s := range strings.Split(initStepsStr, "\n") {
				s = strings.TrimSpace(s)
				if s != "" {
					initSteps = append(initSteps, s)
				}
			}
		}

		ctx := context.Background()
		c := client.New()

		project, err := c.CreateProject(ctx, client.CreateProjectRequest{
			Name:          name,
			GitRepoURL:    gitRepoURL,
			DefaultBranch: defaultBranch,
			InitSteps:     initSteps,
		})
		if err != nil {
			fmt.Printf("创建项目失败: %v\n", err)
			return
		}

		fmt.Printf("项目创建成功!\n")
		fmt.Printf("  ID: %s\n", project.ID)
		fmt.Printf("  名称: %s\n", project.Name)
		fmt.Printf("  Git仓库: %s\n", project.GitRepoURL)
		fmt.Printf("  默认分支: %s\n", project.DefaultBranch)
	},
}

var projectUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新项目信息",
	Example: `  taskmanager project update <project_id> --name "新名称" --git-repo-url "https://new-repo-url"`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("错误: 缺少 project_id 参数")
			cmd.Usage()
			return
		}
		projectID := args[0]

		name, _ := cmd.Flags().GetString("name")
		gitRepoURL, _ := cmd.Flags().GetString("git-repo-url")
		defaultBranch, _ := cmd.Flags().GetString("default-branch")
		initStepsStr, _ := cmd.Flags().GetString("init-steps")

		var initSteps []string
		if initStepsStr != "" {
			for _, s := range strings.Split(initStepsStr, "\n") {
				s = strings.TrimSpace(s)
				if s != "" {
					initSteps = append(initSteps, s)
				}
			}
		}

		ctx := context.Background()
		c := client.New()

		// 先获取现有项目信息
		existing, err := c.GetProject(ctx, projectID)
		if err != nil {
			fmt.Printf("获取项目失败: %v\n", err)
			return
		}

		// 构建更新请求
		req := client.UpdateProjectRequest{
			ID: projectID,
		}

		if name != "" {
			req.Name = name
		} else {
			req.Name = existing.Name
		}

		if gitRepoURL != "" {
			req.GitRepoURL = gitRepoURL
		} else {
			req.GitRepoURL = existing.GitRepoURL
		}

		if defaultBranch != "" {
			req.DefaultBranch = defaultBranch
		} else {
			req.DefaultBranch = existing.DefaultBranch
		}

		if len(initSteps) > 0 {
			req.InitSteps = initSteps
		} else {
			req.InitSteps = existing.InitSteps
		}

		project, err := c.UpdateProject(ctx, req)
		if err != nil {
			fmt.Printf("更新项目失败: %v\n", err)
			return
		}

		fmt.Printf("项目更新成功!\n")
		fmt.Printf("  ID: %s\n", project.ID)
		fmt.Printf("  名称: %s\n", project.Name)
		fmt.Printf("  Git仓库: %s\n", project.GitRepoURL)
		fmt.Printf("  默认分支: %s\n", project.DefaultBranch)
	},
}

var projectDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "删除项目",
	Example: `  taskmanager project delete <project_id> [--force]`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("错误: 缺少 project_id 参数")
			cmd.Usage()
			return
		}
		projectID := args[0]

		force, _ := cmd.Flags().GetBool("force")

		// 检查环境变量
		if !force {
			confirmEnv := os.Getenv("CONFIRM_DELETE")
			if confirmEnv == "y" || confirmEnv == "yes" || confirmEnv == "Y" || confirmEnv == "YES" {
				force = true
			} else if confirmEnv == "n" || confirmEnv == "no" || confirmEnv == "N" || confirmEnv == "NO" {
				fmt.Println("取消删除")
				return
			}
		}

		if !force {
			fmt.Printf("确认删除项目 %s? (y/N): ", projectID)
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("取消删除")
				return
			}
		}

		ctx := context.Background()
		c := client.New()

		err := c.DeleteProject(ctx, projectID)
		if err != nil {
			fmt.Printf("删除项目失败: %v\n", err)
			return
		}

		fmt.Printf("项目 %s 已删除\n", projectID)
	},
}

// ==================== Heartbeat 子命令 ====================

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

// ==================== Dispatch 子命令 ====================

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
			ID:                 projectID,
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

// ==================== 辅助函数 ====================

func boolToStr(b bool) string {
	if b {
		return "开启"
	}
	return "关闭"
}

func init() {
	rootCmd.AddCommand(projectCmd)

	// 注册子命令
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectGetCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectUpdateCmd)
	projectCmd.AddCommand(projectDeleteCmd)

	// 心跳子命令
	projectCmd.AddCommand(projectHeartbeatCmd)
	projectHeartbeatCmd.AddCommand(heartbeatStatusCmd)
	projectHeartbeatCmd.AddCommand(heartbeatEnableCmd)
	projectHeartbeatCmd.AddCommand(heartbeatDisableCmd)
	projectHeartbeatCmd.AddCommand(heartbeatSetIntervalCmd)
	projectHeartbeatCmd.AddCommand(heartbeatSetMdCmd)
	projectHeartbeatCmd.AddCommand(heartbeatSetAgentCmd)

	// 派发子命令
	projectCmd.AddCommand(projectDispatchCmd)
	projectDispatchCmd.AddCommand(dispatchGetCmd)
	projectDispatchCmd.AddCommand(dispatchSetCmd)
	projectDispatchCmd.AddCommand(dispatchClearCmd)

	// create 命令的 flags
	projectCreateCmd.Flags().StringP("name", "n", "", "项目名称 (必填)")
	projectCreateCmd.Flags().StringP("git-repo-url", "r", "", "Git仓库地址")
	projectCreateCmd.Flags().StringP("default-branch", "b", "main", "默认分支")
	projectCreateCmd.Flags().StringP("init-steps", "s", "", "初始化步骤 (每行一个)")

	// update 命令的 flags
	projectUpdateCmd.Flags().StringP("name", "n", "", "项目名称")
	projectUpdateCmd.Flags().StringP("git-repo-url", "r", "", "Git仓库地址")
	projectUpdateCmd.Flags().StringP("default-branch", "b", "", "默认分支")
	projectUpdateCmd.Flags().StringP("init-steps", "s", "", "初始化步骤 (每行一个)")

	// heartbeat enable 命令的 flags
	heartbeatEnableCmd.Flags().IntP("interval", "i", 0, "心跳间隔 (分钟)")
	heartbeatEnableCmd.Flags().StringP("md-content", "m", "", "心跳模板内容 (Markdown)")

	// heartbeat set-template 命令的 flags
	heartbeatSetMdCmd.Flags().StringP("content", "c", "", "心跳模板内容 (Markdown, 必填)")

	// heartbeat set-agent 命令的 flags
	heartbeatSetAgentCmd.Flags().StringP("agent-code", "a", "", "Agent编码 (必填)")

	// dispatch set 命令的 flags
	dispatchSetCmd.Flags().StringP("channel-code", "c", "", "渠道编码")
	dispatchSetCmd.Flags().StringP("session-key", "k", "", "会话密钥")

	// delete 命令的 flags
	projectDeleteCmd.Flags().BoolP("force", "f", false, "跳过确认直接删除")
}
