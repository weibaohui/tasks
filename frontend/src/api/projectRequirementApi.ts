import apiClient from './client';
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

export interface PaginatedRequirements {
  items: Requirement[];
  total: number;
}

export async function listRequirementsPaginated(params: {
  projectId?: string;
  statuses?: string[];
  limit?: number;
  offset?: number;
}): Promise<PaginatedRequirements> {
  const response = await apiClient.get<PaginatedRequirements>('/requirements', {
    params: {
      ...(params.projectId ? { project_id: params.projectId } : {}),
      ...(params.statuses && params.statuses.length > 0 ? { status: params.statuses.join(',') } : {}),
      limit: params.limit ?? 10,
      offset: params.offset ?? 0,
    },
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

// deleteRequirement 删除单个需求
export async function deleteRequirement(id: string): Promise<void> {
  await apiClient.delete('/requirements', { params: { id } });
}

// batchDeleteRequirements 批量删除需求
export async function batchDeleteRequirements(ids: string[]): Promise<void> {
  await apiClient.post('/requirements/batch-delete', { ids });
}

// updateRequirementStatus 更新需求状态（用于修复异常状态）
export async function updateRequirementStatus(id: string, newStatus: string): Promise<void> {
  await apiClient.put('/requirements/status', { id, new_status: newStatus });
}

// 状态转换历史记录
export interface TransitionLog {
  id: string;
  requirement_id: string;
  from_state: string;
  to_state: string;
  trigger: string;
  triggered_by: string;
  remark: string;
  result: string;
  error_message: string;
  created_at: number;
}

// 获取需求的状态转换历史
export async function getRequirementTransitionHistory(id: string): Promise<TransitionLog[]> {
  const response = await apiClient.get<TransitionLog[]>('/requirements/transition-history', {
    params: { id },
  });
  return response.data;
}

// 状态统计数据
export interface StatusStat {
  status: string;
  count: number;
}

// 获取状态统计数据（动态从数据库提取）
export async function getStatusStats(projectId?: string): Promise<StatusStat[]> {
  const response = await apiClient.get<StatusStat[]>('/requirements/status-stats', {
    params: projectId ? { project_id: projectId } : undefined,
  });
  return response.data;
}
