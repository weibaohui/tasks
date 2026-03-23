/**
 * 会话（Session）相关类型定义
 */

export interface Session {
  id: string;
  user_code: string;
  agent_code: string;
  channel_code: string;
  session_key: string;
  external_id: string;
  last_active_at: number | null;
  metadata: Record<string, unknown>;
  created_at: number;
  updated_at: number;
}

export interface CreateSessionRequest {
  user_code: string;
  channel_code: string;
  agent_code: string;
  session_key: string;
  external_id: string;
  metadata: Record<string, unknown>;
}
