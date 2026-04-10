/**
 * MCPServerBindingCard - MCP Server 绑定卡片
 */
import React from 'react';
import { ApiOutlined } from '@ant-design/icons';
import { Button, Card, Popconfirm, Select, Switch, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { FormInstance } from 'antd/es/form';
import type { AgentMCPBinding, MCPServer } from '../../../../types/mcp';
import type { Agent } from '../../../../types/agent';
import { ActionGroup } from "@/components/ActionGroup";

interface MCPServerBindingCardProps {
  editing: Agent | null;
  mcpServers: MCPServer[];
  mcpBindings: AgentMCPBinding[];
  mcpLoading: boolean;
  mcpForm: FormInstance;
  onReloadMCP: () => Promise<void>;
  onCreateBinding: (mcpServerId: string) => Promise<void>;
  onUpdateBinding: (bindingId: string, fields: Record<string, unknown>) => Promise<void>;
  onDeleteBinding: (bindingId: string) => Promise<void>;
  onOpenToolsDrawer: (binding: AgentMCPBinding) => void;
}

export const MCPServerBindingCard: React.FC<MCPServerBindingCardProps> = ({
  editing, mcpServers, mcpBindings, mcpLoading, mcpForm,
  onReloadMCP, onCreateBinding, onUpdateBinding, onDeleteBinding, onOpenToolsDrawer,
}) => {
  const columns: ColumnsType<AgentMCPBinding> = [
    {
      title: 'MCP Server',
      render: (_: unknown, record: AgentMCPBinding) => {
        const s = mcpServers.find((x) => x.id === record.mcp_server_id);
        return <span>{s ? `${s.name}（${s.code}）` : record.mcp_server_id}</span>;
      },
    },
    {
      title: '工具',
      render: (_: unknown, record: AgentMCPBinding) => {
        const v = record.enabled_tools;
        return v && v.length > 0 ? v.slice(0, 3).map((x) => <Tag key={x}>{x}</Tag>) : <Tag>全部</Tag>;
      },
    },
    {
      title: '状态', width: 90,
      render: (_: unknown, record: AgentMCPBinding) => (
        <Tag color={record.is_active ? 'success' : 'default'}>{record.is_active ? '启用' : '禁用'}</Tag>
      ),
    },
    {
      title: '自动加载', width: 90,
      render: (_: unknown, record: AgentMCPBinding) => (
        <Switch size="small" checked={record.auto_load} checkedChildren="自" unCheckedChildren="手"
          onChange={async () => { await onUpdateBinding(record.id, { auto_load: !record.auto_load }); }} />
      ),
    },
    {
      title: '操作',
      render: (_: unknown, record: AgentMCPBinding) => (
        <ActionGroup size="small">
          <Switch size="small" checked={record.is_active}
            onChange={async () => { await onUpdateBinding(record.id, { is_active: !record.is_active }); }} />
          <Button onClick={() => onOpenToolsDrawer(record)} type="link" size="small" style={{ padding: 0 }}>配置</Button>
          <Popconfirm title="确认解绑该 MCP Server？"
            onConfirm={() => onDeleteBinding(record.id)}>
            <Button danger type="link" size="small" style={{ padding: 0 }}>解绑</Button>
          </Popconfirm>
        </ActionGroup>
      ),
        width: 100,
        fixed: 'left' as const
    },
  ];

  return (
    <Card
      size="small"
      styles={{ body: { padding: 8 } }}
      title={<span><ApiOutlined /> MCP Server 绑定</span>}
      style={{ marginBottom: 8 }}
      extra={editing ? (
        <Button size="small" onClick={onReloadMCP} loading={mcpLoading}>刷新</Button>
      ) : null}
    >
      <div style={{ color: '#999', fontSize: 12, marginBottom: 4 }}>
        说明：不绑定任何 MCP Server 则该 Agent 无法使用 MCP 工具
      </div>
      {!editing && <Tag>请先创建 Agent 后再绑定 MCP Server</Tag>}
      {editing && (
        <>
          <div style={{ marginBottom: 12, display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center' }}>
            <Select placeholder="选择 MCP Server" style={{ flex: 1, minWidth: 260 }}
              value={mcpForm.getFieldValue('mcp_server_id')}
              onChange={(value) => mcpForm.setFieldValue('mcp_server_id', value)}
              options={mcpServers
                .filter((s) => !mcpBindings.some((b) => b.mcp_server_id === s.id))
                .map((s) => ({ value: s.id, label: `${s.name}（${s.code}）` }))}
              showSearch optionFilterProp="label" />
            <Button type="primary" loading={mcpLoading}
              onClick={() => {
                const mcpServerId = mcpForm.getFieldValue('mcp_server_id');
                if (!mcpServerId) return;
                onCreateBinding(mcpServerId);
              }}>
              绑定
            </Button>
          </div>
          <Table<AgentMCPBinding>
            dataSource={mcpBindings} rowKey="id" loading={mcpLoading}
            size="small" pagination={false} scroll={{ x: 520 }}
            columns={columns}
          />
        </>
      )}
    </Card>
  );
};

export default MCPServerBindingCard;