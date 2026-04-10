/**
 * StateMachine Table Component
 */
import React from 'react';
import { Table, Button, Space, Tag, Popconfirm } from 'antd';

import type { StateMachine } from '../../../types/stateMachine';
import { ActionGroup } from "@/components/ActionGroup";

interface StateMachineTableProps {
  items: StateMachine[];
  loading: boolean;
  onEdit: (record: StateMachine) => void;
  onDelete: (id: string) => void;
  onInvoke?: (record: StateMachine) => void;
}

export const StateMachineTable: React.FC<StateMachineTableProps> = ({
  items,
  loading,
  onEdit,
  onDelete,
  onInvoke,
}) => {
  const columns = [
      {
            title: '操作',
            key: 'action',
            render: (_: unknown, record: StateMachine) => (
              <ActionGroup size="small">
                <Button onClick={() => onEdit(record)} type="link" size="small" style={{ padding: 0 }}>
                  编辑
                </Button>
                <Button
                  onClick={() => onInvoke?.(record)} type="link" size="small" style={{ padding: 0 }}
                >
                  调用
                </Button>
                <Popconfirm
                  title="确定删除此状态机？"
                  onConfirm={() => onDelete(record.id)}
                  okText="确认"
                  cancelText="取消"
                >
                  <Button danger type="link" size="small" style={{ padding: 0 }}>
                    删除
                  </Button>
                </Popconfirm>
              </ActionGroup>
            ),
              width: 100,
              fixed: 'left' as const
          },
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      width: 150,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '初始状态',
      dataIndex: ['config', 'initial_state'],
      key: 'initial_state',
      width: 120,
      render: (initialState: string, record: StateMachine) => {
        const state = record.config.states.find((s) => s.id === initialState);
        return state?.name || initialState;
      },
    },
    {
      title: '状态数',
      dataIndex: ['config', 'states'],
      key: 'states_count',
      width: 80,
      render: (states: StateMachine['config']['states']) => states?.length || 0,
    },
    {
      title: '转换数',
      dataIndex: ['config', 'transitions'],
      key: 'transitions_count',
      width: 80,
      render: (transitions: StateMachine['config']['transitions']) => transitions?.length || 0,
    },
    {
      title: '终态',
      key: 'final_states',
      width: 150,
      render: (_: unknown, record: StateMachine) => {
        const finalStates = record.config.states.filter((s) => s.is_final);
        return (
          <Space size={[0, 4]} wrap>
            {finalStates.map((s) => (
              <Tag key={s.id} color="green">
                {s.name}
              </Tag>
            ))}
          </Space>
        );
      },
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      render: (createdAt: string) => new Date(createdAt).toLocaleString(),
    }
  ];

  return (
    <Table<StateMachine>
      rowKey="id"
      loading={loading}
      dataSource={items}
      columns={columns}
      pagination={false}
      size="small"
      scroll={{ x: 1000 }}
    />
  );
};
