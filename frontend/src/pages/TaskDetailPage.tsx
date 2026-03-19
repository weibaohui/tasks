/**
 * 任务详情页面
 * 展示单个任务的完整信息
 */
import React, { useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Spin } from 'antd';
import { TaskDetail } from '../components/TaskDetail';
import { useTaskWebSocket } from '../hooks/useTaskWebSocket';
import { useTaskStore } from '../stores/taskStore';
import { useTaskOperations } from '../hooks/useTaskOperations';

export const TaskDetailPage: React.FC = () => {
  const { taskId } = useParams<{ taskId: string }>();
  const navigate = useNavigate();
  const { currentTask, loading, fetchTask } = useTaskStore();
  const { cancelTask } = useTaskOperations();

  useTaskWebSocket(currentTask?.trace_id || '');

  useEffect(() => {
    if (taskId) {
      fetchTask(taskId);
    }
  }, [taskId, fetchTask]);

  const handleCancel = async () => {
    if (!taskId) return;
    await cancelTask(taskId);
    fetchTask(taskId);
  };

  return (
    <div style={{ padding: 24 }}>
      <TaskDetail
        task={currentTask}
        loading={loading}
        onCancel={handleCancel}
        onBack={() => navigate(-1)}
      />
    </div>
  );
};
