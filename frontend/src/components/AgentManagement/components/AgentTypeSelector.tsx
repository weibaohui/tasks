/**
 * AgentTypeSelector - 新建 Agent 第一步：选择类型
 */
import React from 'react';
import { Modal, Card, Typography, Space } from 'antd';
import { RobotOutlined, CodeOutlined } from '@ant-design/icons';

const { Title, Text } = Typography;

export type AgentTypeOption = {
  type: string;
  title: string;
  description: string;
  icon: React.ReactNode;
};

const AGENT_TYPE_OPTIONS: AgentTypeOption[] = [
  {
    type: 'BareLLM',
    title: '个人助理',
    description: '通用对话型 Agent，可绑定 Skills 和 MCP 工具',
    icon: <RobotOutlined style={{ fontSize: 32, color: '#1677ff' }} />,
  },
  {
    type: 'CodingAgent',
    title: 'Claude Code',
    description: '基于 Claude Code CLI 的编程 Agent',
    icon: <CodeOutlined style={{ fontSize: 32, color: '#52c41a' }} />,
  },
  {
    type: 'OpenCodeAgent',
    title: 'OpenCode',
    description: '基于 OpenCode CLI 的编程 Agent',
    icon: <CodeOutlined style={{ fontSize: 32, color: '#722ed1' }} />,
  },
];

interface AgentTypeSelectorProps {
  open: boolean;
  onSelect: (type: string) => void;
  onCancel: () => void;
}

export const AgentTypeSelector: React.FC<AgentTypeSelectorProps> = ({
  open,
  onSelect,
  onCancel,
}) => {
  return (
    <Modal
      title={<Title level={4} style={{ margin: 0 }}>选择 Agent 类型</Title>}
      open={open}
      onCancel={onCancel}
      footer={null}
      width={600}
      centered
    >
      <Space direction="vertical" style={{ width: '100%', marginTop: 16 }} size="middle">
        {AGENT_TYPE_OPTIONS.map((option) => (
          <Card
            key={option.type}
            hoverable
            onClick={() => onSelect(option.type)}
            styles={{ body: { padding: 16 } }}
          >
            <Space size="large" align="start">
              {option.icon}
              <div>
                <Text strong style={{ fontSize: 16, display: 'block' }}>
                  {option.title}
                </Text>
                <Text type="secondary">{option.description}</Text>
              </div>
            </Space>
          </Card>
        ))}
      </Space>
    </Modal>
  );
};

export default AgentTypeSelector;
