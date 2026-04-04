/**
 * StateMachine Edit Drawer Component
 */
import React, { useEffect } from 'react';
import { Drawer, Form, Input, Button, Space, message, Alert, Tabs } from 'antd';
import type { StateMachine, CreateStateMachineRequest } from '../../../types/stateMachine';

const DEFAULT_YAML = `name: example_flow
description: 示例状态机流程
initial_state: created

states:
  - id: created
    name: 已创建
    is_final: false
  - id: in_progress
    name: 进行中
    is_final: false
  - id: completed
    name: 已完成
    is_final: true

transitions:
  - from: created
    to: in_progress
    trigger: start
    description: 开始处理

  - from: in_progress
    to: completed
    trigger: complete
    description: 完成处理
`;

interface StateMachineEditDrawerProps {
  open: boolean;
  editing: StateMachine | null;
  saving: boolean;
  onClose: () => void;
  onSubmit: (values: CreateStateMachineRequest) => Promise<void>;
}

export const StateMachineEditDrawer: React.FC<StateMachineEditDrawerProps> = ({
  open,
  editing,
  saving,
  onClose,
  onSubmit,
}) => {
  const [form] = Form.useForm<{ name: string; description: string; config: string }>();
  const [activeTab, setActiveTab] = React.useState('visual');
  const [visualStates, setVisualStates] = React.useState<
    { id: string; name: string; isFinal: boolean }[]
  >([]);
  const [visualTransitions, setVisualTransitions] = React.useState<
    { from: string; to: string; trigger: string; description: string }[]
  >([]);

  useEffect(() => {
    if (open) {
      if (editing) {
        form.setFieldsValue({
          name: editing.name,
          description: editing.description,
          config: JSON.stringify(editing.config, null, 2),
        });
        // Parse existing config to visual format
        try {
          const config = editing.config;
          setVisualStates(
            config.states.map((s) => ({
              id: s.id,
              name: s.name,
              isFinal: s.is_final,
            })),
          );
          setVisualTransitions(
            config.transitions.map((t) => ({
              from: t.from,
              to: t.to,
              trigger: t.trigger,
              description: t.description || '',
            })),
          );
        } catch {
          setVisualStates([]);
          setVisualTransitions([]);
        }
      } else {
        form.setFieldsValue({
          name: '',
          description: '',
          config: DEFAULT_YAML,
        });
        // Reset to default template
        setVisualStates([
          { id: 'created', name: '已创建', isFinal: false },
          { id: 'in_progress', name: '进行中', isFinal: false },
          { id: 'completed', name: '已完成', isFinal: true },
        ]);
        setVisualTransitions([
          { from: 'created', to: 'in_progress', trigger: 'start', description: '开始处理' },
          { from: 'in_progress', to: 'completed', trigger: 'complete', description: '完成处理' },
        ]);
      }
    }
  }, [open, editing, form]);

  const handleYamlChange = (value: string) => {
    try {
      const parsed = JSON.parse(value);
      if (parsed.states) {
        setVisualStates(
          parsed.states.map((s: { id: string; name: string; is_final: boolean }) => ({
            id: s.id,
            name: s.name,
            isFinal: s.is_final,
          })),
        );
      }
      if (parsed.transitions) {
        setVisualTransitions(
          parsed.transitions.map((t: { from: string; to: string; trigger: string; description?: string }) => ({
            from: t.from,
            to: t.to,
            trigger: t.trigger,
            description: t.description || '',
          })),
        );
      }
    } catch {
      // Not JSON, ignore
    }
  };

  const handleVisualStateChange = (
    index: number,
    field: 'id' | 'name' | 'isFinal',
    value: string | boolean,
  ) => {
    setVisualStates((prev) => {
      const updated = [...prev];
      updated[index] = { ...updated[index], [field]: value };
      return updated;
    });
  };

  const handleVisualTransitionChange = (
    index: number,
    field: 'from' | 'to' | 'trigger' | 'description',
    value: string,
  ) => {
    setVisualTransitions((prev) => {
      const updated = [...prev];
      updated[index] = { ...updated[index], [field]: value };
      return updated;
    });
  };

  const addState = () => {
    setVisualStates((prev) => [
      ...prev,
      { id: `state_${prev.length + 1}`, name: `状态${prev.length + 1}`, isFinal: false },
    ]);
  };

  const removeState = (index: number) => {
    setVisualStates((prev) => prev.filter((_, i) => i !== index));
  };

  const addTransition = () => {
    setVisualTransitions((prev) => [
      ...prev,
      { from: visualStates[0]?.id || '', to: visualStates[0]?.id || '', trigger: '', description: '' },
    ]);
  };

  const removeTransition = (index: number) => {
    setVisualTransitions((prev) => prev.filter((_, i) => i !== index));
  };

  const buildYamlFromVisual = (): string => {
    const name = form.getFieldValue('name') || 'unnamed';
    const description = form.getFieldValue('description') || '';

    const config: Record<string, unknown> = {
      name,
      description,
      initial_state: visualStates.find((s) => !s.isFinal)?.id || visualStates[0]?.id || '',
      states: visualStates.map((s) => ({
        id: s.id,
        name: s.name,
        is_final: s.isFinal,
      })),
      transitions: visualTransitions.map((t) => ({
        from: t.from,
        to: t.to,
        trigger: t.trigger,
        description: t.description,
      })),
    };

    return JSON.stringify(config, null, 2);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      let configYaml = values.config;

      // If using visual mode, build YAML from visual data
      if (activeTab === 'visual') {
        configYaml = buildYamlFromVisual();
      }

      await onSubmit({
        name: values.name,
        description: values.description || '',
        config: configYaml,
      });
    } catch (err) {
      if (err instanceof Error) {
        message.error(err.message);
      }
    }
  };

  return (
    <Drawer
      title={editing ? '编辑状态机' : '新建状态机'}
      placement="right"
      width={800}
      onClose={onClose}
      open={open}
      extra={
        <Space>
          <Button onClick={onClose}>取消</Button>
          <Button type="primary" onClick={handleSubmit} loading={saving}>
            保存
          </Button>
        </Space>
      }
    >
      <Form form={form} layout="vertical">
        <Form.Item
          label="名称"
          name="name"
          rules={[{ required: true, message: '请输入状态机名称' }]}
        >
          <Input placeholder="例如：需求流程" />
        </Form.Item>

        <Form.Item
          label="描述"
          name="description"
        >
          <Input.TextArea placeholder="描述状态机的用途" />
        </Form.Item>
      </Form>

      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: 'visual',
            label: '可视化编辑',
            children: (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
                <Alert
                  message="可视化编辑器"
                  description="使用下方表格编辑状态机的状态和转换规则"
                  type="info"
                  showIcon
                />

                <div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                    <strong>状态列表</strong>
                    <Button size="small" onClick={addState}>
                      添加状态
                    </Button>
                  </div>
                  <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                    <thead>
                      <tr>
                        <th style={{ border: '1px solid #ddd', padding: 8 }}>ID</th>
                        <th style={{ border: '1px solid #ddd', padding: 8 }}>名称</th>
                        <th style={{ border: '1px solid #ddd', padding: 8 }}>终态</th>
                        <th style={{ border: '1px solid #ddd', padding: 8 }}>操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      {visualStates.map((state, index) => (
                        <tr key={index}>
                          <td style={{ border: '1px solid #ddd', padding: 4 }}>
                            <Input
                              size="small"
                              value={state.id}
                              onChange={(e) =>
                                handleVisualStateChange(index, 'id', e.target.value)
                              }
                            />
                          </td>
                          <td style={{ border: '1px solid #ddd', padding: 4 }}>
                            <Input
                              size="small"
                              value={state.name}
                              onChange={(e) =>
                                handleVisualStateChange(index, 'name', e.target.value)
                              }
                            />
                          </td>
                          <td style={{ border: '1px solid #ddd', padding: 4, textAlign: 'center' }}>
                            <input
                              type="checkbox"
                              checked={state.isFinal}
                              onChange={(e) =>
                                handleVisualStateChange(index, 'isFinal', e.target.checked)
                              }
                            />
                          </td>
                          <td style={{ border: '1px solid #ddd', padding: 4 }}>
                            <Button size="small" danger onClick={() => removeState(index)}>
                              删除
                            </Button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                <div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                    <strong>转换规则</strong>
                    <Button size="small" onClick={addTransition}>
                      添加转换
                    </Button>
                  </div>
                  <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                    <thead>
                      <tr>
                        <th style={{ border: '1px solid #ddd', padding: 8 }}>源状态</th>
                        <th style={{ border: '1px solid #ddd', padding: 8 }}>目标状态</th>
                        <th style={{ border: '1px solid #ddd', padding: 8 }}>触发器</th>
                        <th style={{ border: '1px solid #ddd', padding: 8 }}>描述</th>
                        <th style={{ border: '1px solid #ddd', padding: 8 }}>操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      {visualTransitions.map((transition, index) => (
                        <tr key={index}>
                          <td style={{ border: '1px solid #ddd', padding: 4 }}>
                            <select
                              style={{ width: '100%' }}
                              value={transition.from}
                              onChange={(e) =>
                                handleVisualTransitionChange(index, 'from', e.target.value)
                              }
                            >
                              {visualStates.map((s) => (
                                <option key={s.id} value={s.id}>
                                  {s.name}
                                </option>
                              ))}
                            </select>
                          </td>
                          <td style={{ border: '1px solid #ddd', padding: 4 }}>
                            <select
                              style={{ width: '100%' }}
                              value={transition.to}
                              onChange={(e) =>
                                handleVisualTransitionChange(index, 'to', e.target.value)
                              }
                            >
                              {visualStates.map((s) => (
                                <option key={s.id} value={s.id}>
                                  {s.name}
                                </option>
                              ))}
                            </select>
                          </td>
                          <td style={{ border: '1px solid #ddd', padding: 4 }}>
                            <Input
                              size="small"
                              value={transition.trigger}
                              onChange={(e) =>
                                handleVisualTransitionChange(index, 'trigger', e.target.value)
                              }
                            />
                          </td>
                          <td style={{ border: '1px solid #ddd', padding: 4 }}>
                            <Input
                              size="small"
                              value={transition.description}
                              onChange={(e) =>
                                handleVisualTransitionChange(index, 'description', e.target.value)
                              }
                            />
                          </td>
                          <td style={{ border: '1px solid #ddd', padding: 4 }}>
                            <Button size="small" danger onClick={() => removeTransition(index)}>
                              删除
                            </Button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            ),
          },
          {
            key: 'yaml',
            label: 'YAML 编辑',
            children: (
              <div>
                <Alert
                  message="YAML 格式"
                  description="直接编辑状态机的 YAML 配置"
                  type="info"
                  showIcon
                  style={{ marginBottom: 16 }}
                />
                <Form.Item
                  name="config"
                  rules={[{ required: true, message: '请输入状态机配置' }]}
                >
                  <Input.TextArea
                    rows={20}
                    style={{ fontFamily: 'monospace' }}
                    onChange={(e) => handleYamlChange(e.target.value)}
                  />
                </Form.Item>
              </div>
            ),
          },
        ]}
      />
    </Drawer>
  );
};