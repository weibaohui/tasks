package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

// ==================== Heartbeat Subcommands ====================

var projectHeartbeatCmd = &cobra.Command{
	Use:   "heartbeat",
	Short: "心跳管理",
	Long:  `管理项目的心跳配置，支持一个项目配置多个心跳`,
}

var heartbeatListCmd = &cobra.Command{
	Use:   "list <project_id>",
	Short: "列出项目心跳",
	Example: `  taskmanager project heartbeat list <project_id>`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectID := args[0]
		ctx := context.Background()
		c := client.New()

		heartbeats, err := c.ListHeartbeats(ctx, projectID)
		if err != nil {
			fmt.Printf("列出心跳失败: %v\n", err)
			return
		}

		fmt.Println("\n项目心跳列表:")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("%-20s %-10s %-10s %-15s %-15s %s\n", "心跳ID", "状态", "间隔(分钟)", "Agent编码", "需求类型", "名称")
		fmt.Println("--------------------------------------------------------------------------------")
		for _, hb := range heartbeats {
			idStr := hb.ID
			if len(idStr) > 18 {
				idStr = idStr[:18] + "..."
			}
			status := "关闭"
			if hb.Enabled {
				status = "开启"
			}
			agentCode := hb.AgentCode
			if agentCode == "" {
				agentCode = "-"
			}
			reqType := hb.RequirementType
			if reqType == "" {
				reqType = "heartbeat"
			}
			fmt.Printf("%-20s %-10s %-10d %-15s %-15s %s\n",
				idStr,
				status,
				hb.IntervalMinutes,
				agentCode,
				reqType,
				hb.Name)
		}
		fmt.Println()
	},
}

var heartbeatCreateCmd = &cobra.Command{
	Use:   "create <project_id>",
	Short: "创建心跳",
	Example: `  taskmanager project heartbeat create <project_id> --name <name> --interval <minutes> --agent-code <code> --type <type>`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectID := args[0]
		name, _ := cmd.Flags().GetString("name")
		interval, _ := cmd.Flags().GetInt("interval")
		agentCode, _ := cmd.Flags().GetString("agent-code")
		reqType, _ := cmd.Flags().GetString("type")
		content, _ := cmd.Flags().GetString("content")

		if name == "" || agentCode == "" {
			fmt.Println("错误: --name 和 --agent-code 参数必填")
			cmd.Usage()
			return
		}
		if interval < 1 {
			interval = 30
		}
		if reqType == "" {
			reqType = "heartbeat"
		}

		ctx := context.Background()
		c := client.New()

		hb, err := c.CreateHeartbeat(ctx, client.CreateHeartbeatRequest{
			ProjectID:       projectID,
			Name:            name,
			IntervalMinutes: interval,
			MDContent:       content,
			AgentCode:       agentCode,
			RequirementType: reqType,
		})
		if err != nil {
			fmt.Printf("创建心跳失败: %v\n", err)
			return
		}

		fmt.Printf("心跳创建成功!\n")
		fmt.Printf("  ID:   %s\n", hb.ID)
		fmt.Printf("  名称: %s\n", hb.Name)
		fmt.Printf("  间隔: %d 分钟\n", hb.IntervalMinutes)
	},
}

var heartbeatUpdateCmd = &cobra.Command{
	Use:   "update <heartbeat_id>",
	Short: "更新心跳",
	Example: `  taskmanager project heartbeat update <heartbeat_id> --name <name> --interval <minutes>`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		heartbeatID := args[0]
		name, _ := cmd.Flags().GetString("name")
		interval, _ := cmd.Flags().GetInt("interval")
		agentCode, _ := cmd.Flags().GetString("agent-code")
		reqType, _ := cmd.Flags().GetString("type")
		content, _ := cmd.Flags().GetString("content")
		enabledStr, _ := cmd.Flags().GetString("enabled")

		ctx := context.Background()
		c := client.New()

		hb, err := c.GetHeartbeat(ctx, heartbeatID)
		if err != nil {
			fmt.Printf("查找心跳失败: %v\n", err)
			return
		}

		req := client.UpdateHeartbeatRequest{
			Name:            hb.Name,
			IntervalMinutes: hb.IntervalMinutes,
			MDContent:       hb.MDContent,
			AgentCode:       hb.AgentCode,
			RequirementType: hb.RequirementType,
			Enabled:         hb.Enabled,
		}
		if cmd.Flags().Changed("name") {
			req.Name = name
		}
		if cmd.Flags().Changed("interval") {
			req.IntervalMinutes = interval
		}
		if cmd.Flags().Changed("agent-code") {
			req.AgentCode = agentCode
		}
		if cmd.Flags().Changed("type") {
			req.RequirementType = reqType
		}
		if cmd.Flags().Changed("content") {
			req.MDContent = content
		}
		if enabledStr != "" {
			req.Enabled = enabledStr == "true"
		}

		updated, err := c.UpdateHeartbeat(ctx, heartbeatID, req)
		if err != nil {
			fmt.Printf("更新心跳失败: %v\n", err)
			return
		}

		fmt.Printf("心跳更新成功: %s\n", updated.Name)
	},
}

