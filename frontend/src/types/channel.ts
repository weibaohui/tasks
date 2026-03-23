/**
 * 渠道（Channel）相关类型定义
 */

export interface Channel {
  id: string;
  channel_code: string;
  user_code: string;
  agent_code: string;
  name: string;
  type: string;
  is_active: boolean;
  allow_from: string[];
  config: Record<string, unknown>;
  created_at: number;
  updated_at: number;
}

export interface CreateChannelRequest {
  user_code: string;
  name: string;
  type: string;
  config: Record<string, unknown>;
  allow_from: string[];
  agent_code: string;
}

export interface UpdateChannelRequest {
  name?: string;
  config?: Record<string, unknown>;
  allow_from?: string[];
  is_active?: boolean;
  agent_code?: string;
}
