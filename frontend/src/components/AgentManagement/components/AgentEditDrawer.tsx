/**
 * AgentEditDrawer - 统一的 Agent 编辑抽屉
 */
import React from 'react';
import { Button, Drawer, Form, Space, Tabs } from 'antd';
import type { FormInstance } from 'antd/es/form';
import type { Agent } from '../../../types/agent';
import type { AgentFormValues } from '../hooks';
import { BasicInfoTab } from '../tabs/BasicInfoTab';
import { SkillsToolsTab } from '../tabs/SkillsToolsTab';
import { PersonalityTab } from '../tabs/PersonalityTab';
import { ClaudeCodeTab } from '../tabs/ClaudeCodeTab';

interface AgentEditDrawerProps {
  open: boolean;
  editing: Agent | null;
  form: FormInstance<AgentFormValues>;
  screens: { xs: boolean };
  saving: boolean;
  activeTab: string;
  editingSections: Record<string, boolean>;
  savingSections: Record<string, boolean>;
  modelOptions: Array<{ value: string; label: string }>;
  claudeCodeModelOptions: Array<{ value: string; label: string }>;
  llmProviderOptions: Array<{ value: string; label: string }>;
  llmProvidersLoading: boolean;
  skillsOptions: any[];
  builtInTools: any[];
  mcpServers: any[];
  mcpBindings: any[];
  mcpLoading: boolean;
  mcpForm: FormInstance<any>;
  onClose: () => void;
  onSubmit: () => Promise<void>;
  onTabChange: (tab: string) => void;
  onToggleSectionEdit: (section: string) => void;
  onPatchSection: (section: string, fields: any) => Promise<void>;
  onReloadMCP: () => Promise<void>;
  onCreateBinding: (mcpServerId: string) => Promise<void>;
  onUpdateBinding: (bindingId: string, fields: any) => Promise<void>;
  onDeleteBinding: (bindingId: string) => Promise<void>;
  onOpenToolsDrawer: (binding: any) => void;
}

export const AgentEditDrawer: React.FC<AgentEditDrawerProps> = ({
  open, editing, form, screens, saving, activeTab, editingSections, savingSections,
  modelOptions, claudeCodeModelOptions, llmProviderOptions, llmProvidersLoading,
  skillsOptions, builtInTools, mcpServers, mcpBindings, mcpLoading, mcpForm,
  onClose, onSubmit, onTabChange, onToggleSectionEdit, onPatchSection,
  onReloadMCP, onCreateBinding, onUpdateBinding, onDeleteBinding, onOpenToolsDrawer,
}) => {
  const handleCreate = async () => {
    try {
      await form.validateFields();
      await onSubmit();
    } catch (error) {
      // validation or submission errors handled by form
    }
  };

  // 监听表单类型变化，决定显示哪些 Tab
  const agentType = Form.useWatch('agent_type', form) || (editing?.agent_type);
  const isCodingAgent = agentType === 'CodingAgent';
  const isBareLLM = agentType === 'BareLLM';

  // 根据当前 Agent 类型选择对应的模型选项
  const currentModelOptions = isCodingAgent ? claudeCodeModelOptions : modelOptions;

  const tabItems = [
    {
      key: 'basic',
      label: '基本信息',
      children: (
        <BasicInfoTab
          form={form}
          editing={editing}
          editingSections={editingSections}
          screens={screens}
          toggleSectionEdit={onToggleSectionEdit}
          handlePatchSection={onPatchSection}
          modelOptions={currentModelOptions}
          llmProviderOptions={llmProviderOptions}
          llmProvidersLoading={llmProvidersLoading}
        />
      ),
    },
  ];

  if (isBareLLM) {
    tabItems.push(
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
      }
    );
  } else if (isCodingAgent) {
    tabItems.push(
      {
        key: 'claudecode',
        label: 'Claude Code 配置',
        children: (
          <ClaudeCodeTab
            form={form}
            editing={editing}
            editingSections={editingSections}
            screens={screens}
            toggleSectionEdit={onToggleSectionEdit}
            handlePatchSection={onPatchSection}
            agentType={agentType}
          />
        ),
      }
    );
  }

  return (
    <Drawer
      title={editing ? '编辑 Agent' : '新建 Agent'}
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
        <Tabs
          activeKey={activeTab}
          onChange={onTabChange}
          tabBarStyle={{ padding: '0 12px', margin: 0 }}
          items={tabItems}
        />
      </Form>
    </Drawer>
  );
};
