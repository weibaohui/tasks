/**
 * Agents 管理页面
 * 支持 Agent 的新增、编辑、删除、启用/停用
 */
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Card, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Table, Tag, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { createAgent, deleteAgent, listAgents, updateAgent } from '../api/agentApi';
import { useAuthStore } from '../stores/authStore';
import type { Agent, CreateAgentRequest, UpdateAgentRequest } from '../types/agent';

type AgentFormValues = {
  name: string;
  description: string;
  identity_content: string;
  soul_content: string;
  agents_content: string;
  user_content: string;
  tools_content: string;
  model: string;
  max_tokens: number;
  temperature: number;
  max_iterations: number;
  history_messages: number;
  skills_list: string[];
  tools_list: string[];
  is_default: boolean;
  is_active: boolean;
  enable_thinking_process: boolean;
};

export const AgentManagementPage: React.FC = () => {
  const { user } = useAuthStore();
  const userCode = user?.user_code || '';
  const [items, setItems] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<Agent | null>(null);
  const [form] = Form.useForm<AgentFormValues>();

  /**
   * 拉取 Agent 列表
   */
  const fetchList = useCallback(async () => {
    if (!userCode) {
      setItems([]);
      return;
    }
    setLoading(true);
    try {
      const data = await listAgents(userCode);
      setItems(data);
    } catch (_error) {
      message.error('获取 Agent 列表失败');
    } finally {
      setLoading(false);
    }
  }, [userCode]);

  /**
   * 删除 Agent
   */
  const handleDelete = useCallback(async (id: string) => {
    try {
      await deleteAgent(id);
      message.success('删除成功');
      await fetchList();
    } catch (_error) {
      message.error('删除失败');
    }
  }, [fetchList]);

  /**
   * 保存 Agent（创建/更新）
   */
  const handleSubmit = useCallback(async (values: AgentFormValues) => {
    if (!userCode) {
      message.error('未获取到用户信息，请重新登录');
      return;
    }
    setSaving(true);
    try {
      if (editing) {
        const req: UpdateAgentRequest = {
          name: values.name,
          description: values.description,
          identity_content: values.identity_content,
          soul_content: values.soul_content,
          agents_content: values.agents_content,
          user_content: values.user_content,
          tools_content: values.tools_content,
          model: values.model,
          max_tokens: values.max_tokens,
          temperature: values.temperature,
          max_iterations: values.max_iterations,
          history_messages: values.history_messages,
          skills_list: values.skills_list || [],
          tools_list: values.tools_list || [],
          is_default: values.is_default,
          is_active: values.is_active,
          enable_thinking_process: values.enable_thinking_process,
        };
        await updateAgent(editing.id, req);
        message.success('更新成功');
      } else {
        const req: CreateAgentRequest = {
          user_code: userCode,
          name: values.name,
          description: values.description,
          identity_content: values.identity_content,
          soul_content: values.soul_content,
          agents_content: values.agents_content,
          user_content: values.user_content,
          tools_content: values.tools_content,
          model: values.model,
          max_tokens: values.max_tokens,
          temperature: values.temperature,
          max_iterations: values.max_iterations,
          history_messages: values.history_messages,
          skills_list: values.skills_list || [],
          tools_list: values.tools_list || [],
          is_default: values.is_default,
          enable_thinking_process: values.enable_thinking_process,
        };
        await createAgent(req);
        message.success('创建成功');
      }
      setOpen(false);
      setEditing(null);
      form.resetFields();
      await fetchList();
    } catch (_error) {
      message.error('保存失败');
    } finally {
      setSaving(false);
    }
  }, [editing, fetchList, form, userCode]);

  const columns: ColumnsType<Agent> = useMemo(
    () => [
      {
        title: '名称',
        dataIndex: 'name',
        key: 'name',
        ellipsis: true,
      },
      {
        title: 'Code',
        dataIndex: 'agent_code',
        key: 'agent_code',
        width: 180,
        render: (v: string) => <Tag color="blue">{v}</Tag>,
      },
      {
        title: '模型',
        dataIndex: 'model',
        key: 'model',
        width: 180,
        ellipsis: true,
      },
      {
        title: '默认',
        dataIndex: 'is_default',
        key: 'is_default',
        width: 80,
        render: (v: boolean) => (v ? <Tag color="green">是</Tag> : <Tag>否</Tag>),
      },
      {
        title: '启用',
        dataIndex: 'is_active',
        key: 'is_active',
        width: 80,
        render: (v: boolean) => (v ? <Tag color="green">启用</Tag> : <Tag color="red">停用</Tag>),
      },
      {
        title: '操作',
        key: 'action',
        width: 200,
        render: (_: unknown, record: Agent) => (
          <Space>
            <Button
              type="primary"
              onClick={() => {
                setEditing(record);
                setOpen(true);
                form.setFieldsValue({
                  name: record.name,
                  description: record.description,
                  identity_content: record.identity_content,
                  soul_content: record.soul_content,
                  agents_content: record.agents_content,
                  user_content: record.user_content,
                  tools_content: record.tools_content,
                  model: record.model,
                  max_tokens: record.max_tokens,
                  temperature: record.temperature,
                  max_iterations: record.max_iterations,
                  history_messages: record.history_messages,
                  skills_list: record.skills_list || [],
                  tools_list: record.tools_list || [],
                  is_default: record.is_default,
                  is_active: record.is_active,
                  enable_thinking_process: record.enable_thinking_process,
                });
              }}
            >
              编辑
            </Button>
            <Popconfirm title="确认删除该 Agent？" onConfirm={() => handleDelete(record.id)}>
              <Button danger>删除</Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [form, handleDelete],
  );

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  return (
    <div style={{ padding: 24 }}>
      <Card
        title={`Agents 管理 (${items.length})`}
        extra={
          <Space>
            <Button onClick={() => fetchList()}>刷新</Button>
            <Button
              type="primary"
              onClick={() => {
                setEditing(null);
                setOpen(true);
                form.setFieldsValue({
                  name: '',
                  description: '',
                  identity_content: '',
                  soul_content: '',
                  agents_content: '',
                  user_content: '',
                  tools_content: '',
                  model: '',
                  max_tokens: 4096,
                  temperature: 0.2,
                  max_iterations: 8,
                  history_messages: 20,
                  skills_list: [],
                  tools_list: [],
                  is_default: false,
                  is_active: true,
                  enable_thinking_process: false,
                });
              }}
            >
              新建 Agent
            </Button>
          </Space>
        }
      >
        <Table<Agent> rowKey="id" loading={loading} dataSource={items} columns={columns} />
      </Card>

      <Modal
        title={editing ? '编辑 Agent' : '新建 Agent'}
        open={open}
        onCancel={() => {
          setOpen(false);
          setEditing(null);
        }}
        footer={null}
        width={980}
      >
        <Form layout="vertical" form={form} onFinish={handleSubmit}>
          <Space style={{ width: '100%' }} align="start">
            <div style={{ flex: 1 }}>
              <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]}>
                <Input />
              </Form.Item>
            </div>
            <div style={{ flex: 1 }}>
              <Form.Item label="模型" name="model" rules={[{ required: true, message: '请输入模型' }]}>
                <Input placeholder="例如：gpt-4o-mini / llama3" />
              </Form.Item>
            </div>
          </Space>

          <Form.Item label="描述" name="description">
            <Input.TextArea rows={2} />
          </Form.Item>

          <Space style={{ width: '100%' }} align="start">
            <div style={{ width: 180 }}>
              <Form.Item label="Max Tokens" name="max_tokens">
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </div>
            <div style={{ width: 180 }}>
              <Form.Item label="Temperature" name="temperature">
                <InputNumber min={0} max={2} step={0.1} style={{ width: '100%' }} />
              </Form.Item>
            </div>
            <div style={{ width: 180 }}>
              <Form.Item label="最大迭代" name="max_iterations">
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </div>
            <div style={{ width: 180 }}>
              <Form.Item label="历史消息数" name="history_messages">
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </div>
            <div style={{ width: 140 }}>
              <Form.Item label="设为默认" name="is_default" valuePropName="checked">
                <Switch checkedChildren="是" unCheckedChildren="否" />
              </Form.Item>
            </div>
            <div style={{ width: 140 }}>
              <Form.Item label="启用" name="is_active" valuePropName="checked">
                <Switch checkedChildren="启用" unCheckedChildren="停用" />
              </Form.Item>
            </div>
          </Space>

          <Space style={{ width: '100%' }} align="start">
            <div style={{ flex: 1 }}>
              <Form.Item label="Skills（可多选/自定义）" name="skills_list">
                <Select mode="tags" placeholder="输入后回车添加" />
              </Form.Item>
            </div>
            <div style={{ flex: 1 }}>
              <Form.Item label="Tools（可多选/自定义）" name="tools_list">
                <Select mode="tags" placeholder="输入后回车添加" />
              </Form.Item>
            </div>
            <div style={{ width: 200 }}>
              <Form.Item label="展示思考过程" name="enable_thinking_process" valuePropName="checked">
                <Switch checkedChildren="开启" unCheckedChildren="关闭" />
              </Form.Item>
            </div>
          </Space>

          <Form.Item label="Identity Content" name="identity_content">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item label="Soul Content" name="soul_content">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item label="Agents Content" name="agents_content">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item label="User Content" name="user_content">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item label="Tools Content" name="tools_content">
            <Input.TextArea rows={3} />
          </Form.Item>

          <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button
              onClick={() => {
                setOpen(false);
                setEditing(null);
              }}
            >
              取消
            </Button>
            <Button type="primary" htmlType="submit" loading={saving}>
              保存
            </Button>
          </Space>
        </Form>
      </Modal>
    </div>
  );
};
