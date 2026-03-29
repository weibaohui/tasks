/**
 * ToolsConfigCard - 工具配置卡片
 */
import React from 'react';
import { Card, Form, Select, Space, Switch, Tag } from 'antd';
import { ToolOutlined } from '@ant-design/icons';
import type { FormInstance } from 'antd/es/form';
import type { Agent } from '../../../../types/agent';
import type { BuiltInTool } from '../../../../api/taskApi';

interface ToolsConfigCardProps {
  form: FormInstance;
  editing: Agent | null;
  editingSections: Record<string, boolean>;
  toggleSectionEdit: (section: string) => void;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
  builtInTools: BuiltInTool[];
}

export const ToolsConfigCard: React.FC<ToolsConfigCardProps> = ({
  form, editing, editingSections, toggleSectionEdit, handlePatchSection, builtInTools,
}) => {
  const isEditing = editingSections.toolsConfig;

  return (
    <Card
      size="small"
      styles={{ body: { padding: 8 } }}
      title={<span><ToolOutlined /> 工具配置</span>}
      extra={
        editing ? (
          isEditing ? (
            <Space>
              <Switch size="small" checkedChildren="保存" unCheckedChildren="取消" checked={false}
                onChange={() => {
                  const tools = form.getFieldValue('tools_list') as string[] || [];
                  handlePatchSection('toolsConfig', { tools_list: tools });
                }} />
              <Switch size="small" checkedChildren="保存" unCheckedChildren="编辑" checked={true}
                onChange={() => toggleSectionEdit('toolsConfig')} />
            </Space>
          ) : (
            <Switch size="small" checkedChildren="保存" unCheckedChildren="编辑" checked={false}
              onChange={() => toggleSectionEdit('toolsConfig')} />
          )
        ) : null
      }
    >
      {!isEditing ? (
        <div>
          {(form.getFieldValue('tools_list') as string[] || []).length === 0 ? (
            <span style={{ color: '#999' }}>未配置工具（不限）</span>
          ) : (
            <Space wrap>
              {(form.getFieldValue('tools_list') as string[] || []).map((t) => (
                <Tag key={t} color="cyan">{t}</Tag>
              ))}
            </Space>
          )}
        </div>
      ) : (
        <div>
          <Form.Item label="Tools（可多选/自定义）" name="tools_list" style={{ marginBottom: 4 }}>
            <Select mode="tags" placeholder="输入后回车添加"
              options={builtInTools.map((t) => ({
                value: t.name,
                label: t.description ? `${t.name} - ${t.description}` : t.name,
              }))} />
          </Form.Item>
          <div style={{ color: '#999', fontSize: 12 }}>
            说明：绑定 Skills 会自动添加 use_skill，绑定 MCP 会自动添加 use_mcp 和 call_mcp_tool
          </div>
        </div>
      )}
    </Card>
  );
};

export default ToolsConfigCard;
