/**
 * Requirement Type Management Page
 */
import React, { useEffect, useState } from 'react';
import { Card, Typography, Space, Button, Table, Popconfirm, message, Tag } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { requirementTypeApi, type RequirementType, type CreateRequirementTypeRequest } from '../../api/requirementTypeApi';

const { Title } = Typography;

// 内置需求类型代码，这些类型不允许删除
const BUILT_IN_TYPES = ['normal', 'heartbeat'];

// 检查是否为内置类型
const isBuiltInType = (code: string): boolean => BUILT_IN_TYPES.includes(code);

interface RequirementTypeManagementPageProps {
  projectId: string;
}

export const RequirementTypeManagementPage: React.FC<RequirementTypeManagementPageProps> = ({ projectId }) => {
  const [types, setTypes] = useState<RequirementType[]>([]);
  const [loading, setLoading] = useState(false);
  const [createLoading, setCreateLoading] = useState(false);
  const [newCode, setNewCode] = useState('');
  const [newName, setNewName] = useState('');

  const fetchTypes = async () => {
    setLoading(true);
    try {
      const data = await requirementTypeApi.list(projectId);
      setTypes(data);
    } catch (error) {
      message.error('获取需求类型列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTypes();
  }, [projectId]);

  const handleCreate = async () => {
    if (!newCode.trim() || !newName.trim()) {
      message.warning('请输入类型代码和名称');
      return;
    }

    // 检查是否为内置类型代码
    if (isBuiltInType(newCode.trim())) {
      message.warning('不能使用内置类型代码');
      return;
    }

    setCreateLoading(true);
    try {
      const data: CreateRequirementTypeRequest = {
        project_id: projectId,
        code: newCode.trim(),
        name: newName.trim(),
      };
      await requirementTypeApi.create(data);
      message.success('创建成功');
      setNewCode('');
      setNewName('');
      fetchTypes();
    } catch (error) {
      message.error('创建失败');
    } finally {
      setCreateLoading(false);
    }
  };

  const handleDelete = async (type: RequirementType) => {
    if (isBuiltInType(type.code)) {
      message.error('内置类型不能删除');
      return;
    }

    try {
      await requirementTypeApi.delete(type.id);
      message.success('删除成功');
      fetchTypes();
    } catch (error: any) {
      if (error?.response?.status === 403) {
        message.error('内置类型不能删除');
      } else {
        message.error('删除失败');
      }
    }
  };

  const columns = [
      {
            title: '操作',
            key: 'action',
            render: (_: unknown, record: RequirementType) => {
              if (isBuiltInType(record.code)) {
                return <span style={{ color: '#999' }}>不可删除</span>;
              }
              return (
                <Popconfirm
                  title="确定删除此需求类型？"
                  onConfirm={() => handleDelete(record)}
                  okText="确定"
                  cancelText="取消"
                >
                  <Button danger type="link" size="small" style={{ padding: 0 }}>
                    删除
                  </Button>
                </Popconfirm>
              );
            },
              width: 100,
              fixed: 'left' as const
          },
    {
      title: '代码',
      dataIndex: 'code',
      key: 'code',
      render: (code: string) => (
        <Space>
          <span>{code}</span>
          {isBuiltInType(code) && <Tag color="blue">内置</Tag>}
        </Space>
      ),
    },
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
    },
    {
      title: '颜色',
      dataIndex: 'color',
      key: 'color',
      render: (color: string) => color ? <Tag color={color}>{color}</Tag> : '-',
    }
  ];

  return (
    <Card
      title={<Title level={5} style={{ margin: 0 }}>需求类型管理</Title>}
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={fetchTypes} loading={loading}>
            刷新
          </Button>
        </Space>
      }
    >
      <Space style={{ marginBottom: 16 }} wrap>
        <input
          placeholder="类型代码"
          value={newCode}
          onChange={(e) => setNewCode(e.target.value)}
          style={{ width: 120, padding: '4px 8px', borderRadius: 4, border: '1px solid #d9d9d9' }}
        />
        <input
          placeholder="类型名称"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          style={{ width: 120, padding: '4px 8px', borderRadius: 4, border: '1px solid #d9d9d9' }}
        />
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={handleCreate}
          loading={createLoading}
        >
          添加
        </Button>
      </Space>

      <Table
        columns={columns}
        dataSource={types}
        rowKey="id"
        loading={loading}
        pagination={false}
      />
    </Card>
  );
};

export default RequirementTypeManagementPage;