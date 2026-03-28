/**
 * Session 和 Channel 领域模型单元测试
 */
package domain

import (
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	session, err := NewSession(
		NewSessionID("sess-001"),
		"user-001",
		"channel-001",
		"key-001",
		"ext-001",
		"agent-001",
	)

	if err != nil {
		t.Fatalf("创建 Session 失败: %v", err)
	}

	if session.ID() != NewSessionID("sess-001") {
		t.Errorf("期望 ID 为 sess-001, 实际为 %s", session.ID())
	}

	if session.UserCode() != "user-001" {
		t.Errorf("期望 UserCode 为 user-001, 实际为 %s", session.UserCode())
	}

	if session.ChannelCode() != "channel-001" {
		t.Errorf("期望 ChannelCode 为 channel-001, 实际为 %s", session.ChannelCode())
	}

	if session.SessionKey() != "key-001" {
		t.Errorf("期望 SessionKey 为 key-001, 实际为 %s", session.SessionKey())
	}

	if session.ExternalID() != "ext-001" {
		t.Errorf("期望 ExternalID 为 ext-001, 实际为 %s", session.ExternalID())
	}

	if session.AgentCode() != "agent-001" {
		t.Errorf("期望 AgentCode 为 agent-001, 实际为 %s", session.AgentCode())
	}
}

func TestNewSession_EmptyID(t *testing.T) {
	_, err := NewSession(
		NewSessionID(""),
		"user-001",
		"channel-001",
		"key-001",
		"ext-001",
		"agent-001",
	)

	if err != ErrSessionIDRequired {
		t.Errorf("期望返回 ErrSessionIDRequired, 实际返回 %v", err)
	}
}

func TestNewSession_EmptyUserCode(t *testing.T) {
	_, err := NewSession(
		NewSessionID("sess-001"),
		"   ",
		"channel-001",
		"key-001",
		"ext-001",
		"agent-001",
	)

	if err != ErrSessionUserCodeMissing {
		t.Errorf("期望返回 ErrSessionUserCodeMissing, 实际返回 %v", err)
	}
}

func TestNewSession_EmptyChannelCode(t *testing.T) {
	_, err := NewSession(
		NewSessionID("sess-001"),
		"user-001",
		"   ",
		"key-001",
		"ext-001",
		"agent-001",
	)

	if err != ErrSessionChannelMissing {
		t.Errorf("期望返回 ErrSessionChannelMissing, 实际返回 %v", err)
	}
}

func TestNewSession_EmptySessionKey(t *testing.T) {
	_, err := NewSession(
		NewSessionID("sess-001"),
		"user-001",
		"channel-001",
		"   ",
		"ext-001",
		"agent-001",
	)

	if err != ErrSessionKeyRequired {
		t.Errorf("期望返回 ErrSessionKeyRequired, 实际返回 %v", err)
	}
}

func TestSession_Touch(t *testing.T) {
	session, _ := NewSession(
		NewSessionID("sess-001"),
		"user-001",
		"channel-001",
		"key-001",
		"ext-001",
		"agent-001",
	)

	originalLastActive := session.LastActiveAt()
	time.Sleep(10 * time.Millisecond)

	session.Touch()

	if session.LastActiveAt().Equal(*originalLastActive) {
		t.Error("LastActiveAt 应该更新")
	}
}

func TestSession_SetMetadata(t *testing.T) {
	session, _ := NewSession(
		NewSessionID("sess-001"),
		"user-001",
		"channel-001",
		"key-001",
		"ext-001",
		"agent-001",
	)

	metadata := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}
	session.SetMetadata(metadata)

	result := session.Metadata()
	if result["key1"] != "value1" {
		t.Errorf("期望 key1 为 value1, 实际为 %v", result["key1"])
	}
}

func TestSession_SetMetadata_ReturnsCopy(t *testing.T) {
	session, _ := NewSession(
		NewSessionID("sess-001"),
		"user-001",
		"channel-001",
		"key-001",
		"ext-001",
		"agent-001",
	)

	session.SetMetadata(map[string]interface{}{"key": "original"})

	metadata := session.Metadata()
	metadata["key"] = "modified"

	metadata2 := session.Metadata()
	if metadata2["key"] == "modified" {
		t.Error("Metadata 应返回拷贝，不应受外部修改影响")
	}
}

