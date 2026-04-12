package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var requirementTypeListCmd = &cobra.Command{
	Use:   "list",
	Short: "查询项目下的需求类型列表",
	Example: `  taskmanager requirement-type list --project-id <id>`,
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project-id")

		if projectID == "" {
			fmt.Print(`{"error":"--project-id is required"}`)
			return
		}

		ctx := context.Background()
		c := client.New()

		types, err := c.ListRequirementTypes(ctx, projectID)
		if err != nil {
			printJSONError("list requirement types failed: %v", err)
			return
		}

		result := map[string]interface{}{
			"types": types,
			"count": len(types),
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func registerRequirementTypeListCommands() {
	requirementTypeListCmd.Flags().StringP("project-id", "p", "", "项目 ID (必填)")
}