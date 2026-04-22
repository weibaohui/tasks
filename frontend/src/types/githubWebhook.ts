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
  method: string;
  headers: string;
  payload: string;
  status: 'received' | 'processed' | 'failed';
  trigger_heartbeat_id: string;
  requirement_id: string;
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

// 支持的 ATG (AtomGit) 事件类型
export const ATG_EVENT_TYPES = [
  { value: 'push_events', label: '推送事件' },
  { value: 'tag_push_events', label: 'Tag 推送事件' },
  { value: 'issues_events', label: 'Issue 事件' },
  { value: 'note_events', label: '评论事件' },
  { value: 'merge_requests_events', label: 'Pull Request 事件' },
] as const;

export type ATGEventType = typeof ATG_EVENT_TYPES[number]['value'];

// 事件类型到心跳 requirement_type 的映射
export const EVENT_TO_REQUIREMENT_TYPE: Record<string, { github: string; atg: string }> = {
  // issues 相关
  'issues': { github: 'github_issue', atg: 'atg_issue' },
  'issues_events': { github: 'github_issue', atg: 'atg_issue' },
  'issue_comment': { github: 'github_issue', atg: 'atg_issue' },
  'note_events': { github: 'github_issue', atg: 'atg_issue' }, // 评论映射到 issue 类型

  // push 相关
  'push': { github: 'github_coding', atg: 'atg_coding' },
  'push_events': { github: 'github_coding', atg: 'atg_coding' },
  'tag_push_events': { github: 'github_coding', atg: 'atg_coding' },
  'create': { github: 'github_coding', atg: 'atg_coding' },
  'delete': { github: 'github_coding', atg: 'atg_coding' },

  // PR/merge 相关
  'pull_request': { github: 'github_pr_review', atg: 'atg_pr_review' },
  'merge_requests_events': { github: 'github_pr_review', atg: 'atg_pr_review' },
  'pull_request_review': { github: 'github_pr_review', atg: 'atg_pr_review' },
  'pull_request_review_comment': { github: 'github_pr_review', atg: 'atg_pr_review' },

  // 其他
  'release': { github: 'github_doc', atg: 'atg_doc' },
  'workflow_run': { github: 'github_coding', atg: 'atg_coding' },
  'fork': { github: 'github_coding', atg: 'atg_coding' },
  'star': { github: 'github_issue', atg: 'atg_issue' },
  'watch': { github: 'github_issue', atg: 'atg_issue' },
};
