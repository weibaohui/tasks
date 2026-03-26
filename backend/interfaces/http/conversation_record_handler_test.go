/**
 * Conversation Record Handler 测试
 */
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type mockConvRecordRepo struct {
	records map[string]*domain.ConversationRecord
}

func newMockConvRecordRepo() *mockConvRecordRepo {
	return &mockConvRecordRepo{
		records: make(map[string]*domain.ConversationRecord),
	}
}

func (m *mockConvRecordRepo) Save(ctx context.Context, record *domain.ConversationRecord) error {
	m.records[record.ID().String()] = record
	return nil
}

func (m *mockConvRecordRepo) FindByID(ctx context.Context, id domain.ConversationRecordID) (*domain.ConversationRecord, error) {
	record, ok := m.records[id.String()]
	if !ok {
		return nil, nil
	}
	return record, nil
}

func (m *mockConvRecordRepo) FindByTraceID(ctx context.Context, traceID string, limit int) ([]*domain.ConversationRecord, error) {
	var result []*domain.ConversationRecord
	for _, record := range m.records {
		if record.TraceID() == traceID {
			result = append(result, record)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockConvRecordRepo) FindBySessionKey(ctx context.Context, sessionKey string, limit int) ([]*domain.ConversationRecord, error) {
	var result []*domain.ConversationRecord
	for _, record := range m.records {
		if record.SessionKey() == sessionKey {
			result = append(result, record)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockConvRecordRepo) List(ctx context.Context, filter domain.ConversationRecordListFilter) ([]*domain.ConversationRecord, error) {
	var result []*domain.ConversationRecord
	for _, record := range m.records {
		if filter.TraceID != "" && record.TraceID() != filter.TraceID {
			continue
		}
		if filter.SessionKey != "" && record.SessionKey() != filter.SessionKey {
			continue
		}
		if filter.UserCode != "" && record.UserCode() != filter.UserCode {
			continue
		}
		if filter.AgentCode != "" && record.AgentCode() != filter.AgentCode {
			continue
		}
		if filter.ChannelCode != "" && record.ChannelCode() != filter.ChannelCode {
			continue
		}
		if filter.EventType != "" && record.EventType() != filter.EventType {
			continue
		}
		if filter.Role != "" && record.Role() != filter.Role {
			continue
		}
		result = append(result, record)
	}

	// Apply offset and limit
	if filter.Offset > 0 {
		if filter.Offset >= len(result) {
			return []*domain.ConversationRecord{}, nil
		}
		result = result[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}

	return result, nil
}

func (m *mockConvRecordRepo) GetStats(ctx context.Context, filter domain.ConversationStatsFilter) (*domain.ConversationStats, error) {
	return &domain.ConversationStats{
		TotalPromptTokens:     0,
		TotalCompletionTokens: 0,
		TotalTokens:           0,
		DailyTrends:           []domain.DailyTokenTrend{},
		AgentDistribution:     []domain.AgentStats{},
		ChannelDistribution:   []domain.ChannelStats{},
		RoleDistribution:      []domain.RoleStats{},
		TotalSessions:         0,
		TotalRecords:          len(m.records),
	}, nil
}

type mockConvRecordIDGen struct {
	count int
}

func (m *mockConvRecordIDGen) Generate() string {
	m.count++
	return "conv-id-" + strconv.Itoa(m.count)
}

func setupTestConvRecordHandler() (*ConversationRecordHandler, *mockConvRecordRepo) {
	repo := newMockConvRecordRepo()
	idGen := &mockConvRecordIDGen{}
	recordService := application.NewConversationRecordApplicationService(repo, idGen)
	handler := NewConversationRecordHandler(recordService)
	return handler, repo
}

func setupConvRecordMux(handler *ConversationRecordHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/conversation-records", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handler.CreateRecord(w, r)
		case http.MethodGet:
			handler.ListRecords(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v1/conversation-records/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.GetRecord(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	return mux
}

func TestCreateConversationRecord(t *testing.T) {
	handler, _ := setupTestConvRecordHandler()
	mux := setupConvRecordMux(handler)

	body := `{
		"trace_id": "trace-123",
		"span_id": "span-456",
		"parent_span_id": "span-000",
		"event_type": "llm_call",
		"timestamp": 1699999999000,
		"session_key": "session-abc",
		"role": "assistant",
		"content": "Hello, world!",
		"prompt_tokens": 100,
		"completion_tokens": 50,
		"total_tokens": 150,
		"user_code": "usr_001",
		"agent_code": "agt_001",
		"channel_code": "ch_001",
		"channel_type": "feishu"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversation-records", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusCreated, w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["id"] == nil || resp["id"] == "" {
		t.Error("响应中应该包含 id")
	}

	if resp["trace_id"] != "trace-123" {
		t.Errorf("期望 trace_id 为 'trace-123', 实际为 '%v'", resp["trace_id"])
	}

	if resp["event_type"] != "llm_call" {
		t.Errorf("期望 event_type 为 'llm_call', 实际为 '%v'", resp["event_type"])
	}

	if resp["user_code"] != "usr_001" {
		t.Errorf("期望 user_code 为 'usr_001', 实际为 '%v'", resp["user_code"])
	}

	if resp["agent_code"] != "agt_001" {
		t.Errorf("期望 agent_code 为 'agt_001', 实际为 '%v'", resp["agent_code"])
	}
}

func TestCreateConversationRecord_InvalidJSON(t *testing.T) {
	handler, _ := setupTestConvRecordHandler()
	mux := setupConvRecordMux(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversation-records", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestListConversationRecords(t *testing.T) {
	handler, repo := setupTestConvRecordHandler()
	mux := setupConvRecordMux(handler)
	ctx := context.Background()

	// 先创建一些记录
	record, _ := domain.NewConversationRecord(
		domain.NewConversationRecordID("conv-1"),
		"trace-123",
		"llm_call",
	)
	record.SetScope("session-abc", "usr_001", "agt_001", "ch_001", "feishu")
	repo.Save(ctx, record)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversation-records?trace_id=trace-123", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, w.Code)
	}

	var resp []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp) < 1 {
		t.Error("期望至少有一条记录")
	}
}

func TestListConversationRecords_WithFilters(t *testing.T) {
	handler, repo := setupTestConvRecordHandler()
	mux := setupConvRecordMux(handler)
	ctx := context.Background()

	record1, _ := domain.NewConversationRecord(
		domain.NewConversationRecordID("conv-1"),
		"trace-123",
		"llm_call",
	)
	record1.SetScope("session-abc", "usr_001", "agt_001", "", "")
	repo.Save(ctx, record1)

	record2, _ := domain.NewConversationRecord(
		domain.NewConversationRecordID("conv-2"),
		"trace-123",
		"tool_call",
	)
	record2.SetScope("session-abc", "usr_001", "agt_001", "", "")
	repo.Save(ctx, record2)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversation-records?user_code=usr_001&event_type=llm_call", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, w.Code)
	}

	var resp []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp) != 1 {
		t.Errorf("期望 1 条记录, 实际为 %d", len(resp))
	}

	if resp[0]["event_type"] != "llm_call" {
		t.Errorf("期望 event_type 为 'llm_call', 实际为 '%v'", resp[0]["event_type"])
	}
}

func TestGetConversationRecord(t *testing.T) {
	handler, repo := setupTestConvRecordHandler()
	mux := setupConvRecordMux(handler)
	ctx := context.Background()

	record, _ := domain.NewConversationRecord(
		domain.NewConversationRecordID("conv-1"),
		"trace-123",
		"llm_call",
	)
	record.SetScope("session-abc", "usr_001", "agt_001", "", "")
	repo.Save(ctx, record)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversation-records/?id=conv-1", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["id"] != "conv-1" {
		t.Errorf("期望 id 为 'conv-1', 实际为 '%v'", resp["id"])
	}
}

func TestGetConversationRecord_NotFound(t *testing.T) {
	handler, _ := setupTestConvRecordHandler()
	mux := setupConvRecordMux(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversation-records/?id=non-existent", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusNotFound, w.Code)
	}
}

func TestGetConversationRecord_MissingID(t *testing.T) {
	handler, _ := setupTestConvRecordHandler()
	mux := setupConvRecordMux(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversation-records/", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}
}

func TestConversationRecordToMap(t *testing.T) {
	_, repo := setupTestConvRecordHandler()
	ctx := context.Background()

	record, _ := domain.NewConversationRecord(
		domain.NewConversationRecordID("conv-1"),
		"trace-123",
		"llm_call",
	)
	record.SetSpan("span-1", "span-0")
	record.SetScope("session-abc", "usr_001", "agt_001", "ch_001", "feishu")
	record.SetMessage("assistant", "Hello")
	record.SetTokenUsage(100, 50, 150, 0, 0)
	repo.Save(ctx, record)

	m := conversationRecordToMap(record)

	if m["id"] != "conv-1" {
		t.Errorf("期望 id 为 'conv-1', 实际为 '%v'", m["id"])
	}

	if m["trace_id"] != "trace-123" {
		t.Errorf("期望 trace_id 为 'trace-123', 实际为 '%v'", m["trace_id"])
	}

	if m["session_key"] != "session-abc" {
		t.Errorf("期望 session_key 为 'session-abc', 实际为 '%v'", m["session_key"])
	}

	if m["user_code"] != "usr_001" {
		t.Errorf("期望 user_code 为 'usr_001', 实际为 '%v'", m["user_code"])
	}

	if m["agent_code"] != "agt_001" {
		t.Errorf("期望 agent_code 为 'agt_001', 实际为 '%v'", m["agent_code"])
	}

	if m["prompt_tokens"] != 100 {
		t.Errorf("期望 prompt_tokens 为 100, 实际为 %v", m["prompt_tokens"])
	}

	if m["total_tokens"] != 150 {
		t.Errorf("期望 total_tokens 为 150, 实际为 %v", m["total_tokens"])
	}
}