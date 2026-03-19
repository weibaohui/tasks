/**
 * 任务详情组件
 * 展示任务详细信息
 */
import React from 'react';
import { Card, Descriptions, Button, Space, Spin } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { StatusBadge } from '../StatusBadge';
import { ProgressBar } from '../ProgressBar';
import type { Task } from '../../types/task';

interface TaskDetailProps {
  task: Task | null;
  loading?: boolean;
  onCancel?: () => void;
  onBack?: () => void;
}

export const TaskDetail: React.FC<TaskDetailProps> = ({ task, loading, onCancel, onBack }) => {
  if (loading || !task) {
    return <Spin style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />;
  }

  return (
    <div>
      {onBack && (
        <Button icon={<ArrowLeftOutlined />} onClick={onBack} style={{ marginBottom: 16 }}>
          返回
        </Button>
      )}

      <Card title="任务详情">
        <Descriptions column={2} bordered>
          <Descriptions.Item label="任务ID">{task.id}</Descriptions.Item>
          <Descriptions.Item label="任务名称">{task.name}</Descriptions.Item>
          <Descriptions.Item label="状态">
            <StatusBadge status={task.status} />
          </Descriptions.Item>
          <Descriptions.Item label="类型">{task.type}</Descriptions.Item>
          <Descriptions.Item label="优先级">{task.priority}</Descriptions.Item>
          <Descriptions.Item label="超时时间">{task.timeout}ms</Descriptions.Item>
          <Descriptions.Item label="创建时间">
            {new Date(task.created_at).toLocaleString()}
          </Descriptions.Item>
          <Descriptions.Item label="开始时间">
            {task.started_at ? new Date(task.started_at).toLocaleString() : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="完成时间">
            {task.finished_at ? new Date(task.finished_at).toLocaleString() : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="重试次数">{task.max_retries}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="执行进度" style={{ marginTop: 16 }}>
        <ProgressBar progress={task.progress} />
        <p><strong>阶段：</strong>{task.progress.stage || '-'}</p>
        <p><strong>详情：</strong>{task.progress.detail || '-'}</p>
      </Card>

      {task.error && (
        <Card title="错误信息" style={{ marginTop: 16 }}>
          <pre style={{ color: 'red' }}>{task.error}</pre>
        </Card>
      )}

      {task.result && (
        <Card title="执行结果" style={{ marginTop: 16 }}>
          <pre>{JSON.stringify(task.result, null, 2)}</pre>
        </Card>
      )}

      {task.status === 'running' && onCancel && (
        <Card style={{ marginTop: 16 }}>
          <Space>
            <Button danger onClick={onCancel}>
              取消任务
            </Button>
          </Space>
        </Card>
      )}
    </div>
  );
};
