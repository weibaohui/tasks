/**
 * 任务详情抽屉组件
 * 展示任务详情和子任务列表
 */
import React, { useCallback, useEffect, useState } from 'react';
import { Drawer, Descriptions, Tag, Button, Space, Spin, Row, Col, Divider, Tree } from 'antd';
import { TeamOutlined, ReloadOutlined } from '@ant-design/icons';
import { StatusBadge } from '../StatusBadge';
import { ProgressBar } from '../ProgressBar';
import { TodoList } from '../TodoList';
import { getAllTasks, getTask } from '../../api/taskApi';
import type { Task, TodoItem, TodoList as TodoListType, TaskStatus } from '../../types/task';

// 解析 TodoList 从 task.todo_list 字段
const parseTodoList = (todoListStr: string | undefined): TodoListType | null => {
  if (!todoListStr) return null;
  try {
    return JSON.parse(todoListStr) as TodoListType;
  } catch {
    return null;
  }
};

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

  const normalizeStatus = useCallback((status: Task['status']): TodoItem['status'] => {
    if (status === 'pending') return 'distributed';
    if (status === 'running') return 'running';
    if (status === 'completed') return 'completed';
    if (status === 'failed') return 'failed';
    return 'cancelled';
  }, []);

  const buildTodoList = useCallback((currentTask: Task, allTasks: Task[]): TodoListType | null => {
    const childTasks = allTasks.filter((t) => t.parent_id === currentTask.id);
    const dbTodoList = parseTodoList(currentTask.todo_list);
    const baseMap = new Map<string, TodoItem>();

    if (dbTodoList?.items) {
      dbTodoList.items.forEach((item) => {
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
        progress: Math.round(child.progress?.value || 0),
        span_id: child.span_id,
        created_at: child.created_at,
        completed_at: child.finished_at,
      };
    });

    if (mergedItems.length === 0 && dbTodoList) {
      return dbTodoList;
    }

    return {
      task_id: currentTask.id,
      items: mergedItems,
      created_at: dbTodoList?.created_at || Date.now(),
      updated_at: Date.now(),
    };
  }, [normalizeStatus]);

  const loadTask = useCallback(async (silent: boolean, preferredTaskId?: string) => {
    if (!taskId) return;
    if (!silent) setLoading(true);
    try {
      const [taskResponse, tasksResponse] = await Promise.all([getTask(taskId), getAllTasks()]);
      setTask(taskResponse);
      setSelectedTaskId((prev) => prev || taskResponse.id);

      const sameTraceTasks = tasksResponse.tasks.filter((t) => t.trace_id === taskResponse.trace_id);
      setTraceTasks(sameTraceTasks);

      const selectedId = preferredTaskId || selectedTaskId || taskResponse.id;
      const currentSelectedTask = sameTraceTasks.find((t) => t.id === selectedId) || taskResponse;
      setTodoList(buildTodoList(currentSelectedTask, tasksResponse.tasks));
    } catch (error) {
      console.error('Failed to load task:', error);
    } finally {
      if (!silent) setLoading(false);
    }
  }, [taskId, selectedTaskId, buildTodoList]);

  useEffect(() => {
    if (!taskId || !open) return;
    loadTask(false);
  }, [taskId, open, loadTask]);

  useEffect(() => {
    if (!selectedTaskId || !open) return;
    if (selectedTaskId === taskId) return;
    loadTask(false, selectedTaskId);
  }, [selectedTaskId, open, taskId, loadTask]);

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
              <Descriptions.Item label="深度">{activeTask.depth}</Descriptions.Item>
              <Descriptions.Item label="父Span">{activeTask.parent_span || '-'}</Descriptions.Item>
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

            {activeTask.task_requirement && (
              <Descriptions column={1} bordered size="small" title="任务要求">
                <Descriptions.Item>{activeTask.task_requirement}</Descriptions.Item>
              </Descriptions>
            )}

            {activeTask.acceptance_criteria && (
              <Descriptions column={1} bordered size="small" title="验收标准">
                <Descriptions.Item>{activeTask.acceptance_criteria}</Descriptions.Item>
              </Descriptions>
            )}

            <Descriptions column={1} bordered size="small" title="执行进度">
              <Descriptions.Item>
                <ProgressBar progress={activeTask.progress} />
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

            <Divider />

            <ExecutionSummaryPanel task={activeTask} traceTasks={traceTasks} />

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

