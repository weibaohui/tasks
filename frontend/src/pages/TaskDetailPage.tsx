/**
 * 任务详情页面
 * 展示单个任务的完整信息
 */
import React, { useEffect, useRef } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import { Row, Col } from 'antd';
import { TaskDetail } from '../components/TaskDetail';
import { TodoList } from '../components/TodoList';
import { useTaskWebSocket } from '../hooks/useTaskWebSocket';
import { useTaskStore } from '../stores/taskStore';
import { useTaskOperations } from '../hooks/useTaskOperations';
import type { TodoList as TodoListType } from '../types/task';

export const TaskDetailPage: React.FC = () => {
  const { taskId } = useParams<{ taskId: string }>();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { currentTask, loading, fetchTask } = useTaskStore();
  const { cancelTask, startTask } = useTaskOperations();
  const autoStartedRef = useRef(false);

  useTaskWebSocket(currentTask?.trace_id || '');

  useEffect(() => {
    if (taskId) {
      fetchTask(taskId);
    }
  }, [taskId, fetchTask]);

  useEffect(() => {
    const action = searchParams.get('action');
    if (action === 'start' && currentTask?.status === 'pending' && taskId && !autoStartedRef.current) {
      autoStartedRef.current = true;
      startTask(taskId).then(() => {
        fetchTask(taskId);
        navigate(`/tasks/${taskId}`, { replace: true });
      });
    }
  }, [currentTask, taskId, searchParams, startTask, fetchTask, navigate]);

  const handleCancel = async () => {
    if (!taskId) return;
    await cancelTask(taskId);
    fetchTask(taskId);
  };

  const todoList: TodoListType | null = (() => {
    if (!currentTask?.metadata?.todo_list) return null;
    try {
      return JSON.parse(currentTask.metadata.todo_list as string);
    } catch {
      return null;
    }
  })();

  return (
    <div style={{ padding: 24 }}>
      <Row gutter={16}>
        <Col span={16}>
          <TaskDetail
            task={currentTask}
            loading={loading}
            onCancel={handleCancel}
            onBack={() => navigate(-1)}
            onViewTree={(traceId) => navigate(`/tasks/trace/${traceId}/tree`)}
          />
        </Col>
        <Col span={8}>
          <TodoList todoList={todoList} loading={loading} />
        </Col>
      </Row>
    </div>
  );
};
