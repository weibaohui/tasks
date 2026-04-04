/**
 * Task related types - for backward compatibility
 * Note: Task management module has been removed, these types are kept for components that still reference them
 */

export interface BuiltInTool {
  name: string;
  description: string;
}

export interface Task {
  id: string;
  name: string;
  status: string;
  trace_id?: string;
  parent_id?: string;
  created_at: number;
  updated_at: number;
}

export interface TaskTreeNode extends Task {
  children?: TaskTreeNode[];
}
