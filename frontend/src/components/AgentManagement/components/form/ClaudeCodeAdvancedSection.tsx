/**
 * ClaudeCodeAdvancedCard - Claude Code 高级设置卡片
 */
import React from 'react';
import { ApiOutlined } from '@ant-design/icons';
import { Button, Card, Form, Input, InputNumber, Space, Switch } from 'antd';
import type { FormInstance } from 'antd/es/form';
import type { Agent, ClaudeCodeConfig } from '../../../../types/agent';

interface ClaudeCodeAdvancedCardProps {
  form: FormInstance;
  editing: Agent | null;
  editingSections: Record<string, boolean>;
  screens: Record<string, boolean>;
  toggleSectionEdit: (section: string) => void;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
}

export const ClaudeCodeAdvancedCard: React.FC<ClaudeCodeAdvancedCardProps> = ({
  form, editing, editingSections, screens, toggleSectionEdit, handlePatchSection,
}) => {
  const isEditing = editingSections.claudeCodeAdvanced;

  const handleSave = () => {
    const config = form.getFieldValue('claude_code_config') as ClaudeCodeConfig || {};
    handlePatchSection('claudeCodeAdvanced', { claude_code_config: config });
  };

  return (
    <Card
      size="small"
      styles={{ body: { padding: 8 } }}
      title={<span><ApiOutlined /> 高级设置</span>}
      style={{ marginBottom: 8 }}
      extra={
        editing ? (
          isEditing ? (
            <Space>
              <Button size="small" type="primary" onClick={handleSave}>保存</Button>
              <Button size="small" onClick={() => toggleSectionEdit('claudeCodeAdvanced')}>取消</Button>
            </Space>
          ) : (
            <Button size="small" onClick={() => toggleSectionEdit('claudeCodeAdvanced')}>编辑</Button>
          )
        ) : null
      }
    >
      {!isEditing ? (
        <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 8 }}>
          <div><span style={{ color: '#999' }}>备用模型：</span>{form.getFieldValue('claude_code_config')?.fallback_model || '无'}</div>
          <div><span style={{ color: '#999' }}>文件检查点：</span>{form.getFieldValue('claude_code_config')?.file_checkpointing ? '是' : '否'}</div>
          <div><span style={{ color: '#999' }}>继续会话：</span>{form.getFieldValue('claude_code_config')?.continue_conversation ? '是' : '否'}</div>
          <div><span style={{ color: '#999' }}>Fork 会话：</span>{form.getFieldValue('claude_code_config')?.fork_session ? '是' : '否'}</div>
          <div><span style={{ color: '#999' }}>追加提示词：</span>{form.getFieldValue('claude_code_config')?.append_system_prompt ? '有' : '无'}</div>
          <div><span style={{ color: '#999' }}>CLI 路径：</span>{form.getFieldValue('claude_code_config')?.cli_path || '默认'}</div>
          <div><span style={{ color: '#999' }}>最大预算 USD：</span>{form.getFieldValue('claude_code_config')?.max_budget_usd || '无限制'}</div>
          <div><span style={{ color: '#999' }}>部分消息：</span>{form.getFieldValue('claude_code_config')?.include_partial_messages ? '启用' : '禁用'}</div>
        </div>
      ) : (
        <div>
          <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 12 }}>
            <Form.Item label="备用模型" name={['claude_code_config', 'fallback_model']}>
              <Input placeholder="主模型不可用时使用" />
            </Form.Item>
            <Form.Item label="最大预算 (USD)" name={['claude_code_config', 'max_budget_usd']}>
              <InputNumber min={0} step={0.1} style={{ width: '100%' }} placeholder="0 表示无限制" />
            </Form.Item>
            <Form.Item label="CLI 路径" name={['claude_code_config', 'cli_path']}>
              <Input placeholder="留空使用默认路径" />
            </Form.Item>
          </div>
          <Form.Item label="追加系统提示词" name={['claude_code_config', 'append_system_prompt']} style={{ marginBottom: 8 }}>
            <Input.TextArea rows={2} placeholder="在现有系统提示词后追加内容" />
          </Form.Item>
          <Space direction={screens.xs ? 'vertical' : 'horizontal'} style={{ display: 'flex' }} align="start">
            <Form.Item label="文件检查点" name={['claude_code_config', 'file_checkpointing']} valuePropName="checked" style={{ marginBottom: 0 }}>
              <Switch checkedChildren="启用" unCheckedChildren="禁用" />
            </Form.Item>
            <Form.Item label="继续会话" name={['claude_code_config', 'continue_conversation']} valuePropName="checked" style={{ marginBottom: 0 }}>
              <Switch checkedChildren="启用" unCheckedChildren="禁用" />
            </Form.Item>
            <Form.Item label="Fork 会话" name={['claude_code_config', 'fork_session']} valuePropName="checked" style={{ marginBottom: 0 }}>
              <Switch checkedChildren="启用" unCheckedChildren="禁用" />
            </Form.Item>
            <Form.Item label="部分消息" name={['claude_code_config', 'include_partial_messages']} valuePropName="checked" style={{ marginBottom: 0 }}>
              <Switch checkedChildren="启用" unCheckedChildren="禁用" />
            </Form.Item>
          </Space>
        </div>
      )}
    </Card>
  );
};

export default ClaudeCodeAdvancedCard;