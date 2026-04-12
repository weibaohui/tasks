package persistence

import (
	"fmt"
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteAgentRepository struct {
	db *sql.DB
}

func NewSQLiteAgentRepository(db *sql.DB) *SQLiteAgentRepository {
	return &SQLiteAgentRepository{db: db}
}

func (r *SQLiteAgentRepository) Save(ctx context.Context, agent *domain.Agent) error {
	snap := agent.ToSnapshot()
	skillsJSON, _ := json.Marshal(snap.SkillsList)
	toolsJSON, _ := json.Marshal(snap.ToolsList)

	// 序列化 claude_code_config
	var configJSON []byte
	// Handle nullable LLMProviderID
	var llmProviderID interface{}
	if snap.LLMProviderID.String() != "" {
		llmProviderID = snap.LLMProviderID.String()
	} else {
		llmProviderID = nil
	}
var err error
	if snap.ClaudeCodeConfig != nil {
		configJSON, err = json.Marshal(snap.ClaudeCodeConfig)
		if err != nil {
			configJSON = []byte("{}")
		}
	} else {
		configJSON = []byte("{}")
	}

	query := `
		INSERT INTO agents (
			id, agent_code, user_code, name, description, identity_content, soul_content, agents_content,
			user_content, tools_content, model, llm_provider_id, max_tokens, temperature, max_iterations, history_messages,
			skills_list, tools_list, is_active, is_default, enable_thinking_process, agent_type, shadow_from, created_at, updated_at,
			claude_code_config
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			description=excluded.description,
			identity_content=excluded.identity_content,
			soul_content=excluded.soul_content,
			agents_content=excluded.agents_content,
			user_content=excluded.user_content,
			tools_content=excluded.tools_content,
			model=excluded.model,
			llm_provider_id=excluded.llm_provider_id,
			max_tokens=excluded.max_tokens,
			temperature=excluded.temperature,
			max_iterations=excluded.max_iterations,
			history_messages=excluded.history_messages,
			skills_list=excluded.skills_list,
			tools_list=excluded.tools_list,
			is_active=excluded.is_active,
			is_default=excluded.is_default,
			enable_thinking_process=excluded.enable_thinking_process,
			agent_type=excluded.agent_type,
			shadow_from=excluded.shadow_from,
			updated_at=excluded.updated_at,
			claude_code_config=excluded.claude_code_config
	`

	_, err = r.db.ExecContext(
		ctx,
		query,
		snap.ID.String(),
		snap.AgentCode.String(),
		snap.UserCode,
		snap.Name,
		snap.Description,
		snap.IdentityContent,
		snap.SoulContent,
		snap.AgentsContent,
		snap.UserContent,
		snap.ToolsContent,
		snap.Model,
		llmProviderID,
		snap.MaxTokens,
		snap.Temperature,
		snap.MaxIterations,
		snap.HistoryMessages,
		skillsJSON,
		toolsJSON,
		boolToInt(snap.IsActive),
		boolToInt(snap.IsDefault),
		boolToInt(snap.EnableThinkingProcess),
		snap.AgentType.String(),
		snap.ShadowFrom,
		snap.CreatedAt.Unix(),
		snap.UpdatedAt.Unix(),
		string(configJSON),
	)
	return err
}

func (r *SQLiteAgentRepository) FindByID(ctx context.Context, id domain.AgentID) (*domain.Agent, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, agent_code, user_code, name,
		COALESCE(description, '') as description,
		COALESCE(identity_content, '') as identity_content,
		COALESCE(soul_content, '') as soul_content,
		COALESCE(agents_content, '') as agents_content,
		COALESCE(user_content, '') as user_content,
		COALESCE(tools_content, '') as tools_content,
		COALESCE(model, '') as model,
		llm_provider_id,
		max_tokens, temperature, max_iterations, history_messages,
		COALESCE(skills_list, '[]') as skills_list,
		COALESCE(tools_list, '[]') as tools_list,
		is_active, is_default, enable_thinking_process, agent_type,
		COALESCE(shadow_from, '') as shadow_from,
		created_at, updated_at,
		COALESCE(claude_code_config, '{}') as claude_code_config
		FROM agents WHERE id = ?`, id.String())
	return scanAgent(row)
}

func (r *SQLiteAgentRepository) FindByAgentCode(ctx context.Context, code domain.AgentCode) (*domain.Agent, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, agent_code, user_code, name,
		COALESCE(description, '') as description,
		COALESCE(identity_content, '') as identity_content,
		COALESCE(soul_content, '') as soul_content,
		COALESCE(agents_content, '') as agents_content,
		COALESCE(user_content, '') as user_content,
		COALESCE(tools_content, '') as tools_content,
		COALESCE(model, '') as model,
		llm_provider_id,
		max_tokens, temperature, max_iterations, history_messages,
		COALESCE(skills_list, '[]') as skills_list,
		COALESCE(tools_list, '[]') as tools_list,
		is_active, is_default, enable_thinking_process, agent_type,
		COALESCE(shadow_from, '') as shadow_from,
		created_at, updated_at,
		COALESCE(claude_code_config, '{}') as claude_code_config
		FROM agents WHERE agent_code = ?`, code.String())
	return scanAgent(row)
}

