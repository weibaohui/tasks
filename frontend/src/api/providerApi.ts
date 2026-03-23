/**
 * LLM Provider API 调用模块
 */
import apiClient from './taskApi';
import type { CreateProviderRequest, LLMProvider, TestProviderResult, UpdateProviderRequest } from '../types/provider';

/**
 * 获取 LLM Provider 列表
 */
export async function listProviders(userCode: string): Promise<LLMProvider[]> {
  const response = await apiClient.get<LLMProvider[]>('/providers', { params: { user_code: userCode } });
  return response.data;
}

/**
 * 创建 LLM Provider
 */
export async function createProvider(request: CreateProviderRequest): Promise<LLMProvider> {
  const response = await apiClient.post<LLMProvider>('/providers', request);
  return response.data;
}

/**
 * 更新 LLM Provider
 */
export async function updateProvider(id: string, request: UpdateProviderRequest): Promise<LLMProvider> {
  const response = await apiClient.put<LLMProvider>('/providers', request, { params: { id } });
  return response.data;
}

/**
 * 删除 LLM Provider
 */
export async function deleteProvider(id: string): Promise<void> {
  await apiClient.delete('/providers', { params: { id } });
}

/**
 * 测试 Provider 连接
 */
export async function testProviderConnection(id: string): Promise<TestProviderResult> {
  const response = await apiClient.post<TestProviderResult>('/providers/test', undefined, { params: { id } });
  return response.data;
}
