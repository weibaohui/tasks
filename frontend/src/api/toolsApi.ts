/**
 * Tools API - for built-in tools listing
 */
import apiClient from './client';
import type { BuiltInTool } from '../types/task';

export async function listBuiltInTools(): Promise<BuiltInTool[]> {
  const response = await apiClient.get<BuiltInTool[]>('/tools/builtin');
  return response.data;
}
