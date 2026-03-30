package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/weibh/taskmanager/domain"
)

// SQLiteRequirementHookActionLogRepository Hook 执行日志仓储实现
type SQLiteRequirementHookActionLogRepository struct {
	db *sql.DB
}

// NewSQLiteRequirementHookActionLogRepository 创建仓储
func NewSQLiteRequirementHookActionLogRepository(db *sql.DB) *SQLiteRequirementHookActionLogRepository {
	return &SQLiteRequirementHookActionLogRepository{db: db}
}

func (r *SQLiteRequirementHookActionLogRepository) Save(ctx context.Context, log *domain.RequirementHookActionLog) error {
	query := `
		INSERT INTO requirement_hook_action_logs (id, hook_config_id, requirement_id, trigger_point, action_type, status, input_context, result, error, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status=excluded.status,
			result=excluded.result,
			error=excluded.error,
			completed_at=excluded.completed_at
	`
	var completedAtUnix interface{}
	if log.CompletedAt != nil {
		completedAtUnix = log.CompletedAt.Unix()
	}

	_, err := r.db.ExecContext(ctx, query,
		log.ID,
		log.HookConfigID,
		log.RequirementID,
		log.TriggerPoint,
		log.ActionType,
		log.Status,
		log.InputContext,
		log.Result,
		log.Error,
		log.StartedAt.Unix(),
		completedAtUnix,
	)
	return err
}

func (r *SQLiteRequirementHookActionLogRepository) FindByRequirementID(ctx context.Context, requirementID string) ([]*domain.RequirementHookActionLog, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, hook_config_id, requirement_id, trigger_point, action_type, status, input_context, result, error, started_at, completed_at
		FROM requirement_hook_action_logs WHERE requirement_id = ? ORDER BY started_at DESC`, requirementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHookActionLogs(rows)
}

func (r *SQLiteRequirementHookActionLogRepository) FindByHookConfigID(ctx context.Context, hookConfigID string, limit int) ([]*domain.RequirementHookActionLog, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, hook_config_id, requirement_id, trigger_point, action_type, status, input_context, result, error, started_at, completed_at
		FROM requirement_hook_action_logs WHERE hook_config_id = ? ORDER BY started_at DESC LIMIT ?`, hookConfigID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHookActionLogs(rows)
}

func scanHookActionLogs(rows *sql.Rows) ([]*domain.RequirementHookActionLog, error) {
	logs := make([]*domain.RequirementHookActionLog, 0)
	for rows.Next() {
		item, err := scanHookActionLog(rows)
		if err != nil {
			return nil, err
		}
		if item != nil {
			logs = append(logs, item)
		}
	}
	return logs, rows.Err()
}

func scanHookActionLog(scanner rowScanner) (*domain.RequirementHookActionLog, error) {
	var (
		id             string
		hookConfigID   string
		requirementID  string
		triggerPoint   string
		actionType     string
		status         string
		inputContext   sql.NullString
		result         sql.NullString
		errMsg         sql.NullString
		startedAtUnix  int64
		completedAtUnix sql.NullInt64
	)
	err := scanner.Scan(&id, &hookConfigID, &requirementID, &triggerPoint, &actionType, &status, &inputContext, &result, &errMsg, &startedAtUnix, &completedAtUnix)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	log := &domain.RequirementHookActionLog{
		ID:            id,
		HookConfigID:  hookConfigID,
		RequirementID: requirementID,
		TriggerPoint:  triggerPoint,
		ActionType:    actionType,
		Status:        status,
		StartedAt:     time.Unix(startedAtUnix, 0),
	}

	if inputContext.Valid {
		log.InputContext = inputContext.String
	}
	if result.Valid {
		log.Result = result.String
	}
	if errMsg.Valid {
		log.Error = errMsg.String
	}
	if completedAtUnix.Valid {
		t := time.Unix(completedAtUnix.Int64, 0)
		log.CompletedAt = &t
	}

	return log, nil
}
