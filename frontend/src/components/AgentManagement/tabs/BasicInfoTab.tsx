/**
 * BasicInfoTab - 基础信息 Tab
 */
import React from 'react';
import type { FormInstance } from 'antd/es/form';
import type { Agent } from '../../../types/agent';

import { BasicInfoCard } from '../components/form/BasicInfoSection';
import { ModelConfigCard } from '../components/form/ModelConfigSection';
import { SwitchSection } from '../components/form/SwitchSection';

interface BasicInfoTabProps {
  form: FormInstance;
  editing: Agent | null;
  editingSections: Record<string, boolean>;
  screens: Record<string, boolean>;
  toggleSectionEdit: (section: string) => void;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
  modelOptions: Array<{ value: string; label: string }>;
  providersLoading: boolean;
}

export const BasicInfoTab: React.FC<BasicInfoTabProps> = ({
  form, editing, editingSections, screens,
  toggleSectionEdit, handlePatchSection, modelOptions, providersLoading,
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
        screens={screens} modelOptions={modelOptions} providersLoading={providersLoading}
      />
      <SwitchSection editing={editing} screens={screens} handlePatchSection={handlePatchSection} />
    </div>
  );
};

export default BasicInfoTab;
