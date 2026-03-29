/**
 * 数据库 Schema 定义
 */
package persistence

import (
	"database/sql"
	"fmt"
)

// Schema SQL Schema 定义
const Schema = `
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL,
    span_id TEXT NOT NULL,
    parent_id TEXT,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL,
    acceptance_criteria TEXT,
    task_requirement TEXT,
    task_conclusion TEXT,
    user_code TEXT,
    agent_code TEXT,
    channel_code TEXT,
    session_key TEXT,
    todo_list TEXT,
    analysis TEXT,
    subtask_records TEXT,
    depth INTEGER NOT NULL DEFAULT 0,
    parent_span TEXT,
    timeout INTEGER NOT NULL,
    max_retries INTEGER NOT NULL,
    priority INTEGER NOT NULL,
    status INTEGER NOT NULL,
    progress INTEGER,
    error_msg TEXT,
    created_at INTEGER NOT NULL,
    started_at INTEGER,
    finished_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_tasks_trace_id ON tasks(trace_id);
CREATE INDEX IF NOT EXISTS idx_tasks_parent_id ON tasks(parent_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_user_code ON tasks(user_code);
CREATE INDEX IF NOT EXISTS idx_tasks_agent_code ON tasks(agent_code);
CREATE INDEX IF NOT EXISTS idx_tasks_channel_code ON tasks(channel_code);
CREATE INDEX IF NOT EXISTS idx_tasks_session_key ON tasks(session_key);

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    user_code TEXT NOT NULL UNIQUE,
    username TEXT NOT NULL UNIQUE,
    email TEXT,
    display_name TEXT,
    password_hash TEXT NOT NULL,
    is_active INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_user_code ON users(user_code);

CREATE TABLE IF NOT EXISTS agents (
    id TEXT PRIMARY KEY,
    agent_code TEXT NOT NULL UNIQUE,
    user_code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    identity_content TEXT,
    soul_content TEXT,
    agents_content TEXT,
    user_content TEXT,
    tools_content TEXT,
    model TEXT,
    max_tokens INTEGER NOT NULL,
    temperature REAL NOT NULL,
    max_iterations INTEGER NOT NULL,
    history_messages INTEGER NOT NULL,
    skills_list TEXT,
    tools_list TEXT,
    is_active INTEGER NOT NULL,
    is_default INTEGER NOT NULL,
    enable_thinking_process INTEGER NOT NULL,
    agent_type TEXT NOT NULL DEFAULT 'BareLLM',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agents_agent_code ON agents(agent_code);
CREATE INDEX IF NOT EXISTS idx_agents_user_code ON agents(user_code);

CREATE TABLE IF NOT EXISTS llm_providers (
    id TEXT PRIMARY KEY,
    user_code TEXT NOT NULL,
    provider_key TEXT NOT NULL,
    provider_name TEXT,
    api_key TEXT,
    api_base TEXT,
    provider_type TEXT NOT NULL DEFAULT 'openai',
    extra_headers TEXT,
    supported_models TEXT,
    default_model TEXT,
    is_default INTEGER NOT NULL,
    priority INTEGER NOT NULL,
    auto_merge INTEGER NOT NULL,
    embedding_models TEXT,
    default_embedding_model TEXT,
    is_active INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_llm_providers_user_code ON llm_providers(user_code);
CREATE INDEX IF NOT EXISTS idx_llm_providers_default ON llm_providers(is_default, is_active);

CREATE TABLE IF NOT EXISTS channels (
    id TEXT PRIMARY KEY,
    channel_code TEXT NOT NULL UNIQUE,
    user_code TEXT NOT NULL,
    agent_code TEXT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    is_active INTEGER NOT NULL,
    allow_from TEXT,
    config TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_channels_channel_code ON channels(channel_code);
CREATE INDEX IF NOT EXISTS idx_channels_user_code ON channels(user_code);
CREATE INDEX IF NOT EXISTS idx_channels_agent_code ON channels(agent_code);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_code TEXT NOT NULL,
    agent_code TEXT,
    channel_code TEXT NOT NULL,
    session_key TEXT NOT NULL UNIQUE,
    external_id TEXT,
    last_active_at INTEGER,
    metadata TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_session_key ON sessions(session_key);
CREATE INDEX IF NOT EXISTS idx_sessions_user_code ON sessions(user_code);
CREATE INDEX IF NOT EXISTS idx_sessions_channel_code ON sessions(channel_code);

CREATE TABLE IF NOT EXISTS cron_jobs (
    id TEXT PRIMARY KEY,
    user_code TEXT NOT NULL,
    channel_code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    cron_expression TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',
    prompt TEXT NOT NULL,
    model_selection_mode TEXT NOT NULL DEFAULT 'auto',
    model_id TEXT,
    model_name TEXT,
    target_channel_code TEXT,
    target_user_code TEXT,
    is_active INTEGER NOT NULL,
    last_run_at INTEGER,
    last_run_status TEXT,
    last_run_result TEXT,
    next_run_at INTEGER,
    run_count INTEGER NOT NULL DEFAULT 0,
    fail_count INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_cron_jobs_user_code ON cron_jobs(user_code);
CREATE INDEX IF NOT EXISTS idx_cron_jobs_channel_code ON cron_jobs(channel_code);
CREATE INDEX IF NOT EXISTS idx_cron_jobs_next_run_at ON cron_jobs(next_run_at);
CREATE INDEX IF NOT EXISTS idx_cron_jobs_is_active ON cron_jobs(is_active);

CREATE TABLE IF NOT EXISTS conversation_records (
    id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL,
    span_id TEXT,
    parent_span_id TEXT,
    event_type TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    session_key TEXT,
    role TEXT,
    content TEXT,
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    reasoning_tokens INTEGER NOT NULL DEFAULT 0,
    cached_tokens INTEGER NOT NULL DEFAULT 0,
    user_code TEXT,
    agent_code TEXT,
    channel_code TEXT,
    channel_type TEXT,
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_conv_records_event_type ON conversation_records(event_type);
CREATE INDEX IF NOT EXISTS idx_conv_records_session_key ON conversation_records(session_key);
CREATE INDEX IF NOT EXISTS idx_conv_records_timestamp ON conversation_records(timestamp);
CREATE INDEX IF NOT EXISTS idx_conv_records_trace_id ON conversation_records(trace_id);
CREATE INDEX IF NOT EXISTS idx_conv_records_role ON conversation_records(role);
CREATE INDEX IF NOT EXISTS idx_conv_records_user_code ON conversation_records(user_code);
CREATE INDEX IF NOT EXISTS idx_conv_records_agent_code ON conversation_records(agent_code);
CREATE INDEX IF NOT EXISTS idx_conv_records_channel_code ON conversation_records(channel_code);
CREATE INDEX IF NOT EXISTS idx_conv_records_channel_type ON conversation_records(channel_type);
CREATE INDEX IF NOT EXISTS idx_conv_records_user_code_timestamp ON conversation_records(user_code, timestamp);
CREATE INDEX IF NOT EXISTS idx_conv_records_agent_code_timestamp ON conversation_records(agent_code, timestamp);
CREATE INDEX IF NOT EXISTS idx_conv_records_channel_code_timestamp ON conversation_records(channel_code, timestamp);
CREATE INDEX IF NOT EXISTS idx_conv_records_session_key_timestamp ON conversation_records(session_key, timestamp);

CREATE TABLE IF NOT EXISTS mcp_servers (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    transport_type TEXT NOT NULL,
    command TEXT,
    args TEXT,
    url TEXT,
    env_vars TEXT,
    status TEXT NOT NULL DEFAULT 'inactive',
    capabilities TEXT,
    last_connected_at INTEGER,
    error_message TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_mcp_servers_code ON mcp_servers(code);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_status ON mcp_servers(status);

CREATE TABLE IF NOT EXISTS agent_mcp_bindings (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    mcp_server_id TEXT NOT NULL,
    enabled_tools TEXT,
    is_active INTEGER NOT NULL DEFAULT 1,
    auto_load INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agent_mcp_bindings_agent_id ON agent_mcp_bindings(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_mcp_bindings_mcp_server_id ON agent_mcp_bindings(mcp_server_id);

CREATE TABLE IF NOT EXISTS mcp_tools (
    id TEXT PRIMARY KEY,
    mcp_server_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    input_schema TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    deleted_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_mcp_tools_mcp_server_id ON mcp_tools(mcp_server_id);
CREATE INDEX IF NOT EXISTS idx_mcp_tools_name ON mcp_tools(name);
CREATE INDEX IF NOT EXISTS idx_mcp_tools_deleted_at ON mcp_tools(deleted_at);

CREATE TABLE IF NOT EXISTS mcp_tool_logs (
    id TEXT PRIMARY KEY,
    session_key TEXT NOT NULL,
    mcp_server_id TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    parameters TEXT,
    result TEXT,
    error_message TEXT,
    execute_time INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_mcp_tool_logs_session_key ON mcp_tool_logs(session_key);
CREATE INDEX IF NOT EXISTS idx_mcp_tool_logs_mcp_server_id ON mcp_tool_logs(mcp_server_id);
CREATE INDEX IF NOT EXISTS idx_mcp_tool_logs_tool_name ON mcp_tool_logs(tool_name);
CREATE INDEX IF NOT EXISTS idx_mcp_tool_logs_created_at ON mcp_tool_logs(created_at);
`

