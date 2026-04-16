package claudecode

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/trace"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

func (p *ClaudeCodeProcessor) Process(ctx context.Context, msg *bus.InboundMessage, session *ClaudeCodeSession, agent *domain.Agent) (string, error) {
	// 优先从 context 获取已有的 trace_id，如果不存在则生成新的
	traceID := trace.MustGetTraceID(ctx)
	spanID := trace.MustGetSpanID(ctx)
	if traceID == "" {
		var newCtx context.Context
		newCtx, traceID, spanID = trace.StartTrace(ctx)
		ctx = newCtx
	}
	_ = spanID // 未使用

	preview := msg.Content
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	agentCode := ""
	if agent != nil {
		agentCode = agent.AgentCode().String()
	}
	p.logger.Info("ClaudeCode 处理消息",
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("session_key", msg.SessionKey()),
		zap.String("agent_code", agentCode),
		zap.String("内容", preview),
	)

	// 保存 trace_id 到需求表
	if msg.Metadata != nil {
		if requirementIDStr, ok := msg.Metadata["requirement_id"].(string); ok && requirementIDStr != "" {
			requirementID := domain.NewRequirementID(requirementIDStr)
			requirement, err := p.requirementRepo.FindByID(ctx, requirementID)
			if err != nil {
				p.logger.Warn("查找需求失败，无法保存 trace_id", zap.Error(err))
			} else if requirement != nil {
				requirement.SetTraceID(traceID)
				if err := p.requirementRepo.Save(ctx, requirement); err != nil {
					p.logger.Warn("保存 trace_id 到需求表失败", zap.Error(err))
				} else {
					p.logger.Info("已保存 trace_id 到需求表",
						zap.String("requirement_id", requirementIDStr),
						zap.String("trace_id", traceID),
					)
				}
			}
		}
	}

	// 获取 Provider 配置（优先使用 Agent 指定的 Provider）
	provider, err := p.resolveProvider(ctx, agent)
	if err != nil {
		p.logger.Warn("获取 LLM Provider 失败，使用默认配置", zap.Error(err))
		provider = nil
	}

	// 获取超时配置
	timeout := 180 * 60 // 180分钟
	if agent != nil && agent.ClaudeCodeConfig() != nil && agent.ClaudeCodeConfig().Timeout > 0 {
		timeout = agent.ClaudeCodeConfig().Timeout
	}

	// 调用 Claude Code（使用独立的 context，避免被取消）
	queryCtx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// 从会话获取 CLI Session UUID，用于会话恢复
	cliSessionID := ""
	if session != nil {
		cliSessionID = session.CliSessionID
	}

	// 调用 Claude Code
	response, newCliSessionID, err := p.queryClaudeCode(queryCtx, msg, cliSessionID, traceID, provider, agent)
	if err != nil {
		p.logger.Error("Claude Code 调用失败", zap.Error(err))
		return "", fmt.Errorf("Claude Code 调用失败: %w", err)
	}

	// 如果返回了新的 CLI Session ID，更新会话
	if newCliSessionID != "" && session != nil {
		session.CliSessionID = newCliSessionID
		p.logger.Info("Claude Code 会话已保存",
			zap.String("session_key", msg.SessionKey()),
			zap.String("cli_session_id", newCliSessionID),
		)
	}

	p.logger.Info("Claude Code 返回响应",
		zap.String("session_key", msg.SessionKey()),
		zap.String("response_length", fmt.Sprintf("%d", len(response))),
		zap.String("response_preview", func() string {
			if len(response) > 100 {
				return response[:100] + "..."
			}
			return response
		}()),
	)

	return response, nil
}

