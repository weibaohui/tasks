/**
 * Agent 管理页面
 * 支持 Agent 的新增、编辑、删除、启用/停用
 */
import React from 'react';
import { Button, Card, Form, Space, Typography } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useAgentManagement, AgentFormValues } from './hooks';
import { AgentTable } from './components/AgentTable';
import { BareLLMAgentEditDrawer } from './components/BareLLMAgentEditDrawer';
import { CodingAgentEditDrawer } from './components/CodingAgentEditDrawer';

const { Title } = Typography;

export const AgentManagementPage: React.FC = () => {
  const [form] = Form.useForm<AgentFormValues>();
  const [mcpForm] = Form.useForm<{ mcp_server_id: string; is_active: boolean; auto_load: boolean }>();
  const [toolsForm] = Form.useForm<{ all_tools: boolean; enabled_tools: string[] }>();

  const {
    items, loading, saving, open, editing, providersLoading,
    modelOptions, activeTab, mcpLoading, mcpServers, mcpBindings,
    builtInTools, skillsOptions, editingSections, savingSections,
    fetchList, openEditor, closeEditor, handleDelete, handlePatchSection,
    handleSetDefault, handleToggleThinking, handleUpdateAgent, handleSubmit,
    setActiveTab, toggleSectionEdit, reloadMCP, handleCreateBinding,
    handleUpdateBinding, handleDeleteBinding, handleOpenToolsDrawer,
    mcpForm: mcpFormInstance,
  } = useAgentManagement({ form, mcpForm, toolsForm });

  // 新建时使用表单中的类型，默认为 BareLLM
  const currentAgentType = editing?.agent_type || 'BareLLM';
  const isCodingAgent = currentAgentType === 'CodingAgent';

  // 控制显示哪个 Drawer：编辑时根据类型，新建时根据选择的类型
  const showCodingDrawer = open && isCodingAgent;
  const showBareLLMDrawer = open && !isCodingAgent;

  const handleTabChange = (tab: string) => {
    setActiveTab(tab as 'basic' | 'skills' | 'personality' | 'claudecode');
  };

  return (
    <div style={{ padding: 0 }}>
      <Card
        title={<Title level={3} style={{ margin: 0 }}>Agent 管理</Title>}
        extra={
          <Space>
            <Button onClick={fetchList}>刷新</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openEditor(null)}>
              新建 Agent
            </Button>
          </Space>
        }
      >
        <AgentTable
          items={items}
          loading={loading}
          screens={{ xs: false }}
          onEdit={openEditor}
          onDelete={handleDelete}
          onSetDefault={handleSetDefault}
          onToggleThinking={handleToggleThinking}
          onUpdateAgent={handleUpdateAgent}
        />
      </Card>

      {/* CodingAgent 编辑抽屉 */}
      <CodingAgentEditDrawer
        open={showCodingDrawer}
        editing={editing}
        form={form}
        screens={{ xs: false }}
        saving={saving}
        activeTab={activeTab}
        editingSections={editingSections}
        agentType={currentAgentType}
        onClose={closeEditor}
        onSubmit={handleSubmit}
        onTabChange={handleTabChange}
        onToggleSectionEdit={toggleSectionEdit}
        onPatchSection={handlePatchSection}
      />

      {/* BareLLM Agent 编辑抽屉 */}
      <BareLLMAgentEditDrawer
        open={showBareLLMDrawer}
        editing={editing}
        form={form}
        screens={{ xs: false }}
        saving={saving}
        activeTab={activeTab}
        editingSections={editingSections}
        savingSections={savingSections}
        providersLoading={providersLoading}
        modelOptions={modelOptions}
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
