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

需求类型管理:
  taskmanager requirement-type list   列出需求类型
  taskmanager requirement-type create 创建需求类型

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

Agent管理:
  taskmanager agent list         列出 Agent
  taskmanager agent create       创建 Agent
  taskmanager agent update       更新 Agent
  taskmanager agent delete       删除 Agent
  taskmanager agent enable       启用 Agent
  taskmanager agent disable      禁用 Agent

渠道管理:
  taskmanager channel list       列出渠道
  taskmanager channel create     创建渠道
  taskmanager channel update     更新渠道
  taskmanager channel delete     删除渠道

Provider管理:
  taskmanager provider list      列出 Provider
  taskmanager provider create    创建 Provider
  taskmanager provider update    更新 Provider
  taskmanager provider delete    删除 Provider
  taskmanager provider test      测试 Provider 连接

MCP管理:
  taskmanager mcp list           列出 MCP 服务器
  taskmanager mcp create         创建 MCP 服务器
  taskmanager mcp update         更新 MCP 服务器
  taskmanager mcp delete         删除 MCP 服务器
  taskmanager mcp test           测试 MCP 服务器连接
  taskmanager mcp refresh        刷新 MCP 服务器工具

会话管理:
  taskmanager session list      列出会话
  taskmanager session delete     删除会话

用户管理:
  taskmanager user list          列出用户
  taskmanager user create        创建用户
  taskmanager user update        更新用户
  taskmanager user delete        删除用户

技能管理:
  taskmanager skill list         列出技能
  taskmanager skill get          获取技能详情

其他:
  taskmanager project list      列出项目

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
