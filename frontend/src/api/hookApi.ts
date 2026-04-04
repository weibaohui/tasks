/**
 * Hook 配置 API 调用模块
 */
import apiClient from './client';
import type { HookConfig, CreateHookConfigRequest, UpdateHookConfigRequest, HookActionLog } from '../types/hook';

/**
 * 获取 Hook 配置列表
 */
export async function listHookConfigs(projectId?: string, triggerPoint?: string): Promise<HookConfig[]> {
  const params: Record<string, string> = {};
  if (projectId) {
    params.project_id = projectId;
  }
  if (triggerPoint) {
    params.trigger_point = triggerPoint;
  }
  const response = await apiClient.get<HookConfig[]>('/hook-configs', { params });
  return response.data;
}

/**
 * 获取单个 Hook 配置
 */
export async function getHookConfig(id: string): Promise<HookConfig> {
  const response = await apiClient.get<HookConfig>('/hook-configs', { params: { id } });
  return response.data;
}

/**
 * 创建 Hook 配置
 */
export async function createHookConfig(request: CreateHookConfigRequest): Promise<HookConfig> {
  const response = await apiClient.post<HookConfig>('/hook-configs', request);
  return response.data;
}

/**
 * 更新 Hook 配置
 */
export async function updateHookConfig(request: UpdateHookConfigRequest): Promise<HookConfig> {
  const response = await apiClient.put<HookConfig>('/hook-configs', request);
  return response.data;
}

/**
 * 删除 Hook 配置
 */
export async function deleteHookConfig(id: string): Promise<void> {
  await apiClient.delete('/hook-configs', { params: { id } });
}

/**
 * 启用 Hook 配置
 */
export async function enableHookConfig(id: string): Promise<HookConfig> {
  const response = await apiClient.patch<HookConfig>(`/hook-configs/${id}/enable`);
  return response.data;
}

/**
 * 禁用 Hook 配置
 */
export async function disableHookConfig(id: string): Promise<HookConfig> {
  const response = await apiClient.patch<HookConfig>(`/hook-configs/${id}/disable`);
  return response.data;
}

/**
 * 获取 Hook 执行日志
 */
export async function listHookLogs(requirementId?: string, hookConfigId?: string): Promise<HookActionLog[]> {
  const params: Record<string, string> = {};
  if (requirementId) {
    params.requirement_id = requirementId;
  }
  if (hookConfigId) {
    params.hook_config_id = hookConfigId;
  }
  const response = await apiClient.get<HookActionLog[]>('/hook-logs', { params });
  return response.data;
}