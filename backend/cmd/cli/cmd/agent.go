package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Agent 管理",
	Long:  `列出、创建、更新和删除 Agent`,
}

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出 Agent",
	Example: `  taskmanager agent list`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		c := client.New()

		agents, err := c.ListAgents(ctx)
		if err != nil {
			printJSONError("list agents failed: %v", err)
			return
		}

		type AgentInfo struct {
			ID        string `json:"id"`
			AgentCode string `json:"agent_code"`
			Name      string `json:"name"`
			AgentType string `json:"agent_type"`
			IsActive  bool   `json:"is_active"`
		}

		items := make([]AgentInfo, 0, len(agents))
		for _, a := range agents {
			items = append(items, AgentInfo{
				ID:        a.ID,
				AgentCode: a.Code,
				Name:      a.Name,
				AgentType: a.AgentType,
				IsActive:  a.IsActive,
			})
		}

		jsonBytes, _ := json.Marshal(items)
		fmt.Print(string(jsonBytes))
	},
}

var agentCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建 Agent",
	Example: `  taskmanager agent create --user-code <code> --name <name> --type <type> --model <model>
  taskmanager agent create -u user1 -n "CodingAgent" -t coding -m claude-sonnet-4-20250514`,
	Run: func(cmd *cobra.Command, args []string) {
		userCode, _ := cmd.Flags().GetString("user-code")
		name, _ := cmd.Flags().GetString("name")
		agentType, _ := cmd.Flags().GetString("type")
		model, _ := cmd.Flags().GetString("model")
		description, _ := cmd.Flags().GetString("description")
		providerID, _ := cmd.Flags().GetString("provider-id")
		maxTokens, _ := cmd.Flags().GetInt("max-tokens")
		temperature, _ := cmd.Flags().GetFloat64("temperature")
		maxIterations, _ := cmd.Flags().GetInt("max-iterations")
		historyMessages, _ := cmd.Flags().GetInt("history-messages")
		isDefault, _ := cmd.Flags().GetBool("default")
		enableThinking, _ := cmd.Flags().GetBool("thinking")

		if userCode == "" || name == "" || agentType == "" || model == "" {
			printJSONError("--user-code, --name, --type and --model are required")
			return
		}

		ctx := context.Background()
		c := client.New()

		a, err := c.CreateAgent(ctx, client.CreateAgentAPIRequest{
			UserCode:             userCode,
			Name:                 name,
			AgentType:            agentType,
			Description:          description,
			IdentityContent:      "",
			SoulContent:          "",
			AgentsContent:        "",
			UserContent:          "",
			ToolsContent:         "",
			Model:                model,
			LLMProviderID:        providerID,
			MaxTokens:            maxTokens,
			Temperature:          temperature,
			MaxIterations:        maxIterations,
			HistoryMessages:      historyMessages,
			SkillsList:           []string{},
			ToolsList:            []string{},
			IsDefault:            isDefault,
			EnableThinkingProcess: enableThinking,
		})
		if err != nil {
			printJSONError("create agent failed: %v", err)
			return
		}

		result := map[string]string{
			"id":         a.ID,
			"agent_code": a.Code,
			"name":       a.Name,
			"message":    "created",
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

var agentUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新 Agent",
	Example: `  taskmanager agent update <id> --name <name>
  taskmanager agent update <id> --model claude-sonnet-4-20250514`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("agent id is required")
			return
		}
		id := args[0]

		name, _ := cmd.Flags().GetString("name")
		model, _ := cmd.Flags().GetString("model")
		providerID, _ := cmd.Flags().GetString("provider-id")
		maxTokens, _ := cmd.Flags().GetInt("max-tokens")
		temperature, _ := cmd.Flags().GetFloat64("temperature")
		maxIterations, _ := cmd.Flags().GetInt("max-iterations")
		historyMessages, _ := cmd.Flags().GetInt("history-messages")
		isDefault, _ := cmd.Flags().GetBool("default")
		enableThinking, _ := cmd.Flags().GetBool("thinking")

		req := client.UpdateAgentAPIRequest{}
		if cmd.Flags().Changed("name") {
			req.Name = &name
		}
		if cmd.Flags().Changed("model") {
			req.Model = &model
		}
		if cmd.Flags().Changed("provider-id") {
			req.LLMProviderID = &providerID
		}
		if cmd.Flags().Changed("max-tokens") {
			req.MaxTokens = &maxTokens
		}
		if cmd.Flags().Changed("temperature") {
			req.Temperature = &temperature
		}
		if cmd.Flags().Changed("max-iterations") {
			req.MaxIterations = &maxIterations
		}
		if cmd.Flags().Changed("history-messages") {
			req.HistoryMessages = &historyMessages
		}
		if cmd.Flags().Changed("default") {
			req.IsDefault = &isDefault
		}
		if cmd.Flags().Changed("thinking") {
			req.EnableThinkingProcess = &enableThinking
		}

		ctx := context.Background()
		c := client.New()

		a, err := c.UpdateAgent(ctx, id, req)
		if err != nil {
			printJSONError("update agent failed: %v", err)
			return
		}

		result := map[string]string{
			"id":         a.ID,
			"agent_code": a.Code,
			"name":       a.Name,
			"message":    "updated",
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

var agentDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "删除 Agent",
	Example: `  taskmanager agent delete <id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("agent id is required")
			return
		}
		id := args[0]

		ctx := context.Background()
		c := client.New()

		if err := c.DeleteAgent(ctx, id); err != nil {
			printJSONError("delete agent failed: %v", err)
			return
		}

		result := map[string]string{"message": "deleted"}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

var agentEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "启用 Agent",
	Example: `  taskmanager agent enable <id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("agent id is required")
			return
		}
		id := args[0]

		ctx := context.Background()
		c := client.New()

		v := true
		_, err := c.UpdateAgent(ctx, id, client.UpdateAgentAPIRequest{IsActive: &v})
		if err != nil {
			printJSONError("enable agent failed: %v", err)
			return
		}

		result := map[string]string{"message": "enabled"}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

var agentDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "禁用 Agent",
	Example: `  taskmanager agent disable <id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("agent id is required")
			return
		}
		id := args[0]

		ctx := context.Background()
		c := client.New()

		v := false
		_, err := c.UpdateAgent(ctx, id, client.UpdateAgentAPIRequest{IsActive: &v})
		if err != nil {
			printJSONError("disable agent failed: %v", err)
			return
		}

		result := map[string]string{"message": "disabled"}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentCreateCmd)
	agentCmd.AddCommand(agentUpdateCmd)
	agentCmd.AddCommand(agentDeleteCmd)
	agentCmd.AddCommand(agentEnableCmd)
	agentCmd.AddCommand(agentDisableCmd)

	agentCreateCmd.Flags().String("user-code", "", "用户代码 (必填)")
	agentCreateCmd.Flags().StringP("name", "n", "", "Agent 名称 (必填)")
	agentCreateCmd.Flags().StringP("type", "t", "", "Agent 类型: coding, bare_llm (必填)")
	agentCreateCmd.Flags().StringP("model", "m", "", "模型 (必填)")
	agentCreateCmd.Flags().String("description", "", "描述")
	agentCreateCmd.Flags().String("provider-id", "", "LLM Provider ID")
	agentCreateCmd.Flags().Int("max-tokens", 4096, "最大 Token 数")
	agentCreateCmd.Flags().Float64("temperature", 0.7, "温度")
	agentCreateCmd.Flags().Int("max-iterations", 50, "最大迭代次数")
	agentCreateCmd.Flags().Int("history-messages", 20, "历史消息数")
	agentCreateCmd.Flags().Bool("default", false, "设为默认 Agent")
	agentCreateCmd.Flags().Bool("thinking", false, "启用思考过程")
	agentUpdateCmd.Flags().String("name", "", "Agent 名称")
	agentUpdateCmd.Flags().String("model", "", "模型")
	agentUpdateCmd.Flags().String("provider-id", "", "LLM Provider ID")
	agentUpdateCmd.Flags().Int("max-tokens", 0, "最大 Token 数")
	agentUpdateCmd.Flags().Float64("temperature", 0, "温度")
	agentUpdateCmd.Flags().Int("max-iterations", 0, "最大迭代次数")
	agentUpdateCmd.Flags().Int("history-messages", 0, "历史消息数")
	agentUpdateCmd.Flags().Bool("default", false, "设为默认 Agent")
	agentUpdateCmd.Flags().Bool("thinking", false, "启用思考过程")
}