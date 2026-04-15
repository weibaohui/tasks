import apiClient from './client';
import type { Heartbeat, CreateHeartbeatRequest, UpdateHeartbeatRequest } from '../types/heartbeat';

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
