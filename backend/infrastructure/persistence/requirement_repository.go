package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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
			id, project_id, title, description, acceptance_criteria, status,
			temp_workspace_root, assignee_agent_code, assignee_agent_name, replica_agent_code, replica_agent_name, replica_agent_shadow_from, dispatch_session_key, workspace_path, last_error,
			started_at, completed_at, created_at, updated_at,
			requirement_type, agent_runtime_status, agent_runtime_started_at, agent_runtime_ended_at, agent_runtime_error, agent_runtime_result, agent_runtime_prompt, agent_runtime_agent_type, trace_id,
			prompt_tokens, completion_tokens, total_tokens, progress_data
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title=excluded.title,
			description=excluded.description,
			acceptance_criteria=excluded.acceptance_criteria,
			status=excluded.status,
			temp_workspace_root=excluded.temp_workspace_root,
			assignee_agent_code=excluded.assignee_agent_code,
			assignee_agent_name=excluded.assignee_agent_name,
			replica_agent_code=excluded.replica_agent_code,
			replica_agent_name=excluded.replica_agent_name,
			replica_agent_shadow_from=excluded.replica_agent_shadow_from,
			dispatch_session_key=excluded.dispatch_session_key,
			workspace_path=excluded.workspace_path,
			last_error=excluded.last_error,
			started_at=excluded.started_at,
			completed_at=excluded.completed_at,
			updated_at=excluded.updated_at,
			requirement_type=excluded.requirement_type,
			agent_runtime_status=excluded.agent_runtime_status,
			agent_runtime_started_at=excluded.agent_runtime_started_at,
			agent_runtime_ended_at=excluded.agent_runtime_ended_at,
			agent_runtime_error=excluded.agent_runtime_error,
			agent_runtime_result=excluded.agent_runtime_result,
			agent_runtime_prompt=excluded.agent_runtime_prompt,
			agent_runtime_agent_type=excluded.agent_runtime_agent_type,
		    trace_id=excluded.trace_id,
			prompt_tokens=excluded.prompt_tokens,
			completion_tokens=excluded.completion_tokens,
			total_tokens=excluded.total_tokens,
			progress_data=excluded.progress_data
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
		snap.TempWorkspaceRoot,
		snap.AssigneeAgentCode,
		snap.AssigneeAgentName,
		snap.ReplicaAgentCode,
		snap.ReplicaAgentName,
		snap.ReplicaAgentShadowFrom,
		snap.DispatchSessionKey,
		snap.WorkspacePath,
		snap.LastError,
		timePtrToUnix(snap.StartedAt),
		timePtrToUnix(snap.CompletedAt),
		snap.CreatedAt.Unix(),
		snap.UpdatedAt.Unix(),
		string(snap.RequirementType),
		snap.AgentRuntimeStatus,
		timePtrToUnix(snap.AgentRuntimeStartedAt),
		timePtrToUnix(snap.AgentRuntimeEndedAt),
		snap.AgentRuntimeError,
		snap.AgentRuntimeResult,
		snap.AgentRuntimePrompt,
		snap.AgentRuntimeAgentType,
		snap.TraceID,
		snap.PromptTokens,
		snap.CompletionTokens,
		snap.TotalTokens,
		snap.ProgressData,
	)
	return err
}

func (r *SQLiteRequirementRepository) FindByID(ctx context.Context, id domain.RequirementID) (*domain.Requirement, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, title, COALESCE(description, ''), COALESCE(acceptance_criteria, ''),
		       status, COALESCE(temp_workspace_root, ''), COALESCE(assignee_agent_code, ''), COALESCE(assignee_agent_name, ''), COALESCE(replica_agent_code, ''), COALESCE(replica_agent_name, ''), COALESCE(replica_agent_shadow_from, ''),
		       COALESCE(dispatch_session_key, ''), COALESCE(workspace_path, ''),
		       COALESCE(last_error, ''), started_at, completed_at, created_at, updated_at,
		       COALESCE(requirement_type, 'normal'), COALESCE(agent_runtime_status, ''), agent_runtime_started_at, agent_runtime_ended_at, COALESCE(agent_runtime_error, ''), COALESCE(agent_runtime_result, ''), COALESCE(agent_runtime_prompt, ''), COALESCE(agent_runtime_agent_type, ''), COALESCE(trace_id, ''), prompt_tokens, completion_tokens, total_tokens, COALESCE(progress_data, '')
		FROM requirements WHERE id = ?`, id.String())
	return scanRequirement(row)
}

func (r *SQLiteRequirementRepository) FindByTraceID(ctx context.Context, traceID string) (*domain.Requirement, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, title, COALESCE(description, ''), COALESCE(acceptance_criteria, ''),
		       status, COALESCE(temp_workspace_root, ''), COALESCE(assignee_agent_code, ''), COALESCE(assignee_agent_name, ''), COALESCE(replica_agent_code, ''), COALESCE(replica_agent_name, ''), COALESCE(replica_agent_shadow_from, ''),
		       COALESCE(dispatch_session_key, ''), COALESCE(workspace_path, ''),
		       COALESCE(last_error, ''), started_at, completed_at, created_at, updated_at,
		       COALESCE(requirement_type, 'normal'), COALESCE(agent_runtime_status, ''), agent_runtime_started_at, agent_runtime_ended_at, COALESCE(agent_runtime_error, ''), COALESCE(agent_runtime_result, ''), COALESCE(agent_runtime_prompt, ''), COALESCE(agent_runtime_agent_type, ''), COALESCE(trace_id, ''), prompt_tokens, completion_tokens, total_tokens, COALESCE(progress_data, '')
		FROM requirements WHERE trace_id = ? LIMIT 1`, traceID)
	return scanRequirement(row)
}

