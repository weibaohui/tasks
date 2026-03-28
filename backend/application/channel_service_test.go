package application

import (
	"context"
	"strconv"
	"testing"

	"github.com/weibh/taskmanager/domain"
)

type mockChannelRepo struct {
	channels map[string]*domain.Channel
}

func newMockChannelRepo() *mockChannelRepo {
	return &mockChannelRepo{
		channels: make(map[string]*domain.Channel),
	}
}

func (m *mockChannelRepo) Save(ctx context.Context, channel *domain.Channel) error {
	m.channels[channel.ID().String()] = channel
	return nil
}

func (m *mockChannelRepo) FindByID(ctx context.Context, id domain.ChannelID) (*domain.Channel, error) {
	channel, ok := m.channels[id.String()]
	if !ok {
		return nil, nil
	}
	return channel, nil
}

func (m *mockChannelRepo) FindByCode(ctx context.Context, code domain.ChannelCode) (*domain.Channel, error) {
	for _, channel := range m.channels {
		if channel.ChannelCode().String() == code.String() {
			return channel, nil
		}
	}
	return nil, nil
}

func (m *mockChannelRepo) FindByUserCode(ctx context.Context, userCode string) ([]*domain.Channel, error) {
	var result []*domain.Channel
	for _, channel := range m.channels {
		if channel.UserCode() == userCode {
			result = append(result, channel)
		}
	}
	return result, nil
}

func (m *mockChannelRepo) FindByAgentCode(ctx context.Context, agentCode string) ([]*domain.Channel, error) {
	var result []*domain.Channel
	for _, channel := range m.channels {
		if channel.AgentCode() == agentCode {
			result = append(result, channel)
		}
	}
	return result, nil
}

func (m *mockChannelRepo) FindActiveByUserCode(ctx context.Context, userCode string) ([]*domain.Channel, error) {
	var result []*domain.Channel
	for _, channel := range m.channels {
		if channel.UserCode() == userCode && channel.IsActive() {
			result = append(result, channel)
		}
	}
	return result, nil
}

func (m *mockChannelRepo) FindActive(ctx context.Context) ([]*domain.Channel, error) {
	var result []*domain.Channel
	for _, channel := range m.channels {
		if channel.IsActive() {
			result = append(result, channel)
		}
	}
	return result, nil
}

func (m *mockChannelRepo) Delete(ctx context.Context, id domain.ChannelID) error {
	delete(m.channels, id.String())
	return nil
}

type mockChannelIDGen struct {
	count int
}

func (m *mockChannelIDGen) Generate() string {
	m.count++
	return "channel-id-" + strconv.Itoa(m.count)
}

func setupTestChannelSvc() *ChannelApplicationService {
	repo := newMockChannelRepo()
	idGen := &mockChannelIDGen{}
	return NewChannelApplicationService(repo, idGen)
}