func TestSession_SetAgentCode(t *testing.T) {
	session, _ := NewSession(
		NewSessionID("sess-001"),
		"user-001",
		"channel-001",
		"key-001",
		"ext-001",
		"",
	)

	session.SetAgentCode("new-agent")

	if session.AgentCode() != "new-agent" {
		t.Errorf("期望 AgentCode 为 new-agent, 实际为 %s", session.AgentCode())
	}
}

func TestSession_ToSnapshot(t *testing.T) {
	session, _ := NewSession(
		NewSessionID("sess-001"),
		"user-001",
		"channel-001",
		"key-001",
		"ext-001",
		"agent-001",
	)
	session.SetMetadata(map[string]interface{}{"k": "v"})

	snap := session.ToSnapshot()

	if snap.ID != session.ID() {
		t.Errorf("ID 不匹配")
	}
	if snap.UserCode != "user-001" {
		t.Errorf("UserCode 不匹配")
	}
	if snap.AgentCode != "agent-001" {
		t.Errorf("AgentCode 不匹配")
	}
}

func TestSession_FromSnapshot(t *testing.T) {
	session, _ := NewSession(
		NewSessionID("sess-001"),
		"user-001",
		"channel-001",
		"key-001",
		"ext-001",
		"agent-001",
	)

	snap := SessionSnapshot{
		ID:          NewSessionID("sess-002"),
		UserCode:    "user-002",
		AgentCode:   "agent-002",
		ChannelCode: "channel-002",
		SessionKey:  "key-002",
		ExternalID:  "ext-002",
		Metadata:    map[string]interface{}{"k": "v"},
	}

	session.FromSnapshot(snap)

	if session.ID() != NewSessionID("sess-002") {
		t.Errorf("ID 不匹配")
	}
	if session.UserCode() != "user-002" {
		t.Errorf("UserCode 不匹配")
	}
	if session.AgentCode() != "agent-002" {
		t.Errorf("AgentCode 不匹配")
	}
}

func TestCloneTimePtr(t *testing.T) {
	now := time.Now()
	cloned := cloneTimePtr(&now)

	// 克隆的时间初始应该相等
	if !cloned.Equal(now) {
		t.Error("克隆的时间应该相等")
	}

	// 修改原始时间不应影响克隆
	now = now.Add(time.Hour)
	if cloned.Equal(now) {
		t.Error("修改原始时间不应影响克隆")
	}

	// nil 测试
	var nilTime *time.Time
	if cloneTimePtr(nilTime) != nil {
		t.Error("nil 应返回 nil")
	}
}

// Channel tests

func TestNewChannel(t *testing.T) {
	channel, err := NewChannel(
		NewChannelID("ch-001"),
		NewChannelCode("ch-code"),
		"user-001",
		"Test Channel",
		ChannelTypeFeishu,
	)

	if err != nil {
		t.Fatalf("创建 Channel 失败: %v", err)
	}

	if channel.ID() != NewChannelID("ch-001") {
		t.Errorf("期望 ID 为 ch-001, 实际为 %s", channel.ID())
	}

	if channel.ChannelCode() != NewChannelCode("ch-code") {
		t.Errorf("期望 ChannelCode 为 ch-code, 实际为 %s", channel.ChannelCode())
	}

	if channel.Name() != "Test Channel" {
		t.Errorf("期望 Name 为 Test Channel, 实际为 %s", channel.Name())
	}

	if channel.Type() != ChannelTypeFeishu {
		t.Errorf("期望 Type 为 feishu, 实际为 %s", channel.Type())
	}

	if !channel.IsActive() {
		t.Error("期望默认 IsActive 为 true")
	}
}

func TestNewChannel_EmptyID(t *testing.T) {
	_, err := NewChannel(
		NewChannelID(""),
		NewChannelCode("ch-code"),
		"user-001",
		"Test Channel",
		ChannelTypeFeishu,
	)

	if err != ErrChannelIDRequired {
		t.Errorf("期望返回 ErrChannelIDRequired, 实际返回 %v", err)
	}
}

