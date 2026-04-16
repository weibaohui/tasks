package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var requirementListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出需求",
	Example: `  taskmanager requirement list
  taskmanager requirement list --project-id <id>
  taskmanager requirement list --all
  taskmanager requirement list --todo
  taskmanager requirement list --status coding
  taskmanager requirement list --requirement-type normal
  taskmanager requirement list --sort-by created_at --order desc`,
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project-id")
		showAll, _ := cmd.Flags().GetBool("all")
		todoOnly, _ := cmd.Flags().GetBool("todo")
		statusFilter, _ := cmd.Flags().GetString("status")
		requirementType, _ := cmd.Flags().GetString("requirement-type")
		sortBy, _ := cmd.Flags().GetString("sort-by")
		order, _ := cmd.Flags().GetString("order")

		ctx := context.Background()
		c := client.New()

		// 构建查询参数
		params := make(map[string]string)
		if projectID != "" {
			params["project_id"] = projectID
		}
		// --todo 优先于 --status，如果设置了 --todo，则使用 todo 作为状态过滤
		if todoOnly {
			params["status"] = "todo"
		} else if statusFilter != "" {
			params["status"] = statusFilter
		}
		if requirementType != "" {
			params["requirement_type"] = requirementType
		}
		if sortBy != "" {
			params["sort_by"] = sortBy
		}
		if order != "" {
			params["order"] = order
		}

		requirements, err := c.ListRequirementsWithParams(ctx, params)
		if err != nil {
			printJSONError("列出需求失败: %v", err)
			return
		}

		// 过滤逻辑
		var filtered []client.Requirement
		for _, req := range *requirements {
			// 如果不是 --all，过滤掉心跳需求
			if !showAll {
				if req.RequirementType == "heartbeat" {
					continue
				}
				// 兼容旧数据：标题以[心跳]开头也算心跳需求
				if strings.HasPrefix(req.Title, "[心跳]") {
					continue
				}
			}

			// 过滤状态：--todo 优先于 --status
			if todoOnly {
				if req.Status != "todo" {
					continue
				}
			}

			filtered = append(filtered, req)
		}

		// 输出 JSON 格式（紧凑）
		type RequirementInfo struct {
			ID                 string `json:"id"`
			ProjectID          string `json:"project_id"`
			Title              string `json:"title"`
			Status             string `json:"status"`
			RequirementType    string `json:"requirement_type"`
			AssigneeAgentCode  string `json:"assignee_agent_code,omitempty"`
			ReplicaAgentCode   string `json:"replica_agent_code,omitempty"`
			WorkspacePath      string `json:"workspace_path,omitempty"`
			DispatchSessionKey string `json:"dispatch_session_key,omitempty"`
			CreatedAt          int64  `json:"created_at"`
			UpdatedAt          int64  `json:"updated_at"`
			StartedAt          *int64 `json:"started_at,omitempty"`
			CompletedAt        *int64 `json:"completed_at,omitempty"`
			LastError          string `json:"last_error,omitempty"`
		}

		items := make([]RequirementInfo, 0, len(filtered))
		for _, req := range filtered {
			items = append(items, RequirementInfo{
				ID:                 req.ID,
				ProjectID:          req.ProjectID,
				Title:              req.Title,
				Status:             req.Status,
				RequirementType:    req.RequirementType,
				AssigneeAgentCode:  req.AssigneeAgentCode,
				ReplicaAgentCode:   req.ReplicaAgentCode,
				WorkspacePath:      req.WorkspacePath,
				DispatchSessionKey: req.DispatchSessionKey,
				CreatedAt:          req.CreatedAt,
				UpdatedAt:          req.UpdatedAt,
				StartedAt:          req.StartedAt,
				CompletedAt:        req.CompletedAt,
				LastError:          req.LastError,
			})
		}

		// 紧凑 JSON 输出
		jsonBytes, err := json.Marshal(items)
		if err != nil {
			printJSONError("序列化失败: %v", err)
			return
		}
		fmt.Print(string(jsonBytes))
	},
}

func registerRequirementListCommands() {
	requirementListCmd.Flags().StringP("project-id", "p", "", "项目 ID (可选)")
	requirementListCmd.Flags().BoolP("all", "a", false, "显示所有需求（包括心跳需求）")
	requirementListCmd.Flags().BoolP("todo", "t", false, "只显示待处理的需求 (status=todo)")
	requirementListCmd.Flags().StringP("status", "s", "", "按状态过滤 (todo/preparing/coding/pr_opened/failed/completed/done)")
	requirementListCmd.Flags().String("requirement-type", "", "按需求类型过滤 (normal/heartbeat)")
	requirementListCmd.Flags().String("sort-by", "created_at", "排序字段 (created_at/updated_at/started_at)")
	requirementListCmd.Flags().String("order", "desc", "排序方向 (asc/desc)")
}
