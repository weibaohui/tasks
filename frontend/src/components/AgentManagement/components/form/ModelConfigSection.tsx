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
  llmProviderOptions: Array<{ value: string; label: string }>;
  llmProvidersLoading: boolean;
}

export const ModelConfigCard: React.FC<ModelConfigCardProps> = ({
  form, editing, editingSections, toggleSectionEdit, handlePatchSection, screens, modelOptions,
  llmProviderOptions, llmProvidersLoading,
}) => {
  const isEditing = !editing || editingSections.modelConfig;
  const agentType = Form.useWatch('agent_type', form) || editing?.agent_type;
  const showClaudeCodeFields = agentType === 'CodingAgent';

  const handleSave = () => {
    const values = form.getFieldsValue(['model', 'llm_provider_id', 'max_tokens', 'temperature', 'max_iterations', 'history_messages']);
    const maxThinkingTokens = form.getFieldValue(['claude_code_config', 'max_thinking_tokens']);
    // 过滤 undefined 值，避免传递给 API
    const filteredValues = Object.fromEntries(
      Object.entries(values).filter(([, v]) => v !== undefined)
    );
    if (typeof maxThinkingTokens === 'number') {
      filteredValues.claude_code_config = { max_thinking_tokens: maxThinkingTokens };
    }
    handlePatchSection('modelConfig', filteredValues);
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
          <div><span style={{ color: '#999' }}>LLM Provider：</span>{llmProviderOptions.find(p => p.value === form.getFieldValue('llm_provider_id'))?.label || '-'}</div>
          <div><span style={{ color: '#999' }}>模型：</span>{form.getFieldValue('model') || '-'}</div>
          <div><span style={{ color: '#999' }}>Max Tokens：</span>{form.getFieldValue('max_tokens')}</div>
          <div><span style={{ color: '#999' }}>Temperature：</span>{form.getFieldValue('temperature')}</div>
          <div><span style={{ color: '#999' }}>最大迭代：</span>{form.getFieldValue('max_iterations')}</div>
          <div><span style={{ color: '#999' }}>历史消息数：</span>{form.getFieldValue('history_messages')}</div>
          {showClaudeCodeFields && (
            <div><span style={{ color: '#999' }}>Claude Code 思考 Token：</span>{(() => { const v = form.getFieldValue(['claude_code_config', 'max_thinking_tokens']); return v === 0 || v == null ? '-' : v; })()}</div>
          )}
        </div>
      ) : (
        <div>
          <Form.Item label="LLM Provider" name="llm_provider_id" rules={[{ required: true, message: '请选择 LLM Provider' }]}>
            <Select
              showSearch
              allowClear
              loading={llmProvidersLoading}
              options={llmProviderOptions}
              placeholder={llmProvidersLoading ? '正在加载 Provider 列表...' : '请选择 LLM Provider'}
              notFoundContent={llmProvidersLoading ? '正在加载...' : '没有可选 Provider'}
            />
          </Form.Item>
          <Form.Item label="模型" name="model">
            <Select showSearch allowClear options={modelOptions}
              placeholder="自动匹配 (留空)"
              notFoundContent="没有可选模型"
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
            {showClaudeCodeFields && (
              <Form.Item label="Claude Code 思考 Token" name={['claude_code_config', 'max_thinking_tokens']}>
                <InputNumber min={0} style={{ width: '100%' }} placeholder="留空使用默认值" />
              </Form.Item>
            )}
          </div>
        </div>
      )}
    </Card>
  );
};

export default ModelConfigCard;
