/**
 * OpenCodeTab - OpenCode 配置 Tab
 */
import React from 'react';
import { Card } from 'antd';
import type { FormInstance } from 'antd/es/form';
import type { Agent } from '../../../types/agent';

import { OpenCodeBasicCard } from '../components/form/OpenCodeBasicSection';

interface OpenCodeTabProps {
  form: FormInstance;
  editing: Agent | null;
  editingSections: Record<string, boolean>;
  screens: Record<string, boolean>;
  toggleSectionEdit: (section: string) => void;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
  agentType: string;
}

export const OpenCodeTab: React.FC<OpenCodeTabProps> = ({
  form, editing, editingSections, screens, toggleSectionEdit, handlePatchSection, agentType,
}) => {
  if (agentType !== 'OpenCodeAgent') {
    return (
      <Card size="small" styles={{ body: { padding: 8 } }} title="OpenCode 配置">
        <div style={{ color: '#999', textAlign: 'center', padding: '20px 0' }}>
          OpenCode 配置仅适用于 OpenCodeAgent 类型<br />
          <span style={{ color: '#999' }}>请在「基础信息」中修改 Agent 类型</span>
        </div>
      </Card>
    );
  }

  return (
    <>
      <OpenCodeBasicCard
        form={form} editing={editing} editingSections={editingSections}
        screens={screens} toggleSectionEdit={toggleSectionEdit} handlePatchSection={handlePatchSection}
      />
    </>
  );
};

export default OpenCodeTab;