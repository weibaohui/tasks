package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var requirementResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "重置需求状态，允许重新派发",
	Example: `  taskmanager requirement reset --id <requirement_id>`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")

		if id == "" {
			fmt.Print(`{"error":"--id is required"}`)
			return
		}

		ctx := context.Background()
		c := client.New()

		requirement, err := c.ResetRequirement(ctx, id)
		if err != nil {
			fmt.Printf(`{"error":"reset requirement failed: %v"}`, err)
			return
		}

		fmt.Printf(`{"id":"%s","status":"%s","message":"reset successfully"}`, id, requirement.Status)
	},
}

func init() {
	requirementResetCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
}
