package cmd

func init() {
	serverStartCmd.Flags().Int("port", 0, "服务器端口（默认 13618）")
	serverStopCmd.Flags().Bool("force", false, "强制停止服务器")
	serverLogsCmd.Flags().BoolP("follow", "f", false, "实时跟踪日志输出")
	serverLogsCmd.Flags().IntP("lines", "n", 50, "显示最后 N 行日志")

	serverCmd.AddCommand(serverStartCmd)
	serverCmd.AddCommand(serverStopCmd)
	serverCmd.AddCommand(serverStatusCmd)
	serverCmd.AddCommand(serverLogsCmd)
	serverCmd.AddCommand(serverRestartCmd)

	rootCmd.AddCommand(serverCmd)
}
