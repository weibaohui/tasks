import apiClient from './client';
import type { HeartbeatTemplate, CreateHeartbeatTemplateRequest } from '../types/heartbeat_template';

export async function listHeartbeatTemplates(): Promise<HeartbeatTemplate[]> {
  const res = await apiClient.get('/heartbeat-templates');
  return res.data;
}

export async function createHeartbeatTemplate(data: CreateHeartbeatTemplateRequest): Promise<HeartbeatTemplate> {
  const res = await apiClient.post('/heartbeat-templates', data);
  return res.data;
}

export async function deleteHeartbeatTemplate(id: string): Promise<void> {
  await apiClient.delete(`/heartbeat-templates/${id}`);
}
