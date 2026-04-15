package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteHeartbeatTemplateRepository struct {
	db *sql.DB
}

func NewSQLiteHeartbeatTemplateRepository(db *sql.DB) *SQLiteHeartbeatTemplateRepository {
	return &SQLiteHeartbeatTemplateRepository{db: db}
}

func (r *SQLiteHeartbeatTemplateRepository) Save(ctx context.Context, t *domain.HeartbeatTemplate) error {
	snap := t.ToSnapshot()
	query := `
		INSERT INTO heartbeat_templates (id, name, md_content, requirement_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			md_content=excluded.md_content,
			requirement_type=excluded.requirement_type,
			updated_at=excluded.updated_at
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		snap.ID.String(),
		snap.Name,
		snap.MDContent,
		snap.RequirementType,
		snap.CreatedAt.Unix(),
		snap.UpdatedAt.Unix(),
	)
	return err
}

func (r *SQLiteHeartbeatTemplateRepository) FindByID(ctx context.Context, id domain.HeartbeatTemplateID) (*domain.HeartbeatTemplate, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, md_content, requirement_type, created_at, updated_at
		FROM heartbeat_templates WHERE id = ?`, id.String())
	return scanHeartbeatTemplate(row)
}

func (r *SQLiteHeartbeatTemplateRepository) FindAll(ctx context.Context) ([]*domain.HeartbeatTemplate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, md_content, requirement_type, created_at, updated_at
		FROM heartbeat_templates ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHeartbeatTemplates(rows)
}

func (r *SQLiteHeartbeatTemplateRepository) Delete(ctx context.Context, id domain.HeartbeatTemplateID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM heartbeat_templates WHERE id = ?`, id.String())
	return err
}

func scanHeartbeatTemplates(rows *sql.Rows) ([]*domain.HeartbeatTemplate, error) {
	templates := make([]*domain.HeartbeatTemplate, 0)
	for rows.Next() {
		t, err := scanHeartbeatTemplate(rows)
		if err != nil {
			return nil, err
		}
		if t != nil {
			templates = append(templates, t)
		}
	}
	return templates, rows.Err()
}

func scanHeartbeatTemplate(scanner rowScanner) (*domain.HeartbeatTemplate, error) {
	var (
		idStr           string
		name            string
		mdContent       string
		requirementType string
		createdAtUnix   int64
		updatedAtUnix   int64
	)
	err := scanner.Scan(&idStr, &name, &mdContent, &requirementType, &createdAtUnix, &updatedAtUnix)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t := &domain.HeartbeatTemplate{}
	t.FromSnapshot(domain.HeartbeatTemplateSnapshot{
		ID:              domain.NewHeartbeatTemplateID(idStr),
		Name:            name,
		MDContent:       mdContent,
		RequirementType: requirementType,
		CreatedAt:       time.Unix(createdAtUnix, 0),
		UpdatedAt:       time.Unix(updatedAtUnix, 0),
	})
	return t, nil
}
