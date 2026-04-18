import apiClient from './client';

export interface SystemLogLine {
  line: number;
  text: string;
}

export interface SystemLogTailResponse {
  path: string;
  keyword: string;
  total: number;
  truncated: boolean;
  lines: SystemLogLine[];
}

export interface SystemLogConfigResponse {
  path: string;
}

export interface SystemLogStreamEvent {
  type: 'snapshot' | 'append' | 'reset' | 'error';
  path?: string;
  keyword?: string;
  total?: number;
  truncated?: boolean;
  lines?: SystemLogLine[];
  message?: string;
}

/**
 * 获取日志配置。
 */
export async function getSystemLogConfig(): Promise<SystemLogConfigResponse> {
  const resp = await apiClient.get<SystemLogConfigResponse>('/system/logs/config');
  return resp.data;
}

/**
 * 拉取日志尾部数据。
 */
export async function getSystemLogTail(params: { lines: number; keyword?: string }): Promise<SystemLogTailResponse> {
  const resp = await apiClient.get<SystemLogTailResponse>('/system/logs/tail', {
    params,
  });
  return resp.data;
}

/**
 * 清空日志文件内容（不删除文件）。
 */
export async function clearSystemLog(): Promise<void> {
  await apiClient.post('/system/logs/clear');
}

/**
 * 构建日志 SSE 流地址。
 */
export function buildSystemLogStreamURL(params: { lines: number; keyword?: string }): string {
  const query = new URLSearchParams();
  query.set('lines', String(params.lines));
  if (params.keyword && params.keyword.trim()) {
    query.set('keyword', params.keyword.trim());
  }

  const token = localStorage.getItem('auth_token');
  if (token) {
    query.set('token', token);
  }

  return `/api/v1/system/logs/stream?${query.toString()}`;
}
