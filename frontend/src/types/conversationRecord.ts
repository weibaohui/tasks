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
  start_time?: string;
  end_time?: string;
  limit?: number;
  offset?: number;
}

/**
 * 对话统计数据
 */
export interface TokenStats {
  total_prompt_tokens: number;
  total_completion_tokens: number;
  total_tokens: number;
  daily_trends: {
    date: string;
    prompt_tokens: number;
    complete_tokens: number;
    total_tokens: number;
  }[];
}

export interface AgentDistribution {
  code: string;
  name: string;
  count: number;
  tokens: number;
}

export interface ChannelDistribution {
  type: string;
  count: number;
}

export interface RoleDistribution {
  role: string;
  count: number;
}

export interface ProjectDistribution {
  project_id: string;
  name: string;
  tokens: number;
}

export interface AgentTypeDistribution {
  agent_type: string;
  tokens: number;
}

export interface SessionStats {
  total_sessions: number;
  avg_messages_per_session: number;
  avg_response_time_ms: number;
}

export interface ConversationStats {
  token_stats: TokenStats;
  agent_distribution: AgentDistribution[];
  channel_distribution: ChannelDistribution[];
  role_distribution: RoleDistribution[];
  project_distribution: ProjectDistribution[];
  agent_type_distribution: AgentTypeDistribution[];
  session_stats: SessionStats;
}
