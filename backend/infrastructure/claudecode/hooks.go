package claudecode

import (
	"context"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

func (p *ClaudeCodeProcessor) triggerClaudeCodeFinishedHook(ctx context.Context, msg *bus.InboundMessage, agent *domain.Agent, success bool, finalResult string, capturedUsage *domain.Usage) {
	// 检查是否有 requirement_id 元数据
	if msg.Metadata == nil {
		p.logger.Debug("Claude Code 完成，无 requirement_id 元数据")
		return
	}

	requirementIDStr, ok := msg.Metadata["requirement_id"].(string)
	if !ok || requirementIDStr == "" {
		p.logger.Debug("Claude Code 完成，requirement_id 为空或不存在",
			zap.Any("metadata", msg.Metadata))
		return
	}

	// 查找 requirement
	requirementID := domain.NewRequirementID(requirementIDStr)
	requirement, err := p.requirementRepo.FindByID(ctx, requirementID)
	if err != nil {
		p.logger.Error("Claude Code 完成，查找 requirement 失败",
			zap.String("requirement_id", requirementIDStr),
			zap.Error(err))
		return
	}
	if requirement == nil {
		p.logger.Warn("Claude Code 完成，requirement 不存在",
			zap.String("requirement_id", requirementIDStr))
		return
	}

	p.logger.Info("Claude Code 完成，触发 claude_code_finished hook",
		zap.String("requirement_id", requirementIDStr),
		zap.String("requirement_title", requirement.Title()),
		zap.Bool("success", success))

	// **立即清理分身**（代码约束，不是 Hook）
	// 在触发任何 hook 之前清理分身，确保清理一定会执行
	if p.replicaCleanupSvc != nil {
		_ = p.replicaCleanupSvc.CleanupReplica(ctx, requirement.ReplicaAgentCode(), requirement.WorkspacePath())
		requirement.SetReplicaAgentCode("")
		requirement.SetWorkspacePath("")
		if err := p.requirementRepo.Save(ctx, requirement); err != nil {
			p.logger.Error("Claude Code 完成，保存 requirement 失败",
				zap.String("requirement_id", requirementIDStr),
				zap.Error(err))
			return
		}
	}

	// 成功完成时，标记需求为 completed 状态并保存执行结果
	if success {
		requirement.MarkCompleted()
		requirement.SetClaudeRuntimeResult(finalResult)
		if err := p.requirementRepo.Save(ctx, requirement); err != nil {
			p.logger.Error("Claude Code 完成，保存 requirement 失败",
				zap.String("requirement_id", requirementIDStr),
				zap.Error(err))
			return
		}
		p.logger.Info("Claude Code 完成，requirement 已标记为 completed",
			zap.String("requirement_id", requirementIDStr),
			zap.Any("result_length", len(finalResult)))
	}

	// 使用 capturedUsage（从 ResultMessage 直接捕获的 token usage）
	traceID := requirement.TraceID()
	if capturedUsage != nil {
		// 计算 total：如果 TotalTokens 为 0，用 prompt + completion 作为 total
		total := capturedUsage.TotalTokens
		if total == 0 {
			total = capturedUsage.PromptTokens + capturedUsage.CompletionTokens
		}
		requirement.SetTokenUsage(capturedUsage.PromptTokens, capturedUsage.CompletionTokens, total)
		if err := p.requirementRepo.Save(ctx, requirement); err != nil {
			p.logger.Warn("保存 token 使用量到需求表失败", zap.Error(err))
		} else {
			p.logger.Info("已保存 token 使用量",
				zap.String("requirement_id", requirementIDStr),
				zap.String("trace_id", traceID),
				zap.Int("prompt_tokens", capturedUsage.PromptTokens),
				zap.Int("completion_tokens", capturedUsage.CompletionTokens),
				zap.Int("total_tokens", total),
			)
		}
	}
}
