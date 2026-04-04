/**
 * MCP API 调用模块
 */
import apiClient from './client';
import type { AxiosError } from 'axios';
import type {
  MCPServer,
  CreateMCPServerRequest,
  UpdateMCPServerRequest,
  AgentMCPBinding,
  CreateBindingRequest,
  UpdateBindingRequest,
} from '../types/mcp';
import type { MCPTool } from '../types/mcp';

/**
 * 提取后端错误信息（用于前端 toast 展示）
 */
export function getMCPErrorMessage(err: unknown): string {
  const e = err as AxiosError<any>;
  const msg = e?.response?.data?.message;
  if (typeof msg === 'string' && msg.trim()) return msg.trim();
  return '请求失败';
}

/**
 * 列出 MCP 服务器
 */
export async function listMCPServers(): Promise<MCPServer[]> {
  const resp = await apiClient.get<MCPServer[]>('/mcp/servers');
  return resp.data;
}

/**
 * 获取单个 MCP 服务器
 */
export async function getMCPServer(id: string): Promise<MCPServer> {
  const resp = await apiClient.get<MCPServer>('/mcp/servers', { params: { id } });
  return resp.data;
}

/**
 * 创建 MCP 服务器
 */
export async function createMCPServer(req: CreateMCPServerRequest): Promise<MCPServer> {
  const resp = await apiClient.post<MCPServer>('/mcp/servers', req);
  return resp.data;
}

/**
 * 更新 MCP 服务器
 */
export async function updateMCPServer(id: string, req: UpdateMCPServerRequest): Promise<MCPServer> {
  const resp = await apiClient.put<MCPServer>('/mcp/servers', req, { params: { id } });
  return resp.data;
}

/**
 * 删除 MCP 服务器
 */
export async function deleteMCPServer(id: string): Promise<void> {
  await apiClient.delete('/mcp/servers', { params: { id } });
}

/**
 * 测试 MCP 服务器连接
 */
export async function testMCPServer(id: string): Promise<{ message: string }> {
  const resp = await apiClient.post<{ message: string }>('/mcp/servers/test', undefined, { params: { id } });
  return resp.data;
}

/**
 * 刷新 MCP 工具能力
 */
export async function refreshMCPServer(id: string): Promise<{ message: string }> {
  const resp = await apiClient.post<{ message: string }>('/mcp/servers/refresh', undefined, { params: { id } });
  return resp.data;
}

/**
 * 列出服务器工具
 */
export async function listMCPTools(id: string): Promise<MCPTool[]> {
  const resp = await apiClient.get<MCPTool[]>('/mcp/servers/tools', { params: { id } });
  return resp.data;
}

/**
 * 列出 Agent 的 MCP 绑定
 */
export async function listBindings(agentId: string): Promise<AgentMCPBinding[]> {
  const resp = await apiClient.get<AgentMCPBinding[]>('/mcp/bindings', { params: { agent_id: agentId } });
  return resp.data;
}

/**
 * 创建绑定
 */
export async function createBinding(req: CreateBindingRequest): Promise<AgentMCPBinding> {
  const resp = await apiClient.post<AgentMCPBinding>('/mcp/bindings', req);
  return resp.data;
}

/**
 * 更新绑定
 */
export async function updateBinding(id: string, req: UpdateBindingRequest): Promise<AgentMCPBinding> {
  const resp = await apiClient.put<AgentMCPBinding>('/mcp/bindings', req, { params: { id } });
  return resp.data;
}

/**
 * 删除绑定
 */
export async function deleteBinding(id: string): Promise<void> {
  await apiClient.delete('/mcp/bindings', { params: { id } });
}
