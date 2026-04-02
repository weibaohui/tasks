/**
 * BareLLMAgentEditDrawer - BareLLM Agent 编辑抽屉（无 Claude Code Tab）
 */
import React from 'react';
import { Button, Drawer, Form, Input, Select, Space, Tabs } from 'antd';
import type { FormInstance } from 'antd/es/form';
import type { Agent } from '../../../types/agent';
import type { AgentFormValues } from '../hooks';
import { BasicInfoTab } from '../tabs/BasicInfoTab';
import { SkillsToolsTab } from '../tabs/SkillsToolsTab';
import { PersonalityTab } from '../tabs/PersonalityTab';

interface BareLLMAgentEditDrawerProps {
  open: boolean;
  editing: Agent | null;
  form: FormInstance<AgentFormValues>;
  screens: { xs: boolean };
  saving: boolean;
  activeTab: string;
  editingSections: Record<string, boolean>;
  savingSections: Record<string, boolean>;
  providersLoading: boolean;
  modelOptions: Array<{ value: string; label: string }>;
  providerOptions: Array<{ value: string; label: string }>;
  llmProviderOptions: Array<{ value: string; label: string }>;
  llmProvidersLoading: boolean;
  skillsOptions: any[];
  builtInTools: any[];
  mcpServers: any[];
  mcpBindings: any[];
  mcpLoading: boolean;
  mcpForm: FormInstance<any>;
  onClose: () => void;
  onSubmit: () => void;
  onTabChange: (tab: string) => void;
  onToggleSectionEdit: (section: string) => void;
  onPatchSection: (section: string, fields: any) => Promise<void>;
  onReloadMCP: () => Promise<void>;
  onCreateBinding: (mcpServerId: string) => Promise<void>;
  onUpdateBinding: (bindingId: string, fields: any) => Promise<void>;
  onDeleteBinding: (bindingId: string) => Promise<void>;
  onOpenToolsDrawer: (binding: any) => void;
}

export const BareLLMAgentEditDrawer: React.FC<BareLLMAgentEditDrawerProps> = ({
  open,
  editing,
  form,
  screens,
  saving,
  activeTab,
  editingSections,
  savingSections,
  providersLoading,
  modelOptions,
  providerOptions,
  llmProviderOptions,
  llmProvidersLoading,
  skillsOptions,
  builtInTools,
  mcpServers,
  mcpBindings,
  mcpLoading,
  mcpForm,
  onClose,
  onSubmit,
  onTabChange,
  onToggleSectionEdit,
  onPatchSection,
  onReloadMCP,
  onCreateBinding,
  onUpdateBinding,
  onDeleteBinding,
  onOpenToolsDrawer,
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
      title={editing ? '编辑个人助理' : '新建个人助理'}
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
              initialValue="BareLLM"
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
                key: 'basic',
                label: '模型配置',
                children: (
                  <BasicInfoTab
                    form={form}
                    editing={editing}
                    editingSections={editingSections}
                    screens={screens}
                    toggleSectionEdit={onToggleSectionEdit}
                    handlePatchSection={onPatchSection}
                    modelOptions={modelOptions}
                    providerOptions={providerOptions}
                    providersLoading={providersLoading}
                    llmProviderOptions={llmProviderOptions}
                    llmProvidersLoading={llmProvidersLoading}
                  />
                ),
              },
              {
                key: 'skills',
                label: '技能工具',
                children: (
                  <SkillsToolsTab
                    form={form}
                    editing={editing}
                    editingSections={editingSections}
                    toggleSectionEdit={onToggleSectionEdit}
                    handlePatchSection={onPatchSection}
                    skillsOptions={skillsOptions}
                    builtInTools={builtInTools}
                    mcpServers={mcpServers}
                    mcpBindings={mcpBindings}
                    mcpLoading={mcpLoading}
                    mcpForm={mcpForm}
                    onReloadMCP={onReloadMCP}
                    onCreateBinding={onCreateBinding}
                    onUpdateBinding={onUpdateBinding}
                    onDeleteBinding={onDeleteBinding}
                    onOpenToolsDrawer={onOpenToolsDrawer}
                  />
                ),
              },
              {
                key: 'personality',
                label: '人格属性',
                children: (
                  <PersonalityTab
                    form={form}
                    editing={editing}
                    editingSections={editingSections}
                    savingSections={savingSections}
                    toggleSectionEdit={onToggleSectionEdit}
                    handlePatchSection={onPatchSection}
                  />
                ),
              },
            ]}
          />
        )}
      </Form>
    </Drawer>
  );
};

export default BareLLMAgentEditDrawer;