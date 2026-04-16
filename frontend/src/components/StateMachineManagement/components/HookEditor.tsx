/**
 * StateMachine Hook Editor Component
 * Hook编辑Modal
 */
import React from 'react';
import { Modal, Form, Input, Select, Divider, Alert, Tag, Space, Collapse } from 'antd';
import { ThunderboltOutlined, CheckOutlined, InfoCircleOutlined } from '@ant-design/icons';
import type { TransitionHook } from '../../../types/stateMachine';
import { hookExamples, examplesByCategory, categoryNames, type HookExample } from './hookExamples';

interface HookEditorProps {
  open: boolean;
  editing: { transitionIndex: number; hookIndex: number } | null;
  hook: TransitionHook;
  selectedExample: HookExample | null;
  onSave: () => void;
  onCancel: () => void;
  onChange: (hook: TransitionHook) => void;
  onSelectExample: (example: HookExample | null) => void;
}

// 模板变量
const TEMPLATE_VARIABLES = [
  { key: 'requirement_id', label: 'requirement_id', desc: '需求ID' },
  { key: 'project_id', label: 'project_id', desc: '项目ID' },
  { key: 'state_machine_id', label: 'state_machine_id', desc: '状态机ID' },
  { key: 'from_state', label: 'from_state', desc: '源状态' },
  { key: 'to_state', label: 'to_state', desc: '目标状态' },
  { key: 'trigger', label: 'trigger', desc: '触发器' },
  { key: 'trigger_id', label: 'trigger_id', desc: '转换规则ID' },
  { key: 'hook_name', label: 'hook_name', desc: 'Hook名称' },
  { key: 'hook_type', label: 'hook_type', desc: 'Hook类型' },
];

const METADATA_VARIABLES = [
  { key: 'REQUIREMENT_ID', label: 'REQUIREMENT_ID', desc: '需求ID（环境变量格式）' },
  { key: 'PROJECT_ID', label: 'PROJECT_ID', desc: '项目ID（环境变量格式）' },
  { key: 'STATE_MACHINE_NAME', label: 'STATE_MACHINE_NAME', desc: '状态机名称' },
  { key: 'REQUIREMENT_TYPE', label: 'REQUIREMENT_TYPE', desc: '需求类型（normal/heartbeat）' },
  { key: 'REQUIREMENT_STATUS', label: 'REQUIREMENT_STATUS', desc: '需求当前状态' },
  { key: 'REQUIREMENT_TITLE', label: 'REQUIREMENT_TITLE', desc: '需求标题' },
];

const CLI_METADATA_VARIABLES = [
  { key: 'operator', label: 'operator', desc: '操作人（通过 --metadata 注入）' },
  { key: 'source', label: 'source', desc: '触发来源（通过 --metadata 注入）' },
];

