/**
 * Channel package 单元测试
 */
package channel

import (
	"context"
	"errors"
	"testing"

	"github.com/weibh/taskmanager/pkg/bus"
)

// mockChannel - 用于测试的 Channel 模拟
type mockChannel struct {
	nameVal    string
	typeVal    string
	startErr   error
	stopCalled bool
}

func (m *mockChannel) Name() string                    { return m.nameVal }
func (m *mockChannel) Type() string                   { return m.typeVal }
func (m *mockChannel) Start(ctx context.Context) error { return m.startErr }
func (m *mockChannel) Stop()                          { m.stopCalled = true }

// mockChannelFactory - 用于测试的 ChannelFactory 模拟
type mockChannelFactory struct {
	channel *mockChannel
	err     error
}

func (f *mockChannelFactory) create(config map[string]interface{}, messageBus *bus.MessageBus) (Channel, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.channel, nil
}

func TestNewRegistry(t *testing.T) {
	msgBus := bus.NewMessageBus(nil)
	registry := NewRegistry(msgBus)
	if registry == nil {
		t.Fatal("NewRegistry 不应返回 nil")
	}
}

func TestRegistry_Register(t *testing.T) {
	msgBus := bus.NewMessageBus(nil)
	registry := NewRegistry(msgBus)

	channel := &mockChannel{nameVal: "test", typeVal: "feishu"}
	factory := func(config map[string]interface{}, mb *bus.MessageBus) (Channel, error) {
		return channel, nil
	}

	registry.Register("feishu", factory)

	f, ok := registry.GetFactory("feishu")
	if !ok {
		t.Error("期望找到 feishu factory")
	}
	if f == nil {
		t.Error("factory 不应为 nil")
	}
}

func TestRegistry_GetFactory_NotFound(t *testing.T) {
	msgBus := bus.NewMessageBus(nil)
	registry := NewRegistry(msgBus)

	_, ok := registry.GetFactory("nonexistent")
	if ok {
		t.Error("不应找到 nonexistent factory")
	}
}

func TestRegistry_CreateChannel(t *testing.T) {
	msgBus := bus.NewMessageBus(nil)
	registry := NewRegistry(msgBus)

	channel := &mockChannel{nameVal: "test", typeVal: "feishu"}
	factory := func(config map[string]interface{}, mb *bus.MessageBus) (Channel, error) {
		return channel, nil
	}

	registry.Register("feishu", factory)

	created, err := registry.CreateChannel("feishu", map[string]interface{}{})
	if err != nil {
		t.Fatalf("CreateChannel 失败: %v", err)
	}
	if created.Name() != "test" {
		t.Errorf("期望 name 为 test, 实际为 %s", created.Name())
	}
}

func TestRegistry_CreateChannel_NotFound(t *testing.T) {
	msgBus := bus.NewMessageBus(nil)
	registry := NewRegistry(msgBus)

	_, err := registry.CreateChannel("nonexistent", map[string]interface{}{})
	if err == nil {
		t.Error("期望错误")
	}
}

func TestRegistry_ListTypes(t *testing.T) {
	msgBus := bus.NewMessageBus(nil)
	registry := NewRegistry(msgBus)

	factory := func(config map[string]interface{}, mb *bus.MessageBus) (Channel, error) {
		return &mockChannel{}, nil
	}

	registry.Register("feishu", factory)
	registry.Register("dingtalk", factory)

	types := registry.ListTypes()
	if len(types) != 2 {
		t.Errorf("期望 2 个类型, 实际为 %d", len(types))
	}
}

func TestNewManager(t *testing.T) {
	msgBus := bus.NewMessageBus(nil)
	manager := NewManager(msgBus)
	if manager == nil {
		t.Fatal("NewManager 不应返回 nil")
	}
}

func TestManager_Register(t *testing.T) {
	msgBus := bus.NewMessageBus(nil)
	manager := NewManager(msgBus)

	channel := &mockChannel{nameVal: "test-channel", typeVal: "feishu"}
	manager.Register(channel)

	if manager.Get("test-channel") == nil {
		t.Error("期望找到 test-channel")
	}
}

func TestManager_Get_NotFound(t *testing.T) {
	msgBus := bus.NewMessageBus(nil)
	manager := NewManager(msgBus)

	if manager.Get("nonexistent") != nil {
		t.Error("不应找到 nonexistent channel")
	}
}

func TestManager_StartAll(t *testing.T) {
	msgBus := bus.NewMessageBus(nil)
	manager := NewManager(msgBus)

	channel := &mockChannel{nameVal: "test", typeVal: "feishu"}
	manager.Register(channel)

	err := manager.StartAll(context.Background())
	if err != nil {
		t.Fatalf("StartAll 失败: %v", err)
	}
}

