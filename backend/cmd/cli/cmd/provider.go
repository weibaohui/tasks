package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/cmd/cli/client"
)

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "LLM Provider 管理",
	Long:  `列出、创建、更新和删除 LLM Provider`,
}

var providerListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出 Provider",
	Example: `  taskmanager provider list
  taskmanager provider list --user-code <code>`,
	Run: func(cmd *cobra.Command, args []string) {
		userCode, _ := cmd.Flags().GetString("user-code")

		ctx := context.Background()
		c := client.New()

		providers, err := c.ListProviders(ctx, userCode)
		if err != nil {
			printJSONError("list providers failed: %v", err)
			return
		}

		type ProviderInfo struct {
			ID           string               `json:"id"`
			ProviderKey  string               `json:"provider_key"`
			ProviderName string               `json:"provider_name"`
			APIBase      string               `json:"api_base"`
			ProviderType string               `json:"provider_type"`
			DefaultModel string               `json:"default_model"`
			IsDefault    bool                 `json:"is_default"`
			IsActive     bool                 `json:"is_active"`
			Priority     int                  `json:"priority"`
		}

		items := make([]ProviderInfo, 0, len(providers))
		for _, p := range providers {
			items = append(items, ProviderInfo{
				ID:           p.ID,
				ProviderKey:  p.ProviderKey,
				ProviderName: p.ProviderName,
				APIBase:      p.APIBase,
				ProviderType: p.ProviderType,
				DefaultModel: p.DefaultModel,
				IsDefault:    p.IsDefault,
				IsActive:     p.IsActive,
				Priority:     p.Priority,
			})
		}

		jsonBytes, _ := json.Marshal(items)
		fmt.Print(string(jsonBytes))
	},
}

var providerCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建 Provider",
	Example: `  taskmanager provider create --user-code <code> --key <key> --name <name> --api-base <url> --api-key <key> --type <type>
  taskmanager provider create -u user1 -k openai -n "OpenAI" --api-base https://api.openai.com/v1 --api-key sk-xxx`,
	Run: func(cmd *cobra.Command, args []string) {
		userCode, _ := cmd.Flags().GetString("user-code")
		key, _ := cmd.Flags().GetString("key")
		name, _ := cmd.Flags().GetString("name")
		apiBase, _ := cmd.Flags().GetString("api-base")
		apiKey, _ := cmd.Flags().GetString("api-key")
		providerType, _ := cmd.Flags().GetString("type")
		defaultModel, _ := cmd.Flags().GetString("default-model")
		priority, _ := cmd.Flags().GetInt("priority")
		isDefault, _ := cmd.Flags().GetBool("default")
		autoMerge, _ := cmd.Flags().GetBool("auto-merge")

		if userCode == "" || key == "" || apiBase == "" {
			printJSONError("--user-code, --key and --api-base are required")
			return
		}

		if providerType == "" {
			providerType = "openai"
		}

		ctx := context.Background()
		c := client.New()

		p, err := c.CreateProvider(ctx, client.CreateProviderAPIRequest{
			UserCode:        userCode,
			ProviderKey:     key,
			ProviderName:    name,
			APIBase:         apiBase,
			APIKey:          apiKey,
			ProviderType:    providerType,
			ExtraHeaders:    make(map[string]string),
			SupportedModels: nil,
			DefaultModel:    defaultModel,
			IsDefault:       isDefault,
			Priority:        priority,
			AutoMerge:       autoMerge,
		})
		if err != nil {
			printJSONError("create provider failed: %v", err)
			return
		}

		result := map[string]string{
			"id":            p.ID,
			"provider_key":  p.ProviderKey,
			"provider_name": p.ProviderName,
			"message":       "created",
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

var providerUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新 Provider",
	Example: `  taskmanager provider update <id> --name <name>
  taskmanager provider update <id> --default-model gpt-4
  taskmanager provider update <id> --active`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("provider id is required")
			return
		}
		id := args[0]

		name, _ := cmd.Flags().GetString("name")
		apiBase, _ := cmd.Flags().GetString("api-base")
		apiKey, _ := cmd.Flags().GetString("api-key")
		providerType, _ := cmd.Flags().GetString("type")
		defaultModel, _ := cmd.Flags().GetString("default-model")
		priority, _ := cmd.Flags().GetInt("priority")
		isDefault, _ := cmd.Flags().GetBool("default")
		isActive, _ := cmd.Flags().GetBool("active")
		autoMerge, _ := cmd.Flags().GetBool("auto-merge")

		req := client.UpdateProviderAPIRequest{}
		if cmd.Flags().Changed("name") {
			req.ProviderName = &name
		}
		if cmd.Flags().Changed("api-base") {
			req.APIBase = &apiBase
		}
		if cmd.Flags().Changed("api-key") && apiKey != "" {
			req.APIKey = &apiKey
		}
		if cmd.Flags().Changed("type") {
			req.ProviderType = &providerType
		}
		if cmd.Flags().Changed("default-model") {
			req.DefaultModel = &defaultModel
		}
		if cmd.Flags().Changed("priority") {
			req.Priority = &priority
		}
		if cmd.Flags().Changed("default") {
			req.IsDefault = &isDefault
		}
		if cmd.Flags().Changed("active") {
			req.IsActive = &isActive
		}
		if cmd.Flags().Changed("auto-merge") {
			req.AutoMerge = &autoMerge
		}

		ctx := context.Background()
		c := client.New()

		p, err := c.UpdateProvider(ctx, id, req)
		if err != nil {
			printJSONError("update provider failed: %v", err)
			return
		}

		result := map[string]string{
			"id":            p.ID,
			"provider_key":  p.ProviderKey,
			"provider_name": p.ProviderName,
			"message":       "updated",
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

var providerDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "删除 Provider",
	Example: `  taskmanager provider delete <id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("provider id is required")
			return
		}
		id := args[0]

		ctx := context.Background()
		c := client.New()

		if err := c.DeleteProvider(ctx, id); err != nil {
			printJSONError("delete provider failed: %v", err)
			return
		}

		result := map[string]string{"message": "deleted"}
		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

var providerTestCmd = &cobra.Command{
	Use:   "test",
	Short: "测试 Provider 连接",
	Example: `  taskmanager provider test <id>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			printJSONError("provider id is required")
			return
		}
		id := args[0]

		ctx := context.Background()
		c := client.New()

		result, err := c.TestProvider(ctx, id)
		if err != nil {
			printJSONError("test provider failed: %v", err)
			return
		}

		jsonBytes, _ := json.Marshal(result)
		fmt.Print(string(jsonBytes))
	},
}

