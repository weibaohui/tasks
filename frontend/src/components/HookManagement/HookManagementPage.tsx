/**
 * Hook 配置管理页面
 * 支持 Hook 配置的查看、新增、编辑、删除、启用/停用
 */
import React, { useEffect, useState } from 'react';
import { Card, Typography, Space, Button, Table, Tag, Drawer } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useHookManagement } from './hooks';
import { HookTable } from './components/HookTable';
import { HookEditDrawer } from './components/HookEditDrawer';
import type { HookConfig, CreateHookConfigRequest } from '../../types/hook';

const { Title } = Typography;

export const HookManagementPage: React.FC = () => {
  const {
    items,
    loading,
    saving,
    logs,
    logsLoading,
    open,
    editing,
    fetchList,
    openEditor,
    closeEditor,
    handleDelete,
    handleToggleEnabled,
    handleSubmit,
    fetchLogs,
    clearLogs,
  } = useHookManagement();

  const [logsDrawerOpen, setLogsDrawerOpen] = useState(false);

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  const handleViewLogs = (record: HookConfig) => {
    fetchLogs(undefined, record.id);
    setLogsDrawerOpen(true);
  };

  const handleCloseLogs = () => {
    setLogsDrawerOpen(false);
    clearLogs();
  };

  const logColumns = [
    { title: '需求 ID', dataIndex: 'requirement_id', key: 'requirement_id', width: 150 },
    { title: '触发点', dataIndex: 'trigger_point', key: 'trigger_point', width: 120 },
    { title: '动作类型', dataIndex: 'action_type', key: 'action_type', width: 100 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: string) => {
        const color = status === 'success' ? 'green' : status === 'failed' ? 'red' : 'default';
        return <Tag color={color}>{status}</Tag>;
      },
    },
    { title: '输入', dataIndex: 'input_context', key: 'input_context', ellipsis: true },
    { title: '结果', dataIndex: 'result', key: 'result', ellipsis: true },
    { title: '错误', dataIndex: 'error', key: 'error', ellipsis: true },
    {
      title: '开始时间',
      dataIndex: 'started_at',
      key: 'started_at',
      width: 160,
      render: (ts: number) => new Date(ts).toLocaleString(),
    },
  ];

  return (
    <div style={{ padding: 0 }}>
      <Card
        title={<Title level={3} style={{ margin: 0 }}>Hook 配置管理</Title>}
        extra={
          <Space>
            <Button onClick={fetchList}>刷新</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openEditor(null)}>
              新建 Hook
            </Button>
          </Space>
        }
      >
        <HookTable
          items={items}
          loading={loading}
          onEdit={openEditor}
          onDelete={handleDelete}
          onToggleEnabled={handleToggleEnabled}
          onViewLogs={handleViewLogs}
        />
      </Card>

      {/* 编辑抽屉 */}
      <HookEditDrawer
        open={open}
        editing={editing}
        saving={saving}
        onClose={closeEditor}
        onSubmit={handleSubmit as (values: CreateHookConfigRequest) => Promise<void>}
      />

      {/* 日志抽屉 */}
      <Drawer
        title="Hook 执行日志"
        placement="right"
        width={900}
        onClose={handleCloseLogs}
        open={logsDrawerOpen}
      >
        <Table
          columns={logColumns}
          dataSource={logs}
          loading={logsLoading}
          rowKey="id"
          pagination={{ pageSize: 20 }}
          size="small"
          scroll={{ x: 1200 }}
        />
      </Drawer>
    </div>
  );
};

export default HookManagementPage;