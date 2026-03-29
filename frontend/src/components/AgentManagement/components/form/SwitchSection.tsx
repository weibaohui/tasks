/**
 * SwitchSection - 开关配置卡片
 */
import React from 'react';
import { Card, Form, Space, Switch } from 'antd';
import type { Agent } from '../../../../types/agent';

interface SwitchSectionProps {
  editing: Agent | null;
  screens: Record<string, boolean>;
  handlePatchSection: (section: string, fields: Record<string, unknown>) => Promise<void>;
}

export const SwitchSection: React.FC<SwitchSectionProps> = ({
  editing, screens, handlePatchSection,
}) => (
  <Card size="small" styles={{ body: { padding: 8 } }} title={<span>开关设置</span>}>
    <Space direction={screens.xs ? 'vertical' : 'horizontal'} style={{ display: 'flex' }} align="start">
      <Form.Item label="设为默认" name="is_default" valuePropName="checked" style={{ marginBottom: 0 }}>
        <Switch checkedChildren="默认" unCheckedChildren="非默认"
          onChange={(checked) => { if (editing) handlePatchSection('_switch_default', { is_default: checked }); }} />
      </Form.Item>
      <Form.Item label="启用" name="is_active" valuePropName="checked" style={{ marginBottom: 0 }}>
        <Switch checkedChildren="启用" unCheckedChildren="停用" disabled={!editing}
          onChange={(checked) => { if (editing) handlePatchSection('_switch_active', { is_active: checked }); }} />
      </Form.Item>
      <Form.Item label="展示思考过程" name="enable_thinking_process" valuePropName="checked" style={{ marginBottom: 0 }}>
        <Switch checkedChildren="开启" unCheckedChildren="关闭"
          onChange={(checked) => { if (editing) handlePatchSection('_switch_thinking', { enable_thinking_process: checked }); }} />
      </Form.Item>
    </Space>
  </Card>
);

export default SwitchSection;
