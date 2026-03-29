/**
 * BareLLMAgentEditDrawer - BareLLM Agent 编辑抽屉（无 Claude Code Tab）
 */
import React from 'react';
import { Button, Drawer, Form, Space, Tabs } from 'antd';
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
  open, editing, form, screens, saving, activeTab, editingSections, savingSections,
  providersLoading, modelOptions, skillsOptions, builtInTools, mcpServers, mcpBindings,
  mcpLoading, mcpForm, onClose, onSubmit, onTabChange, onToggleSectionEdit, onPatchSection,
  onReloadMCP, onCreateBinding, onUpdateBinding, onDeleteBinding, onOpenToolsDrawer,
}) => {
  return (
    <Drawer
      title={editing ? '编辑 BareLLM Agent' : '新建 BareLLM Agent'}
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
              key: 'basic',
              label: '基础信息',
              children: (
                <BasicInfoTab
                  form={form} editing={editing} editingSections={editingSections}
                  screens={screens} toggleSectionEdit={onToggleSectionEdit}
                  handlePatchSection={onPatchSection} modelOptions={modelOptions}
                  providersLoading={providersLoading}
                />
              ),
            },
            {
              key: 'skills',
              label: '技能工具',
              children: (
                <SkillsToolsTab
                  form={form} editing={editing} editingSections={editingSections}
                  toggleSectionEdit={onToggleSectionEdit} handlePatchSection={onPatchSection}
                  skillsOptions={skillsOptions} builtInTools={builtInTools}
                  mcpServers={mcpServers} mcpBindings={mcpBindings}
                  mcpLoading={mcpLoading} mcpForm={mcpForm}
                  onReloadMCP={onReloadMCP} onCreateBinding={onCreateBinding}
                  onUpdateBinding={onUpdateBinding} onDeleteBinding={onDeleteBinding}
                  onOpenToolsDrawer={onOpenToolsDrawer}
                />
              ),
            },
            {
              key: 'personality',
              label: '人格属性',
              children: (
                <PersonalityTab
                  form={form} editing={editing} editingSections={editingSections}
                  savingSections={savingSections} toggleSectionEdit={onToggleSectionEdit}
                  handlePatchSection={onPatchSection}
                />
              ),
            },
          ]}
        />
      </Form>
    </Drawer>
  );
};

export default BareLLMAgentEditDrawer;
