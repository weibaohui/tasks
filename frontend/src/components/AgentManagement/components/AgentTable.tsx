/**
 * AgentTable - Agent 卡片组件
 */
import React, { useState } from 'react';
import { Button, Card, Flex, Input, Switch, Tag, Typography } from 'antd';
import { CheckOutlined, CloseOutlined, DeleteOutlined, EditOutlined } from '@ant-design/icons';
import type { Agent } from '../../../types/agent';

const { Text } = Typography;

interface AgentTableProps {
  items: Agent[];
  loading: boolean;
  screens: Record<string, boolean>;
  onEdit: (agent: Agent) => void;
  onDelete: (id: string) => void;
  onSetDefault: (agent: Agent) => void;
  onToggleThinking: (agent: Agent, enabled: boolean) => void;
  onUpdateAgent: (id: string, fields: { name?: string; description?: string }) => Promise<void>;
}

export const AgentTable: React.FC<AgentTableProps> = ({
  items,
  loading,
  onEdit,
  onDelete,
  onSetDefault,
  onToggleThinking,
  onUpdateAgent,
}) => {
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState('');
  const [editDesc, setEditDesc] = useState('');
  const [saving, setSaving] = useState(false);

  const startEdit = (record: Agent) => {
    setEditingId(record.id);
    setEditName(record.name || '');
    setEditDesc(record.description || '');
  };

  const cancelEdit = () => {
    setEditingId(null);
    setEditName('');
    setEditDesc('');
  };

  const saveEdit = async () => {
    if (!editingId) return;
    setSaving(true);
    try {
      await onUpdateAgent(editingId, { name: editName, description: editDesc });
      setEditingId(null);
    } finally {
      setSaving(false);
    }
  };

  const typeMap: Record<string, { label: string; color: string }> = {
    BareLLM: { label: '个人助理', color: 'default' },
    CodingAgent: { label: '编程', color: 'blue' },
  };

  const cardStyle: React.CSSProperties = {
    width: 340,
    borderRadius: 12,
    boxShadow: '0 2px 12px rgba(0,0,0,0.08)',
    transition: 'all 0.2s ease',
  };

  const renderAgentCard = (agent: Agent) => {
    const isEditing = editingId === agent.id;
    const typeInfo = typeMap[agent.agent_type] || { label: agent.agent_type || 'BareLLM', color: 'default' };

    return (
      <Card
        key={agent.id}
        style={cardStyle}
        styles={{ body: { padding: 20 } }}
        hoverable
      >
        <Flex vertical gap={12}>
          {/* 名称和类型 */}
          <Flex align="center" justify="space-between">
            {isEditing ? (
              <Input
                style={{ flex: 1 }}
                size="small"
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
                placeholder="名称"
              />
            ) : (
              <Flex align="center" gap={8}>
                <Text strong style={{ fontSize: 17 }}>{agent.name}</Text>
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => startEdit(agent)}
                  style={{ color: '#999' }}
                />
              </Flex>
            )}
            <Tag color={typeInfo.color} style={{ marginLeft: 8 }}>{typeInfo.label}</Tag>
          </Flex>

          {/* 描述 */}
          {isEditing ? (
            <Input.TextArea
              value={editDesc}
              onChange={(e) => setEditDesc(e.target.value)}
              placeholder="描述（可选）"
              rows={2}
              style={{ width: '100%' }}
            />
          ) : (
            <Text type="secondary" style={{ fontSize: 13, lineHeight: 1.6 }}>
              {agent.description || '暂无描述'}
            </Text>
          )}

          {/* 状态标签 */}
          <Flex gap={8} align="center">
            <Text type="secondary" style={{ fontSize: 12 }}>思考</Text>
            <Switch
              size="small"
              checked={agent.enable_thinking_process}
              onChange={(checked) => onToggleThinking(agent, checked)}
            />
            {agent.is_default && <Tag color="gold">默认</Tag>}
            <Tag color={agent.is_active ? 'success' : 'default'}>
              {agent.is_active ? '启用' : '停用'}
            </Tag>
          </Flex>

          {/* 编辑操作按钮 */}
          {isEditing && (
            <Flex gap={8}>
              <Button
                type="primary"
                size="small"
                icon={<CheckOutlined />}
                onClick={saveEdit}
                loading={saving}
              >
                保存
              </Button>
              <Button size="small" icon={<CloseOutlined />} onClick={cancelEdit}>
                取消
              </Button>
            </Flex>
          )}

          {/* 底部操作栏 */}
          {!isEditing && (
            <Flex gap={4} justify="flex-end" style={{ marginTop: 8, paddingTop: 12, borderTop: '1px solid #f0f0f0' }}>
              <Button size="small" icon={<EditOutlined />} onClick={() => onEdit(agent)}>
                编辑
              </Button>
              {!agent.is_default && (
                <Button size="small" onClick={() => onSetDefault(agent)}>
                  设为默认
                </Button>
              )}
              <Button
                size="small"
                danger
                icon={<DeleteOutlined />}
                onClick={() => onDelete(agent.id)}
              >
                删除
              </Button>
            </Flex>
          )}
        </Flex>
      </Card>
    );
  };

  if (loading) {
    return (
      <Flex justify="center" align="center" style={{ minHeight: 200 }}>
        <Text type="secondary">加载中...</Text>
      </Flex>
    );
  }

  if (items.length === 0) {
    return (
      <Flex justify="center" align="center" style={{ minHeight: 200 }}>
        <Text type="secondary">暂无 Agent，点击右上角新建</Text>
      </Flex>
    );
  }

  return (
    <Flex gap="large" wrap style={{ padding: '8px 0' }}>
      {items.map(renderAgentCard)}
    </Flex>
  );
};

export default AgentTable;