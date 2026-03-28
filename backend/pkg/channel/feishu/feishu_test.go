/**
 * Feishu package 单元测试
 */
package feishu

import (
	"testing"
)

func TestNewSyncMap(t *testing.T) {
	m := newSyncMap(100)
	if m == nil {
		t.Fatal("newSyncMap 不应返回 nil")
	}
	if m.maxSize != 100 {
		t.Errorf("期望 maxSize 为 100, 实际为 %d", m.maxSize)
	}
}

func TestSyncMap_Add(t *testing.T) {
	m := newSyncMap(10)

	// First add should succeed
	if !m.add("key1") {
		t.Error("期望第一次添加成功")
	}

	// Second add should fail (duplicate)
	if m.add("key1") {
		t.Error("期望第二次添加失败")
	}

	// Different key should succeed
	if !m.add("key2") {
		t.Error("期望添加不同 key 成功")
	}
}

func TestSyncMap_Add_Expiration(t *testing.T) {
	m := newSyncMap(3)

	// Add 3 items
	m.add("key1")
	m.add("key2")
	m.add("key3")

	// Adding a 4th key should trigger cleanup
	result := m.add("key4")

	// Cleanup should have occurred, but key4 should still be added
	if !result {
		t.Error("期望添加 key4 成功")
	}

	// Adding same key should fail
	if m.add("key4") {
		t.Error("期望重复添加 key4 失败")
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "string value",
			input:    map[string]interface{}{"key": "value"},
			key:      "key",
			expected: "value",
		},
		{
			name:     "non-string value",
			input:    map[string]interface{}{"key": 123},
			key:      "key",
			expected: "",
		},
		{
			name:     "missing key",
			input:    map[string]interface{}{"other": "value"},
			key:      "key",
			expected: "",
		},
		{
			name:     "nil value",
			input:    map[string]interface{}{"key": nil},
			key:      "key",
			expected: "",
		},
		{
			name:     "empty string",
			input:    map[string]interface{}{"key": ""},
			key:      "key",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getString(tt.input, tt.key)
			if result != tt.expected {
				t.Errorf("期望 %q, 实际为 %q", tt.expected, result)
			}
		})
	}
}

func TestConfig_Defaults(t *testing.T) {
	cfg := &Config{}
	if cfg.AppID != "" {
		t.Errorf("期望 AppID 为空, 实际为 %s", cfg.AppID)
	}
	if cfg.AppSecret != "" {
		t.Errorf("期望 AppSecret 为空, 实际为 %s", cfg.AppSecret)
	}
	if cfg.ChannelCode != "" {
		t.Errorf("期望 ChannelCode 为空, 实际为 %s", cfg.ChannelCode)
	}
}

func TestChannelTypeFeishu(t *testing.T) {
	if ChannelTypeFeishu != "feishu" {
		t.Errorf("期望 ChannelTypeFeishu 为 feishu, 实际为 %s", ChannelTypeFeishu)
	}
}

func TestReactionInfo(t *testing.T) {
	info := &reactionInfo{
		messageID:  "msg-001",
		reactionID: "reaction-001",
	}

	if info.messageID != "msg-001" {
		t.Errorf("期望 messageID 为 msg-001, 实际为 %s", info.messageID)
	}
	if info.reactionID != "reaction-001" {
		t.Errorf("期望 reactionID 为 reaction-001, 实际为 %s", info.reactionID)
	}
}

func TestSyncMap_ConcurrentAdd(t *testing.T) {
	m := newSyncMap(1000)

	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(id int) {
			m.add("key")
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestSyncMap_LargeMaxSize(t *testing.T) {
	m := newSyncMap(100000)
	if m.maxSize != 100000 {
		t.Errorf("期望 maxSize 为 100000, 实际为 %d", m.maxSize)
	}
}

func TestSyncMap_ZeroMaxSize(t *testing.T) {
	m := newSyncMap(0)
	if m.maxSize != 0 {
		t.Errorf("期望 maxSize 为 0, 实际为 %d", m.maxSize)
	}
	// Adding to zero-size map should not panic
	m.add("key")
}

func TestFactory_ConfigParsing(t *testing.T) {
	config := map[string]interface{}{
		"app_id":             "app-123",
		"app_secret":         "secret-456",
		"encrypt_key":        "encrypt-789",
		"verification_token": "token-abc",
		"channel_code":       "channel-def",
		"channel_id":         "channel-ghi",
		"agent_code":         "agent-jkl",
		"user_code":          "user-mno",
		"allow_from":         []interface{}{"ip1", "ip2"},
	}

	// Factory is hard to test without mocking, but we can test getString
	cfg := &Config{
		AppID:             getString(config, "app_id"),
		AppSecret:         getString(config, "app_secret"),
		EncryptKey:        getString(config, "encrypt_key"),
		VerificationToken: getString(config, "verification_token"),
		ChannelCode:       getString(config, "channel_code"),
		ChannelID:         getString(config, "channel_id"),
		AgentCode:         getString(config, "agent_code"),
		UserCode:          getString(config, "user_code"),
	}

	if cfg.AppID != "app-123" {
		t.Errorf("期望 AppID 为 app-123, 实际为 %s", cfg.AppID)
	}
	if cfg.AppSecret != "secret-456" {
		t.Errorf("期望 AppSecret 为 secret-456, 实际为 %s", cfg.AppSecret)
	}
	if cfg.EncryptKey != "encrypt-789" {
		t.Errorf("期望 EncryptKey 为 encrypt-789, 实际为 %s", cfg.EncryptKey)
	}
	if cfg.ChannelCode != "channel-def" {
		t.Errorf("期望 ChannelCode 为 channel-def, 实际为 %s", cfg.ChannelCode)
	}
}

func TestMessageEvent(t *testing.T) {
	event := &MessageEvent{
		MessageID: "msg-001",
		ChatID:    "chat-001",
		MsgType:   "text",
		Content:   "hello",
		SenderID:  "user-001",
	}

	if event.MessageID != "msg-001" {
		t.Errorf("期望 MessageID 为 msg-001, 实际为 %s", event.MessageID)
	}
	if event.Content != "hello" {
		t.Errorf("期望 Content 为 hello, 实际为 %s", event.Content)
	}
}
