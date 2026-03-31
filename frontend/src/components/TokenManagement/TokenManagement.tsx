import React, { useEffect, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Modal,
  Popconfirm,
  Space,
  Table,
  Tag,
  message,
  Typography,
  Alert,
} from 'antd';
import { PlusOutlined, DeleteOutlined, WarningOutlined } from '@ant-design/icons';
import { createToken, deleteToken, listTokens } from '../../api/authApi';
import type { UserToken, CreateTokenRequest, CreateTokenResponse } from '../../types/user';

const { Text } = Typography;

export const TokenManagement: React.FC = () => {
  const [tokens, setTokens] = useState<UserToken[]>([]);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [newToken, setNewToken] = useState<CreateTokenResponse | null>(null);
  const [createForm] = Form.useForm();

  const fetchTokens = async () => {
    setLoading(true);
    try {
      const data = await listTokens();
      setTokens(data);
    } catch (_error) {
      message.error('获取Token列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTokens();
  }, []);

  const handleCreate = async (values: { name: string; description: string; expires_in_days: number }) => {
    try {
      const request: CreateTokenRequest = {
        name: values.name,
        description: values.description,
        expires_in_days: values.expires_in_days,
      };
      const response = await createToken(request);
      setNewToken(response);
      message.success('Token创建成功，请立即复制保存！');
      setCreateOpen(false);
      createForm.resetFields();
      fetchTokens();
    } catch (_error) {
      message.error('创建Token失败');
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteToken(id);
      message.success('删除Token成功');
      fetchTokens();
    } catch (_error) {
      message.error('删除Token失败');
    }
  };

  const handleCloseNewTokenModal = () => {
    setNewToken(null);
  };

  const formatExpiration = (expiresAt?: number): React.ReactNode => {
    if (!expiresAt) {
      return <Tag color="green">永久</Tag>;
    }
    const now = Date.now();
    if (expiresAt < now) {
      return <Tag color="red">已过期</Tag>;
    }
    const days = Math.ceil((expiresAt - now) / (1000 * 60 * 60 * 24));
    if (days <= 30) {
      return <Tag color="orange">{days}天后过期</Tag>;
    }
    return <Tag color="blue">{Math.floor(days / 30)}个月后过期</Tag>;
  };

  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '状态',
      key: 'status',
      render: (_: unknown, record: UserToken) => (
        <Space>
          {record.is_expired ? (
            <Tag color="red">已过期</Tag>
          ) : !record.is_active ? (
            <Tag color="default">已禁用</Tag>
          ) : (
            <Tag color="success">正常</Tag>
          )}
        </Space>
      ),
    },
    {
      title: '过期时间',
      key: 'expires_at',
      render: (_: unknown, record: UserToken) => formatExpiration(record.expires_at),
    },
    {
      title: '最后使用',
      key: 'last_used_at',
      render: (_: unknown, record: UserToken) =>
        record.last_used_at ? (
          <Text type="secondary">
            {new Date(record.last_used_at).toLocaleString('zh-CN')}
          </Text>
        ) : (
          <Text type="secondary">从未使用</Text>
        ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (createdAt: number) => new Date(createdAt).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: UserToken) => (
        <Popconfirm
          title="确认删除此Token？"
          description="删除后，使用此Token的API调用将立即失效"
          onConfirm={() => handleDelete(record.id)}
          okText="确认删除"
          cancelText="取消"
          okButtonProps={{ danger: true }}
        >
          <Button danger size="small" icon={<DeleteOutlined />}>
            删除
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <Card title="API Token 管理" extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
      创建Token
    </Button>}>
      {tokens.length === 0 && !loading ? (
        <Alert
          message="暂无API Token"
          description="创建Token后，可以用于API调用，无需每次登录"
          type="info"
          showIcon
        />
      ) : (
        <Table
          dataSource={tokens}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={false}
        />
      )}

      {/* 创建Token弹窗 */}
      <Modal
        title="创建API Token"
        open={createOpen}
        onCancel={() => {
          setCreateOpen(false);
          createForm.resetFields();
        }}
        footer={null}
      >
        <Form form={createForm} layout="vertical" onFinish={handleCreate}>
          <Form.Item
            name="name"
            label="Token名称"
            rules={[{ required: true, message: '请输入Token名称' }]}
          >
            <Input placeholder="例如：生产环境API" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea placeholder="描述此Token的用途" rows={2} />
          </Form.Item>
          <Form.Item
            name="expires_in_days"
            label="过期时间"
            initialValue={30}
            rules={[{ required: true }]}
          >
            <Input type="number" min={0} placeholder="0表示永久不过期" />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                创建
              </Button>
              <Button onClick={() => setCreateOpen(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 显示新创建的Token */}
      <Modal
        title={
          <Space>
            <WarningOutlined style={{ color: '#faad14' }} />
            Token创建成功
          </Space>
        }
        open={!!newToken}
        onCancel={handleCloseNewTokenModal}
        footer={
          <Button type="primary" onClick={handleCloseNewTokenModal}>
            我已保存
          </Button>
        }
      >
        <Alert
          message="请立即复制并保存此Token"
          description="Token只会显示一次，之后无法再次查看。如果丢失，请删除后重新创建。"
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <Input.Group compact>
          <Input value={newToken?.token || ''} readOnly style={{ width: 'calc(100% - 100px)' }} />
          <Button
            onClick={() => {
              navigator.clipboard.writeText(newToken?.token || '');
              message.success('已复制到剪贴板');
            }}
          >
            复制
          </Button>
        </Input.Group>
        <div style={{ marginTop: 16 }}>
          <Text type="secondary">Token名称: {newToken?.name}</Text>
          <br />
          <Text type="secondary">
            过期时间:{' '}
            {newToken?.expires_at
              ? new Date(newToken.expires_at).toLocaleString('zh-CN')
              : '永久'}
          </Text>
        </div>
      </Modal>
    </Card>
  );
};

export default TokenManagement;
