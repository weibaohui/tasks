import apiClient from './taskApi';
import type {
  CreateProjectRequest,
  CreateRequirementRequest,
  Project,
  Requirement,
  UpdateProjectRequest,
  UpdateRequirementRequest,
} from '../types/projectRequirement';

export async function listProjects(): Promise<Project[]> {
  const response = await apiClient.get<Project[]>('/projects');
  return response.data;
}

export async function createProject(payload: CreateProjectRequest): Promise<Project> {
  const response = await apiClient.post<Project>('/projects', payload);
  return response.data;
}

export async function updateProject(payload: UpdateProjectRequest): Promise<Project> {
  const response = await apiClient.put<Project>('/projects', payload);
  return response.data;
}

export async function deleteProject(id: string): Promise<void> {
  await apiClient.delete('/projects', { params: { id } });
}

export async function listRequirements(projectId?: string): Promise<Requirement[]> {
  const response = await apiClient.get<Requirement[]>('/requirements', {
    params: projectId ? { project_id: projectId } : undefined,
  });
  return response.data;
}

export async function createRequirement(payload: CreateRequirementRequest): Promise<Requirement> {
  const response = await apiClient.post<Requirement>('/requirements', payload);
  return response.data;
}

export async function updateRequirement(payload: UpdateRequirementRequest): Promise<Requirement> {
  const response = await apiClient.put<Requirement>('/requirements', payload);
  return response.data;
}

export async function dispatchRequirement(requirementId: string, agentId: string): Promise<{
  requirement_id: string;
  status: string;
  dev_state: string;
  workspace_path: string;
  replica_agent_id: string;
  task_id: string;
}> {
  const response = await apiClient.post('/requirements/dispatch', {
    requirement_id: requirementId,
    agent_id: agentId,
  });
  return response.data;
}

export async function reportRequirementPROpened(requirementId: string, prUrl: string, branchName: string): Promise<Requirement> {
  const response = await apiClient.post<Requirement>('/requirements/pr', {
    requirement_id: requirementId,
    pr_url: prUrl,
    branch_name: branchName,
  });
  return response.data;
}
