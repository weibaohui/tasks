/**
 * Bash 工具
 * 用于在服务器上执行 Bash 命令
 */
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/weibh/taskmanager/infrastructure/llm"
)

// BashTool Bash 命令执行工具
type BashTool struct{}

// NewBashTool 创建 Bash 工具
func NewBashTool() *BashTool {
	return &BashTool{}
}

var _ llm.Tool = (*BashTool)(nil)

// Name 返回工具名称
func (t *BashTool) Name() string {
	return "bash"
}

// Description 返回工具描述
func (t *BashTool) Description() string {
	return `执行 Bash 命令并返回输出结果。
用于在服务器上执行 shell 命令，如文件操作、进程管理等。
注意：此工具会阻塞直到命令执行完成，耗时操作请谨慎使用。`
}

// Parameters 返回参数 schema
func (t *BashTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "要执行的 Bash 命令"
			},
			"timeout": {
				"type": "number",
				"description": "超时时间（秒），默认 30 秒",
				"default": 30
			}
		},
		"required": ["command"]
	}`)
}

// Execute 执行命令
func (t *BashTool) Execute(ctx context.Context, input json.RawMessage) (*llm.ToolResult, error) {
	var args struct {
		Command string  `json:"command"`
		Timeout float64 `json:"timeout"`
	}

	if err := json.Unmarshal(input, &args); err != nil {
		return &llm.ToolResult{
			ID:    "",
			Output: "",
			Error:  fmt.Sprintf("解析参数失败: %v", err),
		}, nil
	}

	if args.Command == "" {
		return &llm.ToolResult{
			ID:    "",
			Output: "",
			Error:  "command 参数不能为空",
		}, nil
	}

	// 默认超时 30 秒
	timeout := time.Duration(args.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// 创建带超时的 context
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 执行命令
	cmd := exec.CommandContext(cmdCtx, "bash", "-c", args.Command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += "STDERR: " + stderr.String()
	}

	if ctx.Err() == context.DeadlineExceeded {
		return &llm.ToolResult{
			ID:    "",
			Output: output,
			Error:  fmt.Sprintf("命令执行超时 (%.0f 秒)", args.Timeout),
		}, nil
	}

	if err != nil {
		if output != "" {
			output += "\n"
		}
		output += fmt.Sprintf("命令执行失败: %v", err)
	}

	return &llm.ToolResult{
		ID:     "",
		Output: output,
		Error:  "",
	}, nil
}
