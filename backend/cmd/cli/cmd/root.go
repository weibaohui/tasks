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
  taskmanager server start        启动服务
  taskmanager server stop         停止服务
  taskmanager server status       查看服务状态
  taskmanager server logs         查看服务日志

需求管理:
  taskmanager requirement create    创建新需求
  taskmanager requirement list      列出需求
  taskmanager requirement get       获取需求详情
  taskmanager requirement delete    删除需求
  taskmanager requirement dispatch 派发需求
  taskmanager requirement complete 完成需求
  taskmanager requirement reset    重置需求

状态机管理:
  taskmanager statemachine create      创建状态机
  taskmanager statemachine list        列出状态机
  taskmanager statemachine delete      删除状态机
  taskmanager statemachine transition  触发状态转换
  taskmanager statemachine state       获取需求状态
  taskmanager statemachine history     获取转换历史
  taskmanager statemachine bind        绑定需求类型

状态查询:
  taskmanager state summary <project-id>   项目状态统计
  taskmanager state history <req-id>      需求转换历史

Hook管理:
  taskmanager hook list          列出 Hook 配置

其他:
  taskmanager project list      列出项目
  taskmanager agent list       列出 Agent

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
