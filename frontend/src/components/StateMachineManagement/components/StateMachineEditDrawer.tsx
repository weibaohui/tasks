/**
 * StateMachine Edit Drawer Component
 */
import React, { useEffect } from 'react';
import { Drawer, Form, Input, Button, Space, message, Alert, Tabs, Select } from 'antd';
import { FileTextOutlined } from '@ant-design/icons';
import * as yaml from 'js-yaml';
import type { StateMachine, CreateStateMachineRequest, TransitionHook } from '../../../types/stateMachine';
import { stateMachineTemplates, type StateMachineTemplate } from './stateMachineTemplates';
import { StateMachineVisualEditor } from './StateMachineVisualEditor';
import { HookEditor } from './HookEditor';
import { AIGuideEditor } from './AIGuideEditor';

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
  id?: string;
  from: string;
  to: string;
  trigger: string;
  description: string;
  hooks: TransitionHook[];
}

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

  // Hook editor state
  const [hookModalOpen, setHookModalOpen] = React.useState(false);
  const [editingHook, setEditingHook] = React.useState<{ transitionIndex: number; hookIndex: number } | null>(null);
  const [currentHook, setCurrentHook] = React.useState<TransitionHook>({
    name: '',
    type: 'webhook',
    config: { url: '', method: 'POST', command: '' },
    retry: 0,
    timeout: 30,
  });
  const [selectedExample, setSelectedExample] = React.useState<any | null>(null);

  // AI Guide editor state
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

  // Template state
  const [selectedTemplate, setSelectedTemplate] = React.useState<StateMachineTemplate | null>(null);
  const pendingTemplateRef = React.useRef<StateMachineTemplate | null>(null);

  // Apply template
  const applyTemplate = (template: StateMachineTemplate) => {
    setSelectedTemplate(template);
    if (activeTab === 'yaml') {
      form.setFieldsValue({ name: template.name, description: template.description, config: template.yaml });
    } else {
      pendingTemplateRef.current = template;
    }
    setActiveTab('yaml');
    message.success(`已加载模板：${template.name}`);
  };

  // Listen for tab changes
  useEffect(() => {
    if (activeTab === 'yaml' && pendingTemplateRef.current) {
      const template = pendingTemplateRef.current;
      form.setFieldsValue({ name: template.name, description: template.description, config: template.yaml });
      pendingTemplateRef.current = null;
    }
  }, [activeTab, form]);

  // Sync YAML to visual when switching to visual tab
  useEffect(() => {
    if (activeTab !== 'visual') return;
    const configValue = form.getFieldValue('config') || '';
    const parsed = parseConfig(configValue);
    if (!parsed) return;
    if (parsed.states && Array.isArray(parsed.states)) {
      setVisualStates(parsed.states.map((s: YamlParsedState) => ({
        id: s.id,
        name: s.name,
        isFinal: s.is_final ?? s.isFinal ?? false,
        aiGuide: s.ai_guide ?? s.aiGuide ?? '',
        autoInit: s.auto_init ?? s.autoInit ?? '',
        successCriteria: s.success_criteria ?? s.successCriteria ?? '',
        failureCriteria: s.failure_criteria ?? s.failureCriteria ?? '',
        triggers: s.triggers || [],
      })));
    }
    if (parsed.transitions && Array.isArray(parsed.transitions)) {
      setVisualTransitions(parsed.transitions.map((t: any) => ({
        id: t.id || '',
        from: t.from,
        to: t.to,
        trigger: t.trigger,
        description: t.description || '',
        hooks: t.hooks || [],
      })));
    }
  }, [activeTab, form]);

  // Sync visual to YAML
  useEffect(() => {
    if (activeTab !== 'visual') return;
    const name = form.getFieldValue('name') || 'unnamed';
    const description = form.getFieldValue('description') || '';
    const config: Record<string, unknown> = {
      name,
      description,
      initial_state: visualStates.find((s) => !s.isFinal)?.id || visualStates[0]?.id || '',
      states: visualStates.map((s) => {
        const state: Record<string, unknown> = { id: s.id, name: s.name, is_final: s.isFinal };
        if (s.aiGuide) state.ai_guide = s.aiGuide;
        if (s.autoInit) state.auto_init = s.autoInit;
        if (s.successCriteria) state.success_criteria = s.successCriteria;
        if (s.failureCriteria) state.failure_criteria = s.failureCriteria;
        if (s.triggers && s.triggers.length > 0) state.triggers = s.triggers;
        return state;
      }),
      transitions: visualTransitions.map((t) => {
        const transition: Record<string, unknown> = {
          from: t.from,
          to: t.to,
          trigger: t.trigger,
          description: t.description,
        };
        if (t.id) transition.id = t.id;
        if (t.hooks.length > 0) transition.hooks = t.hooks;
        return transition;
      }),
    };
    form.setFieldsValue({ config: yaml.dump(config, { lineWidth: -1 }) });
  }, [visualStates, visualTransitions, activeTab, form]);

  // Initialize form when drawer opens
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
          setVisualStates(config.states.map((s: any) => ({
            id: s.id,
            name: s.name,
            isFinal: s.is_final,
            aiGuide: s.ai_guide ?? s.aiGuide ?? '',
            autoInit: s.auto_init ?? s.autoInit ?? '',
            successCriteria: s.success_criteria ?? s.successCriteria ?? '',
            failureCriteria: s.failure_criteria ?? s.failureCriteria ?? '',
            triggers: s.triggers || [],
          })));
          setVisualTransitions(config.transitions.map((t: any) => ({
            from: t.from,
            to: t.to,
            trigger: t.trigger,
            description: t.description || '',
            hooks: t.hooks || [],
          })));
        } catch {
          setVisualStates([]);
          setVisualTransitions([]);
        }
      } else {
        form.setFieldsValue({ name: '', description: '', config: DEFAULT_YAML });
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

  const parseConfig = (value: string): Record<string, unknown> | null => {
    try {
      return JSON.parse(value);
    } catch {
      try {
        return yaml.load(value) as Record<string, unknown>;
      } catch {
        return null;
      }
    }
  };

  const handleYamlChange = (value: string) => {
    const parsed = parseConfig(value);
    if (!parsed) return;
    if (parsed.states && Array.isArray(parsed.states)) {
      setVisualStates(parsed.states.map((s: YamlParsedState) => ({
        id: s.id,
        name: s.name,
        isFinal: s.is_final ?? s.isFinal ?? false,
        aiGuide: s.ai_guide ?? s.aiGuide ?? '',
        autoInit: s.auto_init ?? s.autoInit ?? '',
        successCriteria: s.success_criteria ?? s.successCriteria ?? '',
        failureCriteria: s.failure_criteria ?? s.failureCriteria ?? '',
        triggers: s.triggers || [],
      })));
    }
    if (parsed.transitions && Array.isArray(parsed.transitions)) {
      setVisualTransitions(parsed.transitions.map((t: any) => ({
        id: t.id || '',
        from: t.from,
        to: t.to,
        trigger: t.trigger,
        description: t.description || '',
        hooks: t.hooks || [],
      })));
    }
  };

  // Hook management
  const openAddHook = (transitionIndex: number) => {
    setEditingHook({ transitionIndex, hookIndex: -1 });
    setCurrentHook({ name: '', type: 'webhook', config: { url: '', method: 'POST' }, retry: 0, timeout: 30 });
    setHookModalOpen(true);
  };

  const openEditHook = (transitionIndex: number, hookIndex: number) => {
    const hook = visualTransitions[transitionIndex].hooks[hookIndex];
    setEditingHook({ transitionIndex, hookIndex });
    setCurrentHook({ name: hook.name, type: hook.type, config: { ...hook.config }, retry: hook.retry || 0, timeout: hook.timeout || 30 });
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
    if (!currentHook.name) { message.warning('请输入 Hook 名称'); return; }
    if (currentHook.type === 'webhook' && !currentHook.config.url) { message.warning('请输入 Webhook URL'); return; }
    if (currentHook.type === 'command' && !currentHook.config.command) { message.warning('请输入要执行的命令'); return; }
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

  const buildYamlFromVisual = (): string => {
    const name = form.getFieldValue('name') || 'unnamed';
    const description = form.getFieldValue('description') || '';
    const config: Record<string, unknown> = {
      name,
      description,
      initial_state: visualStates.find((s) => !s.isFinal)?.id || visualStates[0]?.id || '',
      states: visualStates.map((s) => {
        const state: Record<string, unknown> = { id: s.id, name: s.name, is_final: s.isFinal };
        if (s.aiGuide) state.ai_guide = s.aiGuide;
        if (s.autoInit) state.auto_init = s.autoInit;
        if (s.successCriteria) state.success_criteria = s.successCriteria;
        if (s.failureCriteria) state.failure_criteria = s.failureCriteria;
        if (s.triggers && s.triggers.length > 0) state.triggers = s.triggers;
        return state;
      }),
      transitions: visualTransitions.map((t) => {
        const transition: Record<string, unknown> = {
          from: t.from,
          to: t.to,
          trigger: t.trigger,
          description: t.description,
        };
        if (t.id) transition.id = t.id;
        if (t.hooks.length > 0) transition.hooks = t.hooks;
        return transition;
      }),
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
      await onSubmit({ name: values.name, description: values.description || '', config: configYaml });
    } catch (err) {
      if (err instanceof Error) message.error(err.message);
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
            <Button type="primary" onClick={handleSubmit} loading={saving}>保存</Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical">
          {!editing && (
            <Form.Item label="快速开始">
              <Space>
                <Select
                  placeholder="从模板创建..."
                  style={{ width: 200 }}
                  value={selectedTemplate?.id}
                  onChange={(value) => { const t = stateMachineTemplates.find((t) => t.id === value); if (t) applyTemplate(t); }}
                  options={stateMachineTemplates.map((t) => ({ value: t.id, label: <Space><FileTextOutlined />{t.name}</Space> }))}
                />
                <Button type="link" size="small" onClick={() => setSelectedTemplate(null)}>清空</Button>
              </Space>
            </Form.Item>
          )}
          <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入状态机名称' }]}>
            <Input placeholder="例如：需求流程" />
          </Form.Item>
          <Form.Item label="描述" name="description">
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
                  <StateMachineVisualEditor
                    visualStates={visualStates}
                    visualTransitions={visualTransitions}
                    onStatesChange={setVisualStates}
                    onTransitionsChange={setVisualTransitions}
                    onEditAIGuide={openEditAIGuide}
                    onOpenAddHook={openAddHook}
                    onOpenEditHook={openEditHook}
                    onRemoveHook={removeHook}
                  />
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
                    <Form.Item name="config" rules={[{ required: true, message: '请输入状态机配置' }]}>
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

      <HookEditor
        open={hookModalOpen}
        editing={editingHook}
        hook={currentHook}
        selectedExample={selectedExample}
        onSave={saveHook}
        onCancel={() => { setHookModalOpen(false); setSelectedExample(null); }}
        onChange={setCurrentHook}
        onSelectExample={(ex) => {
          if (!ex) { setSelectedExample(null); return; }
          setSelectedExample(ex);
          setCurrentHook({
            name: ex.name,
            type: ex.type,
            config: ex.type === 'webhook' ? { url: ex.config.url || '', method: ex.config.method || 'POST' } : { command: ex.config.command || '' },
            retry: ex.retry || 0,
            timeout: ex.timeout || 30,
          });
        }}
      />

      <AIGuideEditor
        open={aiGuideModalOpen}
        stateName={visualStates[editingStateIndex || 0]?.name || ''}
        state={currentAIGuide}
        onSave={saveAIGuide}
        onCancel={() => { setAiGuideModalOpen(false); setEditingStateIndex(null); }}
        onChange={setCurrentAIGuide}
        onAddTrigger={addTriggerGuide}
        onRemoveTrigger={removeTriggerGuide}
        onUpdateTrigger={updateTriggerGuide}
      />
    </>
  );
};
