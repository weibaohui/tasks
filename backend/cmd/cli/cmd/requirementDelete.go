package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var requirementDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "删除需求",
	Example: `  taskmanager requirement delete <requirement-id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("需求 ID 不能为空")
			return
		}

		requirementID := args[0]

		ctx := context.Background()
		c := client.New()

		err := c.DeleteRequirement(ctx, requirementID)
		if err != nil {
			printJSONError("删除需求失败: %v", err)
			return
		}

		fmt.Print(`{"message":"删除成功"}`)
	},
}