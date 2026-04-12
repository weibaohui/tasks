/**
 * StateMachine Visual Editor Component
 * 可视化状态和转换编辑
 */
import React from 'react';
import { Table, Button, Tag } from 'antd';
import { PlusOutlined, EditOutlined } from '@ant-design/icons';
import type { TransitionHook } from '../../../types/stateMachine';

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

interface TransitionWithHooks {
  from: string;
  to: string;
  trigger: string;
  description: string;
  hooks: TransitionHook[];
}

interface StateMachineVisualEditorProps {
  visualStates: StateWithAIGuide[];
  visualTransitions: TransitionWithHooks[];
  onStatesChange: (states: StateWithAIGuide[]) => void;
  onTransitionsChange: (transitions: TransitionWithHooks[]) => void;
  onEditAIGuide: (stateIndex: number) => void;
  onOpenAddHook: (transitionIndex: number) => void;
  onOpenEditHook: (transitionIndex: number, hookIndex: number) => void;
  onRemoveHook: (transitionIndex: number, hookIndex: number) => void;
}

export const StateMachineVisualEditor: React.FC<StateMachineVisualEditorProps> = ({
  visualStates,
  visualTransitions,
  onStatesChange,
  onTransitionsChange,
  onEditAIGuide,
  onOpenAddHook,
  onOpenEditHook,
  onRemoveHook,
}) => {
  const handleVisualStateChange = (
    index: number,
    field: 'id' | 'name' | 'isFinal',
    value: string | boolean,
  ) => {
    const updated = [...visualStates];
    updated[index] = { ...updated[index], [field]: value };
    onStatesChange(updated);
  };

  const addState = () => {
    onStatesChange([
      ...visualStates,
      { id: `state_${visualStates.length + 1}`, name: `状态${visualStates.length + 1}`, isFinal: false },
    ]);
  };

  const removeState = (index: number) => {
    onStatesChange(visualStates.filter((_, i) => i !== index));
  };

  const addTransition = () => {
    onTransitionsChange([
      ...visualTransitions,
      { from: visualStates[0]?.id || '', to: visualStates[0]?.id || '', trigger: '', description: '', hooks: [] },
    ]);
  };

  const removeTransition = (index: number) => {
    onTransitionsChange(visualTransitions.filter((_, i) => i !== index));
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
      {/* 状态列表 */}
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
            {
              title: '操作',
              key: 'action',
              width: 100,
              fixed: 'left' as const,
              render: (_, __, index: number) => (
                <Button danger onClick={() => removeState(index)} type="link" size="small" style={{ padding: 0 }}>
                  删除
                </Button>
              ),
            },
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
                  onChange={(e) => handleVisualStateChange(index, 'isFinal', e.target.checked)}
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
                  onClick={() => onEditAIGuide(index)}
                >
                  {record.aiGuide ? '已配置' : '配置'}
                </Button>
              ),
            },
          ]}
        />
      </div>

      {/* 转换规则 */}
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
              title: '操作',
              key: 'action',
              width: 100,
              fixed: 'left' as const,
              render: (_, __, index: number) => (
                <Button danger onClick={() => removeTransition(index)} type="link" size="small" style={{ padding: 0 }}>
                  删除
                </Button>
              ),
            },
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
                        onRemoveHook(transitionIndex, hookIndex);
                      }}
                      icon={<EditOutlined style={{ cursor: 'pointer' }} />}
                      onClick={() => onOpenEditHook(transitionIndex, hookIndex)}
                      color={hook.type === 'command' ? 'orange' : 'blue'}
                    >
                      {hook.type === 'command' ? '📦' : '🌐'} {hook.name}
                    </Tag>
                  ))}
                  <Button
                    size="small"
                    type="dashed"
                    icon={<PlusOutlined />}
                    onClick={() => onOpenAddHook(transitionIndex)}
                  >
                    添加 Hook
                  </Button>
                </div>
              ),
            },
          ]}
        />
      </div>
    </div>
  );
};