func init() {
	providerCmd.AddCommand(providerListCmd)
	providerCmd.AddCommand(providerCreateCmd)
	providerCmd.AddCommand(providerUpdateCmd)
	providerCmd.AddCommand(providerDeleteCmd)
	providerCmd.AddCommand(providerTestCmd)

	providerListCmd.Flags().String("user-code", "", "用户代码 (可选)")
	providerCreateCmd.Flags().String("user-code", "", "用户代码 (必填)")
	providerCreateCmd.Flags().StringP("key", "k", "", "Provider Key (必填)")
	providerCreateCmd.Flags().StringP("name", "n", "", "Provider 名称")
	providerCreateCmd.Flags().String("api-base", "", "API Base URL (必填)")
	providerCreateCmd.Flags().String("api-key", "", "API Key")
	providerCreateCmd.Flags().String("type", "", "Provider 类型: openai, anthropic (默认: openai)")
	providerCreateCmd.Flags().String("default-model", "", "默认模型")
	providerCreateCmd.Flags().Int("priority", 0, "优先级")
	providerCreateCmd.Flags().Bool("default", false, "设为默认 Provider")
	providerCreateCmd.Flags().Bool("auto-merge", true, "自动合并")
	providerUpdateCmd.Flags().String("name", "", "Provider 名称")
	providerUpdateCmd.Flags().String("api-base", "", "API Base URL")
	providerUpdateCmd.Flags().String("api-key", "", "API Key (留空则不更新)")
	providerUpdateCmd.Flags().String("type", "", "Provider 类型")
	providerUpdateCmd.Flags().String("default-model", "", "默认模型")
	providerUpdateCmd.Flags().Int("priority", 0, "优先级")
	providerUpdateCmd.Flags().Bool("default", false, "设为默认 Provider")
	providerUpdateCmd.Flags().Bool("active", false, "启用 Provider")
	providerUpdateCmd.Flags().Bool("auto-merge", true, "自动合并")
}