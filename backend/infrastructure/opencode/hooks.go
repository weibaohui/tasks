package opencode

import (
	"context"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

func (p *OpenCodeProcessor) triggerOpenCodeFinishedHook(ctx context.Context, msg *bus.InboundMessage, agent *domain.Agent, success bool, finalResult string, capturedUsage *TokenUsage) {
	if msg.Metadata == nil {
		p.logger.Debug("OpenCode 完成，无 requirement_id 元数据")
		return
	}

	requirementIDStr, ok := msg.Metadata["requirement_id"].(string)
	if !ok || requirementIDStr == "" {
		p.logger.Debug("OpenCode 完成，requirement_id 为空或不存在",
			zap.Any("metadata", msg.Metadata))
		return
	}

	requirementID := domain.NewRequirementID(requirementIDStr)
	requirement, err := p.requirementRepo.FindByID(ctx, requirementID)
	if err != nil {
		p.logger.Error("OpenCode 完成，查找 requirement 失败",
			zap.String("requirement_id", requirementIDStr),
			zap.Error(err))
		return
	}
	if requirement == nil {
		p.logger.Warn("OpenCode 完成，requirement 不存在",
			zap.String("requirement_id", requirementIDStr))
		return
	}

	p.logger.Info("OpenCode 完成，触发 opencode_finished hook",
		zap.String("requirement_id", requirementIDStr),
		zap.String("requirement_title", requirement.Title()),
		zap.Bool("success", success))

	if p.replicaCleanupSvc != nil {
		_ = p.replicaCleanupSvc.CleanupReplica(ctx, requirement.ReplicaAgentCode(), requirement.WorkspacePath())
		requirement.SetReplicaAgentCode("")
		requirement.SetWorkspacePath("")
		if err := p.requirementRepo.Save(ctx, requirement); err != nil {
			p.logger.Error("OpenCode 完成，保存 requirement 失败",
				zap.String("requirement_id", requirementIDStr),
				zap.Error(err))
			return
		}
	}

	if success {
		requirement.MarkCompleted()
		requirement.SetClaudeRuntimeResult(finalResult)
		if err := p.requirementRepo.Save(ctx, requirement); err != nil {
			p.logger.Error("OpenCode 完成，保存 requirement 失败",
				zap.String("requirement_id", requirementIDStr),
				zap.Error(err))
			return
		}
		p.logger.Info("OpenCode 完成，requirement 已标记为 completed",
			zap.String("requirement_id", requirementIDStr),
			zap.Int("result_length", len(finalResult)))
	}

	if capturedUsage != nil {
		total := capturedUsage.Total
		if total == 0 {
			total = capturedUsage.Input + capturedUsage.Output
		}
		requirement.SetTokenUsage(capturedUsage.Input, capturedUsage.Output, total)
		if err := p.requirementRepo.Save(ctx, requirement); err != nil {
			p.logger.Warn("保存 token 使用量到需求表失败", zap.Error(err))
		} else {
			p.logger.Info("已保存 token 使用量",
				zap.String("requirement_id", requirementIDStr),
				zap.Int("prompt_tokens", capturedUsage.Input),
				zap.Int("completion_tokens", capturedUsage.Output),
				zap.Int("total_tokens", total),
			)
		}
	}
}
