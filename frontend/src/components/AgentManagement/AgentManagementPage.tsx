/**
 * Agent 管理页面
 * 支持 Agent 的新增、编辑、删除、启用/停用
 */
import React, { useMemo, useState } from 'react';
import { Button, Card, Form, Segmented, Space, Typography } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useAgentManagement, AgentFormValues } from './hooks';
import { AgentTable } from './components/AgentTable';
import { AgentEditDrawer } from './components/AgentEditDrawer';
import { AgentTypeSelector } from './components/AgentTypeSelector';

const { Title } = Typography;

type AgentGroup = 'all' | 'assistant' | 'coding';

export const AgentManagementPage: React.FC = () => {
  const [form] = Form.useForm<AgentFormValues>();
  const [mcpForm] = Form.useForm<{ mcp_server_id: string; is_active: boolean; auto_load: boolean }>();
  const [toolsForm] = Form.useForm<{ all_tools: boolean; enabled_tools: string[] }>();
  const [activeGroup, setActiveGroup] = useState<AgentGroup>('all');

  const {
    items, loading, saving, open, editing, createTypeOpen, providersLoading,
    modelOptions, claudeCodeModelOptions, llmProviderOptions, activeTab, mcpLoading, mcpServers, mcpBindings,
    builtInTools, skillsOptions, editingSections, savingSections,
    fetchList, openEditor, closeEditor, handleDelete, handlePatchSection,
    handleSetDefault, handleToggleThinking, handleUpdateAgent, handleSubmit,
    setActiveTab, toggleSectionEdit, reloadMCP, handleCreateBinding,
    handleUpdateBinding, handleDeleteBinding, handleOpenToolsDrawer,
    startCreateFlow, selectCreateType, cancelCreateType,
    mcpForm: mcpFormInstance,
  } = useAgentManagement({ form, mcpForm, toolsForm });

  const handleTabChange = (tab: string) => {
    setActiveTab(tab as 'basic' | 'skills' | 'personality' | 'claudecode' | 'opencode');
  };

  const groupedItems = useMemo(() => {
    if (activeGroup === 'all') return items;
    if (activeGroup === 'assistant') return items.filter((a) => a.agent_type === 'BareLLM');
    if (activeGroup === 'coding') return items.filter((a) => a.agent_type === 'CodingAgent' || a.agent_type === 'OpenCodeAgent');
    return items;
  }, [items, activeGroup]);

  return (
    <div style={{ padding: 0 }}>
      <Card
        title={<Title level={3} style={{ margin: 0 }}>Agent 工坊</Title>}
        extra={
          <Space>
            <Button onClick={fetchList}>刷新</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={startCreateFlow}>
              新建 Agent
            </Button>
          </Space>
        }
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <Segmented
            value={activeGroup}
            onChange={(v) => setActiveGroup(v as AgentGroup)}
            options={[
              { label: '全部', value: 'all' },
              { label: '个人助理', value: 'assistant' },
              { label: '编程 Agent', value: 'coding' },
            ]}
          />
          <AgentTable
            items={groupedItems}
            loading={loading}
            screens={{ xs: false }}
            onEdit={openEditor}
            onDelete={handleDelete}
            onSetDefault={handleSetDefault}
            onToggleThinking={handleToggleThinking}
            onUpdateAgent={handleUpdateAgent}
          />
        </Space>
      </Card>

      <AgentTypeSelector
        open={createTypeOpen}
        onSelect={selectCreateType}
        onCancel={cancelCreateType}
      />

      <AgentEditDrawer
        open={open}
        editing={editing}
        form={form}
        screens={{ xs: false }}
        saving={saving}
        activeTab={activeTab}
        editingSections={editingSections}
        savingSections={savingSections}
        modelOptions={modelOptions}
        claudeCodeModelOptions={claudeCodeModelOptions}
        llmProviderOptions={llmProviderOptions}
        llmProvidersLoading={providersLoading}
        skillsOptions={skillsOptions}
        builtInTools={builtInTools}
        mcpServers={mcpServers}
        mcpBindings={mcpBindings}
        mcpLoading={mcpLoading}
        mcpForm={mcpFormInstance}
        onClose={closeEditor}
        onSubmit={handleSubmit}
        onTabChange={handleTabChange}
        onToggleSectionEdit={toggleSectionEdit}
        onPatchSection={handlePatchSection}
        onReloadMCP={reloadMCP}
        onCreateBinding={handleCreateBinding}
        onUpdateBinding={handleUpdateBinding}
        onDeleteBinding={handleDeleteBinding}
        onOpenToolsDrawer={handleOpenToolsDrawer}
      />
    </div>
  );
};

export default AgentManagementPage;
