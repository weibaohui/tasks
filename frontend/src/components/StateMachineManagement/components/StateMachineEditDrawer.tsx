/**
 * StateMachine Edit Drawer Component
 */
import React, { useEffect } from 'react';
import { Drawer, Form, Input, Button, Space, message, Alert, Tabs, Modal, Select, Table, Tag, Divider, Collapse } from 'antd';
import { PlusOutlined, EditOutlined, InfoCircleOutlined, ThunderboltOutlined, CheckOutlined } from '@ant-design/icons';
import type { StateMachine, CreateStateMachineRequest, TransitionHook } from '../../../types/stateMachine';
import { hookExamples, examplesByCategory, categoryNames, type HookExample } from './hookExamples';

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

// 可用的模板变量
const TEMPLATE_VARIABLES = [
  { key: 'requirement_id', label: 'requirement_id', desc: '需求ID' },
  { key: 'project_id', label: 'project_id', desc: '项目ID' },
  { key: 'state_machine_id', label: 'state_machine_id', desc: '状态机ID' },
  { key: 'from_state', label: 'from_state', desc: '源状态' },
  { key: 'to_state', label: 'to_state', desc: '目标状态' },
  { key: 'trigger', label: 'trigger', desc: '触发器' },
  { key: 'hook_name', label: 'hook_name', desc: 'Hook名称' },
  { key: 'hook_type', label: 'hook_type', desc: 'Hook类型' },
];

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
  const [editingHook, setEditingHook] = React.useState<{ transitionIndex: number; hookIndex: number } | null>(null);
  const [currentHook, setCurrentHook] = React.useState<TransitionHook>({
    name: '',
    type: 'webhook',
    config: { url: '', method: 'POST', command: '' },
    retry: 0,
    timeout: 30,
  });
  const [selectedExample, setSelectedExample] = React.useState<HookExample | null>(null);

  // 应用示例模板
  const applyExample = (example: HookExample) => {
    setSelectedExample(example);
    setCurrentHook({
      name: example.name,
      type: example.type,
      config: example.type === 'webhook'
        ? { url: example.config.url || '', method: example.config.method || 'POST' }
        : { command: example.config.command || '' },
      retry: example.retry || 0,
      timeout: example.timeout || 30,
    });
  };

  // 根据当前选择的类型筛选示例
  const filteredExamples = hookExamples.filter((e) => e.type === currentHook.type);

  useEffect(() => {
    if (open) {
      if (editing) {
        form.setFieldsValue({
          name: editing.name,
          description: editing.description,
          config: JSON.stringify(editing.config, null, 2),
        });
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
  const openAddHook = (transitionIndex: number) => {
    setEditingHook({ transitionIndex, hookIndex: -1 });
    setCurrentHook({
      name: '',
      type: 'webhook',
      config: { url: '', method: 'POST', command: '' },
      retry: 0,
      timeout: 30,
    });
    setHookModalOpen(true);
  };

  const openEditHook = (transitionIndex: number, hookIndex: number) => {
    const hook = visualTransitions[transitionIndex].hooks[hookIndex];
    setEditingHook({ transitionIndex, hookIndex });
    setCurrentHook({
      name: hook.name,
      type: hook.type,
      config: { ...hook.config },
      retry: hook.retry || 0,
      timeout: hook.timeout || 30,
    });
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
      if (editingHook && editingHook.hookIndex >= 0) {
        updated[editingHook.transitionIndex].hooks[editingHook.hookIndex] = { ...currentHook };
      } else {
        updated[editingHook!.transitionIndex].hooks.push({ ...currentHook });
      }
      return updated;
    });

    setHookModalOpen(false);
  };

  const insertVariable = (field: 'url' | 'command', variable: string) => {
    if (field === 'url') {
      const url = (currentHook.config.url as string) || '';
      setCurrentHook({
        ...currentHook,
        config: { ...currentHook.config, url: url + `{{${variable}}}` },
      });
    } else {
      const cmd = (currentHook.config.command as string) || '';
      setCurrentHook({
        ...currentHook,
        config: { ...currentHook.config, command: cmd + `{{${variable}}}` },
      });
    }
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
        width={1000}
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
                    description="使用表格编辑状态机的状态和转换规则，可为转换添加 Hook"
                    type="info"
                    showIcon
                  />

                  <div>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                      <strong>状态列表</strong>
                      <Button size="small" onClick={addState} icon={<PlusOutlined />}>
                        添加状态
                      </Button>
                    </div>
                    <Table
                      dataSource={visualStates}
                      rowKey={(_, index) => `state-${index}`}
                      pagination={false}
                      size="small"
                      columns={[
                        { title: 'ID', dataIndex: 'id', key: 'id', width: 120 },
                        { title: '名称', dataIndex: 'name', key: 'name', width: 150 },
                        {
                          title: '终态',
                          dataIndex: 'isFinal',
                          key: 'isFinal',
                          width: 80,
                          render: (isFinal: boolean, _, index: number) => (
                            <input
                              type="checkbox"
                              checked={isFinal}
                              onChange={(e) =>
                                handleVisualStateChange(index, 'isFinal', e.target.checked)
                              }
                            />
                          ),
                        },
                        {
                          title: '操作',
                          key: 'action',
                          width: 80,
                          render: (_, __, index: number) => (
                            <Button size="small" danger onClick={() => removeState(index)}>
                              删除
                            </Button>
                          ),
                        },
                      ]}
                    />
                  </div>

                  <div>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                      <strong>转换规则</strong>
                      <Button size="small" onClick={addTransition} icon={<PlusOutlined />}>
                        添加转换
                      </Button>
                    </div>
                    <Table
                      dataSource={visualTransitions}
                      rowKey={(_, index) => `transition-${index}`}
                      pagination={false}
                      size="small"
                      columns={[
                        {
                          title: '源状态',
                          dataIndex: 'from',
                          key: 'from',
                          width: 120,
                          render: (from: string) => {
                            const state = visualStates.find((s) => s.id === from);
                            return state?.name || from;
                          },
                        },
                        {
                          title: '目标状态',
                          dataIndex: 'to',
                          key: 'to',
                          width: 120,
                          render: (to: string) => {
                            const state = visualStates.find((s) => s.id === to);
                            return state?.name || to;
                          },
                        },
                        { title: '触发器', dataIndex: 'trigger', key: 'trigger', width: 100 },
                        { title: '描述', dataIndex: 'description', key: 'description' },
                        {
                          title: 'Hooks',
                          key: 'hooks',
                          width: 220,
                          render: (_, record, transitionIndex: number) => (
                            <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                              {record.hooks.map((hook, hookIndex) => (
                                <Tag
                                  key={hookIndex}
                                  closable
                                  onClose={(e) => {
                                    e.stopPropagation();
                                    removeHook(transitionIndex, hookIndex);
                                  }}
                                  icon={<EditOutlined style={{ cursor: 'pointer' }} />}
                                  onClick={() => openEditHook(transitionIndex, hookIndex)}
                                  color={hook.type === 'command' ? 'orange' : 'blue'}
                                >
                                  {hook.type === 'command' ? '📦' : '🌐'} {hook.name}
                                </Tag>
                              ))}
                              <Button
                                size="small"
                                type="dashed"
                                icon={<PlusOutlined />}
                                onClick={() => openAddHook(transitionIndex)}
                              >
                                添加 Hook
                              </Button>
                            </div>
                          ),
                        },
                        {
                          title: '操作',
                          key: 'action',
                          width: 80,
                          render: (_, __, index: number) => (
                            <Button size="small" danger onClick={() => removeTransition(index)}>
                              删除
                            </Button>
                          ),
                        },
                      ]}
                    />
                  </div>
                </div>
              ),
            },
            {
              key: 'yaml',
              label: 'JSON 编辑',
              children: (
                <div>
                  <Alert
                    message="JSON 格式"
                    description="直接编辑状态机的 JSON 配置，支持 webhook 和 command 类型的 Hook"
                    type="info"
                    showIcon
                    style={{ marginBottom: 16 }}
                  />
                  <Form.Item
                    name="config"
                    rules={[{ required: true, message: '请输入状态机配置' }]}
                  >
                    <Input.TextArea
                      rows={25}
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
        title={editingHook && editingHook.hookIndex >= 0 ? '编辑 Hook' : '添加 Hook'}
        open={hookModalOpen}
        onOk={saveHook}
        onCancel={() => {
          setHookModalOpen(false);
          setSelectedExample(null);
        }}
        okText="保存"
        cancelText="取消"
        width={900}
      >
        <div style={{ display: 'flex', gap: 24, minHeight: 400 }}>
          {/* 左侧：示例库 */}
          <div style={{ width: 280, borderRight: '1px solid #f0f0f0', paddingRight: 16 }}>
            <div style={{ marginBottom: 12 }}>
              <Space>
                <ThunderboltOutlined />
                <strong>示例模板</strong>
              </Space>
            </div>
            <Collapse
              defaultActiveKey={['notification', 'deployment']}
              ghost
              style={{ background: 'transparent' }}
            >
              {(Object.keys(examplesByCategory) as Array<keyof typeof examplesByCategory>).map((category) => {
                const examples = examplesByCategory[category].filter((e) => e.type === currentHook.type);
                if (examples.length === 0) return null;
                return (
                  <Collapse.Panel
                    key={category}
                    header={<span style={{ fontSize: 12 }}>{categoryNames[category]} ({examples.length})</span>}
                  >
                    {examples.map((example) => (
                      <div
                        key={example.id}
                        onClick={() => applyExample(example)}
                        style={{
                          padding: '8px 12px',
                          marginBottom: 4,
                          borderRadius: 6,
                          border: selectedExample?.id === example.id ? '2px solid #1890ff' : '1px dashed #d9d9d9',
                          background: selectedExample?.id === example.id ? '#e6f7ff' : '#fafafa',
                          cursor: 'pointer',
                          transition: 'all 0.2s',
                        }}
                      >
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <span style={{ fontWeight: 500, fontSize: 13 }}>{example.name}</span>
                          {selectedExample?.id === example.id && <CheckOutlined style={{ color: '#1890ff' }} />}
                        </div>
                        <div style={{ fontSize: 11, color: '#888', marginTop: 2 }}>{example.description}</div>
                      </div>
                    ))}
                  </Collapse.Panel>
                );
              })}
            </Collapse>
            {filteredExamples.length === 0 && (
              <div style={{ textAlign: 'center', color: '#999', padding: 20 }}>
                先选择 Hook 类型
              </div>
            )}
          </div>

          {/* 右侧：表单编辑 */}
          <div style={{ flex: 1 }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              <Form layout="vertical">
                <Form.Item label="Hook 名称" required>
                  <Input
                    value={currentHook.name}
                    onChange={(e) => setCurrentHook({ ...currentHook, name: e.target.value })}
                    placeholder="例如：发送通知、执行部署"
                  />
                </Form.Item>

                <Form.Item label="Hook 类型" required>
                  <Select
                    value={currentHook.type}
                    onChange={(value) => {
                      setSelectedExample(null);
                      setCurrentHook({
                        ...currentHook,
                        type: value,
                        config: value === 'webhook' ? { url: '', method: 'POST' } : { command: '' },
                      });
                    }}
                  >
                    <Select.Option value="webhook">
                      <span>🌐 Webhook</span>
                    </Select.Option>
                    <Select.Option value="command">
                      <span>📦 命令执行</span>
                    </Select.Option>
                  </Select>
                </Form.Item>
              </Form>

              <Divider style={{ margin: '8px 0' }} />

              {currentHook.type === 'webhook' && (
                <div>
                  <Form layout="vertical">
                    <Form.Item label="Webhook URL" required>
                      <Input
                        value={(currentHook.config.url as string) || ''}
                        onChange={(e) => setCurrentHook({
                          ...currentHook,
                          config: { ...currentHook.config, url: e.target.value },
                        })}
                        placeholder="https://example.com/webhook"
                        addonAfter={
                          <Tag color="blue" style={{ margin: 0 }}>可使用 {`{{variable}}`}</Tag>
                        }
                      />
                    </Form.Item>
                    <Form.Item label="请求方法">
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
                  </Form>
                </div>
              )}

              {currentHook.type === 'command' && (
                <div>
                  <Form layout="vertical">
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
                  </Form>
                </div>
              )}

              <Form layout="horizontal" style={{ display: 'flex', gap: 16 }}>
                <Form.Item label="超时时间（秒）" style={{ flex: 1 }}>
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
                <Form.Item label="重试次数" style={{ flex: 1 }}>
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
              </Form>

              <Divider style={{ margin: '8px 0' }} />

              {/* 模板变量说明 */}
              <Alert
                type="info"
                showIcon
                icon={<InfoCircleOutlined />}
                message="可用模板变量"
                description={
                  <div>
                    <p style={{ marginBottom: 8 }}>
                      点击以下变量可插入到 {currentHook.type === 'webhook' ? 'URL' : '命令'} 中：
                    </p>
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                      {TEMPLATE_VARIABLES.map((v) => (
                        <Tag
                          key={v.key}
                          color="cyan"
                          style={{ cursor: 'pointer' }}
                          onClick={() => insertVariable(currentHook.type === 'webhook' ? 'url' : 'command', v.key)}
                        >
                          {`{{${v.key}}}`}
                        </Tag>
                      ))}
                    </div>
                  </div>
                }
              />
            </div>
          </div>
        </div>
      </Modal>
    </>
  );
};