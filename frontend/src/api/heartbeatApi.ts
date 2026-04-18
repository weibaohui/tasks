import apiClient from './client';
import type { Heartbeat, CreateHeartbeatRequest, UpdateHeartbeatRequest, HeartbeatRunPage, HeartbeatRunRecord } from '../types/heartbeat';

export async function listHeartbeats(projectId: string): Promise<Heartbeat[]> {
  const res = await apiClient.get('/heartbeats', { params: { project_id: projectId } });
  return res.data;
}

export async function createHeartbeat(data: CreateHeartbeatRequest): Promise<Heartbeat> {
  const res = await apiClient.post('/heartbeats', data);
  return res.data;
}

export async function getHeartbeat(id: string): Promise<Heartbeat> {
  const res = await apiClient.get(`/heartbeats/${id}`);
  return res.data;
}

export async function updateHeartbeat(id: string, data: UpdateHeartbeatRequest): Promise<Heartbeat> {
  const res = await apiClient.put(`/heartbeats/${id}`, data);
  return res.data;
}

export async function deleteHeartbeat(id: string): Promise<void> {
  await apiClient.delete(`/heartbeats/${id}`);
}

export async function listHeartbeatRuns(id: string, limit: number = 20): Promise<HeartbeatRunRecord[]> {
  const res = await apiClient.get(`/heartbeats/${id}/runs`, { params: { limit } });
  return res.data;
}

export async function listProjectHeartbeatRuns(
  projectId: string,
  params?: { limit?: number; offset?: number; statuses?: string[] }
): Promise<HeartbeatRunPage> {
  const res = await apiClient.get(`/projects/${projectId}/heartbeat-runs`, {
    params: {
      limit: params?.limit ?? 20,
      offset: params?.offset ?? 0,
      statuses: params?.statuses && params.statuses.length > 0 ? params.statuses.join(',') : undefined,
    },
  });
  return res.data;
}

export async function triggerHeartbeat(id: string): Promise<void> {
  await apiClient.post(`/heartbeats/${id}/trigger`);
}
