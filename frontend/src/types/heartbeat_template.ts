export interface HeartbeatTemplate {
  id: string;
  name: string;
  md_content: string;
  requirement_type: string;
  created_at: number;
  updated_at: number;
}

export interface CreateHeartbeatTemplateRequest {
  name: string;
  md_content: string;
  requirement_type: string;
}
