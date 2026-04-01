/**
 * Hook 配置编辑抽屉
 */
import React, { useEffect } from 'react';
import { Drawer, Form, Input, Select, Switch, Space, Button, Divider, Typography } from 'antd';
import type { HookConfig, CreateHookConfigRequest } from '../../../types/hook';
import { TRIGGER_POINTS, ACTION_TYPES } from '../../../types/hook';

const { Text } = Typography;

interface HookEditDrawerProps {
  open: boolean;
  editing: HookConfig | null;
  saving: boolean;
  onClose: () => void;
  onSubmit: (values: CreateHookConfigRequest) => Promise<void>;
}

interface FormValues {
  name: string;
  trigger_point: string;
  action_type: string;
  action_config: string;
  enabled: boolean;
  priority: number;
}

export const HookEditDrawer: React.FC<HookEditDrawerProps> = ({
  open,
  editing,
  saving,
  onClose,
  onSubmit,
}) => {
  const [form] = Form.useForm<FormValues>();

  useEffect(() => {
    if (open) {
      if (editing) {
        form.setFieldsValue({
          name: editing.name,
          trigger_point: editing.trigger_point,
          action_type: editing.action_type,
          action_config: editing.action_config,
          enabled: editing.enabled,
          priority: editing.priority,
        });
      } else {
        form.setFieldsValue({
          name: '',
          trigger_point: 'start_dispatch',
          action_type: 'trigger_agent',
          action_config: '',
          enabled: true,
          priority: 0,
        });
      }
    }
  }, [open, editing, form]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      await onSubmit(values as CreateHookConfigRequest);
      form.resetFields();
    } catch (err) {
      // 表单验证失败
      console.error(err);
    }
  };

  const handleClose = () => {
    form.resetFields();
    onClose();
  };

  return (
    <Drawer
      title={editing ? '编辑 Hook 配置' : '新建 Hook 配置'}
      placement="right"
      width={500}
      onClose={handleClose}
      open={open}
      extra={
        <Space>
          <Button onClick={handleClose}>取消</Button>
          <Button type="primary" loading={saving} onClick={handleSubmit}>
            {editing ? '更新' : '创建'}
          </Button>
        </Space>
      }
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="name"
          label="名称"
          rules={[{ required: true, message: '请输入名称' }]}
        >
          <Input placeholder="例如：派发时通知" />
        </Form.Item>

        <Form.Item
          name="trigger_point"
          label="触发点"
          rules={[{ required: true, message: '请选择触发点' }]}
        >
          <Select placeholder="选择触发点">
            {TRIGGER_POINTS.map((tp) => (
              <Select.Option key={tp.value} value={tp.value}>
                {tp.label}
              </Select.Option>
            ))}
          </Select>
        </Form.Item>

        <Form.Item
          name="action_type"
          label="动作类型"
          rules={[{ required: true, message: '请选择动作类型' }]}
        >
          <Select placeholder="选择动作类型">
            {ACTION_TYPES.map((at) => (
              <Select.Option key={at.value} value={at.value}>
                {at.label}
              </Select.Option>
            ))}
          </Select>
        </Form.Item>

        <Form.Item
          name="action_config"
          label="动作配置"
          rules={[{ required: true, message: '请输入动作配置' }]}
          extra={
            <div style={{ marginTop: 8 }}>
              <Text type="secondary">触发 Agent 示例配置：</Text>
              <pre style={{ fontSize: 12, background: '#f5f5f5', padding: 8, borderRadius: 4 }}>
{`{
  "agent_code": "coding-agent-01",
  "task_prompt_template": "请处理需求 {{requirement_title}}，描述：{{requirement_description}}"
}`}
              </pre>
              <Text type="secondary">可用变量：</Text>
              <ul style={{ fontSize: 12, color: '#888' }}>
                <li><code>{'{{requirement_id}}'}</code> - 需求 ID</li>
                <li><code>{'{{requirement_title}}'}</code> - 需求标题</li>
                <li><code>{'{{requirement_description}}'}</code> - 需求描述</li>
                <li><code>{'{{requirement_status}}'}</code> - 需求状态</li>
                <li><code>{'{{project_id}}'}</code> - 项目 ID</li>
                <li><code>{'{{project_name}}'}</code> - 项目名称</li>
              </ul>
            </div>
          }
        >
          <Input.TextArea
            placeholder='{"agent_code": "xxx", "task_prompt_template": "..."}'
            rows={6}
            style={{ fontFamily: 'monospace' }}
          />
        </Form.Item>

        <Form.Item name="priority" label="优先级" initialValue={0}>
          <Input type="number" placeholder="数值越大越优先执行" />
        </Form.Item>

        <Form.Item name="enabled" label="启用" valuePropName="checked" initialValue={true}>
          <Switch />
        </Form.Item>
      </Form>

      <Divider />

      <div>
        <Text strong>触发点说明：</Text>
        <ul style={{ fontSize: 13, color: '#666' }}>
          <li><b>start_dispatch</b> - 需求开始派发时</li>
          <li><b>mark_coding</b> - 需求开始编码时</li>
          <li><b>mark_failed</b> - 需求标记为失败时</li>
          <li><b>mark_pr_opened</b> - 需求 PR 已打开时</li>
        </ul>
      </div>
    </Drawer>
  );
};