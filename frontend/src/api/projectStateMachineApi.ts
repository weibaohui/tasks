import apiClient from './client';

// 项目状态机映射
type ProjectStateMachineMapping = {
  id: string;
  requirement_type: string;
  state_machine_id: string;
  state_machine_name?: string;
  created_at: number;
  updated_at: number;
};

// 设置项目状态机请求
type SetProjectStateMachineRequest = {
  requirement_type: string;
  state_machine_id: string;
};

// 获取项目的状态机映射列表
export async function listProjectStateMachines(projectId: string): Promise<ProjectStateMachineMapping[]> {
  const response = await apiClient.get<ProjectStateMachineMapping[]>(`/projects/${projectId}/state-machines`);
  return response.data;
}

// 设置项目状态机映射
export async function setProjectStateMachine(
  projectId: string,
  data: SetProjectStateMachineRequest
): Promise<ProjectStateMachineMapping> {
  const response = await apiClient.post<ProjectStateMachineMapping>(`/projects/${projectId}/state-machines`, data);
  return response.data;
}

// 获取指定类型的项目状态机映射
export async function getProjectStateMachineByType(
  projectId: string,
  requirementType: string
): Promise<ProjectStateMachineMapping> {
  const response = await apiClient.get<ProjectStateMachineMapping>(
    `/projects/${projectId}/state-machines/${requirementType}`
  );
  return response.data;
}

// 删除项目状态机映射
export async function deleteProjectStateMachine(id: string): Promise<void> {
  await apiClient.delete(`/project-state-machines/${id}`);
}

// 获取可用的需求类型列表
export async function getAvailableRequirementTypes(): Promise<{ types: string[] }> {
  const response = await apiClient.get<{ types: string[] }>('/project-state-machines/requirement-types');
  return response.data;
}

export type { ProjectStateMachineMapping, SetProjectStateMachineRequest };
