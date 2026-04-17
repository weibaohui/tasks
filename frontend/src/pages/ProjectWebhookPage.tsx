import React, { useEffect, useState, useMemo } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Modal,
  Popconfirm,
  Space,
  Switch,
  Table,
  Tag,
  message,
  Select,
  Tabs,
} from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import {
  listWebhookConfigs,
  createWebhookConfig,
  deleteWebhookConfig,
  enableWebhook,
  disableWebhook,
  listEventLogs,
  listBindings,
  createBinding,
  deleteBinding,
  listHeartbeatsForBinding,
  type GitHubWebhookConfig,
  type WebhookEventLog,
  type WebhookHeartbeatBinding,
  type HeartbeatOption,
} from '../api/githubWebhookApi';
import { listProjects } from '../api/projectRequirementApi';
import { GITHUB_EVENT_TYPES } from '../types/githubWebhook';
import type { Project } from '../types/projectRequirement';
import { useAuthStore } from '../stores/authStore';

export const ProjectWebhookPage: React.FC = () => {
  const { user } = useAuthStore();
  const [configs, setConfigs] = useState<GitHubWebhookConfig[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();
  const [selectedConfig, setSelectedConfig] = useState<GitHubWebhookConfig | null>(null);

  // Event logs state
  const [eventLogs, setEventLogs] = useState<WebhookEventLog[]>([]);
  const [logsLoading, setLogsLoading] = useState(false);

  // Bindings state
  const [bindings, setBindings] = useState<WebhookHeartbeatBinding[]>([]);
  const [heartbeats, setHeartbeats] = useState<HeartbeatOption[]>([]);
  const [bindingModalOpen, setBindingModalOpen] = useState(false);
  const [bindingForm] = Form.useForm();

  const fetchConfigs = async () => {
    setLoading(true);
    try {
      const data = await listWebhookConfigs();
      setConfigs(data);
    } catch {
      message.error('加载 Webhook 配置失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchProjects = async () => {
    try {
      const data = await listProjects();
      setProjects(data);
    } catch {
      // silent
    }
  };

  const fetchEventLogs = async (configId: string) => {
    setLogsLoading(true);
    try {
      const data = await listEventLogs(configId);
      setEventLogs(data);
    } catch {
      message.error('加载事件日志失败');
    } finally {
      setLogsLoading(false);
    }
  };

  const fetchBindings = async (configId: string) => {
    try {
      const data = await listBindings(configId);
      setBindings(data);
    } catch {
      message.error('加载心跳绑定失败');
    }
  };

  const fetchHeartbeats = async (projectId: string) => {
    try {
      const data = await listHeartbeatsForBinding(projectId);
      setHeartbeats(data);
    } catch {
      message.error('加载心跳列表失败');
    }
  };

  useEffect(() => {
    fetchConfigs();
    fetchProjects();
  }, [user?.user_code]);

  const handleOpenCreate = () => {
    form.resetFields();
    setModalOpen(true);
  };

  const handleSubmit = async (values: { project_id: string; repo: string }) => {
    try {
      await createWebhookConfig(values.project_id, values.repo);
      message.success('创建 Webhook 配置成功');
      setModalOpen(false);
      fetchConfigs();
    } catch {
      message.error('创建 Webhook 配置失败');
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteWebhookConfig(id);
      message.success('删除 Webhook 配置成功');
      fetchConfigs();
    } catch {
      message.error('删除 Webhook 配置失败');
    }
  };

  const handleToggleEnabled = async (config: GitHubWebhookConfig) => {
    try {
      if (config.enabled) {
        await disableWebhook(config.id);
        message.success('已停用 Webhook');
      } else {
        await enableWebhook(config.id);
        message.success('已启用 Webhook，Forwarder 启动中...');
      }
      fetchConfigs();
    } catch {
      message.error(config.enabled ? '停用失败' : '启用失败');
    }
  };

  const handleOpenBindings = async (config: GitHubWebhookConfig) => {
    setSelectedConfig(config);
    await Promise.all([
      fetchEventLogs(config.id),
      fetchBindings(config.id),
      fetchHeartbeats(config.project_id),
    ]);
  };

  const handleCreateBinding = async (values: { event_type: string; heartbeat_id: string }) => {
    if (!selectedConfig) return;
    try {
      await createBinding(selectedConfig.project_id, selectedConfig.id, values.event_type, values.heartbeat_id);
      message.success('创建绑定成功');
      setBindingModalOpen(false);
      bindingForm.resetFields();
      fetchBindings(selectedConfig.id);
    } catch {
      message.error('创建绑定失败');
    }
  };

  const handleDeleteBinding = async (id: string) => {
    try {
      await deleteBinding(id);
      message.success('删除绑定成功');
      if (selectedConfig) {
        fetchBindings(selectedConfig.id);
      }
    } catch {
      message.error('删除绑定失败');
    }
  };

  const columns: ColumnsType<GitHubWebhookConfig> = [
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_, record) => (
        <Space>
          <Button type="link" onClick={() => handleOpenBindings(record)}>
            事件与绑定
          </Button>
          <Popconfirm
            title="确认删除该配置？"
            onConfirm={() => handleDelete(record.id)}
          >
            <Button type="link" danger>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 100,
      render: (enabled: boolean, record) => (
        <Switch
          checked={enabled}
          onChange={() => handleToggleEnabled(record)}
          checkedChildren="启用"
          unCheckedChildren="停用"
        />
      ),
    },
    {
      title: 'Webhook 地址',
      dataIndex: 'webhook_url',
      key: 'webhook_url',
      width: 350,
      render: (webhookURL: string, record: GitHubWebhookConfig) => (
        record.running && webhookURL ? (
          <Tag color="green" title={webhookURL} style={{ maxWidth: 330, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {webhookURL}
          </Tag>
        ) : (
          <Tag color="default">未运行</Tag>
        )
      ),
    },
    {
      title: '项目',
      dataIndex: 'project_id',
      key: 'project_id',
      width: 200,
      render: (projectId: string) => {
        const project = projects.find((p) => p.id === projectId);
        return project?.name || projectId;
      },
    },
    {
      title: 'GitHub Repo',
      dataIndex: 'repo',
      key: 'repo',
      width: 200,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 170,
      render: (time: number) => new Date(time).toLocaleString(),
    },
  ];

  const eventLogColumns: ColumnsType<WebhookEventLog> = [
    {
      title: '时间',
      dataIndex: 'received_at',
      key: 'received_at',
      width: 170,
      render: (time: number) => new Date(time).toLocaleString(),
    },
    {
      title: '事件类型',
      dataIndex: 'event_type',
      key: 'event_type',
      width: 150,
      render: (type: string) => <Tag color="blue">{type}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const colorMap: Record<string, string> = {
          received: 'default',
          processed: 'success',
          failed: 'error',
        };
        return <Tag color={colorMap[status] || 'default'}>{status}</Tag>;
      },
    },
    {
      title: '触发的心跳',
      dataIndex: 'trigger_heartbeat_id',
      key: 'trigger_heartbeat_id',
      ellipsis: true,
      render: (id: string) => id || '-',
    },
    {
      title: '错误信息',
      dataIndex: 'error_message',
      key: 'error_message',
      ellipsis: true,
      render: (msg: string) => msg || '-',
    },
  ];

  const bindingColumns: ColumnsType<WebhookHeartbeatBinding> = [
    {
      title: 'GitHub 事件',
      dataIndex: 'github_event_type',
      key: 'github_event_type',
      width: 180,
      render: (type: string) => <Tag color="blue">{type}</Tag>,
    },
    {
      title: '触发的心跳',
      dataIndex: 'heartbeat_id',
      key: 'heartbeat_id',
      render: (id: string) => {
        const hb = heartbeats.find((h) => h.id === id);
        return hb?.name || id;
      },
    },
    {
      title: '心跳间隔',
      key: 'interval',
      width: 100,
      render: (_, record) => {
        const hb = heartbeats.find((h) => h.id === record.heartbeat_id);
        return hb ? `${hb.interval_minutes} 分钟` : '-';
      },
    },
    {
      title: 'Agent',
      key: 'agent',
      width: 150,
      render: (_, record) => {
        const hb = heartbeats.find((h) => h.id === record.heartbeat_id);
        return hb?.agent_code || '-';
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_, record) => (
        <Popconfirm
          title="确认删除该绑定？"
          onConfirm={() => handleDeleteBinding(record.id)}
        >
          <Button type="link" danger>
            删除
          </Button>
        </Popconfirm>
      ),
    },
  ];

  const tabItems = useMemo(() => {
    if (!selectedConfig) return [];
    return [
      {
        key: 'bindings',
        label: `心跳绑定 (${bindings.length})`,
        children: (
          <div>
            <div style={{ marginBottom: 16 }}>
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => setBindingModalOpen(true)}
              >
                添加绑定
              </Button>
            </div>
            <Table
              rowKey="id"
              loading={logsLoading}
              dataSource={bindings}
              columns={bindingColumns}
              pagination={false}
              size="small"
            />
          </div>
        ),
      },
      {
        key: 'logs',
        label: `事件日志 (${eventLogs.length})`,
        children: (
          <Table
            rowKey="id"
            loading={logsLoading}
            dataSource={eventLogs}
            columns={eventLogColumns}
            pagination={{ pageSize: 20 }}
            size="small"
          />
        ),
      },
    ];
  }, [selectedConfig, bindings, eventLogs, logsLoading]);

  return (
    <div style={{ padding: 0 }}>
      <Card
        title={`GitHub Webhook 配置 (${configs.length})`}
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={fetchConfigs}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleOpenCreate}>
              新建配置
            </Button>
          </Space>
        }
      >
        <Table
          rowKey="id"
          loading={loading}
          dataSource={configs}
          columns={columns}
          pagination={false}
          scroll={{ x: 'max-content' }}
        />
      </Card>

      <Modal
        title="新建 Webhook 配置"
        open={modalOpen}
        onOk={() => form.submit()}
        onCancel={() => setModalOpen(false)}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item
            label="关联项目"
            name="project_id"
            rules={[{ required: true, message: '请选择项目' }]}
          >
            <Select
              showSearch
              placeholder="选择项目"
              options={projects.map((p) => ({
                label: p.name,
                value: p.id,
              }))}
              filterOption={(input, option) =>
                (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
              }
            />
          </Form.Item>
          <Form.Item
            label="GitHub Repo"
            name="repo"
            rules={[{ required: true, message: '请输入 GitHub 仓库' }]}
            extra="格式: owner/repo，例如 weibaohui/tasks"
          >
            <Input placeholder="weibaohui/tasks" />
          </Form.Item>
        </Form>
      </Modal>

      {/* Bindings and Logs Modal */}
      <Modal
        title={
          selectedConfig
            ? `Webhook 配置 - ${projects.find((p) => p.id === selectedConfig.project_id)?.name || selectedConfig.project_id} / ${selectedConfig.repo}`
            : 'Webhook 配置'
        }
        open={!!selectedConfig}
        onCancel={() => {
          setSelectedConfig(null);
          setBindings([]);
          setEventLogs([]);
        }}
        footer={null}
        width={900}
        destroyOnClose
      >
        <Tabs items={tabItems} />
      </Modal>

      {/* Create Binding Modal */}
      <Modal
        title="添加心跳绑定"
        open={bindingModalOpen}
        onOk={() => bindingForm.submit()}
        onCancel={() => {
          setBindingModalOpen(false);
          bindingForm.resetFields();
        }}
        destroyOnClose
      >
        <Form form={bindingForm} layout="vertical" onFinish={handleCreateBinding}>
          <Form.Item
            label="GitHub 事件类型"
            name="event_type"
            rules={[{ required: true, message: '请选择事件类型' }]}
          >
            <Select
              placeholder="选择 GitHub 事件"
              options={GITHUB_EVENT_TYPES.map((e) => ({
                label: e.label,
                value: e.value,
              }))}
            />
          </Form.Item>
          <Form.Item
            label="触发的心跳"
            name="heartbeat_id"
            rules={[{ required: true, message: '请选择心跳' }]}
          >
            <Select
              placeholder="选择心跳"
              options={heartbeats
                .filter((h) => h.enabled)
                .map((h) => ({
                  label: `${h.name} (${h.interval_minutes}分钟, ${h.agent_code})`,
                  value: h.id,
                }))}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};
