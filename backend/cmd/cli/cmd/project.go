package cmd

func registerProjectCommands() {
	// 注册子命令
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectGetCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectUpdateCmd)
	projectCmd.AddCommand(projectDeleteCmd)

	// 心跳子命令（在 project_heartbeat.go 中注册具体子命令）
	projectCmd.AddCommand(projectHeartbeatCmd)

	// 场景子命令（在 project_scenario.go 中注册具体子命令）
	projectCmd.AddCommand(projectScenarioCmd)

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

	// dispatch set 命令的 flags
	dispatchSetCmd.Flags().StringP("channel-code", "c", "", "渠道编码")
	dispatchSetCmd.Flags().StringP("session-key", "k", "", "会话密钥")

	// delete 命令的 flags
	projectDeleteCmd.Flags().BoolP("force", "f", false, "跳过确认直接删除")
}
