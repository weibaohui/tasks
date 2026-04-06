/**
 * Requirement Type API
 */
import apiClient from './client';

export interface RequirementType {
  id: string;
  project_id: string;
  code: string;
  name: string;
  description?: string;
  icon?: string;
  color?: string;
  sort_order: number;
  state_machine_id?: string;
  created_at: number;
  updated_at: number;
}

export interface CreateRequirementTypeRequest {
  project_id: string;
  code: string;
  name: string;
  description?: string;
  icon?: string;
  color?: string;
}

export const requirementTypeApi = {
  list: async (projectId: string): Promise<RequirementType[]> => {
    const response = await apiClient.get<RequirementType[]>(
      `/requirement-types?project_id=${projectId}`
    );
    return response.data;
  },

  create: async (data: CreateRequirementTypeRequest): Promise<RequirementType> => {
    const response = await apiClient.post<RequirementType>(
      '/requirement-types',
      data
    );
    return response.data;
  },

  delete: async (id: string): Promise<void> => {
    await apiClient.delete(`/requirement-types?id=${id}`);
  },
};