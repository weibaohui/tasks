package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/domain"
)

var requirementResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "重置需求状态，允许重新派发",
	Example: `  taskmanager requirement reset --id <requirement_id>`,
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

		// 重置需求状态
		if err := requirement.Redispatch(); err != nil {
			fmt.Printf("重置需求失败: %v\n", err)
			return
		}

		if err := requirementRepo.Save(ctx, requirement); err != nil {
			fmt.Printf("保存需求失败: %v\n", err)
			return
		}

		fmt.Printf("需求 %s 已重置，可以重新派发\n", id)
		fmt.Printf("状态: %s / %s\n", requirement.Status(), requirement.DevState())
	},
}

func init() {
	requirementResetCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
}
