import React, { useEffect, useState } from 'react';
import { Button, Card, Form, message, Select, Space, Table, Tag, Tooltip } from 'antd';
import { LinkOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import {
  deleteProjectStateMachine,
  listProjectStateMachines,
  setProjectStateMachine,
  type ProjectStateMachineMapping,
} from '../../api/projectStateMachineApi';
import { listStateMachines } from '../../api/stateMachineApi';
import { requirementTypeApi, type RequirementType } from '../../api/requirementTypeApi';
import type { StateMachine } from '../../types/stateMachine';

interface ProjectStateMachineConfigProps {
  projectId: string;
}

// Default fallback type config when API data is not available
const defaultTypeConfig: Record<string, { label: string; color: string; description: string }> = {
  normal: {
    label: '普通需求',
    color: 'blue',
    description: '普通流程需求，需要人工触发',
  },
  heartbeat: {
    label: '心跳需求',
    color: 'green',
    description: '自动触发的心跳任务',
  },
};

export const ProjectStateMachineConfig: React.FC<ProjectStateMachineConfigProps> = ({ projectId }) => {
  const [form] = Form.useForm();
  const [mappings, setMappings] = useState<ProjectStateMachineMapping[]>([]);
  const [stateMachines, setStateMachines] = useState<StateMachine[]>([]);
  const [requirementTypes, setRequirementTypes] = useState<RequirementType[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  // 获取状态机列表
  const fetchStateMachines = async () => {
    try {
      const data = await listStateMachines();
      setStateMachines(data);
    } catch (_error) {
      message.error('获取状态机列表失败');
    }
  };

  // 获取项目状态机映射
  const fetchMappings = async () => {
    setLoading(true);
    try {
      const data = await listProjectStateMachines(projectId);
      setMappings(data);
    } catch (_error) {
      message.error('获取项目状态机配置失败');
    } finally {
      setLoading(false);
    }
  };

  // 获取需求类型列表
  const fetchRequirementTypes = async () => {
    try {
      const data = await requirementTypeApi.list(projectId);
      setRequirementTypes(data);
    } catch (_error) {
      // 使用空数组，让下面的 getTypeConfig 回退到默认配置
      setRequirementTypes([]);
    }
  };

  // 获取类型配置（优先从 API，失败时使用默认配置）
  const getTypeConfig = (code: string): { label: string; color: string; description: string } => {
    const apiType = requirementTypes.find((t) => t.code === code);
    if (apiType) {
      return {
        label: apiType.name || code,
        color: apiType.color || 'default',
        description: apiType.description || '',
      };
    }
    return defaultTypeConfig[code] || { label: code, color: 'default', description: '' };
  };

  useEffect(() => {
    fetchStateMachines();
    fetchMappings();
    fetchRequirementTypes();
  }, [projectId]);

  // 保存状态机映射
  const handleSubmit = async (values: { requirement_type: string; state_machine_id: string }) => {
    setSubmitting(true);
    try {
      await setProjectStateMachine(projectId, {
        requirement_type: values.requirement_type,
        state_machine_id: values.state_machine_id,
      });
      message.success('状态机配置已保存');
      form.resetFields();
      fetchMappings();
    } catch (_error) {
      message.error('保存失败');
    } finally {
      setSubmitting(false);
    }
  };

  // 删除状态机映射
  const handleDelete = async (id: string) => {
    try {
      await deleteProjectStateMachine(id);
      message.success('已删除');
      fetchMappings();
    } catch (_error) {
      message.error('删除失败');
    }
  };

  const columns: ColumnsType<ProjectStateMachineMapping> = [
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
                // 打开状态机管理页面
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
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: ProjectStateMachineMapping) => (
        <Button danger onClick={() => handleDelete(record.id)} type="link" size="small" style={{ padding: 0 }}>
          删除
        </Button>
      ),
        width: 100,
        fixed: 'left' as const
    },
  ];

  // 获取未配置的需求类型
  const getAvailableTypes = () => {
    const configuredTypes = new Set(mappings.map((m) => m.requirement_type));
    // 合并 API 类型和默认类型
    const allTypes: Array<[string, { label: string; color: string; description: string }]> = [];

    // 添加默认类型
    Object.entries(defaultTypeConfig).forEach(([code, config]) => {
      if (!configuredTypes.has(code)) {
        allTypes.push([code, config]);
      }
    });

    // 添加 API 返回的类型（跳过已配置的）
    requirementTypes.forEach((t) => {
      if (!configuredTypes.has(t.code) && !defaultTypeConfig[t.code]) {
        allTypes.push([t.code, { label: t.name, color: t.color || 'default', description: t.description || '' }]);
      }
    });

    return allTypes;
  };

  const availableTypes = getAvailableTypes();

  return (
    <div>
      <Card title="添加状态机关联" style={{ marginBottom: 16 }}>
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

      <Card title="已配置的状态机关联">
        <Table
          columns={columns}
          dataSource={mappings}
          rowKey="id"
          loading={loading}
          pagination={false}
          locale={{ emptyText: '暂无配置，请上方添加' }}
        />
      </Card>
    </div>
  );
};
