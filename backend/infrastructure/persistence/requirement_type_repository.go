package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteRequirementTypeEntityRepository struct {
	db *sql.DB
}

func NewSQLiteRequirementTypeEntityRepository(db *sql.DB) *SQLiteRequirementTypeEntityRepository {
	return &SQLiteRequirementTypeEntityRepository{db: db}
}

func (r *SQLiteRequirementTypeEntityRepository) Save(ctx context.Context, rt *domain.RequirementTypeEntity) error {
	snap := rt.ToSnapshot()
	query := `
		INSERT INTO requirement_types (
			id, project_id, code, name, description, icon, color, sort_order, state_machine_id, is_system, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			description=excluded.description,
			icon=excluded.icon,
			color=excluded.color,
			sort_order=excluded.sort_order,
			state_machine_id=excluded.state_machine_id,
			is_system=excluded.is_system,
			updated_at=excluded.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		snap.ID.String(),
		snap.ProjectID.String(),
		snap.Code,
		snap.Name,
		snap.Description,
		snap.Icon,
		snap.Color,
		snap.SortOrder,
		snap.StateMachineID,
		boolToInt(snap.IsSystem),
		snap.CreatedAt.Unix(),
		snap.UpdatedAt.Unix(),
	)
	return err
}

func (r *SQLiteRequirementTypeEntityRepository) FindByID(ctx context.Context, id domain.RequirementTypeEntityID) (*domain.RequirementTypeEntity, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, code, name, COALESCE(description, ''), COALESCE(icon, ''), COALESCE(color, ''),
		       sort_order, COALESCE(state_machine_id, ''), is_system, created_at, updated_at
		FROM requirement_types WHERE id = ?`, id.String())
	return scanRequirementTypeEntity(row)
}

func (r *SQLiteRequirementTypeEntityRepository) FindByProjectID(ctx context.Context, projectID domain.ProjectID) ([]*domain.RequirementTypeEntity, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, code, name, COALESCE(description, ''), COALESCE(icon, ''), COALESCE(color, ''),
		       sort_order, COALESCE(state_machine_id, ''), is_system, created_at, updated_at
		FROM requirement_types WHERE project_id = ? ORDER BY sort_order ASC, created_at ASC`, projectID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRequirementTypeEntities(rows)
}

func (r *SQLiteRequirementTypeEntityRepository) FindByCode(ctx context.Context, projectID domain.ProjectID, code string) (*domain.RequirementTypeEntity, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, code, name, COALESCE(description, ''), COALESCE(icon, ''), COALESCE(color, ''),
		       sort_order, COALESCE(state_machine_id, ''), is_system, created_at, updated_at
		FROM requirement_types WHERE project_id = ? AND code = ?`, projectID.String(), code)
	return scanRequirementTypeEntity(row)
}

func (r *SQLiteRequirementTypeEntityRepository) Delete(ctx context.Context, id domain.RequirementTypeEntityID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM requirement_types WHERE id = ?`, id.String())
	return err
}

func scanRequirementTypeEntities(rows *sql.Rows) ([]*domain.RequirementTypeEntity, error) {
	types := make([]*domain.RequirementTypeEntity, 0)
	for rows.Next() {
		item, err := scanRequirementTypeEntity(rows)
		if err != nil {
			return nil, err
		}
		if item != nil {
			types = append(types, item)
		}
	}
	return types, rows.Err()
}

func scanRequirementTypeEntity(scanner rowScanner) (*domain.RequirementTypeEntity, error) {
	var (
		idStr          string
		projectIDStr   string
		code           string
		name           string
		description    string
		icon           string
		color          string
		sortOrder      int
		stateMachineID string
		isSystem       int
		createdAtUnix  int64
		updatedAtUnix  int64
	)
	err := scanner.Scan(
		&idStr,
		&projectIDStr,
		&code,
		&name,
		&description,
		&icon,
		&color,
		&sortOrder,
		&stateMachineID,
		&isSystem,
		&createdAtUnix,
		&updatedAtUnix,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rt := &domain.RequirementTypeEntity{}
	rt.FromSnapshot(domain.RequirementTypeEntitySnapshot{
		ID:             domain.NewRequirementTypeEntityID(idStr),
		ProjectID:      domain.NewProjectID(projectIDStr),
		Code:           code,
		Name:           name,
		Description:    description,
		Icon:           icon,
		Color:          color,
		SortOrder:      sortOrder,
		StateMachineID: stateMachineID,
		IsSystem:       isSystem == 1,
		CreatedAt:      time.Unix(createdAtUnix, 0),
		UpdatedAt:      time.Unix(updatedAtUnix, 0),
	})
	return rt, nil
}