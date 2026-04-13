/**
 * OpenCodeBasicCard - OpenCode 基本配置卡片
 */
import React from 'react';
import { CodeOutlined } from '@ant-design/icons';
import { Button, Card, Form, Input, InputNumber, Select, Space, Switch } from 'antd';
import type { FormInstance } from 'antd/es/form';
import type { Agent, OpenCodeConfig } from '../../../../types/agent';

interface OpenCodeBasicCardProps {
  form: FormInstance;
  editing: Agent | null;
  editingSections: Record<string, boolean>;
  screens: Record<string, boolean>;
  toggleSectionEdit: (section: string) => void;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
}

export const OpenCodeBasicCard: React.FC<OpenCodeBasicCardProps> = ({
  form, editing, editingSections, screens, toggleSectionEdit, handlePatchSection,
}) => {
  const isEditing = !editing || editingSections.openCodeConfig;

  const agentTypeLabels: Record<string, string> = {
    build: 'Build - 全功能构建',
    plan: 'Plan - 规划模式',
    explore: 'Explore - 探索模式',
    general: 'General - 通用模式',
    compaction: 'Compaction - 精简模式',
  };

  const getAgentTypeLabel = (type?: string) => {
    if (!type) return '-';
    return agentTypeLabels[type] || type;
  };

  const handleSave = () => {
    const config = form.getFieldValue('opencode_config') as OpenCodeConfig || {};
    handlePatchSection('openCodeConfig', { opencode_config: config });
  };

  return (
    <Card
      size="small"
      styles={{ body: { padding: 8 } }}
      title={<span><CodeOutlined /> OpenCode 基本配置</span>}
      style={{ marginBottom: 8 }}
      extra={
        editing ? (
          isEditing ? (
            <Space>
              <Button size="small" type="primary" onClick={handleSave}>保存</Button>
              <Button size="small" onClick={() => toggleSectionEdit('openCodeConfig')}>取消</Button>
            </Space>
          ) : (
            <Button size="small" onClick={() => toggleSectionEdit('openCodeConfig')}>编辑</Button>
          )
        ) : null
      }
    >
      {!isEditing ? (
        <>
          <div style={{ marginBottom: 8 }}>
            <span style={{ color: '#999' }}>系统提示词：</span>
            <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', marginTop: 2, padding: '4px 8px', backgroundColor: '#fafafa', borderRadius: 4, fontSize: 13 }}>
              {form.getFieldValue('opencode_config')?.system_prompt || '-'}
            </div>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 8 }}>
            <div><span style={{ color: '#999' }}>Agent 类型：</span>{getAgentTypeLabel(form.getFieldValue('opencode_config')?.agent_type)}</div>
            <div><span style={{ color: '#999' }}>工作目录：</span>{form.getFieldValue('opencode_config')?.work_dir || '-'}</div>
            <div><span style={{ color: '#999' }}>超时(秒)：</span>{(() => { const v = form.getFieldValue('opencode_config')?.timeout; return v === 0 || v == null ? '-' : v; })()}</div>
            <div><span style={{ color: '#999' }}>模型变体：</span>{form.getFieldValue('opencode_config')?.variant || '-'}</div>
            <div><span style={{ color: '#999' }}>继续会话：</span>{form.getFieldValue('opencode_config')?.continue_conversation === true ? '是' : form.getFieldValue('opencode_config')?.continue_conversation === false ? '否' : '-'}</div>
            <div><span style={{ color: '#999' }}>分叉会话：</span>{form.getFieldValue('opencode_config')?.fork_session === true ? '是' : form.getFieldValue('opencode_config')?.fork_session === false ? '否' : '-'}</div>
            <div><span style={{ color: '#999' }}>显示思考：</span>{form.getFieldValue('opencode_config')?.show_thinking === true ? '是' : form.getFieldValue('opencode_config')?.show_thinking === false ? '否' : '-'}</div>
            <div><span style={{ color: '#999' }}>跳过权限：</span>{form.getFieldValue('opencode_config')?.skip_permissions === true ? '是' : form.getFieldValue('opencode_config')?.skip_permissions === false ? '否' : '-'}</div>
          </div>
        </>
      ) : (
        <div>
          <Form.Item label="系统提示词" name={['opencode_config', 'system_prompt']} style={{ marginBottom: 8 }}>
            <Input.TextArea rows={3} placeholder="设置 OpenCode 的系统提示词" />
          </Form.Item>
          <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 12 }}>
            <Form.Item label="Agent 类型" name={['opencode_config', 'agent_type']}>
              <Select placeholder="选择 Agent 类型"
                options={[
                  { value: 'build', label: 'Build - 全功能构建' },
                  { value: 'plan', label: 'Plan - 规划模式' },
                  { value: 'explore', label: 'Explore - 探索模式' },
                  { value: 'general', label: 'General - 通用模式' },
                  { value: 'compaction', label: 'Compaction - 精简模式' },
                ]} />
            </Form.Item>
            <Form.Item label="工作目录" name={['opencode_config', 'work_dir']}>
              <Input placeholder="留空使用默认目录" />
            </Form.Item>
            <Form.Item label="超时(秒)" name={['opencode_config', 'timeout']}>
              <InputNumber min={1} style={{ width: '100%' }} placeholder="留空使用默认值" />
            </Form.Item>
            <Form.Item label="模型变体" name={['opencode_config', 'variant']}>
              <Input placeholder="如 claude-sonnet-4-20250514" />
            </Form.Item>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 12 }}>
            <Form.Item label="继续会话" name={['opencode_config', 'continue_conversation']} valuePropName="checked">
              <Switch checkedChildren="启用" unCheckedChildren="禁用" />
            </Form.Item>
            <Form.Item label="分叉会话" name={['opencode_config', 'fork_session']} valuePropName="checked">
              <Switch checkedChildren="启用" unCheckedChildren="禁用" />
            </Form.Item>
            <Form.Item label="显示思考" name={['opencode_config', 'show_thinking']} valuePropName="checked">
              <Switch checkedChildren="启用" unCheckedChildren="禁用" />
            </Form.Item>
            <Form.Item label="跳过权限" name={['opencode_config', 'skip_permissions']} valuePropName="checked">
              <Switch checkedChildren="启用" unCheckedChildren="禁用" />
            </Form.Item>
          </div>
        </div>
      )}
    </Card>
  );
};

export default OpenCodeBasicCard;