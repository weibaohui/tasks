import React, { useEffect, useState } from 'react';
import {
  Button,
  Card,
  Drawer,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Tooltip,
  message,
  Typography,
  Divider,
} from 'antd';
import { PlusOutlined, DeleteOutlined, EyeOutlined, EditOutlined, MinusCircleOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import {
  listHeartbeatScenarios,
  createHeartbeatScenario,
  updateHeartbeatScenario,
  deleteHeartbeatScenario,
  type HeartbeatScenario,
  type HeartbeatScenarioItem,
} from '../api/heartbeatScenarioApi';
import { listAgents } from '../api/agentApi';
import { useAuthStore } from '../stores/authStore';
import type { Agent } from '../types/agent';

const { TextArea } = Input;
const { Text } = Typography;

const requirementTypeOptions = [
  { label: '心跳', value: 'heartbeat' },
  { label: '普通需求', value: 'normal' },
  { label: 'PR审查', value: 'pr_review' },
  { label: '优化', value: 'optimization' },
  { label: 'GitHub Issue', value: 'github_issue' },
  { label: 'GitHub Coding', value: 'github_coding' },
  { label: 'GitHub PR Review', value: 'github_pr_review' },
  { label: 'GitHub Doc', value: 'github_doc' },
  { label: 'GitHub Test', value: 'github_test' },
];

export const HeartbeatScenarioManagementPage: React.FC = () => {
  const { user } = useAuthStore();
  const [scenarios, setScenarios] = useState<HeartbeatScenario[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingScenario, setEditingScenario] = useState<HeartbeatScenario | null>(null);
  const [viewingScenario, setViewingScenario] = useState<HeartbeatScenario | null>(null);
  const [form] = Form.useForm();

  const fetchScenarios = async () => {
    setLoading(true);
    try {
      const data = await listHeartbeatScenarios();
      setScenarios(data);
    } catch {
      message.error('加载场景列表失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchAgents = async () => {
    try {
      if (user?.user_code) {
        const data = await listAgents(user.user_code);
        setAgents(data);
      }
    } catch {
      // silent
    }
  };

  useEffect(() => {
    fetchScenarios();
    fetchAgents();
  }, [user?.user_code]);

  const handleOpenCreate = () => {
    setEditingScenario(null);
    form.resetFields();
    form.setFieldsValue({
      enabled: true,
      items: [],
    });
    setModalOpen(true);
  };

  const handleOpenEdit = (scenario: HeartbeatScenario) => {
    setEditingScenario(scenario);
    form.setFieldsValue({
      code: scenario.code,
      name: scenario.name,
      description: scenario.description,
      enabled: scenario.enabled,
      items: scenario.items.map((item) => ({
        ...item,
      })),
    });
    setModalOpen(true);
  };

  const handleSubmit = async (values: any) => {
    try {
      const items: HeartbeatScenarioItem[] = (values.items || []).map((it: any) => ({
        name: it.name || '',
        interval_minutes: it.interval_minutes || 30,
        md_content: it.md_content || '',
        agent_code: it.agent_code || '',
        requirement_type: it.requirement_type || 'heartbeat',
        sort_order: it.sort_order ?? 0,
      }));
      if (editingScenario) {
        await updateHeartbeatScenario(editingScenario.code, {
          name: values.name,
          description: values.description,
          items,
          enabled: values.enabled,
        });
        message.success('更新场景成功');
      } else {
        await createHeartbeatScenario({
          code: values.code,
          name: values.name,
          description: values.description,
          items,
          enabled: values.enabled,
        });
        message.success('创建场景成功');
      }
      setModalOpen(false);
      fetchScenarios();
    } catch (_err) {
      message.error(editingScenario ? '更新场景失败' : '创建场景失败');
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteHeartbeatScenario(id);
      message.success('删除场景成功');
      fetchScenarios();
    } catch {
      message.error('删除场景失败');
    }
  };

  const columns: ColumnsType<HeartbeatScenario> = [
    {
      title: '操作',
      key: 'action',
      fixed: 'left',
      width: 120,
      render: (_: unknown, record: HeartbeatScenario) => (
        <Space>
          <Tooltip title="查看详情">
            <Button type="link" icon={<EyeOutlined />} onClick={() => setViewingScenario(record)} />
          </Tooltip>
          <Tooltip title="编辑">
            <Button type="link" icon={<EditOutlined />} onClick={() => handleOpenEdit(record)} />
          </Tooltip>
          <Popconfirm
            title="确认删除该场景？"
            onConfirm={() => handleDelete(record.id)}
          >
            <Button
              type="link"
              danger
              icon={<DeleteOutlined />}
            />
          </Popconfirm>
        </Space>
      ),
    },
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      width: 180,
    },
    {
      title: 'Code',
      dataIndex: 'code',
      key: 'code',
      width: 160,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '心跳项数',
      dataIndex: 'items',
      key: 'item_count',
      width: 100,
      align: 'center',
      render: (items: HeartbeatScenarioItem[]) => items?.length || 0,
    },
    {
      title: '类型',
      dataIndex: 'is_built_in',
      key: 'is_built_in',
      width: 100,
      render: (isBuiltIn: boolean) =>
        isBuiltIn ? <Tag color="blue">内置</Tag> : <Tag>自定义</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (enabled: boolean) => (enabled ? <Tag color="success">启用</Tag> : <Tag>禁用</Tag>),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 170,
      render: (time: number) => new Date(time).toLocaleString(),
    },
  ];

  return (
    <div style={{ padding: 0 }}>
      <Card
        title={`心跳场景管理 (${scenarios.length})`}
        extra={
          <Space>
            <Button onClick={fetchScenarios}>刷新</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleOpenCreate}>
              新建场景
            </Button>
          </Space>
        }
      >
        <Table
          rowKey="id"
          loading={loading}
          dataSource={scenarios}
          columns={columns}
          scroll={{ x: 'max-content' }}
        />
      </Card>

      <Modal
        title={editingScenario ? '编辑场景' : '新建场景'}
        open={modalOpen}
        onOk={() => form.submit()}
        onCancel={() => setModalOpen(false)}
        width={800}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item
            label="场景 Code"
            name="code"
            rules={[{ required: true, message: '请输入场景 Code' }]}
          >
            <Input disabled={!!editingScenario} placeholder="例如：github_dev_workflow" />
          </Form.Item>
          <Form.Item
            label="场景名称"
            name="name"
            rules={[{ required: true, message: '请输入场景名称' }]}
          >
            <Input placeholder="例如：GitHub 开发协作工作流" />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input placeholder="简要描述该场景的用途" />
          </Form.Item>
          <Form.Item label="启用状态" name="enabled" valuePropName="checked">
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>

          <Divider orientation="left">心跳项配置</Divider>
          <Form.List name="items">
            {(fields, { add, remove }) => (
              <div>
                {fields.map(({ key, name, ...restField }) => (
                  <Card
                    key={key}
                    size="small"
                    title={`心跳项 #${name + 1}`}
                    style={{ marginBottom: 12 }}
                    extra={
                      <Button
                        type="text"
                        danger
                        icon={<MinusCircleOutlined />}
                        onClick={() => remove(name)}
                      >
                        删除
                      </Button>
                    }
                  >
                    <Space style={{ width: '100%' }} align="start" wrap>
                      <Form.Item
                        {...restField}
                        name={[name, 'name']}
                        rules={[{ required: true, message: '请输入名称' }]}
                        label="名称"
                      >
                        <Input placeholder="心跳名称" />
                      </Form.Item>
                      <Form.Item
                        {...restField}
                        name={[name, 'interval_minutes']}
                        rules={[{ required: true, min: 1, type: 'number' }]}
                        label="间隔（分钟）"
                      >
                        <InputNumber min={1} style={{ width: 120 }} />
                      </Form.Item>
                      <Form.Item
                        {...restField}
                        name={[name, 'agent_code']}
                        rules={[{ required: true, message: '请选择 Agent' }]}
                        label="Agent"
                      >
                        <Select
                          style={{ width: 200 }}
                          placeholder="选择 Agent"
                          options={agents.map((a) => ({
                            label: `${a.name} (${a.agent_code})`,
                            value: a.agent_code,
                          }))}
                        />
                      </Form.Item>
                      <Form.Item
                        {...restField}
                        name={[name, 'requirement_type']}
                        rules={[{ required: true, message: '请选择需求类型' }]}
                        label="需求类型"
                      >
                        <Select style={{ width: 160 }} options={requirementTypeOptions} />
                      </Form.Item>
                      <Form.Item
                        {...restField}
                        name={[name, 'sort_order']}
                        label="排序"
                      >
                        <InputNumber style={{ width: 80 }} />
                      </Form.Item>
                    </Space>
                    <Form.Item
                      {...restField}
                      name={[name, 'md_content']}
                      label="Prompt 模板（MD）"
                      style={{ marginTop: 8 }}
                    >
                      <TextArea
                        rows={6}
                        style={{ fontFamily: 'monospace' }}
                        placeholder="输入心跳 Prompt 模板，可用变量如 ${project.name}、${project.git_repo_url}"
                      />
                    </Form.Item>
                  </Card>
                ))}
                <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                  添加心跳项
                </Button>
              </div>
            )}
          </Form.List>
        </Form>
      </Modal>

      <Drawer
        title={viewingScenario ? `${viewingScenario.name}（${viewingScenario.code}）` : '场景详情'}
        open={!!viewingScenario}
        onClose={() => setViewingScenario(null)}
        width={720}
      >
        {viewingScenario && (
          <div>
            <Text type="secondary">{viewingScenario.description || '暂无描述'}</Text>
            <Divider />
            <Space style={{ marginBottom: 16 }} wrap>
              <Tag color={viewingScenario.is_built_in ? 'blue' : 'default'}>
                {viewingScenario.is_built_in ? '内置场景' : '自定义场景'}
              </Tag>
              <Tag color={viewingScenario.enabled ? 'success' : 'default'}>
                {viewingScenario.enabled ? '已启用' : '已禁用'}
              </Tag>
              <Text>共 {viewingScenario.items.length} 个心跳项</Text>
            </Space>

            {viewingScenario.items.map((item, idx) => (
              <Card
                key={idx}
                size="small"
                title={
                  <Space>
                    <span>{item.name}</span>
                    <Tag>{item.interval_minutes} 分钟</Tag>
                    <Tag>{item.agent_code}</Tag>
                    <Tag>
                      {requirementTypeOptions.find((o) => o.value === item.requirement_type)?.label ||
                        item.requirement_type}
                    </Tag>
                  </Space>
                }
                style={{ marginBottom: 12 }}
              >
                <pre
                  style={{
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                    fontFamily: 'monospace',
                    fontSize: 12,
                    background: '#f6f8fa',
                    padding: 12,
                    borderRadius: 6,
                  }}
                >
                  {item.md_content}
                </pre>
              </Card>
            ))}
          </div>
        )}
      </Drawer>
    </div>
  );
};
