package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

var requirementCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建新需求",
	Example: `  taskmanager requirement create --project-id <id> --title <title> --description <desc>
  taskmanager requirement create -p <id> -t <title> -d <desc> -a <acceptance>`,
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project-id")
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")
		acceptance, _ := cmd.Flags().GetString("acceptance-criteria")
		tempWorkspace, _ := cmd.Flags().GetString("temp-workspace-root")

		if projectID == "" || title == "" {
			fmt.Println("错误: --project-id 和 --title 是必填参数")
			cmd.Usage()
			return
		}

		_, _, appService, _, cleanup := getRequirementRepos()
		defer cleanup()

		ctx := context.Background()

		requirement, err := appService.CreateRequirement(ctx, application.CreateRequirementCommand{
			ProjectID:          domain.NewProjectID(projectID),
			Title:              title,
			Description:        description,
			AcceptanceCriteria: acceptance,
			TempWorkspaceRoot:  tempWorkspace,
		})
		if err != nil {
			fmt.Printf("创建需求失败: %v\n", err)
			return
		}

		fmt.Printf("需求创建成功！\nID: %s\n标题: %s\n", requirement.ID().String(), requirement.Title())
	},
}

func init() {
	requirementCreateCmd.Flags().StringP("project-id", "p", "", "项目 ID (必填)")
	requirementCreateCmd.Flags().StringP("title", "t", "", "需求标题 (必填)")
	requirementCreateCmd.Flags().StringP("description", "d", "", "需求描述")
	requirementCreateCmd.Flags().StringP("acceptance-criteria", "a", "", "验收标准")
	requirementCreateCmd.Flags().StringP("temp-workspace-root", "", "", "临时工作目录根路径")
}