var heartbeatDeleteCmd = &cobra.Command{
	Use:   "delete <heartbeat_id>",
	Short: "删除心跳",
	Example: `  taskmanager project heartbeat delete <heartbeat_id>`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		heartbeatID := args[0]
		ctx := context.Background()
		c := client.New()

		if err := c.DeleteHeartbeat(ctx, heartbeatID); err != nil {
			fmt.Printf("删除心跳失败: %v\n", err)
			return
		}
		fmt.Printf("心跳已删除: %s\n", heartbeatID)
	},
}

var heartbeatEnableCmd = &cobra.Command{
	Use:   "enable <heartbeat_id>",
	Short: "开启心跳",
	Example: `  taskmanager project heartbeat enable <heartbeat_id>`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		heartbeatID := args[0]
		ctx := context.Background()
		c := client.New()

		hb, err := c.GetHeartbeat(ctx, heartbeatID)
		if err != nil {
			fmt.Printf("查找心跳失败: %v\n", err)
			return
		}

		req := client.UpdateHeartbeatRequest{
			Name:            hb.Name,
			IntervalMinutes: hb.IntervalMinutes,
			MDContent:       hb.MDContent,
			AgentCode:       hb.AgentCode,
			RequirementType: hb.RequirementType,
			Enabled:         true,
		}
		_, err = c.UpdateHeartbeat(ctx, heartbeatID, req)
		if err != nil {
			fmt.Printf("开启心跳失败: %v\n", err)
			return
		}
		fmt.Printf("心跳已开启: %s\n", heartbeatID)
	},
}

var heartbeatDisableCmd = &cobra.Command{
	Use:   "disable <heartbeat_id>",
	Short: "关闭心跳",
	Example: `  taskmanager project heartbeat disable <heartbeat_id>`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		heartbeatID := args[0]
		ctx := context.Background()
		c := client.New()

		hb, err := c.GetHeartbeat(ctx, heartbeatID)
		if err != nil {
			fmt.Printf("查找心跳失败: %v\n", err)
			return
		}

		req := client.UpdateHeartbeatRequest{
			Name:            hb.Name,
			IntervalMinutes: hb.IntervalMinutes,
			MDContent:       hb.MDContent,
			AgentCode:       hb.AgentCode,
			RequirementType: hb.RequirementType,
			Enabled:         false,
		}
		_, err = c.UpdateHeartbeat(ctx, heartbeatID, req)
		if err != nil {
			fmt.Printf("关闭心跳失败: %v\n", err)
			return
		}
		fmt.Printf("心跳已关闭: %s\n", heartbeatID)
	},
}

var heartbeatTriggerCmd = &cobra.Command{
	Use:   "trigger <heartbeat_id>",
	Short: "手动触发一次心跳",
	Example: `  taskmanager project heartbeat trigger <heartbeat_id>`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		heartbeatID := args[0]
		ctx := context.Background()
		c := client.New()

		if err := c.TriggerHeartbeat(ctx, heartbeatID); err != nil {
			fmt.Printf("触发心跳失败: %v\n", err)
			return
		}
		fmt.Printf("心跳已触发: %s\n", heartbeatID)
	},
}

func init() {
	// create flags
	heartbeatCreateCmd.Flags().String("name", "", "心跳名称")
	heartbeatCreateCmd.Flags().Int("interval", 30, "心跳间隔（分钟）")
	heartbeatCreateCmd.Flags().String("agent-code", "", "执行 Agent 编码")
	heartbeatCreateCmd.Flags().String("type", "heartbeat", "生成的需求类型")
	heartbeatCreateCmd.Flags().String("content", "", "心跳 Prompt 模板内容")

	// update flags
	heartbeatUpdateCmd.Flags().String("name", "", "心跳名称")
	heartbeatUpdateCmd.Flags().Int("interval", 30, "心跳间隔（分钟）")
	heartbeatUpdateCmd.Flags().String("agent-code", "", "执行 Agent 编码")
	heartbeatUpdateCmd.Flags().String("type", "heartbeat", "生成的需求类型")
	heartbeatUpdateCmd.Flags().String("content", "", "心跳 Prompt 模板内容")
	heartbeatUpdateCmd.Flags().String("enabled", "", "是否启用（true/false）")

	projectHeartbeatCmd.AddCommand(heartbeatListCmd)
	projectHeartbeatCmd.AddCommand(heartbeatCreateCmd)
	projectHeartbeatCmd.AddCommand(heartbeatUpdateCmd)
	projectHeartbeatCmd.AddCommand(heartbeatDeleteCmd)
	projectHeartbeatCmd.AddCommand(heartbeatEnableCmd)
	projectHeartbeatCmd.AddCommand(heartbeatDisableCmd)
	projectHeartbeatCmd.AddCommand(heartbeatTriggerCmd)

	// 兼容旧命令别名（弃用提示）
	_ = strconv.Atoi
}
