package cmd

import (
	"context"
	"encoding/json"
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
  taskmanager requirement list --todo
  taskmanager requirement list --status coding`,
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project-id")
		showAll, _ := cmd.Flags().GetBool("all")
		todoOnly, _ := cmd.Flags().GetBool("todo")
		statusFilter, _ := cmd.Flags().GetString("status")

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
			fmt.Printf(`{"error":"列出需求失败: %v"}`, err)
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

		// 过滤状态：--todo 优先于 --status
		if todoOnly {
			filtered := make([]*domain.Requirement, 0)
			for _, req := range requirements {
				if req.Status() == domain.RequirementStatusTodo {
					filtered = append(filtered, req)
				}
			}
			requirements = filtered
		} else if statusFilter != "" {
			filtered := make([]*domain.Requirement, 0)
			for _, req := range requirements {
				if string(req.Status()) == statusFilter {
					filtered = append(filtered, req)
				}
			}
			requirements = filtered
		}

		// 输出 JSON 格式（紧凑）
		type RequirementInfo struct {
			ID                  string `json:"id"`
			ProjectID           string `json:"project_id"`
			Title               string `json:"title"`
			Status              string `json:"status"`
			RequirementType     string `json:"requirement_type"`
			AssigneeAgentCode   string `json:"assignee_agent_code,omitempty"`
			ReplicaAgentCode    string `json:"replica_agent_code,omitempty"`
			WorkspacePath       string `json:"workspace_path,omitempty"`
			DispatchSessionKey  string `json:"dispatch_session_key,omitempty"`
			CreatedAt           int64  `json:"created_at"`
			UpdatedAt           int64  `json:"updated_at"`
			StartedAt           *int64 `json:"started_at,omitempty"`
			CompletedAt         *int64 `json:"completed_at,omitempty"`
			LastError           string `json:"last_error,omitempty"`
		}

		items := make([]RequirementInfo, 0, len(requirements))
		for _, req := range requirements {
			var startedAt, completedAt *int64
			if req.StartedAt() != nil {
				t := req.StartedAt().Unix()
				startedAt = &t
			}
			if req.CompletedAt() != nil {
				t := req.CompletedAt().Unix()
				completedAt = &t
			}
			items = append(items, RequirementInfo{
				ID:                 req.ID().String(),
				ProjectID:          req.ProjectID().String(),
				Title:              req.Title(),
				Status:             string(req.Status()),
				RequirementType:    string(req.RequirementType()),
				AssigneeAgentCode:  req.AssigneeAgentCode(),
				ReplicaAgentCode:   req.ReplicaAgentCode(),
				WorkspacePath:      req.WorkspacePath(),
				DispatchSessionKey: req.DispatchSessionKey(),
				CreatedAt:          req.CreatedAt().Unix(),
				UpdatedAt:          req.UpdatedAt().Unix(),
				StartedAt:          startedAt,
				CompletedAt:        completedAt,
				LastError:          req.LastError(),
			})
		}

		// 紧凑 JSON 输出
		jsonBytes, _ := json.Marshal(items)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	requirementListCmd.Flags().StringP("project-id", "p", "", "项目 ID (可选)")
	requirementListCmd.Flags().BoolP("all", "a", false, "显示所有需求（包括心跳需求）")
	requirementListCmd.Flags().BoolP("todo", "t", false, "只显示待处理的需求 (status=todo)")
	requirementListCmd.Flags().StringP("status", "s", "", "按状态过滤 (todo/preparing/coding/pr_opened/failed/completed/done)")
}
