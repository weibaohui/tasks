package statemachine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/weibh/taskmanager/domain/statemachine"
	"go.uber.org/zap"
)

// TransitionExecutor 转换钩子执行器
type TransitionExecutor struct {
	logger           *zap.Logger
	httpClient       *http.Client
	heartbeatTrigger interface {
		Trigger(ctx context.Context, heartbeatID string) error
	}
}

// NewTransitionExecutor 创建执行器
func NewTransitionExecutor(logger *zap.Logger) *TransitionExecutor {
	return &TransitionExecutor{
		logger: logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetHeartbeatTrigger 设置心跳触发器
func (e *TransitionExecutor) SetHeartbeatTrigger(trigger interface {
	Trigger(ctx context.Context, heartbeatID string) error
}) {
	e.heartbeatTrigger = trigger
}

// ExecuteHooks 异步执行 hooks
func (e *TransitionExecutor) ExecuteHooks(ctx context.Context, hooks []statemachine.TransitionHook, hookCtx statemachine.HookContext) {
	go func() {
		for _, hook := range hooks {
			e.executeHook(ctx, hook, hookCtx)
		}
	}()
}

func (e *TransitionExecutor) executeHook(ctx context.Context, hook statemachine.TransitionHook, hookCtx statemachine.HookContext) {
	logger := e.logger.With(
		zap.String("hook", hook.Name),
		zap.String("requirement_id", hookCtx.RequirementID),
	)

	// 构建执行上下文（用于模板替换）
	execCtx := map[string]interface{}{
		"requirement_id":    hookCtx.RequirementID,
		"state_machine_id":  hookCtx.StateMachineID,
		"from_state":        hookCtx.FromState,
		"to_state":          hookCtx.ToState,
		"trigger":           hookCtx.Trigger,
		"hook_name":         hook.Name,
		"hook_type":         hook.Type,
	}

	// 合并 metadata 到执行上下文
	for k, v := range hookCtx.Metadata {
		execCtx[k] = v
	}

	// 执行重试
	maxRetries := hook.Retry
	if maxRetries < 0 {
		maxRetries = 0
	}

	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			logger.Info("retrying hook", zap.Int("attempt", i))
			time.Sleep(time.Second) // 简单重试延迟
		}

		var err error
		switch hook.Type {
		case "webhook":
			err = e.executeWebhook(ctx, hook, execCtx)
		case "command":
			err = e.executeCommand(ctx, hook, execCtx)
		case "trigger_heartbeat":
			err = e.executeTriggerHeartbeat(ctx, hook, execCtx)
		default:
			logger.Warn("unknown hook type, treating as webhook", zap.String("type", hook.Type))
			err = e.executeWebhook(ctx, hook, execCtx)
		}

		if err == nil {
			logger.Info("hook executed successfully")
			return
		}
		lastErr = err
		logger.Warn("hook execution failed", zap.Error(err))
	}

	// 所有重试都失败了
	logger.Error("hook execution failed after all retries", zap.Error(lastErr))

	// 触发补偿（当前仅记录日志，待后续扩展）
	e.executeCompensation(ctx, hook, hookCtx.RequirementID, lastErr)
}

func (e *TransitionExecutor) executeWebhook(ctx context.Context, hook statemachine.TransitionHook, hookCtx map[string]interface{}) error {
	// 获取 URL
	url, ok := hook.Config["url"].(string)
	if !ok {
		return fmt.Errorf("url not found in hook config")
	}

	// 替换模板变量
	url = e.interpolate(url, hookCtx)

	method := "POST"
	if m, ok := hook.Config["method"].(string); ok {
		method = m
	}

	timeout := 30 * time.Second
	if t, ok := hook.Config["timeout"].(int); ok {
		timeout = time.Duration(t) * time.Second
	}

	// 构建请求体
	body, err := json.Marshal(hookCtx)
	if err != nil {
		return fmt.Errorf("failed to marshal hook request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}

	return nil
}

// executeCommand 执行二进制命令
func (e *TransitionExecutor) executeCommand(ctx context.Context, hook statemachine.TransitionHook, hookCtx map[string]interface{}) error {
	// 获取命令
	cmdStr, ok := hook.Config["command"].(string)
	if !ok {
		return fmt.Errorf("command not found in hook config")
	}

	// 替换模板变量
	cmdStr = e.interpolate(cmdStr, hookCtx)

	// 解析命令和参数
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := parts[0]
	args := parts[1:]

	// 获取超时配置
	timeout := 60 * time.Second
	if t, ok := hook.Config["timeout"].(int); ok {
		timeout = time.Duration(t) * time.Second
	}

	// 创建带超时的上下文
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 执行命令
	execCmd := exec.CommandContext(cmdCtx, cmd, args...)
	execCmd.Env = []string{} // 可扩展：添加环境变量

	output, err := execCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %w, output: %s", err, string(output))
	}

	e.logger.Info("command executed successfully",
		zap.String("command", cmdStr),
		zap.String("output", string(output)))
	return nil
}

func (e *TransitionExecutor) executeTriggerHeartbeat(ctx context.Context, hook statemachine.TransitionHook, hookCtx map[string]interface{}) error {
	if e.heartbeatTrigger == nil {
		return fmt.Errorf("heartbeat trigger not configured")
	}

	hbIDRaw, ok := hook.Config["heartbeat_id"]
	if !ok {
		return fmt.Errorf("heartbeat_id not found in hook config")
	}

	hbID, ok := hbIDRaw.(string)
	if !ok {
		return fmt.Errorf("heartbeat_id must be a string")
	}

	hbID = e.interpolate(hbID, hookCtx)
	return e.heartbeatTrigger.Trigger(ctx, hbID)
}

// interpolate 替换模板变量
func (e *TransitionExecutor) interpolate(s string, ctx map[string]interface{}) string {
	result := s
	for key, value := range ctx {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

func (e *TransitionExecutor) executeCompensation(ctx context.Context, hook statemachine.TransitionHook, requirementID string, err error) {
	// 预留接口，当前只记录日志
	e.logger.Warn("compensation triggered (not implemented)",
		zap.String("hook", hook.Name),
		zap.String("requirement_id", requirementID),
		zap.Error(err))
}