// ProcessWithStreaming 处理 CodingAgent 消息，带流式回调
func (p *ClaudeCodeProcessor) ProcessWithStreaming(ctx context.Context, msg *bus.InboundMessage, session *ClaudeCodeSession, agent *domain.Agent, callback StreamingCallback) error {
	// 优先从 context 获取已有的 trace_id，如果不存在则生成新的
	traceID := trace.MustGetTraceID(ctx)
	spanID := trace.MustGetSpanID(ctx)
	if traceID == "" {
		var newCtx context.Context
		newCtx, traceID, spanID = trace.StartTrace(ctx)
		ctx = newCtx
	}
	_ = spanID // 未使用

	preview := msg.Content
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	agentCode := ""
	if agent != nil {
		agentCode = agent.AgentCode().String()
	}
	p.logger.Info("ClaudeCode 流式处理消息",
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("session_key", msg.SessionKey()),
		zap.String("agent_code", agentCode),
		zap.String("内容", preview),
	)

	// 保存 trace_id 到需求表
	if msg.Metadata != nil {
		if requirementIDStr, ok := msg.Metadata["requirement_id"].(string); ok && requirementIDStr != "" {
			requirementID := domain.NewRequirementID(requirementIDStr)
			requirement, err := p.requirementRepo.FindByID(ctx, requirementID)
			if err != nil {
				p.logger.Warn("查找需求失败，无法保存 trace_id", zap.Error(err))
			} else if requirement != nil {
				requirement.SetTraceID(traceID)
				if err := p.requirementRepo.Save(ctx, requirement); err != nil {
					p.logger.Warn("保存 trace_id 到需求表失败", zap.Error(err))
				} else {
					p.logger.Info("已保存 trace_id 到需求表",
						zap.String("requirement_id", requirementIDStr),
						zap.String("trace_id", traceID),
					)
				}
			}
		}
	}

	// 获取 Provider 配置（优先使用 Agent 指定的 Provider）
	provider, err := p.resolveProvider(ctx, agent)
	if err != nil {
		p.logger.Warn("获取 LLM Provider 失败，使用默认配置", zap.Error(err))
		provider = nil
	}

	// 获取超时配置
	timeout := 180 * 60 // 180分钟
	if agent != nil && agent.ClaudeCodeConfig() != nil && agent.ClaudeCodeConfig().Timeout > 0 {
		timeout = agent.ClaudeCodeConfig().Timeout
	}

	// 调用 Claude Code（使用独立的 context）
	queryCtx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	cliSessionID := ""
	if session != nil {
		cliSessionID = session.CliSessionID
	}

	if callback != nil {
		callback.OnStart()
	}

	// 流式调用 Claude Code
	newCliSessionID, capturedUsage, err := p.queryClaudeCodeStreaming(queryCtx, msg, msg.Content, cliSessionID, traceID, provider, agent, callback)
	if err != nil {
		p.logger.Error("Claude Code 流式调用失败", zap.Error(err))
		if callback != nil {
			callback.OnError(err)
		}
		// 即使出错也要触发清理 hook
		p.triggerClaudeCodeFinishedHook(ctx, msg, agent, false, "", nil)
		return fmt.Errorf("Claude Code 调用失败: %w", err)
	}

	// 如果返回了新的 CLI Session ID，更新会话
	if newCliSessionID != "" && session != nil {
		session.CliSessionID = newCliSessionID
		p.logger.Info("Claude Code 会话已保存",
			zap.String("session_key", msg.SessionKey()),
			zap.String("cli_session_id", newCliSessionID),
		)
	}

	// 获取最终结果
	finalResult := ""
	if callback != nil {
		finalResult = callback.GetFinalResult()
	}

	// 检测常见的 Claude Code CLI 失败模式（如未登录）
	if strings.Contains(finalResult, "Not logged in") {
		loginErr := fmt.Errorf("Claude Code 未登录，请先在终端运行 `claude login` 进行认证")
		p.logger.Error("Claude Code 未登录，无法执行", zap.String("final_result", finalResult))
		if callback != nil {
			callback.OnError(loginErr)
		}
		p.triggerClaudeCodeFinishedHook(ctx, msg, agent, false, finalResult, capturedUsage)
		return loginErr
	}

	// Claude Code 执行完成，触发 claude_code_finished hook
	p.triggerClaudeCodeFinishedHook(ctx, msg, agent, true, finalResult, capturedUsage)

	return nil
}

// queryClaudeCodeStreaming 流式调用 Claude Code SDK
// 返回 sessionID, 捕获的 token usage, error
