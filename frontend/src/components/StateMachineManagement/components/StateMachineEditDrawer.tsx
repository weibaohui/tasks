/**
 * StateMachine Edit Drawer Component
 */
import React, { useEffect } from 'react';
import { Drawer, Form, Input, Button, Space, message, Alert, Tabs, Modal, Select, Table, Tag, Divider, Collapse } from 'antd';
import { PlusOutlined, EditOutlined, InfoCircleOutlined, ThunderboltOutlined, CheckOutlined, FileTextOutlined } from '@ant-design/icons';
import * as yaml from 'js-yaml';
import type { StateMachine, CreateStateMachineRequest, TransitionHook } from '../../../types/stateMachine';
import { hookExamples, examplesByCategory, categoryNames, type HookExample } from './hookExamples';
import { stateMachineTemplates, type StateMachineTemplate } from './stateMachineTemplates';

const DEFAULT_YAML = `name: example_flow
description: 示例状态机流程
initial_state: created

states:
  - id: created
    name: 已创建
    is_final: false
    # ai_guide: |           # 即将支持：AI操作指南（可选）
    #   ## 当前阶段任务
    #   1. 分析需求
    #   2. 制定实现计划
    # auto_init: |          # 即将支持：自动初始化命令（可选）
    #   git clone {{git_repo_url}} .
    # success_criteria: ... # 即将支持：成功判断标准（可选）
    # failure_criteria: ... # 即将支持：失败判断标准（可选）

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

// 需求元数据变量（由分身Agent注入）
const METADATA_VARIABLES = [
  { key: 'REQUIREMENT_ID', label: 'REQUIREMENT_ID', desc: '需求ID（环境变量格式）' },
  { key: 'PROJECT_ID', label: 'PROJECT_ID', desc: '项目ID（环境变量格式）' },
  { key: 'STATE_MACHINE_NAME', label: 'STATE_MACHINE_NAME', desc: '状态机名称' },
  { key: 'REQUIREMENT_TYPE', label: 'REQUIREMENT_TYPE', desc: '需求类型（normal/heartbeat）' },
  { key: 'REQUIREMENT_STATUS', label: 'REQUIREMENT_STATUS', desc: '需求当前状态' },
  { key: 'REQUIREMENT_TITLE', label: 'REQUIREMENT_TITLE', desc: '需求标题' },
];

// CLI 注入的自定义元数据
const CLI_METADATA_VARIABLES = [
  { key: 'operator', label: 'operator', desc: '操作人（通过 --metadata 注入）' },
  { key: 'source', label: 'source', desc: '触发来源（通过 --metadata 注入）' },
];

interface TransitionWithHooks {
  from: string;
  to: string;
  trigger: string;
  description: string;
  hooks: TransitionHook[];
}

// 扩展 State 类型，包含 AI Guide 字段
interface StateWithAIGuide {
  id: string;
  name: string;
  isFinal: boolean;
  aiGuide?: string;
  autoInit?: string;
  successCriteria?: string;
  failureCriteria?: string;
  triggers?: { trigger: string; description?: string; condition?: string }[];
}

// YAML 解析后的原始状态类型（snake_case）
interface YamlParsedState {
  id: string;
  name: string;
  is_final?: boolean;
  isFinal?: boolean;
  ai_guide?: string;
  aiGuide?: string;
  auto_init?: string;
  autoInit?: string;
  success_criteria?: string;
  successCriteria?: string;
  failure_criteria?: string;
  failureCriteria?: string;
  triggers?: { trigger: string; description?: string; condition?: string }[];
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
  const [visualStates, setVisualStates] = React.useState<StateWithAIGuide[]>([]);
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
  const [selectedTemplate, setSelectedTemplate] = React.useState<StateMachineTemplate | null>(null);

  // AI Guide 编辑弹窗状态
  const [aiGuideModalOpen, setAiGuideModalOpen] = React.useState(false);
  const [editingStateIndex, setEditingStateIndex] = React.useState<number | null>(null);
  const [currentAIGuide, setCurrentAIGuide] = React.useState<StateWithAIGuide>({
    id: '',
    name: '',
    isFinal: false,
    aiGuide: '',
    autoInit: '',
    successCriteria: '',
    failureCriteria: '',
    triggers: [],
  });

  // 应用 Hook 示例模板
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

  // 待应用的模板（用于在标签页切换后设置表单值）
  const pendingTemplateRef = React.useRef<StateMachineTemplate | null>(null);

  // 应用状态机模板
  const applyTemplate = (template: StateMachineTemplate) => {
    setSelectedTemplate(template);
    // 如果已经在 yaml 标签页，直接设置表单值
    if (activeTab === 'yaml') {
      form.setFieldsValue({
        name: template.name,
        description: template.description,
        config: template.yaml,
      });
    } else {
      // 否则标记为待处理，等待标签页切换后应用
      pendingTemplateRef.current = template;
    }
    // 切换到 YAML 编辑模式
    setActiveTab('yaml');
    message.success(`已加载模板：${template.name}`);
  };

  // 监听标签页变化，在切换到 yaml 标签页后应用待处理的模板值
  useEffect(() => {
    if (activeTab === 'yaml' && pendingTemplateRef.current) {
      const template = pendingTemplateRef.current;
      form.setFieldsValue({
        name: template.name,
        description: template.description,
        config: template.yaml,
      });
      pendingTemplateRef.current = null;
    }
  }, [activeTab, form]);

  // 监听标签页变化，在切换到 visual 标签页时解析 YAML 并更新可视化状态
  useEffect(() => {
    if (activeTab !== 'visual') return;

    const configValue = form.getFieldValue('config') || '';
    const parsed = parseConfig(configValue);
    if (!parsed) return;

    if (parsed.states && Array.isArray(parsed.states)) {
      setVisualStates(
        parsed.states.map((s: YamlParsedState) => ({
          id: s.id,
          name: s.name,
          isFinal: s.is_final ?? s.isFinal ?? false,
          aiGuide: s.ai_guide ?? s.aiGuide ?? '',
          autoInit: s.auto_init ?? s.autoInit ?? '',
          successCriteria: s.success_criteria ?? s.successCriteria ?? '',
          failureCriteria: s.failure_criteria ?? s.failureCriteria ?? '',
          triggers: s.triggers || [],
        })),
      );
    }
    if (parsed.transitions && Array.isArray(parsed.transitions)) {
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
  }, [activeTab, form]);

  // 根据当前选择的类型筛选 Hook 示例
  const filteredExamples = hookExamples.filter((e) => e.type === currentHook.type);

  // 可视化编辑变化时同步到 YAML
  useEffect(() => {
    // 只有在可视化编辑模式下才自动同步
    if (activeTab !== 'visual') return;

    const name = form.getFieldValue('name') || 'unnamed';
    const description = form.getFieldValue('description') || '';

    const config: Record<string, unknown> = {
      name,
      description,
      initial_state: visualStates.find((s) => !s.isFinal)?.id || visualStates[0]?.id || '',
      states: visualStates.map((s) => {
        const state: Record<string, unknown> = {
          id: s.id,
          name: s.name,
          is_final: s.isFinal,
        };
        // 只在有值时添加 AI Guide 字段
        if (s.aiGuide) state.ai_guide = s.aiGuide;
        if (s.autoInit) state.auto_init = s.autoInit;
        if (s.successCriteria) state.success_criteria = s.successCriteria;
        if (s.failureCriteria) state.failure_criteria = s.failureCriteria;
        if (s.triggers && s.triggers.length > 0) state.triggers = s.triggers;
        return state;
      }),
      transitions: visualTransitions.map((t) => ({
        from: t.from,
        to: t.to,
        trigger: t.trigger,
        description: t.description,
        hooks: t.hooks.length > 0 ? t.hooks : undefined,
      })),
    };

    // 始终输出 YAML 格式
    form.setFieldsValue({ config: yaml.dump(config, { lineWidth: -1 }) });
  }, [visualStates, visualTransitions, activeTab, form]);

  useEffect(() => {
    if (open) {
      if (editing) {
        form.setFieldsValue({
          name: editing.name,
          description: editing.description,
          config: yaml.dump(editing.config, { lineWidth: -1 }),
        });
        try {
          const config = editing.config;
          setVisualStates(
            config.states.map((s) => ({
              id: s.id,
              name: s.name,
              isFinal: s.is_final,
              aiGuide: (s as YamlParsedState).ai_guide ?? (s as YamlParsedState).aiGuide ?? '',
              autoInit: (s as YamlParsedState).auto_init ?? (s as YamlParsedState).autoInit ?? '',
              successCriteria: (s as YamlParsedState).success_criteria ?? (s as YamlParsedState).successCriteria ?? '',
              failureCriteria: (s as YamlParsedState).failure_criteria ?? (s as YamlParsedState).failureCriteria ?? '',
              triggers: (s as YamlParsedState).triggers || [],
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

  // 解析配置（支持 JSON 和 YAML）
  const parseConfig = (value: string): Record<string, unknown> | null => {
    try {
      // 先尝试 JSON
      return JSON.parse(value);
    } catch {
      try {
        // 再尝试 YAML
        return yaml.load(value) as Record<string, unknown>;
      } catch {
        return null;
      }
    }
  };

  // YAML/JSON 编辑变化时同步到可视化
  const handleYamlChange = (value: string) => {
    const parsed = parseConfig(value);
    if (!parsed) return;

    if (parsed.states && Array.isArray(parsed.states)) {
      setVisualStates(
        parsed.states.map((s: YamlParsedState) => ({
          id: s.id,
          name: s.name,
          isFinal: s.is_final ?? s.isFinal ?? false,
          aiGuide: s.ai_guide ?? s.aiGuide ?? '',
          autoInit: s.auto_init ?? s.autoInit ?? '',
          successCriteria: s.success_criteria ?? s.successCriteria ?? '',
          failureCriteria: s.failure_criteria ?? s.failureCriteria ?? '',
          triggers: s.triggers || [],
        })),
      );
    }
    if (parsed.transitions && Array.isArray(parsed.transitions)) {
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

  // AI Guide management
  const openEditAIGuide = (stateIndex: number) => {
    const state = visualStates[stateIndex];
    setEditingStateIndex(stateIndex);
    setCurrentAIGuide({
      id: state.id,
      name: state.name,
      isFinal: state.isFinal,
      aiGuide: state.aiGuide || '',
      autoInit: state.autoInit || '',
      successCriteria: state.successCriteria || '',
      failureCriteria: state.failureCriteria || '',
      triggers: state.triggers || [],
    });
    setAiGuideModalOpen(true);
  };

  const saveAIGuide = () => {
    if (editingStateIndex === null) return;

    setVisualStates((prev) => {
      const updated = [...prev];
      updated[editingStateIndex] = { ...currentAIGuide };
      return updated;
    });
    setAiGuideModalOpen(false);
    setEditingStateIndex(null);
  };

  const addTriggerGuide = () => {
    setCurrentAIGuide((prev) => ({
      ...prev,
      triggers: [...(prev.triggers || []), { trigger: '', description: '', condition: '' }],
    }));
  };

  const removeTriggerGuide = (index: number) => {
    setCurrentAIGuide((prev) => ({
      ...prev,
      triggers: (prev.triggers || []).filter((_, i) => i !== index),
    }));
  };

  const updateTriggerGuide = (index: number, field: 'trigger' | 'description' | 'condition', value: string) => {
    setCurrentAIGuide((prev) => {
      const updatedTriggers = [...(prev.triggers || [])];
      updatedTriggers[index] = { ...updatedTriggers[index], [field]: value };
      return { ...prev, triggers: updatedTriggers };
    });
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
      states: visualStates.map((s) => {
        const state: Record<string, unknown> = {
          id: s.id,
          name: s.name,
          is_final: s.isFinal,
        };
        // 只在有值时添加 AI Guide 字段
        if (s.aiGuide) state.ai_guide = s.aiGuide;
        if (s.autoInit) state.auto_init = s.autoInit;
        if (s.successCriteria) state.success_criteria = s.successCriteria;
        if (s.failureCriteria) state.failure_criteria = s.failureCriteria;
        if (s.triggers && s.triggers.length > 0) state.triggers = s.triggers;
        return state;
      }),
      transitions: visualTransitions.map((t) => ({
        from: t.from,
        to: t.to,
        trigger: t.trigger,
        description: t.description,
        hooks: t.hooks.length > 0 ? t.hooks : undefined,
      })),
    };

    return yaml.dump(config, { lineWidth: -1 });
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
          {/* 模板选择器 */}
          {!editing && (
            <Form.Item label="快速开始">
              <Space>
                <Select
                  placeholder="从模板创建..."
                  style={{ width: 200 }}
                  value={selectedTemplate?.id}
                  onChange={(value) => {
                    const template = stateMachineTemplates.find((t) => t.id === value);
                    if (template) applyTemplate(template);
                  }}
                  options={stateMachineTemplates.map((t) => ({
                    value: t.id,
                    label: (
                      <Space>
                        <FileTextOutlined />
                        {t.name}
                      </Space>
                    ),
                  }))}
                />
                <Button
                  type="link"
                  size="small"
                  onClick={() => setSelectedTemplate(null)}
                >
                  清空
                </Button>
              </Space>
            </Form.Item>
          )}

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
                          title: 'AI Guide',
                          key: 'aiGuide',
                          width: 100,
                          render: (_, record, index: number) => (
                            <Button
                              size="small"
                              type={record.aiGuide ? 'primary' : 'default'}
                              ghost={!!record.aiGuide}
                              onClick={() => openEditAIGuide(index)}
                            >
                              {record.aiGuide ? '已配置' : '配置'}
                            </Button>
                          ),
                        },
                        {
                          title: '操作',
                          key: 'action',
                          render: (_, __, index: number) => (
                            <Button danger onClick={() => removeState(index)} type="link" size="small" style={{ padding: 0 }}>
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
                                  key={`${transitionIndex}-${hookIndex}-${hook.name}`}
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
                          render: (_, __, index: number) => (
                            <Button danger onClick={() => removeTransition(index)} type="link" size="small" style={{ padding: 0 }}>
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
              label: 'YAML 编辑',
              children: (
                <div>
                  <Alert
                    message="YAML 格式"
                    description={
                      <div>
                        <p>直接编辑状态机的 YAML 配置，支持 webhook 和 command 类型的 Hook</p>
                        <p style={{ marginTop: 8, marginBottom: 0 }}>
                          <strong>即将支持：</strong>状态可配置 AI Guide（ai_guide）、自动初始化（auto_init）、成功/失败判断标准等字段
                        </p>
                      </div>
                    }
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
        </Form>
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
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                    {/* 基础变量 */}
                    <div>
                      <p style={{ marginBottom: 8, fontWeight: 500 }}>
                        基础变量（点击插入）：
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

                    {/* 需求元数据变量 */}
                    <div>
                      <p style={{ marginBottom: 8, fontWeight: 500 }}>
                        需求元数据（由分身Agent自动注入）：
                      </p>
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                        {METADATA_VARIABLES.map((v) => (
                          <Tag
                            key={v.key}
                            color="blue"
                            style={{ cursor: 'pointer' }}
                            onClick={() => insertVariable(currentHook.type === 'webhook' ? 'url' : 'command', v.key)}
                          >
                            {`{{${v.key}}}`}
                          </Tag>
                        ))}
                      </div>
                    </div>

                    {/* CLI 自定义元数据 */}
                    <div>
                      <p style={{ marginBottom: 8, fontWeight: 500 }}>
                        CLI 注入的自定义变量（通过 --metadata 参数）：
                      </p>
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                        {CLI_METADATA_VARIABLES.map((v) => (
                          <Tag
                            key={v.key}
                            color="orange"
                            style={{ cursor: 'pointer' }}
                            onClick={() => insertVariable(currentHook.type === 'webhook' ? 'url' : 'command', v.key)}
                          >
                            {`{{${v.key}}}`}
                          </Tag>
                        ))}
                      </div>
                      <p style={{ marginTop: 8, fontSize: 12, color: '#666' }}>
                        {'示例：taskmanager statemachine execute --machine=workflow --from=todo --trigger=complete --metadata=\'{"operator":"zhangsan"}\''}
                      </p>
                    </div>
                  </div>
                }
              />
            </div>
          </div>
        </div>
      </Modal>

      {/* AI Guide 编辑 Modal */}
      <Modal
        title={editingStateIndex !== null ? `编辑 AI Guide: ${visualStates[editingStateIndex]?.name || ''}` : '编辑 AI Guide'}
        open={aiGuideModalOpen}
        onOk={saveAIGuide}
        onCancel={() => {
          setAiGuideModalOpen(false);
          setEditingStateIndex(null);
        }}
        okText="保存"
        cancelText="取消"
        width={800}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16, maxHeight: '70vh', overflow: 'auto' }}>
          <Alert
            type="info"
            showIcon
            message="AI Guide 配置"
            description="配置当前状态的 AI 执行指南，包括任务说明、成功/失败判断标准、可用触发器等。"
          />

          <Form layout="vertical">
            <Form.Item label="AI 操作指南 (ai_guide)" extra="Markdown 格式，告诉 AI 当前阶段应该做什么">
              <Input.TextArea
                value={currentAIGuide.aiGuide}
                onChange={(e) => setCurrentAIGuide({ ...currentAIGuide, aiGuide: e.target.value })}
                placeholder="## 当前阶段任务\n1. 分析需求\n2. 编写代码\n3. 运行测试"
                rows={8}
              />
            </Form.Item>

            <Form.Item label="自动初始化命令 (auto_init)" extra="进入此状态时自动执行的 Shell 命令（可选）">
              <Input.TextArea
                value={currentAIGuide.autoInit}
                onChange={(e) => setCurrentAIGuide({ ...currentAIGuide, autoInit: e.target.value })}
                placeholder="#!/bin/bash\ngit clone {{git_repo_url}} ."
                rows={4}
                style={{ fontFamily: 'monospace' }}
              />
            </Form.Item>

            <Form.Item label="成功判断标准 (success_criteria)" extra="AI 根据此标准判断任务是否成功完成">
              <Input.TextArea
                value={currentAIGuide.successCriteria}
                onChange={(e) => setCurrentAIGuide({ ...currentAIGuide, successCriteria: e.target.value })}
                placeholder="测试全部通过，代码符合规范"
                rows={2}
              />
            </Form.Item>

            <Form.Item label="失败判断标准 (failure_criteria)" extra="AI 根据此标准判断任务是否失败">
              <Input.TextArea
                value={currentAIGuide.failureCriteria}
                onChange={(e) => setCurrentAIGuide({ ...currentAIGuide, failureCriteria: e.target.value })}
                placeholder="无法实现需求或遇到技术障碍"
                rows={2}
              />
            </Form.Item>

            <Divider style={{ margin: '8px 0' }} />

            <Form.Item label="可用触发器">
              <div style={{ marginBottom: 8 }}>
                <Button size="small" onClick={addTriggerGuide} icon={<PlusOutlined />}>
                  添加触发器
                </Button>
              </div>
              {(currentAIGuide.triggers || []).map((t, index) => (
                <div key={index} style={{ display: 'flex', gap: 8, marginBottom: 8, alignItems: 'flex-start' }}>
                  <Input
                    placeholder="触发器名称"
                    value={t.trigger}
                    onChange={(e) => updateTriggerGuide(index, 'trigger', e.target.value)}
                    style={{ width: 120 }}
                  />
                  <Input
                    placeholder="描述"
                    value={t.description}
                    onChange={(e) => updateTriggerGuide(index, 'description', e.target.value)}
                    style={{ flex: 1 }}
                  />
                  <Input
                    placeholder="触发条件"
                    value={t.condition}
                    onChange={(e) => updateTriggerGuide(index, 'condition', e.target.value)}
                    style={{ flex: 1 }}
                  />
                  <Button size="small" danger onClick={() => removeTriggerGuide(index)}>
                    删除
                  </Button>
                </div>
              ))}
              {(currentAIGuide.triggers || []).length === 0 && (
                <div style={{ color: '#999', fontSize: 12 }}>暂无触发器配置</div>
              )}
            </Form.Item>
          </Form>
        </div>
      </Modal>
    </>
  );
};