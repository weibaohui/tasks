import React, { useEffect, useState, useMemo } from 'react';
import {
  Alert,
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
import { TraceViewer } from '../components/TraceViewer';
import {
  listWebhookConfigs,
  createWebhookConfig,
  deleteWebhookConfig,
  enableWebhook,
  disableWebhook,
  listEventLogs,
  clearEventLogs,
  listBindings,
  createBinding,
  deleteBinding,
  listHeartbeatsForBinding,
  retriggerHeartbeat,
  checkWebhookURL,
  updateWebhookURL,
  type GitHubWebhookConfig,
  type WebhookEventLog,
  type WebhookHeartbeatBinding,
  type HeartbeatOption,
} from '../api/githubWebhookApi';
import { listProjects, getRequirement } from '../api/projectRequirementApi';
import { GITHUB_EVENT_TYPES, ATG_EVENT_TYPES, EVENT_TO_REQUIREMENT_TYPE } from '../types/githubWebhook';
import type { Project } from '../types/projectRequirement';
import { detectPlatformType } from '../types/projectRequirement';
import { useAuthStore } from '../stores/authStore';

interface ProjectWebhookPageProps {
  selectedProject?: Project | null;
}

/**
 * normalizeGitHubRepo 将项目仓库地址标准化为 owner/repo 格式。
 */
function normalizeGitHubRepo(repoURL: string): string {
  const trimmed = (repoURL || '').trim();
  if (!trimmed) {
    return '';
  }
  if (/^[^/\s]+\/[^/\s]+$/.test(trimmed)) {
    return trimmed.replace(/\.git$/, '');
  }
  const match = trimmed.match(/github\.com[:/]+([^/\s]+)\/([^/\s]+?)(?:\.git)?$/i);
  if (!match) {
    return '';
  }
  return `${match[1]}/${match[2]}`;
}

export const ProjectWebhookPage: React.FC<ProjectWebhookPageProps> = ({ selectedProject = null }) => {
  const { user } = useAuthStore();
  const [configs, setConfigs] = useState<GitHubWebhookConfig[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();
  const [selectedConfig, setSelectedConfig] = useState<GitHubWebhookConfig | null>(null);
  const [bindingsLoading, setBindingsLoading] = useState(false);

  // Event logs state
  const [eventLogs, setEventLogs] = useState<WebhookEventLog[]>([]);
  const [logsLoading, setLogsLoading] = useState(false);
  const [logsTotal, setLogsTotal] = useState(0);
  const [logsOffset, setLogsOffset] = useState(0);
  const logsLimit = 20;

  // Bindings state
  const [bindings, setBindings] = useState<WebhookHeartbeatBinding[]>([]);
  const [heartbeats, setHeartbeats] = useState<HeartbeatOption[]>([]);
  const [bindingModalOpen, setBindingModalOpen] = useState(false);
  const [bindingForm] = Form.useForm();
  const scopedRepo = useMemo(() => normalizeGitHubRepo(selectedProject?.git_repo_url || ''), [selectedProject?.git_repo_url]);
  const isProjectScoped = !!selectedProject?.id;

  // 当前项目平台类型 - 从 selectedConfig 派生（避免 prop 滞后问题）
  const platformType = useMemo(() => {
    if (!selectedConfig) return null;
    const project = projects.find((p) => p.id === selectedConfig.project_id);
    if (!project) return null;
    return detectPlatformType(project.git_repo_url);
  }, [selectedConfig?.project_id, projects]);

  // 绑定modal中选中事件类型state
  const [selectedEventType, setSelectedEventType] = useState<string | null>(null);

  // 根据选中事件类型过滤后的心跳列表
  const filteredHeartbeats = useMemo(() => {
    if (!selectedEventType || !platformType) {
      return heartbeats.filter((h) => h.enabled);
    }
    const mapping = EVENT_TO_REQUIREMENT_TYPE[selectedEventType];
    if (!mapping) {
      return heartbeats.filter((h) => h.enabled);
    }
    const requiredType = platformType === 'github' ? mapping.github : mapping.atg;
    return heartbeats.filter((h) => h.enabled && h.requirement_type === requiredType);
  }, [heartbeats, selectedEventType, platformType]);

  // Trace viewer state
  const [traceVisible, setTraceVisible] = useState(false);
  const [currentTraceId, setCurrentTraceId] = useState('');

  // Payload 查看 state
  const [payloadModalVisible, setPayloadModalVisible] = useState(false);
  const [currentPayload, setCurrentPayload] = useState('');
  const [currentEventType, setCurrentEventType] = useState('');
  const [currentMethod, setCurrentMethod] = useState('');
  const [currentHeaders, setCurrentHeaders] = useState('');

  const fetchConfigs = async () => {
    setLoading(true);
    try {
      const data = await listWebhookConfigs();
      if (selectedProject?.id) {
        setConfigs(data.filter((item) => item.project_id === selectedProject.id));
      } else {
        setConfigs(data);
      }
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

  const fetchEventLogs = async (configId: string, offset = 0) => {
    setLogsLoading(true);
    try {
      const data = await listEventLogs(configId, logsLimit, offset);
      setEventLogs(data.data);
      setLogsTotal(data.total);
      setLogsOffset(offset);
    } catch {
      message.error('加载事件日志失败');
    } finally {
      setLogsLoading(false);
    }
  };

  const clearLogs = async (configId: string) => {
    try {
      await clearEventLogs(configId);
      message.success('日志已清空');
      fetchEventLogs(configId, 0);
    } catch {
      message.error('清空日志失败');
    }
  };

  const fetchBindings = async (configId: string) => {
    setBindingsLoading(true);
    try {
      const data = await listBindings(configId);
      setBindings(data);
    } catch {
      message.error('加载心跳绑定失败');
    } finally {
      setBindingsLoading(false);
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
  }, [user?.user_code, selectedProject?.id]);

  useEffect(() => {
    setSelectedConfig(null);
    setBindings([]);
    setEventLogs([]);
  }, [selectedProject?.id]);

  /**
   * 当配置列表变化时自动选中可用配置，并加载下方双 Tab 数据。
   */
  useEffect(() => {
    if (configs.length === 0) {
      setSelectedConfig(null);
      setBindings([]);
      setEventLogs([]);
      return;
    }
    const matched = selectedConfig ? configs.find((item) => item.id === selectedConfig.id) : null;
    const target = matched || configs[0];
    void handleOpenBindings(target);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [configs]);

  const handleOpenCreate = () => {
    if (isProjectScoped) {
      if (!scopedRepo) {
        message.error('当前项目仓库信息无效，请先在项目中配置 GitHub 仓库地址');
        return;
      }
      form.resetFields();
      form.setFieldsValue({
        project_id: selectedProject?.id,
        repo: scopedRepo,
      });
    } else {
      form.resetFields();
    }
    setModalOpen(true);
  };

  const handleSubmit = async (values: { project_id: string; repo: string }) => {
    const projectID = isProjectScoped ? (selectedProject?.id || '') : values.project_id;
    const repo = isProjectScoped ? scopedRepo : values.repo;
    if (!projectID || !repo) {
      message.error('缺少项目或仓库信息，无法创建 Webhook 配置');
      return;
    }
    try {
      await createWebhookConfig(projectID, repo);
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

  /**
   * handleOpenBindings 选择当前配置并加载其绑定、日志与可绑定心跳。
   */
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

  const handleRetrigger = async (heartbeatId: string) => {
    try {
      await retriggerHeartbeat(heartbeatId);
      message.success('已重新触发心跳');
    } catch {
      message.error('触发失败');
    }
  };

  const handleRetriggerByEventType = async (eventType: string) => {
    // 根据事件类型查找绑定的心跳并触发
    try {
      const bindingsData = await listBindings(selectedConfig?.id || '');
      const binding = bindingsData.find((b) => b.github_event_type === eventType);
      if (binding) {
        await handleRetrigger(binding.heartbeat_id);
      } else {
        message.error('未找到该事件类型的绑定');
      }
    } catch {
      message.error('触发失败');
    }
  };

  const handleViewTrace = async (requirementId: string) => {
    try {
      const requirement = await getRequirement(requirementId);
      if (requirement.trace_id) {
        setCurrentTraceId(requirement.trace_id);
        setTraceVisible(true);
      } else {
        message.warning('该需求没有 trace_id');
      }
    } catch {
      message.error('获取需求信息失败');
    }
  };

  const handleCheckWebhookURL = async (configId: string) => {
    try {
      const result = await checkWebhookURL(configId);
      if (result.needs_update) {
        Modal.confirm({
          title: 'Webhook URL 已过期',
          content: (
            <div>
              <p>检测到 GitHub 上的 Webhook URL 与本地记录不一致，需要更新。</p>
              <p>当前 URL: <code>{result.current_url}</code></p>
              <p>预期 URL: <code>{result.expected_url}</code></p>
            </div>
          ),
          onOk: async () => {
            try {
              await updateWebhookURL(configId);
              message.success('Webhook URL 已更新');
              fetchConfigs();
            } catch {
              message.error('更新失败');
            }
          },
        });
      } else {
        message.success('Webhook URL 已是最新');
      }
    } catch {
      message.error('检测失败');
    }
  };

  const columns: ColumnsType<GitHubWebhookConfig> = [
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_, record) => (
        <Space>
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
        record.enabled && webhookURL ? (
          <Space>
            <Tag color="green" title={webhookURL} style={{ maxWidth: 250, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {webhookURL}
            </Tag>
            <Button type="link" size="small" onClick={() => handleCheckWebhookURL(record.id)}>
              检测
            </Button>
          </Space>
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
      key: 'triggered_heartbeats',
      ellipsis: true,
      width: 200,
      render: (_, record) => {
        const triggered = record.triggered_heartbeats || [];
        if (triggered.length === 0) {
          return record.trigger_heartbeat_id || '-';
        }
        if (triggered.length === 1) {
          return triggered[0].heartbeat_id || '-';
        }
        return `${triggered.length} 个心跳`;
      },
    },
    {
      title: '错误信息',
      dataIndex: 'error_message',
      key: 'error_message',
      ellipsis: true,
      width: 150,
      render: (msg: string) => msg || '-',
    },
    {
      title: '原始内容',
      key: 'payload',
      width: 80,
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          onClick={() => {
            setCurrentPayload(record.payload || '');
            setCurrentEventType(record.event_type || '');
            setCurrentMethod(record.method || '');
            setCurrentHeaders(record.headers || '');
            setPayloadModalVisible(true);
          }}
        >
          查看
        </Button>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_, record) => {
        const triggered = record.triggered_heartbeats || [];
        // 如果有触发的心跳，显示"查看链路"按钮
        const hasTriggered = triggered.length > 0;
        const firstTriggered = hasTriggered ? triggered[0] : null;

        // 使用触发的心跳中的 requirement_id（优先）或旧字段
        const requirementId = hasTriggered && firstTriggered?.requirement_id
          ? firstTriggered.requirement_id
          : record.requirement_id;

        return (
          <Space>
            {requirementId && (
              <Button
                type="link"
                size="small"
                onClick={() => handleViewTrace(requirementId)}
              >
                查看链路
              </Button>
            )}
            {hasTriggered ? (
              // 有触发的心跳，显示重新触发按钮（使用第一个）
              <Button
                type="link"
                size="small"
                onClick={() => firstTriggered && handleRetrigger(firstTriggered.heartbeat_id)}
              >
                重新触发
              </Button>
            ) : record.trigger_heartbeat_id ? (
              // 兼容旧数据
              <Button
                type="link"
                size="small"
                onClick={() => handleRetrigger(record.trigger_heartbeat_id)}
              >
                重新触发
              </Button>
            ) : (
              // 没有任何触发记录，显示手动触发
              <Button
                type="link"
                size="small"
                onClick={() => handleRetriggerByEventType(record.event_type)}
              >
                手动触发
              </Button>
            )}
          </Space>
        );
      },
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
              <Space>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={() => selectedConfig && fetchBindings(selectedConfig.id)}
                >
                  刷新
                </Button>
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={async () => {
                    if (selectedConfig) {
                      await fetchHeartbeats(selectedConfig.project_id);
                    }
                    setBindingModalOpen(true);
                  }}
                >
                  添加绑定
                </Button>
              </Space>
            </div>
            <Table
              rowKey="id"
              loading={bindingsLoading}
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
        label: `事件日志 (${logsTotal})`,
        children: (
          <div>
            <div style={{ marginBottom: 16 }}>
              <Space>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={() => selectedConfig && fetchEventLogs(selectedConfig.id, 0)}
                >
                  刷新
                </Button>
                <Popconfirm
                  title="确定清空所有事件日志吗？"
                  onConfirm={() => selectedConfig && clearLogs(selectedConfig.id)}
                >
                  <Button danger>清空日志</Button>
                </Popconfirm>
              </Space>
            </div>
            <Table
              rowKey="id"
              loading={logsLoading}
              dataSource={eventLogs}
              columns={eventLogColumns}
              pagination={{
                pageSize: logsLimit,
                total: logsTotal,
                current: Math.floor(logsOffset / logsLimit) + 1,
                onChange: (page) => fetchEventLogs(selectedConfig!.id, (page - 1) * logsLimit),
              }}
              size="small"
              scroll={{ x: 'max-content' }}
              expandable={{
                expandedRowRender: (record) => {
                  const triggered = record.triggered_heartbeats || [];
                  if (triggered.length === 0) {
                    return <span>无触发的心跳</span>;
                  }
                  return (
                    <div style={{ margin: '8px 0' }}>
                      <div style={{ fontWeight: 500, marginBottom: 8 }}>触发的心跳列表：</div>
                      <Table
                        key={record.id}
                        dataSource={triggered}
                        columns={[
                          {
                            title: '心跳 ID',
                            dataIndex: 'heartbeat_id',
                            key: 'heartbeat_id',
                            width: 200,
                            render: (id: string) => {
                              const hb = heartbeats.find((h) => h.id === id);
                              return hb?.name || id;
                            },
                          },
                          {
                            title: '需求 ID',
                            dataIndex: 'requirement_id',
                            key: 'requirement_id',
                            ellipsis: true,
                            render: (id: string) => id || '-',
                          },
                          {
                            title: '触发时间',
                            dataIndex: 'triggered_at',
                            key: 'triggered_at',
                            width: 170,
                            render: (time: number) => new Date(time).toLocaleString(),
                          },
                          {
                            title: '操作',
                            key: 'action',
                            width: 120,
                            render: (_, t) => (
                              <Space>
                                {t.requirement_id && (
                                  <Button
                                    type="link"
                                    size="small"
                                    onClick={() => handleViewTrace(t.requirement_id)}
                                  >
                                    查看链路
                                  </Button>
                                )}
                                <Button
                                  type="link"
                                  size="small"
                                  onClick={() => handleRetrigger(t.heartbeat_id)}
                                >
                                  重新触发
                                </Button>
                              </Space>
                            ),
                          },
                        ]}
                        pagination={false}
                        size="small"
                        rowKey={(record) => record.id}
                      />
                    </div>
                  );
                },
                rowExpandable: (record) => (record.triggered_heartbeats?.length || 0) > 0,
              }}
            />
          </div>
        ),
      },
    ];
  }, [selectedConfig, bindings, eventLogs, bindingsLoading, logsLoading, logsTotal, logsOffset, logsLimit]);

  return (
    <div style={{ padding: 0 }}>
      <Card
        title={`GitHub Webhook 配置 (${configs.length})${selectedProject ? ` - ${selectedProject.name}` : ''}`}
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
          onRow={(record) => ({
            onClick: () => {
              void handleOpenBindings(record);
            },
          })}
          rowClassName={(record) => (selectedConfig?.id === record.id ? 'ant-table-row-selected' : '')}
        />
      </Card>

      <Card
        style={{ marginTop: 16 }}
        title={
          selectedConfig
            ? `配置详情：${projects.find((p) => p.id === selectedConfig.project_id)?.name || selectedConfig.project_id} / ${selectedConfig.repo}`
            : '配置详情'
        }
      >
        {selectedConfig ? (
          <Tabs items={tabItems} />
        ) : (
          <Alert
            type="info"
            showIcon
            message="暂无可管理的配置"
            description="请先创建 Webhook 配置，创建后会自动在下方显示绑定事件列表和事件日志。"
          />
        )}
      </Card>

      <Modal
        title="新建 Webhook 配置"
        open={modalOpen}
        onOk={() => form.submit()}
        onCancel={() => setModalOpen(false)}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          {isProjectScoped ? (
            <>
              <Alert
                type="info"
                showIcon
                style={{ marginBottom: 12 }}
                message="已使用当前项目自动填充"
                description="新建配置会直接使用当前已选择项目及其仓库信息，无需重复填写。"
              />
              <Form.Item label="关联项目">
                <Input value={selectedProject?.name || ''} disabled />
              </Form.Item>
              <Form.Item label="GitHub Repo" extra="自动从项目仓库地址解析">
                <Input value={scopedRepo || '未配置'} disabled />
              </Form.Item>
              <Form.Item name="project_id" hidden>
                <Input />
              </Form.Item>
              <Form.Item name="repo" hidden>
                <Input />
              </Form.Item>
            </>
          ) : (
            <>
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
            </>
          )}
        </Form>
      </Modal>

      {/* Bindings and Logs Modal */}
      <Modal
        title={
          selectedConfig
            ? `Webhook 配置 - ${projects.find((p) => p.id === selectedConfig.project_id)?.name || selectedConfig.project_id} / ${selectedConfig.repo}`
            : 'Webhook 配置'
        }
        open={false}
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
          setSelectedEventType(null);
        }}
        destroyOnClose
      >
        <Form form={bindingForm} layout="vertical" onFinish={handleCreateBinding}>
          <Form.Item
            label={`${platformType === 'github' ? 'GitHub' : 'AtomGit'} 事件类型`}
            name="event_type"
            rules={[{ required: true, message: '请选择事件类型' }]}
          >
            <Select
              placeholder={`选择 ${platformType === 'github' ? 'GitHub' : 'AtomGit'} 事件`}
              options={(platformType === 'github' ? GITHUB_EVENT_TYPES : ATG_EVENT_TYPES).map((e) => ({
                label: e.label,
                value: e.value,
              }))}
              onChange={(value) => setSelectedEventType(value)}
            />
          </Form.Item>
          <Form.Item
            label="触发的心跳"
            name="heartbeat_id"
            rules={[{ required: true, message: '请选择心跳' }]}
            extra={selectedEventType && filteredHeartbeats.length === 0 ? '当前事件类型没有匹配的心跳，请先为项目应用对应场景' : ''}
          >
            <Select
              placeholder="选择心跳"
              options={filteredHeartbeats.map((h) => ({
                label: `${h.name} (${h.interval_minutes}分钟, ${h.agent_code})`,
                value: h.id,
              }))}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* Trace Viewer Modal */}
      <TraceViewer
        traceId={currentTraceId}
        visible={traceVisible}
        onClose={() => {
          setTraceVisible(false);
          setCurrentTraceId('');
        }}
      />

      {/* Payload 查看 Modal */}
      <Modal
        title={`原始事件内容 - ${currentEventType}`}
        open={payloadModalVisible}
        onCancel={() => setPayloadModalVisible(false)}
        footer={null}
        width={900}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          {/* 请求信息头部 */}
          <div style={{ display: 'flex', gap: 16, padding: '8px 12px', background: '#f0f5ff', borderRadius: 4, fontSize: 13 }}>
            <span><strong>Method:</strong> {currentMethod || 'POST'}</span>
            <span><strong>Event:</strong> {currentEventType}</span>
          </div>

          {/* Headers */}
          <div>
            <div style={{ fontWeight: 500, marginBottom: 8, color: '#333' }}>Headers</div>
            <pre style={{
              maxHeight: 150,
              overflow: 'auto',
              background: '#f8f8f8',
              padding: 12,
              borderRadius: 4,
              fontSize: 12,
              fontFamily: 'Monaco, Menlo, monospace',
              whiteSpace: 'pre-wrap',
              margin: 0,
            }}>
              {currentHeaders || '(无 headers)'}
            </pre>
          </div>

          {/* Payload */}
          <div>
            <div style={{ fontWeight: 500, marginBottom: 8, color: '#333' }}>Payload</div>
            <pre style={{
              maxHeight: 400,
              overflow: 'auto',
              background: '#f8f8f8',
              padding: 12,
              borderRadius: 4,
              fontSize: 12,
              fontFamily: 'Monaco, Menlo, monospace',
              whiteSpace: 'pre-wrap',
              margin: 0,
            }}>
              {(() => {
                try {
                  return JSON.stringify(JSON.parse(currentPayload || '{}'), null, 2);
                } catch {
                  return currentPayload || '(无 payload)';
                }
              })()}
            </pre>
          </div>
        </div>
      </Modal>
    </div>
  );
};