func (r *SQLiteRequirementRepository) FindByProjectID(ctx context.Context, projectID domain.ProjectID) ([]*domain.Requirement, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, title, COALESCE(description, ''), COALESCE(acceptance_criteria, ''),
		       status, COALESCE(temp_workspace_root, ''), COALESCE(assignee_agent_code, ''), COALESCE(assignee_agent_name, ''), COALESCE(replica_agent_code, ''), COALESCE(replica_agent_name, ''), COALESCE(replica_agent_shadow_from, ''),
		       COALESCE(dispatch_session_key, ''), COALESCE(workspace_path, ''),
		       COALESCE(last_error, ''), started_at, completed_at, created_at, updated_at,
		       COALESCE(requirement_type, 'normal'), COALESCE(agent_runtime_status, ''), agent_runtime_started_at, agent_runtime_ended_at, COALESCE(agent_runtime_error, ''), COALESCE(agent_runtime_result, ''), COALESCE(agent_runtime_prompt, ''), COALESCE(agent_runtime_agent_type, ''), COALESCE(trace_id, ''), prompt_tokens, completion_tokens, total_tokens, COALESCE(progress_data, '')
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
		       status, COALESCE(temp_workspace_root, ''), COALESCE(assignee_agent_code, ''), COALESCE(assignee_agent_name, ''), COALESCE(replica_agent_code, ''), COALESCE(replica_agent_name, ''), COALESCE(replica_agent_shadow_from, ''),
		       COALESCE(dispatch_session_key, ''), COALESCE(workspace_path, ''),
		       COALESCE(last_error, ''), started_at, completed_at, created_at, updated_at,
		       COALESCE(requirement_type, 'normal'), COALESCE(agent_runtime_status, ''), agent_runtime_started_at, agent_runtime_ended_at, COALESCE(agent_runtime_error, ''), COALESCE(agent_runtime_result, ''), COALESCE(agent_runtime_prompt, ''), COALESCE(agent_runtime_agent_type, ''), COALESCE(trace_id, ''), prompt_tokens, completion_tokens, total_tokens, COALESCE(progress_data, '')
		FROM requirements ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRequirements(rows)
}

const requirementColumns = `id, project_id, title, COALESCE(description, ''), COALESCE(acceptance_criteria, ''),
	status, COALESCE(temp_workspace_root, ''), COALESCE(assignee_agent_code, ''), COALESCE(assignee_agent_name, ''), COALESCE(replica_agent_code, ''), COALESCE(replica_agent_name, ''), COALESCE(replica_agent_shadow_from, ''),
	COALESCE(dispatch_session_key, ''), COALESCE(workspace_path, ''),
	COALESCE(last_error, ''), started_at, completed_at, created_at, updated_at,
	COALESCE(requirement_type, 'normal'), COALESCE(agent_runtime_status, ''), agent_runtime_started_at, agent_runtime_ended_at, COALESCE(agent_runtime_error, ''), COALESCE(agent_runtime_result, ''), COALESCE(agent_runtime_prompt, ''), COALESCE(agent_runtime_agent_type, ''), COALESCE(trace_id, ''), prompt_tokens, completion_tokens, total_tokens, COALESCE(progress_data, '')`

func (r *SQLiteRequirementRepository) List(ctx context.Context, filter domain.RequirementListFilter) ([]*domain.Requirement, error) {
	where, args := r.buildWhereClause(filter)
	query := fmt.Sprintf(`SELECT %s FROM requirements %s ORDER BY created_at DESC LIMIT ? OFFSET ?`, requirementColumns, where)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRequirements(rows)
}

