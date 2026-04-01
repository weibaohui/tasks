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

export async function dispatchRequirement(requirementId: string, agentCode: string, channelCode: string, sessionKey: string): Promise<{
  requirement_id: string;
  status: string;
  workspace_path: string;
  replica_agent_code: string;
  task_id: string;
}> {
  const response = await apiClient.post('/requirements/dispatch', {
    requirement_id: requirementId,
    agent_code: agentCode,
    channel_code: channelCode,
    session_key: sessionKey,
  });
  return response.data;
}

export async function reportRequirementPROpened(requirementId: string): Promise<Requirement> {
  const response = await apiClient.post<Requirement>('/requirements/pr', {
    requirement_id: requirementId,
  });
  return response.data;
}

export async function redispatchRequirement(requirementId: string): Promise<Requirement> {
  const response = await apiClient.post<Requirement>('/requirements/redispatch', {
    requirement_id: requirementId,
  });
  return response.data;
}

// copyAndDispatchRequirement 复制需求并派发新副本
// 创建一个新需求（复制原需求内容，标题增加"[重新派发]"标记），然后派发
export async function copyAndDispatchRequirement(requirementId: string): Promise<Requirement> {
  const response = await apiClient.post<Requirement>('/requirements/copy-and-dispatch', {
    requirement_id: requirementId,
  });
  return response.data;
}
