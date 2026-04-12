package persistence
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
