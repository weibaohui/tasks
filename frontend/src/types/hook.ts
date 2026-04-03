/**
 * Hook 配置类型
 */
export interface HookConfig {
  id: string;
  project_id?: string;
  name: string;
  trigger_point: string;
  action_type: string;
  action_config: string;
  enabled: boolean;
  priority: number;
  created_at: number;
  updated_at: number;
}

export interface CreateHookConfigRequest {
  project_id?: string;
  name: string;
  trigger_point: string;
  action_type: string;
  action_config: string;
  enabled: boolean;
  priority: number;
}

export interface UpdateHookConfigRequest {
  id: string;
  project_id?: string;
  name?: string;
  trigger_point?: string;
  action_type?: string;
  action_config?: string;
  enabled?: boolean;
  priority?: number;
}

/**
 * Hook 执行日志
 */
export interface HookActionLog {
  id: string;
  hook_config_id: string;
  requirement_id: string;
  trigger_point: string;
  action_type: string;
  status: string;
  input_context: string;
  result: string;
  error: string;
  started_at: number;
  completed_at?: number;
}

/**
 * 支持的触发点
 */
export const TRIGGER_POINTS = [
  { value: 'start_dispatch', label: '开始派发 (start_dispatch)' },
  { value: 'mark_coding', label: '开始编码 (mark_coding)' },
  { value: 'claude_code_finished', label: 'Claude Code 结束 (claude_code_finished)' },
  { value: 'mark_failed', label: '标记失败 (mark_failed)' },
  { value: 'mark_pr_opened', label: 'PR 已打开 (mark_pr_opened)' },
] as const;

/**
 * 支持的动作类型
 */
export const ACTION_TYPES = [
  { value: 'coding_agent', label: 'Coding Agent (coding_agent)' },
  { value: 'notification', label: '通知 (notification)' },
  { value: 'webhook', label: 'Webhook (webhook)' },
] as const;

export type TriggerPoint = typeof TRIGGER_POINTS[number]['value'];
export type ActionType = typeof ACTION_TYPES[number]['value'];