/**
 * 任务操作 Hook
 * 提供创建任务、取消任务等操作
 */
import { useState } from 'react';
import { message } from 'antd';
import * as taskApi from '../api/taskApi';
import type { CreateTaskRequest, CreateTaskResponse } from '../types/task';

export function useTaskOperations() {
  const [creating, setCreating] = useState(false);

  const createTask = async (
    request: CreateTaskRequest
  ): Promise<CreateTaskResponse | null> => {
    setCreating(true);
    try {
      const response = await taskApi.createTask(request);
      message.success('任务创建成功');
      return response;
    } catch (error) {
      message.error('创建任务失败');
      return null;
    } finally {
      setCreating(false);
    }
  };

  const cancelTask = async (taskId: string): Promise<boolean> => {
    try {
      await taskApi.cancelTask(taskId);
      message.success('任务已取消');
      return true;
    } catch (error) {
      message.error('取消任务失败');
      return false;
    }
  };

  return {
    creating,
    createTask,
    cancelTask,
  };
}
