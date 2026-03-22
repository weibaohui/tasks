/**
 * 任务 API 调用模块
 * 封装与后端 HTTP API 的交互
 */
import axios from 'axios';
import type {
  Task,
  TaskListResponse,
  CreateTaskRequest,
  CreateTaskResponse,
  TaskTreeNode,
} from '../types/task';

const BASE_URL = '/api/v1';

/**
 * 创建 axios 实例，配置基础参数
 */
const apiClient = axios.create({
  baseURL: BASE_URL,
  timeout: 10000,
});

/**
 * 创建任务
 * POST /api/v1/tasks
 */
export async function createTask(request: CreateTaskRequest): Promise<CreateTaskResponse> {
  const response = await apiClient.post<CreateTaskResponse>('/tasks', request);
  return response.data;
}

/**
 * 获取单个任务
 * GET /api/v1/tasks?id=xxx
 */
export async function getTask(taskId: string): Promise<Task> {
  const response = await apiClient.get<Task>('/tasks', {
    params: { id: taskId },
  });
  return response.data;
}

/**
 * 获取任务列表（全部）
 * GET /api/v1/tasks/all
 */
export async function getAllTasks(): Promise<TaskListResponse> {
  const response = await apiClient.get<TaskListResponse>('/tasks/all');
  return response.data;
}

/**
 * 获取任务列表（按 trace_id）
 * GET /api/v1/tasks/trace/{trace_id}
 */
export async function listTasksByTrace(traceId: string): Promise<TaskListResponse> {
  const response = await apiClient.get<TaskListResponse>(`/tasks/trace/${traceId}`);
  return response.data;
}

/**
 * 获取任务树
 * GET /api/v1/traces/{trace_id}/tree
 */
export async function getTaskTree(traceId: string): Promise<TaskTreeNode[]> {
  const response = await apiClient.get<TaskTreeNode[]>(`/traces/${traceId}/tree`);
  return response.data;
}

/**
 * 取消任务
 * POST /api/v1/tasks/{id}/cancel
 */
export async function cancelTask(taskId: string): Promise<void> {
  await apiClient.post(`/tasks/${taskId}/cancel`);
}

/**
 * 启动任务
 * POST /api/v1/tasks/{id}/start
 */
export async function startTask(taskId: string): Promise<void> {
  await apiClient.post(`/tasks/${taskId}/start`);
}

export async function clearAllTasks(): Promise<{ message: string; deleted: number }> {
  const response = await apiClient.post<{ message: string; deleted: number }>('/tasks/clear');
  return response.data;
}

export default apiClient;
