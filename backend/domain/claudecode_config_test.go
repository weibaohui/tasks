package domain

import (
	"encoding/json"
	"testing"
)

func TestDefaultClaudeCodeConfig_Timeout(t *testing.T) {
	config := DefaultClaudeCodeConfig()

	if config.Timeout != 120 {
		t.Errorf("期望Timeout为120, 实际为%d", config.Timeout)
	}
}

func TestClaudeCodeConfig_ToJSON_FromJSON_WithTimeout(t *testing.T) {
	tests := []struct {
		name   string
		config *ClaudeCodeConfig
	}{
		{
			name: "包含Timeout配置",
			config: &ClaudeCodeConfig{
				Model:             "claude-3-5-sonnet",
				MaxThinkingTokens: 8000,
				Timeout:           600,
				PermissionMode:    PermissionModeBypassPermissions,
			},
		},
		{
			name: "Timeout为0",
			config: &ClaudeCodeConfig{
				Model:  "claude-3-5-sonnet",
				Timeout: 0,
			},
		},
		{
			name: "Timeout为300",
			config: &ClaudeCodeConfig{
				Model:  "claude-3-5-sonnet",
				Timeout: 300,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 序列化
			jsonStr, err := tt.config.ToJSON()
			if err != nil {
				t.Fatalf("ToJSON失败: %v", err)
			}

			// 反序列化
			newConfig := &ClaudeCodeConfig{}
			err = newConfig.FromJSON(jsonStr)
			if err != nil {
				t.Fatalf("FromJSON失败: %v", err)
			}

			// 验证Timeout字段
			if newConfig.Timeout != tt.config.Timeout {
				t.Errorf("Timeout不匹配: 期望%d, 实际%d", tt.config.Timeout, newConfig.Timeout)
			}

			// 验证其他字段也正确
			if newConfig.Model != tt.config.Model {
				t.Errorf("Model不匹配: 期望%s, 实际%s", tt.config.Model, newConfig.Model)
			}
		})
	}
}

func TestClaudeCodeConfig_MergeWith_Timeout(t *testing.T) {
	tests := []struct {
		name          string
		base          *ClaudeCodeConfig
		other         *ClaudeCodeConfig
		expectedValue int
	}{
		{
			name:          "other Timeout大于0,应覆盖base",
			base:          &ClaudeCodeConfig{Timeout: 120},
			other:         &ClaudeCodeConfig{Timeout: 600},
			expectedValue: 600,
		},
		{
			name:          "other Timeout为0,应保留base",
			base:          &ClaudeCodeConfig{Timeout: 120},
			other:         &ClaudeCodeConfig{Timeout: 0},
			expectedValue: 120,
		},
		{
			name:          "base Timeout为0,other Timeout大于0,应使用other",
			base:          &ClaudeCodeConfig{Timeout: 0},
			other:         &ClaudeCodeConfig{Timeout: 600},
			expectedValue: 600,
		},
		{
			name:          "两者Timeout都为0",
			base:          &ClaudeCodeConfig{Timeout: 0},
			other:         &ClaudeCodeConfig{Timeout: 0},
			expectedValue: 0,
		},
		{
			name:          "other为nil,应保留base",
			base:          &ClaudeCodeConfig{Timeout: 120},
			other:         nil,
			expectedValue: 120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建副本避免修改原对象
			base := &ClaudeCodeConfig{}
			if tt.base != nil {
				*base = *tt.base
			}

			base.MergeWith(tt.other)

			if base.Timeout != tt.expectedValue {
				t.Errorf("期望Timeout为%d, 实际为%d", tt.expectedValue, base.Timeout)
			}
		})
	}
}

func TestClaudeCodeConfig_JSONSerialization_TimeoutField(t *testing.T) {
	config := &ClaudeCodeConfig{
		Model:  "test-model",
		Timeout: 600,
	}

	jsonStr, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("JSON Marshal失败: %v", err)
	}

	// 验证JSON中包含timeout字段
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(jsonStr, &jsonMap); err != nil {
		t.Fatalf("JSON Unmarshal失败: %v", err)
	}

	timeout, ok := jsonMap["timeout"]
	if !ok {
		t.Error("JSON中应包含timeout字段")
	}

	timeoutFloat, ok := timeout.(float64)
	if !ok {
		t.Fatalf("timeout应为float64, 实际为%T", timeout)
	}

	if int(timeoutFloat) != 600 {
		t.Errorf("timeout值不正确: 期望600, 实际%v", timeoutFloat)
	}
}

func TestClaudeCodeConfig_FromJSON_EmptyString(t *testing.T) {
	config := &ClaudeCodeConfig{Timeout: 600}

	err := config.FromJSON("")
	if err != nil {
		t.Errorf("空字符串不应返回错误: %v", err)
	}

	// Timeout应该保持原值不变
	if config.Timeout != 600 {
		t.Errorf("Timeout不应被修改: 期望600, 实际%d", config.Timeout)
	}
}

func TestClaudeCodeConfig_ToJSON_NilReceiver(t *testing.T) {
	var config *ClaudeCodeConfig

	jsonStr, err := config.ToJSON()
	if err != nil {
		t.Fatalf("nil ToJSON不应返回错误: %v", err)
	}

	if jsonStr != "" {
		t.Errorf("nil ToJSON应返回空字符串, 实际为%s", jsonStr)
	}
}