/**
 * 渠道管理页面
 * 支持 Channel 的新增、编辑、删除、启用/停用
 */
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  message,
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { listAgents } from '../api/agentApi';
import { createChannel, deleteChannel, listChannels, updateChannel } from '../api/channelApi';
import { useAuthStore } from '../stores/authStore';
import type { Agent } from '../types/agent';
import type { Channel } from '../types/channel';
import { ChannelTypeLabels } from '../types/channel';
import { ActionGroup } from "@/components/ActionGroup";

type ChannelType = 'feishu' | 'dingtalk' | 'matrix' | 'websocket';

type ChannelFormValues = {
  name: string;
  type: ChannelType;
  agent_code: string;
  is_active: boolean;
  allow_from: string;
  config: {
    app_id?: string;
    app_secret?: string;
    encrypt_key?: string;
    verification_token?: string;
    client_id?: string;
    client_secret?: string;
    homeserver?: string;
    user_id?: string;
    token?: string;
    addr?: string;
    path?: string;
  };
};

/**
 * 根据渠道类型获取配置字段
 */
function getConfigFields(type: ChannelType | undefined) {
  switch (type) {
    case 'feishu':
      return (
        <>
          <Form.Item name={['config', 'app_id']} label="App ID" rules={[{ required: true, message: '请输入 App ID' }]}>
            <Input placeholder="例如：cli_a93d6ef856781bc6" />
          </Form.Item>
          <Form.Item name={['config', 'app_secret']} label="App Secret" rules={[{ required: true, message: '请输入 App Secret' }]}>
            <Input.Password placeholder="例如：xvijM8ZZqPIwNBho2dmflhNWHZNRMd51" />
          </Form.Item>
          <Form.Item name={['config', 'encrypt_key']} label="Encrypt Key">
            <Input placeholder="加密 Key（可选）" />
          </Form.Item>
          <Form.Item name={['config', 'verification_token']} label="Verification Token">
            <Input placeholder="验证 Token（可选）" />
          </Form.Item>
        </>
      );
    case 'dingtalk':
      return (
        <>
          <Form.Item name={['config', 'client_id']} label="Client ID" rules={[{ required: true, message: '请输入 Client ID' }]}>
            <Input placeholder="钉钉应用的 Client ID" />
          </Form.Item>
          <Form.Item name={['config', 'client_secret']} label="Client Secret" rules={[{ required: true, message: '请输入 Client Secret' }]}>
            <Input.Password placeholder="钉钉应用的 Client Secret" />
          </Form.Item>
        </>
      );
    case 'matrix':
      return (
        <>
          <Form.Item name={['config', 'homeserver']} label="Homeserver" rules={[{ required: true, message: '请输入 Homeserver' }]}>
            <Input placeholder="https://matrix.org" />
          </Form.Item>
          <Form.Item name={['config', 'user_id']} label="User ID" rules={[{ required: true, message: '请输入 User ID' }]}>
            <Input placeholder="@user:matrix.org" />
          </Form.Item>
          <Form.Item name={['config', 'token']} label="Token" rules={[{ required: true, message: '请输入 Token' }]}>
            <Input.Password placeholder="访问 Token" />
          </Form.Item>
        </>
      );
    case 'websocket':
      return (
        <>
          <Form.Item name={['config', 'addr']} label="监听地址" initialValue=":8080">
            <Input placeholder=":8080" />
          </Form.Item>
          <Form.Item name={['config', 'path']} label="路径" initialValue="/ws">
            <Input placeholder="/ws" />
          </Form.Item>
        </>
      );
    default:
      return null;
  }
}

