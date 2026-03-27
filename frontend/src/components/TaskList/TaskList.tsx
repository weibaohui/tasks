/**
 * 任务列表组件
 * 展示任务列表，支持查看详情、查看对话和取消操作
 */
import React, { useState } from 'react';
import { Table, Tag, Space, Button } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { MessageOutlined } from '@ant-design/icons';
import { StatusBadge } from '../StatusBadge';
import { ProgressBar } from '../ProgressBar';
import { ConversationChatModal } from '../ConversationChatModal';
import type { Task } from '../../types/task';
import { useNavigate } from 'react-router-dom';

interface TaskListProps {
  tasks: Task[];
  loading?: boolean;
  onCancel?: (taskId: string) => void;
}

export const TaskList: React.FC<TaskListProps> = ({ tasks, loading, onCancel }) => {
  const navigate = useNavigate();
  const [chatTraceId, setChatTraceId] = useState<string | null>(null);
  const [chatOpen, setChatOpen] = useState(false);

  const handleOpenChat = (traceId: string) => {
    setChatTraceId(traceId);
    setChatOpen(true);
  };

  const columns: ColumnsType<Task> = [
    {
      title: '任务名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: Task) => (
        <a onClick={() => navigate(`/tasks/${record.id}`)}>{name}</a>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: Task['status']) => <StatusBadge status={status} />,
    },
    {
      title: '进度',
      dataIndex: 'progress',
      key: 'progress',
      render: (progress: Task['progress']) => <ProgressBar progress={progress} />,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: Task['type']) => <Tag>{type}</Tag>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      sorter: (a, b) => a.created_at - b.created_at,
      defaultSortOrder: 'descend',
      render: (timestamp: number) => new Date(timestamp).toLocaleString(),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: Task) => (
        <Space>
          {record.status === 'pending' && (
            <Button size="small" type="primary" onClick={() => navigate(`/tasks/${record.id}?action=start`)}>
              启动
            </Button>
          )}
          <Button size="small" onClick={() => navigate(`/tasks/${record.id}`)}>
            详情
          </Button>
          <Button
            size="small"
            icon={<MessageOutlined />}
            onClick={() => handleOpenChat(record.trace_id)}
          >
            对话
          </Button>
          {record.status === 'running' && (
            <Button size="small" danger onClick={() => onCancel?.(record.id)}>
              取消
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <>
      <Table columns={columns} dataSource={tasks} rowKey="id" loading={loading} />
      <ConversationChatModal
        traceId={chatTraceId}
        open={chatOpen}
        onClose={() => setChatOpen(false)}
      />
    </>
  );
};