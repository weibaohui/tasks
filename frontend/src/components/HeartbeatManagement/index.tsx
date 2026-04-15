import React, { useEffect, useState } from 'react';
import { Button, Form, Input, InputNumber, Modal, Select, Space, Switch, Table, Tag, message, Tooltip } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import type { Heartbeat } from '../../types/heartbeat';
import type { Agent } from '../../types/agent';
import { listHeartbeats, createHeartbeat, updateHeartbeat, deleteHeartbeat } from '../../api/heartbeatApi';
import { HeartbeatTemplateEditor } from '../HeartbeatTemplate';

interface HeartbeatManagementProps {
  projectId: string;
  agents: Agent[];
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

export const HeartbeatManagement: React.FC<HeartbeatManagementProps> = ({ projectId, agents }) => {
  const [heartbeats, setHeartbeats] = useState<Heartbeat[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingHeartbeat, setEditingHeartbeat] = useState<Heartbeat | null>(null);
  const [form] = Form.useForm();

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

  const columns = [
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
    },
    {
      title: '需求类型',
      dataIndex: 'requirement_type',
      key: 'requirement_type',
      width: 110,
      render: (type: string) => (
        <Tag color={typeColorMap[type] || 'default'}>
          {requirementTypeOptions.find((o) => o.value === type)?.label || type}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (enabled: boolean) => (enabled ? <Tag color="success">启用</Tag> : <Tag>禁用</Tag>),
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
            <Form.Item label="执行 Agent" name="agent_code" rules={[{ required: true, message: '请选择Agent' }]}>
              <Select
                style={{ width: 200 }}
                placeholder="选择Agent"
                options={agents.map((a) => ({ label: `${a.name} (${a.agent_code})`, value: a.agent_code }))}
              />
            </Form.Item>
            <Form.Item label="需求类型" name="requirement_type" rules={[{ required: true }]}>
              <Select style={{ width: 140 }} options={requirementTypeOptions} />
            </Form.Item>
            <Form.Item label="启用" name="enabled" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Space>
          <Form.Item name="md_content" hidden>
            <Input />
          </Form.Item>
          <HeartbeatTemplateEditor
            value={form.getFieldValue('md_content')}
            onChange={(value) => form.setFieldValue('md_content', value)}
          />
        </Form>
      </Modal>
    </div>
  );
};
