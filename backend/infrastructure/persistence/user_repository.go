package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteUserRepository struct {
	db *sql.DB
}

func NewSQLiteUserRepository(db *sql.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{db: db}
}

func (r *SQLiteUserRepository) Save(ctx context.Context, user *domain.User) error {
	snap := user.ToSnapshot()
	query := `
		INSERT INTO users (id, user_code, username, email, display_name, password_hash, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			email=excluded.email,
			display_name=excluded.display_name,
			password_hash=excluded.password_hash,
			is_active=excluded.is_active,
			updated_at=excluded.updated_at
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		snap.ID.String(),
		snap.UserCode.String(),
		snap.Username,
		snap.Email,
		snap.DisplayName,
		snap.PasswordHash,
		boolToInt(snap.IsActive),
		snap.CreatedAt.Unix(),
		snap.UpdatedAt.Unix(),
	)
	return err
}

func (r *SQLiteUserRepository) FindByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
	query := `SELECT id, user_code, username, email, display_name, password_hash, is_active, created_at, updated_at FROM users WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id.String())
	return scanUser(row)
}

func (r *SQLiteUserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `SELECT id, user_code, username, email, display_name, password_hash, is_active, created_at, updated_at FROM users WHERE username = ?`
	row := r.db.QueryRowContext(ctx, query, username)
	return scanUser(row)
}

func (r *SQLiteUserRepository) FindByUserCode(ctx context.Context, userCode domain.UserCode) (*domain.User, error) {
	query := `SELECT id, user_code, username, email, display_name, password_hash, is_active, created_at, updated_at FROM users WHERE user_code = ?`
	row := r.db.QueryRowContext(ctx, query, userCode.String())
	return scanUser(row)
}

func (r *SQLiteUserRepository) FindAll(ctx context.Context) ([]*domain.User, error) {
	query := `SELECT id, user_code, username, email, display_name, password_hash, is_active, created_at, updated_at FROM users ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*domain.User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		if user != nil {
			users = append(users, user)
		}
	}
	return users, rows.Err()
}

func (r *SQLiteUserRepository) Delete(ctx context.Context, id domain.UserID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id.String())
	return err
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanUser(scanner rowScanner) (*domain.User, error) {
	var (
		idStr        string
		userCodeStr  string
		username     string
		email        string
		displayName  string
		passwordHash string
		isActiveInt  int
		createdAt    int64
		updatedAt    int64
	)

	err := scanner.Scan(
		&idStr,
		&userCodeStr,
		&username,
		&email,
		&displayName,
		&passwordHash,
		&isActiveInt,
		&createdAt,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	user := &domain.User{}
	user.FromSnapshot(domain.UserSnapshot{
		ID:           domain.NewUserID(idStr),
		UserCode:     domain.NewUserCode(userCodeStr),
		Username:     username,
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: passwordHash,
		IsActive:     isActiveInt == 1,
		CreatedAt:    time.Unix(createdAt, 0),
		UpdatedAt:    time.Unix(updatedAt, 0),
	})
	return user, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
