package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var projectScenarioCmd = &cobra.Command{
	Use:   "scenario",
	Short: "项目心跳场景管理",
	Long:  `为项目应用预定义的心跳场景，一键创建完整的心跳集`,
}

var scenarioListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有心跳场景",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		c := client.New()

		scenarios, err := c.ListHeartbeatScenarios(ctx)
		if err != nil {
			fmt.Printf("列出场景失败: %v\n", err)
			return
		}

		fmt.Println("\n心跳场景列表:")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("%-20s %-10s %-10s %s\n", "场景编码", "内置", "条目数", "名称")
		fmt.Println("--------------------------------------------------------------------------------")
		for _, s := range scenarios {
			builtIn := "否"
			if s.IsBuiltIn {
				builtIn = "是"
			}
			fmt.Printf("%-20s %-10s %-10d %s\n", s.Code, builtIn, len(s.Items), s.Name)
		}
		fmt.Println()
	},
}

var scenarioApplyCmd = &cobra.Command{
	Use:   "apply <project_id>",
	Short: "为项目应用心跳场景",
	Example: `  taskmanager project scenario apply <project_id> --code github_dev_workflow`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectID := args[0]
		code, _ := cmd.Flags().GetString("code")
		if code == "" {
			fmt.Println("错误: --code 参数必填")
			cmd.Usage()
			return
		}

		ctx := context.Background()
		c := client.New()

		if err := c.ApplyHeartbeatScenario(ctx, projectID, code); err != nil {
			fmt.Printf("应用场景失败: %v\n", err)
			return
		}
		fmt.Printf("场景 '%s' 已成功应用到项目 %s\n", code, projectID)
	},
}

func init() {
	scenarioApplyCmd.Flags().String("code", "", "场景编码")

	projectScenarioCmd.AddCommand(scenarioListCmd)
	projectScenarioCmd.AddCommand(scenarioApplyCmd)
}
