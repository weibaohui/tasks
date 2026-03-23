package persistence

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteConversationRecordRepository struct {
	db *sql.DB
}

func NewSQLiteConversationRecordRepository(db *sql.DB) *SQLiteConversationRecordRepository {
	return &SQLiteConversationRecordRepository{db: db}
}

func (r *SQLiteConversationRecordRepository) Save(ctx context.Context, record *domain.ConversationRecord) error {
	snap := record.ToSnapshot()
	query := `
		INSERT INTO conversation_records (
			id, trace_id, span_id, parent_span_id, event_type, timestamp, session_key, role, content,
			prompt_tokens, completion_tokens, total_tokens, reasoning_tokens, cached_tokens,
			user_code, agent_code, channel_code, channel_type, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			trace_id=excluded.trace_id,
			span_id=excluded.span_id,
			parent_span_id=excluded.parent_span_id,
			event_type=excluded.event_type,
			timestamp=excluded.timestamp,
			session_key=excluded.session_key,
			role=excluded.role,
			content=excluded.content,
			prompt_tokens=excluded.prompt_tokens,
			completion_tokens=excluded.completion_tokens,
			total_tokens=excluded.total_tokens,
			reasoning_tokens=excluded.reasoning_tokens,
			cached_tokens=excluded.cached_tokens,
			user_code=excluded.user_code,
			agent_code=excluded.agent_code,
			channel_code=excluded.channel_code,
			channel_type=excluded.channel_type
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		snap.ID.String(),
		snap.TraceID,
		snap.SpanID,
		snap.ParentSpanID,
		snap.EventType,
		snap.Timestamp.Unix(),
		snap.SessionKey,
		snap.Role,
		snap.Content,
		snap.PromptTokens,
		snap.CompletionTokens,
		snap.TotalTokens,
		snap.ReasoningTokens,
		snap.CachedTokens,
		snap.UserCode,
		snap.AgentCode,
		snap.ChannelCode,
		snap.ChannelType,
		snap.CreatedAt.Unix(),
	)
	return err
}

func (r *SQLiteConversationRecordRepository) FindByID(ctx context.Context, id domain.ConversationRecordID) (*domain.ConversationRecord, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, trace_id, span_id, parent_span_id, event_type, timestamp, session_key, role, content, prompt_tokens, completion_tokens, total_tokens, reasoning_tokens, cached_tokens, user_code, agent_code, channel_code, channel_type, created_at FROM conversation_records WHERE id = ?`,
		id.String(),
	)
	return scanConversationRecord(row)
}

func (r *SQLiteConversationRecordRepository) FindByTraceID(ctx context.Context, traceID string, limit int) ([]*domain.ConversationRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, trace_id, span_id, parent_span_id, event_type, timestamp, session_key, role, content, prompt_tokens, completion_tokens, total_tokens, reasoning_tokens, cached_tokens, user_code, agent_code, channel_code, channel_type, created_at FROM conversation_records WHERE trace_id = ? ORDER BY timestamp DESC LIMIT ?`,
		traceID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConversationRecords(rows)
}

func (r *SQLiteConversationRecordRepository) FindBySessionKey(ctx context.Context, sessionKey string, limit int) ([]*domain.ConversationRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, trace_id, span_id, parent_span_id, event_type, timestamp, session_key, role, content, prompt_tokens, completion_tokens, total_tokens, reasoning_tokens, cached_tokens, user_code, agent_code, channel_code, channel_type, created_at FROM conversation_records WHERE session_key = ? ORDER BY timestamp DESC LIMIT ?`,
		sessionKey,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConversationRecords(rows)
}

func (r *SQLiteConversationRecordRepository) List(ctx context.Context, filter domain.ConversationRecordListFilter) ([]*domain.ConversationRecord, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`SELECT id, trace_id, span_id, parent_span_id, event_type, timestamp, session_key, role, content, prompt_tokens, completion_tokens, total_tokens, reasoning_tokens, cached_tokens, user_code, agent_code, channel_code, channel_type, created_at FROM conversation_records WHERE 1=1`)
	args := make([]interface{}, 0, 9)

	if filter.TraceID != "" {
		queryBuilder.WriteString(` AND trace_id = ?`)
		args = append(args, filter.TraceID)
	}
	if filter.SessionKey != "" {
		queryBuilder.WriteString(` AND session_key = ?`)
		args = append(args, filter.SessionKey)
	}
	if filter.UserCode != "" {
		queryBuilder.WriteString(` AND user_code = ?`)
		args = append(args, filter.UserCode)
	}
	if filter.AgentCode != "" {
		queryBuilder.WriteString(` AND agent_code = ?`)
		args = append(args, filter.AgentCode)
	}
	if filter.ChannelCode != "" {
		queryBuilder.WriteString(` AND channel_code = ?`)
		args = append(args, filter.ChannelCode)
	}
	if filter.EventType != "" {
		queryBuilder.WriteString(` AND event_type = ?`)
		args = append(args, filter.EventType)
	}
	if filter.Role != "" {
		queryBuilder.WriteString(` AND role = ?`)
		args = append(args, filter.Role)
	}

	queryBuilder.WriteString(` ORDER BY timestamp DESC LIMIT ? OFFSET ?`)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConversationRecords(rows)
}

func scanConversationRecords(rows *sql.Rows) ([]*domain.ConversationRecord, error) {
	records := make([]*domain.ConversationRecord, 0)
	for rows.Next() {
		record, err := scanConversationRecord(rows)
		if err != nil {
			return nil, err
		}
		if record != nil {
			records = append(records, record)
		}
	}
	return records, rows.Err()
}

func scanConversationRecord(scanner rowScanner) (*domain.ConversationRecord, error) {
	var (
		idStr            string
		traceID          string
		spanID           string
		parentSpanID     string
		eventType        string
		timestampUnix    int64
		sessionKey       string
		role             string
		content          string
		promptTokens     int
		completionTokens int
		totalTokens      int
		reasoningTokens  int
		cachedTokens     int
		userCode         string
		agentCode        string
		channelCode      string
		channelType      string
		createdAtUnix    int64
	)

	err := scanner.Scan(
		&idStr,
		&traceID,
		&spanID,
		&parentSpanID,
		&eventType,
		&timestampUnix,
		&sessionKey,
		&role,
		&content,
		&promptTokens,
		&completionTokens,
		&totalTokens,
		&reasoningTokens,
		&cachedTokens,
		&userCode,
		&agentCode,
		&channelCode,
		&channelType,
		&createdAtUnix,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	record := &domain.ConversationRecord{}
	record.FromSnapshot(domain.ConversationRecordSnapshot{
		ID:               domain.NewConversationRecordID(idStr),
		TraceID:          traceID,
		SpanID:           spanID,
		ParentSpanID:     parentSpanID,
		EventType:        eventType,
		Timestamp:        time.Unix(timestampUnix, 0),
		SessionKey:       sessionKey,
		Role:             role,
		Content:          content,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		ReasoningTokens:  reasoningTokens,
		CachedTokens:     cachedTokens,
		UserCode:         userCode,
		AgentCode:        agentCode,
		ChannelCode:      channelCode,
		ChannelType:      channelType,
		CreatedAt:        time.Unix(createdAtUnix, 0),
	})
	return record, nil
}
