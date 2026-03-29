/**
 * MCP Tools Drawer - MCP 工具配置抽屉
 */
import React from 'react';
import { Button, Drawer, Form, Select, Space, Switch } from 'antd';
import type { FormInstance } from 'antd/es/form';
import type { AgentMCPBinding, MCPTool } from '../../../types/mcp';

interface MCPToolsDrawerProps {
  open: boolean;
  editingBinding: AgentMCPBinding | null;
  toolsDrawerLoading: boolean;
  toolsForServer: MCPTool[];
  toolsForm: FormInstance;
  screens: Record<string, boolean>;
  onClose: () => void;
  onSave: (enabledTools: string[]) => Promise<void>;
}

export const MCPToolsDrawer: React.FC<MCPToolsDrawerProps> = ({
  open,
  editingBinding,
  toolsDrawerLoading,
  toolsForServer,
  toolsForm,
  screens,
  onClose,
  onSave,
}) => {
  const handleClose = () => {
    onClose();
    toolsForm.resetFields();
  };

  const handleSave = () => {
    toolsForm.submit();
  };

  return (
    <Drawer
      title="配置 MCP 工具"
      placement="right"
      open={open}
      onClose={handleClose}
      width={screens.xs ? '100%' : 520}
      destroyOnClose
      extra={
        <Space>
          <Button onClick={handleClose}>
            取消
          </Button>
          <Button
            type="primary"
            loading={toolsDrawerLoading}
            onClick={handleSave}
            disabled={!editingBinding}
          >
            保存
          </Button>
        </Space>
      }
    >
      <Form
        form={toolsForm}
        layout="vertical"
        onFinish={async (values) => {
          if (!editingBinding) return;
          const enabled = values.all_tools ? [] : (values.enabled_tools || []);
          await onSave(enabled);
        }}
      >
        <Form.Item name="all_tools" valuePropName="checked" label="启用全部工具">
          <Switch checkedChildren="全部" unCheckedChildren="选择" />
        </Form.Item>
        <Form.Item shouldUpdate noStyle>
          {() => {
            const all = Boolean(toolsForm.getFieldValue('all_tools'));
            return (
              <Form.Item
                name="enabled_tools"
                label="选择启用工具"
                hidden={all}
                rules={all ? undefined : [{ required: true, message: '请选择至少一个工具，或开启"全部工具"' }]}
              >
                <Select
                  mode="multiple"
                  placeholder="选择工具"
                  loading={toolsDrawerLoading}
                  options={toolsForServer.map((t) => ({ value: t.name, label: t.name }))}
                  allowClear
                />
              </Form.Item>
            );
          }}
        </Form.Item>
      </Form>
    </Drawer>
  );
};

export default MCPToolsDrawer;