const ExpandableText: React.FC<{ text: string; maxLen?: number }> = ({ text, maxLen = 60 }) => {
  const [expanded, setExpanded] = useState(false);
  if (text.length <= maxLen) {
    return <span style={{ whiteSpace: 'pre-wrap' }}>{text}</span>;
  }
  return expanded ? (
    <span>
      <span style={{ whiteSpace: 'pre-wrap' }}>{text}</span>
      <a onClick={() => setExpanded(false)} style={{ marginLeft: 4, fontSize: 12 }}>收起</a>
    </span>
  ) : (
    <span>
      {text.slice(0, maxLen)}...
      <a onClick={() => setExpanded(true)} style={{ marginLeft: 4, fontSize: 12 }}>展开</a>
    </span>
  );
};

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

/**
 * 任务执行结果面板
 */
const ExecutionSummaryPanel: React.FC<{ task: Task; traceTasks: Task[] }> = ({ task, traceTasks }) => {
  const childTasks = traceTasks.filter((t) => t.parent_id === task.id);

  // 如果没有任何结果信息，则不显示
  if (!task.task_conclusion && !task.analysis && childTasks.length === 0) {
    return null;
  }

  return (
    <Descriptions column={1} bordered size="small" title="执行结果">
      <Descriptions.Item>
        {task.task_conclusion && (
          <div style={{ marginBottom: 16 }}>
            <div style={{ fontWeight: 500, marginBottom: 8 }}>任务结论：</div>
            <div style={{ whiteSpace: 'pre-wrap' }}>{task.task_conclusion}</div>
          </div>
        )}

        {task.analysis && (
          <div style={{ marginBottom: 16 }}>
            <div style={{ fontWeight: 500, marginBottom: 8 }}>分析：</div>
            <div style={{ whiteSpace: 'pre-wrap', color: '#666' }}>{task.analysis}</div>
          </div>
        )}

        {childTasks.length > 0 && (
          <div>
            <div style={{ fontWeight: 500, marginBottom: 8 }}>子任务：</div>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
              <thead>
                <tr style={{ background: '#fafafa' }}>
                  <th style={{ padding: '6px 8px', border: '1px solid #f0f0f0', textAlign: 'left' }}>任务ID</th>
                  <th style={{ padding: '6px 8px', border: '1px solid #f0f0f0', textAlign: 'left' }}>名称</th>
                  <th style={{ padding: '6px 8px', border: '1px solid #f0f0f0', textAlign: 'left' }}>状态</th>
                  <th style={{ padding: '6px 8px', border: '1px solid #f0f0f0', textAlign: 'left' }}>结论</th>
                </tr>
              </thead>
              <tbody>
                {childTasks.map((child) => (
                  <tr key={child.id}>
                    <td style={{ padding: '6px 8px', border: '1px solid #f0f0f0', fontFamily: 'monospace' }}>
                      {child.id.slice(0, 8)}...
                    </td>
                    <td style={{ padding: '6px 8px', border: '1px solid #f0f0f0' }}>{child.name}</td>
                    <td style={{ padding: '6px 8px', border: '1px solid #f0f0f0' }}>
                      <StatusBadge status={child.status as TaskStatus} />
                    </td>
                    <td style={{ padding: '6px 8px', border: '1px solid #f0f0f0', color: '#52c41a' }}>
                      {child.task_conclusion
                        ? <ExpandableText text={child.task_conclusion} />
                        : '-'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Descriptions.Item>
    </Descriptions>
  );
};