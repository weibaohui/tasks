import apiClient from './client';

export interface HeartbeatScenarioItem {
  name: string;
  interval_minutes: number;
  md_content: string;
  agent_code: string;
  requirement_type: string;
  sort_order: number;
}

export interface HeartbeatScenario {
  id: string;
  code: string;
  name: string;
  description: string;
  items: HeartbeatScenarioItem[];
  enabled: boolean;
  is_built_in: boolean;
  created_at: number;
  updated_at: number;
}

export interface CreateScenarioRequest {
  code: string;
  name: string;
  description: string;
  items: HeartbeatScenarioItem[];
  enabled: boolean;
}

export interface UpdateScenarioRequest {
  name: string;
  description: string;
  items: HeartbeatScenarioItem[];
  enabled: boolean;
}

export async function listHeartbeatScenarios(): Promise<HeartbeatScenario[]> {
  const response = await apiClient.get<HeartbeatScenario[]>('/heartbeat-scenarios');
  return response.data;
}

export async function getHeartbeatScenario(code: string): Promise<HeartbeatScenario> {
  const response = await apiClient.get<HeartbeatScenario>(`/heartbeat-scenarios/${code}`);
  return response.data;
}

export async function createHeartbeatScenario(request: CreateScenarioRequest): Promise<HeartbeatScenario> {
  const response = await apiClient.post<HeartbeatScenario>('/heartbeat-scenarios', request);
  return response.data;
}

export async function updateHeartbeatScenario(code: string, request: UpdateScenarioRequest): Promise<HeartbeatScenario> {
  const response = await apiClient.put<HeartbeatScenario>(`/heartbeat-scenarios/${code}`, request);
  return response.data;
}

export async function deleteHeartbeatScenario(id: string): Promise<void> {
  await apiClient.delete(`/heartbeat-scenarios/${id}`);
}

export async function applyHeartbeatScenario(projectId: string, scenarioCode: string): Promise<void> {
  await apiClient.post(`/projects/${projectId}/apply-scenario`, { scenario_code: scenarioCode });
}
