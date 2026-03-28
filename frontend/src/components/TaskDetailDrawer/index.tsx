/**
 * 任务详情抽屉组件
 * 展示任务详情和子任务列表
 */
import React, { useCallback, useEffect, useState } from 'react';
import { Drawer, Tag, Button, Space, Spin, Row, Col, Divider, Tree, Card } from 'antd';
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
          <Col style={{ width: 300, flexShrink: 0 }}>
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

          <Col flex="1">
            <Card size="small" title="基本信息" style={{ marginBottom: 16 }}>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '8px 24px' }}>
                <div><span style={{ color: '#999' }}>状态：</span><StatusBadge status={activeTask.status as TaskStatus} /></div>
                <div><span style={{ color: '#999' }}>类型：</span><Tag color="blue">{activeTask.type}</Tag></div>
                <div style={{ gridColumn: '1 / -1' }}><span style={{ color: '#999' }}>任务名称：</span>{activeTask.name}</div>
                <div><span style={{ color: '#999' }}>优先级：</span>{activeTask.priority}</div>
                <div><span style={{ color: '#999' }}>超时：</span>{Math.round(activeTask.timeout / 1e9)}s</div>
                <div><span style={{ color: '#999' }}>创建时间：</span>{new Date(activeTask.created_at).toLocaleString()}</div>
                <div><span style={{ color: '#999' }}>开始时间：</span>{activeTask.started_at ? new Date(activeTask.started_at).toLocaleString() : '-'}</div>
                <div><span style={{ color: '#999' }}>完成时间：</span>{activeTask.finished_at ? new Date(activeTask.finished_at).toLocaleString() : '-'}</div>
              </div>
            </Card>
            <Divider />

            <Card size="small" title="任务详情" style={{ marginBottom: 16 }}>
              {activeTask.description && (
                <div style={{ marginBottom: 12 }}>
                  <div style={{ fontWeight: 500, marginBottom: 4, color: '#666' }}>描述</div>
                  <div style={{ whiteSpace: 'pre-wrap', lineHeight: 1.6 }}>{activeTask.description}</div>
                </div>
              )}

              {activeTask.task_requirement && (
                <div style={{ marginBottom: 12 }}>
                  <div style={{ fontWeight: 500, marginBottom: 4, color: '#666' }}>任务要求</div>
                  <div style={{ whiteSpace: 'pre-wrap', lineHeight: 1.6 }}>{activeTask.task_requirement}</div>
                </div>
              )}

              {activeTask.acceptance_criteria && (
                <div style={{ marginBottom: 12 }}>
                  <div style={{ fontWeight: 500, marginBottom: 4, color: '#666' }}>验收标准</div>
                  <div style={{ whiteSpace: 'pre-wrap', lineHeight: 1.6 }}>{activeTask.acceptance_criteria}</div>
                </div>
              )}

              <div>
                <div style={{ fontWeight: 500, marginBottom: 4, color: '#666' }}>执行进度</div>
                <ProgressBar progress={activeTask.progress} />
              </div>
            </Card>

            {activeTask.error && (
              <Card size="small" title="错误信息" style={{ marginBottom: 16, borderColor: '#ffccc7' }}>
                <pre style={{ color: 'red', margin: 0, whiteSpace: 'pre-wrap' }}>{activeTask.error}</pre>
              </Card>
            )}

            {(activeTask.task_conclusion || activeTask.analysis) && (
              <Card size="small" title="执行结论" style={{ marginBottom: 16 }}>
                {activeTask.task_conclusion && (
                  <div style={{ marginBottom: activeTask.analysis ? 16 : 0 }}>
                    <div style={{ fontWeight: 500, marginBottom: 4, color: '#666' }}>任务结论</div>
                    <ExpandableContent text={activeTask.task_conclusion} maxLen={400} />
                  </div>
                )}
                {activeTask.analysis && (
                  <div>
                    <div style={{ fontWeight: 500, marginBottom: 4, color: '#666' }}>分析</div>
                    <ExpandableContent text={activeTask.analysis} maxLen={400} style={{ color: '#666' }} />
                  </div>
                )}
              </Card>
            )}

            <Divider />

            <TodoList
              todoList={todoList}
              childTasks={traceTasks.filter((t) => t.parent_id === activeTask.id)}
              loading={loading}
            />
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
      height: 'calc(100vh - 120px)',
      overflow: 'auto',
      background: '#fafafa',
    }}
  >
    {children}
  </div>
);

const ExpandableContent: React.FC<{ text: string; maxLen?: number; style?: React.CSSProperties }> = ({
  text,
  maxLen = 400,
  style,
}) => {
  const [expanded, setExpanded] = useState(false);

  if (text.length <= maxLen) {
    return <div style={{ whiteSpace: 'pre-wrap', lineHeight: 1.8, ...style }}>{text}</div>;
  }

  return (
    <div style={{ ...style }}>
      <div style={{ whiteSpace: 'pre-wrap', lineHeight: 1.8 }}>
        {expanded ? text : text.slice(0, maxLen) + '...'}
      </div>
      <a onClick={() => setExpanded(!expanded)} style={{ fontSize: 13 }}>
        {expanded ? '收起' : '更多'}
      </a>
    </div>
  );
};

