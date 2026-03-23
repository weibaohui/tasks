/**
 * 会话（Session）API 调用模块
 */
import apiClient from './taskApi';
import type { CreateSessionRequest, Session } from '../types/session';

/**
 * 获取用户会话列表
 */
export async function listUserSessions(userCode: string): Promise<Session[]> {
  const response = await apiClient.get<Session[]>('/sessions', { params: { user_code: userCode } });
  return response.data;
}

/**
 * 创建会话
 */
export async function createSession(request: CreateSessionRequest): Promise<Session> {
  const response = await apiClient.post<Session>('/sessions', request);
  return response.data;
}

/**
 * 删除会话
 */
export async function deleteSession(sessionKey: string): Promise<void> {
  await apiClient.delete('/sessions', { params: { session_key: sessionKey } });
}

/**
 * 获取会话元数据
 */
export async function getSessionMetadata(sessionKey: string): Promise<Record<string, unknown>> {
  const response = await apiClient.get<Record<string, unknown>>(`/sessions/${encodeURIComponent(sessionKey)}/metadata`);
  return response.data;
}

/**
 * 更新会话元数据
 */
export async function updateSessionMetadata(sessionKey: string, metadata: Record<string, unknown>): Promise<void> {
  await apiClient.put(`/sessions/${encodeURIComponent(sessionKey)}/metadata`, metadata);
}
