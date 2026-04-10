package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteUserTokenRepository struct {
	db *sql.DB
}

func NewSQLiteUserTokenRepository(db *sql.DB) *SQLiteUserTokenRepository {
	return &SQLiteUserTokenRepository{db: db}
}

func (r *SQLiteUserTokenRepository) Save(ctx context.Context, token *domain.UserToken) error {
	snap := token.ToSnapshot()
	query := `
		INSERT INTO user_tokens (id, user_id, name, description, token_hash, token_value, expires_at, last_used_at, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			description=excluded.description,
			token_value=excluded.token_value,
			expires_at=excluded.expires_at,
			last_used_at=excluded.last_used_at,
			is_active=excluded.is_active
	`
	var expiresAt *int64
	if snap.ExpiresAt != nil {
		expiresAt = snap.ExpiresAt
	}
	var lastUsedAt *int64
	if snap.LastUsedAt != nil {
		lastUsedAt = snap.LastUsedAt
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		snap.ID.String(),
		snap.UserID.String(),
		snap.Name,
		snap.Description,
		token.TokenHash(),
		token.TokenValue(),
		expiresAt,
		lastUsedAt,
		boolToInt(snap.IsActive),
		snap.CreatedAt,
	)
	return err
}

func (r *SQLiteUserTokenRepository) FindByID(ctx context.Context, id domain.UserTokenID) (*domain.UserToken, error) {
	query := `SELECT id, user_id, name, description, token_hash, token_value, expires_at, last_used_at, is_active, created_at FROM user_tokens WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id.String())
	return scanUserToken(row)
}

func (r *SQLiteUserTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.UserToken, error) {
	query := `SELECT id, user_id, name, description, token_hash, token_value, expires_at, last_used_at, is_active, created_at FROM user_tokens WHERE token_hash = ?`
	row := r.db.QueryRowContext(ctx, query, tokenHash)
	return scanUserToken(row)
}

func (r *SQLiteUserTokenRepository) FindByUserID(ctx context.Context, userID domain.UserID) ([]*domain.UserToken, error) {
	query := `SELECT id, user_id, name, description, token_hash, token_value, expires_at, last_used_at, is_active, created_at FROM user_tokens WHERE user_id = ? ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := make([]*domain.UserToken, 0)
	for rows.Next() {
		token, err := scanUserToken(rows)
		if err != nil {
			return nil, err
		}
		if token != nil {
			tokens = append(tokens, token)
		}
	}
	return tokens, rows.Err()
}

func (r *SQLiteUserTokenRepository) Delete(ctx context.Context, id domain.UserTokenID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_tokens WHERE id = ?`, id.String())
	return err
}

func (r *SQLiteUserTokenRepository) DeleteByUserID(ctx context.Context, userID domain.UserID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_tokens WHERE user_id = ?`, userID.String())
	return err
}

func scanUserToken(scanner rowScanner) (*domain.UserToken, error) {
	var (
		id           string
		userID       string
		name         string
		description  string
		tokenHash    string
		tokenValue   sql.NullString
		expiresAt    *int64
		lastUsedAt   *int64
		isActiveInt  int
		createdAt    int64
	)

	err := scanner.Scan(
		&id,
		&userID,
		&name,
		&description,
		&tokenHash,
		&tokenValue,
		&expiresAt,
		&lastUsedAt,
		&isActiveInt,
		&createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var expiresAtTime *time.Time
	if expiresAt != nil {
		t := time.UnixMilli(*expiresAt)
		expiresAtTime = &t
	}
	var lastUsedAtTime *time.Time
	if lastUsedAt != nil {
		t := time.UnixMilli(*lastUsedAt)
		lastUsedAtTime = &t
	}

	token, err := domain.NewUserToken(
		domain.NewUserTokenID(id),
		domain.NewUserID(userID),
		name,
		description,
		tokenHash,
		expiresAtTime,
	)
	if err != nil {
		return nil, err
	}
	if tokenValue.Valid {
		token.SetTokenValue(tokenValue.String)
	}
	if lastUsedAtTime != nil {
		token.UpdateLastUsed()
	}
	if isActiveInt != 1 {
		token.Deactivate()
	}
	return token, nil
}
