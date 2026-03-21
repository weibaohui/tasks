/**
 * TodoList 组件
 * 展示父任务的子任务列表及进度
 */
import React from 'react';
import { Card, Progress, Tag, Tooltip, Space } from 'antd';
import { CheckCircleOutlined, ClockCircleOutlined, CloseCircleOutlined, SyncOutlined } from '@ant-design/icons';
import type { TodoItem, TodoStatus } from '../../types/task';

const statusConfig: Record<TodoStatus, { color: string; icon: React.ReactNode; text: string }> = {
  distributed: { color: 'blue', icon: <ClockCircleOutlined />, text: '已分发' },
  running: { color: 'processing', icon: <SyncOutlined />, text: '执行中' },
  completed: { color: 'success', icon: <CheckCircleOutlined />, text: '已完成' },
  failed: { color: 'error', icon: <CloseCircleOutlined />, text: '失败' },
  cancelled: { color: 'default', icon: <CloseCircleOutlined />, text: '已取消' },
};

const taskTypeLabels: Record<string, string> = {
  data_processing: '数据处理',
  file_operation: '文件操作',
  api_call: 'API调用',
  custom: '自定义',
};

interface TodoItemRowProps {
  item: TodoItem;
}

const TodoItemRow: React.FC<TodoItemRowProps> = ({ item }) => {
  const config = statusConfig[item.status];
  const isActive = item.status === 'running' || item.status === 'distributed';

  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      padding: '8px 12px',
      borderBottom: '1px solid #f0f0f0',
      background: item.status === 'completed' ? '#f6ffed' : undefined,
    }}>
      <span style={{ fontSize: 16, marginRight: 8 }}>{config.icon}</span>

      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontWeight: 500 }}>{item.goal}</span>
          <Tag color={statusConfig[item.status].color === 'success' ? undefined : statusConfig[item.status].color}>
            {taskTypeLabels[item.sub_task_type] || item.sub_task_type}
          </Tag>
          <Tag>{config.text}</Tag>
        </div>

        {isActive && (
          <Progress
            percent={item.progress}
            size="small"
            strokeColor={item.progress >= 50 ? '#52c41a' : '#1890ff'}
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
  );
};

interface TodoListProps {
  todoList: {
    task_id: string;
    items: TodoItem[];
    created_at: number;
    updated_at: number;
  } | null;
  loading?: boolean;
}

export const TodoList: React.FC<TodoListProps> = ({ todoList, loading }) => {
  if (loading) {
    return <Card title="📋 子任务列表" loading />;
  }

  if (!todoList || todoList.items.length === 0) {
    return (
      <Card title="📋 子任务列表">
        <div style={{ textAlign: 'center', color: '#999', padding: 24 }}>
          暂无子任务
        </div>
      </Card>
    );
  }

  const completedCount = todoList.items.filter(item => item.status === 'completed').length;
  const totalCount = todoList.items.length;
  const overallProgress = Math.round((completedCount / totalCount) * 100);

  return (
    <Card
      title={
        <Space>
          <span>📋 子任务列表</span>
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
      <div style={{ maxHeight: 400, overflowY: 'auto' }}>
        {todoList.items.map((item, index) => (
          <TodoItemRow key={item.sub_task_id || index} item={item} />
        ))}
      </div>

      <div style={{ marginTop: 12, fontSize: 12, color: '#999' }}>
        更新于: {new Date(todoList.updated_at).toLocaleString()}
      </div>
    </Card>
  );
};

export default TodoList;
