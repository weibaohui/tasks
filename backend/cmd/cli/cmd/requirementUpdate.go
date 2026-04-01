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

		// 首先获取当前需求详情，用于填充未提供的字段
		currentReq, err := c.GetRequirement(ctx, id)
		if err != nil {
			printJSONError("find requirement failed: %v", err)
			return
		}

		// 使用新值或保留原值
		newTitle := title
		if newTitle == "" {
			newTitle = currentReq.Title
		}
		newDesc := description
		if newDesc == "" {
			newDesc = currentReq.Description
		}
		newAcceptance := acceptance
		if newAcceptance == "" {
			newAcceptance = currentReq.AcceptanceCriteria
		}
		newTempWorkspace := tempWorkspace
		if newTempWorkspace == "" {
			newTempWorkspace = currentReq.TempWorkspaceRoot
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

func init() {
	requirementUpdateCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
	requirementUpdateCmd.Flags().StringP("title", "t", "", "需求标题")
	requirementUpdateCmd.Flags().StringP("description", "d", "", "需求描述")
	requirementUpdateCmd.Flags().StringP("acceptance-criteria", "a", "", "验收标准")
	requirementUpdateCmd.Flags().StringP("temp-workspace-root", "", "", "临时工作目录根路径")
}
