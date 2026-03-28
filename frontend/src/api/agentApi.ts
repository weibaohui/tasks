/**
 * Agent API 调用模块
 */
import apiClient from './taskApi';
import type { Agent, CreateAgentRequest, PatchAgentRequest, UpdateAgentRequest } from '../types/agent';

/**
 * 获取 Agent 列表
 */
export async function listAgents(userCode: string): Promise<Agent[]> {
  const response = await apiClient.get<Agent[]>('/agents', { params: { user_code: userCode } });
  return response.data;
}

/**
 * 创建 Agent
 */
export async function createAgent(request: CreateAgentRequest): Promise<Agent> {
  const response = await apiClient.post<Agent>('/agents', request);
  return response.data;
}

/**
 * 更新 Agent（全量）
 */
export async function updateAgent(id: string, request: UpdateAgentRequest): Promise<Agent> {
  const response = await apiClient.put<Agent>('/agents', request, { params: { id } });
  return response.data;
}

/**
 * 局部更新 Agent（只更新提供的字段）
 */
export async function patchAgent(id: string, request: PatchAgentRequest): Promise<Agent> {
  const response = await apiClient.patch<Agent>('/agents', request, { params: { id } });
  return response.data;
}

/**
 * 删除 Agent
 */
export async function deleteAgent(id: string): Promise<void> {
  await apiClient.delete('/agents', { params: { id } });
}