func (r *SQLiteRequirementRepository) Count(ctx context.Context, filter domain.RequirementListFilter) (int, error) {
	where, args := r.buildWhereClause(filter)
	query := fmt.Sprintf(`SELECT COUNT(*) FROM requirements %s`, where)

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func (r *SQLiteRequirementRepository) buildWhereClause(filter domain.RequirementListFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if filter.ProjectID != nil {
		conditions = append(conditions, "project_id = ?")
		args = append(args, filter.ProjectID.String())
	}
	if len(filter.Statuses) == 1 {
		conditions = append(conditions, "status = ?")
		args = append(args, filter.Statuses[0])
	} else if len(filter.Statuses) > 1 {
		placeholders := make([]string, len(filter.Statuses))
		for i, s := range filter.Statuses {
			placeholders[i] = "?"
			args = append(args, s)
		}
		conditions = append(conditions, "status IN ("+strings.Join(placeholders, ", ")+")")
	}

	if len(conditions) > 0 {
		return "WHERE " + strings.Join(conditions, " AND "), args
	}
	return "", args
}

func (r *SQLiteRequirementRepository) Delete(ctx context.Context, id domain.RequirementID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM requirements WHERE id = ?`, id.String())
	return err
}

// GetStatusStats 获取所有状态的统计数据（动态从数据库提取）
func (r *SQLiteRequirementRepository) GetStatusStats(ctx context.Context, projectID *domain.ProjectID) ([]domain.StatusStat, error) {
	var query string
	var args []interface{}

	if projectID != nil {
		query = `SELECT status, COUNT(*) as count FROM requirements WHERE project_id = ? GROUP BY status ORDER BY count DESC`
		args = append(args, projectID.String())
	} else {
		query = `SELECT status, COUNT(*) as count FROM requirements GROUP BY status ORDER BY count DESC`
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]domain.StatusStat, 0)
	for rows.Next() {
		var stat domain.StatusStat
		if err := rows.Scan(&stat.Status, &stat.Count); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	return stats, rows.Err()
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
		tempWorkspaceRoot        string
		assigneeAgentCode        string
		assigneeAgentName        string
		replicaAgentCode         string
		replicaAgentName         string
		replicaAgentShadowFrom   string
		dispatchSessionKey       string
		workspacePath            string
		lastError                string
		startedAtUnix            sql.NullInt64
		completedAtUnix          sql.NullInt64
		createdAtUnix            int64
		updatedAtUnix            int64
		requirementType         string
		agentRuntimeStatus      string
		agentRuntimeStartedAt   sql.NullInt64
		agentRuntimeEndedAt     sql.NullInt64
		agentRuntimeError       string
		agentRuntimeResult      string
		agentRuntimePrompt      string
		agentRuntimeAgentType   string
		traceID                 string
		promptTokens            int
		completionTokens        int
		totalTokens             int
		progressData            string
	)
	err := scanner.Scan(
		&idStr,
		&projectIDStr,
		&title,
		&description,
		&acceptance,
		&statusStr,
		&tempWorkspaceRoot,
		&assigneeAgentCode,
		&assigneeAgentName,
		&replicaAgentCode,
		&replicaAgentName,
		&replicaAgentShadowFrom,
		&dispatchSessionKey,
		&workspacePath,
		&lastError,
		&startedAtUnix,
		&completedAtUnix,
		&createdAtUnix,
		&updatedAtUnix,
		&requirementType,
		&agentRuntimeStatus,
		&agentRuntimeStartedAt,
		&agentRuntimeEndedAt,
		&agentRuntimeError,
		&agentRuntimeResult,
		&agentRuntimePrompt,
		&agentRuntimeAgentType,
		&traceID,
		&promptTokens,
		&completionTokens,
		&totalTokens,
		&progressData,
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
		TempWorkspaceRoot:       tempWorkspaceRoot,
		AssigneeAgentCode:       assigneeAgentCode,
		AssigneeAgentName:       assigneeAgentName,
		ReplicaAgentCode:        replicaAgentCode,
		ReplicaAgentName:        replicaAgentName,
		ReplicaAgentShadowFrom:  replicaAgentShadowFrom,
		DispatchSessionKey:      dispatchSessionKey,
		WorkspacePath:           workspacePath,
		LastError:               lastError,
		StartedAt:               unixToTimePtr(startedAtUnix),
		CompletedAt:             unixToTimePtr(completedAtUnix),
		CreatedAt:               time.Unix(createdAtUnix, 0),
		UpdatedAt:               time.Unix(updatedAtUnix, 0),
		RequirementType:         domain.RequirementType(requirementType),
		AgentRuntimeStatus:      agentRuntimeStatus,
		AgentRuntimeStartedAt:   unixToTimePtr(agentRuntimeStartedAt),
		AgentRuntimeEndedAt:     unixToTimePtr(agentRuntimeEndedAt),
		AgentRuntimeError:       agentRuntimeError,
		AgentRuntimeResult:      agentRuntimeResult,
		AgentRuntimePrompt:      agentRuntimePrompt,
		AgentRuntimeAgentType:   agentRuntimeAgentType,
		TraceID:                 traceID,
		PromptTokens:            promptTokens,
		CompletionTokens:        completionTokens,
		TotalTokens:             totalTokens,
		ProgressData:            progressData,
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
