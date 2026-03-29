/**
 * ClaudeCodeBasicCard - Claude Code 基本配置卡片
 */
import React from 'react';
import { ApiOutlined } from '@ant-design/icons';
import { Card, Form, Input, InputNumber, Select, Space, Switch } from 'antd';
import type { FormInstance } from 'antd/es/form';
import type { Agent, ClaudeCodeConfig } from '../../../../types/agent';

interface ClaudeCodeBasicCardProps {
  form: FormInstance;
  editing: Agent | null;
  editingSections: Record<string, boolean>;
  screens: Record<string, boolean>;
  toggleSectionEdit: (section: string) => void;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
}

export const ClaudeCodeBasicCard: React.FC<ClaudeCodeBasicCardProps> = ({
  form, editing, editingSections, screens, toggleSectionEdit, handlePatchSection,
}) => {
  const isEditing = editingSections.claudeCodeConfig;

  return (
    <Card
      size="small"
      styles={{ body: { padding: 8 } }}
      title={<span><ApiOutlined /> Claude Code 基本配置</span>}
      style={{ marginBottom: 8 }}
      extra={
        editing ? (
          isEditing ? (
            <Space>
              <Switch size="small" checkedChildren="保存" unCheckedChildren="取消" checked={false}
                onChange={() => {
                  const config = form.getFieldValue('claude_code_config') as ClaudeCodeConfig || {};
                  handlePatchSection('claudeCodeConfig', { claude_code_config: config });
                }} />
              <Switch size="small" checkedChildren="保存" unCheckedChildren="编辑" checked={true}
                onChange={() => toggleSectionEdit('claudeCodeConfig')} />
            </Space>
          ) : (
            <Switch size="small" checkedChildren="保存" unCheckedChildren="编辑" checked={false}
              onChange={() => toggleSectionEdit('claudeCodeConfig')} />
          )
        ) : null
      }
    >
      {!isEditing ? (
        <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 8 }}>
          <div><span style={{ color: '#999' }}>模型：</span>{form.getFieldValue('claude_code_config')?.model || 'MiniMax-M2.7-highspeed'}</div>
          <div><span style={{ color: '#999' }}>最大思考 Token：</span>{form.getFieldValue('claude_code_config')?.max_thinking_tokens || 8000}</div>
          <div><span style={{ color: '#999' }}>权限模式：</span>{form.getFieldValue('claude_code_config')?.permission_mode || 'default'}</div>
          <div><span style={{ color: '#999' }}>恢复会话：</span>{form.getFieldValue('claude_code_config')?.resume ? '是' : '否'}</div>
          <div><span style={{ color: '#999' }}>最大对话轮次：</span>{form.getFieldValue('claude_code_config')?.max_turns || '无限制'}</div>
          <div><span style={{ color: '#999' }}>工作目录：</span>{form.getFieldValue('claude_code_config')?.cwd || '默认'}</div>
        </div>
      ) : (
        <div>
          <Form.Item label="模型" name={['claude_code_config', 'model']} style={{ marginBottom: 8 }}>
            <Input placeholder="MiniMax-M2.7-highspeed" />
          </Form.Item>
          <Form.Item label="系统提示词" name={['claude_code_config', 'system_prompt']} style={{ marginBottom: 8 }}>
            <Input.TextArea rows={3} placeholder="设置 Claude Code 的系统提示词" />
          </Form.Item>
          <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 12 }}>
            <Form.Item label="最大思考 Token" name={['claude_code_config', 'max_thinking_tokens']}>
              <InputNumber min={0} style={{ width: '100%' }} placeholder="8000" />
            </Form.Item>
            <Form.Item label="权限模式" name={['claude_code_config', 'permission_mode']}>
              <Select placeholder="选择权限模式"
                options={[
                  { value: 'default', label: 'Default - 标准处理' },
                  { value: 'acceptEdits', label: 'AcceptEdits - 自动接受编辑' },
                  { value: 'plan', label: 'Plan - 计划模式' },
                  { value: 'bypassPermissions', label: 'Bypass - 绕过权限' },
                ]} />
            </Form.Item>
            <Form.Item label="最大对话轮次" name={['claude_code_config', 'max_turns']}>
              <InputNumber min={0} style={{ width: '100%' }} placeholder="0 表示无限制" />
            </Form.Item>
            <Form.Item label="工作目录" name={['claude_code_config', 'cwd']}>
              <Input placeholder="留空使用默认目录" />
            </Form.Item>
          </div>
          <Form.Item label="恢复会话" name={['claude_code_config', 'resume']} valuePropName="checked" style={{ marginBottom: 0 }}>
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </div>
      )}
    </Card>
  );
};

export default ClaudeCodeBasicCard;
