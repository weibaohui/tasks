export interface Project {
  id: string;
  name: string;
  git_repo_url: string;
  default_branch: string;
  init_steps: string[];
  heartbeat_enabled: boolean;
  heartbeat_interval_minutes: number;
  heartbeat_md_content: string;
  agent_code: string;
  dispatch_channel_code: string;
  dispatch_session_key: string;
  created_at: number;
  updated_at: number;
}

export interface RequirementAgentInfo {
  id?: string;
  agent_code?: string;
  name?: string;
  shadow_from?: string;
}

export interface TodoItem {
  content: string;
  status: string;
  priority?: string;
}

export interface ProgressData {
  items: TodoItem[];
  percent: number;
  updated_at: number;
}

export interface Requirement {
  id: string;
  project_id: string;
  title: string;
  description: string;
  acceptance_criteria: string;
  temp_workspace_root: string;
  status: string;
  assignee_agent_code: string;
  replica_agent_code: string;
  assignee_agent?: RequirementAgentInfo | null;
  replica_agent?: RequirementAgentInfo | null;
  workspace_path: string;
  last_error: string;
  dispatch_session_key: string;
  trace_id?: string;
  prompt_tokens?: number;
  completion_tokens?: number;
  total_tokens?: number;
  progress_data?: ProgressData | string | null;
  agent_runtime?: {
    status?: string;
    last_error?: string;
    started_at?: number | null;
    ended_at?: number | null;
    prompt?: string;
    result?: string;
    agent_type?: string;
  } | null;
  started_at: number | null;
  completed_at: number | null;
  created_at: number;
  updated_at: number;
  requirement_type?: string;
}

export interface CreateProjectRequest {
  name: string;
  git_repo_url: string;
  default_branch: string;
  init_steps: string[];
}

export interface UpdateProjectRequest {
  id: string;
  name?: string;
  git_repo_url?: string;
  default_branch?: string;
  init_steps?: string[];
  heartbeat_enabled?: boolean;
  heartbeat_interval_minutes?: number;
  heartbeat_md_content?: string;
  agent_code?: string;
  dispatch_channel_code?: string;
  dispatch_session_key?: string;
}

export interface CreateRequirementRequest {
  project_id: string;
  title: string;
  description: string;
  acceptance_criteria: string;
  temp_workspace_root: string;
  requirement_type?: string;
}

export interface UpdateRequirementRequest {
  id: string;
  title?: string;
  description?: string;
  acceptance_criteria?: string;
  temp_workspace_root?: string;
  requirement_type?: string;
}
