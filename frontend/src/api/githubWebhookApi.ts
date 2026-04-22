import apiClient from './client';

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

export interface TriggeredHeartbeat {
  id: string;
  heartbeat_id: string;
  requirement_id: string;
  triggered_at: number;
}

export interface WebhookEventLog {
  id: string;
  project_id: string;
  event_type: string;
  method: string;
  headers: string;
  payload: string;
  status: string;
  trigger_heartbeat_id: string;
  requirement_id: string;
  error_message: string;
  received_at: number;
  triggered_heartbeats?: TriggeredHeartbeat[];
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

// 列出所有 webhook 配置
export async function listWebhookConfigs(): Promise<GitHubWebhookConfig[]> {
  const res = await apiClient.get('/github-webhooks/configs');
  return res.data;
}

// 创建 webhook 配置
export async function createWebhookConfig(projectId: string, repo: string): Promise<GitHubWebhookConfig> {
  const res = await apiClient.post('/github-webhooks/configs', { project_id: projectId, repo });
  return res.data;
}

// 更新 webhook 配置
export async function updateWebhookConfig(id: string, repo: string): Promise<void> {
  await apiClient.put(`/github-webhooks/configs/${id}`, { repo });
}

// 删除 webhook 配置
export async function deleteWebhookConfig(id: string): Promise<void> {
  await apiClient.delete(`/github-webhooks/configs/${id}`);
}

// 启用 webhook
export async function enableWebhook(id: string): Promise<GitHubWebhookConfig> {
  const res = await apiClient.post(`/github-webhooks/configs/${id}/enable`);
  return res.data;
}

// 停用 webhook
export async function disableWebhook(id: string): Promise<GitHubWebhookConfig> {
  const res = await apiClient.post(`/github-webhooks/configs/${id}/disable`);
  return res.data;
}

// 获取 forwarder 状态
export async function getForwarderStatus(id: string): Promise<{ running: boolean; webhook_url: string }> {
  const res = await apiClient.get(`/github-webhooks/configs/${id}/status`);
  return res.data;
}

// 检查 webhook URL 是否需要更新
export async function checkWebhookURL(id: string): Promise<{ needs_update: boolean; current_url: string; expected_url: string }> {
  const res = await apiClient.get(`/github-webhooks/configs/${id}/check-url`);
  return res.data;
}

// 更新 webhook URL
export async function updateWebhookURL(id: string): Promise<{ message: string; old_url?: string; webhook_url: string }> {
  const res = await apiClient.post(`/github-webhooks/configs/${id}/update-url`);
  return res.data;
}

// 列出事件日志（分页）
export interface EventLogsResponse {
  data: WebhookEventLog[];
  total: number;
  limit: number;
  offset: number;
}

export async function listEventLogs(configId: string, limit = 20, offset = 0): Promise<EventLogsResponse> {
  const res = await apiClient.get(`/github-webhooks/configs/${configId}/event-logs`, {
    params: { limit, offset },
  });
  return res.data;
}

// 清空事件日志
export async function clearEventLogs(configId: string): Promise<{ message: string }> {
  const res = await apiClient.delete(`/github-webhooks/configs/${configId}/event-logs`);
  return res.data;
}

// 列出心跳绑定
export async function listBindings(configId: string): Promise<WebhookHeartbeatBinding[]> {
  const res = await apiClient.get(`/github-webhooks/configs/${configId}/bindings`);
  return res.data;
}

// 创建心跳绑定
export async function createBinding(
  projectId: string,
  configId: string,
  eventType: string,
  heartbeatId: string
): Promise<WebhookHeartbeatBinding> {
  const res = await apiClient.post('/github-webhooks/bindings', {
    project_id: projectId,
    config_id: configId,
    event_type: eventType,
    heartbeat_id: heartbeatId,
  });
  return res.data;
}

// 删除心跳绑定
export async function deleteBinding(id: string): Promise<void> {
  await apiClient.delete(`/github-webhooks/bindings/${id}`);
}

// 列出项目的所有心跳（用于选择绑定）
export async function listHeartbeatsForBinding(projectId: string): Promise<HeartbeatOption[]> {
  const res = await apiClient.get('/github-webhooks/heartbeats', { params: { project_id: projectId } });
  return res.data;
}

// 重新触发事件日志关联的心跳
export async function retriggerHeartbeat(heartbeatId: string): Promise<void> {
  await apiClient.post(`/heartbeats/${heartbeatId}/trigger`);
}
