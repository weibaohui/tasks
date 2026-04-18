import React, { useEffect, useState } from 'react';
import { Alert, Button, Card, Form, message, Select, Space, Table, Tag, Tooltip } from 'antd';
import { LinkOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { Project } from '../../types/projectRequirement';
import {
  deleteProjectStateMachine,
  listProjectStateMachines,
  setProjectStateMachine,
  type ProjectStateMachineMapping,
} from '../../api/projectStateMachineApi';
import { listStateMachines } from '../../api/stateMachineApi';
import { requirementTypeApi, type RequirementType } from '../../api/requirementTypeApi';
import type { StateMachine } from '../../types/stateMachine';

interface ProjectStateMachinePanelProps {
  project: Project | null;
}

export const ProjectStateMachinePanel: React.FC<ProjectStateMachinePanelProps> = ({ project }) => {
  const [form] = Form.useForm();
  const [mappings, setMappings] = useState<ProjectStateMachineMapping[]>([]);
  const [stateMachines, setStateMachines] = useState<StateMachine[]>([]);
  const [requirementTypes, setRequirementTypes] = useState<RequirementType[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const fetchStateMachines = async () => {
    try {
      const data = await listStateMachines();
      setStateMachines(data);
    } catch {
      message.error('获取状态机列表失败');
    }
  };

  const fetchMappings = async () => {
    if (!project?.id) return;
    setLoading(true);
    try {
      const data = await listProjectStateMachines(project.id);
      setMappings(data);
    } catch {
      message.error('获取项目状态机配置失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchRequirementTypes = async () => {
    if (!project?.id) return;
    try {
      const data = await requirementTypeApi.list(project.id);
      setRequirementTypes(data);
    } catch {
      setRequirementTypes([]);
    }
  };

  const getTypeConfig = (code: string): { label: string; color: string; description: string } => {
    const apiType = requirementTypes.find((t) => t.code === code);
    if (apiType) {
      return {
        label: apiType.name || code,
        color: apiType.color || 'default',
        description: apiType.description || '',
      };
    }
    return { label: code, color: 'default', description: '' };
  };

  useEffect(() => {
    if (project?.id) {
      void fetchStateMachines();
      void fetchMappings();
      void fetchRequirementTypes();
    }
  }, [project?.id]);

  const handleSubmit = async (values: { requirement_type: string; state_machine_id: string }) => {
    if (!project?.id) return;
    setSubmitting(true);
    try {
      await setProjectStateMachine(project.id, {
        requirement_type: values.requirement_type,
        state_machine_id: values.state_machine_id,
      });
      message.success('状态机配置已保存');
      form.resetFields();
      void fetchMappings();
    } catch {
      message.error('保存失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteProjectStateMachine(id);
      message.success('已删除');
      void fetchMappings();
    } catch {
      message.error('删除失败');
    }
  };

  const columns: ColumnsType<ProjectStateMachineMapping> = [
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: ProjectStateMachineMapping) => (
        <Button danger onClick={() => handleDelete(record.id)} type="link" size="small" style={{ padding: 0 }}>
          删除
        </Button>
      ),
      width: 100,
      fixed: 'left' as const,
    },
    {
      title: '需求类型',
      dataIndex: 'requirement_type',
      key: 'requirement_type',
      width: 120,
      render: (type: string) => {
        const config = getTypeConfig(type);
        return (
          <Tooltip title={config.description}>
            <Tag color={config.color}>{config.label}</Tag>
          </Tooltip>
        );
      },
    },
    {
      title: '关联状态机',
      dataIndex: 'state_machine_name',
      key: 'state_machine_name',
      render: (_text: string, record: ProjectStateMachineMapping) => (
        <Space>
          <span>{record.state_machine_name || '未知状态机'}</span>
          <Tooltip title="点击查看状态机详情">
            <LinkOutlined
              style={{ color: '#1890ff', cursor: 'pointer' }}
              onClick={() => {
                window.open(`/state-machines?id=${record.state_machine_id}`, '_blank');
              }}
            />
          </Tooltip>
        </Space>
      ),
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      key: 'updated_at',
      width: 180,
      render: (timestamp: number) => new Date(timestamp).toLocaleString(),
    },
  ];

  const getAvailableTypes = () => {
    const configuredTypes = new Set(mappings.map((m) => m.requirement_type));
    const allTypes: Array<[string, { label: string; color: string; description: string }]> = [];

    requirementTypes.forEach((t) => {
      if (!configuredTypes.has(t.code)) {
        allTypes.push([t.code, { label: t.name, color: t.color || 'default', description: t.description || '' }]);
      }
    });

    return allTypes;
  };

  const availableTypes = getAvailableTypes();

  if (!project) {
    return <Alert type="info" showIcon message="请先选择项目后再配置状态机" />;
  }

  return (
    <Space direction="vertical" style={{ width: '100%' }}>
      <Card size="small" title="添加状态机关联">
        <Form form={form} layout="inline" onFinish={handleSubmit}>
          <Form.Item
            name="requirement_type"
            rules={[{ required: true, message: '请选择需求类型' }]}
            style={{ width: 200 }}
          >
            <Select placeholder="选择需求类型" disabled={availableTypes.length === 0}>
              {availableTypes.map(([type, config]) => (
                <Select.Option key={type} value={type}>
                  <Tooltip title={config.description}>
                    <Tag color={config.color}>{config.label}</Tag>
                  </Tooltip>
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item
            name="state_machine_id"
            rules={[{ required: true, message: '请选择状态机' }]}
            style={{ width: 300 }}
          >
            <Select placeholder="选择状态机" showSearch optionFilterProp="label" optionLabelProp="label">
              {stateMachines.map((sm) => (
                <Select.Option key={sm.id} value={sm.id} label={sm.name}>
                  <div>
                    <div>{sm.name}</div>
                    <div style={{ fontSize: 12, color: '#999' }}>{sm.description || '无描述'}</div>
                  </div>
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={submitting} disabled={availableTypes.length === 0}>
              保存
            </Button>
          </Form.Item>
        </Form>
        {availableTypes.length === 0 && (
          <div style={{ marginTop: 8, color: '#999' }}>所有需求类型都已配置状态机</div>
        )}
      </Card>

      <Card size="small" title="已配置的状态机关联">
        <Space style={{ marginBottom: 8 }}>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchMappings()} size="small">
            刷新
          </Button>
        </Space>
        <Table
          columns={columns}
          dataSource={mappings}
          rowKey="id"
          loading={loading}
          pagination={false}
          size="small"
          locale={{ emptyText: '暂无配置，请上方添加' }}
        />
      </Card>
    </Space>
  );
};