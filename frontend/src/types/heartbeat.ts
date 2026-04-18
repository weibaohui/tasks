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
  agent_code?: string;
  requirement_type: string;
}

export interface UpdateHeartbeatRequest {
  name: string;
  interval_minutes: number;
  md_content: string;
  agent_code?: string;
  requirement_type: string;
  enabled: boolean;
}

export interface HeartbeatRunRecord {
  requirement_id: string;
  heartbeat_id: string;
  heartbeat_name: string;
  project_id: string;
  trigger_source: string;
  status: string;
  title: string;
  last_error: string;
  error_category: string;
  created_at: number;
}

export interface HeartbeatRunPage {
  data: HeartbeatRunRecord[];
  total: number;
  limit: number;
  offset: number;
}
