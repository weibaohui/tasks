/**
 * 任务状态枚举
 */
export type TaskStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

/**
 * 任务类型枚举
 */
export type TaskType = 'agent';

/**
 * 任务接口
 */
export interface Task {
  id: string;
  trace_id: string;
  span_id: string;
  parent_id?: string;
  name: string;
  description: string;
  type: TaskType;
  status: TaskStatus;
  progress: Progress;
  result?: Result;
  error?: string;
  metadata: Record<string, unknown>;
  timeout: number;
  max_retries: number;
  priority: number;
  created_at: number;
  started_at?: number;
  finished_at?: number;
}

/**
 * 进度接口
 */
export interface Progress {
  total: number;
  current: number;
  percentage: number;
  stage: string;
  detail: string;
  updated_at: number;
}

/**
 * 结果接口
 */
export interface Result {
  data: unknown;
  message: string;
  task_conclusion?: string;
}

/**
 * 创建任务请求
 */
export interface CreateTaskRequest {
  name: string;
  description: string;
  type: TaskType;
  timeout: number;
  max_retries: number;
  priority: number;
  metadata: Record<string, unknown>;
  parent_id?: string;
  trace_id?: string;
  // 渠道信息，用于 LLM 查找
  channel_code?: string;
  user_code?: string;
}

/**
 * WebSocket 消息类型
 */
export interface WSMessage {
  type: string;
  trace_id: string;
  data: unknown;
  timestamp: number;
}

/**
 * 创建任务响应
 */
export interface CreateTaskResponse {
  id: string;
  trace_id: string;
  span_id: string;
  status: TaskStatus;
}

/**
 * 任务列表响应
 */
export interface TaskListResponse {
  tasks: Task[];
  total: number;
}

/**
 * 任务树节点
 */
export interface TaskTreeNode {
  task: Task;
  children: TaskTreeNode[];
}

/**
 * Todo 列表项状态
 */
export type TodoStatus = 'distributed' | 'running' | 'completed' | 'failed' | 'cancelled';

/**
 * Todo 列表项
 */
export interface TodoItem {
  sub_task_id: string;
  sub_task_type: TaskType;
  goal: string;
  status: TodoStatus;
  progress: number;
  span_id: string;
  created_at: number;
  completed_at?: number;
}

/**
 * Todo 列表
 */
export interface TodoList {
  task_id: string;
  items: TodoItem[];
  created_at: number;
  updated_at: number;
}

/**
 * 任务执行摘要
 */
export interface ExecutionSummary {
  task_id: string;
  span_id: string;
  goal: string;
  result: string;
  stage: string;
  completed_at: number;
  status: string;
}
