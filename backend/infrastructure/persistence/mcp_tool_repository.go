package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteMCPToolRepository struct {
	db *sql.DB
}

// NewSQLiteMCPToolRepository 创建 MCP 工具仓储（SQLite 实现）
func NewSQLiteMCPToolRepository(db *sql.DB) *SQLiteMCPToolRepository {
	return &SQLiteMCPToolRepository{db: db}
}

func (r *SQLiteMCPToolRepository) Create(ctx context.Context, tool *domain.MCPToolModel) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO mcp_tools(id, mcp_server_id, name, description, input_schema, created_at, updated_at, deleted_at)
VALUES(?, ?, ?, ?, ?, ?, ?, NULL)
`, tool.ID, tool.MCPServerID.String(), tool.Name, tool.Description, tool.InputSchema, tool.CreatedAt.Unix(), tool.UpdatedAt.Unix())
	return err
}

func (r *SQLiteMCPToolRepository) DeleteByServerID(ctx context.Context, serverID domain.MCPServerID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mcp_tools WHERE mcp_server_id = ?`, serverID.String())
	return err
}

func (r *SQLiteMCPToolRepository) ListByServerID(ctx context.Context, serverID domain.MCPServerID) ([]*domain.MCPToolModel, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, mcp_server_id, name, description, input_schema, created_at, updated_at, deleted_at FROM mcp_tools WHERE mcp_server_id = ? ORDER BY created_at DESC`, serverID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*domain.MCPToolModel
	for rows.Next() {
		var (
			id            string
			serverIDStr   string
			name          string
			description   sql.NullString
			inputSchema   sql.NullString
			createdAtUnix int64
			updatedAtUnix int64
			deletedAtUnix sql.NullInt64
		)
		if err := rows.Scan(&id, &serverIDStr, &name, &description, &inputSchema, &createdAtUnix, &updatedAtUnix, &deletedAtUnix); err != nil {
			return nil, err
		}
		var deletedAt *time.Time
		if deletedAtUnix.Valid {
			v := time.Unix(deletedAtUnix.Int64, 0)
			deletedAt = &v
		}
		list = append(list, &domain.MCPToolModel{
			ID:          id,
			MCPServerID: domain.NewMCPServerID(serverIDStr),
			Name:        name,
			Description: description.String,
			InputSchema: inputSchema.String,
			CreatedAt:   time.Unix(createdAtUnix, 0),
			UpdatedAt:   time.Unix(updatedAtUnix, 0),
			DeletedAt:   deletedAt,
		})
	}
	return list, rows.Err()
}
