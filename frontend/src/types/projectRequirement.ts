export interface Project {
  id: string;
  name: string;
  git_repo_url: string;
  default_branch: string;
  init_steps: string[];
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
  status: 'todo' | 'in_progress' | 'done';
  dev_state: 'idle' | 'preparing' | 'coding' | 'pr_opened' | 'failed';
  assignee_agent_id: string;
  replica_agent_id: string;
  workspace_path: string;
  branch_name: string;
  pr_url: string;
  last_error: string;
  started_at: number | null;
  completed_at: number | null;
  created_at: number;
  updated_at: number;
}

export interface CreateProjectRequest {
  name: string;
  git_repo_url: string;
  default_branch: string;
  init_steps: string[];
}

export interface UpdateProjectRequest extends CreateProjectRequest {
  id: string;
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
