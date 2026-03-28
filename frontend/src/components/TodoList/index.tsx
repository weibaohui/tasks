/**
 * TodoList 组件
 * 展示父任务的子任务列表，点击展开显示执行详情
 */
import React, { useState } from 'react';
import { Card, Progress, Tag, Tooltip, Space, Empty } from 'antd';
import {
  CheckCircleOutlined, ClockCircleOutlined, CloseCircleOutlined,
  SyncOutlined, RightOutlined, DownOutlined,
} from '@ant-design/icons';
import { StatusBadge } from '../StatusBadge';
import type { Task, TodoItem, TodoStatus } from '../../types/task';

const statusConfig: Record<TodoStatus, { color: string; icon: React.ReactNode; text: string }> = {
  distributed: { color: 'blue', icon: <ClockCircleOutlined />, text: '已分发' },
  running: { color: 'processing', icon: <SyncOutlined spin />, text: '执行中' },
  completed: { color: 'success', icon: <CheckCircleOutlined />, text: '已完成' },
  failed: { color: 'error', icon: <CloseCircleOutlined />, text: '失败' },
  cancelled: { color: 'default', icon: <CloseCircleOutlined />, text: '已取消' },
};

const taskTypeLabels: Record<string, string> = {
  agent: 'Agent',
  '3': 'Agent',
};

const getTaskTypeLabel = (taskType: string): string => taskTypeLabels[taskType] || taskType || '未知';

/** 展开/收起箭头 */
const ExpandIcon: React.FC<{ active: boolean }> = ({ active }) => (
  <span style={{ fontSize: 12, color: '#999', marginRight: 4, transition: 'transform 0.2s' }}>
    {active ? <DownOutlined /> : <RightOutlined />}
  </span>
);

/** 子任务执行详情（展开内容） */
const SubTaskDetail: React.FC<{ childTask: Task | undefined }> = ({ childTask }) => {
  if (!childTask) {
    return <div style={{ color: '#999', padding: '8px 0' }}>暂无详细执行信息</div>;
  }

  const fields: { label: string; value: React.ReactNode; visible: boolean }[] = [
    {
      label: '状态',
      value: <StatusBadge status={childTask.status} />,
      visible: true,
    },
    {
      label: '进度',
      value: (
        <Progress
          percent={Math.round(childTask.progress?.value || 0)}
          size="small"
          strokeColor={(childTask.progress?.value || 0) >= 50 ? '#52c41a' : '#1890ff'}
        />
      ),
      visible: childTask.status === 'running' || childTask.status === 'pending',
    },
    {
      label: '任务结论',
      value: <div style={{ whiteSpace: 'pre-wrap', lineHeight: 1.6 }}>{childTask.task_conclusion}</div>,
      visible: !!childTask.task_conclusion,
    },
    {
      label: '分析',
      value: <div style={{ whiteSpace: 'pre-wrap', lineHeight: 1.6, color: '#666' }}>{childTask.analysis}</div>,
      visible: !!childTask.analysis,
    },
    {
      label: '任务要求',
      value: <div style={{ whiteSpace: 'pre-wrap', lineHeight: 1.6 }}>{childTask.task_requirement}</div>,
      visible: !!childTask.task_requirement,
    },
    {
      label: '验收标准',
      value: <div style={{ whiteSpace: 'pre-wrap', lineHeight: 1.6 }}>{childTask.acceptance_criteria}</div>,
      visible: !!childTask.acceptance_criteria,
    },
    {
      label: '错误信息',
      value: <pre style={{ color: 'red', margin: 0, whiteSpace: 'pre-wrap' }}>{childTask.error}</pre>,
      visible: !!childTask.error,
    },
    {
      label: '开始时间',
      value: childTask.started_at ? new Date(childTask.started_at).toLocaleString() : '-',
      visible: true,
    },
    {
      label: '完成时间',
      value: childTask.finished_at ? new Date(childTask.finished_at).toLocaleString() : '-',
      visible: !!childTask.finished_at,
    },
  ];

  const visibleFields = fields.filter((f) => f.visible);

  return (
    <div style={{ padding: '8px 0 0 28px' }}>
      {visibleFields.map((field) => (
        <div key={field.label} style={{ marginBottom: 8, display: 'flex', gap: 8 }}>
          <span style={{ color: '#999', whiteSpace: 'nowrap', minWidth: 70, fontSize: 13 }}>{field.label}:</span>
          <span style={{ flex: 1, fontSize: 13 }}>{field.value}</span>
        </div>
      ))}
    </div>
  );
};