func TestNewChannel_EmptyCode(t *testing.T) {
	_, err := NewChannel(
		NewChannelID("ch-001"),
		NewChannelCode(""),
		"user-001",
		"Test Channel",
		ChannelTypeFeishu,
	)

	if err != ErrChannelCodeRequired {
		t.Errorf("期望返回 ErrChannelCodeRequired, 实际返回 %v", err)
	}
}

func TestNewChannel_EmptyUserCode(t *testing.T) {
	_, err := NewChannel(
		NewChannelID("ch-001"),
		NewChannelCode("ch-code"),
		"   ",
		"Test Channel",
		ChannelTypeFeishu,
	)

	if err != ErrChannelUserCodeRequired {
		t.Errorf("期望返回 ErrChannelUserCodeRequired, 实际返回 %v", err)
	}
}

func TestNewChannel_EmptyName(t *testing.T) {
	_, err := NewChannel(
		NewChannelID("ch-001"),
		NewChannelCode("ch-code"),
		"user-001",
		"   ",
		ChannelTypeFeishu,
	)

	if err != ErrChannelNameRequired {
		t.Errorf("期望返回 ErrChannelNameRequired, 实际返回 %v", err)
	}
}

func TestNewChannel_InvalidType(t *testing.T) {
	_, err := NewChannel(
		NewChannelID("ch-001"),
		NewChannelCode("ch-code"),
		"user-001",
		"Test Channel",
		"invalid",
	)

	if err != ErrChannelTypeInvalid {
		t.Errorf("期望返回 ErrChannelTypeInvalid, 实际返回 %v", err)
	}
}

func TestChannelType_IsValid(t *testing.T) {
	if !ChannelTypeFeishu.IsValid() {
		t.Error("feishu 应为有效类型")
	}

	if !ChannelTypeWebSocket.IsValid() {
		t.Error("websocket 应为有效类型")
	}

	invalid := ChannelType("invalid")
	if invalid.IsValid() {
		t.Error("invalid 应为无效类型")
	}
}

func TestChannel_UpdateName(t *testing.T) {
	channel, _ := NewChannel(
		NewChannelID("ch-001"),
		NewChannelCode("ch-code"),
		"user-001",
		"Original Name",
		ChannelTypeFeishu,
	)

	err := channel.UpdateName("New Name")
	if err != nil {
		t.Fatalf("UpdateName 失败: %v", err)
	}

	if channel.Name() != "New Name" {
		t.Errorf("期望 Name 为 New Name, 实际为 %s", channel.Name())
	}
}

func TestChannel_UpdateName_Empty(t *testing.T) {
	channel, _ := NewChannel(
		NewChannelID("ch-001"),
		NewChannelCode("ch-code"),
		"user-001",
		"Original Name",
		ChannelTypeFeishu,
	)

	err := channel.UpdateName("   ")
	if err != ErrChannelNameRequired {
		t.Errorf("期望返回 ErrChannelNameRequired, 实际返回 %v", err)
	}
}

func TestChannel_UpdateConfig(t *testing.T) {
	channel, _ := NewChannel(
		NewChannelID("ch-001"),
		NewChannelCode("ch-code"),
		"user-001",
		"Test Channel",
		ChannelTypeFeishu,
	)

	config := map[string]interface{}{"key": "value"}
	channel.UpdateConfig(config)

	result := channel.Config()
	if result["key"] != "value" {
		t.Errorf("期望 config key 为 value, 实际为 %v", result["key"])
	}
}

func TestChannel_Config_IncludesAgentAndUser(t *testing.T) {
	channel, _ := NewChannel(
		NewChannelID("ch-001"),
		NewChannelCode("ch-code"),
		"user-001",
		"Test Channel",
		ChannelTypeFeishu,
	)
	channel.BindAgent("agent-001")

	config := channel.Config()

	if config["agent_code"] != "agent-001" {
		t.Errorf("期望 agent_code 为 agent-001, 实际为 %v", config["agent_code"])
	}

	if config["user_code"] != "user-001" {
		t.Errorf("期望 user_code 为 user-001, 实际为 %v", config["user_code"])
	}

	if config["channel_code"] != "ch-code" {
		t.Errorf("期望 channel_code 为 ch-code, 实际为 %v", config["channel_code"])
	}
}

