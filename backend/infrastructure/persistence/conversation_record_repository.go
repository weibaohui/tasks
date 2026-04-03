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
		snap.Timestamp.UnixMilli(),
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
		snap.CreatedAt.UnixMilli(),
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
		Timestamp:        time.UnixMilli(timestampUnix),
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
		CreatedAt:        time.UnixMilli(createdAtUnix),
	})
	return record, nil
}

func (r *SQLiteConversationRecordRepository) GetStats(ctx context.Context, filter domain.ConversationStatsFilter) (*domain.ConversationStats, error) {
	whereBuilder := strings.Builder{}
	whereBuilder.WriteString(` WHERE 1=1`)
	args := make([]interface{}, 0)

	if filter.StartTime != nil {
		whereBuilder.WriteString(` AND timestamp >= ?`)
		args = append(args, filter.StartTime.UnixMilli())
	}
	if filter.EndTime != nil {
		whereBuilder.WriteString(` AND timestamp <= ?`)
		args = append(args, filter.EndTime.UnixMilli())
	}
	if len(filter.AgentCodes) > 0 {
		whereBuilder.WriteString(` AND agent_code IN (?` + strings.Repeat(",?", len(filter.AgentCodes)-1) + `)`)
		for _, code := range filter.AgentCodes {
			args = append(args, code)
		}
	}
	if len(filter.ChannelCodes) > 0 {
		whereBuilder.WriteString(` AND channel_code IN (?` + strings.Repeat(",?", len(filter.ChannelCodes)-1) + `)`)
		for _, code := range filter.ChannelCodes {
			args = append(args, code)
		}
	}
	if len(filter.Roles) > 0 {
		whereBuilder.WriteString(` AND role IN (?` + strings.Repeat(",?", len(filter.Roles)-1) + `)`)
		for _, role := range filter.Roles {
			args = append(args, role)
		}
	}

	whereClause := whereBuilder.String()

	var totalPromptTokens, totalCompletionTokens, totalTokens, totalRecords, totalSessions int
	var row *sql.Row

	row = r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(prompt_tokens), 0), COALESCE(SUM(completion_tokens), 0), COALESCE(SUM(total_tokens), 0), COUNT(*) FROM conversation_records`+whereClause, args...)
	err := row.Scan(&totalPromptTokens, &totalCompletionTokens, &totalTokens, &totalRecords)
	if err != nil {
		return nil, err
	}

	row = r.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT session_key) FROM conversation_records`+whereClause, args...)
	err = row.Scan(&totalSessions)
	if err != nil {
		return nil, err
	}

	dailyTrendsQuery := `SELECT date(timestamp/1000, 'unixepoch') as date, COALESCE(SUM(prompt_tokens), 0), COALESCE(SUM(completion_tokens), 0), COALESCE(SUM(total_tokens), 0) FROM conversation_records` + whereClause + ` GROUP BY date(timestamp/1000, 'unixepoch') ORDER BY date`
	dailyRows, err := r.db.QueryContext(ctx, dailyTrendsQuery, args...)
	if err != nil {
		return nil, err
	}
	defer dailyRows.Close()

	dailyTrends := make([]domain.DailyTokenTrend, 0)
	for dailyRows.Next() {
		var dt domain.DailyTokenTrend
		if err := dailyRows.Scan(&dt.Date, &dt.PromptTokens, &dt.CompletionTokens, &dt.TotalTokens); err != nil {
			return nil, err
		}
		dailyTrends = append(dailyTrends, dt)
	}

	agentRows, err := r.db.QueryContext(ctx, `SELECT agent_code, COUNT(*), COALESCE(SUM(total_tokens), 0) FROM conversation_records`+whereClause+` GROUP BY agent_code`, args...)
	if err != nil {
		return nil, err
	}
	defer agentRows.Close()

	agentDistribution := make([]domain.AgentStats, 0)
	for agentRows.Next() {
		var as domain.AgentStats
		if err := agentRows.Scan(&as.Code, &as.Count, &as.Tokens); err != nil {
			return nil, err
		}
		as.Name = as.Code
		agentDistribution = append(agentDistribution, as)
	}

	channelRows, err := r.db.QueryContext(ctx, `SELECT channel_type, COUNT(*) FROM conversation_records`+whereClause+` GROUP BY channel_type`, args...)
	if err != nil {
		return nil, err
	}
	defer channelRows.Close()

	channelDistribution := make([]domain.ChannelStats, 0)
	for channelRows.Next() {
		var cs domain.ChannelStats
		if err := channelRows.Scan(&cs.Type, &cs.Count); err != nil {
			return nil, err
		}
		channelDistribution = append(channelDistribution, cs)
	}

	roleRows, err := r.db.QueryContext(ctx, `SELECT role, COUNT(*) FROM conversation_records`+whereClause+` GROUP BY role`, args...)
	if err != nil {
		return nil, err
	}
	defer roleRows.Close()

	roleDistribution := make([]domain.RoleStats, 0)
	for roleRows.Next() {
		var rs domain.RoleStats
		if err := roleRows.Scan(&rs.Role, &rs.Count); err != nil {
			return nil, err
		}
		roleDistribution = append(roleDistribution, rs)
	}

	projectRows, err := r.db.QueryContext(ctx, `SELECT r.project_id, p.name, COALESCE(SUM(cr.total_tokens), 0) FROM conversation_records cr JOIN requirements r ON cr.trace_id = r.trace_id JOIN projects p ON r.project_id = p.id`+whereClause+` GROUP BY r.project_id, p.name ORDER BY COALESCE(SUM(cr.total_tokens), 0) DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer projectRows.Close()

	projectDistribution := make([]domain.ProjectStats, 0)
	for projectRows.Next() {
		var ps domain.ProjectStats
		if err := projectRows.Scan(&ps.ProjectID, &ps.Name, &ps.Tokens); err != nil {
			return nil, err
		}
		projectDistribution = append(projectDistribution, ps)
	}

	return &domain.ConversationStats{
		TotalPromptTokens:     totalPromptTokens,
		TotalCompletionTokens: totalCompletionTokens,
		TotalTokens:           totalTokens,
		DailyTrends:           dailyTrends,
		AgentDistribution:     agentDistribution,
		ChannelDistribution:   channelDistribution,
		RoleDistribution:      roleDistribution,
		ProjectDistribution:   projectDistribution,
		TotalSessions:         totalSessions,
		TotalRecords:          totalRecords,
	}, nil
}
