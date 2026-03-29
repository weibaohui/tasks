/**
 * BasicInfoCard - 基本信息卡片
 */
import React from 'react';
import { Card, Form, Input, Select, Space, Switch } from 'antd';
import { EditOutlined } from '@ant-design/icons';
import type { FormInstance } from 'antd/es/form';
import type { Agent } from '../../../../types/agent';

interface BasicInfoCardProps {
  form: FormInstance;
  editing: Agent | null;
  editingSections: Record<string, boolean>;
  toggleSectionEdit: (section: string) => void;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
}

export const BasicInfoCard: React.FC<BasicInfoCardProps> = ({
  form, editing, editingSections, toggleSectionEdit, handlePatchSection,
}) => {
  const isEditing = editingSections.basicInfo;

  return (
    <Card
      size="small"
      styles={{ body: { padding: 8 } }}
      title={<span><EditOutlined /> 基本信息</span>}
      style={{ marginBottom: 8 }}
      extra={
        editing ? (
          isEditing ? (
            <Space>
              <Switch size="small" checkedChildren="保存" unCheckedChildren="取消" checked={false}
                onChange={() => {
                  const values = form.getFieldsValue(['name', 'agent_type', 'description']);
                  if (!values.name) return;
                  handlePatchSection('basicInfo', {
                    name: values.name,
                    agent_type: values.agent_type,
                    description: values.description,
                  });
                }} />
              <Switch size="small" checkedChildren="保存" unCheckedChildren="编辑" checked={true}
                onChange={() => toggleSectionEdit('basicInfo')} />
            </Space>
          ) : (
            <Switch size="small" checkedChildren="保存" unCheckedChildren="编辑" checked={false}
              onChange={() => toggleSectionEdit('basicInfo')} />
          )
        ) : null
      }
    >
      {!isEditing ? (
        <div>
          <div style={{ marginBottom: 8 }}>
            <span style={{ color: '#999', marginRight: 8 }}>名称：</span>
            <span style={{ fontWeight: 500 }}>{form.getFieldValue('name') || '-'}</span>
          </div>
          <div style={{ marginBottom: 8 }}>
            <span style={{ color: '#999', marginRight: 8 }}>类型：</span>
            <span>{form.getFieldValue('agent_type') || 'BareLLM'}</span>
          </div>
          <div>
            <span style={{ color: '#999', marginRight: 8 }}>描述：</span>
            <span>{form.getFieldValue('description') || '-'}</span>
          </div>
        </div>
      ) : (
        <div>
          <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]} style={{ marginBottom: 8 }}>
            <Input placeholder="Agent 名称" />
          </Form.Item>
          <Form.Item label="类型" name="agent_type" style={{ marginBottom: 8 }}>
            <Select placeholder="选择 Agent 类型"
              options={[
                { value: 'BareLLM', label: '个人助理' },
                { value: 'CodingAgent', label: 'CodingAgent - 编程 Agent' },
              ]} />
          </Form.Item>
          <Form.Item label="描述" name="description" style={{ marginBottom: 0 }}>
            <Input.TextArea rows={2} placeholder="Agent 描述" />
          </Form.Item>
        </div>
      )}
    </Card>
  );
};

export default BasicInfoCard;
