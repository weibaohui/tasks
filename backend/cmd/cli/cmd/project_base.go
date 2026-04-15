package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

// ==================== Base Project Commands ====================

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
		fmt.Printf("%-20s %s\n", "ID", "项目名称")
		fmt.Println("--------------------------------------------------------------------------------")
		for _, project := range projects {
			idStr := project.ID
			if len(idStr) > 16 {
				idStr = idStr[:16] + "..."
			}
			fmt.Printf("%-20s %s\n",
				idStr,
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
		fmt.Printf("%-20s %s\n", "派发渠道", project.DispatchChannelCode)
		fmt.Printf("%-20s %s\n", "派发SessionKey", project.DispatchSessionKey)

		if len(project.InitSteps) > 0 {
			fmt.Println("--------------------------------------------------------------------------------")
			fmt.Println("初始化步骤:")
			for i, step := range project.InitSteps {
				fmt.Printf("  %d. %s\n", i+1, step)
			}
		}
		fmt.Println()
	},
}

var projectCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建新项目",
	Example: `  taskmanager project create --name <name> --git-repo-url <url> [--default-branch <branch>]`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		gitRepoURL, _ := cmd.Flags().GetString("git-repo-url")
		defaultBranch, _ := cmd.Flags().GetString("default-branch")
		initStepsText, _ := cmd.Flags().GetString("init-steps")

		if name == "" {
			fmt.Println("错误: --name 参数必填")
			cmd.Usage()
			return
		}
		if gitRepoURL == "" {
			fmt.Println("错误: --git-repo-url 参数必填")
			cmd.Usage()
			return
		}

		ctx := context.Background()
		c := client.New()

		initSteps := strings.Split(initStepsText, "\n")
		var cleanSteps []string
		for _, step := range initSteps {
			step = strings.TrimSpace(step)
			if step != "" {
				cleanSteps = append(cleanSteps, step)
			}
		}

		req := client.CreateProjectRequest{
			Name:          name,
			GitRepoURL:    gitRepoURL,
			DefaultBranch: defaultBranch,
			InitSteps:     cleanSteps,
		}

		project, err := c.CreateProject(ctx, req)
		if err != nil {
			fmt.Printf("创建项目失败: %v\n", err)
			return
		}

		fmt.Printf("\n项目创建成功!\n")
		fmt.Printf("  ID: %s\n", project.ID)
		fmt.Printf("  名称: %s\n", project.Name)
		fmt.Printf("  仓库: %s\n", project.GitRepoURL)
		fmt.Println()
	},
}

var projectUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新项目信息",
	Example: `  taskmanager project update <project_id> [--name <name>] [--git-repo-url <url>]`,
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
		initStepsText, _ := cmd.Flags().GetString("init-steps")

		ctx := context.Background()
		c := client.New()

		// 先获取现有项目信息
		existing, err := c.GetProject(ctx, projectID)
		if err != nil {
			fmt.Printf("获取项目信息失败: %v\n", err)
			return
		}

		// 如果未提供参数，则使用现有值
		if name == "" {
			name = existing.Name
		}
		if gitRepoURL == "" {
			gitRepoURL = existing.GitRepoURL
		}
		if defaultBranch == "" {
			defaultBranch = existing.DefaultBranch
		}

		initSteps := strings.Split(initStepsText, "\n")
		var cleanSteps []string
		for _, step := range initSteps {
			step = strings.TrimSpace(step)
			if step != "" {
				cleanSteps = append(cleanSteps, step)
			}
		}
		if len(cleanSteps) == 0 {
			cleanSteps = existing.InitSteps
		}

		req := client.UpdateProjectRequest{
			ID:            projectID,
			Name:          name,
			GitRepoURL:    gitRepoURL,
			DefaultBranch: defaultBranch,
			InitSteps:     cleanSteps,
		}

		project, err := c.UpdateProject(ctx, req)
		if err != nil {
			fmt.Printf("更新项目失败: %v\n", err)
			return
		}

		fmt.Printf("\n项目更新成功!\n")
		fmt.Printf("  ID: %s\n", project.ID)
		fmt.Printf("  名称: %s\n", project.Name)
		fmt.Println()
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

		if !force {
			fmt.Printf("确认删除项目 %s 吗？此操作不可恢复。\n", projectID)
			fmt.Print("输入 'yes' 确认: ")
			var confirm string
			if _, err := fmt.Scan(&confirm); err != nil || confirm != "yes" {
				fmt.Println("已取消删除")
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

// Helper function for bool to string
func boolToStr(b bool) string {
	if b {
		return "开启"
	}
	return "关闭"
}
