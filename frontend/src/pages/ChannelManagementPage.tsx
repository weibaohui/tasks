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
import type { ColumnsType } from 'antd/es/table';
import { listAgents } from '../api/agentApi';
import { createChannel, deleteChannel, listChannels, listChannelTypes, updateChannel } from '../api/channelApi';
import { useAuthStore } from '../stores/authStore';
import type { Agent } from '../types/agent';
import type { Channel, ChannelTypeOption, CreateChannelRequest, UpdateChannelRequest } from '../types/channel';

type ChannelFormValues = {
  name: string;
  type: string;
  agent_code: string;
  is_active: boolean;
  allow_from: string[];
  config_json: string;
  feishu_app_id?: string;
  feishu_app_secret?: string;
};

/**
 * 解析 JSON 文本为对象（要求是 JSON Object）
 */
function parseJsonObject(jsonText: string): Record<string, unknown> {
  if (!jsonText.trim()) {
    return {};
  }
  const parsed = JSON.parse(jsonText) as unknown;
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('config must be an object');
  }
  return parsed as Record<string, unknown>;
}

/**
 * 将对象转换为用于表单编辑的 JSON 文本
 */
function toJsonText(value: Record<string, unknown> | undefined): string {
  if (!value || Object.keys(value).length === 0) {
    return '';
  }
  return JSON.stringify(value, null, 2);
}

/**
 * 根据表单值生成要提交的 config 对象
 */
function buildChannelConfig(values: ChannelFormValues, channelType: string): Record<string, unknown> {
  const config = parseJsonObject(values.config_json || '');
  if (channelType === 'feishu') {
    config.app_id = (values.feishu_app_id || '').trim();
    config.app_secret = (values.feishu_app_secret || '').trim();
  }
  return config;
}

