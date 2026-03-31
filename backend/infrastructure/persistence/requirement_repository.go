package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteRequirementRepository struct {
	db *sql.DB
}

func NewSQLiteRequirementRepository(db *sql.DB) *SQLiteRequirementRepository {
	return &SQLiteRequirementRepository{db: db}
}

func (r *SQLiteRequirementRepository) Save(ctx context.Context, requirement *domain.Requirement) error {
	snap := requirement.ToSnapshot()
	query := `
		INSERT INTO requirements (
			id, project_id, title, description, acceptance_criteria, status, dev_state,
			temp_workspace_root, assignee_agent_code, replica_agent_code, dispatch_session_key, workspace_path, branch_name, pr_url, last_error,
			started_at, completed_at, created_at, updated_at,
			claude_runtime_status, claude_runtime_started_at, claude_runtime_ended_at, claude_runtime_error
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title=excluded.title,
			description=excluded.description,
			acceptance_criteria=excluded.acceptance_criteria,
			status=excluded.status,
			dev_state=excluded.dev_state,
			temp_workspace_root=excluded.temp_workspace_root,
			assignee_agent_code=excluded.assignee_agent_code,
			replica_agent_code=excluded.replica_agent_code,
			dispatch_session_key=excluded.dispatch_session_key,
			workspace_path=excluded.workspace_path,
			branch_name=excluded.branch_name,
			pr_url=excluded.pr_url,
			last_error=excluded.last_error,
			started_at=excluded.started_at,
			completed_at=excluded.completed_at,
			updated_at=excluded.updated_at,
			claude_runtime_status=excluded.claude_runtime_status,
			claude_runtime_started_at=excluded.claude_runtime_started_at,
			claude_runtime_ended_at=excluded.claude_runtime_ended_at,
			claude_runtime_error=excluded.claude_runtime_error
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		snap.ID.String(),
		snap.ProjectID.String(),
		snap.Title,
		snap.Description,
		snap.AcceptanceCriteria,
		string(snap.Status),
		string(snap.DevState),
		snap.TempWorkspaceRoot,
		snap.AssigneeAgentCode,
		snap.ReplicaAgentCode,
		snap.DispatchSessionKey,
		snap.WorkspacePath,
		snap.BranchName,
		snap.PRURL,
		snap.LastError,
		timePtrToUnix(snap.StartedAt),
		timePtrToUnix(snap.CompletedAt),
		snap.CreatedAt.Unix(),
		snap.UpdatedAt.Unix(),
		snap.ClaudeRuntimeStatus,
		timePtrToUnix(snap.ClaudeRuntimeStartedAt),
		timePtrToUnix(snap.ClaudeRuntimeEndedAt),
		snap.ClaudeRuntimeError,
	)
	return err
}

func (r *SQLiteRequirementRepository) FindByID(ctx context.Context, id domain.RequirementID) (*domain.Requirement, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, title, COALESCE(description, ''), COALESCE(acceptance_criteria, ''),
		       status, dev_state, COALESCE(temp_workspace_root, ''), COALESCE(assignee_agent_code, ''), COALESCE(replica_agent_code, ''),
		       COALESCE(dispatch_session_key, ''), COALESCE(workspace_path, ''), COALESCE(branch_name, ''), COALESCE(pr_url, ''),
		       COALESCE(last_error, ''), started_at, completed_at, created_at, updated_at,
		       COALESCE(claude_runtime_status, ''), claude_runtime_started_at, claude_runtime_ended_at, COALESCE(claude_runtime_error, '')
		FROM requirements WHERE id = ?`, id.String())
	return scanRequirement(row)
}

