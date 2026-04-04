package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
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
		INSERT INTO state_machines (id, project_id, name, description, config, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			config = excluded.config,
			updated_at = excluded.updated_at
	`
	_, err = r.db.ExecContext(ctx, query, sm.ID, sm.ProjectID, sm.Name, sm.Description, configJSON, sm.CreatedAt, sm.UpdatedAt)
	return err
}

func (r *SQLiteStateMachineRepository) GetStateMachine(ctx context.Context, id string) (*state_machine.StateMachine, error) {
	query := `SELECT id, project_id, name, description, config, created_at, updated_at FROM state_machines WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanStateMachine(row)
}

func (r *SQLiteStateMachineRepository) ListStateMachines(ctx context.Context, projectID string) ([]*state_machine.StateMachine, error) {
	query := `SELECT id, project_id, name, description, config, created_at, updated_at FROM state_machines WHERE project_id = ?`
	rows, err := r.db.QueryContext(ctx, query, projectID)
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
	tx, err := r.db.ExecContext(ctx, `DELETE FROM state_machines WHERE id = ?`, id)
	if err != nil {
		return err
	}
	_, err = tx.RowsAffected()
	return err
}

func (r *SQLiteStateMachineRepository) scanStateMachine(row *sql.Row) (*state_machine.StateMachine, error) {
	var sm state_machine.StateMachine
	var configJSON []byte
	err := row.Scan(&sm.ID, &sm.ProjectID, &sm.Name, &sm.Description, &configJSON, &sm.CreatedAt, &sm.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, state_machine.ErrStateMachineNotFound("")
		}
		return nil, err
	}
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
	err := rows.Scan(&sm.ID, &sm.ProjectID, &sm.Name, &sm.Description, &configJSON, &sm.CreatedAt, &sm.UpdatedAt)
	if err != nil {
		return nil, err
	}
	var cfg state_machine.Config
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, err
	}
	sm.Config = &cfg
	return &sm, nil
}

// TypeBinding
func (r *SQLiteStateMachineRepository) SaveTypeBinding(ctx context.Context, binding *state_machine.TypeBinding) error {
	query := `
		INSERT INTO state_machine_type_bindings (id, state_machine_id, requirement_type, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(state_machine_id, requirement_type) DO UPDATE SET
			state_machine_id = excluded.state_machine_id
	`
	_, err := r.db.ExecContext(ctx, query, binding.ID, binding.StateMachineID, binding.RequirementType, binding.CreatedAt)
	return err
}

func (r *SQLiteStateMachineRepository) GetTypeBinding(ctx context.Context, stateMachineID, requirementType string) (*state_machine.TypeBinding, error) {
	query := `SELECT id, state_machine_id, requirement_type, created_at FROM state_machine_type_bindings WHERE state_machine_id = ? AND requirement_type = ?`
	row := r.db.QueryRowContext(ctx, query, stateMachineID, requirementType)
	var binding state_machine.TypeBinding
	err := row.Scan(&binding.ID, &binding.StateMachineID, &binding.RequirementType, &binding.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &binding, nil
}

func (r *SQLiteStateMachineRepository) DeleteTypeBinding(ctx context.Context, stateMachineID, requirementType string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM state_machine_type_bindings WHERE state_machine_id = ? AND requirement_type = ?`, stateMachineID, requirementType)
	return err
}

func (r *SQLiteStateMachineRepository) GetStateMachineByType(ctx context.Context, projectID, requirementType string) (*state_machine.StateMachine, error) {
	query := `
		SELECT sm.id, sm.project_id, sm.name, sm.description, sm.config, sm.created_at, sm.updated_at
		FROM state_machines sm
		JOIN state_machine_type_bindings tb ON sm.id = tb.state_machine_id
		WHERE sm.project_id = ? AND tb.requirement_type = ?
	`
	row := r.db.QueryRowContext(ctx, query, projectID, requirementType)
	sm, err := r.scanStateMachine(row)
	if err != nil {
		return nil, err
	}
	return sm, nil
}

// RequirementState
func (r *SQLiteStateMachineRepository) SaveRequirementState(ctx context.Context, rs *state_machine.RequirementState) error {
	query := `
		INSERT INTO requirement_states (id, requirement_id, state_machine_id, current_state, current_state_name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query, rs.ID, rs.RequirementID, rs.StateMachineID, rs.CurrentState, rs.CurrentStateName, rs.CreatedAt, rs.UpdatedAt)
	return err
}

func (r *SQLiteStateMachineRepository) GetRequirementState(ctx context.Context, requirementID string) (*state_machine.RequirementState, error) {
	query := `SELECT id, requirement_id, state_machine_id, current_state, current_state_name, created_at, updated_at FROM requirement_states WHERE requirement_id = ?`
	row := r.db.QueryRowContext(ctx, query, requirementID)
	var rs state_machine.RequirementState
	err := row.Scan(&rs.ID, &rs.RequirementID, &rs.StateMachineID, &rs.CurrentState, &rs.CurrentStateName, &rs.CreatedAt, &rs.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, state_machine.ErrRequirementStateNotFound(requirementID)
		}
		return nil, err
	}
	return &rs, nil
}

func (r *SQLiteStateMachineRepository) UpdateRequirementState(ctx context.Context, rs *state_machine.RequirementState) error {
	query := `UPDATE requirement_states SET current_state = ?, current_state_name = ?, updated_at = ? WHERE requirement_id = ?`
	_, err := r.db.ExecContext(ctx, query, rs.CurrentState, rs.CurrentStateName, rs.UpdatedAt, rs.RequirementID)
	return err
}

// TransitionLog
func (r *SQLiteStateMachineRepository) SaveTransitionLog(ctx context.Context, log *state_machine.TransitionLog) error {
	query := `
		INSERT INTO transition_logs (id, requirement_id, from_state, to_state, trigger, triggered_by, remark, result, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query, log.ID, log.RequirementID, log.FromState, log.ToState, log.Trigger, log.TriggeredBy, log.Remark, log.Result, log.ErrorMessage, log.CreatedAt)
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
		err := rows.Scan(&log.ID, &log.RequirementID, &log.FromState, &log.ToState, &log.Trigger, &log.TriggeredBy, &log.Remark, &log.Result, &log.ErrorMessage, &log.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, &log)
	}
	return results, rows.Err()
}

// Ensure SQLiteStateMachineRepository implements state_machine.Repository
var _ state_machine.Repository = (*SQLiteStateMachineRepository)(nil)

// generateID 生成 UUID
func generateID() string {
	return uuid.New().String()
}

// generateTimeBasedID 生成基于时间的 ID
func generateTimeBasedID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	now := time.Now().UnixNano()
	for i := range b {
		b[i] = letters[int(now)%len(letters)]
	}
	return string(b)
}
