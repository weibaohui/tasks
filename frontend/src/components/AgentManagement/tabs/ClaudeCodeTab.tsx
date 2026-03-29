/**
 * ClaudeCodeTab - Claude Code 配置 Tab
 */
import React from 'react';
import { Card } from 'antd';
import type { FormInstance } from 'antd/es/form';
import type { Agent } from '../../../types/agent';

import { ClaudeCodeBasicCard } from '../components/form/ClaudeCodeBasicSection';
import { ClaudeCodeToolsCard } from '../components/form/ClaudeCodeToolsSection';
import { ClaudeCodeSandboxCard } from '../components/form/ClaudeCodeSandboxSection';
import { ClaudeCodeAdvancedCard } from '../components/form/ClaudeCodeAdvancedSection';

interface ClaudeCodeTabProps {
  form: FormInstance;
  editing: Agent | null;
  editingSections: Record<string, boolean>;
  screens: Record<string, boolean>;
  toggleSectionEdit: (section: string) => void;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
  agentType: string;
}

export const ClaudeCodeTab: React.FC<ClaudeCodeTabProps> = ({
  form, editing, editingSections, screens, toggleSectionEdit, handlePatchSection, agentType,
}) => {
  if (agentType !== 'CodingAgent') {
    return (
      <Card size="small" styles={{ body: { padding: 8 } }} title="Claude Code 配置">
        <div style={{ color: '#999', textAlign: 'center', padding: '20px 0' }}>
          Claude Code 配置仅适用于 CodingAgent 类型<br />
          <span style={{ color: '#999' }}>请在「基础信息」中修改 Agent 类型</span>
        </div>
      </Card>
    );
  }

  return (
    <>
      <ClaudeCodeBasicCard
        form={form} editing={editing} editingSections={editingSections}
        screens={screens} toggleSectionEdit={toggleSectionEdit} handlePatchSection={handlePatchSection}
      />
      <ClaudeCodeToolsCard
        form={form} editing={editing} editingSections={editingSections}
        toggleSectionEdit={toggleSectionEdit} handlePatchSection={handlePatchSection}
      />
      <ClaudeCodeSandboxCard
        form={form} editing={editing} editingSections={editingSections}
        toggleSectionEdit={toggleSectionEdit} handlePatchSection={handlePatchSection}
      />
      <ClaudeCodeAdvancedCard
        form={form} editing={editing} editingSections={editingSections}
        screens={screens} toggleSectionEdit={toggleSectionEdit} handlePatchSection={handlePatchSection}
      />
    </>
  );
};

export default ClaudeCodeTab;
