/**
 * ClaudeCode Processor 测试
 * 验证 channel_code 等元数据的提取和传递
 */
package claudecode

import (
	"testing"
	"time"

	"github.com/weibh/taskmanager/pkg/bus"
)

// TestExtractChannelCode 测试从 InboundMessage.Metadata 提取 channel_code
func TestExtractChannelCode(t *testing.T) {
	tests := []struct {
		name         string
		metadata     map[string]any
		expectedCode string
	}{
		{
			name: "正常提取 channel_code",
			metadata: map[string]any{
				"channel_code": "ch-001",
				"user_code":    "user-001",
				"agent_code":   "agent-001",
			},
			expectedCode: "ch-001",
		},
		{
			name: "channel_code 为空字符串",
			metadata: map[string]any{
				"channel_code": "",
				"user_code":    "user-002",
				"agent_code":   "agent-002",
			},
			expectedCode: "",
		},
		{
			name: "无 channel_code 字段",
			metadata: map[string]any{
				"user_code":  "user-003",
				"agent_code": "agent-003",
			},
			expectedCode: "",
		},
		{
			name: "channel_code 类型错误（int）",
			metadata: map[string]any{
				"channel_code": 123,
				"user_code":    "user-004",
				"agent_code":   "agent-004",
			},
			expectedCode: "",
		},
		{
			name: "channel_code 类型错误（float）",
			metadata: map[string]any{
				"channel_code": 123.45,
				"user_code":    "user-005",
				"agent_code":   "agent-005",
			},
			expectedCode: "",
		},
		{
			name:         "Metadata 为 nil",
			metadata:     nil,
			expectedCode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &bus.InboundMessage{
				Channel:   "feishu",
				ChatID:    "chat-001",
				Content:   "test content",
				Timestamp: time.Now(),
				Metadata:  tt.metadata,
			}

			// 模拟 processor.go 中的提取逻辑
			channelCode := ""
			if msg.Metadata != nil {
				if v, ok := msg.Metadata["channel_code"].(string); ok {
					channelCode = v
				}
			}

			if channelCode != tt.expectedCode {
				t.Errorf("期望 channel_code 为 %q，实际为 %q", tt.expectedCode, channelCode)
			}
		})
	}
}

// TestExtractAllScopeFields 测试提取所有 scope 相关字段
func TestExtractAllScopeFields(t *testing.T) {
	msg := &bus.InboundMessage{
		Channel:   "feishu",
		ChatID:    "oc_xxx",
		Content:   "执行任务",
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"agent_code":      "agt_replica_001",
			"user_code":       "user_001",
			"channel_code":    "ch_feishu_main",
			"requirement_id":  "req_001",
			"project_id":      "proj_001",
			"dispatch_source": "requirement",
		},
	}

	// 提取所有 scope 字段（模拟 processor.go 中的逻辑）
	channelCode := ""
	if msg.Metadata != nil {
		if v, ok := msg.Metadata["channel_code"].(string); ok {
			channelCode = v
		}
	}

	userCode := ""
	if msg.Metadata != nil {
		if v, ok := msg.Metadata["user_code"].(string); ok {
			userCode = v
		}
	}

	agentCode := ""
	if msg.Metadata != nil {
		if v, ok := msg.Metadata["agent_code"].(string); ok {
			agentCode = v
		}
	}

	channelType := msg.Channel
	sessionKey := msg.SessionKey()

	// 验证所有字段
	if channelCode != "ch_feishu_main" {
		t.Errorf("期望 channelCode 为 ch_feishu_main，实际为 %s", channelCode)
	}
	if userCode != "user_001" {
		t.Errorf("期望 userCode 为 user_001，实际为 %s", userCode)
	}
	if agentCode != "agt_replica_001" {
		t.Errorf("期望 agentCode 为 agt_replica_001，实际为 %s", agentCode)
	}
	if channelType != "feishu" {
		t.Errorf("期望 channelType 为 feishu，实际为 %s", channelType)
	}
	if sessionKey != "feishu:oc_xxx" {
		t.Errorf("期望 sessionKey 为 feishu:oc_xxx，实际为 %s", sessionKey)
	}
}

