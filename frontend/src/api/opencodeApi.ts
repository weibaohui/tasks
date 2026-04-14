/**
 * OpenCode API 调用模块
 */
import apiClient from './client';

/**
 * 获取 OpenCode 可用模型列表
 */
export async function listOpenCodeModels(): Promise<string[]> {
  const response = await apiClient.get<{ models: string[] }>('/opencode/models');
  return response.data.models;
}
