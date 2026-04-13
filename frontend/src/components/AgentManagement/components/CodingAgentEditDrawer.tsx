/**
 * CodingAgentEditDrawer - CodingAgent 编辑抽屉（仅 Claude Code Tab）
 */
import React from 'react';
import { Button, Drawer, Form, Input, Select, Space, Tabs } from 'antd';
import type { FormInstance } from 'antd/es/form';
import type { Agent } from '../../../types/agent';
import type { AgentFormValues } from '../hooks';
import { ClaudeCodeTab } from '../tabs/ClaudeCodeTab';

interface CodingAgentEditDrawerProps {
  open: boolean;
  editing: Agent | null;
  form: FormInstance<AgentFormValues>;
  screens: { xs: boolean };
  saving: boolean;
  activeTab: string;
  editingSections: Record<string, boolean>;
  agentType: string;
  onClose: () => void;
  onSubmit: () => void;
  onTabChange: (tab: string) => void;
  onToggleSectionEdit: (section: string) => void;
  onPatchSection: (section: string, fields: any) => Promise<void>;
}

export const CodingAgentEditDrawer: React.FC<CodingAgentEditDrawerProps> = ({
  open,
  editing,
  form,
  screens,
  saving,
  activeTab,
  editingSections,
  agentType,
  onClose,
  onSubmit,
  onTabChange,
  onToggleSectionEdit,
  onPatchSection,
}) => {
  const handleCreate = async () => {
    try {
      await form.validateFields(['name', 'agent_type', 'description']);
      onSubmit();
    } catch {
      // validation errors shown by form
    }
  };

  return (
    <Drawer
      title={editing ? '编辑编程 Agent' : '新建编程 Agent'}
      placement="right"
      open={open}
      onClose={onClose}
      width={screens.xs ? '100%' : 760}
      styles={{ body: { padding: 0 } }}
      extra={
        editing ? null : (
          <Space>
            <Button onClick={onClose}>取消</Button>
            <Button type="primary" loading={saving} onClick={handleCreate}>
              创建
            </Button>
          </Space>
        )
      }
    >
      <Form
        layout="vertical"
        form={form}
        style={{ height: '100%' }}
        size="small"
      >
        {!editing ? (
          <div style={{ padding: '24px 16px' }}>
            <Form.Item
              label="名称"
              name="name"
              rules={[{ required: true, message: '请输入名称' }]}
              style={{ marginBottom: 16 }}
            >
              <Input placeholder="Agent 名称" />
            </Form.Item>
            <Form.Item
              label="类型"
              name="agent_type"
              initialValue="CodingAgent"
              rules={[{ required: true, message: '请选择类型' }]}
              style={{ marginBottom: 16 }}
            >
              <Select
                placeholder="选择 Agent 类型"
                options={[
                  { value: 'BareLLM', label: '个人助理' },
                  { value: 'CodingAgent', label: '编程 Agent' },
                ]}
              />
            </Form.Item>
            <Form.Item
              label="描述"
              name="description"
              style={{ marginBottom: 0 }}
            >
              <Input.TextArea rows={3} placeholder="Agent 描述（可选）" />
            </Form.Item>
          </div>
        ) : (
          <Tabs
            activeKey={activeTab}
            onChange={onTabChange}
            tabBarStyle={{ padding: '0 12px', margin: 0 }}
            items={[
              {
                key: 'claudecode',
                label: 'Claude Code',
                children: (
                  <div style={{ padding: '0 0 4px', overflow: 'auto' }}>
                    <ClaudeCodeTab
                      form={form}
                      editing={editing}
                      editingSections={editingSections}
                      screens={screens}
                      toggleSectionEdit={onToggleSectionEdit}
                      handlePatchSection={onPatchSection}
                      agentType={agentType}
                    />
                  </div>
                ),
              },
            ]}
          />
        )}
      </Form>
    </Drawer>
  );
};

export default CodingAgentEditDrawer;
