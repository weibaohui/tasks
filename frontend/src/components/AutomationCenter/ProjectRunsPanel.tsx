import React, { useEffect, useState } from 'react';
import { Button, Card, Empty, Space, Table, Tag, message } from 'antd';
import { listProjectHeartbeatRuns } from '../../api/heartbeatApi';
import type { Project } from '../../types/projectRequirement';
import type { HeartbeatRunRecord } from '../../types/heartbeat';

interface ProjectRunsPanelProps {
  project: Project | null;
}

/**
 * ProjectRunsPanel 展示项目维度的心跳执行记录聚合视图。
 */
export const ProjectRunsPanel: React.FC<ProjectRunsPanelProps> = ({ project }) => {
  const [loading, setLoading] = useState(false);
  const [records, setRecords] = useState<HeartbeatRunRecord[]>([]);

  /**
   * fetchRuns 拉取项目维度的心跳执行记录。
   */
  const fetchRuns = async () => {
    if (!project?.id) {
      setRecords([]);
      return;
    }
    setLoading(true);
    try {
      const data = await listProjectHeartbeatRuns(project.id, 50);
      setRecords(data);
    } catch {
      message.error('加载项目执行记录失败');
      setRecords([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void fetchRuns();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [project?.id]);

  if (!project) {
    return <Empty description="请先选择项目后查看运行记录" />;
  }

  return (
    <Card
      size="small"
      title="项目运行记录"
      extra={
        <Space>
          <Tag color="purple">{project.name}</Tag>
          <Button type="link" loading={loading} onClick={() => void fetchRuns()}>
            刷新记录
          </Button>
        </Space>
      }
    >
      <Table<HeartbeatRunRecord>
        rowKey="requirement_id"
        size="small"
        loading={loading}
        dataSource={records}
        pagination={{ pageSize: 10 }}
        columns={[
          {
            title: '心跳',
            dataIndex: 'heartbeat_name',
            key: 'heartbeat_name',
            width: 160,
          },
          {
            title: '触发来源',
            dataIndex: 'trigger_source',
            key: 'trigger_source',
            width: 120,
            render: (source: string) => <Tag color="blue">{source || 'unknown'}</Tag>,
          },
          {
            title: '状态',
            dataIndex: 'status',
            key: 'status',
            width: 100,
            render: (status: string) => <Tag color={status === 'failed' ? 'red' : 'green'}>{status}</Tag>,
          },
          {
            title: '标题',
            dataIndex: 'title',
            key: 'title',
          },
          {
            title: '最近错误',
            dataIndex: 'last_error',
            key: 'last_error',
            render: (value: string) => value || '-',
          },
          {
            title: '触发时间',
            dataIndex: 'created_at',
            key: 'created_at',
            width: 180,
            render: (value: number) => new Date(value).toLocaleString(),
          },
        ]}
      />
    </Card>
  );
};