export const ChannelManagementPage: React.FC = () => {
  const { user } = useAuthStore();
  const userCode = user?.user_code || '';
  const [items, setItems] = useState<Channel[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<Channel | null>(null);
  const [form] = Form.useForm<ChannelFormValues>();

  /**
   * 拉取渠道列表
   */
  const fetchChannels = useCallback(async () => {
    if (!userCode) {
      setItems([]);
      return;
    }
    setLoading(true);
    try {
      const data = await listChannels(userCode);
      setItems(data);
    } catch {
      message.error('获取渠道列表失败');
    } finally {
      setLoading(false);
    }
  }, [userCode]);

  /**
   * 拉取 Agent 列表（用于选择 agent_code）
   */
  const fetchAgents = useCallback(async () => {
    if (!userCode) {
      setAgents([]);
      return;
    }
    try {
      const data = await listAgents(userCode);
      setAgents(data);
    } catch {
      setAgents([]);
    }
  }, [userCode]);

  /**
   * 删除渠道
   */
  const handleDelete = useCallback(async (id: string) => {
    try {
      await deleteChannel(id);
      message.success('删除成功');
      await fetchChannels();
    } catch {
      message.error('删除失败');
    }
  }, [fetchChannels]);

  /**
   * 保存渠道（创建/更新）
   */
  const handleSubmit = useCallback(async (values: ChannelFormValues) => {
    if (!userCode) {
      message.error('未获取到用户信息，请重新登录');
      return;
    }
    setSaving(true);
    try {
      const config = values.config || {};
      const allowFrom = values.allow_from
        ? values.allow_from.split('\n').map((s) => s.trim()).filter(Boolean)
        : [];

      if (editing) {
        await updateChannel(editing.id, {
          name: values.name,
          config,
          allow_from: allowFrom,
          is_active: values.is_active,
          agent_code: values.agent_code || undefined,
        });
        message.success('更新成功');
      } else {
        await createChannel({
          user_code: userCode,
          name: values.name,
          type: values.type,
          config,
          allow_from: allowFrom,
          agent_code: values.agent_code,
        });
        message.success('创建成功');
      }
      setOpen(false);
      setEditing(null);
      form.resetFields();
      await fetchChannels();
    } catch {
      message.error('保存失败');
    } finally {
      setSaving(false);
    }
  }, [editing, fetchChannels, form, userCode]);

  const columns: ColumnsType<Channel> = useMemo(
    () => [
      {
        title: '名称',
        dataIndex: 'name',
        key: 'name',
        ellipsis: true,
      },
      {
        title: 'Code',
        dataIndex: 'channel_code',
        key: 'channel_code',
        width: 180,
        render: (v: string) => <Tag color="blue">{v}</Tag>,
      },
      {
        title: '类型',
        dataIndex: 'type',
        key: 'type',
        width: 120,
        render: (v: string) => <Tag>{ChannelTypeLabels[v] || v}</Tag>,
      },
      {
        title: '绑定 Agent',
        key: 'agent',
        width: 150,
        render: (_: unknown, record: Channel) => {
          const agent = agents.find((a) => a.agent_code === record.agent_code);
          return agent ? (
            <Tag color="geekblue">{agent.name}</Tag>
          ) : (
            <span style={{ color: '#999' }}>未绑定</span>
          );
        },
      },
      {
        title: '状态',
        key: 'status',
        width: 80,
        render: (_: unknown, record: Channel) => (
          <Tag color={record.is_active ? 'success' : 'default'}>
            {record.is_active ? '启用' : '禁用'}
          </Tag>
        ),
      },
      {
        title: '操作',
        key: 'action',
        render: (_: unknown, record: Channel) => (
          <ActionGroup>
            <Button
              icon={<EditOutlined />}
              onClick={() => {
                setEditing(record);
                const config = record.config || {};
                const allowFrom = record.allow_from?.join('\n') || '';
                form.setFieldsValue({
                  name: record.name,
                  type: record.type as ChannelType,
                  agent_code: record.agent_code,
                  is_active: record.is_active,
                  allow_from: allowFrom,
                  config: {
                    app_id: String(config.app_id || ''),
                    app_secret: String(config.app_secret || ''),
                    encrypt_key: String(config.encrypt_key || ''),
                    verification_token: String(config.verification_token || ''),
                    client_id: String(config.client_id || ''),
                    client_secret: String(config.client_secret || ''),
                    homeserver: String(config.homeserver || ''),
                    user_id: String(config.user_id || ''),
                    token: String(config.token || ''),
                    addr: String(config.addr || ''),
                    path: String(config.path || ''),
                  },
                });
                setOpen(true);
              }} type="link" size="small" style={{ padding: 0 }}
            >
              编辑
            </Button>
            <Popconfirm
              title="确认删除"
              description="删除后将无法恢复，是否继续？"
              onConfirm={() => handleDelete(record.id)}
            >
              <Button danger icon={<DeleteOutlined />} type="link" size="small" style={{ padding: 0 }}>
                删除
              </Button>
            </Popconfirm>
          </ActionGroup>
        ),
          width: 100,
          fixed: 'left' as const
    },
    ],
    [agents, form, handleDelete],
  );

  useEffect(() => {
    fetchAgents();
    fetchChannels();
  }, [fetchAgents, fetchChannels]);

  return (
    <div style={{ padding: 0 }}>
      <Card
        title={`渠道管理 (${items.length})`}
        extra={
          <Space>
            <Button
              onClick={() => {
                fetchAgents();
                fetchChannels();
              }}
            >
              刷新
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                setEditing(null);
                form.resetFields();
                form.setFieldsValue({ is_active: true });
                setOpen(true);
              }}
            >
              新建渠道
            </Button>
          </Space>
        }
      >
        <Table<Channel> rowKey="id" loading={loading} dataSource={items} columns={columns} />
      </Card>

      <Modal
        title={editing ? '编辑渠道' : '新建渠道'}
        open={open}
        onCancel={() => {
          setOpen(false);
          setEditing(null);
        }}
        onOk={() => form.submit()}
        confirmLoading={saving}
        width={600}
      >
        <Form layout="vertical" form={form} onFinish={handleSubmit}>
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="渠道名称" />
          </Form.Item>

          <Form.Item
            name="type"
            label="类型"
            rules={editing ? [] : [{ required: true, message: '请选择类型' }]}
          >
            <Select placeholder="选择渠道类型" disabled={!!editing}>
              {Object.entries(ChannelTypeLabels).map(([key, label]) => (
                <Select.Option key={key} value={key}>
                  {label}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="agent_code" label="绑定 Agent">
            <Select placeholder="选择要绑定的 Agent" allowClear>
              {agents.map((agent) => (
                <Select.Option key={agent.agent_code} value={agent.agent_code}>
                  {agent.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="allow_from" label="白名单用户">
            <Input.TextArea rows={3} placeholder="每行一个用户ID，留空表示允许所有用户" />
          </Form.Item>

          <Form.Item noStyle shouldUpdate={(prev, curr) => prev.type !== curr.type}>
            {({ getFieldValue }) => getConfigFields(getFieldValue('type'))}
          </Form.Item>

          <Form.Item name="is_active" valuePropName="checked" initialValue={true}>
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};
