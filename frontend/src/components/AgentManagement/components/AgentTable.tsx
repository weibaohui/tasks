/**
 * AgentTable - Agent 表格组件
 */
import React, { useMemo } from 'react';
import { Button, InputNumber, Popconfirm, Space, Switch, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, EditOutlined } from '@ant-design/icons';
import type { Agent } from '../../../types/agent';

interface AgentTableProps {
  items: Agent[];
  loading: boolean;
  screens: Record<string, boolean>;
  onEdit: (agent: Agent) => void;
  onDelete: (id: string) => void;
  onSetDefault: (agent: Agent) => void;
  onToggleThinking: (agent: Agent, enabled: boolean) => void;
  onUpdateMaxIterations: (agent: Agent, value: number) => void;
}

export const AgentTable: React.FC<AgentTableProps> = ({
  items,
  loading,
  screens,
  onEdit,
  onDelete,
  onSetDefault,
  onToggleThinking,
  onUpdateMaxIterations,
}) => {
  const columns: ColumnsType<Agent> = useMemo(() => [
    ...(screens.xs ? [] : [{ title: 'ID', dataIndex: 'id', key: 'id', width: 120, ellipsis: true }]),
    { title: '名称', dataIndex: 'name', key: 'name', ellipsis: true },
    ...(screens.xs ? [] : [{ title: '描述', dataIndex: 'description', key: 'description', ellipsis: true }]),
    { title: screens.xs ? '模型' : '模型', dataIndex: 'model', key: 'model', width: screens.xs ? 120 : 180, ellipsis: true },
    {
      title: '类型', dataIndex: 'agent_type', key: 'agent_type', width: 100,
      render: (_: unknown, record: Agent) => {
        const typeMap: Record<string, string> = { BareLLM: '裸 LLM', CodingAgent: '编程' };
        return <Tag>{typeMap[record.agent_type] || record.agent_type || 'BareLLM'}</Tag>;
      },
    },
    {
      title: '思考', key: 'thinking', width: 80, align: 'center',
      render: (_: unknown, record: Agent) => (
        <Switch size="small" checked={record.enable_thinking_process} checkedChildren="开" unCheckedChildren="关"
          onChange={(checked) => onToggleThinking(record, checked)} />
      ),
    },
    {
      title: '轮数', key: 'max_iterations', width: 90, align: 'center',
      render: (_: unknown, record: Agent) => (
        <InputNumber size="small" min={1} max={50} defaultValue={record.max_iterations}
          onPressEnter={(e) => {
            const v = Number((e.target as HTMLInputElement).value);
            if (!Number.isNaN(v) && v !== record.max_iterations) onUpdateMaxIterations(record, v);
          }}
          onBlur={(e) => {
            const v = Number((e.target as HTMLInputElement).value);
            if (!Number.isNaN(v) && v !== record.max_iterations) onUpdateMaxIterations(record, v);
          }}
          style={{ width: 70 }} />
      ),
    },
    {
      title: '技能', key: 'skills', width: 70, align: 'center',
      render: (_: unknown, record: Agent) => {
        const count = (record.skills_list || []).length;
        return <Tag color={count === 0 ? 'default' : 'blue'}>{count === 0 ? '不限' : count}</Tag>;
      },
    },
    {
      title: '工具', key: 'tools', width: 70, align: 'center',
      render: (_: unknown, record: Agent) => {
        const count = (record.tools_list || []).length;
        return <Tag color={count === 0 ? 'default' : 'cyan'}>{count === 0 ? '不限' : count}</Tag>;
      },
    },
    {
      title: '状态', key: 'status', width: 120,
      render: (_: unknown, record: Agent) => (
        <Space size="small">
          {record.is_default && <Tag color="gold">默认</Tag>}
          <Tag color={record.is_active ? 'success' : 'default'}>{record.is_active ? '启用' : '停用'}</Tag>
        </Space>
      ),
    },
    {
      title: '操作', key: 'action', width: screens.xs ? 140 : 280,
      render: (_: unknown, record: Agent) => (
        <Space size={[4, 4]} wrap>
          <Button type="text" icon={<EditOutlined />} onClick={() => onEdit(record)}>
            {screens.xs ? '' : '编辑'}
          </Button>
          {!record.is_default && !screens.xs && (
            <Button type="text" onClick={() => onSetDefault(record)}>默认</Button>
          )}
          <Popconfirm title="确认删除该 Agent？" onConfirm={() => onDelete(record.id)}>
            <Button type="text" danger icon={<DeleteOutlined />}>
              {screens.xs ? '' : '删除'}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ], [screens.xs, onEdit, onDelete, onSetDefault, onToggleThinking, onUpdateMaxIterations]);

  return (
    <Table<Agent>
      rowKey="id"
      loading={loading}
      dataSource={items}
      columns={columns}
      size={screens.xs ? 'small' : 'middle'}
      scroll={{ x: screens.xs ? 760 : 'max-content' }}
    />
  );
};

export default AgentTable;
