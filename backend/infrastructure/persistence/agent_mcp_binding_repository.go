package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteAgentMCPBindingRepository struct {
	db *sql.DB
}

// NewSQLiteAgentMCPBindingRepository 创建 Agent-MCP 绑定仓储（SQLite 实现）
func NewSQLiteAgentMCPBindingRepository(db *sql.DB) *SQLiteAgentMCPBindingRepository {
	return &SQLiteAgentMCPBindingRepository{db: db}
}

func (r *SQLiteAgentMCPBindingRepository) Create(ctx context.Context, binding *domain.AgentMCPBinding) error {
	s := binding.ToSnapshot()
	var toolsJSON *string
	if s.EnabledTools != nil {
		b, _ := json.Marshal(s.EnabledTools)
		str := string(b)
		toolsJSON = &str
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO agent_mcp_bindings(
  id, agent_id, mcp_server_id, enabled_tools, is_active, auto_load, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, s.ID.String(), s.AgentID.String(), s.MCPServerID.String(), toolsJSON, ternaryInt(s.IsActive), ternaryInt(s.AutoLoad), s.CreatedAt.Unix(), s.UpdatedAt.Unix())
	return err
}

func (r *SQLiteAgentMCPBindingRepository) Update(ctx context.Context, binding *domain.AgentMCPBinding) error {
	s := binding.ToSnapshot()
	var toolsJSON *string
	if s.EnabledTools != nil {
		b, _ := json.Marshal(s.EnabledTools)
		str := string(b)
		toolsJSON = &str
	}
	_, err := r.db.ExecContext(ctx, `
UPDATE agent_mcp_bindings SET enabled_tools=?, is_active=?, auto_load=?, updated_at=? WHERE id=?
`, toolsJSON, ternaryInt(s.IsActive), ternaryInt(s.AutoLoad), s.UpdatedAt.Unix(), s.ID.String())
	return err
}

func (r *SQLiteAgentMCPBindingRepository) Delete(ctx context.Context, id domain.AgentMCPBindingID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM agent_mcp_bindings WHERE id = ?`, id.String())
	return err
}

func (r *SQLiteAgentMCPBindingRepository) DeleteByAgentAndMCPServer(ctx context.Context, agentID domain.AgentID, serverID domain.MCPServerID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM agent_mcp_bindings WHERE agent_id = ? AND mcp_server_id = ?`, agentID.String(), serverID.String())
	return err
}

func (r *SQLiteAgentMCPBindingRepository) GetByID(ctx context.Context, id domain.AgentMCPBindingID) (*domain.AgentMCPBinding, error) {
	row := r.db.QueryRowContext(ctx, `SELECT * FROM agent_mcp_bindings WHERE id = ?`, id.String())
	return scanBinding(row)
}

func (r *SQLiteAgentMCPBindingRepository) GetByAgentID(ctx context.Context, agentID domain.AgentID) ([]*domain.AgentMCPBinding, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT * FROM agent_mcp_bindings WHERE agent_id = ? ORDER BY created_at DESC`, agentID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*domain.AgentMCPBinding
	for rows.Next() {
		item, err := scanBinding(rows)
		if err != nil {
			return nil, err
		}
		if item != nil {
			list = append(list, item)
		}
	}
	return list, rows.Err()
}

func (r *SQLiteAgentMCPBindingRepository) CheckExists(ctx context.Context, agentID domain.AgentID, serverID domain.MCPServerID) (bool, error) {
	var cnt int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM agent_mcp_bindings WHERE agent_id = ? AND mcp_server_id = ?`, agentID.String(), serverID.String()).Scan(&cnt); err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func scanBinding(scanner interface {
	Scan(dest ...any) error
}) (*domain.AgentMCPBinding, error) {
	var (
		id            string
		agentID       string
		serverID      string
		enabledTools  sql.NullString
		isActiveInt   int
		autoLoadInt   int
		createdAtUnix int64
		updatedAtUnix int64
	)
	if err := scanner.Scan(&id, &agentID, &serverID, &enabledTools, &isActiveInt, &autoLoadInt, &createdAtUnix, &updatedAtUnix); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	entity := domain.NewAgentMCPBinding(domain.NewAgentMCPBindingID(id), domain.NewAgentID(agentID), domain.NewMCPServerID(serverID))
	if enabledTools.Valid && enabledTools.String != "" && enabledTools.String != "null" {
		var tools []string
		_ = json.Unmarshal([]byte(enabledTools.String), &tools)
		entity.SetEnabledTools(tools)
	}
	entity.SetActive(isActiveInt == 1)
	entity.SetAutoLoad(autoLoadInt == 1)
	// patch timestamps
	s := entity.ToSnapshot()
	s.CreatedAt = time.Unix(createdAtUnix, 0)
	s.UpdatedAt = time.Unix(updatedAtUnix, 0)
	entity.FromSnapshot(s)
	return entity, nil
}

func ternaryInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
