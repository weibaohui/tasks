/**
 * ModelConfigCard - 模型配置卡片
 */
import React from 'react';
import { Button, Card, Form, InputNumber, Select, Space } from 'antd';
import { ThunderboltOutlined } from '@ant-design/icons';
import type { FormInstance } from 'antd/es/form';
import type { Agent } from '../../../../types/agent';

interface ModelConfigCardProps {
  form: FormInstance;
  editing: Agent | null;
  editingSections: Record<string, boolean>;
  toggleSectionEdit: (section: string) => void;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
  screens: Record<string, boolean>;
  modelOptions: Array<{ value: string; label: string }>;
  providersLoading: boolean;
}

export const ModelConfigCard: React.FC<ModelConfigCardProps> = ({
  form, editing, editingSections, toggleSectionEdit, handlePatchSection, screens, modelOptions, providersLoading,
}) => {
  const isEditing = editingSections.modelConfig;

  const handleSave = () => {
    const values = form.getFieldsValue(['model', 'max_tokens', 'temperature', 'max_iterations', 'history_messages']);
    if (!values.model) return;
    handlePatchSection('modelConfig', values);
  };

  return (
    <Card
      size="small"
      styles={{ body: { padding: 8 } }}
      title={<span><ThunderboltOutlined /> 模型配置</span>}
      style={{ marginBottom: 8 }}
      extra={
        editing ? (
          isEditing ? (
            <Space>
              <Button size="small" type="primary" onClick={handleSave}>保存</Button>
              <Button size="small" onClick={() => toggleSectionEdit('modelConfig')}>取消</Button>
            </Space>
          ) : (
            <Button size="small" onClick={() => toggleSectionEdit('modelConfig')}>编辑</Button>
          )
        ) : null
      }
    >
      {!isEditing ? (
        <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 8 }}>
          <div><span style={{ color: '#999' }}>模型：</span>{form.getFieldValue('model') || '-'}</div>
          <div><span style={{ color: '#999' }}>Max Tokens：</span>{form.getFieldValue('max_tokens')}</div>
          <div><span style={{ color: '#999' }}>Temperature：</span>{form.getFieldValue('temperature')}</div>
          <div><span style={{ color: '#999' }}>最大迭代：</span>{form.getFieldValue('max_iterations')}</div>
          <div><span style={{ color: '#999' }}>历史消息数：</span>{form.getFieldValue('history_messages')}</div>
        </div>
      ) : (
        <div>
          <Form.Item label="模型" name="model" rules={[{ required: true, message: '请选择模型' }]}>
            <Select showSearch allowClear loading={providersLoading} options={modelOptions}
              placeholder={providersLoading ? '正在加载模型列表...' : '请选择模型'}
              notFoundContent={providersLoading ? '正在加载...' : '没有可选模型'}
              filterOption={(input, option) => {
                const q = input.toLowerCase();
                const v = String(option?.value || '').toLowerCase();
                const l = String(option?.label || '').toLowerCase();
                return v.includes(q) || l.includes(q);
              }} />
          </Form.Item>
          <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 12 }}>
            <Form.Item label="Max Tokens" name="max_tokens">
              <InputNumber min={0} style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item label="Temperature" name="temperature">
              <InputNumber min={0} max={2} step={0.1} style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item label="最大迭代" name="max_iterations">
              <InputNumber min={1} style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item label="历史消息数" name="history_messages">
              <InputNumber min={0} style={{ width: '100%' }} />
            </Form.Item>
          </div>
        </div>
      )}
    </Card>
  );
};

export default ModelConfigCard;