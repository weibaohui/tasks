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
  taskmanager requirement list --all
  taskmanager requirement list --status in_progress
  taskmanager requirement list --dev-state completed`,
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project-id")
		showAll, _ := cmd.Flags().GetBool("all")
		statusFilter, _ := cmd.Flags().GetString("status")
		devStateFilter, _ := cmd.Flags().GetString("dev-state")

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

		// 如果不是 --all，过滤掉心跳需求
		if !showAll {
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

		// 过滤状态
		if statusFilter != "" {
			filtered := make([]*domain.Requirement, 0)
			for _, req := range requirements {
				if string(req.Status()) == statusFilter {
					filtered = append(filtered, req)
				}
			}
			requirements = filtered
		}

		// 过滤开发状态
		if devStateFilter != "" {
			filtered := make([]*domain.Requirement, 0)
			for _, req := range requirements {
				if string(req.DevState()) == devStateFilter {
					filtered = append(filtered, req)
				}
			}
			requirements = filtered
		}

		// 显示类型标记
		typeTag := ""
		if showAll {
			typeTag = "全部"
		} else {
			typeTag = "普通"
		}

		fmt.Printf("\n[%s] 需求列表 (共 %d 个):\n", typeTag, len(requirements))
		fmt.Println("----------------------------------------------------------------------------------------------------------------------------")
		fmt.Printf("%-20s %-10s %-12s %-10s %s\n", "ID", "状态", "开发状态", "类型", "标题")
		fmt.Println("----------------------------------------------------------------------------------------------------------------------------")
		for _, req := range requirements {
			idStr := req.ID().String()
			if len(idStr) > 16 {
				idStr = idStr[:16] + "..."
			}
			// 类型标记
			typeTag := "普通"
			if req.RequirementType() == domain.RequirementTypeHeartbeat {
				typeTag = "心跳"
			} else if strings.HasPrefix(req.Title(), "[心跳]") {
				typeTag = "心跳"
			}
			fmt.Printf("%-20s %-10s %-12s %-10s %s\n",
				idStr,
				req.Status(),
				req.DevState(),
				typeTag,
				req.Title())
		}
		fmt.Println()
	},
}

func init() {
	requirementListCmd.Flags().StringP("project-id", "p", "", "项目 ID (可选)")
	requirementListCmd.Flags().BoolP("all", "a", false, "显示所有需求（包括心跳需求）")
	requirementListCmd.Flags().StringP("status", "s", "", "按状态过滤 (todo/in_progress/done)")
	requirementListCmd.Flags().StringP("dev-state", "d", "", "按开发状态过滤 (idle/preparing/coding/pr_opened/failed/completed)")
}
