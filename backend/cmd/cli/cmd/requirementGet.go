package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/domain"
)

var requirementGetCmd = &cobra.Command{
	Use:   "get",
	Short: "获取需求详情",
	Example: `  taskmanager requirement get --id <id>`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")

		if id == "" {
			fmt.Println("错误: --id 是必填参数")
			cmd.Usage()
			return
		}

		requirementRepo, _, _, _, cleanup := getRequirementRepos()
		defer cleanup()

		ctx := context.Background()

		requirement, err := requirementRepo.FindByID(ctx, domain.NewRequirementID(id))
		if err != nil {
			fmt.Printf("查找需求失败: %v\n", err)
			return
		}
		if requirement == nil {
			fmt.Printf("需求不存在: %s\n", id)
			return
		}

		fmt.Println("\n需求详情:")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("ID: %s\n", requirement.ID().String())
		fmt.Printf("项目ID: %s\n", requirement.ProjectID().String())
		fmt.Printf("标题: %s\n", requirement.Title())
		fmt.Printf("描述: %s\n", requirement.Description())
		fmt.Printf("验收标准: %s\n", requirement.AcceptanceCriteria())
		fmt.Printf("状态: %s / %s\n", requirement.Status(), requirement.DevState())
		fmt.Printf("类型: %s\n", requirement.RequirementType())
		fmt.Printf("工作目录: %s\n", requirement.WorkspacePath())
		fmt.Printf("PR URL: %s\n", requirement.PRURL())
		fmt.Printf("分支: %s\n", requirement.BranchName())
		if requirement.StartedAt() != nil {
			fmt.Printf("开始时间: %s\n", requirement.StartedAt().Format("2006-01-02 15:04:05"))
		}
		if requirement.CompletedAt() != nil {
			fmt.Printf("完成时间: %s\n", requirement.CompletedAt().Format("2006-01-02 15:04:05"))
		}
		if reqResult := requirement.ClaudeRuntimeResult(); reqResult != "" {
			resultPreview := reqResult
			if len(resultPreview) > 100 {
				resultPreview = resultPreview[:100] + "..."
			}
			fmt.Printf("Claude执行结果: %s\n", resultPreview)
		}
		fmt.Println()
	},
}

func init() {
	requirementGetCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
}