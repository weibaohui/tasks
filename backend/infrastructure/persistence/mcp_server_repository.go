package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteMCPServerRepository struct {
	db *sql.DB
}

// NewSQLiteMCPServerRepository 创建 MCP 服务器仓储（SQLite 实现）
func NewSQLiteMCPServerRepository(db *sql.DB) *SQLiteMCPServerRepository {
	return &SQLiteMCPServerRepository{db: db}
}

func (r *SQLiteMCPServerRepository) Create(ctx context.Context, server *domain.MCPServer) error {
	s := server.ToSnapshot()
	argsJSON, _ := json.Marshal(s.Args)
	envJSON, _ := json.Marshal(s.EnvVars)
	capsJSON, _ := json.Marshal(s.Capabilities)
	var lastConnected *int64
	if s.LastConnected != nil {
		v := s.LastConnected.Unix()
		lastConnected = &v
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO mcp_servers(
  id, code, name, description, transport_type, command, args, url, env_vars, status, capabilities, last_connected_at, error_message, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, s.ID.String(), s.Code, s.Name, s.Description, string(s.TransportType), s.Command, string(argsJSON), s.URL, string(envJSON), s.Status, string(capsJSON), lastConnected, s.ErrorMessage, s.CreatedAt.Unix(), s.UpdatedAt.Unix())
	return err
}

func (r *SQLiteMCPServerRepository) Update(ctx context.Context, server *domain.MCPServer) error {
	s := server.ToSnapshot()
	argsJSON, _ := json.Marshal(s.Args)
	envJSON, _ := json.Marshal(s.EnvVars)
	capsJSON, _ := json.Marshal(s.Capabilities)
	var lastConnected *int64
	if s.LastConnected != nil {
		v := s.LastConnected.Unix()
		lastConnected = &v
	}
	_, err := r.db.ExecContext(ctx, `
UPDATE mcp_servers SET
  code=?, name=?, description=?, transport_type=?, command=?, args=?, url=?, env_vars=?, status=?, capabilities=?, last_connected_at=?, error_message=?, updated_at=?
WHERE id=?
`, s.Code, s.Name, s.Description, string(s.TransportType), s.Command, string(argsJSON), s.URL, string(envJSON), s.Status, string(capsJSON), lastConnected, s.ErrorMessage, s.UpdatedAt.Unix(), s.ID.String())
	return err
}

func (r *SQLiteMCPServerRepository) Delete(ctx context.Context, id domain.MCPServerID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mcp_servers WHERE id = ?`, id.String())
	return err
}

func (r *SQLiteMCPServerRepository) GetByID(ctx context.Context, id domain.MCPServerID) (*domain.MCPServer, error) {
	row := r.db.QueryRowContext(ctx, `SELECT * FROM mcp_servers WHERE id = ?`, id.String())
	return scanMCPServer(row)
}

func (r *SQLiteMCPServerRepository) GetByCode(ctx context.Context, code string) (*domain.MCPServer, error) {
	row := r.db.QueryRowContext(ctx, `SELECT * FROM mcp_servers WHERE code = ?`, code)
	return scanMCPServer(row)
}

func (r *SQLiteMCPServerRepository) List(ctx context.Context) ([]*domain.MCPServer, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT * FROM mcp_servers ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMCPServers(rows)
}

func (r *SQLiteMCPServerRepository) ListByStatus(ctx context.Context, status string) ([]*domain.MCPServer, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT * FROM mcp_servers WHERE status = ? ORDER BY created_at DESC`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMCPServers(rows)
}

func (r *SQLiteMCPServerRepository) CheckCodeExists(ctx context.Context, code string) (bool, error) {
	var cnt int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM mcp_servers WHERE code = ?`, code).Scan(&cnt); err != nil {
		return false, err
	}
	return cnt > 0, nil
}

type mcpRowScanner interface {
	Scan(dest ...any) error
}

func scanMCPServers(rows *sql.Rows) ([]*domain.MCPServer, error) {
	var list []*domain.MCPServer
	for rows.Next() {
		item, err := scanMCPServer(rows)
		if err != nil {
			return nil, err
		}
		if item != nil {
			list = append(list, item)
		}
	}
	return list, rows.Err()
}

func scanMCPServer(scanner mcpRowScanner) (*domain.MCPServer, error) {
	var (
		id              string
		code            string
		name            string
		description     sql.NullString
		transportType   string
		command         sql.NullString
		argsJSON        sql.NullString
		url             sql.NullString
		envJSON         sql.NullString
		status          string
		capsJSON        sql.NullString
		lastConnectedAt sql.NullInt64
		errorMessage    sql.NullString
		createdAtUnix   int64
		updatedAtUnix   int64
	)
	err := scanner.Scan(
		&id, &code, &name, &description, &transportType, &command, &argsJSON, &url, &envJSON, &status, &capsJSON, &lastConnectedAt, &errorMessage, &createdAtUnix, &updatedAtUnix,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	args := []string{}
	if argsJSON.Valid && argsJSON.String != "" && argsJSON.String != "null" {
		_ = json.Unmarshal([]byte(argsJSON.String), &args)
	}
	envVars := map[string]string{}
	if envJSON.Valid && envJSON.String != "" && envJSON.String != "null" {
		_ = json.Unmarshal([]byte(envJSON.String), &envVars)
	}
	caps := []domain.MCPTool{}
	if capsJSON.Valid && capsJSON.String != "" && capsJSON.String != "null" {
		_ = json.Unmarshal([]byte(capsJSON.String), &caps)
	}
	var last *time.Time
	if lastConnectedAt.Valid && lastConnectedAt.Int64 > 0 {
		v := time.Unix(lastConnectedAt.Int64, 0)
		last = &v
	}
	entity, _ := domain.NewMCPServer(domain.NewMCPServerID(id), code, name, domain.MCPTransportType(transportType))
	entity.UpdateProfile(domain.MCPProfileUpdate{
		Name: name, Description: description.String, Transport: domain.MCPTransportType(transportType),
		Command: command.String, URL: url.String, Args: args, EnvVars: envVars,
	})
	entity.SetStatus(status, errorMessage.String)
	entity.SetCapabilities(caps)
	// patch timestamps
	snap := entity.ToSnapshot()
	snap.CreatedAt = time.Unix(createdAtUnix, 0)
	snap.UpdatedAt = time.Unix(updatedAtUnix, 0)
	snap.LastConnected = last
	entity.FromSnapshot(snap)
	return entity, nil
}