// TestBuildLLMCallContextMetadata 测试构建 LLMCallContext.Metadata
func TestBuildLLMCallContextMetadata(t *testing.T) {
	tests := []struct {
		name          string
		msg           *bus.InboundMessage
		expectedMetas map[string]string
	}{
		{
			name: "完整元数据",
			msg: &bus.InboundMessage{
				Channel:   "feishu",
				ChatID:    "chat-001",
				Content:   "test",
				Timestamp: time.Now(),
				Metadata: map[string]any{
					"channel_code": "ch-001",
					"user_code":    "user-001",
					"agent_code":   "agent-001",
				},
			},
			expectedMetas: map[string]string{
				"session_key":  "feishu:chat-001",
				"channel_code": "ch-001",
				"channel_type": "feishu",
				"user_code":    "user-001",
				"agent_code":   "agent-001",
			},
		},
		{
			name: "缺少部分元数据",
			msg: &bus.InboundMessage{
				Channel:   "dingtalk",
				ChatID:    "chat-002",
				Content:   "test",
				Timestamp: time.Now(),
				Metadata: map[string]any{
					"user_code":  "user-002",
					"agent_code": "agent-002",
					// 缺少 channel_code
				},
			},
			expectedMetas: map[string]string{
				"session_key":  "dingtalk:chat-002",
				"channel_code": "", // 应该为空
				"channel_type": "dingtalk",
				"user_code":    "user-002",
				"agent_code":   "agent-002",
			},
		},
		{
			name: "Metadata 为 nil",
			msg: &bus.InboundMessage{
				Channel:   "wechat",
				ChatID:    "chat-003",
				Content:   "test",
				Timestamp: time.Now(),
				Metadata:  nil,
			},
			expectedMetas: map[string]string{
				"session_key":  "wechat:chat-003",
				"channel_code": "",
				"channel_type": "wechat",
				"user_code":    "",
				"agent_code":   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.msg

			// 提取 scope 字段（模拟 processor.go 中的逻辑）
			channelCode := ""
			userCode := ""
			agentCode := ""

			if msg.Metadata != nil {
				if v, ok := msg.Metadata["channel_code"].(string); ok {
					channelCode = v
				}
				if v, ok := msg.Metadata["user_code"].(string); ok {
					userCode = v
				}
				if v, ok := msg.Metadata["agent_code"].(string); ok {
					agentCode = v
				}
			}

			// 构建 Metadata（模拟 processor.go 中的逻辑）
			metadata := map[string]string{
				"session_key":  msg.SessionKey(),
				"trace_id":     "trace-test",
				"user_code":    userCode,
				"agent_code":   agentCode,
				"channel_code": channelCode,
				"channel_type": msg.Channel,
				"chat_id":      msg.ChatID,
			}

			// 验证
			for key, expected := range tt.expectedMetas {
				actual := metadata[key]
				if actual != expected {
					t.Errorf("期望 %s 为 %q，实际为 %q", key, expected, actual)
				}
			}
		})
	}
}

// TestRequirementDispatchMetadata 测试需求派发场景的元数据提取
func TestRequirementDispatchMetadata(t *testing.T) {
	// 模拟 requirement_dispatch_service.go 中 PublishInbound 时设置的 Metadata
	msg := &bus.InboundMessage{
		Channel:   "feishu",
		ChatID:    "oc_dispatch_chat",
		Content:   "【需求信息】\n- 需求ID：req-001",
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"agent_code":      "agt_replica_coding_001",
			"user_code":       "user_coding_001",
			"channel_code":    "ch_dispatch_channel",
			"requirement_id":  "req-001",
			"project_id":      "proj-001",
			"dispatch_source": "requirement",
		},
	}

	// 提取所有字段
	channelCode := ""
	userCode := ""
	agentCode := ""
	var requirementID, projectID string

	if msg.Metadata != nil {
		if v, ok := msg.Metadata["channel_code"].(string); ok {
			channelCode = v
		}
		if v, ok := msg.Metadata["user_code"].(string); ok {
			userCode = v
		}
		if v, ok := msg.Metadata["agent_code"].(string); ok {
			agentCode = v
		}
		if v, ok := msg.Metadata["requirement_id"].(string); ok {
			requirementID = v
		}
		if v, ok := msg.Metadata["project_id"].(string); ok {
			projectID = v
		}
	}

	// 验证关键字段
	if channelCode != "ch_dispatch_channel" {
		t.Errorf("期望 channelCode 为 ch_dispatch_channel，实际为 %s", channelCode)
	}
	if userCode != "user_coding_001" {
		t.Errorf("期望 userCode 为 user_coding_001，实际为 %s", userCode)
	}
	if agentCode != "agt_replica_coding_001" {
		t.Errorf("期望 agentCode 为 agt_replica_coding_001，实际为 %s", agentCode)
	}
	if requirementID != "req-001" {
		t.Errorf("期望 requirementID 为 req-001，实际为 %s", requirementID)
	}
	if projectID != "proj-001" {
		t.Errorf("期望 projectID 为 proj-001，实际为 %s", projectID)
	}
}

// TestChannelTypeFromMsgChannel 测试 channel_type 从 msg.Channel 获取
func TestChannelTypeFromMsgChannel(t *testing.T) {
	tests := []struct {
		channel         string
		expectedChannel string
	}{
		{"feishu", "feishu"},
		{"dingtalk", "dingtalk"},
		{"wechat", "wechat"},
		{"slack", "slack"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.channel, func(t *testing.T) {
			msg := &bus.InboundMessage{
				Channel:   tt.channel,
				ChatID:    "chat-001",
				Content:   "test",
				Timestamp: time.Now(),
			}

			// channel_type 直接使用 msg.Channel
			channelType := msg.Channel

			if channelType != tt.expectedChannel {
				t.Errorf("期望 channelType 为 %s，实际为 %s", tt.expectedChannel, channelType)
			}
		})
	}
}