export const ChannelManagementPage: React.FC = () => {
  const { user } = useAuthStore();
  const userCode = user?.user_code || '';
  const [items, setItems] = useState<Channel[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [channelTypes, setChannelTypes] = useState<ChannelTypeOption[]>([]);
  const [channelTypesLoading, setChannelTypesLoading] = useState(false);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<Channel | null>(null);
  const [form] = Form.useForm<ChannelFormValues>();
  const watchedType = Form.useWatch('type', form);

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
    } catch (_error) {
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
    } catch (_error) {
      setAgents([]);
    }
  }, [userCode]);

  const fetchChannelTypes = useCallback(async () => {
    if (!userCode) {
      setChannelTypes([]);
      return;
    }
    setChannelTypesLoading(true);
    try {
      const data = await listChannelTypes();
      setChannelTypes(data);
    } catch (_error) {
      message.error('获取渠道类型失败');
      setChannelTypes([]);
    } finally {
      setChannelTypesLoading(false);
    }
  }, [userCode]);

  const channelTypeOptions = useMemo(
    () => channelTypes.map((t) => ({ value: t.key, label: t.name ? `${t.name}（${t.key}）` : t.key })),
    [channelTypes],
  );

  /**
   * 删除渠道
   */
  const handleDelete = useCallback(async (id: string) => {
    try {
      await deleteChannel(id);
      message.success('删除成功');
      await fetchChannels();
    } catch (_error) {
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
      const channelType = editing ? editing.type : values.type;
      const config = buildChannelConfig(values, channelType);
      if (editing) {
        const req: UpdateChannelRequest = {
          name: values.name,
          config,
          allow_from: values.allow_from || [],
          is_active: values.is_active,
          agent_code: values.agent_code || undefined,
        };
        await updateChannel(editing.id, req);
        message.success('更新成功');
      } else {
        const req: CreateChannelRequest = {
          user_code: userCode,
          name: values.name,
          type: values.type,
          config,
          allow_from: values.allow_from || [],
          agent_code: values.agent_code,
        };
        await createChannel(req);
        message.success('创建成功');
      }
      setOpen(false);
      setEditing(null);
      form.resetFields();
      await fetchChannels();
    } catch (error) {
      if (error instanceof Error && error.message.includes('JSON')) {
        message.error('Config 不是合法 JSON');
      } else {
        message.error('保存失败');
      }
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
        width: 140,
        render: (v: string) => <Tag>{v}</Tag>,
      },
      {
        title: 'Agent Code',
        dataIndex: 'agent_code',
        key: 'agent_code',
        width: 180,
        render: (v: string) => (v ? <Tag color="geekblue">{v}</Tag> : <Tag>-</Tag>),
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
        render: (_: unknown, record: Channel) => (
          <Space>
            <Button
              type="primary"
              onClick={() => {
                setEditing(record);
                setOpen(true);
                form.setFieldsValue({
                  name: record.name,
                  type: record.type,
                  agent_code: record.agent_code,
                  is_active: record.is_active,
                  allow_from: record.allow_from || [],
                  config_json: toJsonText(record.config),
                  feishu_app_id: String((record.config as Record<string, unknown> | undefined)?.app_id || ''),
                  feishu_app_secret: String((record.config as Record<string, unknown> | undefined)?.app_secret || ''),
                });
              }}
            >
              编辑
            </Button>
            <Popconfirm title="确认删除该渠道？" onConfirm={() => handleDelete(record.id)}>
              <Button danger>删除</Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [form, handleDelete],
  );

  useEffect(() => {
    fetchAgents();
    fetchChannels();
    fetchChannelTypes();
  }, [fetchAgents, fetchChannels, fetchChannelTypes]);

  return (
    <div style={{ padding: 24 }}>
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
              onClick={() => {
                setEditing(null);
                setOpen(true);
                form.setFieldsValue({
                  name: '',
                  type: '',
                  agent_code: '',
                  is_active: true,
                  allow_from: [],
                  config_json: '',
                  feishu_app_id: '',
                  feishu_app_secret: '',
                });
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
        footer={null}
        width={920}
      >
        <Form layout="vertical" form={form} onFinish={handleSubmit}>
          <Space style={{ width: '100%' }} align="start">
            <div style={{ flex: 1 }}>
              <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]}>
                <Input />
              </Form.Item>
            </div>
            <div style={{ width: 220 }}>
              <Form.Item label="类型" name="type" rules={editing ? [] : [{ required: true, message: '请选择类型' }]}>
                <Select
                  disabled={!!editing}
                  loading={channelTypesLoading}
                  options={channelTypeOptions}
                  placeholder={channelTypesLoading ? '正在加载类型...' : '请选择渠道类型'}
                  notFoundContent={channelTypesLoading ? '正在加载...' : '暂无可用类型'}
                />
              </Form.Item>
            </div>
            <div style={{ width: 160 }}>
              <Form.Item label="启用" name="is_active" valuePropName="checked">
                <Switch checkedChildren="启用" unCheckedChildren="停用" />
              </Form.Item>
            </div>
          </Space>

          <Space style={{ width: '100%' }} align="start">
            <div style={{ flex: 1 }}>
              <Form.Item label="关联 Agent" name="agent_code">
                <Select
                  allowClear
                  placeholder="可选"
                  options={agents.map((a) => ({ label: `${a.name} (${a.agent_code})`, value: a.agent_code }))}
                />
              </Form.Item>
            </div>
            <div style={{ flex: 1 }}>
              <Form.Item label="Allow From（可多选/自定义）" name="allow_from">
                <Select mode="tags" placeholder="输入后回车添加" />
              </Form.Item>
            </div>
          </Space>

          {watchedType === 'feishu' ? (
            <Space style={{ width: '100%' }} align="start">
              <div style={{ flex: 1 }}>
                <Form.Item
                  label="飞书 App ID"
                  name="feishu_app_id"
                  rules={[{ required: true, message: '请输入飞书 App ID' }]}
                >
                  <Input placeholder="例如：cli_a93d6ef856781bc6" />
                </Form.Item>
              </div>
              <div style={{ flex: 1 }}>
                <Form.Item
                  label="飞书 App Secret"
                  name="feishu_app_secret"
                  rules={[{ required: true, message: '请输入飞书 App Secret' }]}
                >
                  <Input.Password placeholder="例如：xvijM8ZZqPIwNBho2dmflhNWHZNRMd51" />
                </Form.Item>
              </div>
            </Space>
          ) : null}

          <Form.Item label="高级配置（JSON）" name="config_json" tooltip='例如：{"token":"xxx","webhook":"https://..."}'>
            <Input.TextArea rows={8} placeholder="可选，必须是 JSON 对象格式" />
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
