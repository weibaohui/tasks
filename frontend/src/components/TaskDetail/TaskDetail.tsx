/**
 * 任务详情组件
 * 展示任务详细信息
 */
import React from 'react';
import { Card, Descriptions, Button, Space, Spin, Tag } from 'antd';
import { ArrowLeftOutlined, TeamOutlined } from '@ant-design/icons';
import { StatusBadge } from '../StatusBadge';
import { ProgressBar } from '../ProgressBar';
import type { Task } from '../../types/task';

interface TaskDetailProps {
  task: Task | null;
  loading?: boolean;
  onCancel?: () => void;
  onBack?: () => void;
  onViewTree?: (traceId: string) => void;
}

export const TaskDetail: React.FC<TaskDetailProps> = ({ task, loading, onCancel, onBack, onViewTree }) => {
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

      <Card
        title="任务详情"
        extra={onViewTree && (
          <Button icon={<TeamOutlined />} onClick={() => onViewTree(task.trace_id)}>
            查看任务树
          </Button>
        )}
      >
        <Descriptions column={2} bordered>
          <Descriptions.Item label="任务ID">{task.id}</Descriptions.Item>
          <Descriptions.Item label="任务名称">{task.name}</Descriptions.Item>
          <Descriptions.Item label="状态">
            <StatusBadge status={task.status} />
          </Descriptions.Item>
          <Descriptions.Item label="类型">
            <Tag color="blue">{task.type}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="TraceID">
            <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{task.trace_id}</span>
          </Descriptions.Item>
          <Descriptions.Item label="SpanID">
            <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{task.span_id}</span>
          </Descriptions.Item>
          <Descriptions.Item label="父任务">
            {task.parent_id ? <Tag color="orange">{task.parent_id}</Tag> : <span style={{ color: '#999' }}>顶级任务</span>}
          </Descriptions.Item>
          <Descriptions.Item label="优先级">{task.priority}</Descriptions.Item>
          <Descriptions.Item label="超时时间">{task.timeout}ms</Descriptions.Item>
          <Descriptions.Item label="最大重试">{task.max_retries}</Descriptions.Item>
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

        {task.description && (
          <Descriptions column={1} bordered style={{ marginTop: 16 }}>
            <Descriptions.Item label="任务描述">{task.description}</Descriptions.Item>
          </Descriptions>
        )}
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

      {task.result?.task_conclusion && (
        <Card title="执行结论" style={{ marginTop: 16 }}>
          <div style={{ whiteSpace: 'pre-wrap', lineHeight: 1.8 }}>{task.result.task_conclusion}</div>
        </Card>
      )}

      {task.result && (
        <Card title="执行结果" style={{ marginTop: 16 }}>
          <pre>{JSON.stringify(task.result, null, 2)}</pre>
        </Card>
      )}

      {task.metadata && Object.keys(task.metadata).length > 0 && (
        <Card title="元数据" style={{ marginTop: 16 }}>
          <pre>{JSON.stringify(task.metadata, null, 2)}</pre>
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
