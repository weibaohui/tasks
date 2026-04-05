package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/weibh/taskmanager/domain/state_machine"
)

// SQLiteStateMachineRepository SQLite 实现
type SQLiteStateMachineRepository struct {
	db *sql.DB
}

// NewSQLiteStateMachineRepository 创建 repository
func NewSQLiteStateMachineRepository(db *sql.DB) *SQLiteStateMachineRepository {
	return &SQLiteStateMachineRepository{db: db}
}

func (r *SQLiteStateMachineRepository) SaveStateMachine(ctx context.Context, sm *state_machine.StateMachine) error {
	configJSON, err := json.Marshal(sm.Config)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO state_machines (id, name, description, config, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			config = excluded.config,
			updated_at = excluded.updated_at
	`
	_, err = r.db.ExecContext(ctx, query,
		sm.ID, sm.Name, sm.Description, configJSON,
		sm.CreatedAt.UnixMilli(), sm.UpdatedAt.UnixMilli())
	return err
}

func (r *SQLiteStateMachineRepository) GetStateMachine(ctx context.Context, id string) (*state_machine.StateMachine, error) {
	query := `SELECT id, name, description, config, created_at, updated_at FROM state_machines WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanStateMachine(row)
}

func (r *SQLiteStateMachineRepository) ListStateMachines(ctx context.Context) ([]*state_machine.StateMachine, error) {
	query := `SELECT id, name, description, config, created_at, updated_at FROM state_machines`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*state_machine.StateMachine
	for rows.Next() {
		sm, err := r.scanStateMachineWithRows(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, sm)
	}
	return results, rows.Err()
}

func (r *SQLiteStateMachineRepository) DeleteStateMachine(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM state_machines WHERE id = ?`, id)
	return err
}

func (r *SQLiteStateMachineRepository) scanStateMachine(row *sql.Row) (*state_machine.StateMachine, error) {
	var sm state_machine.StateMachine
	var configJSON []byte
	var createdAtMs, updatedAtMs int64
	err := row.Scan(&sm.ID, &sm.Name, &sm.Description, &configJSON, &createdAtMs, &updatedAtMs)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, state_machine.ErrStateMachineNotFound("")
		}
		return nil, err
	}
	sm.CreatedAt = time.UnixMilli(createdAtMs)
	sm.UpdatedAt = time.UnixMilli(updatedAtMs)
	var cfg state_machine.Config
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, err
	}
	sm.Config = &cfg
	return &sm, nil
}

func (r *SQLiteStateMachineRepository) scanStateMachineWithRows(rows *sql.Rows) (*state_machine.StateMachine, error) {
	var sm state_machine.StateMachine
	var configJSON []byte
	var createdAtMs, updatedAtMs int64
	err := rows.Scan(&sm.ID, &sm.Name, &sm.Description, &configJSON, &createdAtMs, &updatedAtMs)
	if err != nil {
		return nil, err
	}
	sm.CreatedAt = time.UnixMilli(createdAtMs)
	sm.UpdatedAt = time.UnixMilli(updatedAtMs)
	var cfg state_machine.Config
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, err
	}
	sm.Config = &cfg
	return &sm, nil
}

// RequirementState
func (r *SQLiteStateMachineRepository) SaveRequirementState(ctx context.Context, rs *state_machine.RequirementState) error {
	query := `
		INSERT INTO requirement_states (id, requirement_id, state_machine_id, current_state, current_state_name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query, rs.ID, rs.RequirementID, rs.StateMachineID, rs.CurrentState, rs.CurrentStateName, rs.CreatedAt.UnixMilli(), rs.UpdatedAt.UnixMilli())
	return err
}

func (r *SQLiteStateMachineRepository) GetRequirementState(ctx context.Context, requirementID string) (*state_machine.RequirementState, error) {
	query := `SELECT id, requirement_id, state_machine_id, current_state, current_state_name, created_at, updated_at FROM requirement_states WHERE requirement_id = ?`
	row := r.db.QueryRowContext(ctx, query, requirementID)
	var rs state_machine.RequirementState
	var createdAtMs, updatedAtMs int64
	err := row.Scan(&rs.ID, &rs.RequirementID, &rs.StateMachineID, &rs.CurrentState, &rs.CurrentStateName, &createdAtMs, &updatedAtMs)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, state_machine.ErrRequirementStateNotFound(requirementID)
		}
		return nil, err
	}
	rs.CreatedAt = time.UnixMilli(createdAtMs)
	rs.UpdatedAt = time.UnixMilli(updatedAtMs)
	return &rs, nil
}

func (r *SQLiteStateMachineRepository) UpdateRequirementState(ctx context.Context, rs *state_machine.RequirementState) error {
	query := `UPDATE requirement_states SET current_state = ?, current_state_name = ?, updated_at = ? WHERE requirement_id = ?`
	_, err := r.db.ExecContext(ctx, query, rs.CurrentState, rs.CurrentStateName, rs.UpdatedAt.UnixMilli(), rs.RequirementID)
	return err
}