func TestChannel_SetAllowFrom(t *testing.T) {
	channel, _ := NewChannel(
		NewChannelID("ch-001"),
		NewChannelCode("ch-code"),
		"user-001",
		"Test Channel",
		ChannelTypeFeishu,
	)

	channel.SetAllowFrom([]string{"ip1", "ip2"})

	allowFrom := channel.AllowFrom()
	if len(allowFrom) != 2 {
		t.Errorf("期望 2 个 allowFrom, 实际为 %d", len(allowFrom))
	}
}

func TestChannel_BindAgent(t *testing.T) {
	channel, _ := NewChannel(
		NewChannelID("ch-001"),
		NewChannelCode("ch-code"),
		"user-001",
		"Test Channel",
		ChannelTypeFeishu,
	)

	channel.BindAgent("agent-001")

	if channel.AgentCode() != "agent-001" {
		t.Errorf("期望 AgentCode 为 agent-001, 实际为 %s", channel.AgentCode())
	}
}

func TestChannel_SetActive(t *testing.T) {
	channel, _ := NewChannel(
		NewChannelID("ch-001"),
		NewChannelCode("ch-code"),
		"user-001",
		"Test Channel",
		ChannelTypeFeishu,
	)

	channel.SetActive(false)

	if channel.IsActive() {
		t.Error("期望 IsActive 为 false")
	}
}

func TestChannel_ToSnapshot(t *testing.T) {
	channel, _ := NewChannel(
		NewChannelID("ch-001"),
		NewChannelCode("ch-code"),
		"user-001",
		"Test Channel",
		ChannelTypeFeishu,
	)
	channel.BindAgent("agent-001")
	channel.SetActive(false)

	snap := channel.ToSnapshot()

	if snap.ID != channel.ID() {
		t.Errorf("ID 不匹配")
	}
	if snap.Name != "Test Channel" {
		t.Errorf("Name 不匹配")
	}
	if snap.AgentCode != "agent-001" {
		t.Errorf("AgentCode 不匹配")
	}
	if snap.IsActive {
		t.Error("IsActive 不匹配")
	}
}

func TestChannel_FromSnapshot(t *testing.T) {
	channel, _ := NewChannel(
		NewChannelID("ch-001"),
		NewChannelCode("ch-code"),
		"user-001",
		"Original",
		ChannelTypeFeishu,
	)

	snap := ChannelSnapshot{
		ID:        NewChannelID("ch-002"),
		Code:      NewChannelCode("new-code"),
		UserCode:  "user-002",
		AgentCode: "agent-002",
		Name:      "New Name",
		Type:      ChannelTypeWebSocket,
		IsActive:  false,
		AllowFrom: []string{"ip1"},
		Config:    map[string]interface{}{"k": "v"},
	}

	channel.FromSnapshot(snap)

	if channel.ID() != NewChannelID("ch-002") {
		t.Errorf("ID 不匹配")
	}
	if channel.ChannelCode() != NewChannelCode("new-code") {
		t.Errorf("ChannelCode 不匹配")
	}
	if channel.AgentCode() != "agent-002" {
		t.Errorf("AgentCode 不匹配")
	}
	if channel.Name() != "New Name" {
		t.Errorf("Name 不匹配")
	}
	if channel.Type() != ChannelTypeWebSocket {
		t.Errorf("Type 不匹配")
	}
	if channel.IsActive() {
		t.Error("IsActive 不匹配")
	}
}

func TestCloneMap(t *testing.T) {
	original := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": map[string]interface{}{"nested": "value"},
	}

	cloned := cloneMap(original)

	if cloned["key1"] != original["key1"] {
		t.Error("key1 值不匹配")
	}

	// 修改原始不应影响克隆
	original["key1"] = "modified"
	if cloned["key1"] == "modified" {
		t.Error("克隆应独立于原始")
	}

	// nil 测试
	nilMap := cloneMap(nil)
	if nilMap == nil {
		t.Error("nil 应返回空 map")
	}
	if len(nilMap) != 0 {
		t.Error("空 map 长度应为 0")
	}
}
