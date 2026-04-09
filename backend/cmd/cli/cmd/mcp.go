package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP 服务器管理",
	Long:  `列出、创建、更新、删除 MCP 服务器，测试连接和刷新工具`,
}

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出 MCP 服务器",
	Example: `  taskmanager mcp list`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		c := client.New()

		servers, err := c.ListMCPServers(ctx)
		if err != nil {
			printJSONError("list mcp servers failed: %v", err)
			return
		}

		type MCPServerInfo struct {
			ID            string   `json:"id"`
			Code          string   `json:"code"`
			Name          string   `json:"name"`
			TransportType string   `json:"transport_type"`
			Status        string   `json:"status"`
			ToolsCount    int      `json:"tools_count"`
			LastConnected *int64   `json:"last_connected,omitempty"`
		}

		items := make([]MCPServerInfo, 0, len(servers))
		for _, s := range servers {
			items = append(items, MCPServerInfo{
				ID:            s.ID,
				Code:          s.Code,
				Name:          s.Name,
				TransportType: s.TransportType,
				Status:        s.Status,
				ToolsCount:    len(s.Capabilities),
				LastConnected: s.LastConnected,
			})
		}

		jsonBytes, _ := json.Marshal(items)
		fmt.Print(string(jsonBytes))
	},
}

var mcpCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建 MCP 服务器",
	Example: `  taskmanager mcp create --code <code> --name <name> --transport <type> --command <cmd>
  taskmanager mcp create -c local-stdio -n "Local STDIO" -t stdio --command node --args server.js`,
	Run: func(cmd *cobra.Command, args []string) {
		code, _ := cmd.Flags().GetString("code")
		name, _ := cmd.Flags().GetString("name")
		transport, _ := cmd.Flags().GetString("transport")
		command, _ := cmd.Flags().GetString("command")
		serverArgs, _ := cmd.Flags().GetStringArray("args")
		url, _ := cmd.Flags().GetString("url")

		if code == "" || name == "" || transport == "" {
			printJSONError("--code, --name and --transport are required")
			return
		}

		ctx := context.Background()
		c := client.New()

		s, err := c.CreateMCPServer(ctx, client.CreateMCPServerAPIRequest{
			Code:          code,
			Name:          name,
			TransportType: transport,
			Command:       command,
			Args:          serverArgs,
			URL:           url,
			EnvVars:       make(map[string]string),
		})
		if err != nil {
			printJSONError("create mcp server failed: %v", err)
			return
		}

		result := map[string]string{
			"id":      s.ID,
			"code":    s.Code,
			"name":    s.Name,
			"message": "created",
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

var mcpUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新 MCP 服务器",
	Example: `  taskmanager mcp update <id> --name <name>
  taskmanager mcp update <id> --command node --args new-server.js`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("mcp server id is required")
			return
		}
		id := args[0]

		name, _ := cmd.Flags().GetString("name")
		transport, _ := cmd.Flags().GetString("transport")
		command, _ := cmd.Flags().GetString("command")
		serverArgs, _ := cmd.Flags().GetStringArray("args")
		url, _ := cmd.Flags().GetString("url")

		req := client.UpdateMCPServerAPIRequest{}
		if cmd.Flags().Changed("name") {
			req.Name = &name
		}
		if cmd.Flags().Changed("transport") {
			req.TransportType = &transport
		}
		if cmd.Flags().Changed("command") {
			req.Command = &command
		}
		if cmd.Flags().Changed("args") {
			req.Args = &serverArgs
		}
		if cmd.Flags().Changed("url") {
			req.URL = &url
		}

		ctx := context.Background()
		c := client.New()

		s, err := c.UpdateMCPServer(ctx, id, req)
		if err != nil {
			printJSONError("update mcp server failed: %v", err)
			return
		}

		result := map[string]string{
			"id":      s.ID,
			"code":    s.Code,
			"name":    s.Name,
			"message": "updated",
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

var mcpDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "删除 MCP 服务器",
	Example: `  taskmanager mcp delete <id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("mcp server id is required")
			return
		}
		id := args[0]

		ctx := context.Background()
		c := client.New()

		if err := c.DeleteMCPServer(ctx, id); err != nil {
			printJSONError("delete mcp server failed: %v", err)
			return
		}

		result := map[string]string{"message": "deleted"}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

var mcpTestCmd = &cobra.Command{
	Use:   "test",
	Short: "测试 MCP 服务器连接",
	Example: `  taskmanager mcp test <id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("mcp server id is required")
			return
		}
		id := args[0]

		ctx := context.Background()
		c := client.New()

		if err := c.TestMCPServer(ctx, id); err != nil {
			printJSONError("test mcp server failed: %v", err)
			return
		}

		result := map[string]string{"message": "connection successful"}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

var mcpRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "刷新 MCP 服务器工具能力",
	Example: `  taskmanager mcp refresh <id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("mcp server id is required")
			return
		}
		id := args[0]

		ctx := context.Background()
		c := client.New()

		if err := c.RefreshMCPServer(ctx, id); err != nil {
			printJSONError("refresh mcp server failed: %v", err)
			return
		}

		result := map[string]string{"message": "tools refreshed"}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	mcpCmd.AddCommand(mcpListCmd)
	mcpCmd.AddCommand(mcpCreateCmd)
	mcpCmd.AddCommand(mcpUpdateCmd)
	mcpCmd.AddCommand(mcpDeleteCmd)
	mcpCmd.AddCommand(mcpTestCmd)
	mcpCmd.AddCommand(mcpRefreshCmd)

	mcpCreateCmd.Flags().StringP("code", "c", "", "服务器编码 (必填)")
	mcpCreateCmd.Flags().StringP("name", "n", "", "服务器名称 (必填)")
	mcpCreateCmd.Flags().StringP("transport", "t", "", "传输类型: stdio, http, sse (必填)")
	mcpCreateCmd.Flags().String("command", "", "命令 (stdio 必填)")
	mcpCreateCmd.Flags().StringArray("args", []string{}, "参数 (可多次指定)")
	mcpCreateCmd.Flags().String("url", "", "服务器 URL (http/sse 必填)")
	mcpUpdateCmd.Flags().String("name", "", "服务器名称")
	mcpUpdateCmd.Flags().String("transport", "", "传输类型")
	mcpUpdateCmd.Flags().String("command", "", "命令")
	mcpUpdateCmd.Flags().StringArray("args", []string{}, "参数")
	mcpUpdateCmd.Flags().String("url", "", "服务器 URL")
}