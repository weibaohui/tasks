package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "taskmanager",
	Short: "TaskManager CLI - 管理任务和需求",
	Long: `TaskManager CLI 是一个用于管理任务和需求 команд-line 工具。

常用命令:
  taskmanager requirement create  创建新需求
  taskmanager requirement list    列出需求
  taskmanager requirement dispatch 派发需求
  taskmanager project list        列出项目
  taskmanager agent list         列出 Agent

配置:
  配置文件路径: ~/.taskmanager/config.yaml
  环境变量: TASKMANAGER_CONFIG (配置文件路径)
  环境变量: TASKMANAGER_DB_PATH (数据库路径)
  环境变量: TASKMANAGER_API_BASE_URL (API 地址)`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func GetRootCmd() *cobra.Command {
	return rootCmd
}
