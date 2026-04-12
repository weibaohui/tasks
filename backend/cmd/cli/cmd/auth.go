package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/infrastructure/config"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "认证配置",
	Long: `配置 TaskManager 服务地址和访问令牌

示例:
  taskmanager auth http://localhost:13618/api/v1 your_token_here

这会将服务地址和 Token 写入配置文件 (~/.taskmanager/config.yaml)`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		serverURL := args[0]
		token := args[1]

		if serverURL == "" || token == "" {
			fmt.Println("错误: 服务地址和 Token 不能为空")
			os.Exit(1)
		}

		// 确保配置目录存在
		if err := config.EnsureConfigDir(); err != nil {
			fmt.Printf("创建配置目录失败: %v\n", err)
			os.Exit(1)
		}

		// 获取配置路径
		home, _ := os.UserHomeDir()
		configPath := fmt.Sprintf("%s/.taskmanager/config.yaml", home)

		// 尝试加载现有配置
		cfg, err := config.LoadFromPath(configPath)
		if err != nil {
			if os.IsNotExist(err) {
				// 文件不存在，使用默认配置
				cfg = defaultConfigForAuth()
			} else {
				fmt.Printf("加载配置文件失败: %v\n", err)
				os.Exit(1)
			}
		}

		// 更新配置
		cfg.API.BaseURL = serverURL
		cfg.API.Token = token

		// 写入配置
		if err := config.SaveConfig(configPath, cfg); err != nil {
			fmt.Printf("保存配置失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("认证配置已保存!")
		fmt.Printf("  服务地址: %s\n", serverURL)
		fmt.Println("  Token: ******** (已隐藏)")
		fmt.Println("")
		fmt.Println("现在可以使用 taskmanager CLI 管理任务了。")
	},
}

func registerAuthCommands() {
}

// defaultConfigForAuth 返回认证专用的默认配置
func defaultConfigForAuth() *config.Config {
	home, _ := os.UserHomeDir()
	defaultDBPath := filepath.Join(home, ".taskmanager", "data.db")

	return &config.Config{
		Server: config.ServerConfig{
			Port: 13618,
		},
		Database: config.DatabaseConfig{
			Path: defaultDBPath,
		},
		API: config.APIConfig{
			BaseURL: "http://localhost:13618/api/v1",
		},
		Logging: config.LoggingConfig{
			Level: "info",
		},
	}
}
