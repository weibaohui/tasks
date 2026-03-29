/**
 * ClaudeCodeSandboxCard - Claude Code 沙箱安全卡片
 */
import React from 'react';
import { ApiOutlined } from '@ant-design/icons';
import { Button, Card, Form, Select, Space, Switch } from 'antd';
import type { FormInstance } from 'antd/es/form';
import type { Agent, ClaudeCodeConfig } from '../../../../types/agent';

interface ClaudeCodeSandboxCardProps {
  form: FormInstance;
  editing: Agent | null;
  editingSections: Record<string, boolean>;
  toggleSectionEdit: (section: string) => void;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
}

export const ClaudeCodeSandboxCard: React.FC<ClaudeCodeSandboxCardProps> = ({
  form, editing, editingSections, toggleSectionEdit, handlePatchSection,
}) => {
  const isEditing = editingSections.claudeCodeSandbox;

  const handleSave = () => {
    const config = form.getFieldValue('claude_code_config') as ClaudeCodeConfig || {};
    handlePatchSection('claudeCodeSandbox', { claude_code_config: config });
  };

  return (
    <Card
      size="small"
      styles={{ body: { padding: 8 } }}
      title={<span><ApiOutlined /> 沙箱安全</span>}
      style={{ marginBottom: 8 }}
      extra={
        editing ? (
          isEditing ? (
            <Space>
              <Button size="small" type="primary" onClick={handleSave}>保存</Button>
              <Button size="small" onClick={() => toggleSectionEdit('claudeCodeSandbox')}>取消</Button>
            </Space>
          ) : (
            <Button size="small" onClick={() => toggleSectionEdit('claudeCodeSandbox')}>编辑</Button>
          )
        ) : null
      }
    >
      {!isEditing ? (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
          <div><span style={{ color: '#999' }}>沙箱启用：</span>{form.getFieldValue('claude_code_config')?.sandbox_enabled ? '是' : '否'}</div>
          <div><span style={{ color: '#999' }}>自动批准 Bash：</span>{form.getFieldValue('claude_code_config')?.auto_allow_bash_if_sandboxed ? '是' : '否'}</div>
          <div><span style={{ color: '#999' }}>排除命令：</span>{(form.getFieldValue('claude_code_config')?.excluded_commands || []).join(', ') || '无'}</div>
        </div>
      ) : (
        <div>
          <Form.Item label="启用沙箱" name={['claude_code_config', 'sandbox_enabled']} valuePropName="checked" style={{ marginBottom: 8 }}>
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
          <Form.Item label="沙箱模式下自动批准 Bash" name={['claude_code_config', 'auto_allow_bash_if_sandboxed']} valuePropName="checked" style={{ marginBottom: 8 }}>
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
          <Form.Item label="排除命令（沙箱绕过）" name={['claude_code_config', 'excluded_commands']} style={{ marginBottom: 0 }}>
            <Select mode="tags" placeholder="输入命令名称后回车"
              options={[
                { value: 'git', label: 'git' },
                { value: 'docker', label: 'docker' },
                { value: 'npm', label: 'npm' },
                { value: 'pnpm', label: 'pnpm' },
                { value: 'make', label: 'make' },
              ]} />
          </Form.Item>
        </div>
      )}
    </Card>
  );
};

export default ClaudeCodeSandboxCard;