package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/domain"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Agent 管理",
	Long:  `列出所有 Agent`,
}

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有 Agent",
	Example: `  taskmanager agent list`,
	Run: func(cmd *cobra.Command, args []string) {
		agentRepo, cleanup := getAgentRepo()
		defer cleanup()

		ctx := context.Background()

		agents, err := agentRepo.FindAll(ctx)
		if err != nil {
			fmt.Printf("列出 Agent 失败: %v\n", err)
			return
		}

		fmt.Printf("\nAgent 列表 (共 %d 个):\n", len(agents))
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("%-20s %-15s %-15s %s\n", "ID", "类型", "状态", "名称")
		fmt.Println("--------------------------------------------------------------------------------")
		for _, agent := range agents {
			agentType := "Unknown"
			if agent.AgentType() == domain.AgentTypeCoding {
				agentType = "CodingAgent"
			} else if agent.AgentType() == domain.AgentTypeBareLLM {
				agentType = "BareLLM"
			}

			status := "禁用"
			if agent.IsActive() {
				status = "启用"
			}

			name := agent.Name()
			if name == "" {
				name = "(无名称)"
			}

			idStr := agent.ID().String()
			if len(idStr) > 16 {
				idStr = idStr[:16] + "..."
			}
			fmt.Printf("%-20s %-15s %-15s %s\n",
				idStr,
				agentType,
				status,
				name)
		}
		fmt.Println()
	},
}

func init() {
	agentCmd.AddCommand(agentListCmd)
}