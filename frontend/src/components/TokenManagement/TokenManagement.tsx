import React, { useEffect, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Modal,
  Popconfirm,
  Radio,
  Select,
  Space,
  Table,
  Tag,
  message,
  Typography,
  Alert,
  Tooltip,
} from 'antd';
import { PlusOutlined, DeleteOutlined, CopyOutlined, CheckOutlined } from '@ant-design/icons';
import { createToken, deleteToken, listTokens } from '../../api/authApi';
import type { UserToken, CreateTokenRequest } from '../../types/user';

const { Text, Paragraph } = Typography;

export const TokenManagement: React.FC = () => {
  const [tokens, setTokens] = useState<UserToken[]>([]);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm] = Form.useForm();
  const [copiedTokenId, setCopiedTokenId] = useState<string | null>(null);

  // 获取当前页面 URL 用于生成命令
  const getCurrentServerUrl = () => {
    return window.location.origin;
  };

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

  const handleCreate = async (values: { name: string; description: string; expires_type: 'permanent' | 'days'; expires_days?: number }) => {
    try {
      const request: CreateTokenRequest = {
        name: values.name,
        description: values.description || '',
        expires_in_days: values.expires_type === 'permanent' ? 0 : (values.expires_days || 30),
      };
      await createToken(request);
      message.success('Token创建成功');
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
      render: (_: unknown, record: UserToken) => {
        const fullCommand = `tm auth ${getCurrentServerUrl()}/api/v1 YOUR_TOKEN`;
        const isCopied = copiedTokenId === record.id;

        return (
          <Space>
            <Tooltip title="复制 tm auth 命令">
              <Button
                size="small"
                icon={isCopied ? <CheckOutlined /> : <CopyOutlined />}
                onClick={() => {
                  navigator.clipboard.writeText(fullCommand);
                  setCopiedTokenId(record.id);
                  message.success('已复制 tm auth 命令到剪贴板');
                  setTimeout(() => setCopiedTokenId(null), 2000);
                }}
              >
                {isCopied ? '已复制' : '复制命令'}
              </Button>
            </Tooltip>
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
          </Space>
        );
      },
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
        <>
          <Alert
            message="CLI 使用方法"
            description={
              <Paragraph style={{ marginBottom: 0 }}>
                使用以下命令配置 CLI 认证：<br />
                <Text code copyable={{ text: `tm auth ${getCurrentServerUrl()}/api/v1 YOUR_TOKEN` }}>
                  tm auth {getCurrentServerUrl()}/api/v1 YOUR_TOKEN
                </Text>
              </Paragraph>
            }
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
          />
          <Table
            dataSource={tokens}
            columns={columns}
            rowKey="id"
            loading={loading}
            pagination={false}
          />
        </>
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
            name="expires_type"
            label="过期时间"
            initialValue="days"
            rules={[{ required: true }]}
          >
            <Radio.Group>
              <Radio value="days">自定义天数</Radio>
              <Radio value="permanent">永不过期</Radio>
            </Radio.Group>
          </Form.Item>
          <Form.Item
            noStyle
            shouldUpdate={(prev, curr) => prev.expires_type !== curr.expires_type}
          >
            {({ getFieldValue }) =>
              getFieldValue('expires_type') === 'days' && (
                <Form.Item
                  name="expires_days"
                  label="天数"
                  initialValue={30}
                  rules={[{ required: true, message: '请选择天数' }]}
                >
                  <Select
                    placeholder="选择过期天数"
                    options={[
                      { label: '7天', value: 7 },
                      { label: '30天', value: 30 },
                      { label: '90天', value: 90 },
                      { label: '180天', value: 180 },
                      { label: '365天', value: 365 },
                    ]}
                  />
                </Form.Item>
              )
            }
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
    </Card>
  );
};

export default TokenManagement;
