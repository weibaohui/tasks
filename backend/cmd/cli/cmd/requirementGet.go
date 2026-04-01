package cmd

import (
	"context"
	"encoding/json"
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
			fmt.Print(`{"error":"--id is required"}`)
			return
		}

		requirementRepo, _, _, _, cleanup := getRequirementRepos()
		defer cleanup()

		ctx := context.Background()

		requirement, err := requirementRepo.FindByID(ctx, domain.NewRequirementID(id))
		if err != nil {
			printJSONError("find requirement failed: %v", err)
			return
		}
		if requirement == nil {
			printJSONError("requirement not found: %s", id)
			return
		}

		printRequirementJSON(requirement)
	},
}

func printRequirementJSON(r *domain.Requirement) {
	type RequirementDetail struct {
		ID                    string  `json:"id"`
		ProjectID             string  `json:"project_id"`
		Title                 string  `json:"title"`
		Description           string  `json:"description"`
		AcceptanceCriteria    string  `json:"acceptance_criteria"`
		Status                string  `json:"status"`
		RequirementType       string  `json:"requirement_type"`
		AssigneeAgentCode     string  `json:"assignee_agent_code,omitempty"`
		ReplicaAgentCode      string  `json:"replica_agent_code,omitempty"`
		WorkspacePath         string  `json:"workspace_path,omitempty"`
		DispatchSessionKey    string  `json:"dispatch_session_key,omitempty"`
		LastError             string  `json:"last_error,omitempty"`
		CreatedAt             int64   `json:"created_at"`
		UpdatedAt             int64   `json:"updated_at"`
		StartedAt            *int64  `json:"started_at,omitempty"`
		CompletedAt          *int64  `json:"completed_at,omitempty"`
		ClaudeRuntimeStatus   string  `json:"claude_runtime_status,omitempty"`
		ClaudeRuntimePrompt   string  `json:"claude_runtime_prompt,omitempty"`
		ClaudeRuntimeResult   string  `json:"claude_runtime_result,omitempty"`
		ClaudeRuntimeError    string  `json:"claude_runtime_error,omitempty"`
		ClaudeRuntimeStartedAt *int64 `json:"claude_runtime_started_at,omitempty"`
		ClaudeRuntimeEndedAt   *int64 `json:"claude_runtime_ended_at,omitempty"`
	}

	var startedAt, completedAt, rtStartedAt, rtEndedAt *int64
	if r.StartedAt() != nil {
		t := r.StartedAt().Unix()
		startedAt = &t
	}
	if r.CompletedAt() != nil {
		t := r.CompletedAt().Unix()
		completedAt = &t
	}
	if r.ClaudeRuntimeStartedAt() != nil {
		t := r.ClaudeRuntimeStartedAt().Unix()
		rtStartedAt = &t
	}
	if r.ClaudeRuntimeEndedAt() != nil {
		t := r.ClaudeRuntimeEndedAt().Unix()
		rtEndedAt = &t
	}

	detail := RequirementDetail{
		ID:                   r.ID().String(),
		ProjectID:            r.ProjectID().String(),
		Title:                r.Title(),
		Description:          r.Description(),
		AcceptanceCriteria:   r.AcceptanceCriteria(),
		Status:               string(r.Status()),
		RequirementType:      string(r.RequirementType()),
		AssigneeAgentCode:    r.AssigneeAgentCode(),
		ReplicaAgentCode:     r.ReplicaAgentCode(),
		WorkspacePath:        r.WorkspacePath(),
		DispatchSessionKey:   r.DispatchSessionKey(),
		LastError:            r.LastError(),
		CreatedAt:            r.CreatedAt().Unix(),
		UpdatedAt:            r.UpdatedAt().Unix(),
		StartedAt:            startedAt,
		CompletedAt:          completedAt,
		ClaudeRuntimeStatus:  r.ClaudeRuntimeStatus(),
		ClaudeRuntimePrompt:  r.ClaudeRuntimePrompt(),
		ClaudeRuntimeResult:  r.ClaudeRuntimeResult(),
		ClaudeRuntimeError:   r.ClaudeRuntimeError(),
		ClaudeRuntimeStartedAt: rtStartedAt,
		ClaudeRuntimeEndedAt:   rtEndedAt,
	}

	jsonBytes, _ := json.Marshal(detail)
	fmt.Print(string(jsonBytes))
}

func init() {
	requirementGetCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
}