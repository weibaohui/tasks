/**
 * CLI 工具 - 任务管理 CLI
 */
package main

import (
	"github.com/weibh/taskmanager/cmd/cli/cmd"
)

func main() {
	cmd.AddCommands()
	cmd.Execute()
}