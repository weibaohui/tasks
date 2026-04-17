package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteHeartbeatScenarioRepository struct {
	db *sql.DB
}

func NewSQLiteHeartbeatScenarioRepository(db *sql.DB) *SQLiteHeartbeatScenarioRepository {
	return &SQLiteHeartbeatScenarioRepository{db: db}
}

func (r *SQLiteHeartbeatScenarioRepository) Save(ctx context.Context, scenario *domain.HeartbeatScenario) error {
	snap := scenario.ToSnapshot()
	itemsJSON, err := json.Marshal(snap.Items)
	if err != nil {
		return err
	}
	query := `
		INSERT INTO heartbeat_scenarios (id, code, name, description, items, enabled, is_built_in, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			code=excluded.code,
			name=excluded.name,
			description=excluded.description,
			items=excluded.items,
			enabled=excluded.enabled,
			is_built_in=excluded.is_built_in,
			updated_at=excluded.updated_at
	`
	_, err = r.db.ExecContext(
		ctx,
		query,
		snap.ID.String(),
		snap.Code,
		snap.Name,
		snap.Description,
		string(itemsJSON),
		boolToInt(snap.Enabled),
		boolToInt(snap.IsBuiltIn),
		snap.CreatedAt.Unix(),
		snap.UpdatedAt.Unix(),
	)
	return err
}

func (r *SQLiteHeartbeatScenarioRepository) FindByID(ctx context.Context, id domain.HeartbeatScenarioID) (*domain.HeartbeatScenario, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, code, name, description, items, enabled, is_built_in, created_at, updated_at
		FROM heartbeat_scenarios WHERE id = ?`, id.String())
	return scanHeartbeatScenario(row)
}

func (r *SQLiteHeartbeatScenarioRepository) FindByCode(ctx context.Context, code string) (*domain.HeartbeatScenario, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, code, name, description, items, enabled, is_built_in, created_at, updated_at
		FROM heartbeat_scenarios WHERE code = ?`, code)
	return scanHeartbeatScenario(row)
}

func (r *SQLiteHeartbeatScenarioRepository) FindAll(ctx context.Context) ([]*domain.HeartbeatScenario, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, code, name, description, items, enabled, is_built_in, created_at, updated_at
		FROM heartbeat_scenarios ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	scenarios := make([]*domain.HeartbeatScenario, 0)
	for rows.Next() {
		s, err := scanHeartbeatScenario(rows)
		if err != nil {
			return nil, err
		}
		if s != nil {
			scenarios = append(scenarios, s)
		}
	}
	return scenarios, rows.Err()
}

func (r *SQLiteHeartbeatScenarioRepository) Delete(ctx context.Context, id domain.HeartbeatScenarioID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM heartbeat_scenarios WHERE id = ?`, id.String())
	return err
}

func scanHeartbeatScenario(scanner rowScanner) (*domain.HeartbeatScenario, error) {
	var (
		idStr       string
		code        string
		name        string
		description string
		itemsJSON   []byte
		enabled     int
		isBuiltIn   int
		createdAt   int64
		updatedAt   int64
	)
	err := scanner.Scan(&idStr, &code, &name, &description, &itemsJSON, &enabled, &isBuiltIn, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var items []domain.HeartbeatScenarioItem
	if err := json.Unmarshal(itemsJSON, &items); err != nil {
		items = []domain.HeartbeatScenarioItem{}
	}
	s := &domain.HeartbeatScenario{}
	s.FromSnapshot(domain.HeartbeatScenarioSnapshot{
		ID:          domain.NewHeartbeatScenarioID(idStr),
		Code:        code,
		Name:        name,
		Description: description,
		Items:       items,
		Enabled:     enabled == 1,
		IsBuiltIn:   isBuiltIn == 1,
		CreatedAt:   time.Unix(createdAt, 0),
		UpdatedAt:   time.Unix(updatedAt, 0),
	})
	return s, nil
}