// InitSchema 初始化数据库 Schema
func InitSchema(db *sql.DB) error {
	if _, err := db.Exec(Schema); err != nil {
		return err
	}
	if err := migrateTasksNewColumns(db); err != nil {
		return err
	}
	if err := migrateAgentMCPBindingColumn(db); err != nil {
		return err
	}
	if err := migrateLLMProviderTypeColumn(db); err != nil {
		return err
	}
	if err := migrateDropResultColumn(db); err != nil {
		return err
	}
	if err := migrateAgentTypeColumn(db); err != nil {
		return err
	}
	if err := migrateAgentProviderKeyColumn(db); err != nil {
		return err
	}
	if err := migrateTasksTimeoutToSeconds(db); err != nil {
		return err
	}
	return migrateConversationRecordsTimestampToMillis(db)
}

// migrateTasksNewColumns 迁移 tasks 表新增字段
func migrateTasksNewColumns(db *sql.DB) error {
	newColumns := []struct {
		name       string
		sqlType    string
		oldName    string // 如果旧列存在，数据迁移到新列
		defaultVal string // 默认值
	}{
		{"acceptance_criteria", "TEXT", "", ""},
		{"task_requirement", "TEXT", "llm_reason", ""},
		{"task_conclusion", "TEXT", "", ""},
		{"user_code", "TEXT", "", ""},
		{"agent_code", "TEXT", "", ""},
		{"channel_code", "TEXT", "", ""},
		{"session_key", "TEXT", "", ""},
		{"todo_list", "TEXT", "", ""},
		{"analysis", "TEXT", "", ""},
		{"depth", "INTEGER", "", "0"},
		{"parent_span", "TEXT", "", ""},
		{"subtask_records", "TEXT", "", ""},
	}

	for _, col := range newColumns {
		has, err := tableHasColumn(db, "tasks", col.name)
		if err != nil {
			return err
		}
		if !has {
			if col.oldName != "" {
				// 检查旧列是否存在，如果存在则重命名
				oldHas, err := tableHasColumn(db, "tasks", col.oldName)
				if err != nil {
					return err
				}
				if oldHas {
					// 重命名旧列为新列
					if _, err := db.Exec(fmt.Sprintf("ALTER TABLE tasks RENAME COLUMN %s TO %s", col.oldName, col.name)); err != nil {
						return err
					}
					continue
				}
			}
			// 添加新列
			sql := fmt.Sprintf("ALTER TABLE tasks ADD COLUMN %s %s", col.name, col.sqlType)
			if col.defaultVal != "" {
				sql += " NOT NULL DEFAULT " + col.defaultVal
			}
			if _, err := db.Exec(sql); err != nil {
				return err
			}
		}
	}

	return nil
}