// TransitionLog
func (r *SQLiteStateMachineRepository) SaveTransitionLog(ctx context.Context, log *state_machine.TransitionLog) error {
	query := `
		INSERT INTO transition_logs (id, requirement_id, from_state, to_state, trigger, triggered_by, remark, result, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query, log.ID, log.RequirementID, log.FromState, log.ToState, log.Trigger, log.TriggeredBy, log.Remark, log.Result, log.ErrorMessage, log.CreatedAt.UnixMilli())
	return err
}

func (r *SQLiteStateMachineRepository) ListTransitionLogs(ctx context.Context, requirementID string) ([]*state_machine.TransitionLog, error) {
	query := `SELECT id, requirement_id, from_state, to_state, trigger, triggered_by, remark, result, error_message, created_at FROM transition_logs WHERE requirement_id = ? ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, requirementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*state_machine.TransitionLog
	for rows.Next() {
		var log state_machine.TransitionLog
		var createdAtMs int64
		err := rows.Scan(&log.ID, &log.RequirementID, &log.FromState, &log.ToState, &log.Trigger, &log.TriggeredBy, &log.Remark, &log.Result, &log.ErrorMessage, &createdAtMs)
		if err != nil {
			return nil, err
		}
		log.CreatedAt = time.UnixMilli(createdAtMs)
		results = append(results, &log)
	}
	return results, rows.Err()
}

// ProjectStateMachine
func (r *SQLiteStateMachineRepository) SaveProjectStateMachine(ctx context.Context, psm *state_machine.ProjectStateMachine) error {
	query := `
		INSERT INTO project_state_machines (id, project_id, requirement_type, state_machine_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id, requirement_type) DO UPDATE SET
			state_machine_id = excluded.state_machine_id,
			updated_at = excluded.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		psm.ID(), psm.ProjectID(), string(psm.RequirementType()), psm.StateMachineID(),
		psm.CreatedAt().UnixMilli(), psm.UpdatedAt().UnixMilli())
	return err
}

func (r *SQLiteStateMachineRepository) GetProjectStateMachine(ctx context.Context, projectID string, requirementType state_machine.RequirementType) (*state_machine.ProjectStateMachine, error) {
	query := `SELECT id, project_id, requirement_type, state_machine_id, created_at, updated_at FROM project_state_machines WHERE project_id = ? AND requirement_type = ?`
	row := r.db.QueryRowContext(ctx, query, projectID, string(requirementType))
	return r.scanProjectStateMachine(row)
}

func (r *SQLiteStateMachineRepository) ListProjectStateMachines(ctx context.Context, projectID string) ([]*state_machine.ProjectStateMachine, error) {
	query := `SELECT id, project_id, requirement_type, state_machine_id, created_at, updated_at FROM project_state_machines WHERE project_id = ?`
	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*state_machine.ProjectStateMachine
	for rows.Next() {
		psm, err := r.scanProjectStateMachineWithRows(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, psm)
	}
	return results, rows.Err()
}

func (r *SQLiteStateMachineRepository) DeleteProjectStateMachine(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM project_state_machines WHERE id = ?`, id)
	return err
}

func (r *SQLiteStateMachineRepository) DeleteProjectStateMachinesByProject(ctx context.Context, projectID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM project_state_machines WHERE project_id = ?`, projectID)
	return err
}

func (r *SQLiteStateMachineRepository) scanProjectStateMachine(row *sql.Row) (*state_machine.ProjectStateMachine, error) {
	var snap state_machine.ProjectStateMachineSnapshot
	var createdAtMs, updatedAtMs int64
	var reqType string
	err := row.Scan(&snap.ID, &snap.ProjectID, &reqType, &snap.StateMachineID, &createdAtMs, &updatedAtMs)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, state_machine.ErrProjectStateMachineNotFound
		}
		return nil, err
	}
	snap.RequirementType = state_machine.RequirementType(reqType)
	snap.CreatedAt = time.UnixMilli(createdAtMs)
	snap.UpdatedAt = time.UnixMilli(updatedAtMs)

	psm := &state_machine.ProjectStateMachine{}
	psm.FromSnapshot(snap)
	return psm, nil
}

func (r *SQLiteStateMachineRepository) scanProjectStateMachineWithRows(rows *sql.Rows) (*state_machine.ProjectStateMachine, error) {
	var snap state_machine.ProjectStateMachineSnapshot
	var createdAtMs, updatedAtMs int64
	var reqType string
	err := rows.Scan(&snap.ID, &snap.ProjectID, &reqType, &snap.StateMachineID, &createdAtMs, &updatedAtMs)
	if err != nil {
		return nil, err
	}
	snap.RequirementType = state_machine.RequirementType(reqType)
	snap.CreatedAt = time.UnixMilli(createdAtMs)
	snap.UpdatedAt = time.UnixMilli(updatedAtMs)

	psm := &state_machine.ProjectStateMachine{}
	psm.FromSnapshot(snap)
	return psm, nil
}

// Ensure SQLiteStateMachineRepository implements state_machine.Repository
var _ state_machine.Repository = (*SQLiteStateMachineRepository)(nil)
