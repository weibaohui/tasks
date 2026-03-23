/**
 * 对话记录（Conversation Record）API 调用模块
 */
import apiClient from './taskApi';
import type { ConversationRecord, ListConversationRecordsQuery } from '../types/conversationRecord';

/**
 * 获取对话记录列表
 */
export async function listConversationRecords(query: ListConversationRecordsQuery): Promise<ConversationRecord[]> {
  const response = await apiClient.get<ConversationRecord[]>('/conversation-records', { params: query });
  return response.data;
}

/**
 * 获取单条对话记录
 */
export async function getConversationRecord(id: string): Promise<ConversationRecord> {
  const response = await apiClient.get<ConversationRecord>('/conversation-records', { params: { id } });
  return response.data;
}
