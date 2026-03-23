package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteSessionRepository struct {
	db *sql.DB
}

func NewSQLiteSessionRepository(db *sql.DB) *SQLiteSessionRepository {
	return &SQLiteSessionRepository{db: db}
}

func (r *SQLiteSessionRepository) Save(ctx context.Context, session *domain.Session) error {
	snap := session.ToSnapshot()
	metadataJSON, _ := json.Marshal(snap.Metadata)
	var lastActiveUnix *int64
	if snap.LastActive != nil {
		v := snap.LastActive.Unix()
		lastActiveUnix = &v
	}

	query := `
		INSERT INTO sessions (
			id, user_code, agent_code, channel_code, session_key, external_id, last_active_at, metadata, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			agent_code=excluded.agent_code,
			channel_code=excluded.channel_code,
			external_id=excluded.external_id,
			last_active_at=excluded.last_active_at,
			metadata=excluded.metadata,
			updated_at=excluded.updated_at
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		snap.ID.String(),
		snap.UserCode,
		snap.AgentCode,
		snap.ChannelCode,
		snap.SessionKey,
		snap.ExternalID,
		lastActiveUnix,
		metadataJSON,
		snap.CreatedAt.Unix(),
		snap.UpdatedAt.Unix(),
	)
	return err
}

func (r *SQLiteSessionRepository) FindByID(ctx context.Context, id domain.SessionID) (*domain.Session, error) {
	row := r.db.QueryRowContext(ctx, `SELECT * FROM sessions WHERE id = ?`, id.String())
	return scanSession(row)
}

func (r *SQLiteSessionRepository) FindBySessionKey(ctx context.Context, sessionKey string) (*domain.Session, error) {
	row := r.db.QueryRowContext(ctx, `SELECT * FROM sessions WHERE session_key = ?`, sessionKey)
	return scanSession(row)
}

func (r *SQLiteSessionRepository) FindByUserCode(ctx context.Context, userCode string) ([]*domain.Session, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT * FROM sessions WHERE user_code = ? ORDER BY created_at DESC`, userCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

func (r *SQLiteSessionRepository) FindByChannelCode(ctx context.Context, channelCode string) ([]*domain.Session, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT * FROM sessions WHERE channel_code = ? ORDER BY created_at DESC`, channelCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

func (r *SQLiteSessionRepository) FindActiveByUserCode(ctx context.Context, userCode string) ([]*domain.Session, error) {
	return r.FindByUserCode(ctx, userCode)
}

func (r *SQLiteSessionRepository) DeleteBySessionKey(ctx context.Context, sessionKey string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE session_key = ?`, sessionKey)
	return err
}

func (r *SQLiteSessionRepository) DeleteByChannelCode(ctx context.Context, channelCode string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE channel_code = ?`, channelCode)
	return err
}

func scanSessions(rows *sql.Rows) ([]*domain.Session, error) {
	sessions := make([]*domain.Session, 0)
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		if session != nil {
			sessions = append(sessions, session)
		}
	}
	return sessions, rows.Err()
}

func scanSession(scanner rowScanner) (*domain.Session, error) {
	var (
		idStr          string
		userCode       string
		agentCode      string
		channelCode    string
		sessionKey     string
		externalID     string
		lastActiveUnix sql.NullInt64
		metadataJSON   []byte
		createdAtUnix  int64
		updatedAtUnix  int64
	)

	err := scanner.Scan(
		&idStr,
		&userCode,
		&agentCode,
		&channelCode,
		&sessionKey,
		&externalID,
		&lastActiveUnix,
		&metadataJSON,
		&createdAtUnix,
		&updatedAtUnix,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	metadata := map[string]interface{}{}
	_ = json.Unmarshal(metadataJSON, &metadata)
	var lastActive *time.Time
	if lastActiveUnix.Valid {
		t := time.Unix(lastActiveUnix.Int64, 0)
		lastActive = &t
	}

	session := &domain.Session{}
	session.FromSnapshot(domain.SessionSnapshot{
		ID:          domain.NewSessionID(idStr),
		UserCode:    userCode,
		AgentCode:   agentCode,
		ChannelCode: channelCode,
		SessionKey:  sessionKey,
		ExternalID:  externalID,
		LastActive:  lastActive,
		Metadata:    metadata,
		CreatedAt:   time.Unix(createdAtUnix, 0),
		UpdatedAt:   time.Unix(updatedAtUnix, 0),
	})
	return session, nil
}
