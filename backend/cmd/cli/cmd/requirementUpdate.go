package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/domain"
)

var requirementUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新需求",
	Example: `  taskmanager requirement update --id <id> --title <new-title>
  taskmanager requirement update -i <id> -t <title> -d <desc>`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")
		acceptance, _ := cmd.Flags().GetString("acceptance-criteria")

		if id == "" {
			fmt.Println("错误: --id 是必填参数")
			cmd.Usage()
			return
		}

		requirementRepo, _, _, _, cleanup := getRequirementRepos()
		defer cleanup()

		ctx := context.Background()

		req, err := requirementRepo.FindByID(ctx, domain.NewRequirementID(id))
		if err != nil {
			fmt.Printf("查找需求失败: %v\n", err)
			return
		}
		if req == nil {
			fmt.Printf("需求不存在: %s\n", id)
			return
		}

		newTitle := title
		if newTitle == "" {
			newTitle = req.Title()
		}
		newDesc := description
		if newDesc == "" {
			newDesc = req.Description()
		}
		newCriteria := acceptance
		if newCriteria == "" {
			newCriteria = req.AcceptanceCriteria()
		}

		if err := req.UpdateContent(newTitle, newDesc, newCriteria, req.TempWorkspaceRoot()); err != nil {
			fmt.Printf("更新需求失败: %v\n", err)
			return
		}
		if err := requirementRepo.Save(ctx, req); err != nil {
			fmt.Printf("保存需求失败: %v\n", err)
			return
		}

		fmt.Printf("需求更新成功！\nID: %s\n标题: %s\n", req.ID().String(), req.Title())
	},
}

func init() {
	requirementUpdateCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
	requirementUpdateCmd.Flags().StringP("title", "t", "", "需求标题")
	requirementUpdateCmd.Flags().StringP("description", "d", "", "需求描述")
	requirementUpdateCmd.Flags().StringP("acceptance-criteria", "a", "", "验收标准")
}
