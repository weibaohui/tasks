package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/infrastructure/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置管理",
	Long:  `初始化和显示配置`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化配置文件",
	Example: `  taskmanager config init`,
	Run: func(cmd *cobra.Command, args []string) {
		// 优先使用环境变量指定的配置路径
		configPath := os.Getenv("TASKMANAGER_CONFIG")
		if configPath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Printf("获取 home 目录失败: %v\n", err)
				return
			}
			configPath = filepath.Join(home, ".taskmanager", "config.yaml")
		}

		if err := config.WriteDefaultConfig(configPath); err != nil {
			fmt.Printf("创建配置文件失败: %v\n", err)
			return
		}

		fmt.Printf("配置文件已创建: %s\n", configPath)
		fmt.Println("")
		fmt.Println("请编辑配置文件设置 API Token:")
		fmt.Printf("  vim %s\n", configPath)
		fmt.Println("")
		fmt.Println("获取 Token:")
		fmt.Println("  1. 启动 server: cd backend && go run cmd/server/main.go create-admin")
		fmt.Println("  2. 登录 Web UI，在 Personal Access Token 页面生成 Token")
		fmt.Println("  3. 将 Token 填入配置文件的 api.token 字段")
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "显示当前配置",
	Example: `  taskmanager config show`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("加载配置失败: %v\n", err)
			return
		}

		fmt.Println("当前配置:")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("Database Path: %s\n", cfg.Database.Path)
		fmt.Printf("API Base URL: %s\n", cfg.API.BaseURL)
		if cfg.API.Token != "" {
			fmt.Printf("API Token: %s...\n", cfg.API.Token[:min(8, len(cfg.API.Token))])
		} else {
			fmt.Println("API Token: (未配置)")
		}
		fmt.Printf("Log Level: %s\n", cfg.Logging.Level)
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println("")
		fmt.Println("配置加载来源:")
		configPath := ""
		if p := os.Getenv("TASKMANAGER_CONFIG"); p != "" {
			configPath = fmt.Sprintf("TASKMANAGER_CONFIG=%s", p)
		} else {
			home, _ := os.UserHomeDir()
			defaultPath := filepath.Join(home, ".taskmanager", "config.yaml")
			if _, err := os.Stat(defaultPath); err == nil {
				configPath = fmt.Sprintf("~/.taskmanager/config.yaml (default)")
			} else {
				configPath = "无配置文件，使用环境变量或默认值"
			}
		}
		fmt.Printf("  %s\n", configPath)
		fmt.Println("")
		fmt.Println("环境变量覆盖:")
		if apiURL := os.Getenv("TASKMANAGER_API_BASE_URL"); apiURL != "" {
			fmt.Printf("  TASKMANAGER_API_BASE_URL=%s\n", apiURL)
		}
		if apiURL := os.Getenv("API_BASE_URL"); apiURL != "" {
			fmt.Printf("  API_BASE_URL=%s\n", apiURL)
		}
		if port := os.Getenv("TASKMANAGER_SERVER_PORT"); port != "" {
			fmt.Printf("  TASKMANAGER_SERVER_PORT=%s\n", port)
		}
	},
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}