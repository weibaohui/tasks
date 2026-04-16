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

export async function listHeartbeatScenarios(): Promise<HeartbeatScenario[]> {
  const response = await apiClient.get<HeartbeatScenario[]>('/heartbeat-scenarios');
  return response.data;
}

export async function applyHeartbeatScenario(projectId: string, scenarioCode: string): Promise<void> {
  await apiClient.post(`/projects/${projectId}/apply-scenario`, { scenario_code: scenarioCode });
}