// migrateAgentTypeColumn 迁移 agents 表新增 agent_type 字段
func migrateAgentTypeColumn(db *sql.DB) error {
	has, err := tableHasColumn(db, "agents", "agent_type")
	if err != nil {
		return err
	}
	if !has {
		if _, err := db.Exec("ALTER TABLE agents ADD COLUMN agent_type TEXT NOT NULL DEFAULT 'BareLLM'"); err != nil {
			return err
		}
	}
	return nil
}

// migrateAgentProviderKeyColumn 迁移 agents 表新增 provider_key 字段
func migrateAgentProviderKeyColumn(db *sql.DB) error {
	has, err := tableHasColumn(db, "agents", "provider_key")
	if err != nil {
		return err
	}
	if !has {
		if _, err := db.Exec("ALTER TABLE agents ADD COLUMN provider_key TEXT NOT NULL DEFAULT ''"); err != nil {
			return err
		}
	}
	return nil
}

// migrateDropResultColumn 删除 tasks 表的 result 列
func migrateDropResultColumn(db *sql.DB) error {
	has, err := tableHasColumn(db, "tasks", "result")
	if err != nil {
		return err
	}
	if has {
		if _, err := db.Exec("ALTER TABLE tasks DROP COLUMN result"); err != nil {
			return err
		}
	}
	return nil
}

