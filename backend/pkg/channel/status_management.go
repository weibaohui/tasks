package channel

import (
	"context"
	"time"
	"github.com/weibh/taskmanager/domain"
	"go.uber.org/zap"
	"strings"
	"github.com/weibh/taskmanager/application"
)

func (p *MessageProcessor) updateAgentRuntimeStatus(ctx context.Context, sessionKey, agentType, status, lastError string) {
	if p.sessionService == nil || strings.TrimSpace(sessionKey) == "" {
		return
	}

	metadata, err := p.sessionService.GetSessionMetadata(ctx, sessionKey)
	if err != nil {
		p.logger.Debug("更新 Agent 运行状态时读取会话失败",
			zap.String("session_key", sessionKey),
			zap.Error(err),
		)
		return
	}

	merged := map[string]interface{}{}
	for k, v := range metadata {
		merged[k] = v
	}

	runtime := map[string]interface{}{}
	if existingRuntime, ok := merged["agent_runtime"].(map[string]interface{}); ok {
		for k, v := range existingRuntime {
			runtime[k] = v
		}
	}

	now := time.Now().UnixMilli()
	runtime["status"] = status
	runtime["agent_type"] = agentType
	runtime["is_running"] = status == "running"
	runtime["updated_at"] = now
	if status == "running" {
		runtime["started_at"] = now
		runtime["ended_at"] = nil
		runtime["last_error"] = ""
	} else {
		runtime["ended_at"] = now
		runtime["last_error"] = lastError
	}
	merged["agent_runtime"] = runtime

	if err := p.sessionService.UpdateSessionMetadata(ctx, application.UpdateSessionMetadataCommand{
		SessionKey: sessionKey,
		Metadata:   merged,
	}); err != nil {
		p.logger.Warn("更新 Agent 运行状态失败",
			zap.String("session_key", sessionKey),
			zap.String("status", status),
			zap.Error(err),
		)
	}
}

// updateRequirementAgentRuntimeStatus 更新需求的 Agent Runtime 状态
func (p *MessageProcessor) updateRequirementAgentRuntimeStatus(ctx context.Context, requirementID string, agentType string, status string, lastError string) {
	if p.requirementRepo == nil || strings.TrimSpace(requirementID) == "" {
		return
	}

	req, err := p.requirementRepo.FindByID(ctx, domain.NewRequirementID(requirementID))
	if err != nil || req == nil {
		p.logger.Debug("更新需求 Agent Runtime 状态时查找需求失败",
			zap.String("requirement_id", requirementID),
			zap.Error(err),
		)
		return
	}

	switch status {
	case "running":
		req.StartAgentRuntime(agentType)
	case "completed":
		req.EndAgentRuntime(true, "")
	case "failed":
		req.EndAgentRuntime(false, lastError)
	default:
		p.logger.Warn("未知的 Agent Runtime 状态",
			zap.String("requirement_id", requirementID),
			zap.String("status", status),
		)
		return
	}

	if err := p.requirementRepo.Save(ctx, req); err != nil {
		p.logger.Warn("更新需求 Agent Runtime 状态失败",
			zap.String("requirement_id", requirementID),
			zap.String("status", status),
			zap.Error(err),
		)
	}
}

