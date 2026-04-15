/**
 * SkillsToolsTab - 技能工具 Tab
 */
import React from 'react';
import type { FormInstance } from 'antd/es/form';
import type { AgentMCPBinding, MCPServer } from '../../../types/mcp';
import type { BuiltInTool } from '../../../types/task';
import type { Skill } from '../../../api/skillApi';
import type { Agent } from '../../../types/agent';

import { SkillsConfigCard } from '../components/form/SkillsConfigSection';
import { MCPServerBindingCard } from '../components/form/MCPServerBindingSection';
import { ToolsConfigCard } from '../components/form/ToolsConfigSection';
import { QuickEntryCard } from '../components/form/QuickEntryCard';

interface SkillsToolsTabProps {
  form: FormInstance;
  editing: Agent | null;
  editingSections: Record<string, boolean>;
  toggleSectionEdit: (section: string) => void;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
  skillsOptions: Skill[];
  builtInTools: BuiltInTool[];
  mcpServers: MCPServer[];
  mcpBindings: AgentMCPBinding[];
  mcpLoading: boolean;
  mcpForm: FormInstance;
  onReloadMCP: () => Promise<void>;
  onCreateBinding: (mcpServerId: string) => Promise<void>;
  onUpdateBinding: (bindingId: string, fields: Record<string, unknown>) => Promise<void>;
  onDeleteBinding: (bindingId: string) => Promise<void>;
  onOpenToolsDrawer: (binding: AgentMCPBinding) => void;
}

export const SkillsToolsTab: React.FC<SkillsToolsTabProps> = ({
  form, editing, editingSections, toggleSectionEdit, handlePatchSection,
  skillsOptions, builtInTools, mcpServers, mcpBindings, mcpLoading, mcpForm,
  onReloadMCP, onCreateBinding, onUpdateBinding, onDeleteBinding, onOpenToolsDrawer,
}) => {
  return (
    <div style={{ padding: '0 0 4px', overflow: 'auto' }}>
      <div style={{ display: 'flex', gap: 12, marginBottom: 12 }}>
        <QuickEntryCard
          title="Skills 库"
          description="管理全局 Skills 列表"
          onClick={() => window.open('/skills', '_blank')}
        />
        <QuickEntryCard
          title="MCP 服务"
          description="管理 MCP 服务器配置"
          onClick={() => window.open('/mcp', '_blank')}
        />
      </div>
      <SkillsConfigCard
        form={form} editing={editing} editingSections={editingSections}
        toggleSectionEdit={toggleSectionEdit} handlePatchSection={handlePatchSection}
        skillsOptions={skillsOptions}
      />
      <MCPServerBindingCard
        editing={editing} mcpServers={mcpServers} mcpBindings={mcpBindings}
        mcpLoading={mcpLoading} mcpForm={mcpForm}
        onReloadMCP={onReloadMCP} onCreateBinding={onCreateBinding}
        onUpdateBinding={onUpdateBinding} onDeleteBinding={onDeleteBinding}
        onOpenToolsDrawer={onOpenToolsDrawer}
      />
      <ToolsConfigCard
        form={form} editing={editing} editingSections={editingSections}
        toggleSectionEdit={toggleSectionEdit} handlePatchSection={handlePatchSection}
        builtInTools={builtInTools}
      />
    </div>
  );
};

export default SkillsToolsTab;
