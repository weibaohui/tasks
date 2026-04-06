/**
 * Requirement Type Management Page
 */
import React, { useEffect, useState } from 'react';
import { Card, Typography, Space, Button, Table, Popconfirm, message } from 'antd';
import { PlusOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons';
import { requirementTypeApi, type RequirementType, type CreateRequirementTypeRequest } from '../../api/requirementTypeApi';

const { Title } = Typography;

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

  const columns = [
    {
      title: '代码',
      dataIndex: 'code',
      key: 'code',
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
      title: '图标',
      dataIndex: 'icon',
      key: 'icon',
    },
    {
      title: '颜色',
      dataIndex: 'color',
      key: 'color',
    },
    {
      title: '排序',
      dataIndex: 'sort_order',
      key: 'sort_order',
    },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown) => (
        <Popconfirm
          title="确定删除此需求类型？"
          onConfirm={() => message.info('删除功能待实现')}
          okText="确定"
          cancelText="取消"
        >
          <Button size="small" danger icon={<DeleteOutlined />}>
            删除
          </Button>
        </Popconfirm>
      ),
    },
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