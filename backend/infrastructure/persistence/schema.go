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

CREATE TABLE IF NOT EXISTS user_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    token_hash TEXT NOT NULL,
    expires_at INTEGER,
    last_used_at INTEGER,
    is_active INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_user_tokens_user_id ON user_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_user_tokens_token_hash ON user_tokens(token_hash);

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
    llm_provider_id TEXT REFERENCES llm_providers(id),
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
    shadow_from TEXT,
    claude_code_config TEXT,
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

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    git_repo_url TEXT NOT NULL,
    default_branch TEXT NOT NULL DEFAULT 'main',
    init_steps TEXT NOT NULL,
    heartbeat_enabled INTEGER NOT NULL DEFAULT 0,
    heartbeat_interval_minutes INTEGER NOT NULL DEFAULT 60,
    heartbeat_md_content TEXT NOT NULL DEFAULT '',
    agent_code TEXT NOT NULL DEFAULT '',
    dispatch_channel_code TEXT NOT NULL DEFAULT '',
    dispatch_session_key TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_projects_name ON projects(name);

CREATE TABLE IF NOT EXISTS requirements (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    acceptance_criteria TEXT,
    status TEXT NOT NULL DEFAULT 'todo',
    temp_workspace_root TEXT,
    assignee_agent_code TEXT,
    replica_agent_code TEXT,
    dispatch_session_key TEXT,
    workspace_path TEXT,
    last_error TEXT,
    started_at INTEGER,
    completed_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    claude_runtime_status TEXT,
    claude_runtime_started_at INTEGER,
    claude_runtime_ended_at INTEGER,
    claude_runtime_error TEXT,
    claude_runtime_result TEXT,
    claude_runtime_prompt TEXT,
    trace_id TEXT,
    prompt_tokens INTEGER DEFAULT 0,
    completion_tokens INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    requirement_type TEXT NOT NULL DEFAULT 'normal',
    previous_status TEXT,
    FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE INDEX IF NOT EXISTS idx_requirements_project_id ON requirements(project_id);
CREATE INDEX IF NOT EXISTS idx_requirements_status ON requirements(status);
CREATE INDEX IF NOT EXISTS idx_requirements_trace_id ON requirements(trace_id);

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

-- 状态机相关表
CREATE TABLE IF NOT EXISTS state_machines (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    config TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS requirement_states (
    id TEXT PRIMARY KEY,
    requirement_id TEXT NOT NULL UNIQUE,
    state_machine_id TEXT NOT NULL,
    current_state TEXT NOT NULL,
    current_state_name TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_req_states_requirement ON requirement_states(requirement_id);
CREATE INDEX IF NOT EXISTS idx_req_states_machine ON requirement_states(state_machine_id);

CREATE TABLE IF NOT EXISTS transition_logs (
    id TEXT PRIMARY KEY,
    requirement_id TEXT NOT NULL,
    from_state TEXT NOT NULL,
    to_state TEXT NOT NULL,
    trigger TEXT NOT NULL,
    triggered_by TEXT NOT NULL,
    remark TEXT,
    result TEXT NOT NULL,
    error_message TEXT,
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_transition_logs_requirement ON transition_logs(requirement_id);
CREATE INDEX IF NOT EXISTS idx_transition_logs_created ON transition_logs(created_at);

-- 项目状态机关联表
CREATE TABLE IF NOT EXISTS project_state_machines (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    requirement_type TEXT NOT NULL,
    state_machine_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (state_machine_id) REFERENCES state_machines(id) ON DELETE CASCADE,
    UNIQUE(project_id, requirement_type)
);

CREATE INDEX IF NOT EXISTS idx_project_state_machines_project ON project_state_machines(project_id);
CREATE INDEX IF NOT EXISTS idx_project_state_machines_machine ON project_state_machines(state_machine_id);

-- 需求类型表
CREATE TABLE IF NOT EXISTS requirement_types (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    icon TEXT,
    color TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    state_machine_id TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    UNIQUE(project_id, code)
);

CREATE INDEX IF NOT EXISTS idx_requirement_types_project ON requirement_types(project_id);
CREATE INDEX IF NOT EXISTS idx_requirement_types_code ON requirement_types(code);
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
	if err := migrateTasksTimeoutToSeconds(db); err != nil {
		return err
	}
	if err := migrateRequirementsNewColumns(db); err != nil {
		return err
	}
	if err := migrateAgentShadowFrom(db); err != nil {
		return err
	}
	if err := migrateProjectsNewColumns(db); err != nil {
		return err
	}
	if err := migrateRequirementsClaudeRuntime(db); err != nil {
		return err
	}
	if err := migrateRequirementType(db); err != nil {
		return err
	}
	if err := migrateConversationRecordsTimestampToMillis(db); err != nil {
		return err
	}
	if err := migrateAgentLLMProviderID(db); err != nil {
		return err
	}
	return migrateRequirementsTraceIDIndex(db)
}

// migrateStateMachineTables 迁移状态机相关表（预留，未来可通过 migrations 调用）
func migrateStateMachineTables(db *sql.DB) error {
	// 表已通过 CREATE TABLE IF NOT EXISTS 创建
	// 此处可添加未来需要的迁移逻辑
	return nil
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

func migrateRequirementsNewColumns(db *sql.DB) error {
	has, err := tableHasColumn(db, "requirements", "temp_workspace_root")
	if err != nil {
		return err
	}
	if !has {
		if _, err := db.Exec("ALTER TABLE requirements ADD COLUMN temp_workspace_root TEXT"); err != nil {
			return err
		}
	}
	hasDispatchSessionKey, err := tableHasColumn(db, "requirements", "dispatch_session_key")
	if err != nil {
		return err
	}
	if !hasDispatchSessionKey {
		if _, err := db.Exec("ALTER TABLE requirements ADD COLUMN dispatch_session_key TEXT"); err != nil {
			return err
		}
	}

	// 重命名 assignee_agent_id -> assignee_agent_code
	hasOldAssignee, err := tableHasColumn(db, "requirements", "assignee_agent_id")
	if err != nil {
		return err
	}
	hasNewAssignee, err := tableHasColumn(db, "requirements", "assignee_agent_code")
	if err != nil {
		return err
	}
	if hasOldAssignee && !hasNewAssignee {
		if _, err := db.Exec("ALTER TABLE requirements RENAME COLUMN assignee_agent_id TO assignee_agent_code"); err != nil {
			return err
		}
	}

	// 重命名 replica_agent_id -> replica_agent_code
	hasOldReplica, err := tableHasColumn(db, "requirements", "replica_agent_id")
	if err != nil {
		return err
	}
	hasNewReplica, err := tableHasColumn(db, "requirements", "replica_agent_code")
	if err != nil {
		return err
	}
	if hasOldReplica && !hasNewReplica {
		if _, err := db.Exec("ALTER TABLE requirements RENAME COLUMN replica_agent_id TO replica_agent_code"); err != nil {
			return err
		}
	}

	return nil
}

// migrateRequirementsClaudeRuntime 迁移 requirements 表新增 claude_runtime 相关字段
func migrateRequirementsClaudeRuntime(db *sql.DB) error {
	columns := []struct {
		name         string
		sqlType      string
		backfillZero bool
	}{
		{"claude_runtime_status", "TEXT", false},
		{"claude_runtime_started_at", "INTEGER", false},
		{"claude_runtime_ended_at", "INTEGER", false},
		{"claude_runtime_error", "TEXT", false},
		{"claude_runtime_result", "TEXT", false},
		{"claude_runtime_prompt", "TEXT", false},
		{"trace_id", "TEXT", false},
		{"prompt_tokens", "INTEGER NOT NULL DEFAULT 0", true},
		{"completion_tokens", "INTEGER NOT NULL DEFAULT 0", true},
		{"total_tokens", "INTEGER NOT NULL DEFAULT 0", true},
	}

	for _, col := range columns {
		has, err := tableHasColumn(db, "requirements", col.name)
		if err != nil {
			return err
		}
		if !has {
			if _, err := db.Exec(fmt.Sprintf("ALTER TABLE requirements ADD COLUMN %s %s", col.name, col.sqlType)); err != nil {
				return err
			}
		}
		// 对需要回填的列，将历史 NULL 值更新为 0
		if col.backfillZero {
			if _, err := db.Exec(fmt.Sprintf("UPDATE requirements SET %s = 0 WHERE %s IS NULL", col.name, col.name)); err != nil {
				return err
			}
		}
	}
	return nil
}

// migrateRequirementType 迁移 requirements 表新增 requirement_type 字段
func migrateRequirementType(db *sql.DB) error {
	has, err := tableHasColumn(db, "requirements", "requirement_type")
	if err != nil {
		return err
	}
	if !has {
		if _, err := db.Exec("ALTER TABLE requirements ADD COLUMN requirement_type TEXT NOT NULL DEFAULT 'normal'"); err != nil {
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

// migrateAgentShadowFrom 迁移 agents 表新增 shadow_from 字段
func migrateAgentShadowFrom(db *sql.DB) error {
	has, err := tableHasColumn(db, "agents", "shadow_from")
	if err != nil {
		return err
	}
	if !has {
		if _, err := db.Exec("ALTER TABLE agents ADD COLUMN shadow_from TEXT"); err != nil {
			return err
		}
	}
	return nil
}

// migrateProjectsNewColumns 迁移 projects 表新增字段
func migrateProjectsNewColumns(db *sql.DB) error {
	// 添加 dispatch_channel_code 列
	has, err := tableHasColumn(db, "projects", "dispatch_channel_code")
	if err != nil {
		return err
	}
	if !has {
		if _, err := db.Exec("ALTER TABLE projects ADD COLUMN dispatch_channel_code TEXT NOT NULL DEFAULT ''"); err != nil {
			return err
		}
	}

	// 添加 dispatch_session_key 列
	has, err = tableHasColumn(db, "projects", "dispatch_session_key")
	if err != nil {
		return err
	}
	if !has {
		if _, err := db.Exec("ALTER TABLE projects ADD COLUMN dispatch_session_key TEXT NOT NULL DEFAULT ''"); err != nil {
			return err
		}
	}

	// 将 heartbeat_agent_id 重命名为 agent_code
	hasOldColumn, err := tableHasColumn(db, "projects", "heartbeat_agent_id")
	if err != nil {
		return err
	}
	hasNewColumn, err := tableHasColumn(db, "projects", "agent_code")
	if err != nil {
		return err
	}
	if hasOldColumn && !hasNewColumn {
		if _, err := db.Exec("ALTER TABLE projects RENAME COLUMN heartbeat_agent_id TO agent_code"); err != nil {
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

// migrateAgentLLMProviderID 迁移 agents 表新增 llm_provider_id 字段
func migrateAgentLLMProviderID(db *sql.DB) error {
	has, err := tableHasColumn(db, "agents", "llm_provider_id")
	if err != nil {
		return err
	}
	if !has {
		if _, err := db.Exec("ALTER TABLE agents ADD COLUMN llm_provider_id TEXT REFERENCES llm_providers(id)"); err != nil {
			return err
		}
	}
	return nil
}

// migrateRequirementsTraceIDIndex 迁移 requirements 表新增 trace_id 索引
func migrateRequirementsTraceIDIndex(db *sql.DB) error {
	_, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_requirements_trace_id ON requirements(trace_id)")
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
