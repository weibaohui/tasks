package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var requirementGetCmd = &cobra.Command{
	Use:   "get",
	Short: "获取需求详情",
	Example: `  taskmanager requirement get --id <id>`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")

		if id == "" {
			fmt.Print(`{"error":"--id is required"}`)
			return
		}

		ctx := context.Background()
		c := client.New()

		requirement, err := c.GetRequirement(ctx, id)
		if err != nil {
			printJSONError("find requirement failed: %v", err)
			return
		}

		printRequirementJSON(requirement)
	},
}

func printRequirementJSON(r *client.Requirement) {
	type RequirementDetail struct {
		ID                     string `json:"id"`
		ProjectID              string `json:"project_id"`
		Title                  string `json:"title"`
		Description            string `json:"description"`
		AcceptanceCriteria     string `json:"acceptance_criteria"`
		Status                 string `json:"status"`
		RequirementType        string `json:"requirement_type"`
		AssigneeAgentCode      string `json:"assignee_agent_code,omitempty"`
		ReplicaAgentCode       string `json:"replica_agent_code,omitempty"`
		WorkspacePath          string `json:"workspace_path,omitempty"`
		DispatchSessionKey     string `json:"dispatch_session_key,omitempty"`
		LastError              string `json:"last_error,omitempty"`
		CreatedAt              int64  `json:"created_at"`
		UpdatedAt              int64  `json:"updated_at"`
		StartedAt              *int64 `json:"started_at,omitempty"`
		CompletedAt            *int64 `json:"completed_at,omitempty"`
		ClaudeRuntimeStatus    string `json:"claude_runtime_status,omitempty"`
		ClaudeRuntimePrompt    string `json:"claude_runtime_prompt,omitempty"`
		ClaudeRuntimeResult    string `json:"claude_runtime_result,omitempty"`
		ClaudeRuntimeError     string `json:"claude_runtime_error,omitempty"`
		ClaudeRuntimeStartedAt *int64 `json:"claude_runtime_started_at,omitempty"`
		ClaudeRuntimeEndedAt   *int64 `json:"claude_runtime_ended_at,omitempty"`
	}

	detail := RequirementDetail{
		ID:                 r.ID,
		ProjectID:          r.ProjectID,
		Title:              r.Title,
		Description:        r.Description,
		AcceptanceCriteria: r.AcceptanceCriteria,
		Status:             r.Status,
		RequirementType:    r.RequirementType,
		AssigneeAgentCode:  r.AssigneeAgentCode,
		ReplicaAgentCode:   r.ReplicaAgentCode,
		WorkspacePath:      r.WorkspacePath,
		DispatchSessionKey: r.DispatchSessionKey,
		LastError:          r.LastError,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
		StartedAt:          r.StartedAt,
		CompletedAt:        r.CompletedAt,
	}

	// 填充 ClaudeRuntime 字段
	if r.ClaudeRuntime != nil {
		if v, ok := r.ClaudeRuntime["status"].(string); ok {
			detail.ClaudeRuntimeStatus = v
		}
		if v, ok := r.ClaudeRuntime["prompt"].(string); ok {
			detail.ClaudeRuntimePrompt = v
		}
		if v, ok := r.ClaudeRuntime["result"].(string); ok {
			detail.ClaudeRuntimeResult = v
		}
		if v, ok := r.ClaudeRuntime["last_error"].(string); ok {
			detail.ClaudeRuntimeError = v
		}
		if v, ok := r.ClaudeRuntime["started_at"].(float64); ok {
			t := int64(v)
			detail.ClaudeRuntimeStartedAt = &t
		}
		if v, ok := r.ClaudeRuntime["ended_at"].(float64); ok {
			t := int64(v)
			detail.ClaudeRuntimeEndedAt = &t
		}
	}

	jsonBytes, _ := json.Marshal(detail)
	fmt.Print(string(jsonBytes))
}

func init() {
	requirementGetCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
}
