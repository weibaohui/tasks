/**
 * 对话记录（Conversation Record）相关类型定义
 */

export interface ConversationRecord {
  id: string;
  trace_id: string;
  span_id: string;
  parent_span_id: string;
  event_type: string;
  timestamp: number;
  session_key: string;
  role: string;
  content: string;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  reasoning_tokens: number;
  cached_tokens: number;
  user_code: string;
  agent_code: string;
  channel_code: string;
  channel_type: string;
  created_at: number;
}

export interface ListConversationRecordsQuery {
  trace_id?: string;
  session_key?: string;
  user_code?: string;
  agent_code?: string;
  channel_code?: string;
  event_type?: string;
  role?: string;
  limit?: number;
  offset?: number;
}
