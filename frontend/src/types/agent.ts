/**
 * Agent 相关类型定义
 */

// PermissionMode 定义 Claude Code 权限处理模式
export type PermissionMode = 'default' | 'acceptEdits' | 'plan' | 'bypassPermissions';

// SandboxNetworkConfig 网络配置
export interface SandboxNetworkConfig {
  allow_unix_sockets?: boolean;
  allow_all_unix_sockets?: boolean;
  allow_local_binding?: boolean;
  http_proxy_port?: number;
  socks_proxy_port?: number;
}

// SandboxIgnoreViolations 忽略的沙箱违规
export interface SandboxIgnoreViolations {
  file?: string[];
  network?: string[];
}

// McpServerConfig MCP 服务器配置
export interface McpServerConfig {
  command?: string;
  args?: string[];
  env?: Record<string, string>;
}

// ClaudeCodeConfig Claude Code 配置
export interface ClaudeCodeConfig {
  // === Tab 1: 基本设置 ===
  model?: string;
  provider_key?: string;
  system_prompt?: string;
  max_thinking_tokens?: number;
  permission_mode?: PermissionMode;
  allowed_tools?: string[];
  disallowed_tools?: string[];
  max_turns?: number;
  cwd?: string;
  resume?: boolean;

  // === Tab 2: 高级设置 ===
  fallback_model?: string;
  append_system_prompt?: string;
  file_checkpointing?: boolean;
  continue_conversation?: boolean;
  fork_session?: boolean;

  // 沙箱安全
  sandbox_enabled?: boolean;
  auto_allow_bash_if_sandboxed?: boolean;
  excluded_commands?: string[];
  sandbox_network?: SandboxNetworkConfig;
  ignore_violations?: SandboxIgnoreViolations;

  // MCP & 插件
  mcp_servers?: Record<string, McpServerConfig>;
  plugins?: string[];
  local_plugin?: string;

  // 输出 & 调试
  json_schema?: Record<string, object>;
  include_partial_messages?: boolean;
  max_budget_usd?: number;
  debug_writer?: string;
  stderr_callback?: string;

  // 其他
  betas?: string[];
  cli_path?: string;
  env?: Record<string, string>;
  extra_args?: Record<string, string>;
  settings?: string;
  setting_sources?: string[];
}

export interface Agent {
  id: string;
  agent_code: string;
  agent_type: string;
  user_code: string;
  name: string;
  description: string;
  identity_content: string;
  soul_content: string;
  agents_content: string;
  user_content: string;
  tools_content: string;
  model: string;
  provider_key: string;
  max_tokens: number;
  temperature: number;
  max_iterations: number;
  history_messages: number;
  skills_list: string[];
  tools_list: string[];
  is_active: boolean;
  is_default: boolean;
  enable_thinking_process: boolean;
  shadow_from?: string; // 分身来源：如果不为空，则表示是某 Agent 的分身
  claude_code_config?: ClaudeCodeConfig;
  created_at: number;
  updated_at: number;
}

export interface CreateAgentRequest {
  user_code: string;
  name: string;
  agent_type: string;
  description: string;
  identity_content: string;
  soul_content: string;
  agents_content: string;
  user_content: string;
  tools_content: string;
  model: string;
  provider_key: string;
  max_tokens: number;
  temperature: number;
  max_iterations: number;
  history_messages: number;
  skills_list: string[];
  tools_list: string[];
  is_default: boolean;
  enable_thinking_process: boolean;
  claude_code_config?: ClaudeCodeConfig;
}

export interface UpdateAgentRequest {
  name: string;
  agent_type: string;
  description: string;
  identity_content: string;
  soul_content: string;
  agents_content: string;
  user_content: string;
  tools_content: string;
  model: string;
  provider_key: string;
  max_tokens: number;
  temperature: number;
  max_iterations: number;
  history_messages: number;
  skills_list: string[];
  tools_list: string[];
  is_active?: boolean;
  is_default?: boolean;
  enable_thinking_process?: boolean;
  claude_code_config?: ClaudeCodeConfig;
}

export interface PatchAgentRequest {
  name?: string;
  agent_type?: string;
  description?: string;
  identity_content?: string;
  soul_content?: string;
  agents_content?: string;
  user_content?: string;
  tools_content?: string;
  model?: string;
  provider_key?: string;
  max_tokens?: number;
  temperature?: number;
  max_iterations?: number;
  history_messages?: number;
  skills_list?: string[];
  tools_list?: string[];
  is_active?: boolean;
  is_default?: boolean;
  enable_thinking_process?: boolean;
  claude_code_config?: ClaudeCodeConfig;
}
