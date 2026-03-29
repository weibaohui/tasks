/**
 * PersonalityTab - 人格属性 Tab
 */
import React from 'react';
import { FileTextOutlined } from '@ant-design/icons';
import { Button, Card, Form, Input, Space } from 'antd';
import type { FormInstance } from 'antd/es/form';
import type { Agent } from '../../../types/agent';

interface PersonalityTabProps {
  form: FormInstance;
  editing: Agent | null;
  editingSections: Record<string, boolean>;
  savingSections: Record<string, boolean>;
  toggleSectionEdit: (section: string) => void;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
}

interface MDConfig {
  key: string;
  file: string;
  desc: string;
}

const MD_CONFIGS: MDConfig[] = [
  { key: 'identity_content', file: 'IDENTITY.md', desc: '智能伙伴的名字、性格和身份定义' },
  { key: 'soul_content', file: 'SOUL.md', desc: '智能伙伴的信念、风格和行为准则' },
  { key: 'agents_content', file: 'AGENTS.md', desc: '智能伙伴的工作流程和记忆规则' },
  { key: 'user_content', file: 'USER.md', desc: '关于用户的信息和偏好' },
  { key: 'tools_content', file: 'TOOLS.md', desc: '智能伙伴的本地笔记和速查表' },
];

/** MD 配置文件卡片 */
const MDConfigCard: React.FC<{
  config: MDConfig;
  form: FormInstance;
  editing: Agent | null;
  editingSections: Record<string, boolean>;
  savingSections: Record<string, boolean>;
  toggleSectionEdit: (section: string) => void;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
}> = ({ config, form, editing, editingSections, savingSections, toggleSectionEdit, handlePatchSection }) => {
  const isEditing = editingSections[config.key];

  return (
    <Card
      key={config.key}
      size="small"
      styles={{ body: { padding: 8 } }}
      style={{ marginBottom: 8 }}
      title={
        <span>
          <FileTextOutlined style={{ marginRight: 8 }} />
          {config.file}
        </span>
      }
      extra={
        editing ? (
          isEditing ? (
            <Space>
              <Button
                type="link"
                size="small"
                loading={savingSections[config.key]}
                onClick={() => {
                  const content = form.getFieldValue(config.key) as string || '';
                  handlePatchSection(config.key, { [config.key]: content });
                }}
              >
                保存
              </Button>
              <Button type="link" size="small" onClick={() => toggleSectionEdit(config.key)}>
                取消
              </Button>
            </Space>
          ) : (
            <Button type="link" size="small" onClick={() => toggleSectionEdit(config.key)}>
              编辑
            </Button>
          )
        ) : null
      }
    >
      <div style={{ color: '#999', fontSize: 12, marginBottom: isEditing ? 8 : 0 }}>
        {config.desc}
      </div>
      {!editing || !isEditing ? (
        <pre style={{
          margin: 0,
          padding: 4,
          background: '#fafafa',
          borderRadius: 4,
          fontSize: 12,
          fontFamily: 'monospace',
          maxHeight: 150,
          overflow: 'auto',
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
        }}>
          {(form.getFieldValue(config.key) as string) || '(空)'}
        </pre>
      ) : (
        <Form.Item name={config.key} style={{ marginBottom: 0 }}>
          <Input.TextArea
            rows={8}
            style={{ fontFamily: 'monospace', fontSize: 12 }}
          />
        </Form.Item>
      )}
    </Card>
  );
};

/** PersonalityTab 主组件 */
export const PersonalityTab: React.FC<PersonalityTabProps> = ({
  form,
  editing,
  editingSections,
  savingSections,
  toggleSectionEdit,
  handlePatchSection,
}) => {
  return (
    <div style={{ padding: '0 0 4px', overflow: 'auto' }}>
      {MD_CONFIGS.map((config) => (
        <MDConfigCard
          key={config.key}
          config={config}
          form={form}
          editing={editing}
          editingSections={editingSections}
          savingSections={savingSections}
          toggleSectionEdit={toggleSectionEdit}
          handlePatchSection={handlePatchSection}
        />
      ))}
    </div>
  );
};

export default PersonalityTab;
