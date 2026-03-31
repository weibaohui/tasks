/**
 * Hook 配置表格
 */
import React from 'react';
import { Table, Button, Space, Switch, Popconfirm } from 'antd';
import { EditOutlined, DeleteOutlined, FileTextOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { HookConfig } from '../../../types/hook';
import { TRIGGER_POINTS, ACTION_TYPES } from '../../../types/hook';

interface HookTableProps {
  items: HookConfig[];
  loading: boolean;
  onEdit: (record: HookConfig) => void;
  onDelete: (id: string) => void;
  onToggleEnabled: (record: HookConfig) => void;
  onViewLogs: (record: HookConfig) => void;
}

const getTriggerPointLabel = (value: string): string => {
  const found = TRIGGER_POINTS.find((tp) => tp.value === value);
  return found ? found.label : value;
};

const getActionTypeLabel = (value: string): string => {
  const found = ACTION_TYPES.find((at) => at.value === value);
  return found ? found.label : value;
};

export const HookTable: React.FC<HookTableProps> = ({
  items,
  loading,
  onEdit,
  onDelete,
  onToggleEnabled,
  onViewLogs,
}) => {
  const columns: ColumnsType<HookConfig> = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      width: 150,
    },
    {
      title: '触发点',
      dataIndex: 'trigger_point',
      key: 'trigger_point',
      width: 200,
      render: (value: string) => getTriggerPointLabel(value),
    },
    {
      title: '动作类型',
      dataIndex: 'action_type',
      key: 'action_type',
      width: 180,
      render: (value: string) => getActionTypeLabel(value),
    },
    {
      title: '配置',
      dataIndex: 'action_config',
      key: 'action_config',
      ellipsis: true,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 80,
    },
    {
      title: '启用',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (enabled: boolean, record) => (
        <Switch checked={enabled} onChange={() => onToggleEnabled(record)} />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<FileTextOutlined />}
            onClick={() => onViewLogs(record)}
          >
            日志
          </Button>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => onEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定删除此 Hook 配置？"
            onConfirm={() => onDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Table
      columns={columns}
      dataSource={items}
      loading={loading}
      rowKey="id"
      pagination={{ pageSize: 10 }}
      size="small"
    />
  );
};