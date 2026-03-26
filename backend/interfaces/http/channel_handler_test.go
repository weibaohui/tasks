/**
 * ChannelHandler 单元测试
 */
package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

// mockChannelRepository - 用于测试的 Channel 仓库模拟
type mockChannelRepository struct {
	channels   map[domain.ChannelID]*domain.Channel
	channelCodes map[string]*domain.Channel
}

func newMockChannelRepository() *mockChannelRepository {
	return &mockChannelRepository{
		channels:    make(map[domain.ChannelID]*domain.Channel),
		channelCodes: make(map[string]*domain.Channel),
	}
}

func (r *mockChannelRepository) Save(ctx context.Context, channel *domain.Channel) error {
	r.channels[channel.ID()] = channel
	r.channelCodes[channel.ChannelCode().String()] = channel
	return nil
}

func (r *mockChannelRepository) Delete(ctx context.Context, id domain.ChannelID) error {
	delete(r.channels, id)
	return nil
}

func (r *mockChannelRepository) FindByID(ctx context.Context, id domain.ChannelID) (*domain.Channel, error) {
	return r.channels[id], nil
}

func (r *mockChannelRepository) FindByCode(ctx context.Context, code domain.ChannelCode) (*domain.Channel, error) {
	return r.channelCodes[code.String()], nil
}

func (r *mockChannelRepository) FindByUserCode(ctx context.Context, userCode string) ([]*domain.Channel, error) {
	var result []*domain.Channel
	for _, ch := range r.channels {
		if ch.UserCode() == userCode {
			result = append(result, ch)
		}
	}
	return result, nil
}

func (r *mockChannelRepository) FindByAgentCode(ctx context.Context, agentCode string) ([]*domain.Channel, error) {
	var result []*domain.Channel
	for _, ch := range r.channels {
		if ch.AgentCode() == agentCode {
			result = append(result, ch)
		}
	}
	return result, nil
}

func (r *mockChannelRepository) FindActiveByUserCode(ctx context.Context, userCode string) ([]*domain.Channel, error) {
	var result []*domain.Channel
	for _, ch := range r.channels {
		if ch.UserCode() == userCode && ch.IsActive() {
			result = append(result, ch)
		}
	}
	return result, nil
}

func (r *mockChannelRepository) FindActive(ctx context.Context) ([]*domain.Channel, error) {
	var result []*domain.Channel
	for _, ch := range r.channels {
		if ch.IsActive() {
			result = append(result, ch)
		}
	}
	return result, nil
}

// mockChannelIDGenerator - 用于测试的 ID 生成器模拟
type mockChannelIDGenerator struct {
	prefix string
	count  int
}

func newMockChannelIDGenerator(prefix string) *mockChannelIDGenerator {
	return &mockChannelIDGenerator{prefix: prefix}
}

func (g *mockChannelIDGenerator) Generate() string {
	g.count++
	return g.prefix + "-id-" + strconv.Itoa(g.count)
}

func TestChannelHandler_ListChannelTypes(t *testing.T) {
	handler := &ChannelHandler{}
	req := httptest.NewRequest("GET", "/channel-types", nil)
	w := httptest.NewRecorder()

	handler.ListChannelTypes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, w.Code)
	}

	var resp []ChannelTypeOption
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if len(resp) != 2 {
		t.Errorf("期望 2 个类型, 实际为 %d", len(resp))
	}

	if resp[0].Key != "feishu" {
		t.Errorf("期望第一个类型为 feishu, 实际为 %s", resp[0].Key)
	}

	if resp[1].Key != "websocket" {
		t.Errorf("期望第二个类型为 websocket, 实际为 %s", resp[1].Key)
	}
}

func TestChannelHandler_CreateChannel_InvalidJSON(t *testing.T) {
	repo := newMockChannelRepository()
	svc := application.NewChannelApplicationService(repo, newMockChannelIDGenerator("ch"))
	handler := NewChannelHandler(svc)

	req := httptest.NewRequest("POST", "/channels", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateChannel(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestChannelHandler_CreateChannel_Success(t *testing.T) {
	repo := newMockChannelRepository()
	svc := application.NewChannelApplicationService(repo, newMockChannelIDGenerator("ch"))
	handler := NewChannelHandler(svc)

	body := `{
		"user_code": "user-001",
		"name": "Test Channel",
		"type": "feishu",
		"config": {},
		"allow_from": [],
		"agent_code": "agent-001"
	}`
	req := httptest.NewRequest("POST", "/channels", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateChannel(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusCreated, w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp["name"] != "Test Channel" {
		t.Errorf("期望 name 为 Test Channel, 实际为 %v", resp["name"])
	}
}

func TestChannelHandler_ListChannels(t *testing.T) {
	repo := newMockChannelRepository()
	svc := application.NewChannelApplicationService(repo, newMockChannelIDGenerator("ch"))

	// 先创建一个 channel
	svc.CreateChannel(context.Background(), application.CreateChannelCommand{
		UserCode: "user-001",
		Name:     "Test Channel",
		Type:     domain.ChannelTypeFeishu,
	})

	handler := NewChannelHandler(svc)

	req := httptest.NewRequest("GET", "/channels?user_code=user-001", nil)
	w := httptest.NewRecorder()

	handler.ListChannels(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, w.Code)
	}

	var resp []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if len(resp) != 1 {
		t.Errorf("期望 1 个 channel, 实际为 %d", len(resp))
	}
}

func TestChannelHandler_GetChannel_NoIDOrCode(t *testing.T) {
	repo := newMockChannelRepository()
	svc := application.NewChannelApplicationService(repo, newMockChannelIDGenerator("ch"))
	handler := NewChannelHandler(svc)

	req := httptest.NewRequest("GET", "/channel", nil)
	w := httptest.NewRecorder()

	handler.GetChannel(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestChannelHandler_UpdateChannel_NoID(t *testing.T) {
	repo := newMockChannelRepository()
	svc := application.NewChannelApplicationService(repo, newMockChannelIDGenerator("ch"))
	handler := NewChannelHandler(svc)

	req := httptest.NewRequest("PUT", "/channel", strings.NewReader("{}"))
	w := httptest.NewRecorder()

	handler.UpdateChannel(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestChannelHandler_DeleteChannel_NoID(t *testing.T) {
	repo := newMockChannelRepository()
	svc := application.NewChannelApplicationService(repo, newMockChannelIDGenerator("ch"))
	handler := NewChannelHandler(svc)

	req := httptest.NewRequest("DELETE", "/channel", nil)
	w := httptest.NewRecorder()

	handler.DeleteChannel(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestChannelToMap(t *testing.T) {
	channel, _ := domain.NewChannel(
		domain.NewChannelID("ch-001"),
		domain.NewChannelCode("ch-code"),
		"user-001",
		"Test Channel",
		domain.ChannelTypeFeishu,
	)

	result := channelToMap(channel)

	if result["id"] != "ch-001" {
		t.Errorf("期望 id 为 ch-001, 实际为 %v", result["id"])
	}

	if result["channel_code"] != "ch-code" {
		t.Errorf("期望 channel_code 为 ch-code, 实际为 %v", result["channel_code"])
	}

	if result["name"] != "Test Channel" {
		t.Errorf("期望 name 为 Test Channel, 实际为 %v", result["name"])
	}

	if result["type"] != "feishu" {
		t.Errorf("期望 type 为 feishu, 实际为 %v", result["type"])
	}
}
