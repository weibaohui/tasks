package persistence
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
    opencode_config TEXT,
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
    max_concurrent_agents INTEGER NOT NULL DEFAULT 2,
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
    agent_runtime_status TEXT,
    agent_runtime_started_at INTEGER,
    agent_runtime_ended_at INTEGER,
    agent_runtime_error TEXT,
    agent_runtime_result TEXT,
    agent_runtime_prompt TEXT,
    agent_runtime_agent_type TEXT,
    trace_id TEXT,
    prompt_tokens INTEGER DEFAULT 0,
    completion_tokens INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    requirement_type TEXT NOT NULL DEFAULT 'normal',
    previous_status TEXT,
    progress_data TEXT,
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
    is_system INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    UNIQUE(project_id, code)
);

CREATE TABLE IF NOT EXISTS heartbeats (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    name TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    interval_minutes INTEGER NOT NULL DEFAULT 60,
    md_content TEXT NOT NULL DEFAULT '',
    agent_code TEXT NOT NULL DEFAULT '',
    requirement_type TEXT NOT NULL DEFAULT 'heartbeat',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_heartbeats_project_id ON heartbeats(project_id);
CREATE INDEX IF NOT EXISTS idx_heartbeats_enabled ON heartbeats(enabled);
`
