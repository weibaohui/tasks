package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var requirementTypeCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建新的需求类型",
	Example: `  taskmanager requirement-type create --project-id <id> --code <code> --name <name> [--description <desc>] [--icon <icon>] [--color <color>]`,
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project-id")
		code, _ := cmd.Flags().GetString("code")
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		icon, _ := cmd.Flags().GetString("icon")
		color, _ := cmd.Flags().GetString("color")

		if projectID == "" || code == "" || name == "" {
			fmt.Print(`{"error":"--project-id, --code and --name are required"}`)
			return
		}

		ctx := context.Background()
		c := client.New()

		rt, err := c.CreateRequirementType(ctx, client.CreateRequirementTypeRequest{
			ProjectID:   projectID,
			Code:       code,
			Name:       name,
			Description: description,
			Icon:       icon,
			Color:      color,
		})
		if err != nil {
			printJSONError("create requirement type failed: %v", err)
			return
		}

		result := map[string]interface{}{
			"id":      rt.ID,
			"code":    rt.Code,
			"name":    rt.Name,
			"message": "created",
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	requirementTypeCreateCmd.Flags().StringP("project-id", "p", "", "项目 ID (必填)")
	requirementTypeCreateCmd.Flags().StringP("code", "c", "", "类型代码 (必填)")
	requirementTypeCreateCmd.Flags().StringP("name", "n", "", "类型名称 (必填)")
	requirementTypeCreateCmd.Flags().StringP("description", "d", "", "类型描述")
	requirementTypeCreateCmd.Flags().StringP("icon", "i", "", "图标")
	requirementTypeCreateCmd.Flags().StringP("color", "", "", "颜色")
}