/**
 * 会话管理页面
 * 支持 Session 的创建、查看元数据、更新元数据、删除
 */
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Drawer,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { listAgents } from '../api/agentApi';
import { listChannels } from '../api/channelApi';
import { createSession, deleteSession, getSessionMetadata, listUserSessions, updateSessionMetadata } from '../api/sessionApi';
import { useAuthStore } from '../stores/authStore';
import type { Agent } from '../types/agent';
import type { Channel } from '../types/channel';
import type { CreateSessionRequest, Session } from '../types/session';
import { ActionGroup } from "@/components/ActionGroup";

type CreateSessionFormValues = {
  agent_code: string;
  channel_code: string;
  session_key: string;
  external_id: string;
  metadata_json: string;
};

type MetadataFormValues = {
  metadata_json: string;
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
    throw new Error('metadata must be an object');
  }
  return parsed as Record<string, unknown>;
}

/**
 * 将对象转换为用于展示/编辑的 JSON 文本
 */
function toJsonText(value: Record<string, unknown> | undefined): string {
  if (!value || Object.keys(value).length === 0) {
    return '';
  }
  return JSON.stringify(value, null, 2);
}

export const SessionManagementPage: React.FC = () => {
  const { user } = useAuthStore();
  const userCode = user?.user_code || '';
  const [items, setItems] = useState<Session[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [editingMetadata, setEditingMetadata] = useState<Session | null>(null);
  const [metadataDrawer, setMetadataDrawer] = useState<Session | null>(null);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm<CreateSessionFormValues>();
  const [metadataForm] = Form.useForm<MetadataFormValues>();

  /**
   * 拉取会话列表
   */
  const fetchList = useCallback(async () => {
    if (!userCode) {
      setItems([]);
      return;
    }
    setLoading(true);
    try {
      const data = await listUserSessions(userCode);
      setItems(data);
    } catch (_error) {
      message.error('获取会话列表失败');
    } finally {
      setLoading(false);
    }
  }, [userCode]);

  /**
   * 拉取依赖数据（Agents/Channels）
   */
  const fetchRefs = useCallback(async () => {
    if (!userCode) {
      setAgents([]);
      setChannels([]);
      return;
    }
    try {
      const [agentList, channelList] = await Promise.all([listAgents(userCode), listChannels(userCode)]);
      setAgents(agentList);
      setChannels(channelList);
    } catch (_error) {
      setAgents([]);
      setChannels([]);
    }
  }, [userCode]);

  /**
   * 删除会话
   */
  const handleDelete = useCallback(async (sessionKey: string) => {
    try {
      await deleteSession(sessionKey);
      message.success('删除成功');
      await fetchList();
    } catch (_error) {
      message.error('删除失败');
    }
  }, [fetchList]);

  /**
   * 创建会话
   */
  const handleCreate = useCallback(async (values: CreateSessionFormValues) => {
    if (!userCode) {
      message.error('未获取到用户信息，请重新登录');
      return;
    }
    setSaving(true);
    try {
      const metadata = parseJsonObject(values.metadata_json);
      const req: CreateSessionRequest = {
        user_code: userCode,
        agent_code: values.agent_code,
        channel_code: values.channel_code,
        session_key: values.session_key,
        external_id: values.external_id,
        metadata,
      };
      await createSession(req);
      message.success('创建成功');
      setOpen(false);
      form.resetFields();
      await fetchList();
    } catch (error) {
      if (error instanceof Error && error.message.includes('JSON')) {
        message.error('Metadata 不是合法 JSON');
      } else {
        message.error('创建失败');
      }
    } finally {
      setSaving(false);
    }
  }, [fetchList, form, userCode]);

  /**
   * 打开元数据抽屉
   */
  const openMetadataDrawer = useCallback(async (session: Session) => {
    try {
      const metadata = await getSessionMetadata(session.session_key);
      setMetadataDrawer({ ...session, metadata });
    } catch (_error) {
      setMetadataDrawer(session);
    }
  }, []);

  /**
   * 保存元数据
   */
  const handleSaveMetadata = useCallback(async (values: MetadataFormValues) => {
    if (!editingMetadata) {
      return;
    }
    setSaving(true);
    try {
      const metadata = parseJsonObject(values.metadata_json);
      await updateSessionMetadata(editingMetadata.session_key, metadata);
      message.success('更新成功');
      setEditingMetadata(null);
      await fetchList();
      if (metadataDrawer && metadataDrawer.session_key === editingMetadata.session_key) {
        setMetadataDrawer({ ...metadataDrawer, metadata });
      }
    } catch (error) {
      if (error instanceof Error && error.message.includes('JSON')) {
        message.error('Metadata 不是合法 JSON');
      } else {
        message.error('更新失败');
      }
    } finally {
      setSaving(false);
    }
  }, [editingMetadata, fetchList, metadataDrawer]);

  const columns: ColumnsType<Session> = useMemo(
    () => [
          {
                  title: '操作',
                  key: 'action',
                  render: (_: unknown, record: Session) => (
                    <ActionGroup>
                      <Button onClick={() => openMetadataDrawer(record)} type="link" size="small" style={{ padding: 0 }}>元数据</Button>
                      <Button
                        onClick={() => {
                          setEditingMetadata(record);
                          metadataForm.setFieldsValue({ metadata_json: toJsonText(record.metadata) });
                        }} type="link" size="small" style={{ padding: 0 }}
                      >
                        编辑数据</Button>
                      <Popconfirm title="确认删除该会话？" onConfirm={() => handleDelete(record.session_key)}>
                        <Button danger type="link" size="small" style={{ padding: 0 }}>删除</Button>
                      </Popconfirm>
                    </ActionGroup>
                  ),
                    width: 100,
                    fixed: 'left' as const
              },
        {
        title: 'Session Key',
        dataIndex: 'session_key',
        key: 'session_key',
        ellipsis: true,
        render: (v: string) => <Tag color="blue">{v}</Tag>,
      },
      {
        title: 'Channel',
        dataIndex: 'channel_code',
        key: 'channel_code',
        width: 180,
        render: (v: string) => (v ? <Tag color="geekblue">{v}</Tag> : <Tag>-</Tag>),
      },
      {
        title: 'Agent',
        dataIndex: 'agent_code',
        key: 'agent_code',
        width: 180,
        render: (v: string) => (v ? <Tag>{v}</Tag> : <Tag>-</Tag>),
      },
      {
        title: 'External ID',
        dataIndex: 'external_id',
        key: 'external_id',
        width: 160,
        ellipsis: true,
      },
      {
        title: '最近活跃',
        dataIndex: 'last_active_at',
        key: 'last_active_at',
        width: 140,
        render: (v: number | null) => (v ? new Date(v).toLocaleString() : '-'),
      }
    ],
    [handleDelete, metadataForm, openMetadataDrawer],
  );

  useEffect(() => {
    fetchRefs();
    fetchList();
  }, [fetchList, fetchRefs]);

  return (
    <div style={{ padding: 0 }}>
      <Card
        title={`会话管理 (${items.length})`}
        extra={
          <Space>
            <Button
              onClick={() => {
                fetchRefs();
                fetchList();
              }}
            >
              刷新
            </Button>
            <Button
              type="primary"
              onClick={() => {
                setOpen(true);
                form.setFieldsValue({
                  agent_code: '',
                  channel_code: '',
                  session_key: '',
                  external_id: '',
                  metadata_json: '',
                });
              }}
            >
              新建会话
            </Button>
          </Space>
        }
      >
        <Table<Session> rowKey="id" loading={loading} dataSource={items} columns={columns} />
      </Card>

      <Modal
        title="新建会话"
        open={open}
        onCancel={() => setOpen(false)}
        footer={null}
        width="100%"
        styles={{ body: { paddingRight: 8 } }}
        className="responsive-modal"
      >
        <Form layout="vertical" form={form} onFinish={handleCreate}>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12 }}>
            <div style={{ flex: '1 1 280px' }}>
              <Form.Item label="Channel" name="channel_code" rules={[{ required: true, message: '请选择 Channel' }]}>
                <Select
                  placeholder="请选择"
                  options={channels.map((c) => ({ label: `${c.name} (${c.channel_code})`, value: c.channel_code }))}
                />
              </Form.Item>
            </div>
            <div style={{ flex: '1 1 280px' }}>
              <Form.Item label="Agent" name="agent_code" rules={[{ required: true, message: '请选择 Agent' }]}>
                <Select
                  placeholder="请选择"
                  options={agents.map((a) => ({ label: `${a.name} (${a.agent_code})`, value: a.agent_code }))}
                />
              </Form.Item>
            </div>
          </div>

          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12 }}>
            <div style={{ flex: '1 1 280px' }}>
              <Form.Item label="Session Key" name="session_key" rules={[{ required: true, message: '请输入 Session Key' }]}>
                <Input />
              </Form.Item>
            </div>
            <div style={{ flex: '1 1 280px' }}>
              <Form.Item label="External ID" name="external_id">
                <Input />
              </Form.Item>
            </div>
          </div>

          <Form.Item label="Metadata（JSON）" name="metadata_json" tooltip='例如：{"user_id":"xxx","source":"web"}'>
            <Input.TextArea rows={8} placeholder="可选，必须是 JSON 对象格式" />
          </Form.Item>

          <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button onClick={() => setOpen(false)}>取消</Button>
            <Button type="primary" htmlType="submit" loading={saving}>
              创建
            </Button>
          </Space>
        </Form>
      </Modal>

      <Drawer
        title="会话元数据"
        open={!!metadataDrawer}
        onClose={() => setMetadataDrawer(null)}
        width="100%"
        styles={{ body: { paddingRight: 8 } }}
        className="responsive-drawer"
      >
        {metadataDrawer ? (
          <div>
            <Space style={{ marginBottom: 12 }}>
              <Tag color="blue">{metadataDrawer.session_key}</Tag>
              <Tag>{metadataDrawer.channel_code}</Tag>
              <Tag>{metadataDrawer.agent_code}</Tag>
            </Space>
            <Typography.Paragraph>
              <pre style={{ whiteSpace: 'pre-wrap' }}>{toJsonText(metadataDrawer.metadata || {})}</pre>
            </Typography.Paragraph>
          </div>
        ) : null}
      </Drawer>

      <Modal
        title="编辑会话元数据"
        open={!!editingMetadata}
        onCancel={() => setEditingMetadata(null)}
        footer={null}
        width="100%"
        styles={{ body: { paddingRight: 8 } }}
        className="responsive-modal"
      >
        <Form layout="vertical" form={metadataForm} onFinish={handleSaveMetadata}>
          <Form.Item label="Metadata（JSON）" name="metadata_json" rules={[{ required: true, message: '请输入 JSON' }]}>
            <Input.TextArea rows={10} placeholder="必须是 JSON 对象格式" />
          </Form.Item>
          <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button onClick={() => setEditingMetadata(null)}>取消</Button>
            <Button type="primary" htmlType="submit" loading={saving}>
              保存
            </Button>
          </Space>
        </Form>
      </Modal>
    </div>
  );
};
