package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var channelCmd = &cobra.Command{
	Use:   "channel",
	Short: "渠道管理",
	Long:  `列出、创建、更新和删除渠道`,
}

var channelListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出渠道",
	Example: `  taskmanager channel list
  taskmanager channel list --user-code <code>`,
	Run: func(cmd *cobra.Command, args []string) {
		userCode, _ := cmd.Flags().GetString("user-code")

		ctx := context.Background()
		c := client.New()

		channels, err := c.ListChannels(ctx, userCode)
		if err != nil {
			printJSONError("list channels failed: %v", err)
			return
		}

		type ChannelInfo struct {
			ID          string   `json:"id"`
			ChannelCode string   `json:"channel_code"`
			Name        string   `json:"name"`
			Type        string   `json:"type"`
			AgentCode   string   `json:"agent_code"`
			IsActive    bool     `json:"is_active"`
			AllowFrom   []string `json:"allow_from"`
		}

		items := make([]ChannelInfo, 0, len(channels))
		for _, ch := range channels {
			items = append(items, ChannelInfo{
				ID:          ch.ID,
				ChannelCode: ch.ChannelCode,
				Name:        ch.Name,
				Type:        ch.Type,
				AgentCode:   ch.AgentCode,
				IsActive:    ch.IsActive,
				AllowFrom:   ch.AllowFrom,
			})
		}

		jsonBytes, _ := json.Marshal(items)
		fmt.Print(string(jsonBytes))
	},
}

var channelCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建渠道",
	Example: `  taskmanager channel create --name <name> --type <type> --user-code <code>
  taskmanager channel create -n mychannel -t feishu -u user1`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		channelType, _ := cmd.Flags().GetString("type")
		userCode, _ := cmd.Flags().GetString("user-code")
		agentCode, _ := cmd.Flags().GetString("agent-code")
		allowFrom, _ := cmd.Flags().GetStringArray("allow-from")

		if name == "" || channelType == "" || userCode == "" {
			printJSONError("--name, --type and --user-code are required")
			return
		}

		ctx := context.Background()
		c := client.New()

		ch, err := c.CreateChannel(ctx, client.CreateChannelAPIRequest{
			UserCode:  userCode,
			Name:      name,
			Type:      channelType,
			Config:    make(map[string]interface{}),
			AllowFrom: allowFrom,
			AgentCode: agentCode,
		})
		if err != nil {
			printJSONError("create channel failed: %v", err)
			return
		}

		result := map[string]string{
			"id":           ch.ID,
			"channel_code": ch.ChannelCode,
			"name":         ch.Name,
			"type":         ch.Type,
			"message":      "created",
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

var channelUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新渠道",
	Example: `  taskmanager channel update <id> --name <name>
  taskmanager channel update <id> --active
  taskmanager channel update <id> --inactive`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("channel id is required")
			return
		}
		id := args[0]

		name, _ := cmd.Flags().GetString("name")
		agentCode, _ := cmd.Flags().GetString("agent-code")
		allowFrom, _ := cmd.Flags().GetStringArray("allow-from")

		req := client.UpdateChannelAPIRequest{}
		if cmd.Flags().Changed("name") {
			req.Name = &name
		}
		if cmd.Flags().Changed("agent-code") {
			req.AgentCode = &agentCode
		}
		if cmd.Flags().Changed("allow-from") {
			req.AllowFrom = &allowFrom
		}
		if cmd.Flags().Changed("active") {
			v := true
			req.IsActive = &v
		}
		if cmd.Flags().Changed("inactive") {
			v := false
			req.IsActive = &v
		}

		ctx := context.Background()
		c := client.New()

		ch, err := c.UpdateChannel(ctx, id, req)
		if err != nil {
			printJSONError("update channel failed: %v", err)
			return
		}

		result := map[string]string{
			"id":           ch.ID,
			"channel_code": ch.ChannelCode,
			"name":         ch.Name,
			"message":      "updated",
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

var channelDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "删除渠道",
	Example: `  taskmanager channel delete <id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("channel id is required")
			return
		}
		id := args[0]

		ctx := context.Background()
		c := client.New()

		if err := c.DeleteChannel(ctx, id); err != nil {
			printJSONError("delete channel failed: %v", err)
			return
		}

		result := map[string]string{"message": "deleted"}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func registerChannelCommands() {
	channelCmd.AddCommand(channelListCmd)
	channelCmd.AddCommand(channelCreateCmd)
	channelCmd.AddCommand(channelUpdateCmd)
	channelCmd.AddCommand(channelDeleteCmd)

	channelListCmd.Flags().String("user-code", "", "用户代码 (可选)")
	channelCreateCmd.Flags().StringP("name", "n", "", "渠道名称 (必填)")
	channelCreateCmd.Flags().StringP("type", "t", "", "渠道类型: feishu, dingtalk, matrix, websocket (必填)")
	channelCreateCmd.Flags().String("user-code", "", "用户代码 (必填)")
	channelCreateCmd.Flags().String("agent-code", "", "绑定的 Agent Code")
	channelCreateCmd.Flags().StringArray("allow-from", []string{}, "白名单用户 (可多次指定)")
	channelUpdateCmd.Flags().String("name", "", "渠道名称")
	channelUpdateCmd.Flags().String("agent-code", "", "绑定的 Agent Code")
	channelUpdateCmd.Flags().StringArray("allow-from", []string{}, "白名单用户")
	channelUpdateCmd.Flags().Bool("active", false, "启用渠道")
	channelUpdateCmd.Flags().Bool("inactive", false, "禁用渠道")
}