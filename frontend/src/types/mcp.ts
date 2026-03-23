/**
 * MCP 类型定义
 */
export type MCPTransportType = 'stdio' | 'http' | 'sse';

/**
 * MCP 工具
 */
export interface MCPTool {
  name: string;
  description?: string;
  input_schema?: Record<string, any>;
}

/**
 * MCP 服务器
 */
export interface MCPServer {
  id: string;
  code: string;
  name: string;
  description?: string;
  transport_type: MCPTransportType;
  command?: string;
  args?: string[];
  url?: string;
  env_vars?: Record<string, string>;
  status: string;
  capabilities?: MCPTool[];
  last_connected?: number | null;
  error_message?: string;
  created_at: number;
  updated_at: number;
}

/**
 * 创建/更新服务器请求
 */
export interface CreateMCPServerRequest {
  code: string;
  name: string;
  description?: string;
  transport_type: MCPTransportType;
  command?: string;
  args?: string[];
  url?: string;
  env_vars?: Record<string, string>;
}

export interface UpdateMCPServerRequest {
  name?: string;
  description?: string;
  transport_type?: MCPTransportType;
  command?: string;
  args?: string[];
  url?: string;
  env_vars?: Record<string, string>;
}

/**
 * Agent-MCP 绑定
 */
export interface AgentMCPBinding {
  id: string;
  agent_id: string;
  mcp_server_id: string;
  enabled_tools: string[] | null;
  is_active: boolean;
  auto_load: boolean;
  created_at: number;
  updated_at: number;
}

/**
 * 创建/更新 绑定请求
 */
export interface CreateBindingRequest {
  agent_id: string;
  mcp_server_id: string;
  enabled_tools?: string[];
  is_active?: boolean;
  auto_load?: boolean;
}

export interface UpdateBindingRequest {
  enabled_tools?: string[];
  is_active?: boolean;
  auto_load?: boolean;
}

