/**
 * Skill 相关类型定义
 */

export interface Skill {
  name: string;
  description: string;
  source: 'workspace' | 'builtin';
  path?: string;
  available?: boolean;
  requires?: string;
}

export interface SkillDetail {
  name: string;
  content: string;
  metadata: Record<string, string>;
  available: boolean;
  requires: string;
  source: 'workspace' | 'builtin';
}

export interface ListResponse<T> {
  items: T[];
  total: number;
}
