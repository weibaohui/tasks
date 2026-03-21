/**
 * 任务详情抽屉组件
 * 展示任务详情和子任务列表
 */
import React, { useEffect, useState } from 'react';
import { Drawer, Descriptions, Tag, Button, Space, Spin, Row, Col, Divider } from 'antd';
import { TeamOutlined, ReloadOutlined } from '@ant-design/icons';
import { StatusBadge } from '../StatusBadge';
import { ProgressBar } from '../ProgressBar';
import { TodoList } from '../TodoList';
import { getAllTasks, getTask } from '../../api/taskApi';
import type { Task, TodoItem, TodoList as TodoListType, TaskStatus } from '../../types/task';

interface TaskDetailDrawerProps {
  taskId: string | null;
  open: boolean;
  onClose: () => void;
}

export const TaskDetailDrawer: React.FC<TaskDetailDrawerProps> = ({ taskId, open, onClose }) => {
  const [task, setTask] = useState<Task | null>(null);
  const [loading, setLoading] = useState(false);
  const [todoList, setTodoList] = useState<TodoListType | null>(null);

  useEffect(() => {
    if (!taskId || !open) return;
    loadTask(false);
    const timer = setInterval(() => {
      loadTask(true);
    }, 2000);
    return () => clearInterval(timer);
  }, [taskId, open]);

  const normalizeStatus = (status: Task['status']): TodoItem['status'] => {
    if (status === 'pending') return 'distributed';
    if (status === 'running') return 'running';
    if (status === 'completed') return 'completed';
    if (status === 'failed') return 'failed';
    return 'cancelled';
  };

  const parseMetadataTodo = (metadata: Record<string, unknown> | undefined): TodoListType | null => {
    if (!metadata?.todo_list) return null;
    try {
      return JSON.parse(metadata.todo_list as string) as TodoListType;
    } catch {
      return null;
    }
  };

  const buildTodoList = (currentTask: Task, allTasks: Task[]): TodoListType | null => {
    const childTasks = allTasks.filter((t) => t.parent_id === currentTask.id);
    const metadataTodo = parseMetadataTodo(currentTask.metadata);
    const baseMap = new Map<string, TodoItem>();

    if (metadataTodo?.items) {
      metadataTodo.items.forEach((item) => {
        baseMap.set(item.sub_task_id, item);
      });
    }

    const mergedItems: TodoItem[] = childTasks.map((child) => {
      const base = baseMap.get(child.id);
      return {
        sub_task_id: child.id,
        sub_task_type: child.type,
        goal: base?.goal || child.name,
        status: normalizeStatus(child.status),
        progress: Math.round(child.progress?.percentage || 0),
        span_id: child.span_id,
        created_at: child.created_at,
        completed_at: child.finished_at,
      };
    });

    if (mergedItems.length === 0 && metadataTodo) {
      return metadataTodo;
    }

    return {
      task_id: currentTask.id,
      items: mergedItems,
      created_at: metadataTodo?.created_at || Date.now(),
      updated_at: Date.now(),
    };
  };

  const loadTask = async (silent: boolean) => {
    if (!taskId) return;
    if (!silent) setLoading(true);
    try {
      const [taskResponse, tasksResponse] = await Promise.all([getTask(taskId), getAllTasks()]);
      setTask(taskResponse);
      setTodoList(buildTodoList(taskResponse, tasksResponse.tasks));
    } catch (error) {
      console.error('Failed to load task:', error);
    } finally {
      if (!silent) setLoading(false);
    }
  };

  const handleRefresh = () => {
    loadTask(false);
  };

  if (!open) return null;

  return (
    <Drawer
      title={
        <Space>
          <TeamOutlined />
          <span>任务详情</span>
          {task && <Tag>{task.status}</Tag>}
        </Space>
      }
      placement="right"
      width={720}
      open={open}
      onClose={onClose}
      extra={
        <Button icon={<ReloadOutlined />} onClick={handleRefresh} loading={loading}>
          刷新
        </Button>
      }
    >
      {loading ? (
        <div style={{ textAlign: 'center', padding: 50 }}>
          <Spin size="large" />
        </div>
      ) : task ? (
        <Row gutter={16}>
          <Col span={24}>
            <Descriptions column={2} bordered size="small" title="基本信息">
              <Descriptions.Item label="任务ID">
                <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{task.id}</span>
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <StatusBadge status={task.status as TaskStatus} />
              </Descriptions.Item>
              <Descriptions.Item label="任务名称">{task.name}</Descriptions.Item>
              <Descriptions.Item label="类型">
                <Tag color="blue">{task.type}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="TraceID">
                <span style={{ fontFamily: 'monospace', fontSize: 11 }}>{task.trace_id}</span>
              </Descriptions.Item>
              <Descriptions.Item label="SpanID">
                <span style={{ fontFamily: 'monospace', fontSize: 11 }}>{task.span_id}</span>
              </Descriptions.Item>
              <Descriptions.Item label="优先级">{task.priority}</Descriptions.Item>
              <Descriptions.Item label="超时">{task.timeout}ms</Descriptions.Item>
              <Descriptions.Item label="创建时间">
                {new Date(task.created_at).toLocaleString()}
              </Descriptions.Item>
              <Descriptions.Item label="开始时间">
                {task.started_at ? new Date(task.started_at).toLocaleString() : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="完成时间">
                {task.finished_at ? new Date(task.finished_at).toLocaleString() : '-'}
              </Descriptions.Item>
            </Descriptions>
          </Col>

          {task.description && (
            <Col span={24} style={{ marginTop: 16 }}>
              <Descriptions column={1} bordered size="small" title="描述">
                <Descriptions.Item>{task.description}</Descriptions.Item>
              </Descriptions>
            </Col>
          )}

          <Col span={24} style={{ marginTop: 16 }}>
            <Descriptions column={1} bordered size="small" title="执行进度">
              <Descriptions.Item>
                <ProgressBar progress={task.progress} />
                <div style={{ marginTop: 8 }}>
                  <Tag color={task.progress.stage ? 'blue' : 'default'}>
                    {task.progress.stage || '无'}
                  </Tag>
                  <span style={{ marginLeft: 8, color: '#666' }}>
                    {task.progress.detail || '-'}
                  </span>
                </div>
              </Descriptions.Item>
            </Descriptions>
          </Col>

          {task.error && (
            <Col span={24} style={{ marginTop: 16 }}>
              <Descriptions column={1} bordered size="small" title="错误信息">
                <Descriptions.Item>
                  <pre style={{ color: 'red', margin: 0 }}>{task.error}</pre>
                </Descriptions.Item>
              </Descriptions>
            </Col>
          )}

          {task.result && (
            <Col span={24} style={{ marginTop: 16 }}>
              <Descriptions column={1} bordered size="small" title="执行结果">
                <Descriptions.Item>
                  <pre style={{ margin: 0 }}>{JSON.stringify(task.result, null, 2)}</pre>
                </Descriptions.Item>
              </Descriptions>
            </Col>
          )}
        </Row>
      ) : (
        <div style={{ textAlign: 'center', padding: 50, color: '#999' }}>
          未找到任务
        </div>
      )}

      <Divider />

      <TodoList todoList={todoList} loading={loading} />
    </Drawer>
  );
};

export default TaskDetailDrawer;