func TestChannelService_CreateChannel(t *testing.T) {
	svc := setupTestChannelSvc()
	ctx := context.Background()

	channel, err := svc.CreateChannel(ctx, CreateChannelCommand{
		UserCode:  "usr_001",
		Name:      "测试渠道",
		Type:      domain.ChannelTypeFeishu,
		Config:    map[string]interface{}{"app_id": "test_app"},
		AllowFrom: []string{"*"},
		AgentCode: "agt_001",
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if channel.Name() != "测试渠道" {
		t.Errorf("期望 name 为 '测试渠道', 实际为 '%s'", channel.Name())
	}

	if channel.Type() != domain.ChannelTypeFeishu {
		t.Errorf("期望 type 为 'feishu', 实际为 '%s'", channel.Type())
	}

	if channel.UserCode() != "usr_001" {
		t.Errorf("期望 user_code 为 'usr_001', 实际为 '%s'", channel.UserCode())
	}

	if channel.AgentCode() != "agt_001" {
		t.Errorf("期望 agent_code 为 'agt_001', 实际为 '%s'", channel.AgentCode())
	}

	if !channel.IsActive() {
		t.Error("新创建的 channel 应该是激活状态")
	}
}

func TestChannelService_CreateChannel_WebSocket(t *testing.T) {
	svc := setupTestChannelSvc()
	ctx := context.Background()

	channel, err := svc.CreateChannel(ctx, CreateChannelCommand{
		UserCode: "usr_001",
		Name:     "WebSocket渠道",
		Type:     domain.ChannelTypeWebSocket,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if channel.Type() != domain.ChannelTypeWebSocket {
		t.Errorf("期望 type 为 'websocket', 实际为 '%s'", channel.Type())
	}
}

func TestChannelService_GetChannel(t *testing.T) {
	svc := setupTestChannelSvc()
	ctx := context.Background()

	created, _ := svc.CreateChannel(ctx, CreateChannelCommand{
		UserCode: "usr_001",
		Name:     "GetTestChannel",
		Type:     domain.ChannelTypeFeishu,
	})

	channel, err := svc.GetChannel(ctx, created.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if channel.Name() != "GetTestChannel" {
		t.Errorf("期望 name 为 'GetTestChannel', 实际为 '%s'", channel.Name())
	}
}

func TestChannelService_GetChannel_NotFound(t *testing.T) {
	svc := setupTestChannelSvc()
	ctx := context.Background()

	_, err := svc.GetChannel(ctx, domain.NewChannelID("non-existent"))
	if err != ErrChannelNotFound {
		t.Errorf("期望 ErrChannelNotFound, 实际为 %v", err)
	}
}

func TestChannelService_GetChannelByCode(t *testing.T) {
	svc := setupTestChannelSvc()
	ctx := context.Background()

	created, _ := svc.CreateChannel(ctx, CreateChannelCommand{
		UserCode: "usr_001",
		Name:     "ByCodeChannel",
		Type:     domain.ChannelTypeFeishu,
	})

	channel, err := svc.GetChannelByCode(ctx, created.ChannelCode())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if channel.Name() != "ByCodeChannel" {
		t.Errorf("期望 name 为 'ByCodeChannel', 实际为 '%s'", channel.Name())
	}
}

func TestChannelService_GetChannelByCode_NotFound(t *testing.T) {
	svc := setupTestChannelSvc()
	ctx := context.Background()

	_, err := svc.GetChannelByCode(ctx, domain.NewChannelCode("chn_non-existent"))
	if err != ErrChannelNotFound {
		t.Errorf("期望 ErrChannelNotFound, 实际为 %v", err)
	}
}

func TestChannelService_ListChannels(t *testing.T) {
	svc := setupTestChannelSvc()
	ctx := context.Background()

	svc.CreateChannel(ctx, CreateChannelCommand{UserCode: "usr_001", Name: "Channel1", Type: domain.ChannelTypeFeishu})
	svc.CreateChannel(ctx, CreateChannelCommand{UserCode: "usr_001", Name: "Channel2", Type: domain.ChannelTypeWebSocket})
	svc.CreateChannel(ctx, CreateChannelCommand{UserCode: "usr_002", Name: "Channel3", Type: domain.ChannelTypeFeishu})

	channels, err := svc.ListChannels(ctx, "usr_001")
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if len(channels) != 2 {
		t.Errorf("期望 2 个 channels, 实际为 %d", len(channels))
	}

	for _, ch := range channels {
		if ch.UserCode() != "usr_001" {
			t.Errorf("期望 user_code 为 'usr_001', 实际为 '%s'", ch.UserCode())
		}
	}
}

func TestChannelService_ListChannels_EmptyUserCode(t *testing.T) {
	svc := setupTestChannelSvc()
	ctx := context.Background()

	_, err := svc.ListChannels(ctx, "")
	if err == nil {
		t.Error("期望错误, 实际无错误")
	}
}

func TestChannelService_UpdateChannel(t *testing.T) {
	svc := setupTestChannelSvc()
	ctx := context.Background()

	created, _ := svc.CreateChannel(ctx, CreateChannelCommand{
		UserCode: "usr_001",
		Name:     "OriginalName",
		Type:     domain.ChannelTypeFeishu,
	})

	newName := "UpdatedName"
	isActive := false
	updated, err := svc.UpdateChannel(ctx, UpdateChannelCommand{
		ID:       created.ID(),
		Name:     &newName,
		IsActive: &isActive,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if updated.Name() != "UpdatedName" {
		t.Errorf("期望 name 为 'UpdatedName', 实际为 '%s'", updated.Name())
	}

	if updated.IsActive() {
		t.Error("channel 应该是非激活状态")
	}
}

func TestChannelService_UpdateChannel_Config(t *testing.T) {
	svc := setupTestChannelSvc()
	ctx := context.Background()

	created, _ := svc.CreateChannel(ctx, CreateChannelCommand{
		UserCode: "usr_001",
		Name:     "ConfigTestChannel",
		Type:     domain.ChannelTypeFeishu,
	})

	newConfig := map[string]interface{}{
		"app_id":     "new_app_id",
		"app_secret": "new_secret",
	}
	updated, err := svc.UpdateChannel(ctx, UpdateChannelCommand{
		ID:     created.ID(),
		Config: &newConfig,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	cfg := updated.Config()
	if cfg["app_id"] != "new_app_id" {
		t.Errorf("期望 app_id 为 'new_app_id', 实际为 '%v'", cfg["app_id"])
	}
}

func TestChannelService_UpdateChannel_BindAgent(t *testing.T) {
	svc := setupTestChannelSvc()
	ctx := context.Background()

	created, _ := svc.CreateChannel(ctx, CreateChannelCommand{
		UserCode: "usr_001",
		Name:     "BindTestChannel",
		Type:     domain.ChannelTypeFeishu,
	})

	newAgentCode := "agt_new_agent"
	updated, err := svc.UpdateChannel(ctx, UpdateChannelCommand{
		ID:        created.ID(),
		AgentCode: &newAgentCode,
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if updated.AgentCode() != "agt_new_agent" {
		t.Errorf("期望 agent_code 为 'agt_new_agent', 实际为 '%s'", updated.AgentCode())
	}
}

func TestChannelService_UpdateChannel_NotFound(t *testing.T) {
	svc := setupTestChannelSvc()
	ctx := context.Background()

	newName := "NewName"
	_, err := svc.UpdateChannel(ctx, UpdateChannelCommand{
		ID:   domain.NewChannelID("non-existent"),
		Name: &newName,
	})
	if err != ErrChannelNotFound {
		t.Errorf("期望 ErrChannelNotFound, 实际为 %v", err)
	}
}

func TestChannelService_DeleteChannel(t *testing.T) {
	svc := setupTestChannelSvc()
	ctx := context.Background()

	created, _ := svc.CreateChannel(ctx, CreateChannelCommand{
		UserCode: "usr_001",
		Name:     "DeleteTestChannel",
		Type:     domain.ChannelTypeFeishu,
	})

	err := svc.DeleteChannel(ctx, created.ID())
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	_, err = svc.GetChannel(ctx, created.ID())
	if err != ErrChannelNotFound {
		t.Errorf("期望 ErrChannelNotFound, 实际为 %v", err)
	}
}

func TestChannelService_DeleteChannel_NotFound(t *testing.T) {
	svc := setupTestChannelSvc()
	ctx := context.Background()

	err := svc.DeleteChannel(ctx, domain.NewChannelID("non-existent"))
	if err != ErrChannelNotFound {
		t.Errorf("期望 ErrChannelNotFound, 实际为 %v", err)
	}
}

func TestChannelService_ListActiveChannels(t *testing.T) {
	svc := setupTestChannelSvc()
	ctx := context.Background()

	channel1, _ := svc.CreateChannel(ctx, CreateChannelCommand{
		UserCode: "usr_001",
		Name:     "ActiveChannel",
		Type:     domain.ChannelTypeFeishu,
	})

	channel2, _ := svc.CreateChannel(ctx, CreateChannelCommand{
		UserCode: "usr_001",
		Name:     "InactiveChannel",
		Type:     domain.ChannelTypeWebSocket,
	})

	// Deactivate channel2
	isActive := false
	svc.UpdateChannel(ctx, UpdateChannelCommand{
		ID:       channel2.ID(),
		IsActive: &isActive,
	})

	channels, err := svc.ListActiveChannels(ctx)
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	for _, ch := range channels {
		if !ch.IsActive() {
			t.Error("列表中所有 channel 都应该是激活状态")
		}
		if ch.ID().String() == channel1.ID().String() && !ch.IsActive() {
			t.Error("channel1 应该是激活状态")
		}
	}

	// channel2 应该不在列表中
	for _, ch := range channels {
		if ch.ID().String() == channel2.ID().String() {
			t.Error("channel2 不应该在激活列表中")
		}
	}
}
