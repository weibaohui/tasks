/**
 * ClaudeCodeBasicCard - Claude Code 基本配置卡片
 */
import React from 'react';
import { ApiOutlined } from '@ant-design/icons';
import { Button, Card, Form, Input, InputNumber, Select, Space, Switch } from 'antd';
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
  const isEditing = !editing || editingSections.claudeCodeConfig;

  const permissionModeLabels: Record<string, string> = {
    default: 'Default - 标准处理',
    acceptEdits: 'AcceptEdits - 自动接受编辑',
    plan: 'Plan - 计划模式',
    bypassPermissions: 'Bypass - 绕过权限',
  };

  const getPermissionModeLabel = (mode?: string) => {
    if (!mode) return '-';
    return permissionModeLabels[mode] || mode;
  };

  const handleSave = () => {
    const config = form.getFieldValue('claude_code_config') as ClaudeCodeConfig || {};
    handlePatchSection('claudeCodeConfig', { claude_code_config: config });
  };

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
              <Button size="small" type="primary" onClick={handleSave}>保存</Button>
              <Button size="small" onClick={() => toggleSectionEdit('claudeCodeConfig')}>取消</Button>
            </Space>
          ) : (
            <Button size="small" onClick={() => toggleSectionEdit('claudeCodeConfig')}>编辑</Button>
          )
        ) : null
      }
    >
      {!isEditing ? (
        <>
          <div style={{ marginBottom: 8 }}>
            <span style={{ color: '#999' }}>系统提示词：</span>
            <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', marginTop: 2, padding: '4px 8px', backgroundColor: '#fafafa', borderRadius: 4, fontSize: 13 }}>
              {form.getFieldValue('claude_code_config')?.system_prompt || '-'}
            </div>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 8 }}>
            <div><span style={{ color: '#999' }}>权限模式：</span>{getPermissionModeLabel(form.getFieldValue('claude_code_config')?.permission_mode)}</div>
            <div><span style={{ color: '#999' }}>恢复会话：</span>{form.getFieldValue('claude_code_config')?.resume === true ? '是' : form.getFieldValue('claude_code_config')?.resume === false ? '否' : '-'}</div>
            <div><span style={{ color: '#999' }}>工作目录：</span>{form.getFieldValue('claude_code_config')?.cwd ?? '-'}</div>
            <div><span style={{ color: '#999' }}>思考 Token：</span>{(() => { const v = form.getFieldValue('claude_code_config')?.max_thinking_tokens; return v === 0 || v == null ? '-' : v; })()}</div>
            <div><span style={{ color: '#999' }}>超时(秒)：</span>{(() => { const v = form.getFieldValue('claude_code_config')?.timeout; return v === 0 || v == null ? '-' : v; })()}</div>
          </div>
        </>
      ) : (
        <div>
          <Form.Item label="系统提示词" name={['claude_code_config', 'system_prompt']} style={{ marginBottom: 8 }}>
            <Input.TextArea rows={3} placeholder="设置 Claude Code 的系统提示词" />
          </Form.Item>
          <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 12 }}>
            <Form.Item label="权限模式" name={['claude_code_config', 'permission_mode']}>
              <Select placeholder="选择权限模式"
                options={[
                  { value: 'default', label: 'Default - 标准处理' },
                  { value: 'acceptEdits', label: 'AcceptEdits - 自动接受编辑' },
                  { value: 'plan', label: 'Plan - 计划模式' },
                  { value: 'bypassPermissions', label: 'Bypass - 绕过权限' },
                ]} />
            </Form.Item>
            <Form.Item label="工作目录" name={['claude_code_config', 'cwd']}>
              <Input placeholder="留空使用默认目录" />
            </Form.Item>
            <Form.Item label="思考 Token" name={['claude_code_config', 'max_thinking_tokens']}>
              <InputNumber min={0} style={{ width: '100%' }} placeholder="留空使用默认值" />
            </Form.Item>
            <Form.Item label="超时(秒)" name={['claude_code_config', 'timeout']}>
              <InputNumber min={1} style={{ width: '100%' }} placeholder="留空使用默认值" />
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
