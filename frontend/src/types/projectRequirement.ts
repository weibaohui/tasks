export interface Project {
  id: string;
  name: string;
  git_repo_url: string;
  default_branch: string;
  init_steps: string[];
  dispatch_channel_code: string;
  dispatch_session_key: string;
  default_agent_code: string;
  max_concurrent_agents: number;
  heartbeat_scenario_code?: string;
  created_at: number;
  updated_at: number;
}

/**
 * 根据 git_repo_url 检测平台类型
 */
export function detectPlatformType(gitRepoUrl: string): 'github' | 'atom_git' {
  const url = gitRepoUrl.toLowerCase();
  if (url.includes('gitcode.com')) {
    return 'atom_git';
  }
  if (url.includes('github.com')) {
    return 'github';
  }
  // 默认返回 github
  return 'github';
}

/**
 * 平台类型显示名称
 */
export function getPlatformDisplayName(platformType: 'github' | 'atom_git'): string {
  switch (platformType) {
    case 'github':
      return 'GitHub';
    case 'atom_git':
      return 'AtomGit';
    default:
      return platformType;
  }
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
  dispatch_channel_code?: string;
  dispatch_session_key?: string;
  default_agent_code?: string;
  max_concurrent_agents?: number;
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