interface TodoListProps {
  todoList: {
    task_id: string;
    items: TodoItem[];
    created_at: number;
    updated_at: number;
  } | null;
  childTasks?: Task[];
  loading?: boolean;
}

export const TodoList: React.FC<TodoListProps> = ({ todoList, childTasks = [], loading }) => {
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(new Set());

  const toggleExpand = (id: string) => {
    setExpandedKeys((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  if (loading) {
    return <Card title="子任务列表" loading />;
  }

  if (!todoList || todoList.items.length === 0) {
    return (
      <Card title="子任务列表">
        <Empty description="暂无子任务" />
      </Card>
    );
  }

  const completedCount = todoList.items.filter((item) => item.status === 'completed').length;
  const totalCount = todoList.items.length;
  const overallProgress = Math.round((completedCount / totalCount) * 100);

  // 建立 childTasks 的快速查找 map
  const childTaskMap = new Map<string, Task>();
  childTasks.forEach((t) => childTaskMap.set(t.id, t));

  return (
    <Card
      title={
        <Space>
          <span>子任务列表</span>
          <Tag color="blue">{totalCount} 个</Tag>
        </Space>
      }
      extra={
        <Space>
          <span style={{ color: '#52c41a', fontWeight: 500 }}>
            {completedCount}/{totalCount} 完成
          </span>
          <Progress type="circle" percent={overallProgress} size={40} />
        </Space>
      }
    >
      <div style={{ maxHeight: 600, overflowY: 'auto' }}>
        {todoList.items.map((item) => {
          const config = statusConfig[item.status] || statusConfig.distributed;
          const isActive = item.status === 'running' || item.status === 'distributed';
          const isExpanded = expandedKeys.has(item.sub_task_id);
          const childTask = childTaskMap.get(item.sub_task_id);

          return (
            <div
              key={item.sub_task_id}
              style={{
                borderBottom: '1px solid #f0f0f0',
                background: isExpanded ? '#fafafa' : undefined,
              }}
            >
              {/* 子任务标题行 - 可点击展开 */}
              <div
                onClick={() => toggleExpand(item.sub_task_id)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  padding: '10px 12px',
                  cursor: 'pointer',
                  transition: 'background 0.2s',
                }}
              >
                <ExpandIcon active={isExpanded} />
                <span style={{ fontSize: 16, marginRight: 8 }}>{config.icon}</span>

                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span style={{ fontWeight: 500 }}>{item.goal || '未命名'}</span>
                    <Tag color="blue">{getTaskTypeLabel(item.sub_task_type)}</Tag>
                    <Tag color={config.color === 'processing' ? undefined : config.color}>
                      {config.text}
                    </Tag>
                  </div>

                  {isActive && (
                    <Progress
                      percent={item.progress || 0}
                      size="small"
                      strokeColor={(item.progress || 0) >= 50 ? '#52c41a' : '#1890ff'}
                      style={{ marginTop: 4 }}
                    />
                  )}
                </div>

                <Tooltip title={`SpanID: ${item.span_id}`}>
                  <span style={{ fontFamily: 'monospace', fontSize: 11, color: '#999' }}>
                    {item.span_id.slice(-12)}
                  </span>
                </Tooltip>
              </div>

              {/* 展开的执行详情 */}
              {isExpanded && <SubTaskDetail childTask={childTask} />}
            </div>
          );
        })}
      </div>

      <div style={{ marginTop: 12, fontSize: 12, color: '#999' }}>
        更新于: {new Date(todoList.updated_at).toLocaleString()}
      </div>
    </Card>
  );
};

export default TodoList;
