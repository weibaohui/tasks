package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var requirementCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建新需求",
	Example: `  taskmanager requirement create --project-id <id> --title <title> --description <desc>
  taskmanager requirement create -p <id> -t <title> -d <desc> -a <acceptance>`,
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project-id")
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")
		acceptance, _ := cmd.Flags().GetString("acceptance-criteria")
		tempWorkspace, _ := cmd.Flags().GetString("temp-workspace-root")

		if projectID == "" || title == "" {
			fmt.Print(`{"error":"--project-id and --title are required"}`)
			return
		}

		ctx := context.Background()
		c := client.New()

		requirement, err := c.CreateRequirement(ctx, client.CreateRequirementRequest{
			ProjectID:          projectID,
			Title:              title,
			Description:        description,
			AcceptanceCriteria: acceptance,
			TempWorkspaceRoot:  tempWorkspace,
		})
		if err != nil {
			printJSONError("create requirement failed: %v", err)
			return
		}

		result := map[string]string{
			"id":      requirement.ID,
			"title":   requirement.Title,
			"status":  requirement.Status,
			"message": "created",
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	requirementCreateCmd.Flags().StringP("project-id", "p", "", "项目 ID (必填)")
	requirementCreateCmd.Flags().StringP("title", "t", "", "需求标题 (必填)")
	requirementCreateCmd.Flags().StringP("description", "d", "", "需求描述")
	requirementCreateCmd.Flags().StringP("acceptance-criteria", "a", "", "验收标准")
	requirementCreateCmd.Flags().StringP("temp-workspace-root", "", "", "临时工作目录根路径")
}
