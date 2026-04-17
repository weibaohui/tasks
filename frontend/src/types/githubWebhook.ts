export interface GitHubWebhookConfig {
  id: string;
  project_id: string;
  repo: string;
  enabled: boolean;
  webhook_url: string;
  running: boolean;
  created_at: number;
  updated_at: number;
}

export interface WebhookEventLog {
  id: string;
  project_id: string;
  event_type: string;
  status: 'received' | 'processed' | 'failed';
  trigger_heartbeat_id: string;
  error_message: string;
  received_at: number;
}

export interface WebhookHeartbeatBinding {
  id: string;
  project_id: string;
  github_webhook_config_id: string;
  github_event_type: string;
  heartbeat_id: string;
  enabled: boolean;
  created_at: number;
}

export interface HeartbeatOption {
  id: string;
  project_id: string;
  name: string;
  enabled: boolean;
  interval_minutes: number;
  agent_code: string;
  requirement_type: string;
}

// 支持的 GitHub 事件类型
export const GITHUB_EVENT_TYPES = [
  { value: 'push', label: 'Push' },
  { value: 'pull_request', label: 'Pull Request' },
  { value: 'issues', label: 'Issues' },
  { value: 'issue_comment', label: 'Issue Comment' },
  { value: 'pull_request_review', label: 'Pull Request Review' },
  { value: 'pull_request_review_comment', label: 'PR Review Comment' },
  { value: 'release', label: 'Release' },
  { value: 'workflow_run', label: 'Workflow Run' },
  { value: 'create', label: 'Create (Branch/Tag)' },
  { value: 'delete', label: 'Delete (Branch/Tag)' },
  { value: 'fork', label: 'Fork' },
  { value: 'star', label: 'Star' },
  { value: 'watch', label: 'Watch' },
] as const;

export type GitHubEventType = typeof GITHUB_EVENT_TYPES[number]['value'];