export const HookEditor: React.FC<HookEditorProps> = ({
  open,
  editing,
  hook,
  selectedExample,
  onSave,
  onCancel,
  onChange,
  onSelectExample,
}) => {
  const filteredExamples = hookExamples.filter((e) => e.type === hook.type);

  const insertVariable = (field: 'url' | 'command' | 'heartbeat_id', variable: string) => {
    if (field === 'url') {
      const url = (hook.config.url as string) || '';
      onChange({ ...hook, config: { ...hook.config, url: url + `{{${variable}}}` } });
    } else if (field === 'command') {
      const cmd = (hook.config.command as string) || '';
      onChange({ ...hook, config: { ...hook.config, command: cmd + `{{${variable}}}` } });
    } else {
      const hbId = (hook.config.heartbeat_id as string) || '';
      onChange({ ...hook, config: { ...hook.config, heartbeat_id: hbId + `{{${variable}}}` } });
    }
  };

  return (
    <Modal
      title={editing && editing.hookIndex >= 0 ? '编辑 Hook' : '添加 Hook'}
      open={open}
      onOk={onSave}
      onCancel={onCancel}
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
            defaultActiveKey={['notification', 'deployment', 'heartbeat']}
            ghost
            style={{ background: 'transparent' }}
          >
            {(Object.keys(examplesByCategory) as Array<keyof typeof examplesByCategory>).map((category) => {
              const examples = examplesByCategory[category].filter((e) => e.type === hook.type);
              if (examples.length === 0) return null;
              return (
                <Collapse.Panel
                  key={category}
                  header={<span style={{ fontSize: 12 }}>{categoryNames[category]} ({examples.length})</span>}
                >
                  {examples.map((example) => (
                    <div
                      key={example.id}
                      onClick={() => onSelectExample(example)}
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
                  value={hook.name}
                  onChange={(e) => onChange({ ...hook, name: e.target.value })}
                  placeholder="例如：发送通知、执行部署"
                />
              </Form.Item>

              <Form.Item label="Hook 类型" required>
                <Select
                  value={hook.type}
                  onChange={(value) => {
                    onSelectExample(null);
                    onChange({
                      ...hook,
                      type: value,
                      config:
                        value === 'webhook'
                          ? { url: '', method: 'POST' }
                          : value === 'trigger_heartbeat'
                          ? { heartbeat_id: '' }
                          : { command: '' },
                    });
                  }}
                >
                  <Select.Option value="webhook">
                    <span>🌐 Webhook</span>
                  </Select.Option>
                  <Select.Option value="command">
                    <span>📦 命令执行</span>
                  </Select.Option>
                  <Select.Option value="trigger_heartbeat">
                    <span>❤️ 触发心跳</span>
                  </Select.Option>
                </Select>
              </Form.Item>
            </Form>

            <Divider style={{ margin: '8px 0' }} />

            {hook.type === 'webhook' && (
              <div>
                <Form layout="vertical">
                  <Form.Item label="Webhook URL" required>
                    <Input
                      value={(hook.config.url as string) || ''}
                      onChange={(e) => onChange({ ...hook, config: { ...hook.config, url: e.target.value } })}
                      placeholder="https://example.com/webhook"
                      addonAfter={<Tag color="blue" style={{ margin: 0 }}>可使用 {`{{variable}}`}</Tag>}
                    />
                  </Form.Item>
                  <Form.Item label="请求方法">
                    <Select
                      value={(hook.config.method as string) || 'POST'}
                      onChange={(value) => onChange({ ...hook, config: { ...hook.config, method: value } })}
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

            {hook.type === 'command' && (
              <div>
                <Form layout="vertical">
                  <Form.Item label="命令" required>
                    <Input.TextArea
                      value={(hook.config.command as string) || ''}
                      onChange={(e) => onChange({ ...hook, config: { ...hook.config, command: e.target.value } })}
                      placeholder="/bin/bash /scripts/deploy.sh {{requirement_id}}"
                      rows={3}
                    />
                  </Form.Item>
                </Form>
              </div>
            )}

            {hook.type === 'trigger_heartbeat' && (
              <div>
                <Form layout="vertical">
                  <Form.Item label="心跳 ID" required>
                    <Input
                      value={(hook.config.heartbeat_id as string) || ''}
                      onChange={(e) => onChange({ ...hook, config: { ...hook.config, heartbeat_id: e.target.value } })}
                      placeholder="hb-001 或 {{trigger_id}}-{{requirement_id}}"
                      addonAfter={<Tag color="blue" style={{ margin: 0 }}>可使用 {`{{variable}}`}</Tag>}
                    />
                  </Form.Item>
                </Form>
              </div>
            )}

            <Form layout="horizontal" style={{ display: 'flex', gap: 16 }}>
              <Form.Item label="超时时间（秒）" style={{ flex: 1 }}>
                <Input
                  type="number"
                  value={hook.timeout || 30}
                  onChange={(e) => onChange({ ...hook, timeout: parseInt(e.target.value) || 30 })}
                  placeholder="30"
                />
              </Form.Item>
              <Form.Item label="重试次数" style={{ flex: 1 }}>
                <Input
                  type="number"
                  value={hook.retry || 0}
                  onChange={(e) => onChange({ ...hook, retry: parseInt(e.target.value) || 0 })}
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
                  <div>
                    <p style={{ marginBottom: 8, fontWeight: 500 }}>基础变量（点击插入）：</p>
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                      {TEMPLATE_VARIABLES.map((v) => (
                        <Tag
                          key={v.key}
                          color="cyan"
                          style={{ cursor: 'pointer' }}
                          onClick={() => insertVariable(hook.type === 'webhook' ? 'url' : hook.type === 'trigger_heartbeat' ? 'heartbeat_id' : 'command', v.key)}
                        >
                          {`{{${v.key}}}`}
                        </Tag>
                      ))}
                    </div>
                  </div>
                  <div>
                    <p style={{ marginBottom: 8, fontWeight: 500 }}>需求元数据（由分身Agent自动注入）：</p>
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                      {METADATA_VARIABLES.map((v) => (
                        <Tag
                          key={v.key}
                          color="blue"
                          style={{ cursor: 'pointer' }}
                          onClick={() => insertVariable(hook.type === 'webhook' ? 'url' : hook.type === 'trigger_heartbeat' ? 'heartbeat_id' : 'command', v.key)}
                        >
                          {`{{${v.key}}}`}
                        </Tag>
                      ))}
                    </div>
                  </div>
                  <div>
                    <p style={{ marginBottom: 8, fontWeight: 500 }}>CLI 注入的自定义变量（通过 --metadata 参数）：</p>
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                      {CLI_METADATA_VARIABLES.map((v) => (
                        <Tag
                          key={v.key}
                          color="orange"
                          style={{ cursor: 'pointer' }}
                          onClick={() => insertVariable(hook.type === 'webhook' ? 'url' : hook.type === 'trigger_heartbeat' ? 'heartbeat_id' : 'command', v.key)}
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
  );
};
