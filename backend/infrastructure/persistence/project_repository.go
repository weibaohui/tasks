package persistence

import (
	"fmt"
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteProjectRepository struct {
	db *sql.DB
}

func NewSQLiteProjectRepository(db *sql.DB) *SQLiteProjectRepository {
	return &SQLiteProjectRepository{db: db}
}

func (r *SQLiteProjectRepository) Save(ctx context.Context, project *domain.Project) error {
	snap := project.ToSnapshot()
	initStepsJSON, _ := json.Marshal(snap.InitSteps)
	query := `
		INSERT INTO projects (id, name, git_repo_url, default_branch, init_steps, dispatch_channel_code, dispatch_session_key, default_agent_code, max_concurrent_agents, heartbeat_scenario_code, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			git_repo_url=excluded.git_repo_url,
			default_branch=excluded.default_branch,
			init_steps=excluded.init_steps,
			dispatch_channel_code=excluded.dispatch_channel_code,
			dispatch_session_key=excluded.dispatch_session_key,
			default_agent_code=excluded.default_agent_code,
			max_concurrent_agents=excluded.max_concurrent_agents,
			heartbeat_scenario_code=excluded.heartbeat_scenario_code,
			updated_at=excluded.updated_at
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		snap.ID.String(),
		snap.Name,
		snap.GitRepoURL,
		snap.DefaultBranch,
		string(initStepsJSON),
		snap.DispatchChannelCode,
		snap.DispatchSessionKey,
		snap.DefaultAgentCode,
		snap.MaxConcurrentAgents,
		snap.HeartbeatScenarioCode,
		snap.CreatedAt.Unix(),
		snap.UpdatedAt.Unix(),
	)
	return err
}

func (r *SQLiteProjectRepository) FindByID(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, git_repo_url, default_branch, init_steps, dispatch_channel_code, dispatch_session_key, COALESCE(default_agent_code, ''), max_concurrent_agents, COALESCE(heartbeat_scenario_code, ''), created_at, updated_at
		FROM projects WHERE id = ?`, id.String())
	return scanProject(row)
}

func (r *SQLiteProjectRepository) FindAll(ctx context.Context) ([]*domain.Project, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, git_repo_url, default_branch, init_steps, dispatch_channel_code, dispatch_session_key, COALESCE(default_agent_code, ''), max_concurrent_agents, COALESCE(heartbeat_scenario_code, ''), created_at, updated_at
		FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	projects := make([]*domain.Project, 0)
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		if project != nil {
			projects = append(projects, project)
		}
	}
	return projects, rows.Err()
}

func (r *SQLiteProjectRepository) Delete(ctx context.Context, id domain.ProjectID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id.String())
	return err
}

func scanProject(scanner rowScanner) (*domain.Project, error) {
	var (
		idStr                 string
		name                  string
		gitRepoURL            string
		defaultBranch         string
		initStepsJSON         []byte
		dispatchChannelCode   string
		dispatchSessionKey    string
		defaultAgentCode      string
		maxConcurrentAgents   int
		heartbeatScenarioCode string
		createdAtUnix         int64
		updatedAtUnix         int64
	)
	err := scanner.Scan(&idStr, &name, &gitRepoURL, &defaultBranch, &initStepsJSON, &dispatchChannelCode, &dispatchSessionKey, &defaultAgentCode, &maxConcurrentAgents, &heartbeatScenarioCode, &createdAtUnix, &updatedAtUnix)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var initSteps []string
	if err := json.Unmarshal(initStepsJSON, &initSteps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal init_steps: %w", err)
	}
	project := &domain.Project{}
	project.FromSnapshot(domain.ProjectSnapshot{
		ID:                    domain.NewProjectID(idStr),
		Name:                  name,
		GitRepoURL:            gitRepoURL,
		DefaultBranch:         defaultBranch,
		InitSteps:             initSteps,
		DispatchChannelCode:   dispatchChannelCode,
		DispatchSessionKey:    dispatchSessionKey,
		DefaultAgentCode:      defaultAgentCode,
		MaxConcurrentAgents:   maxConcurrentAgents,
		HeartbeatScenarioCode: heartbeatScenarioCode,
		CreatedAt:             time.Unix(createdAtUnix, 0),
		UpdatedAt:             time.Unix(updatedAtUnix, 0),
	})
	return project, nil
}
