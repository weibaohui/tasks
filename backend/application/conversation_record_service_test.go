package application

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type mockConversationRecordRepo struct {
	records map[string]*domain.ConversationRecord
	stats   *domain.ConversationStats
	errSave error
}

func newMockConversationRecordRepo() *mockConversationRecordRepo {
	return &mockConversationRecordRepo{
		records: make(map[string]*domain.ConversationRecord),
		stats: &domain.ConversationStats{
			TotalPromptTokens:     1000,
			TotalCompletionTokens: 500,
			TotalTokens:           1500,
			TotalSessions:         10,
			TotalRecords:          100,
		},
	}
}

func (m *mockConversationRecordRepo) Save(ctx context.Context, record *domain.ConversationRecord) error {
	if m.errSave != nil {
		return m.errSave
	}
	m.records[record.ID().String()] = record
	return nil
}

func (m *mockConversationRecordRepo) FindByID(ctx context.Context, id domain.ConversationRecordID) (*domain.ConversationRecord, error) {
	record, ok := m.records[id.String()]
	if !ok {
		return nil, nil
	}
	return record, nil
}

func (m *mockConversationRecordRepo) FindByTraceID(ctx context.Context, traceID string, limit int) ([]*domain.ConversationRecord, error) {
	var result []*domain.ConversationRecord
	for _, record := range m.records {
		if record.TraceID() == traceID {
			result = append(result, record)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockConversationRecordRepo) FindBySessionKey(ctx context.Context, sessionKey string, limit int) ([]*domain.ConversationRecord, error) {
	var result []*domain.ConversationRecord
	for _, record := range m.records {
		if record.SessionKey() == sessionKey {
			result = append(result, record)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockConversationRecordRepo) List(ctx context.Context, filter domain.ConversationRecordListFilter) ([]*domain.ConversationRecord, error) {
	var result []*domain.ConversationRecord
	for _, record := range m.records {
		// Apply filters
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

	// Apply pagination
	start := filter.Offset
	if start > len(result) {
		start = len(result)
	}
	end := start + filter.Limit
	if end > len(result) {
		end = len(result)
	}
	return result[start:end], nil
}

func (m *mockConversationRecordRepo) GetStats(ctx context.Context, filter domain.ConversationStatsFilter) (*domain.ConversationStats, error) {
	return m.stats, nil
}

type mockConversationRecordIDGen struct {
	count int
}

func (m *mockConversationRecordIDGen) Generate() string {
	m.count++
	return "conv-id-" + strconv.Itoa(m.count)
}

func setupTestConversationRecordSvc() (*ConversationRecordApplicationService, *mockConversationRecordRepo, *mockConversationRecordIDGen) {
	repo := newMockConversationRecordRepo()
	idGen := &mockConversationRecordIDGen{}

	svc := NewConversationRecordApplicationService(repo, idGen)
	return svc, repo, idGen
}

func TestConversationRecordService_CreateRecord(t *testing.T) {
	svc, repo, idGen := setupTestConversationRecordSvc()
	ctx := context.Background()

	timestamp := time.Now()
	record, err := svc.CreateRecord(ctx, CreateConversationRecordCommand{
		TraceID:          "trace-001",
		SpanID:           "span-001",
		ParentSpanID:     "parent-001",
		EventType:        "message",
		Timestamp:        &timestamp,
		SessionKey:       "session-001",
		Role:             "user",
		Content:          "Hello, world!",
		PromptTokens:     10,
		CompletionTokens: 5,
		TotalTokens:      15,
		ReasoningTokens:  2,
		CachedTokens:     1,
		UserCode:         "user-001",
		AgentCode:        "agent-001",
		ChannelCode:      "channel-001",
		ChannelType:      "web",
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	// Verify ID was generated
	expectedID := "conv-id-1"
	if record.ID().String() != expectedID {
		t.Errorf("期望 id 为 '%s', 实际为 '%s'", expectedID, record.ID().String())
	}

	if record.TraceID() != "trace-001" {
		t.Errorf("期望 traceID 为 'trace-001', 实际为 '%s'", record.TraceID())
	}

	if record.SpanID() != "span-001" {
		t.Errorf("期望 spanID 为 'span-001', 实际为 '%s'", record.SpanID())
	}

	if record.ParentSpanID() != "parent-001" {
		t.Errorf("期望 parentSpanID 为 'parent-001', 实际为 '%s'", record.ParentSpanID())
	}

	if record.EventType() != "message" {
		t.Errorf("期望 eventType 为 'message', 实际为 '%s'", record.EventType())
	}

	if record.SessionKey() != "session-001" {
		t.Errorf("期望 sessionKey 为 'session-001', 实际为 '%s'", record.SessionKey())
	}

	if record.Role() != "user" {
		t.Errorf("期望 role 为 'user', 实际为 '%s'", record.Role())
	}

	if record.Content() != "Hello, world!" {
		t.Errorf("期望 content 为 'Hello, world!', 实际为 '%s'", record.Content())
	}

	if record.PromptTokens() != 10 {
		t.Errorf("期望 promptTokens 为 10, 实际为 %d", record.PromptTokens())
	}

	if record.CompletionTokens() != 5 {
		t.Errorf("期望 completionTokens 为 5, 实际为 %d", record.CompletionTokens())
	}

	if record.TotalTokens() != 15 {
		t.Errorf("期望 totalTokens 为 15, 实际为 %d", record.TotalTokens())
	}

	if record.ReasoningTokens() != 2 {
		t.Errorf("期望 reasoningTokens 为 2, 实际为 %d", record.ReasoningTokens())
	}

	if record.CachedTokens() != 1 {
		t.Errorf("期望 cachedTokens 为 1, 实际为 %d", record.CachedTokens())
	}

	if record.UserCode() != "user-001" {
		t.Errorf("期望 userCode 为 'user-001', 实际为 '%s'", record.UserCode())
	}

	if record.AgentCode() != "agent-001" {
		t.Errorf("期望 agentCode 为 'agent-001', 实际为 '%s'", record.AgentCode())
	}

	if record.ChannelCode() != "channel-001" {
		t.Errorf("期望 channelCode 为 'channel-001', 实际为 '%s'", record.ChannelCode())
	}

	if record.ChannelType() != "web" {
		t.Errorf("期望 channelType 为 'web', 实际为 '%s'", record.ChannelType())
	}

	// Verify the record was saved
	if len(repo.records) != 1 {
		t.Errorf("期望 repo 中有 1 条记录, 实际为 %d", len(repo.records))
	}

	if idGen.count != 1 {
		t.Errorf("期望 idGen count 为 1, 实际为 %d", idGen.count)
	}
}

func TestConversationRecordService_CreateRecord_WithoutTimestamp(t *testing.T) {
	svc, repo, _ := setupTestConversationRecordSvc()
	ctx := context.Background()

	record, err := svc.CreateRecord(ctx, CreateConversationRecordCommand{
		TraceID:   "trace-002",
		EventType: "message",
		Role:      "assistant",
		Content:   "Response without timestamp",
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if record.ID().String() != "conv-id-1" {
		t.Errorf("期望 id 为 'conv-id-1', 实际为 '%s'", record.ID().String())
	}

	// Timestamp should be set to current time by default
	if record.Timestamp().IsZero() {
		t.Error("期望 timestamp 不为零")
	}

	if len(repo.records) != 1 {
		t.Errorf("期望 repo 中有 1 条记录, 实际为 %d", len(repo.records))
	}
}

func TestConversationRecordService_CreateRecord_ValidationError(t *testing.T) {
	svc, _, _ := setupTestConversationRecordSvc()
	ctx := context.Background()

	// Test empty traceID
	_, err := svc.CreateRecord(ctx, CreateConversationRecordCommand{
		TraceID:   "",
		EventType: "message",
	})
	if err == nil {
		t.Error("期望 traceID 为空时返回错误, 实际为 nil")
	}

	// Test empty eventType
	_, err = svc.CreateRecord(ctx, CreateConversationRecordCommand{
		TraceID:   "trace-003",
		EventType: "",
	})
	if err == nil {
		t.Error("期望 eventType 为空时返回错误, 实际为 nil")
	}
}

func TestConversationRecordService_CreateRecord_SaveError(t *testing.T) {
	svc, repo, _ := setupTestConversationRecordSvc()
	ctx := context.Background()

	repo.errSave = errors.New("database error")

	_, err := svc.CreateRecord(ctx, CreateConversationRecordCommand{
		TraceID:   "trace-error",
		EventType: "message",
	})

	if err == nil {
		t.Fatal("期望有错误, 实际为 nil")
	}

	if !errors.Is(err, repo.errSave) {
		t.Errorf("期望错误包含 'database error', 实际为 %v", err)
	}
}

func TestConversationRecordService_GetRecord(t *testing.T) {
	svc, repo, _ := setupTestConversationRecordSvc()
	ctx := context.Background()

	// Create a record first
	record, _ := domain.NewConversationRecord(
		domain.NewConversationRecordID("conv-get-001"),
		"trace-get",
		"message",
	)
	record.SetMessage("user", "Test content")
	repo.records["conv-get-001"] = record

	// Get the record
	found, err := svc.GetRecord(ctx, "conv-get-001")
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if found == nil {
		t.Fatal("期望找到记录, 实际为 nil")
	}

	if found.ID().String() != "conv-get-001" {
		t.Errorf("期望 id 为 'conv-get-001', 实际为 '%s'", found.ID().String())
	}

	if found.Content() != "Test content" {
		t.Errorf("期望 content 为 'Test content', 实际为 '%s'", found.Content())
	}
}

func TestConversationRecordService_GetRecord_NotFound(t *testing.T) {
	svc, _, _ := setupTestConversationRecordSvc()
	ctx := context.Background()

	_, err := svc.GetRecord(ctx, "non-existent-id")
	if err != ErrConversationRecordNotFound {
		t.Errorf("期望 ErrConversationRecordNotFound, 实际为 %v", err)
	}
}

func TestConversationRecordService_ListRecords(t *testing.T) {
	svc, repo, _ := setupTestConversationRecordSvc()
	ctx := context.Background()

	// Create multiple records
	for i := 1; i <= 5; i++ {
		record, _ := domain.NewConversationRecord(
			domain.NewConversationRecordID("conv-list-"+strconv.Itoa(i)),
			"trace-list",
			"message",
		)
		record.SetScope("session-001", "user-001", "agent-001", "channel-001", "web")
		record.SetMessage("user", "Content "+strconv.Itoa(i))
		repo.records[record.ID().String()] = record
	}

	// Test list all
	records, err := svc.ListRecords(ctx, ListConversationRecordsQuery{})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if len(records) != 5 {
		t.Errorf("期望 5 条记录, 实际为 %d", len(records))
	}
}

func TestConversationRecordService_ListRecords_WithFilters(t *testing.T) {
	svc, repo, _ := setupTestConversationRecordSvc()
	ctx := context.Background()

	// Create records with different attributes
	record1, _ := domain.NewConversationRecord(
		domain.NewConversationRecordID("conv-filter-1"),
		"trace-1",
		"message",
	)
	record1.SetScope("session-1", "user-1", "agent-1", "channel-1", "web")
	record1.SetMessage("user", "Content 1")
	repo.records["conv-filter-1"] = record1

	record2, _ := domain.NewConversationRecord(
		domain.NewConversationRecordID("conv-filter-2"),
		"trace-2",
		"event",
	)
	record2.SetScope("session-2", "user-2", "agent-2", "channel-2", "mobile")
	record2.SetMessage("assistant", "Content 2")
	repo.records["conv-filter-2"] = record2

	// Test filter by TraceID
	records, err := svc.ListRecords(ctx, ListConversationRecordsQuery{
		TraceID: "trace-1",
	})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if len(records) != 1 {
		t.Errorf("期望 1 条记录, 实际为 %d", len(records))
	}

	// Test filter by SessionKey
	records, err = svc.ListRecords(ctx, ListConversationRecordsQuery{
		SessionKey: "session-2",
	})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if len(records) != 1 {
		t.Errorf("期望 1 条记录, 实际为 %d", len(records))
	}

	// Test filter by AgentCode
	records, err = svc.ListRecords(ctx, ListConversationRecordsQuery{
		AgentCode: "agent-1",
	})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if len(records) != 1 {
		t.Errorf("期望 1 条记录, 实际为 %d", len(records))
	}

	// Test filter by EventType
	records, err = svc.ListRecords(ctx, ListConversationRecordsQuery{
		EventType: "event",
	})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if len(records) != 1 {
		t.Errorf("期望 1 条记录, 实际为 %d", len(records))
	}

	// Test filter by Role
	records, err = svc.ListRecords(ctx, ListConversationRecordsQuery{
		Role: "assistant",
	})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if len(records) != 1 {
		t.Errorf("期望 1 条记录, 实际为 %d", len(records))
	}
}

func TestConversationRecordService_ListRecords_WithPagination(t *testing.T) {
	svc, repo, _ := setupTestConversationRecordSvc()
	ctx := context.Background()

	// Create 10 records
	for i := 1; i <= 10; i++ {
		record, _ := domain.NewConversationRecord(
			domain.NewConversationRecordID("conv-page-"+strconv.Itoa(i)),
			"trace-page",
			"message",
		)
		repo.records[record.ID().String()] = record
	}

	// Test with limit
	records, err := svc.ListRecords(ctx, ListConversationRecordsQuery{
		Limit: 3,
	})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if len(records) != 3 {
		t.Errorf("期望 3 条记录, 实际为 %d", len(records))
	}

	// Test with limit and offset
	records, err = svc.ListRecords(ctx, ListConversationRecordsQuery{
		Limit:  3,
		Offset: 5,
	})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if len(records) != 3 {
		t.Errorf("期望 3 条记录, 实际为 %d", len(records))
	}

	// Test with default limit when limit is 0
	records, err = svc.ListRecords(ctx, ListConversationRecordsQuery{
		Limit: 0,
	})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	// Should use default limit of 100
	if len(records) != 10 {
		t.Errorf("期望 10 条记录 (全部), 实际为 %d", len(records))
	}

	// Test max limit enforcement
	records, err = svc.ListRecords(ctx, ListConversationRecordsQuery{
		Limit: 1000, // Exceeds max limit of 500
	})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	// All 10 records should be returned since we're under max limit
	if len(records) != 10 {
		t.Errorf("期望 10 条记录, 实际为 %d", len(records))
	}
}

func TestConversationRecordService_ListRecords_NegativeOffset(t *testing.T) {
	svc, repo, _ := setupTestConversationRecordSvc()
	ctx := context.Background()

	// Create records
	for i := 1; i <= 3; i++ {
		record, _ := domain.NewConversationRecord(
			domain.NewConversationRecordID("conv-neg-"+strconv.Itoa(i)),
			"trace-neg",
			"message",
		)
		repo.records[record.ID().String()] = record
	}

	// Test negative offset (should be treated as 0)
	records, err := svc.ListRecords(ctx, ListConversationRecordsQuery{
		Limit:  10,
		Offset: -5,
	})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if len(records) != 3 {
		t.Errorf("期望 3 条记录, 实际为 %d", len(records))
	}
}

func TestConversationRecordService_GetStats(t *testing.T) {
	svc, _, _ := setupTestConversationRecordSvc()
	ctx := context.Background()

	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	stats, err := svc.GetStats(ctx, GetConversationStatsQuery{
		StartTime:    &startTime,
		EndTime:      &endTime,
		AgentCodes:   []string{"agent-1", "agent-2"},
		ChannelCodes: []string{"channel-1"},
		Roles:        []string{"user", "assistant"},
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if stats == nil {
		t.Fatal("期望 stats 不为 nil")
	}

	if stats.TotalPromptTokens != 1000 {
		t.Errorf("期望 TotalPromptTokens 为 1000, 实际为 %d", stats.TotalPromptTokens)
	}

	if stats.TotalCompletionTokens != 500 {
		t.Errorf("期望 TotalCompletionTokens 为 500, 实际为 %d", stats.TotalCompletionTokens)
	}

	if stats.TotalTokens != 1500 {
		t.Errorf("期望 TotalTokens 为 1500, 实际为 %d", stats.TotalTokens)
	}

	if stats.TotalSessions != 10 {
		t.Errorf("期望 TotalSessions 为 10, 实际为 %d", stats.TotalSessions)
	}

	if stats.TotalRecords != 100 {
		t.Errorf("期望 TotalRecords 为 100, 实际为 %d", stats.TotalRecords)
	}
}

func TestConversationRecordService_GetStats_EmptyFilter(t *testing.T) {
	svc, _, _ := setupTestConversationRecordSvc()
	ctx := context.Background()

	stats, err := svc.GetStats(ctx, GetConversationStatsQuery{})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if stats == nil {
		t.Fatal("期望 stats 不为 nil")
	}

	if stats.TotalRecords != 100 {
		t.Errorf("期望 TotalRecords 为 100, 实际为 %d", stats.TotalRecords)
	}
}

func TestConversationRecordService_CreateRecord_Multiple(t *testing.T) {
	svc, repo, idGen := setupTestConversationRecordSvc()
	ctx := context.Background()

	// Create multiple records
	for i := 1; i <= 3; i++ {
		_, err := svc.CreateRecord(ctx, CreateConversationRecordCommand{
			TraceID:   "trace-multi-" + strconv.Itoa(i),
			EventType: "message",
			Role:      "user",
			Content:   "Message " + strconv.Itoa(i),
		})
		if err != nil {
			t.Fatalf("创建第 %d 条记录时期望无错误, 实际为 %v", i, err)
		}
	}

	// Verify all records were saved
	if len(repo.records) != 3 {
		t.Errorf("期望 repo 中有 3 条记录, 实际为 %d", len(repo.records))
	}

	// Verify ID generator was called correctly
	if idGen.count != 3 {
		t.Errorf("期望 idGen count 为 3, 实际为 %d", idGen.count)
	}

	// Verify each record has unique ID
	ids := make(map[string]bool)
	for _, record := range repo.records {
		if ids[record.ID().String()] {
			t.Errorf("发现重复的 ID: %s", record.ID().String())
		}
		ids[record.ID().String()] = true
	}
}

func TestConversationRecordService_NewService(t *testing.T) {
	repo := newMockConversationRecordRepo()
	idGen := &mockConversationRecordIDGen{}

	svc := NewConversationRecordApplicationService(repo, idGen)

	if svc == nil {
		t.Fatal("期望 svc 不为 nil")
	}

	// Test that the service was initialized with correct dependencies
	// by creating a record
	ctx := context.Background()
	record, err := svc.CreateRecord(ctx, CreateConversationRecordCommand{
		TraceID:   "test-trace",
		EventType: "test-event",
	})

	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}

	if record.ID().String() != "conv-id-1" {
		t.Errorf("期望 id 为 'conv-id-1', 实际为 '%s'", record.ID().String())
	}
}

func TestConversationRecordService_ListRecords_FilterCombinations(t *testing.T) {
	svc, repo, _ := setupTestConversationRecordSvc()
	ctx := context.Background()

	// Create records with various combinations
	record1, _ := domain.NewConversationRecord(
		domain.NewConversationRecordID("conv-combo-1"),
		"trace-a",
		"message",
	)
	record1.SetScope("session-a", "user-a", "agent-a", "channel-a", "web")
	record1.SetMessage("user", "Content A")
	repo.records["conv-combo-1"] = record1

	record2, _ := domain.NewConversationRecord(
		domain.NewConversationRecordID("conv-combo-2"),
		"trace-a",
		"message",
	)
	record2.SetScope("session-a", "user-b", "agent-a", "channel-b", "mobile")
	record2.SetMessage("assistant", "Content B")
	repo.records["conv-combo-2"] = record2

	record3, _ := domain.NewConversationRecord(
		domain.NewConversationRecordID("conv-combo-3"),
		"trace-b",
		"event",
	)
	record3.SetScope("session-b", "user-a", "agent-b", "channel-a", "web")
	record3.SetMessage("user", "Content C")
	repo.records["conv-combo-3"] = record3

	// Test combination: TraceID + AgentCode
	records, err := svc.ListRecords(ctx, ListConversationRecordsQuery{
		TraceID:   "trace-a",
		AgentCode: "agent-a",
	})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if len(records) != 2 {
		t.Errorf("期望 2 条记录, 实际为 %d", len(records))
	}

	// Test combination: UserCode + Role
	records, err = svc.ListRecords(ctx, ListConversationRecordsQuery{
		UserCode: "user-a",
		Role:     "user",
	})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	// record1 (user-a + user) and record3 (user-a + user) both match
	if len(records) != 2 {
		t.Errorf("期望 2 条记录, 实际为 %d", len(records))
	}

	// Test combination: all filters (no match expected)
	records, err = svc.ListRecords(ctx, ListConversationRecordsQuery{
		TraceID:     "trace-a",
		SessionKey:  "session-b",
		UserCode:    "user-c",
		AgentCode:   "agent-c",
		ChannelCode: "channel-c",
		EventType:   "other",
		Role:        "system",
	})
	if err != nil {
		t.Fatalf("期望无错误, 实际为 %v", err)
	}
	if len(records) != 0 {
		t.Errorf("期望 0 条记录, 实际为 %d", len(records))
	}
}
