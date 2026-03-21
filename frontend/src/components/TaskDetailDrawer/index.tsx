/**
 * 任务详情抽屉组件
 * 展示任务详情和子任务列表
 */
import React, { useEffect, useState } from 'react';
import { Drawer, Descriptions, Tag, Button, Space, Spin, Row, Col, Divider, Tree } from 'antd';
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
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [traceTasks, setTraceTasks] = useState<Task[]>([]);
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
      setSelectedTaskId((prev) => prev || taskResponse.id);

      const sameTraceTasks = tasksResponse.tasks.filter((t) => t.trace_id === taskResponse.trace_id);
      setTraceTasks(sameTraceTasks);

      const currentSelectedTask = sameTraceTasks.find((t) => t.id === (selectedTaskId || taskResponse.id)) || taskResponse;
      setTodoList(buildTodoList(currentSelectedTask, tasksResponse.tasks));
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

  const activeTask = traceTasks.find((t) => t.id === selectedTaskId) || task;

  const buildTreeData = (tasks: Task[]) => {
    const map = new Map<string, Task[]>();
    const roots: Task[] = [];
    tasks.forEach((t) => {
      if (!t.parent_id) {
        roots.push(t);
      } else {
        const list = map.get(t.parent_id) || [];
        list.push(t);
        map.set(t.parent_id, list);
      }
    });

    const convert = (node: Task): { key: string; title: React.ReactNode; children?: any[] } => ({
      key: node.id,
      title: (
        <Space size={6}>
          <StatusBadge status={node.status as TaskStatus} />
          <span style={{ maxWidth: 180, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{node.name}</span>
        </Space>
      ),
      children: (map.get(node.id) || []).map(convert),
    });

    return roots.map(convert);
  };

  const treeData = buildTreeData(traceTasks);

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
      width={1200}
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
      ) : activeTask ? (
        <Row gutter={16}>
          <Col span={8}>
            <CardTreeContainer>
              <div style={{ fontWeight: 600, marginBottom: 12 }}>任务树</div>
              <Tree
                treeData={treeData}
                selectedKeys={selectedTaskId ? [selectedTaskId] : []}
                onSelect={(keys) => {
                  const id = keys[0] as string | undefined;
                  if (!id) return;
                  setSelectedTaskId(id);
                  const selected = traceTasks.find((t) => t.id === id);
                  if (selected) {
                    setTodoList(buildTodoList(selected, traceTasks));
                  }
                }}
                defaultExpandAll
                style={{ background: '#fff' }}
              />
            </CardTreeContainer>
          </Col>

          <Col span={16}>
            <Descriptions column={2} bordered size="small" title="基本信息">
              <Descriptions.Item label="任务ID">
                <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{activeTask.id}</span>
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <StatusBadge status={activeTask.status as TaskStatus} />
              </Descriptions.Item>
              <Descriptions.Item label="任务名称">{activeTask.name}</Descriptions.Item>
              <Descriptions.Item label="类型">
                <Tag color="blue">{activeTask.type}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="TraceID">
                <span style={{ fontFamily: 'monospace', fontSize: 11 }}>{activeTask.trace_id}</span>
              </Descriptions.Item>
              <Descriptions.Item label="SpanID">
                <span style={{ fontFamily: 'monospace', fontSize: 11 }}>{activeTask.span_id}</span>
              </Descriptions.Item>
              <Descriptions.Item label="优先级">{activeTask.priority}</Descriptions.Item>
              <Descriptions.Item label="超时">{activeTask.timeout}ms</Descriptions.Item>
              <Descriptions.Item label="创建时间">
                {new Date(activeTask.created_at).toLocaleString()}
              </Descriptions.Item>
              <Descriptions.Item label="开始时间">
                {activeTask.started_at ? new Date(activeTask.started_at).toLocaleString() : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="完成时间">
                {activeTask.finished_at ? new Date(activeTask.finished_at).toLocaleString() : '-'}
              </Descriptions.Item>
            </Descriptions>
            <Divider />

            {activeTask.description && (
              <Descriptions column={1} bordered size="small" title="描述">
                <Descriptions.Item>{activeTask.description}</Descriptions.Item>
              </Descriptions>
            )}

            <Descriptions column={1} bordered size="small" title="执行进度">
              <Descriptions.Item>
                <ProgressBar progress={activeTask.progress} />
                <div style={{ marginTop: 8 }}>
                  <Tag color={activeTask.progress.stage ? 'blue' : 'default'}>
                    {activeTask.progress.stage || '无'}
                  </Tag>
                  <span style={{ marginLeft: 8, color: '#666' }}>
                    {activeTask.progress.detail || '-'}
                  </span>
                </div>
              </Descriptions.Item>
            </Descriptions>
            <Divider />

            {activeTask.error && (
              <Descriptions column={1} bordered size="small" title="错误信息">
                <Descriptions.Item>
                  <pre style={{ color: 'red', margin: 0 }}>{activeTask.error}</pre>
                </Descriptions.Item>
              </Descriptions>
            )}

            {activeTask.result && (
              <Descriptions column={1} bordered size="small" title="执行结果">
                <Descriptions.Item>
                  <pre style={{ margin: 0 }}>{JSON.stringify(activeTask.result, null, 2)}</pre>
                </Descriptions.Item>
              </Descriptions>
            )}

            <Divider />
            <TodoList todoList={todoList} loading={loading} />
          </Col>
        </Row>
      ) : (
        <div style={{ textAlign: 'center', padding: 50, color: '#999' }}>
          未找到任务
        </div>
      )}
    </Drawer>
  );
};

export default TaskDetailDrawer;

const CardTreeContainer: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <div
    style={{
      border: '1px solid #f0f0f0',
      borderRadius: 8,
      padding: 12,
      minHeight: 680,
      maxHeight: 680,
      overflow: 'auto',
      background: '#fafafa',
    }}
  >
    {children}
  </div>
);
