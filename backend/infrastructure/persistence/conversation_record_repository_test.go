/**
 * ConversationRecordRepository 集成测试
 */
package persistence

import (
	"context"
	"database/sql"
	"strconv"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/domain"
)

func setupConvRecordTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("无法连接数据库: %v", err)
	}

	err = InitSchema(db)
	if err != nil {
		t.Fatalf("初始化 Schema 失败: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func createTestRecord(id, traceID, eventType, role, content string) *domain.ConversationRecord {
	record, _ := domain.NewConversationRecord(
		domain.NewConversationRecordID(id),
		traceID,
		eventType,
	)
	record.SetSpan("span-1", "")
	record.SetMessage(role, content)
	record.SetScope("session-1", "user-1", "agent-1", "channel-1", "feishu")
	return record
}

func TestSQLiteConversationRecordRepository_SaveAndFind(t *testing.T) {
	db, cleanup := setupConvRecordTestDB(t)
	defer cleanup()

	repo := NewSQLiteConversationRecordRepository(db)
	ctx := context.Background()

	// 1. 保存记录
	record := createTestRecord("rec-1", "trace-1", "llm_call", "user", "你好")
	err := repo.Save(ctx, record)
	if err != nil {
		t.Fatalf("保存记录失败: %v", err)
	}

	// 2. 查找记录
	found, err := repo.FindByID(ctx, record.ID())
	if err != nil {
		t.Fatalf("查找记录失败: %v", err)
	}
	if found == nil {
		t.Fatal("未找到记录")
	}
	if found.ID() != record.ID() {
		t.Errorf("期望 ID 为 %s，实际为 %s", record.ID(), found.ID())
	}
	if found.TraceID() != "trace-1" {
		t.Errorf("期望 TraceID 为 trace-1，实际为 %s", found.TraceID())
	}
	if found.Content() != "你好" {
		t.Errorf("期望 Content 为 你好，实际为 %s", found.Content())
	}
}

func TestSQLiteConversationRecordRepository_FindByTraceID(t *testing.T) {
	db, cleanup := setupConvRecordTestDB(t)
	defer cleanup()

	repo := NewSQLiteConversationRecordRepository(db)
	ctx := context.Background()
	traceID := "trace-same"

	// 保存同一 trace 的多条记录
	for i := 0; i < 3; i++ {
		record := createTestRecord("rec-"+string(rune('a'+i)), traceID, "llm_call", "user", "msg")
		repo.Save(ctx, record)
	}

	// 保存不同 trace 的记录
	otherRecord := createTestRecord("rec-other", "trace-other", "llm_call", "user", "other")
	repo.Save(ctx, otherRecord)

	// 查询同一 trace 的记录
	records, err := repo.FindByTraceID(ctx, traceID, 100)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(records) != 3 {
		t.Errorf("期望 3 条记录，实际 %d 条", len(records))
	}

	// 验证所有记录的 traceID 相同
	for _, r := range records {
		if r.TraceID() != traceID {
			t.Errorf("期望 TraceID 为 %s，实际为 %s", traceID, r.TraceID())
		}
	}
}

func TestSQLiteConversationRecordRepository_FindBySessionKey(t *testing.T) {
	db, cleanup := setupConvRecordTestDB(t)
	defer cleanup()

	repo := NewSQLiteConversationRecordRepository(db)
	ctx := context.Background()
	sessionKey := "session-test"

	// 保存同一 session 的多条记录
	for i := 0; i < 2; i++ {
		record := createTestRecord("rec-"+string(rune('a'+i)), "trace-"+strconv.Itoa(i), "llm_call", "user", "msg")
		record.SetScope(sessionKey, "", "", "", "")
		repo.Save(ctx, record)
	}

	// 查询
	records, err := repo.FindBySessionKey(ctx, sessionKey, 100)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("期望 2 条记录，实际 %d 条", len(records))
	}
}

func TestSQLiteConversationRecordRepository_List(t *testing.T) {
	db, cleanup := setupConvRecordTestDB(t)
	defer cleanup()

	repo := NewSQLiteConversationRecordRepository(db)
	ctx := context.Background()

	// 创建多条不同 scope 的记录
	testCases := []struct {
		id, traceID, sessionKey, userCode, agentCode string
	}{
		{"1", "t1", "s1", "u1", "a1"},
		{"2", "t1", "s1", "u1", "a2"},
		{"3", "t2", "s2", "u1", "a1"},
		{"4", "t2", "s3", "u2", "a1"},
	}

	for _, tc := range testCases {
		record := createTestRecord("rec-"+tc.id, tc.traceID, "llm_call", "user", "msg")
		record.SetScope(tc.sessionKey, tc.userCode, tc.agentCode, "", "")
		repo.Save(ctx, record)
	}

	// 测试按 traceID 过滤
	filter := domain.ConversationRecordListFilter{TraceID: "t1"}
	records, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("期望 2 条记录，实际 %d 条", len(records))
	}

	// 测试按 agentCode 过滤
	filter = domain.ConversationRecordListFilter{AgentCode: "a1"}
	records, err = repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(records) != 3 {
		t.Errorf("期望 3 条记录，实际 %d 条", len(records))
	}

	// 测试按 sessionKey 过滤
	filter = domain.ConversationRecordListFilter{SessionKey: "s1"}
	records, err = repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("期望 2 条记录，实际 %d 条", len(records))
	}
}

func TestSQLiteConversationRecordRepository_Update(t *testing.T) {
	db, cleanup := setupConvRecordTestDB(t)
	defer cleanup()

	repo := NewSQLiteConversationRecordRepository(db)
	ctx := context.Background()

	// 保存记录
	record := createTestRecord("rec-1", "trace-1", "llm_call", "user", "original")
	repo.Save(ctx, record)

	// 更新记录
	record.SetMessage("assistant", "updated content")
	record.SetTokenUsage(100, 50, 150, 0, 0)
	err := repo.Save(ctx, record)
	if err != nil {
		t.Fatalf("更新记录失败: %v", err)
	}

	// 验证更新
	found, _ := repo.FindByID(ctx, record.ID())
	if found.Content() != "updated content" {
		t.Errorf("期望 Content 为 updated content，实际为 %s", found.Content())
	}
	if found.CompletionTokens() != 50 {
		t.Errorf("期望 CompletionTokens 为 50，实际为 %d", found.CompletionTokens())
	}
}
