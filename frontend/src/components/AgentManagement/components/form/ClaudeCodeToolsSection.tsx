/**
 * ClaudeCodeToolsCard - Claude Code 工具控制卡片
 */
import React from 'react';
import { Card, Form, Select, Space, Switch, Tag } from 'antd';
import { ToolOutlined } from '@ant-design/icons';
import type { FormInstance } from 'antd/es/form';
import type { Agent, ClaudeCodeConfig } from '../../../../types/agent';

interface ClaudeCodeToolsCardProps {
  form: FormInstance;
  editing: Agent | null;
  editingSections: Record<string, boolean>;
  toggleSectionEdit: (section: string) => void;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
}

export const ClaudeCodeToolsCard: React.FC<ClaudeCodeToolsCardProps> = ({
  form, editing, editingSections, toggleSectionEdit, handlePatchSection,
}) => {
  const isEditing = editingSections.claudeCodeTools;

  return (
    <Card
      size="small"
      styles={{ body: { padding: 8 } }}
      title={<span><ToolOutlined /> 工具控制</span>}
      style={{ marginBottom: 8 }}
      extra={
        editing ? (
          isEditing ? (
            <Space>
              <Switch size="small" checkedChildren="保存" unCheckedChildren="取消" checked={false}
                onChange={() => {
                  const config = form.getFieldValue('claude_code_config') as ClaudeCodeConfig || {};
                  handlePatchSection('claudeCodeTools', { claude_code_config: config });
                }} />
              <Switch size="small" checkedChildren="保存" unCheckedChildren="编辑" checked={true}
                onChange={() => toggleSectionEdit('claudeCodeTools')} />
            </Space>
          ) : (
            <Switch size="small" checkedChildren="保存" unCheckedChildren="编辑" checked={false}
              onChange={() => toggleSectionEdit('claudeCodeTools')} />
          )
        ) : null
      }
    >
      {!isEditing ? (
        <div>
          <div style={{ marginBottom: 8 }}>
            <span style={{ color: '#999' }}>允许的工具：</span>
            {(form.getFieldValue('claude_code_config')?.allowed_tools || []).length === 0 ? (
              <Tag>全部</Tag>
            ) : (
              <Space wrap>
                {form.getFieldValue('claude_code_config')?.allowed_tools?.map((t: string) => (
                  <Tag key={t} color="blue">{t}</Tag>
                ))}
              </Space>
            )}
          </div>
          <div>
            <span style={{ color: '#999' }}>禁止的工具：</span>
            {(form.getFieldValue('claude_code_config')?.disallowed_tools || []).length === 0 ? (
              <Tag>无</Tag>
            ) : (
              <Space wrap>
                {form.getFieldValue('claude_code_config')?.disallowed_tools?.map((t: string) => (
                  <Tag key={t} color="red">{t}</Tag>
                ))}
              </Space>
            )}
          </div>
        </div>
      ) : (
        <div>
          <Form.Item label="允许的工具（留空表示全部允许）" name={['claude_code_config', 'allowed_tools']} style={{ marginBottom: 8 }}>
            <Select mode="tags" placeholder="输入工具名称后回车"
              options={[
                { value: 'Read', label: 'Read - 读取文件' },
                { value: 'Write', label: 'Write - 写入文件' },
                { value: 'Edit', label: 'Edit - 编辑文件' },
                { value: 'Bash', label: 'Bash - 执行命令' },
                { value: 'Glob', label: 'Glob - 文件搜索' },
                { value: 'Grep', label: 'Grep - 内容搜索' },
                { value: 'ToolSearch', label: 'ToolSearch - 工具搜索' },
                { value: 'WebFetch', label: 'WebFetch - 网页获取' },
              ]} />
          </Form.Item>
          <Form.Item label="禁止的工具" name={['claude_code_config', 'disallowed_tools']} style={{ marginBottom: 0 }}>
            <Select mode="tags" placeholder="输入工具名称后回车"
              options={[
                { value: 'Bash', label: 'Bash - 执行命令' },
                { value: 'Write', label: 'Write - 写入文件' },
                { value: 'Edit', label: 'Edit - 编辑文件' },
                { value: 'Delete', label: 'Delete - 删除文件' },
              ]} />
          </Form.Item>
        </div>
      )}
    </Card>
  );
};

export default ClaudeCodeToolsCard;
