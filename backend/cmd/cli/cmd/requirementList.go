package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/domain"
)

var requirementListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出需求",
	Example: `  taskmanager requirement list
  taskmanager requirement list --project-id <id>
  taskmanager requirement list --include-heartbeat`,
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project-id")
		includeHeartbeat, _ := cmd.Flags().GetBool("include-heartbeat")

		requirementRepo, _, _, _, cleanup := getRequirementRepos()
		defer cleanup()

		ctx := context.Background()

		var requirements []*domain.Requirement
		var err error

		if projectID != "" {
			requirements, err = requirementRepo.FindByProjectID(ctx, domain.NewProjectID(projectID))
		} else {
			requirements, err = requirementRepo.FindAll(ctx)
		}

		if err != nil {
			fmt.Printf("列出需求失败: %v\n", err)
			return
		}

		// 过滤心跳需求
		if !includeHeartbeat {
			filtered := make([]*domain.Requirement, 0)
			for _, req := range requirements {
				if req.RequirementType() == domain.RequirementTypeHeartbeat {
					continue
				}
				// 兼容旧数据：标题以[心跳]开头也算心跳需求
				if strings.HasPrefix(req.Title(), "[心跳]") {
					continue
				}
				filtered = append(filtered, req)
			}
			requirements = filtered
		}

		fmt.Printf("\n需求列表 (共 %d 个):\n", len(requirements))
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("%-20s %-10s %-10s %s\n", "ID", "状态", "开发状态", "标题")
		fmt.Println("--------------------------------------------------------------------------------")
		for _, req := range requirements {
			idStr := req.ID().String()
			if len(idStr) > 16 {
				idStr = idStr[:16] + "..."
			}
			fmt.Printf("%-20s %-10s %-10s %s\n",
				idStr,
				req.Status(),
				req.DevState(),
				req.Title())
		}
		fmt.Println()
	},
}

func init() {
	requirementListCmd.Flags().StringP("project-id", "p", "", "项目 ID (可选)")
	requirementListCmd.Flags().BoolP("include-heartbeat", "", false, "包含心跳需求（默认不显示）")
}