func (r *SQLiteRequirementRepository) FindByProjectID(ctx context.Context, projectID domain.ProjectID) ([]*domain.Requirement, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, title, COALESCE(description, ''), COALESCE(acceptance_criteria, ''),
		       status, dev_state, COALESCE(temp_workspace_root, ''), COALESCE(assignee_agent_code, ''), COALESCE(replica_agent_code, ''),
		       COALESCE(dispatch_session_key, ''), COALESCE(workspace_path, ''), COALESCE(branch_name, ''), COALESCE(pr_url, ''),
		       COALESCE(last_error, ''), started_at, completed_at, created_at, updated_at,
		       COALESCE(claude_runtime_status, ''), claude_runtime_started_at, claude_runtime_ended_at, COALESCE(claude_runtime_error, '')
		FROM requirements WHERE project_id = ? ORDER BY created_at DESC`, projectID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRequirements(rows)
}

func (r *SQLiteRequirementRepository) FindAll(ctx context.Context) ([]*domain.Requirement, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, title, COALESCE(description, ''), COALESCE(acceptance_criteria, ''),
		       status, dev_state, COALESCE(temp_workspace_root, ''), COALESCE(assignee_agent_code, ''), COALESCE(replica_agent_code, ''),
		       COALESCE(dispatch_session_key, ''), COALESCE(workspace_path, ''), COALESCE(branch_name, ''), COALESCE(pr_url, ''),
		       COALESCE(last_error, ''), started_at, completed_at, created_at, updated_at,
		       COALESCE(claude_runtime_status, ''), claude_runtime_started_at, claude_runtime_ended_at, COALESCE(claude_runtime_error, '')
		FROM requirements ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRequirements(rows)
}

func (r *SQLiteRequirementRepository) Delete(ctx context.Context, id domain.RequirementID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM requirements WHERE id = ?`, id.String())
	return err
}

func scanRequirements(rows *sql.Rows) ([]*domain.Requirement, error) {
	requirements := make([]*domain.Requirement, 0)
	for rows.Next() {
		item, err := scanRequirement(rows)
		if err != nil {
			return nil, err
		}
		if item != nil {
			requirements = append(requirements, item)
		}
	}
	return requirements, rows.Err()
}

func scanRequirement(scanner rowScanner) (*domain.Requirement, error) {
	var (
		idStr                    string
		projectIDStr             string
		title                    string
		description              string
		acceptance               string
		statusStr                string
		devStateStr              string
		tempWorkspaceRoot        string
		assigneeAgentCode        string
		replicaAgentCode         string
		dispatchSessionKey       string
		workspacePath            string
		branchName               string
		prURL                    string
		lastError                string
		startedAtUnix            sql.NullInt64
		completedAtUnix          sql.NullInt64
		createdAtUnix            int64
		updatedAtUnix            int64
		claudeRuntimeStatus     string
		claudeRuntimeStartedAt  sql.NullInt64
		claudeRuntimeEndedAt    sql.NullInt64
		claudeRuntimeError      string
	)
	err := scanner.Scan(
		&idStr,
		&projectIDStr,
		&title,
		&description,
		&acceptance,
		&statusStr,
		&devStateStr,
		&tempWorkspaceRoot,
		&assigneeAgentCode,
		&replicaAgentCode,
		&dispatchSessionKey,
		&workspacePath,
		&branchName,
		&prURL,
		&lastError,
		&startedAtUnix,
		&completedAtUnix,
		&createdAtUnix,
		&updatedAtUnix,
		&claudeRuntimeStatus,
		&claudeRuntimeStartedAt,
		&claudeRuntimeEndedAt,
		&claudeRuntimeError,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	requirement := &domain.Requirement{}
	err = requirement.FromSnapshot(domain.RequirementSnapshot{
		ID:                      domain.NewRequirementID(idStr),
		ProjectID:               domain.NewProjectID(projectIDStr),
		Title:                   title,
		Description:             description,
		AcceptanceCriteria:      acceptance,
		Status:                  domain.RequirementStatus(statusStr),
		DevState:                domain.RequirementDevState(devStateStr),
		TempWorkspaceRoot:       tempWorkspaceRoot,
		AssigneeAgentCode:       assigneeAgentCode,
		ReplicaAgentCode:        replicaAgentCode,
		DispatchSessionKey:      dispatchSessionKey,
		WorkspacePath:           workspacePath,
		BranchName:              branchName,
		PRURL:                   prURL,
		LastError:               lastError,
		StartedAt:               unixToTimePtr(startedAtUnix),
		CompletedAt:             unixToTimePtr(completedAtUnix),
		CreatedAt:               time.Unix(createdAtUnix, 0),
		UpdatedAt:               time.Unix(updatedAtUnix, 0),
		ClaudeRuntimeStatus:     claudeRuntimeStatus,
		ClaudeRuntimeStartedAt:  unixToTimePtr(claudeRuntimeStartedAt),
		ClaudeRuntimeEndedAt:    unixToTimePtr(claudeRuntimeEndedAt),
		ClaudeRuntimeError:      claudeRuntimeError,
	})
	if err != nil {
		return nil, err
	}
	return requirement, nil
}

func timePtrToUnix(v *time.Time) any {
	if v == nil {
		return nil
	}
	return v.Unix()
}

func unixToTimePtr(v sql.NullInt64) *time.Time {
	if !v.Valid {
		return nil
	}
	t := time.Unix(v.Int64, 0)
	return &t
}
