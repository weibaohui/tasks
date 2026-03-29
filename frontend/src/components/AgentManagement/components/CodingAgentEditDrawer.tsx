/**
 * CodingAgentEditDrawer - CodingAgent 编辑抽屉（仅 Claude Code Tab）
 */
import React from 'react';
import { Button, Drawer, Form, Space, Tabs } from 'antd';
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
  open, editing, form, screens, saving, activeTab, editingSections,
  agentType, onClose, onSubmit, onTabChange, onToggleSectionEdit, onPatchSection,
}) => {
  return (
    <Drawer
      title={editing ? '编辑 CodingAgent' : '新建 CodingAgent'}
      placement="right"
      open={open}
      onClose={onClose}
      width={screens.xs ? '100%' : 760}
      styles={{ body: { padding: 0 } }}
      destroyOnClose
      extra={
        !editing ? (
          <Space>
            <Button onClick={onClose}>取消</Button>
            <Button type="primary" onClick={onSubmit} loading={saving}>创建</Button>
          </Space>
        ) : null
      }
    >
      <Form layout="vertical" form={form} onFinish={onSubmit} style={{ height: '100%' }} size="small">
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
      </Form>
    </Drawer>
  );
};

export default CodingAgentEditDrawer;
