/**
 * 对话记录（Conversation Record）API 调用模块
 */
import apiClient from './client';
import type { ConversationRecord, ListConversationRecordsQuery, ConversationStats } from '../types/conversationRecord';

export interface ListConversationRecordsResponse {
  items: ConversationRecord[];
  total: number;
}

/**
 * 获取对话记录列表
 */
export async function listConversationRecords(query: ListConversationRecordsQuery): Promise<ListConversationRecordsResponse> {
  const response = await apiClient.get<ListConversationRecordsResponse>('/conversation-records', { params: query });
  return response.data;
}

/**
 * 获取单条对话记录
 */
export async function getConversationRecord(id: string): Promise<ConversationRecord> {
  const response = await apiClient.get<ConversationRecord>('/conversation-records', { params: { id } });
  return response.data;
}

/**
 * 根据 Session Key 获取对话记录
 */
export async function getConversationRecordsBySession(sessionKey: string): Promise<ConversationRecord[]> {
  const response = await apiClient.get<ConversationRecord[]>(`/conversation-records/session/${sessionKey}`);
  return response.data;
}

/**
 * 根据 Trace ID 获取对话记录
 */
export async function getConversationRecordsByTrace(traceId: string): Promise<ConversationRecord[]> {
  const response = await apiClient.get<ConversationRecord[]>(`/conversation-records/trace/${traceId}`);
  return response.data;
}

/**
 * 获取对话统计数据
 */
export interface StatsParams {
  start_time?: string;
  end_time?: string;
  agent_codes?: string;
  channel_codes?: string;
  roles?: string;
}

export async function getConversationStats(params: StatsParams): Promise<ConversationStats> {
  const response = await apiClient.get<ConversationStats>('/conversation-records/stats', { params });
  return response.data;
}
