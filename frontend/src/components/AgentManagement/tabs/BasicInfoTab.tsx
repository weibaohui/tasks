/**
 * BasicInfoTab - 模型配置 Tab
 */
import React from 'react';
import type { FormInstance } from 'antd/es/form';
import type { Agent } from '../../../types/agent';
import { ModelConfigCard } from '../components/form/ModelConfigSection';

interface BasicInfoTabProps {
  form: FormInstance;
  editing: Agent | null;
  editingSections: Record<string, boolean>;
  screens: Record<string, boolean>;
  toggleSectionEdit: (section: string) => void;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
  modelOptions: Array<{ value: string; label: string }>;
  providerOptions: Array<{ value: string; label: string }>;
  providersLoading: boolean;
}

export const BasicInfoTab: React.FC<BasicInfoTabProps> = ({
  form, editing, editingSections, screens,
  toggleSectionEdit, handlePatchSection, modelOptions, providerOptions, providersLoading,
}) => {
  return (
    <div style={{ padding: '0 0 4px', overflow: 'auto' }}>
      <ModelConfigCard
        form={form} editing={editing} editingSections={editingSections}
        toggleSectionEdit={toggleSectionEdit} handlePatchSection={handlePatchSection}
        screens={screens} modelOptions={modelOptions} providerOptions={providerOptions}
        providersLoading={providersLoading}
      />
    </div>
  );
};

export default BasicInfoTab;