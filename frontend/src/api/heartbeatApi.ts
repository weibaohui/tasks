import apiClient from './client';
import type { Heartbeat, CreateHeartbeatRequest, UpdateHeartbeatRequest, HeartbeatRunRecord } from '../types/heartbeat';

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

export async function listProjectHeartbeatRuns(projectId: string, limit: number = 50): Promise<HeartbeatRunRecord[]> {
  const res = await apiClient.get(`/projects/${projectId}/heartbeat-runs`, { params: { limit } });
  return res.data;
}
