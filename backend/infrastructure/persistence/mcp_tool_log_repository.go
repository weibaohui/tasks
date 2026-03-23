package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteMCPToolLogRepository struct {
	db *sql.DB
}

// NewSQLiteMCPToolLogRepository 创建 MCP 工具日志仓储（SQLite 实现）
func NewSQLiteMCPToolLogRepository(db *sql.DB) *SQLiteMCPToolLogRepository {
	return &SQLiteMCPToolLogRepository{db: db}
}

func (r *SQLiteMCPToolLogRepository) Create(ctx context.Context, log *domain.MCPToolLog) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO mcp_tool_logs(id, session_key, mcp_server_id, tool_name, parameters, result, error_message, execute_time, created_at)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
`, log.ID, log.SessionKey, log.MCPServerID.String(), log.ToolName, log.Parameters, log.Result, log.ErrorMsg, log.ExecuteTime, log.CreatedAt.Unix())
	return err
}

func (r *SQLiteMCPToolLogRepository) ListByServerID(ctx context.Context, serverID domain.MCPServerID, limit int) ([]*domain.MCPToolLog, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, session_key, mcp_server_id, tool_name, parameters, result, error_message, execute_time, created_at FROM mcp_tool_logs WHERE mcp_server_id = ? ORDER BY created_at DESC LIMIT ?`, serverID.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*domain.MCPToolLog
	for rows.Next() {
		var (
			id            string
			sessionKey    string
			serverIDStr   string
			toolName      string
			parameters    sql.NullString
			result        sql.NullString
			errorMsg      sql.NullString
			executeTime   uint
			createdAtUnix int64
		)
		if err := rows.Scan(&id, &sessionKey, &serverIDStr, &toolName, &parameters, &result, &errorMsg, &executeTime, &createdAtUnix); err != nil {
			return nil, err
		}
		list = append(list, &domain.MCPToolLog{
			ID:          id,
			SessionKey:  sessionKey,
			MCPServerID: domain.NewMCPServerID(serverIDStr),
			ToolName:    toolName,
			Parameters:  parameters.String,
			Result:      result.String,
			ErrorMsg:    errorMsg.String,
			ExecuteTime: executeTime,
			CreatedAt:   time.Unix(createdAtUnix, 0),
		})
	}
	return list, rows.Err()
}
