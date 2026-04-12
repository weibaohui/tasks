package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var requirementUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新需求",
	Example: `  taskmanager requirement update --id <id> --title <title> --description <desc>
  taskmanager requirement update -i <id> -t <title> -d <desc> -a <acceptance>`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")
		acceptance, _ := cmd.Flags().GetString("acceptance-criteria")
		tempWorkspace, _ := cmd.Flags().GetString("temp-workspace-root")

		if id == "" {
			fmt.Print(`{"error":"--id is required"}`)
			return
		}

		ctx := context.Background()
		c := client.New()

		// 仅传递用户显式提供的字段（即使是空字符串）
		var newTitle, newDesc, newAcceptance, newTempWorkspace *string
		if cmd.Flags().Changed("title") {
			newTitle = &title
		}
		if cmd.Flags().Changed("description") {
			newDesc = &description
		}
		if cmd.Flags().Changed("acceptance-criteria") {
			newAcceptance = &acceptance
		}
		if cmd.Flags().Changed("temp-workspace-root") {
			newTempWorkspace = &tempWorkspace
		}

		requirement, err := c.UpdateRequirement(ctx, client.UpdateRequirementRequest{
			ID:                 id,
			Title:              newTitle,
			Description:        newDesc,
			AcceptanceCriteria: newAcceptance,
			TempWorkspaceRoot:  newTempWorkspace,
		})
		if err != nil {
			printJSONError("update requirement failed: %v", err)
			return
		}

		result := map[string]string{
			"id":      requirement.ID,
			"title":   requirement.Title,
			"status":  requirement.Status,
			"message": "updated",
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func registerRequirementUpdateCommands() {
	requirementUpdateCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
	requirementUpdateCmd.Flags().StringP("title", "t", "", "需求标题")
	requirementUpdateCmd.Flags().StringP("description", "d", "", "需求描述")
	requirementUpdateCmd.Flags().StringP("acceptance-criteria", "a", "", "验收标准")
	requirementUpdateCmd.Flags().StringP("temp-workspace-root", "", "", "临时工作目录根路径")
}