func TestManager_StartAll_WithError(t *testing.T) {
	msgBus := bus.NewMessageBus(nil)
	manager := NewManager(msgBus)

	channel := &mockChannel{nameVal: "test", typeVal: "feishu", startErr: errors.New("start error")}
	manager.Register(channel)

	err := manager.StartAll(context.Background())
	if err == nil {
		t.Error("期望错误")
	}
}

func TestManager_StopAll(t *testing.T) {
	msgBus := bus.NewMessageBus(nil)
	manager := NewManager(msgBus)

	channel := &mockChannel{nameVal: "test", typeVal: "feishu"}
	manager.Register(channel)

	manager.StopAll()

	if !channel.stopCalled {
		t.Error("期望 Stop 被调用")
	}
}

func TestManager_List(t *testing.T) {
	msgBus := bus.NewMessageBus(nil)
	manager := NewManager(msgBus)

	channel1 := &mockChannel{nameVal: "channel-1", typeVal: "feishu"}
	channel2 := &mockChannel{nameVal: "channel-2", typeVal: "dingtalk"}
	manager.Register(channel1)
	manager.Register(channel2)

	names := manager.List()
	if len(names) != 2 {
		t.Errorf("期望 2 个 channel, 实际为 %d", len(names))
	}
}

func TestChannelType_Constants(t *testing.T) {
	if ChannelTypeFeishu != "feishu" {
		t.Errorf("期望 ChannelTypeFeishu 为 feishu, 实际为 %s", ChannelTypeFeishu)
	}
	if ChannelTypeDingTalk != "dingtalk" {
		t.Errorf("期望 ChannelTypeDingTalk 为 dingtalk, 实际为 %s", ChannelTypeDingTalk)
	}
	if ChannelTypeWeChat != "wechat" {
		t.Errorf("期望 ChannelTypeWeChat 为 wechat, 实际为 %s", ChannelTypeWeChat)
	}
	if ChannelTypeWebSocket != "websocket" {
		t.Errorf("期望 ChannelTypeWebSocket 为 websocket, 实际为 %s", ChannelTypeWebSocket)
	}
}

func TestNewAgentConfigCache(t *testing.T) {
	cache := NewAgentConfigCache()
	if cache == nil {
		t.Fatal("NewAgentConfigCache 不应返回 nil")
	}
}

func TestAgentConfigCache_Get_Set(t *testing.T) {
	cache := NewAgentConfigCache()
	cfg := &AgentConfig{
		AgentCode:    "agent-001",
		Name:         "Test Agent",
		Instructions: "You are a helpful assistant",
		Tools:        []string{"bash", "mcp"},
		MCPs:         []string{"server-1"},
	}

	cache.Set("key-1", cfg)

	retrieved, ok := cache.Get("key-1")
	if !ok {
		t.Error("期望找到 key-1")
	}
	if retrieved.AgentCode != "agent-001" {
		t.Errorf("期望 AgentCode 为 agent-001, 实际为 %s", retrieved.AgentCode)
	}
	if retrieved.Name != "Test Agent" {
		t.Errorf("期望 Name 为 Test Agent, 实际为 %s", retrieved.Name)
	}
	if len(retrieved.Tools) != 2 {
		t.Errorf("期望 Tools 长度为 2, 实际为 %d", len(retrieved.Tools))
	}
}

func TestAgentConfigCache_Get_NotFound(t *testing.T) {
	cache := NewAgentConfigCache()

	_, ok := cache.Get("nonexistent")
	if ok {
		t.Error("不应找到 nonexistent key")
	}
}

func TestAgentConfigCache_Clear(t *testing.T) {
	cache := NewAgentConfigCache()
	cfg := &AgentConfig{
		AgentCode: "agent-001",
		Name:      "Test Agent",
	}

	cache.Set("key-1", cfg)
	cache.Clear("key-1")

	_, ok := cache.Get("key-1")
	if ok {
		t.Error("key-1 已被清除")
	}
}

func TestAgentConfigCache_Overwrite(t *testing.T) {
	cache := NewAgentConfigCache()

	cfg1 := &AgentConfig{AgentCode: "agent-001", Name: "First"}
	cfg2 := &AgentConfig{AgentCode: "agent-002", Name: "Second"}

	cache.Set("key-1", cfg1)
	cache.Set("key-1", cfg2)

	retrieved, _ := cache.Get("key-1")
	if retrieved.AgentCode != "agent-002" {
		t.Errorf("期望 AgentCode 为 agent-002, 实际为 %s", retrieved.AgentCode)
	}
}