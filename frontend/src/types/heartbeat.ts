export interface Heartbeat {
  id: string;
  project_id: string;
  name: string;
  enabled: boolean;
  interval_minutes: number;
  md_content: string;
  agent_code: string;
  requirement_type: string;
  sort_order: number;
  created_at: number;
  updated_at: number;
}

export interface CreateHeartbeatRequest {
  project_id: string;
  name: string;
  interval_minutes: number;
  md_content: string;
  agent_code: string;
  requirement_type: string;
}

export interface UpdateHeartbeatRequest {
  name: string;
  interval_minutes: number;
  md_content: string;
  agent_code: string;
  requirement_type: string;
  enabled: boolean;
}
