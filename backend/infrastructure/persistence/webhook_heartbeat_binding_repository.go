package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteWebhookHeartbeatBindingRepository struct {
	db *sql.DB
}

func NewSQLiteWebhookHeartbeatBindingRepository(db *sql.DB) *SQLiteWebhookHeartbeatBindingRepository {
	return &SQLiteWebhookHeartbeatBindingRepository{db: db}
}

func (r *SQLiteWebhookHeartbeatBindingRepository) Save(ctx context.Context, binding *domain.WebhookHeartbeatBinding) error {
	snap := binding.ToSnapshot()
	query := `
		INSERT INTO webhook_heartbeat_bindings (id, project_id, github_webhook_config_id, github_event_type, heartbeat_id, enabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			github_event_type=excluded.github_event_type,
			heartbeat_id=excluded.heartbeat_id,
			enabled=excluded.enabled
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		snap.ID.String(),
		snap.ProjectID.String(),
		snap.ConfigID.String(),
		snap.GitHubEventType,
		snap.HeartbeatID.String(),
		boolToInt(snap.Enabled),
		snap.CreatedAt.Unix(),
	)
	return err
}

func (r *SQLiteWebhookHeartbeatBindingRepository) FindByID(ctx context.Context, id domain.WebhookHeartbeatBindingID) (*domain.WebhookHeartbeatBinding, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, github_webhook_config_id, github_event_type, heartbeat_id, enabled, created_at
		FROM webhook_heartbeat_bindings WHERE id = ?`, id.String())
	return scanWebhookHeartbeatBinding(row)
}

func (r *SQLiteWebhookHeartbeatBindingRepository) FindByConfigID(ctx context.Context, configID domain.GitHubWebhookConfigID) ([]*domain.WebhookHeartbeatBinding, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, github_webhook_config_id, github_event_type, heartbeat_id, enabled, created_at
		FROM webhook_heartbeat_bindings WHERE github_webhook_config_id = ? ORDER BY created_at`, configID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWebhookHeartbeatBindings(rows)
}

func (r *SQLiteWebhookHeartbeatBindingRepository) FindByConfigIDAndEventType(ctx context.Context, configID domain.GitHubWebhookConfigID, eventType string) ([]*domain.WebhookHeartbeatBinding, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, github_webhook_config_id, github_event_type, heartbeat_id, enabled, created_at
		FROM webhook_heartbeat_bindings WHERE github_webhook_config_id = ? AND github_event_type = ? AND enabled = 1`, configID.String(), eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWebhookHeartbeatBindings(rows)
}

func (r *SQLiteWebhookHeartbeatBindingRepository) Delete(ctx context.Context, id domain.WebhookHeartbeatBindingID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM webhook_heartbeat_bindings WHERE id = ?`, id.String())
	return err
}

func scanWebhookHeartbeatBindings(rows *sql.Rows) ([]*domain.WebhookHeartbeatBinding, error) {
	bindings := make([]*domain.WebhookHeartbeatBinding, 0)
	for rows.Next() {
		binding, err := scanWebhookHeartbeatBinding(rows)
		if err != nil {
			return nil, err
		}
		if binding != nil {
			bindings = append(bindings, binding)
		}
	}
	return bindings, rows.Err()
}

func scanWebhookHeartbeatBinding(scanner rowScanner) (*domain.WebhookHeartbeatBinding, error) {
	var (
		idStr           string
		projectIDStr   string
		configIDStr    string
		eventType      string
		heartbeatIDStr string
		enabled        int
		createdAtUnix  int64
	)
	err := scanner.Scan(&idStr, &projectIDStr, &configIDStr, &eventType, &heartbeatIDStr, &enabled, &createdAtUnix)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	binding := &domain.WebhookHeartbeatBinding{}
	binding.FromSnapshot(domain.WebhookHeartbeatBindingSnapshot{
		ID:              domain.NewWebhookHeartbeatBindingID(idStr),
		ProjectID:       domain.NewProjectID(projectIDStr),
		ConfigID:        domain.NewGitHubWebhookConfigID(configIDStr),
		GitHubEventType: eventType,
		HeartbeatID:     domain.NewHeartbeatID(heartbeatIDStr),
		Enabled:         enabled == 1,
		CreatedAt:       time.Unix(createdAtUnix, 0),
	})
	return binding, nil
}
