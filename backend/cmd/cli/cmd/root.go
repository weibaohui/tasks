package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "taskmanager",
	Short: "TaskManager CLI - 管理任务和需求",
	Long: `TaskManager CLI 是一个用于管理任务和需求 command-line 工具。

服务管理:
  核心服务 (server) - 处理消息、任务调度、渠道网关:
    taskmanager server start        启动核心服务
    taskmanager server stop         停止核心服务
    taskmanager server status       查看核心服务状态
    taskmanager server logs         查看核心服务日志

  Web 服务 (web) - HTTP API + 前端界面:
    taskmanager web start           启动 Web 服务
    taskmanager web stop            停止 Web 服务
    taskmanager web status          查看 Web 服务状态
    taskmanager web logs            查看 Web 服务日志

常用命令:
  taskmanager requirement create    创建新需求
  taskmanager requirement list      列出需求
  taskmanager requirement dispatch  派发需求
  taskmanager project list          列出项目
  taskmanager agent list            列出 Agent

配置:
  配置文件路径: ~/.taskmanager/config.yaml
  环境变量: TASKMANAGER_CONFIG (配置文件路径)
  环境变量: TASKMANAGER_API_BASE_URL (API 地址)`,
}

// Execute 执行根命令并处理错误
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// GetRootCmd 获取根命令实例（用于测试）
func GetRootCmd() *cobra.Command {
	return rootCmd
}
