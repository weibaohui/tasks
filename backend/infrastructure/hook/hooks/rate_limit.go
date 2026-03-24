/**
 * 限流 Hook 实现
 */
package hooks

import (
	"errors"
	"time"

	"github.com/weibh/taskmanager/domain"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// ErrRateLimited 限流错误
var ErrRateLimited = errors.New("rate limit exceeded")

// RateLimitHook 限流
type RateLimitHook struct {
	*domain.BaseHook
	limiter *rate.Limiter
	burst   int
	logger  *zap.Logger
}

// NewRateLimitHook 创建限流 Hook
func NewRateLimitHook(limit rate.Limit, burst int, logger *zap.Logger) *RateLimitHook {
	return &RateLimitHook{
		BaseHook: domain.NewBaseHook("rate_limit", 5, domain.HookTypeLLM),
		limiter:  rate.NewLimiter(limit, burst),
		burst:    burst,
		logger:   logger,
	}
}

// PreLLMCall 检查限流
func (h *RateLimitHook) PreLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
	if !h.limiter.Allow() {
		h.logger.Warn("rate limit exceeded, LLM request blocked",
			zap.Int("burst", h.burst))
		return nil, ErrRateLimited
	}
	return callCtx, nil
}

// SetLimit 更新限流参数
func (h *RateLimitHook) SetLimit(limit rate.Limit) {
	h.limiter.SetLimit(limit)
}

// SetBurst 更新突发容量
func (h *RateLimitHook) SetBurst(burst int) {
	h.limiter.SetBurst(burst)
}

// Wait 等待获取令牌（用于背压）
func (h *RateLimitHook) Wait(ctx *domain.HookContext) error {
	return h.limiter.Wait(ctx)
}

// Reserve 预留令牌
func (h *RateLimitHook) Reserve() *rate.Reservation {
	return h.limiter.Reserve()
}

// WaitDuration 返回需要等待的时间
func (h *RateLimitHook) WaitDuration() time.Duration {
	return time.Duration(1e9 / int(h.limiter.Limit()))
}
