package cmd

import (
	"context"
	"encoding/json"
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
			fmt.Print(`{"error":"--id is required"}`)
			return
		}

		requirementRepo, _, _, _, cleanup := getRequirementRepos()
		defer cleanup()

		ctx := context.Background()

		req, err := requirementRepo.FindByID(ctx, domain.NewRequirementID(id))
		if err != nil {
			printJSONError("find requirement failed: %v", err)
			return
		}
		if req == nil {
			printJSONError("requirement not found: %s", id)
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
			printJSONError("update requirement failed: %v", err)
			return
		}
		if err := requirementRepo.Save(ctx, req); err != nil {
			printJSONError("save requirement failed: %v", err)
			return
		}

		result := map[string]string{
			"id":      req.ID().String(),
			"title":   req.Title(),
			"status":  string(req.Status()),
			"message": "updated",
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	requirementUpdateCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
	requirementUpdateCmd.Flags().StringP("title", "t", "", "需求标题")
	requirementUpdateCmd.Flags().StringP("description", "d", "", "需求描述")
	requirementUpdateCmd.Flags().StringP("acceptance-criteria", "a", "", "验收标准")
}
