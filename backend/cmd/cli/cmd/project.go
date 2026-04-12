package cmd

func registerProjectCommands() {
	// 注册子命令
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectGetCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectUpdateCmd)
	projectCmd.AddCommand(projectDeleteCmd)

	// 心跳子命令
	projectCmd.AddCommand(projectHeartbeatCmd)
	projectHeartbeatCmd.AddCommand(heartbeatStatusCmd)
	projectHeartbeatCmd.AddCommand(heartbeatEnableCmd)
	projectHeartbeatCmd.AddCommand(heartbeatDisableCmd)
	projectHeartbeatCmd.AddCommand(heartbeatSetIntervalCmd)
	projectHeartbeatCmd.AddCommand(heartbeatSetMdCmd)
	projectHeartbeatCmd.AddCommand(heartbeatSetAgentCmd)

	// 派发子命令
	projectCmd.AddCommand(projectDispatchCmd)
	projectDispatchCmd.AddCommand(dispatchGetCmd)
	projectDispatchCmd.AddCommand(dispatchSetCmd)
	projectDispatchCmd.AddCommand(dispatchClearCmd)

	// create 命令的 flags
	projectCreateCmd.Flags().StringP("name", "n", "", "项目名称 (必填)")
	projectCreateCmd.Flags().StringP("git-repo-url", "r", "", "Git仓库地址")
	projectCreateCmd.Flags().StringP("default-branch", "b", "main", "默认分支")
	projectCreateCmd.Flags().StringP("init-steps", "s", "", "初始化步骤 (每行一个)")

	// update 命令的 flags
	projectUpdateCmd.Flags().StringP("name", "n", "", "项目名称")
	projectUpdateCmd.Flags().StringP("git-repo-url", "r", "", "Git仓库地址")
	projectUpdateCmd.Flags().StringP("default-branch", "b", "", "默认分支")
	projectUpdateCmd.Flags().StringP("init-steps", "s", "", "初始化步骤 (每行一个)")

	// heartbeat enable 命令的 flags
	heartbeatEnableCmd.Flags().IntP("interval", "i", 0, "心跳间隔 (分钟)")
	heartbeatEnableCmd.Flags().StringP("md-content", "m", "", "心跳模板内容 (Markdown)")

	// heartbeat set-template 命令的 flags
	heartbeatSetMdCmd.Flags().StringP("content", "c", "", "心跳模板内容 (Markdown, 必填)")

	// heartbeat set-agent 命令的 flags
	heartbeatSetAgentCmd.Flags().StringP("agent-code", "a", "", "Agent编码 (必填)")

	// dispatch set 命令的 flags
	dispatchSetCmd.Flags().StringP("channel-code", "c", "", "渠道编码")
	dispatchSetCmd.Flags().StringP("session-key", "k", "", "会话密钥")

	// delete 命令的 flags
	projectDeleteCmd.Flags().BoolP("force", "f", false, "跳过确认直接删除")
}
