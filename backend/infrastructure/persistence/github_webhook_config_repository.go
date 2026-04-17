package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteGitHubWebhookConfigRepository struct {
	db *sql.DB
}

func NewSQLiteGitHubWebhookConfigRepository(db *sql.DB) *SQLiteGitHubWebhookConfigRepository {
	return &SQLiteGitHubWebhookConfigRepository{db: db}
}

func (r *SQLiteGitHubWebhookConfigRepository) Save(ctx context.Context, config *domain.GitHubWebhookConfig) error {
	snap := config.ToSnapshot()
	query := `
		INSERT INTO github_webhook_configs (id, project_id, repo, enabled, forwarder_pid, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			project_id=excluded.project_id,
			repo=excluded.repo,
			enabled=excluded.enabled,
			forwarder_pid=excluded.forwarder_pid,
			updated_at=excluded.updated_at
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		snap.ID.String(),
		snap.ProjectID.String(),
		snap.Repo,
		boolToInt(snap.Enabled),
		snap.ForwarderPID,
		snap.CreatedAt.Unix(),
		snap.UpdatedAt.Unix(),
	)
	return err
}

func (r *SQLiteGitHubWebhookConfigRepository) FindByID(ctx context.Context, id domain.GitHubWebhookConfigID) (*domain.GitHubWebhookConfig, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, repo, enabled, forwarder_pid, created_at, updated_at
		FROM github_webhook_configs WHERE id = ?`, id.String())
	return scanGitHubWebhookConfig(row)
}

func (r *SQLiteGitHubWebhookConfigRepository) FindByProjectID(ctx context.Context, projectID domain.ProjectID) (*domain.GitHubWebhookConfig, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, repo, enabled, forwarder_pid, created_at, updated_at
		FROM github_webhook_configs WHERE project_id = ?`, projectID.String())
	return scanGitHubWebhookConfig(row)
}

func (r *SQLiteGitHubWebhookConfigRepository) FindAll(ctx context.Context) ([]*domain.GitHubWebhookConfig, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, repo, enabled, forwarder_pid, created_at, updated_at
		FROM github_webhook_configs ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGitHubWebhookConfigs(rows)
}

func (r *SQLiteGitHubWebhookConfigRepository) FindAllEnabled(ctx context.Context) ([]*domain.GitHubWebhookConfig, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, repo, enabled, forwarder_pid, created_at, updated_at
		FROM github_webhook_configs WHERE enabled = 1 ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGitHubWebhookConfigs(rows)
}

func (r *SQLiteGitHubWebhookConfigRepository) Delete(ctx context.Context, id domain.GitHubWebhookConfigID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM github_webhook_configs WHERE id = ?`, id.String())
	return err
}

func scanGitHubWebhookConfigs(rows *sql.Rows) ([]*domain.GitHubWebhookConfig, error) {
	configs := make([]*domain.GitHubWebhookConfig, 0)
	for rows.Next() {
		config, err := scanGitHubWebhookConfig(rows)
		if err != nil {
			return nil, err
		}
		if config != nil {
			configs = append(configs, config)
		}
	}
	return configs, rows.Err()
}

func scanGitHubWebhookConfig(scanner rowScanner) (*domain.GitHubWebhookConfig, error) {
	var (
		idStr          string
		projectIDStr   string
		repo           string
		enabled        int
		forwarderPID   sql.NullInt64
		createdAtUnix  int64
		updatedAtUnix  int64
	)
	err := scanner.Scan(&idStr, &projectIDStr, &repo, &enabled, &forwarderPID, &createdAtUnix, &updatedAtUnix)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	config := &domain.GitHubWebhookConfig{}
	var pid int
	if forwarderPID.Valid {
		pid = int(forwarderPID.Int64)
	}
	config.FromSnapshot(domain.GitHubWebhookConfigSnapshot{
		ID:           domain.NewGitHubWebhookConfigID(idStr),
		ProjectID:    domain.NewProjectID(projectIDStr),
		Repo:         repo,
		Enabled:      enabled == 1,
		ForwarderPID: pid,
		CreatedAt:    time.Unix(createdAtUnix, 0),
		UpdatedAt:    time.Unix(updatedAtUnix, 0),
	})
	return config, nil
}
