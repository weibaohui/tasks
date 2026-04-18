import React, { useEffect, useMemo, useState } from 'react';
import { Button, Card, Collapse, Empty, Modal, Pagination, Select, Space, Switch, Table, Tag, message } from 'antd';
import { listProjectHeartbeatRuns, triggerHeartbeat } from '../../api/heartbeatApi';
import type { Project } from '../../types/projectRequirement';
import type { HeartbeatRunRecord } from '../../types/heartbeat';
import { getRequirement } from '../../api/projectRequirementApi';
import type { Requirement } from '../../types/projectRequirement';

interface ProjectRunsPanelProps {
  project: Project | null;
}

/**
 * ProjectRunsPanel 展示项目维度的心跳执行记录聚合视图。
 */
export const ProjectRunsPanel: React.FC<ProjectRunsPanelProps> = ({ project }) => {
  const [loading, setLoading] = useState(false);
  const [records, setRecords] = useState<HeartbeatRunRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const pageSize = 20;
  const [sourceFilter, setSourceFilter] = useState<string>('all');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailRequirement, setDetailRequirement] = useState<Requirement | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [failedFirst, setFailedFirst] = useState(true);
  const [retryingHeartbeatID, setRetryingHeartbeatID] = useState<string>('');

  /**
   * fetchRuns 拉取项目维度的心跳执行记录。
   */
  const fetchRuns = async (nextPage?: number) => {
    if (!project?.id) {
      setRecords([]);
      setTotal(0);
      return;
    }
    const currentPage = nextPage || page;
    const offset = (currentPage - 1) * pageSize;
    const statuses = statusFilter === 'all' ? [] : [statusFilter];
    setLoading(true);
    try {
      const data = await listProjectHeartbeatRuns(project.id, {
        limit: pageSize,
        offset,
        statuses,
      });
      setRecords(data.data);
      setTotal(data.total);
    } catch {
      message.error('加载项目执行记录失败');
      setRecords([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  };

  /**
   * handleOpenRequirementDetail 打开并加载需求详情。
   */
  const handleOpenRequirementDetail = async (requirementId: string) => {
    setDetailOpen(true);
    setDetailLoading(true);
    try {
      const data = await getRequirement(requirementId);
      setDetailRequirement(data);
    } catch {
      message.error('加载需求详情失败');
      setDetailRequirement(null);
    } finally {
      setDetailLoading(false);
    }
  };

  /**
   * handleRetryHeartbeat 手动重试指定心跳。
   */
  const handleRetryHeartbeat = async (heartbeatID: string) => {
    setRetryingHeartbeatID(heartbeatID);
    try {
      await triggerHeartbeat(heartbeatID);
      message.success('已触发重试');
      await fetchRuns(page);
    } catch {
      message.error('触发重试失败');
    } finally {
      setRetryingHeartbeatID('');
    }
  };

  /**
   * getStatusColor 返回状态标签颜色。
   */
  const getStatusColor = (status: string) => {
    if (status === 'failed') {
      return 'red';
    }
    if (status === 'todo' || status === 'preparing') {
      return 'gold';
    }
    if (status === 'coding' || status === 'running') {
      return 'blue';
    }
    if (status === 'completed' || status === 'done') {
      return 'green';
    }
    return 'default';
  };

  useEffect(() => {
    setPage(1);
  }, [project?.id, statusFilter]);

  useEffect(() => {
    void fetchRuns(page);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [project?.id, page, statusFilter]);

  useEffect(() => {
    if (!autoRefresh || !project?.id) {
      return;
    }
    const timer = setInterval(() => {
      void fetchRuns(page);
    }, 15000);
    return () => clearInterval(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [autoRefresh, project?.id, page, statusFilter]);

  const sourceOptions = ['all', ...Array.from(new Set(records.map((item) => item.trigger_source || 'unknown')))];
  const statusOptions = ['all', 'failed', 'todo', 'preparing', 'running', 'coding', 'completed', 'done'];
  const filteredRecords = records.filter((item) => {
    const sourceOk = sourceFilter === 'all' || (item.trigger_source || 'unknown') === sourceFilter;
    const statusOk = statusFilter === 'all' || (item.status || '') === statusFilter;
    return sourceOk && statusOk;
  });
  const displayedRecords = [...filteredRecords];
  if (failedFirst) {
    displayedRecords.sort((a, b) => {
      const af = a.status === 'failed' ? 0 : 1;
      const bf = b.status === 'failed' ? 0 : 1;
      if (af !== bf) {
        return af - bf;
      }
      return b.created_at - a.created_at;
    });
  }
  const groupedRecords = useMemo(() => {
    const grouped = new Map<string, { heartbeatID: string; heartbeatName: string; items: HeartbeatRunRecord[] }>();
    for (const item of displayedRecords) {
      const key = item.heartbeat_id || `name:${item.heartbeat_name || 'unknown'}`;
      const current = grouped.get(key);
      if (current) {
        current.items.push(item);
      } else {
        grouped.set(key, {
          heartbeatID: item.heartbeat_id,
          heartbeatName: item.heartbeat_name || '未识别心跳',
          items: [item],
        });
      }
    }
    const result = Array.from(grouped.values());
    result.forEach((group) => {
      group.items.sort((a, b) => b.created_at - a.created_at);
    });
    result.sort((a, b) => {
      const aLatest = a.items[0]?.created_at || 0;
      const bLatest = b.items[0]?.created_at || 0;
      const aFailed = a.items.some((it) => it.status === 'failed') ? 0 : 1;
      const bFailed = b.items.some((it) => it.status === 'failed') ? 0 : 1;
      if (failedFirst && aFailed !== bFailed) {
        return aFailed - bFailed;
      }
      return bLatest - aLatest;
    });
    return result;
  }, [displayedRecords, failedFirst]);

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
          <Select
            style={{ width: 160 }}
            value={sourceFilter}
            onChange={setSourceFilter}
            options={sourceOptions.map((value) => ({ label: value === 'all' ? '全部来源' : value, value }))}
          />
          <Select
            style={{ width: 160 }}
            value={statusFilter}
            onChange={setStatusFilter}
            options={statusOptions.map((value) => ({ label: value === 'all' ? '全部状态' : `状态:${value}`, value }))}
          />
          <Space size={4}>
            <span style={{ color: '#666', fontSize: 12 }}>自动刷新</span>
            <Switch size="small" checked={autoRefresh} onChange={setAutoRefresh} />
          </Space>
          <Space size={4}>
            <span style={{ color: '#666', fontSize: 12 }}>失败优先</span>
            <Switch size="small" checked={failedFirst} onChange={setFailedFirst} />
          </Space>
          <Button type="link" loading={loading} onClick={() => void fetchRuns()}>
            刷新记录
          </Button>
        </Space>
      }
    >
      <Collapse
        accordion={false}
        items={groupedRecords.map((group) => ({
          key: `${group.heartbeatID || group.heartbeatName}`,
          label: (
            <Space>
              <Tag color="purple">{group.heartbeatName}</Tag>
              <Tag>{`记录 ${group.items.length}`}</Tag>
              <Tag color={group.items.some((item) => item.status === 'failed') ? 'red' : 'green'}>
                {group.items.some((item) => item.status === 'failed') ? '含失败' : '正常'}
              </Tag>
            </Space>
          ),
          children: (
            <Table<HeartbeatRunRecord>
              rowKey="requirement_id"
              size="small"
              loading={loading}
              dataSource={group.items}
              pagination={false}
              columns={[
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
                  render: (status: string) => <Tag color={getStatusColor(status)}>{status || 'unknown'}</Tag>,
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
                  title: '错误分类',
                  dataIndex: 'error_category',
                  key: 'error_category',
                  width: 120,
                  render: (value: string) => <Tag color={value === 'none' ? 'default' : 'orange'}>{value || 'none'}</Tag>,
                },
                {
                  title: '需求',
                  dataIndex: 'requirement_id',
                  key: 'requirement_id',
                  width: 120,
                  render: (value: string) => (
                    <Button type="link" size="small" onClick={() => void handleOpenRequirementDetail(value)}>
                      查看详情
                    </Button>
                  ),
                },
                {
                  title: '重试',
                  dataIndex: 'heartbeat_id',
                  key: 'retry',
                  width: 110,
                  render: (heartbeatID: string) => (
                    <Button
                      type="link"
                      size="small"
                      loading={retryingHeartbeatID === heartbeatID}
                      disabled={!heartbeatID}
                      onClick={() => void handleRetryHeartbeat(heartbeatID)}
                    >
                      重试心跳
                    </Button>
                  ),
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
          ),
        }))}
      />
      <div style={{ marginTop: 12, display: 'flex', justifyContent: 'flex-end' }}>
        <Pagination
          current={page}
          pageSize={pageSize}
          total={total}
          showSizeChanger={false}
          onChange={(nextPage) => setPage(nextPage)}
          showTotal={(count) => `共 ${count} 条`}
        />
      </div>
      <Modal
        title="需求详情"
        open={detailOpen}
        onCancel={() => setDetailOpen(false)}
        footer={null}
        width={820}
      >
        {detailLoading ? (
          <div style={{ color: '#999' }}>加载中...</div>
        ) : !detailRequirement ? (
          <div style={{ color: '#999' }}>暂无详情</div>
        ) : (
          <Space direction="vertical" style={{ width: '100%' }}>
            <div><strong>需求ID：</strong>{detailRequirement.id}</div>
            <div><strong>标题：</strong>{detailRequirement.title}</div>
            <div><strong>状态：</strong><Tag color={getStatusColor(detailRequirement.status)}>{detailRequirement.status}</Tag></div>
            <div><strong>需求类型：</strong>{detailRequirement.requirement_type || '-'}</div>
            <div><strong>最近错误：</strong>{detailRequirement.last_error || '-'}</div>
            <div><strong>描述：</strong></div>
            <pre style={{ margin: 0, background: '#fafafa', border: '1px solid #f0f0f0', borderRadius: 6, padding: 12, whiteSpace: 'pre-wrap' }}>
              {detailRequirement.description || '无'}
            </pre>
            <div><strong>验收标准：</strong></div>
            <pre style={{ margin: 0, background: '#fafafa', border: '1px solid #f0f0f0', borderRadius: 6, padding: 12, whiteSpace: 'pre-wrap' }}>
              {detailRequirement.acceptance_criteria || '无'}
            </pre>
          </Space>
        )}
      </Modal>
    </Card>
  );
};
