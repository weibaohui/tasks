package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteHeartbeatRepository struct {
	db *sql.DB
}

func NewSQLiteHeartbeatRepository(db *sql.DB) *SQLiteHeartbeatRepository {
	return &SQLiteHeartbeatRepository{db: db}
}

// RunInTx 在同一数据库事务中执行回调，确保场景应用等批量操作具备原子性。
func (r *SQLiteHeartbeatRepository) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	txCtx := withTxContext(ctx, tx)
	if err := fn(txCtx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return rollbackErr
		}
		return err
	}
	return tx.Commit()
}

func (r *SQLiteHeartbeatRepository) Save(ctx context.Context, hb *domain.Heartbeat) error {
	snap := hb.ToSnapshot()
	query := `
		INSERT INTO heartbeats (id, project_id, name, enabled, interval_minutes, md_content, agent_code, requirement_type, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			project_id=excluded.project_id,
			name=excluded.name,
			enabled=excluded.enabled,
			interval_minutes=excluded.interval_minutes,
			md_content=excluded.md_content,
			agent_code=excluded.agent_code,
			requirement_type=excluded.requirement_type,
			sort_order=excluded.sort_order,
			updated_at=excluded.updated_at
	`
	executor := executorFromContext(ctx, r.db)
	_, err := executor.ExecContext(
		ctx,
		query,
		snap.ID.String(),
		snap.ProjectID.String(),
		snap.Name,
		boolToInt(snap.Enabled),
		snap.IntervalMinutes,
		snap.MDContent,
		snap.AgentCode,
		snap.RequirementType,
		snap.SortOrder,
		snap.CreatedAt.Unix(),
		snap.UpdatedAt.Unix(),
	)
	return err
}

func (r *SQLiteHeartbeatRepository) FindByID(ctx context.Context, id domain.HeartbeatID) (*domain.Heartbeat, error) {
	executor := executorFromContext(ctx, r.db)
	row := executor.QueryRowContext(ctx, `
		SELECT id, project_id, name, enabled, interval_minutes, md_content, agent_code, requirement_type, sort_order, created_at, updated_at
		FROM heartbeats WHERE id = ?`, id.String())
	return scanHeartbeat(row)
}

func (r *SQLiteHeartbeatRepository) FindByProjectID(ctx context.Context, projectID domain.ProjectID) ([]*domain.Heartbeat, error) {
	executor := executorFromContext(ctx, r.db)
	rows, err := executor.QueryContext(ctx, `
		SELECT id, project_id, name, enabled, interval_minutes, md_content, agent_code, requirement_type, sort_order, created_at, updated_at
		FROM heartbeats WHERE project_id = ? ORDER BY sort_order, created_at`, projectID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHeartbeats(rows)
}

func (r *SQLiteHeartbeatRepository) FindAllEnabled(ctx context.Context) ([]*domain.Heartbeat, error) {
	executor := executorFromContext(ctx, r.db)
	rows, err := executor.QueryContext(ctx, `
		SELECT id, project_id, name, enabled, interval_minutes, md_content, agent_code, requirement_type, sort_order, created_at, updated_at
		FROM heartbeats WHERE enabled = 1 ORDER BY sort_order, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHeartbeats(rows)
}

func (r *SQLiteHeartbeatRepository) Delete(ctx context.Context, id domain.HeartbeatID) error {
	executor := executorFromContext(ctx, r.db)
	_, err := executor.ExecContext(ctx, `DELETE FROM heartbeats WHERE id = ?`, id.String())
	return err
}

func scanHeartbeats(rows *sql.Rows) ([]*domain.Heartbeat, error) {
	heartbeats := make([]*domain.Heartbeat, 0)
	for rows.Next() {
		hb, err := scanHeartbeat(rows)
		if err != nil {
			return nil, err
		}
		if hb != nil {
			heartbeats = append(heartbeats, hb)
		}
	}
	return heartbeats, rows.Err()
}

func scanHeartbeat(scanner rowScanner) (*domain.Heartbeat, error) {
	var (
		idStr           string
		projectIDStr    string
		name            string
		enabled         int
		intervalMinutes int
		mdContent       string
		agentCode       string
		requirementType string
		sortOrder       int
		createdAtUnix   int64
		updatedAtUnix   int64
	)
	err := scanner.Scan(&idStr, &projectIDStr, &name, &enabled, &intervalMinutes, &mdContent, &agentCode, &requirementType, &sortOrder, &createdAtUnix, &updatedAtUnix)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	hb := &domain.Heartbeat{}
	hb.FromSnapshot(domain.HeartbeatSnapshot{
		ID:              domain.NewHeartbeatID(idStr),
		ProjectID:       domain.NewProjectID(projectIDStr),
		Name:            name,
		Enabled:         enabled == 1,
		IntervalMinutes: intervalMinutes,
		MDContent:       mdContent,
		AgentCode:       agentCode,
		RequirementType: requirementType,
		SortOrder:       sortOrder,
		CreatedAt:       time.Unix(createdAtUnix, 0),
		UpdatedAt:       time.Unix(updatedAtUnix, 0),
	})
	return hb, nil
}
