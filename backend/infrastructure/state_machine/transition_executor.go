package state_machine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/weibh/taskmanager/domain/state_machine"
	"go.uber.org/zap"
)

// TransitionExecutor 转换钩子执行器
type TransitionExecutor struct {
	logger     *zap.Logger
	httpClient *http.Client
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

// ExecuteHooks 异步执行 hooks
func (e *TransitionExecutor) ExecuteHooks(ctx context.Context, hooks []state_machine.TransitionHook, requirementID string) {
	go func() {
		for _, hook := range hooks {
			e.executeHook(ctx, hook, requirementID)
		}
	}()
}

func (e *TransitionExecutor) executeHook(ctx context.Context, hook state_machine.TransitionHook, requirementID string) {
	logger := e.logger.With(zap.String("hook", hook.Name), zap.String("requirement_id", requirementID))

	// 构建请求
	body, err := json.Marshal(map[string]interface{}{
		"requirement_id": requirementID,
		"hook_name":      hook.Name,
		"hook_type":     hook.Type,
	})
	if err != nil {
		logger.Error("failed to marshal hook request", zap.Error(err))
		return
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

		err := e.executeWebhook(ctx, hook, body)
		if err == nil {
			logger.Info("hook executed successfully")
			return
		}
		lastErr = err
		logger.Warn("hook execution failed", zap.Error(err))
	}

	// 所有重试都失败了
	logger.Error("hook execution failed after all retries", zap.Error(lastErr))

	// TODO: 执行补偿（预留接口）
	e.executeCompensation(ctx, hook, requirementID, lastErr)
}

func (e *TransitionExecutor) executeWebhook(ctx context.Context, hook state_machine.TransitionHook, body []byte) error {
	// 获取 URL
	url, ok := hook.Config["url"].(string)
	if !ok {
		return fmt.Errorf("url not found in hook config")
	}

	method := "POST"
	if m, ok := hook.Config["method"].(string); ok {
		method = m
	}

	timeout := 30 * time.Second
	if t, ok := hook.Config["timeout"].(int); ok {
		timeout = time.Duration(t) * time.Second
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

func (e *TransitionExecutor) executeCompensation(ctx context.Context, hook state_machine.TransitionHook, requirementID string, err error) {
	// 预留接口，当前只记录日志
	e.logger.Warn("compensation triggered (not implemented)",
		zap.String("hook", hook.Name),
		zap.String("requirement_id", requirementID),
		zap.Error(err))
}