// migrateTasksTimeoutToSeconds 将 tasks 表中 timeout 从毫秒转换为秒
// 通过检测 timeout > 1000 来判断是否为毫秒值（旧数据），并进行转换
func migrateTasksTimeoutToSeconds(db *sql.DB) error {
	// 检查是否已执行过迁移（通过检测是否有 timeout > 1000 且 < 1e10 的记录）
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE timeout > 1000 AND timeout < 10000000000`).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		// 没有需要转换的数据，可能是新数据或已转换
		return nil
	}
	// 转换：毫秒值 / 1000 = 秒值
	_, err = db.Exec(`UPDATE tasks SET timeout = timeout / 1000 WHERE timeout > 1000 AND timeout < 10000000000`)
	return err
}

func migrateAgentMCPBindingColumn(db *sql.DB) error {
	hasOld, err := tableHasColumn(db, "agent_mcp_bindings", "is_enabled")
	if err != nil {
		return err
	}
	hasNew, err := tableHasColumn(db, "agent_mcp_bindings", "is_active")
	if err != nil {
		return err
	}
	if hasOld && !hasNew {
		if _, err := db.Exec("ALTER TABLE agent_mcp_bindings RENAME COLUMN is_enabled TO is_active"); err != nil {
			return err
		}
	}
	return nil
}

func migrateLLMProviderTypeColumn(db *sql.DB) error {
	hasColumn, err := tableHasColumn(db, "llm_providers", "provider_type")
	if err != nil {
		return err
	}
	if !hasColumn {
		if _, err := db.Exec("ALTER TABLE llm_providers ADD COLUMN provider_type TEXT NOT NULL DEFAULT 'openai'"); err != nil {
			return err
		}
	}
	return nil
}

// migrateConversationRecordsTimestampToMillis 将 conversation_records 表中的秒级时间戳转换为毫秒级
// 通过判断时间戳数值大小来区分：秒级时间戳约10位，毫秒级约13位
func migrateConversationRecordsTimestampToMillis(db *sql.DB) error {
	// 检查 timestamp 列的数据范围
	// 如果最大值小于 1e12（约 2001-09-09，秒级），说明是秒级时间戳
	var maxTimestamp sql.NullInt64
	var maxCreatedAt sql.NullInt64

	err := db.QueryRow(`SELECT MAX(timestamp), MAX(created_at) FROM conversation_records`).Scan(&maxTimestamp, &maxCreatedAt)
	if err != nil {
		return err
	}

	// 如果没有数据，直接返回
	if !maxTimestamp.Valid && !maxCreatedAt.Valid {
		return nil
	}

	// 判断是否为秒级时间戳（小于 1e12 认为是秒级）
	isSeconds := false
	if maxTimestamp.Valid && maxTimestamp.Int64 > 0 && maxTimestamp.Int64 < 1e12 {
		isSeconds = true
	}
	if maxCreatedAt.Valid && maxCreatedAt.Int64 > 0 && maxCreatedAt.Int64 < 1e12 {
		isSeconds = true
	}

	if !isSeconds {
		// 已经是毫秒级或没有需要转换的数据
		return nil
	}

	// 执行转换：秒级 * 1000 = 毫秒级
	_, err = db.Exec(`UPDATE conversation_records SET timestamp = timestamp * 1000, created_at = created_at * 1000 WHERE timestamp < 1000000000000`)
	return err
}

func tableHasColumn(db *sql.DB, tableName, columnName string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			name      string
			columnTyp string
			notNull   int
			defaultV  sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &columnTyp, &notNull, &defaultV, &pk); err != nil {
			return false, err
		}
		if name == columnName {
			return true, nil
		}
	}
	return false, rows.Err()
}
