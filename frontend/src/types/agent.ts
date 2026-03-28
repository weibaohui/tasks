/**
 * Agent 相关类型定义
 */

export interface Agent {
  id: string;
  agent_code: string;
  user_code: string;
  name: string;
  description: string;
  identity_content: string;
  soul_content: string;
  agents_content: string;
  user_content: string;
  tools_content: string;
  model: string;
  max_tokens: number;
  temperature: number;
  max_iterations: number;
  history_messages: number;
  skills_list: string[];
  tools_list: string[];
  is_active: boolean;
  is_default: boolean;
  enable_thinking_process: boolean;
  created_at: number;
  updated_at: number;
}

export interface CreateAgentRequest {
  user_code: string;
  name: string;
  description: string;
  identity_content: string;
  soul_content: string;
  agents_content: string;
  user_content: string;
  tools_content: string;
  model: string;
  max_tokens: number;
  temperature: number;
  max_iterations: number;
  history_messages: number;
  skills_list: string[];
  tools_list: string[];
  is_default: boolean;
  enable_thinking_process: boolean;
}

export interface UpdateAgentRequest {
  name: string;
  description: string;
  identity_content: string;
  soul_content: string;
  agents_content: string;
  user_content: string;
  tools_content: string;
  model: string;
  max_tokens: number;
  temperature: number;
  max_iterations: number;
  history_messages: number;
  skills_list: string[];
  tools_list: string[];
  is_active?: boolean;
  is_default?: boolean;
  enable_thinking_process?: boolean;
}

export interface PatchAgentRequest {
  name?: string;
  description?: string;
  identity_content?: string;
  soul_content?: string;
  agents_content?: string;
  user_content?: string;
  tools_content?: string;
  model?: string;
  max_tokens?: number;
  temperature?: number;
  max_iterations?: number;
  history_messages?: number;
  skills_list?: string[];
  tools_list?: string[];
  is_active?: boolean;
  is_default?: boolean;
  enable_thinking_process?: boolean;
}
