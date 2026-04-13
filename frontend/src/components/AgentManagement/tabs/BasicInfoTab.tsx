/**
 * BasicInfoTab - 模型配置 Tab
 */
import React from 'react';
import type { FormInstance } from 'antd/es/form';
import type { Agent } from '../../../types/agent';
import { BasicInfoCard } from '../components/form/BasicInfoSection';
import { ModelConfigCard } from '../components/form/ModelConfigSection';

interface BasicInfoTabProps {
  form: FormInstance;
  editing: Agent | null;
  editingSections: Record<string, boolean>;
  screens: Record<string, boolean>;
  toggleSectionEdit: (section: string) => void;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
  modelOptions: Array<{ value: string; label: string }>;
  llmProviderOptions: Array<{ value: string; label: string }>;
  llmProvidersLoading: boolean;
}

export const BasicInfoTab: React.FC<BasicInfoTabProps> = ({
  form, editing, editingSections, screens,
  toggleSectionEdit, handlePatchSection, modelOptions,
  llmProviderOptions, llmProvidersLoading,
}) => {
  return (
    <div style={{ padding: '0 0 4px', overflow: 'auto' }}>
      <BasicInfoCard
        form={form} editing={editing} editingSections={editingSections}
        toggleSectionEdit={toggleSectionEdit} handlePatchSection={handlePatchSection}
      />
      <ModelConfigCard
        form={form} editing={editing} editingSections={editingSections}
        toggleSectionEdit={toggleSectionEdit} handlePatchSection={handlePatchSection}
        screens={screens} modelOptions={modelOptions}
        llmProviderOptions={llmProviderOptions}
        llmProvidersLoading={llmProvidersLoading}
      />
    </div>
  );
};

export default BasicInfoTab;
