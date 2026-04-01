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

export interface Requirement {
  id: string;
  project_id: string;
  title: string;
  description: string;
  acceptance_criteria: string;
  temp_workspace_root: string;
  status: 'todo' | 'preparing' | 'coding' | 'pr_opened' | 'failed' | 'completed' | 'done';
  assignee_agent_code: string;
  replica_agent_code: string;
  workspace_path: string;
  last_error: string;
  dispatch_session_key: string;
  claude_runtime?: {
    status?: string;
    is_running?: boolean;
    last_error?: string;
    started_at?: number | null;
    ended_at?: number | null;
    updated_at?: number | null;
    prompt?: string;
    result?: string;
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

export interface UpdateProjectRequest extends CreateProjectRequest {
  id: string;
  heartbeat_enabled: boolean;
  heartbeat_interval_minutes: number;
  heartbeat_md_content: string;
  agent_code: string;
  dispatch_channel_code: string;
  dispatch_session_key: string;
}

export interface CreateRequirementRequest {
  project_id: string;
  title: string;
  description: string;
  acceptance_criteria: string;
  temp_workspace_root: string;
}

export interface UpdateRequirementRequest {
  id: string;
  title: string;
  description: string;
  acceptance_criteria: string;
  temp_workspace_root: string;
}
