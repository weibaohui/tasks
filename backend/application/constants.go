/**
 * 应用层常量定义
 */
package application

import "time"

// 任务执行相关常量
const (
	// DefaultTaskTimeout 默认任务超时时间
	DefaultTaskTimeout = 60 * time.Second

	// DefaultSubTaskDelay 子任务延迟时间
	DefaultSubTaskDelay = 1 * time.Second

	// AgentInitDelay Agent 初始化延迟
	AgentInitDelay = 2 * time.Second
)
