package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "技能管理",
	Long:  `列出和查看技能`,
}

var skillListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出技能",
	Example: `  taskmanager skill list`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		c := client.New()

		skills, err := c.ListSkills(ctx)
		if err != nil {
			printJSONError("list skills failed: %v", err)
			return
		}

		type SkillInfo struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Category    string `json:"category"`
		}

		items := make([]SkillInfo, 0, len(skills))
		for _, s := range skills {
			items = append(items, SkillInfo{
				Name:        s.Name,
				Description: s.Description,
				Category:    s.Category,
			})
		}

		jsonBytes, _ := json.Marshal(items)
		fmt.Print(string(jsonBytes))
	},
}

var skillGetCmd = &cobra.Command{
	Use:   "get",
	Short: "获取技能详情",
	Example: `  taskmanager skill get <name>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("skill name is required")
			return
		}
		name := args[0]

		ctx := context.Background()
		c := client.New()

		skill, err := c.GetSkill(ctx, name)
		if err != nil {
			printJSONError("get skill failed: %v", err)
			return
		}

		jsonBytes, _ := json.Marshal(skill)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	skillCmd.AddCommand(skillListCmd)
	skillCmd.AddCommand(skillGetCmd)
}