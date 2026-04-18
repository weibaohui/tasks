import React, { useEffect, useState } from 'react';
import { Button, Form, Input, InputNumber, Modal, Select, Space, Switch, Table, Tag, message, Tooltip } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined, SaveOutlined, MinusCircleOutlined, FileTextOutlined, HistoryOutlined } from '@ant-design/icons';
import type { Heartbeat, HeartbeatRunRecord } from '../../types/heartbeat';
import type { HeartbeatTemplate } from '../../types/heartbeat_template';
import type { Agent } from '../../types/agent';
import type { RequirementType } from '../../api/requirementTypeApi';
import { listHeartbeats, createHeartbeat, updateHeartbeat, deleteHeartbeat, listHeartbeatRuns } from '../../api/heartbeatApi';
import { listHeartbeatTemplates, createHeartbeatTemplate, deleteHeartbeatTemplate } from '../../api/heartbeatTemplateApi';

interface HeartbeatManagementProps {
  projectId: string;
  agents: Agent[];
  requirementTypes?: RequirementType[];
}

const requirementTypeOptions = [
  { label: '心跳', value: 'heartbeat' },
  { label: '普通需求', value: 'normal' },
  { label: 'PR审查', value: 'pr_review' },
  { label: '优化', value: 'optimization' },
];

const typeColorMap: Record<string, string> = {
  heartbeat: 'green',
  normal: 'blue',
  pr_review: 'orange',
  optimization: 'purple',
};

