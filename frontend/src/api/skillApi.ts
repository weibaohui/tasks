/**
 * Skill API 调用模块
 */
import apiClient from './taskApi';
import type { Skill, SkillDetail, ListResponse } from '../types/skill';

export type { Skill, SkillDetail };

/**
 * 获取所有技能列表
 */
export async function listSkills(): Promise<Skill[]> {
  const response = await apiClient.get<ListResponse<Skill>>('/skills');
  return response.data.items;
}

/**
 * 获取所有技能列表（简单版，用于下拉选择）
 */
export async function listSkillsSimple(): Promise<Skill[]> {
  const response = await apiClient.get<ListResponse<Skill>>('/skills/simple');
  return response.data.items;
}

/**
 * 获取技能详情
 */
export async function getSkill(name: string): Promise<SkillDetail> {
  const response = await apiClient.get<SkillDetail>(`/skills/detail?name=${encodeURIComponent(name)}`);
  return response.data;
}
