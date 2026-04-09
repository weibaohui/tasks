package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "会话管理",
	Long:  `列出和删除会话`,
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出会话",
	Example: `  taskmanager session list
  taskmanager session list --user-code <code>`,
	Run: func(cmd *cobra.Command, args []string) {
		userCode, _ := cmd.Flags().GetString("user-code")

		ctx := context.Background()
		c := client.New()

		sessions, err := c.ListUserSessions(ctx, userCode)
		if err != nil {
			printJSONError("list sessions failed: %v", err)
			return
		}

		type SessionInfo struct {
			SessionKey  string `json:"session_key"`
			UserCode    string `json:"user_code"`
			ChannelCode string `json:"channel_code"`
			AgentCode   string `json:"agent_code"`
			Status      string `json:"status"`
			CreatedAt   int64  `json:"created_at"`
		}

		items := make([]SessionInfo, 0, len(sessions))
		for _, s := range sessions {
			items = append(items, SessionInfo{
				SessionKey:  s.SessionKey,
				UserCode:    s.UserCode,
				ChannelCode: s.ChannelCode,
				AgentCode:   s.AgentCode,
				Status:      s.Status,
				CreatedAt:   s.CreatedAt,
			})
		}

		jsonBytes, _ := json.Marshal(items)
		fmt.Print(string(jsonBytes))
	},
}

var sessionDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "删除会话",
	Example: `  taskmanager session delete <session-key>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("session key is required")
			return
		}
		sessionKey := args[0]

		ctx := context.Background()
		c := client.New()

		if err := c.DeleteSession(ctx, sessionKey); err != nil {
			printJSONError("delete session failed: %v", err)
			return
		}

		result := map[string]string{"message": "deleted"}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionDeleteCmd)

	sessionListCmd.Flags().String("user-code", "", "用户代码 (可选)")
}