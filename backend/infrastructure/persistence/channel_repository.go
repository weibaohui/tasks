package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteChannelRepository struct {
	db *sql.DB
}

func NewSQLiteChannelRepository(db *sql.DB) *SQLiteChannelRepository {
	return &SQLiteChannelRepository{db: db}
}

func (r *SQLiteChannelRepository) Save(ctx context.Context, channel *domain.Channel) error {
	snap := channel.ToSnapshot()
	allowFromJSON, _ := json.Marshal(snap.AllowFrom)
	configJSON, _ := json.Marshal(snap.Config)

	query := `
		INSERT INTO channels (
			id, channel_code, user_code, agent_code, name, type, is_active, allow_from, config, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			agent_code=excluded.agent_code,
			name=excluded.name,
			type=excluded.type,
			is_active=excluded.is_active,
			allow_from=excluded.allow_from,
			config=excluded.config,
			updated_at=excluded.updated_at
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		snap.ID.String(),
		snap.Code.String(),
		snap.UserCode,
		snap.AgentCode,
		snap.Name,
		string(snap.Type),
		boolToInt(snap.IsActive),
		allowFromJSON,
		configJSON,
		snap.CreatedAt.Unix(),
		snap.UpdatedAt.Unix(),
	)
	return err
}

func (r *SQLiteChannelRepository) FindByID(ctx context.Context, id domain.ChannelID) (*domain.Channel, error) {
	row := r.db.QueryRowContext(ctx, `SELECT * FROM channels WHERE id = ?`, id.String())
	return scanChannel(row)
}

func (r *SQLiteChannelRepository) FindByCode(ctx context.Context, code domain.ChannelCode) (*domain.Channel, error) {
	row := r.db.QueryRowContext(ctx, `SELECT * FROM channels WHERE channel_code = ?`, code.String())
	return scanChannel(row)
}

func (r *SQLiteChannelRepository) FindByUserCode(ctx context.Context, userCode string) ([]*domain.Channel, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT * FROM channels WHERE user_code = ? ORDER BY created_at DESC`, userCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChannels(rows)
}

func (r *SQLiteChannelRepository) FindByAgentCode(ctx context.Context, agentCode string) ([]*domain.Channel, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT * FROM channels WHERE agent_code = ? ORDER BY created_at DESC`, agentCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChannels(rows)
}

func (r *SQLiteChannelRepository) FindActiveByUserCode(ctx context.Context, userCode string) ([]*domain.Channel, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT * FROM channels WHERE user_code = ? AND is_active = 1 ORDER BY created_at DESC`, userCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChannels(rows)
}

func (r *SQLiteChannelRepository) Delete(ctx context.Context, id domain.ChannelID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM channels WHERE id = ?`, id.String())
	return err
}

func scanChannels(rows *sql.Rows) ([]*domain.Channel, error) {
	channels := make([]*domain.Channel, 0)
	for rows.Next() {
		channel, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		if channel != nil {
			channels = append(channels, channel)
		}
	}
	return channels, rows.Err()
}

func scanChannel(scanner rowScanner) (*domain.Channel, error) {
	var (
		idStr         string
		codeStr       string
		userCode      string
		agentCode     string
		name          string
		typeStr       string
		isActiveInt   int
		allowFromJSON []byte
		configJSON    []byte
		createdAtUnix int64
		updatedAtUnix int64
	)

	err := scanner.Scan(
		&idStr,
		&codeStr,
		&userCode,
		&agentCode,
		&name,
		&typeStr,
		&isActiveInt,
		&allowFromJSON,
		&configJSON,
		&createdAtUnix,
		&updatedAtUnix,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var allowFrom []string
	_ = json.Unmarshal(allowFromJSON, &allowFrom)
	config := map[string]interface{}{}
	_ = json.Unmarshal(configJSON, &config)

	channel := &domain.Channel{}
	channel.FromSnapshot(domain.ChannelSnapshot{
		ID:        domain.NewChannelID(idStr),
		Code:      domain.NewChannelCode(codeStr),
		UserCode:  userCode,
		AgentCode: agentCode,
		Name:      name,
		Type:      domain.ChannelType(typeStr),
		IsActive:  isActiveInt == 1,
		AllowFrom: allowFrom,
		Config:    config,
		CreatedAt: time.Unix(createdAtUnix, 0),
		UpdatedAt: time.Unix(updatedAtUnix, 0),
	})
	return channel, nil
}
