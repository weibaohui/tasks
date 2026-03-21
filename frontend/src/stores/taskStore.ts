/**
 * 任务状态管理模块
 * 使用 Zustand 管理全局任务状态
 */
import { create } from 'zustand';
import type { Task } from '../types/task';
import * as taskApi from '../api/taskApi';

interface TaskState {
  tasks: Task[];
  currentTask: Task | null;
  loading: boolean;
  error: string | null;
  fetchTasks: (traceId?: string) => Promise<void>;
  fetchTask: (taskId: string) => Promise<void>;
  updateTaskInList: (task: Task) => void;
  addTask: (task: Task) => void;
  clearError: () => void;
}

export const useTaskStore = create<TaskState>((set, get) => ({
  tasks: [],
  currentTask: null,
  loading: false,
  error: null,

  /**
   * 获取任务列表（无 traceId 时获取全部）
   */
  fetchTasks: async (traceId?: string) => {
    set({ loading: true, error: null });
    try {
      const response = traceId
        ? await taskApi.listTasksByTrace(traceId)
        : await taskApi.getAllTasks();
      set({ tasks: response.tasks, loading: false });
    } catch (error) {
      set({ error: '获取任务列表失败', loading: false });
    }
  },

  /**
   * 获取单个任务详情
   */
  fetchTask: async (taskId: string) => {
    set({ loading: true, error: null });
    try {
      const task = await taskApi.getTask(taskId);
      set({ currentTask: task, loading: false });
    } catch (error) {
      set({ error: '获取任务详情失败', loading: false });
    }
  },

  /**
   * 更新列表中的任务
   */
  updateTaskInList: (task: Task) => {
    const tasks = get().tasks.map((t) => (t.id === task.id ? task : t));
    set({ tasks, currentTask: task });
  },

  /**
   * 添加新任务到列表头部
   */
  addTask: (task: Task) => {
    set({ tasks: [task, ...get().tasks] });
  },

  /**
   * 清除错误信息
   */
  clearError: () => set({ error: null }),
}));
