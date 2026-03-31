package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/weibh/taskmanager/domain"
)

// SQLiteRequirementHookConfigRepository Hook 配置仓储实现
type SQLiteRequirementHookConfigRepository struct {
	db *sql.DB
}

// NewSQLiteRequirementHookConfigRepository 创建仓储
func NewSQLiteRequirementHookConfigRepository(db *sql.DB) *SQLiteRequirementHookConfigRepository {
	return &SQLiteRequirementHookConfigRepository{db: db}
}

func (r *SQLiteRequirementHookConfigRepository) Save(ctx context.Context, config *domain.RequirementHookConfig) error {
	query := `
		INSERT INTO requirement_hook_configs (id, project_id, name, trigger_point, action_type, action_config, enabled, priority, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
			project_id=excluded.project_id,
			name=excluded.name,
			trigger_point=excluded.trigger_point,
			action_type=excluded.action_type,
			action_config=excluded.action_config,
			enabled=excluded.enabled,
			priority=excluded.priority,
			updated_at=excluded.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		config.ID,
		config.ProjectID,
		config.Name,
		config.TriggerPoint,
		config.ActionType,
		config.ActionConfig,
		config.Enabled,
		config.Priority,
		config.CreatedAt.Unix(),
		config.UpdatedAt.Unix(),
	)
	return err
}

func (r *SQLiteRequirementHookConfigRepository) FindByID(ctx context.Context, id string) (*domain.RequirementHookConfig, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(project_id, '') as project_id, name, trigger_point, action_type, action_config, enabled, priority, created_at, updated_at
		FROM requirement_hook_configs WHERE id = ?`, id)
	return scanHookConfig(row)
}

func (r *SQLiteRequirementHookConfigRepository) FindByTriggerPoint(ctx context.Context, triggerPoint string) ([]*domain.RequirementHookConfig, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, COALESCE(project_id, '') as project_id, name, trigger_point, action_type, action_config, enabled, priority, created_at, updated_at
		FROM requirement_hook_configs WHERE trigger_point = ? ORDER BY priority`, triggerPoint)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHookConfigs(rows)
}

func (r *SQLiteRequirementHookConfigRepository) FindByProjectID(ctx context.Context, projectID string) ([]*domain.RequirementHookConfig, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, COALESCE(project_id, '') as project_id, name, trigger_point, action_type, action_config, enabled, priority, created_at, updated_at
		FROM requirement_hook_configs WHERE project_id = ? ORDER BY priority`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHookConfigs(rows)
}

func (r *SQLiteRequirementHookConfigRepository) FindEnabledByTriggerPoint(ctx context.Context, triggerPoint string) ([]*domain.RequirementHookConfig, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, COALESCE(project_id, '') as project_id, name, trigger_point, action_type, action_config, enabled, priority, created_at, updated_at
		FROM requirement_hook_configs WHERE trigger_point = ? AND enabled = 1 ORDER BY priority`, triggerPoint)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHookConfigs(rows)
}

func (r *SQLiteRequirementHookConfigRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM requirement_hook_configs WHERE id = ?`, id)
	return err
}

func scanHookConfigs(rows *sql.Rows) ([]*domain.RequirementHookConfig, error) {
	configs := make([]*domain.RequirementHookConfig, 0)
	for rows.Next() {
		item, err := scanHookConfig(rows)
		if err != nil {
			return nil, err
		}
		if item != nil {
			configs = append(configs, item)
		}
	}
	return configs, rows.Err()
}

func scanHookConfig(scanner rowScanner) (*domain.RequirementHookConfig, error) {
	var (
		id            string
		projectID     string
		name          string
		triggerPoint  string
		actionType    string
		actionConfig  string
		enabled       int
		priority      int
		createdAtUnix int64
		updatedAtUnix int64
	)
	err := scanner.Scan(&id, &projectID, &name, &triggerPoint, &actionType, &actionConfig, &enabled, &priority, &createdAtUnix, &updatedAtUnix)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.RequirementHookConfig{
		ID:           id,
		ProjectID:    projectID,
		Name:         name,
		TriggerPoint: triggerPoint,
		ActionType:   actionType,
		ActionConfig: actionConfig,
		Enabled:      enabled == 1,
		Priority:     priority,
		CreatedAt:   time.Unix(createdAtUnix, 0),
		UpdatedAt:   time.Unix(updatedAtUnix, 0),
	}, nil
}