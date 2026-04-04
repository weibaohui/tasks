/**
 * StateMachine Edit Drawer Component
 */
import React, { useEffect } from 'react';
import { Drawer, Form, Input, Button, Space, message, Alert, Tabs, Modal, Select } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import type { StateMachine, CreateStateMachineRequest, TransitionHook } from '../../../types/stateMachine';

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

interface TransitionWithHooks {
  from: string;
  to: string;
  trigger: string;
  description: string;
  hooks: TransitionHook[];
}

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
  const [visualTransitions, setVisualTransitions] = React.useState<TransitionWithHooks[]>([]);
  const [hookModalOpen, setHookModalOpen] = React.useState(false);
  const [editingHookIndex, setEditingHookIndex] = React.useState<{ transitionIndex: number; hookIndex: number } | null>(null);
  const [currentHook, setCurrentHook] = React.useState<TransitionHook>({
    name: '',
    type: 'webhook',
    config: {},
    retry: 0,
    timeout: 30,
  });

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
              hooks: t.hooks || [],
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
          { from: 'created', to: 'in_progress', trigger: 'start', description: '开始处理', hooks: [] },
          { from: 'in_progress', to: 'completed', trigger: 'complete', description: '完成处理', hooks: [] },
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
          parsed.transitions.map((t: { from: string; to: string; trigger: string; description?: string; hooks?: TransitionHook[] }) => ({
            from: t.from,
            to: t.to,
            trigger: t.trigger,
            description: t.description || '',
            hooks: t.hooks || [],
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
      { from: visualStates[0]?.id || '', to: visualStates[0]?.id || '', trigger: '', description: '', hooks: [] },
    ]);
  };

  const removeTransition = (index: number) => {
    setVisualTransitions((prev) => prev.filter((_, i) => i !== index));
  };

  // Hook management
  const [pendingHookTransitionIndex, setPendingHookTransitionIndex] = React.useState<number>(-1);

  const openAddHook = (transitionIndex: number) => {
    setPendingHookTransitionIndex(transitionIndex);
    setEditingHookIndex(null);
    setCurrentHook({
      name: '',
      type: 'webhook',
      config: { url: '', method: 'POST' },
      retry: 0,
      timeout: 30,
    });
    setHookModalOpen(true);
  };

  const openEditHook = (transitionIndex: number, hookIndex: number) => {
    const transition = visualTransitions[transitionIndex];
    const hook = transition.hooks[hookIndex];
    setPendingHookTransitionIndex(transitionIndex);
    setEditingHookIndex({ transitionIndex, hookIndex });
    setCurrentHook({ ...hook });
    setHookModalOpen(true);
  };

  const removeHook = (transitionIndex: number, hookIndex: number) => {
    setVisualTransitions((prev) => {
      const updated = [...prev];
      updated[transitionIndex].hooks = updated[transitionIndex].hooks.filter((_, i) => i !== hookIndex);
      return updated;
    });
  };

  const saveHook = () => {
    if (!currentHook.name) {
      message.warning('请输入 Hook 名称');
      return;
    }
    if (currentHook.type === 'webhook' && !currentHook.config.url) {
      message.warning('请输入 Webhook URL');
      return;
    }
    if (currentHook.type === 'command' && !currentHook.config.command) {
      message.warning('请输入要执行的命令');
      return;
    }

    setVisualTransitions((prev) => {
      const updated = [...prev];
      if (editingHookIndex && editingHookIndex.hookIndex >= 0) {
        // Editing existing hook
        updated[editingHookIndex.transitionIndex].hooks[editingHookIndex.hookIndex] = { ...currentHook };
      } else if (pendingHookTransitionIndex >= 0) {
        // Adding new hook to specific transition
        updated[pendingHookTransitionIndex].hooks.push({ ...currentHook });
      }
      return updated;
    });

    // Fix: properly add hook to correct transition
    if (editingHookIndex) {
      // Already handled above
    } else {
      // Find the transition that was being edited (the one that triggered openAddHook)
      // Since we don't have that info, we need to pass it
    }

    setHookModalOpen(false);
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
        hooks: t.hooks.length > 0 ? t.hooks : undefined,
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
    <>
      <Drawer
        title={editing ? '编辑状态机' : '新建状态机'}
        placement="right"
        width={900}
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
                    description="使用下方表格编辑状态机的状态和转换规则，可为转换添加 Hook"
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
                          <th style={{ border: '1px solid #ddd', padding: 8 }}>Hooks</th>
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
                              <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                                {transition.hooks.map((hook, hIndex) => (
                                  <div key={hIndex} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                                    <Button
                                      size="small"
                                      type="link"
                                      onClick={() => openEditHook(index, hIndex)}
                                      style={{ padding: '0 4px', height: 'auto' }}
                                    >
                                      {hook.type === 'command' ? '📦' : '🌐'} {hook.name}
                                    </Button>
                                    <Button
                                      size="small"
                                      danger
                                      onClick={() => removeHook(index, hIndex)}
                                      style={{ padding: '0 4px', height: 'auto' }}
                                    >
                                      ×
                                    </Button>
                                  </div>
                                ))}
                                <Button
                                  size="small"
                                  type="dashed"
                                  icon={<PlusOutlined />}
                                  onClick={() => {
                                    // Store current transition index for hook creation
                                    setEditingHookIndex({ transitionIndex: index, hookIndex: -1 });
                                    openAddHook(index);
                                  }}
                                  style={{ marginTop: 4 }}
                                >
                                  添加 Hook
                                </Button>
                              </div>
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
                    description="直接编辑状态机的 YAML 配置，支持 webhook 和 command 类型的 Hook"
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

      {/* Hook 编辑 Modal */}
      <Modal
        title={editingHookIndex?.hookIndex === -1 || editingHookIndex === null ? '添加 Hook' : '编辑 Hook'}
        open={hookModalOpen}
        onOk={saveHook}
        onCancel={() => setHookModalOpen(false)}
        okText="保存"
        cancelText="取消"
      >
        <Form layout="vertical">
          <Form.Item label="Hook 名称" required>
            <Input
              value={currentHook.name}
              onChange={(e) => setCurrentHook({ ...currentHook, name: e.target.value })}
              placeholder="例如：发送通知"
            />
          </Form.Item>

          <Form.Item label="Hook 类型" required>
            <Select
              value={currentHook.type}
              onChange={(value) => setCurrentHook({
                ...currentHook,
                type: value,
                config: value === 'webhook' ? { url: '', method: 'POST' } : { command: '' },
              })}
            >
              <Select.Option value="webhook">Webhook</Select.Option>
              <Select.Option value="command">命令执行</Select.Option>
            </Select>
          </Form.Item>

          {currentHook.type === 'webhook' && (
            <>
              <Form.Item label="URL" required>
                <Input
                  value={(currentHook.config.url as string) || ''}
                  onChange={(e) => setCurrentHook({
                    ...currentHook,
                    config: { ...currentHook.config, url: e.target.value },
                  })}
                  placeholder="https://example.com/webhook"
                />
              </Form.Item>
              <Form.Item label="Method">
                <Select
                  value={(currentHook.config.method as string) || 'POST'}
                  onChange={(value) => setCurrentHook({
                    ...currentHook,
                    config: { ...currentHook.config, method: value },
                  })}
                >
                  <Select.Option value="POST">POST</Select.Option>
                  <Select.Option value="GET">GET</Select.Option>
                  <Select.Option value="PUT">PUT</Select.Option>
                  <Select.Option value="DELETE">DELETE</Select.Option>
                </Select>
              </Form.Item>
            </>
          )}

          {currentHook.type === 'command' && (
            <Form.Item label="命令" required>
              <Input.TextArea
                value={(currentHook.config.command as string) || ''}
                onChange={(e) => setCurrentHook({
                  ...currentHook,
                  config: { ...currentHook.config, command: e.target.value },
                })}
                placeholder="/bin/bash /scripts/deploy.sh {{requirement_id}}"
                rows={3}
              />
            </Form.Item>
          )}

          <Form.Item label="超时时间（秒）">
            <Input
              type="number"
              value={currentHook.timeout || 30}
              onChange={(e) => setCurrentHook({
                ...currentHook,
                timeout: parseInt(e.target.value) || 30,
              })}
              placeholder="30"
            />
          </Form.Item>

          <Form.Item label="重试次数">
            <Input
              type="number"
              value={currentHook.retry || 0}
              onChange={(e) => setCurrentHook({
                ...currentHook,
                retry: parseInt(e.target.value) || 0,
              })}
              placeholder="0"
            />
          </Form.Item>

          <Alert
            message="模板变量"
            description={
              <div>
                <p>支持的变量：</p>
                <code>{'{{requirement_id}}'} - 需求ID</code><br />
                <code>{'{{hook_name}}'} - Hook名称</code><br />
                <code>{'{{hook_type}}'} - Hook类型</code>
              </div>
            }
            type="info"
            showIcon
          />
        </Form>
      </Modal>
    </>
  );
};