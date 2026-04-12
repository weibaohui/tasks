/**
 * 数据库 Schema 定义
 * 包含所有表结构和索引
 */
package persistence

import (
	"database/sql"
)

// Schema 完整的数据库 Schema（表结构 + 索引）
const Schema = SchemaTables + SchemaIndexes

// SchemaTables SQL Schema 定义 - 表结构
const SchemaTables = `
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

CREATE TABLE IF NOT EXISTS user_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    token_hash TEXT NOT NULL,
    token_value TEXT,
    expires_at INTEGER,
    last_used_at INTEGER,
    is_active INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);

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

CREATE TABLE IF NOT EXISTS agent_mcp_bindings (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    mcp_server_id TEXT NOT NULL,
    enabled_tools TEXT,
    is_active INTEGER NOT NULL DEFAULT 1,
    auto_load INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL
);

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
`

// SchemaIndexes SQL Schema 定义 - 索引
const SchemaIndexes = `
-- tasks 表索引
CREATE INDEX IF NOT EXISTS idx_tasks_trace_id ON tasks(trace_id);
CREATE INDEX IF NOT EXISTS idx_tasks_parent_id ON tasks(parent_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_user_code ON tasks(user_code);
CREATE INDEX IF NOT EXISTS idx_tasks_agent_code ON tasks(agent_code);
CREATE INDEX IF NOT EXISTS idx_tasks_channel_code ON tasks(channel_code);
CREATE INDEX IF NOT EXISTS idx_tasks_session_key ON tasks(session_key);

-- users 表索引
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_user_code ON users(user_code);

-- user_tokens 表索引
CREATE INDEX IF NOT EXISTS idx_user_tokens_user_id ON user_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_user_tokens_token_hash ON user_tokens(token_hash);

-- agents 表索引
CREATE INDEX IF NOT EXISTS idx_agents_agent_code ON agents(agent_code);
CREATE INDEX IF NOT EXISTS idx_agents_user_code ON agents(user_code);

-- llm_providers 表索引
CREATE INDEX IF NOT EXISTS idx_llm_providers_user_code ON llm_providers(user_code);
CREATE INDEX IF NOT EXISTS idx_llm_providers_default ON llm_providers(is_default, is_active);

-- channels 表索引
CREATE INDEX IF NOT EXISTS idx_channels_channel_code ON channels(channel_code);
CREATE INDEX IF NOT EXISTS idx_channels_user_code ON channels(user_code);
CREATE INDEX IF NOT EXISTS idx_channels_agent_code ON channels(agent_code);

-- sessions 表索引
CREATE INDEX IF NOT EXISTS idx_sessions_session_key ON sessions(session_key);
CREATE INDEX IF NOT EXISTS idx_sessions_user_code ON sessions(user_code);
CREATE INDEX IF NOT EXISTS idx_sessions_channel_code ON sessions(channel_code);

-- projects 表索引
CREATE INDEX IF NOT EXISTS idx_projects_name ON projects(name);

-- requirements 表索引
CREATE INDEX IF NOT EXISTS idx_requirements_project_id ON requirements(project_id);
CREATE INDEX IF NOT EXISTS idx_requirements_status ON requirements(status);
CREATE INDEX IF NOT EXISTS idx_requirements_trace_id ON requirements(trace_id);

-- cron_jobs 表索引
CREATE INDEX IF NOT EXISTS idx_cron_jobs_user_code ON cron_jobs(user_code);
CREATE INDEX IF NOT EXISTS idx_cron_jobs_channel_code ON cron_jobs(channel_code);
CREATE INDEX IF NOT EXISTS idx_cron_jobs_next_run_at ON cron_jobs(next_run_at);
CREATE INDEX IF NOT EXISTS idx_cron_jobs_is_active ON cron_jobs(is_active);

-- conversation_records 表索引
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

-- mcp_servers 表索引
CREATE INDEX IF NOT EXISTS idx_mcp_servers_code ON mcp_servers(code);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_status ON mcp_servers(status);

-- agent_mcp_bindings 表索引
CREATE INDEX IF NOT EXISTS idx_agent_mcp_bindings_agent_id ON agent_mcp_bindings(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_mcp_bindings_mcp_server_id ON agent_mcp_bindings(mcp_server_id);

-- mcp_tools 表索引
CREATE INDEX IF NOT EXISTS idx_mcp_tools_mcp_server_id ON mcp_tools(mcp_server_id);
CREATE INDEX IF NOT EXISTS idx_mcp_tools_name ON mcp_tools(name);
CREATE INDEX IF NOT EXISTS idx_mcp_tools_deleted_at ON mcp_tools(deleted_at);

-- mcp_tool_logs 表索引
CREATE INDEX IF NOT EXISTS idx_mcp_tool_logs_session_key ON mcp_tool_logs(session_key);
CREATE INDEX IF NOT EXISTS idx_mcp_tool_logs_mcp_server_id ON mcp_tool_logs(mcp_server_id);
CREATE INDEX IF NOT EXISTS idx_mcp_tool_logs_tool_name ON mcp_tool_logs(tool_name);
CREATE INDEX IF NOT EXISTS idx_mcp_tool_logs_created_at ON mcp_tool_logs(created_at);

-- requirement_states 表索引
CREATE INDEX IF NOT EXISTS idx_req_states_requirement ON requirement_states(requirement_id);
CREATE INDEX IF NOT EXISTS idx_req_states_machine ON requirement_states(state_machine_id);

-- transition_logs 表索引
CREATE INDEX IF NOT EXISTS idx_transition_logs_requirement ON transition_logs(requirement_id);
CREATE INDEX IF NOT EXISTS idx_transition_logs_created ON transition_logs(created_at);

-- project_state_machines 表索引
CREATE INDEX IF NOT EXISTS idx_project_state_machines_project ON project_state_machines(project_id);
CREATE INDEX IF NOT EXISTS idx_project_state_machines_machine ON project_state_machines(state_machine_id);

-- requirement_types 表索引
CREATE INDEX IF NOT EXISTS idx_requirement_types_project ON requirement_types(project_id);
CREATE INDEX IF NOT EXISTS idx_requirement_types_code ON requirement_types(code);
`

// InitSchema 初始化数据库 Schema（表结构 + 索引）
func InitSchema(db *sql.DB) error {
	if _, err := db.Exec(Schema); err != nil {
		return err
	}
	return nil
}