export const HeartbeatManagement: React.FC<HeartbeatManagementProps> = ({ projectId, agents, requirementTypes }) => {
  const [heartbeats, setHeartbeats] = useState<Heartbeat[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingHeartbeat, setEditingHeartbeat] = useState<Heartbeat | null>(null);
  const [form] = Form.useForm();

  const [templates, setTemplates] = useState<HeartbeatTemplate[]>([]);
  const [templateModalOpen, setTemplateModalOpen] = useState(false);
  const [templateForm] = Form.useForm();
  const [savingTemplate, setSavingTemplate] = useState(false);
  const [runsModalOpen, setRunsModalOpen] = useState(false);
  const [runsLoading, setRunsLoading] = useState(false);
  const [runs, setRuns] = useState<HeartbeatRunRecord[]>([]);
  const [runsHeartbeatName, setRunsHeartbeatName] = useState('');
  const dynamicRequirementTypeOptions = requirementTypes && requirementTypes.length > 0
    ? requirementTypes.map((item) => ({ label: `${item.name} (${item.code})`, value: item.code }))
    : requirementTypeOptions;

  const fetchHeartbeats = async () => {
    setLoading(true);
    try {
      const data = await listHeartbeats(projectId);
      setHeartbeats(data);
    } catch {
      message.error('加载心跳列表失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchTemplates = async () => {
    try {
      const data = await listHeartbeatTemplates();
      setTemplates(data);
    } catch {
      // 静默失败，不阻塞主流程
    }
  };

  useEffect(() => {
    fetchHeartbeats();
  }, [projectId]);

  const handleAdd = () => {
    setEditingHeartbeat(null);
    form.resetFields();
    form.setFieldsValue({
      enabled: true,
      interval_minutes: 30,
      requirement_type: 'heartbeat',
      md_content: '',
    });
    fetchTemplates();
    setModalOpen(true);
  };

  const handleEdit = (hb: Heartbeat) => {
    setEditingHeartbeat(hb);
    form.setFieldsValue({
      name: hb.name,
      enabled: hb.enabled,
      interval_minutes: hb.interval_minutes,
      agent_code: hb.agent_code,
      requirement_type: hb.requirement_type || 'heartbeat',
      md_content: hb.md_content,
    });
    fetchTemplates();
    setModalOpen(true);
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteHeartbeat(id);
      message.success('删除成功');
      fetchHeartbeats();
    } catch {
      message.error('删除失败');
    }
  };

  const handleToggleEnabled = async (id: string, enabled: boolean) => {
    try {
      const hb = heartbeats.find((h) => h.id === id);
      if (!hb) return;
      await updateHeartbeat(id, {
        name: hb.name,
        interval_minutes: hb.interval_minutes,
        md_content: hb.md_content,
        agent_code: hb.agent_code,
        requirement_type: hb.requirement_type || 'heartbeat',
        enabled,
      });
      message.success(enabled ? '已启用' : '已禁用');
      fetchHeartbeats();
    } catch {
      message.error('切换状态失败');
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      if (editingHeartbeat) {
        await updateHeartbeat(editingHeartbeat.id, {
          name: values.name,
          interval_minutes: values.interval_minutes,
          md_content: values.md_content,
          agent_code: values.agent_code,
          requirement_type: values.requirement_type,
          enabled: values.enabled,
        });
        message.success('更新成功');
      } else {
        await createHeartbeat({
          project_id: projectId,
          name: values.name,
          interval_minutes: values.interval_minutes,
          md_content: values.md_content,
          agent_code: values.agent_code,
          requirement_type: values.requirement_type,
        });
        message.success('创建成功');
      }
      setModalOpen(false);
      fetchHeartbeats();
    } catch {
      // validation or request error
    }
  };

  const handleApplyTemplate = (templateId: string) => {
    const t = templates.find((item) => item.id === templateId);
    if (!t) return;
    form.setFieldValue('md_content', t.md_content);
    form.setFieldValue('requirement_type', t.requirement_type);
    message.success(`已应用模板：${t.name}`);
  };

  const handleSaveTemplate = async () => {
    try {
      const values = await templateForm.validateFields();
      const mdContent = form.getFieldValue('md_content') || '';
      const requirementType = form.getFieldValue('requirement_type') || 'heartbeat';
      setSavingTemplate(true);
      await createHeartbeatTemplate({
        name: values.name,
        md_content: mdContent,
        requirement_type: requirementType,
      });
      message.success('保存模板成功');
      setTemplateModalOpen(false);
      templateForm.resetFields();
      fetchTemplates();
    } catch {
      // validation or request error
    } finally {
      setSavingTemplate(false);
    }
  };

  const handleDeleteTemplate = async (e: React.MouseEvent, templateId: string) => {
    e.stopPropagation();
    try {
      await deleteHeartbeatTemplate(templateId);
      message.success('删除模板成功');
      fetchTemplates();
    } catch {
      message.error('删除模板失败');
    }
  };

  const handleViewRuns = async (hb: Heartbeat) => {
    setRunsHeartbeatName(hb.name);
    setRunsModalOpen(true);
    setRunsLoading(true);
    try {
      const data = await listHeartbeatRuns(hb.id, 20);
      setRuns(data);
    } catch {
      message.error('加载心跳执行记录失败');
      setRuns([]);
    } finally {
      setRunsLoading(false);
    }
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 120,
      render: (id: string) => (
        <span style={{ fontFamily: 'monospace', fontSize: 12, color: '#666' }}>{id}</span>
      ),
    },
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '间隔(分钟)',
      dataIndex: 'interval_minutes',
      key: 'interval_minutes',
      width: 110,
    },
    {
      title: 'Agent',
      dataIndex: 'agent_code',
      key: 'agent_code',
      width: 140,
      render: (code: string) => {
        const agent = agents.find((a) => a.agent_code === code);
        return agent?.name || code;
      },
    },
    {
      title: '需求类型',
      dataIndex: 'requirement_type',
      key: 'requirement_type',
      width: 110,
      render: (type: string) => (
        <Tag color={typeColorMap[type] || 'default'}>
          {dynamicRequirementTypeOptions.find((o) => o.value === type)?.label || type}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (enabled: boolean, record: Heartbeat) => (
        <Switch
          checked={enabled}
          onChange={(checked) => handleToggleEnabled(record.id, checked)}
          checkedChildren="启用"
          unCheckedChildren="禁用"
        />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 130,
      render: (_: unknown, record: Heartbeat) => (
        <Space>
          <Tooltip title="编辑">
            <Button type="link" icon={<EditOutlined />} onClick={() => handleEdit(record)} />
          </Tooltip>
          <Tooltip title="删除">
            <Button type="link" danger icon={<DeleteOutlined />} onClick={() => handleDelete(record.id)} />
          </Tooltip>
          <Tooltip title="执行记录">
            <Button type="link" icon={<HistoryOutlined />} onClick={() => handleViewRuns(record)} />
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          新增心跳
        </Button>
      </div>
      <Table
        rowKey="id"
        dataSource={heartbeats}
        columns={columns}
        loading={loading}
        size="small"
        pagination={false}
      />
      <Modal
        title={editingHeartbeat ? '编辑心跳' : '新增心跳'}
        open={modalOpen}
        onOk={handleModalOk}
        onCancel={() => setModalOpen(false)}
        width={720}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入心跳名称' }]}>
            <Input placeholder="例如：PR检查" />
          </Form.Item>
          <Space style={{ width: '100%' }} align="start">
            <Form.Item label="间隔（分钟）" name="interval_minutes" rules={[{ required: true, min: 1, type: 'number' }]}>
              <InputNumber min={1} style={{ width: 120 }} />
            </Form.Item>
            <Form.Item label="执行 Agent（可选）" name="agent_code">
              <Select
                style={{ width: 200 }}
                placeholder="不选则使用项目默认Agent"
                allowClear
                options={agents.map((a) => ({ label: `${a.name} (${a.agent_code})`, value: a.agent_code }))}
              />
            </Form.Item>
            <Form.Item label="需求类型" name="requirement_type" rules={[{ required: true }]}>
              <Select style={{ width: 220 }} options={dynamicRequirementTypeOptions} />
            </Form.Item>
            <Form.Item label="启用" name="enabled" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Space>

          <Form.Item label="模板">
            <Space>
              <Select
                placeholder="选择模板..."
                style={{ width: 240 }}
                allowClear
                options={templates.map((t) => ({ label: t.name, value: t.id }))}
                onChange={(value) => value && handleApplyTemplate(value as string)}
                dropdownRender={(menu) => (
                  <div>
                    {menu}
                  </div>
                )}
                optionRender={(option) => (
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <span>{option.label}</span>
                    <Tooltip title="删除模板">
                      <Button
                        type="text"
                        danger
                        size="small"
                        icon={<MinusCircleOutlined />}
                        onClick={(e) => handleDeleteTemplate(e, option.value as string)}
                      />
                    </Tooltip>
                  </div>
                )}
              />
              <Button icon={<SaveOutlined />} onClick={() => setTemplateModalOpen(true)}>
                保存为模板
              </Button>
            </Space>
          </Form.Item>

          <Form.Item
            name="md_content"
            label={
              <Space>
                <FileTextOutlined />
                <span>心跳模板</span>
              </Space>
            }
            extra="使用模板变量：\${project.id}, \${project.name}, \${project.git_repo_url}, \${project.default_branch}, \${timestamp}"
          >
            <Input.TextArea
              rows={20}
              style={{ fontFamily: 'monospace' }}
              placeholder="输入心跳模板内容..."
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="保存为心跳模板"
        open={templateModalOpen}
        onOk={handleSaveTemplate}
        onCancel={() => { setTemplateModalOpen(false); templateForm.resetFields(); }}
        confirmLoading={savingTemplate}
        destroyOnClose
      >
        <Form form={templateForm} layout="vertical">
          <Form.Item label="模板名称" name="name" rules={[{ required: true, message: '请输入模板名称' }]}>
            <Input placeholder="例如：PR检查模板" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`执行记录 - ${runsHeartbeatName}`}
        open={runsModalOpen}
        onCancel={() => setRunsModalOpen(false)}
        footer={null}
        width={900}
      >
        <Table<HeartbeatRunRecord>
          rowKey="requirement_id"
          dataSource={runs}
          loading={runsLoading}
          size="small"
          pagination={{ pageSize: 10 }}
          columns={[
            {
              title: '触发来源',
              dataIndex: 'trigger_source',
              key: 'trigger_source',
              width: 110,
              render: (source: string) => <Tag color="blue">{source || 'unknown'}</Tag>,
            },
            {
              title: '状态',
              dataIndex: 'status',
              key: 'status',
              width: 100,
              render: (status: string) => <Tag color={status === 'failed' ? 'red' : 'green'}>{status}</Tag>,
            },
            {
              title: '标题',
              dataIndex: 'title',
              key: 'title',
            },
            {
              title: '最近错误',
              dataIndex: 'last_error',
              key: 'last_error',
              render: (err: string) => err || '-',
            },
            {
              title: '错误分类',
              dataIndex: 'error_category',
              key: 'error_category',
              width: 120,
              render: (value: string) => <Tag color={value === 'none' ? 'default' : 'orange'}>{value || 'none'}</Tag>,
            },
            {
              title: '创建时间',
              dataIndex: 'created_at',
              key: 'created_at',
              width: 180,
              render: (value: number) => new Date(value).toLocaleString(),
            },
          ]}
        />
      </Modal>
    </div>
  );
};