func (r *SQLiteAgentRepository) FindByUserCode(ctx context.Context, userCode string) ([]*domain.Agent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, agent_code, user_code, name,
		COALESCE(description, '') as description,
		COALESCE(identity_content, '') as identity_content,
		COALESCE(soul_content, '') as soul_content,
		COALESCE(agents_content, '') as agents_content,
		COALESCE(user_content, '') as user_content,
		COALESCE(tools_content, '') as tools_content,
		COALESCE(model, '') as model,
		llm_provider_id,
		max_tokens, temperature, max_iterations, history_messages,
		COALESCE(skills_list, '[]') as skills_list,
		COALESCE(tools_list, '[]') as tools_list,
		is_active, is_default, enable_thinking_process, agent_type,
		COALESCE(shadow_from, '') as shadow_from,
		created_at, updated_at,
		COALESCE(claude_code_config, '{}') as claude_code_config
		FROM agents WHERE user_code = ? ORDER BY created_at DESC`, userCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAgents(rows)
}

func (r *SQLiteAgentRepository) FindAll(ctx context.Context) ([]*domain.Agent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, agent_code, user_code, name,
		COALESCE(description, '') as description,
		COALESCE(identity_content, '') as identity_content,
		COALESCE(soul_content, '') as soul_content,
		COALESCE(agents_content, '') as agents_content,
		COALESCE(user_content, '') as user_content,
		COALESCE(tools_content, '') as tools_content,
		COALESCE(model, '') as model,
		llm_provider_id,
		max_tokens, temperature, max_iterations, history_messages,
		COALESCE(skills_list, '[]') as skills_list,
		COALESCE(tools_list, '[]') as tools_list,
		is_active, is_default, enable_thinking_process, agent_type,
		COALESCE(shadow_from, '') as shadow_from,
		created_at, updated_at,
		COALESCE(claude_code_config, '{}') as claude_code_config
		FROM agents ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAgents(rows)
}

func (r *SQLiteAgentRepository) Delete(ctx context.Context, id domain.AgentID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM agents WHERE id = ?`, id.String())
	return err
}

func scanAgents(rows *sql.Rows) ([]*domain.Agent, error) {
	agents := make([]*domain.Agent, 0)
	for rows.Next() {
		agent, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		if agent != nil {
			agents = append(agents, agent)
		}
	}
	return agents, rows.Err()
}

func scanAgent(scanner rowScanner) (*domain.Agent, error) {
	var (
		idStr                string
		agentCodeStr         string
		userCode             string
		name                 string
		description          string
		identityContent      string
		soulContent          string
		agentsContent        string
		userContent          string
		toolsContent         string
		model                string
		llmProviderIDStr     sql.NullString
		maxTokens            int
		temperature          float64
		maxIterations        int
		historyMessages      int
		skillsJSON           []byte
		toolsJSON            []byte
		isActiveInt          int
		isDefaultInt         int
		enableThinkingInt    int
		agentTypeStr         string
		shadowFrom           string
		createdAtUnix        int64
		updatedAtUnix        int64
		claudeCodeConfigJSON []byte
	)

	err := scanner.Scan(
		&idStr,
		&agentCodeStr,
		&userCode,
		&name,
		&description,
		&identityContent,
		&soulContent,
		&agentsContent,
		&userContent,
		&toolsContent,
		&model,
		&llmProviderIDStr,
		&maxTokens,
		&temperature,
		&maxIterations,
		&historyMessages,
		&skillsJSON,
		&toolsJSON,
		&isActiveInt,
		&isDefaultInt,
		&enableThinkingInt,
		&agentTypeStr,
		&shadowFrom,
		&createdAtUnix,
		&updatedAtUnix,
		&claudeCodeConfigJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var skills []string
	var tools []string
	if err := json.Unmarshal(skillsJSON, &skills); err != nil {
		return nil, fmt.Errorf("failed to unmarshal skills: %w", err)
	}
	if err := json.Unmarshal(toolsJSON, &tools); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools: %w", err)
	}

	var claudeCodeConfig *domain.ClaudeCodeConfig
	if len(claudeCodeConfigJSON) > 0 && string(claudeCodeConfigJSON) != "{}" {
		claudeCodeConfig = &domain.ClaudeCodeConfig{}
		if err := json.Unmarshal(claudeCodeConfigJSON, claudeCodeConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal claude_code_config: %w", err)
		}
	}

	// Handle nullable LLMProviderID
	var llmProviderID string
	if llmProviderIDStr.Valid {
		llmProviderID = llmProviderIDStr.String
	}

	agent := &domain.Agent{}
	agent.FromSnapshot(domain.AgentSnapshot{
		ID:                    domain.NewAgentID(idStr),
		AgentCode:             domain.NewAgentCode(agentCodeStr),
		AgentType:             domain.AgentType(agentTypeStr),
		UserCode:              userCode,
		Name:                  name,
		Description:           description,
		IdentityContent:       identityContent,
		SoulContent:           soulContent,
		AgentsContent:         agentsContent,
		UserContent:           userContent,
		ToolsContent:          toolsContent,
		Model:                 model,
		LLMProviderID:         domain.NewLLMProviderID(llmProviderID),
		MaxTokens:             maxTokens,
		Temperature:           temperature,
		MaxIterations:         maxIterations,
		HistoryMessages:       historyMessages,
		SkillsList:            skills,
		ToolsList:             tools,
		IsActive:              isActiveInt == 1,
		IsDefault:             isDefaultInt == 1,
		EnableThinkingProcess: enableThinkingInt == 1,
		ShadowFrom:            shadowFrom,
		ClaudeCodeConfig:      claudeCodeConfig,
		CreatedAt:             time.Unix(createdAtUnix, 0),
		UpdatedAt:             time.Unix(updatedAtUnix, 0),
	})
	return agent, nil
}
