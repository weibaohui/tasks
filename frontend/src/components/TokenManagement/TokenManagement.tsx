import React, { useEffect, useState } from 'react';
import {
  Button,
  Form,
  Input,
  Modal,
  Popconfirm,
  Radio,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
  Tooltip,
  Empty,
} from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  CopyOutlined,
  CheckOutlined,
  QuestionCircleOutlined,
} from '@ant-design/icons';
import { createToken, deleteToken, listTokens } from '../../api/authApi';
import type { UserToken, CreateTokenRequest } from '../../types/user';

const { Text, Paragraph, Title } = Typography;

export const TokenManagement: React.FC = () => {
  const [tokens, setTokens] = useState<UserToken[]>([]);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm] = Form.useForm();
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [tokenResult, setTokenResult] = useState<{ token: string; name: string } | null>(null);
  const [guideOpen, setGuideOpen] = useState(false);

  const serverUrl = `${window.location.origin}/api/v1`;

  const fetchTokens = async () => {
    setLoading(true);
    try {
      const data = await listTokens();
      setTokens(data);
    } catch {
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
      const result = await createToken(request);
      message.success('Token创建成功');
      setCreateOpen(false);
      createForm.resetFields();
      fetchTokens();
      setTokenResult({ token: result.token, name: result.name });
    } catch {
      message.error('创建Token失败');
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteToken(id);
      message.success('删除Token成功');
      fetchTokens();
    } catch {
      message.error('删除Token失败');
    }
  };

  const copyToClipboard = async (text: string, key: string, successMsg: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedKey(key);
      message.success(successMsg);
      window.setTimeout(() => setCopiedKey(null), 2000);
    } catch {
      message.error('复制失败，请手动复制');
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
      render: (_: unknown, record: UserToken) => (
        <Space>
          <Tooltip title="复制 Token">
            <Button
              size="small"
              icon={copiedKey === `token-${record.id}` ? <CheckOutlined /> : <CopyOutlined />}
              onClick={() => copyToClipboard(
                record.token_value || '',
                `token-${record.id}`,
                '已复制 Token 到剪贴板',
              )}
            >
              {copiedKey === `token-${record.id}` ? '已复制' : '复制Token'}
            </Button>
          </Tooltip>
          <Tooltip title="复制 taskmanager auth 命令">
            <Button
              size="small"
              icon={copiedKey === `cmd-${record.id}` ? <CheckOutlined /> : <CopyOutlined />}
              onClick={() => copyToClipboard(
                `taskmanager auth ${serverUrl} ${record.token_value || ''}`,
                `cmd-${record.id}`,
                '已复制命令到剪贴板',
              )}
            >
              {copiedKey === `cmd-${record.id}` ? '已复制' : '复制命令'}
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
      ),
    },
  ];

  return (
    <div style={{ background: '#fff', borderRadius: 8, padding: 24 }}>
      {/* 标题区 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 24 }}>
        <div>
          <Title level={4} style={{ margin: 0 }}>API Token</Title>
          <Text type="secondary">管理 API Token，用于 CLI 认证和 API 调用</Text>
        </div>
        <Space>
          <Button icon={<QuestionCircleOutlined />} onClick={() => setGuideOpen(true)}>
            使用说明
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            创建 Token
          </Button>
        </Space>
      </div>

      {/* 表格区 */}
      {tokens.length === 0 && !loading ? (
        <Empty
          description="暂无 API Token"
          style={{ padding: '60px 0' }}
        >
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            创建第一个 Token
          </Button>
        </Empty>
      ) : (
        <Table
          dataSource={tokens}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={false}
        />
      )}

      {/* CLI 使用说明弹窗 */}
      <Modal
        title="CLI 使用说明"
        open={guideOpen}
        onCancel={() => setGuideOpen(false)}
        footer={<Button onClick={() => setGuideOpen(false)}>关闭</Button>}
        width={560}
      >
        <Paragraph>
          创建 Token 后，使用以下命令配置 CLI 认证：
        </Paragraph>
        <div style={{
          padding: '12px 16px',
          backgroundColor: '#f5f5f5',
          borderRadius: 6,
          fontFamily: 'monospace',
          fontSize: 13,
          marginBottom: 16,
          border: '1px solid #d9d9d9',
        }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span>taskmanager auth {serverUrl} {'<TOKEN>'}</span>
            <Button
              size="small"
              type="link"
              icon={copiedKey === 'guide-cmd' ? <CheckOutlined /> : <CopyOutlined />}
              onClick={() => copyToClipboard(
                `taskmanager auth ${serverUrl} YOUR_TOKEN`,
                'guide-cmd',
                '已复制命令模板到剪贴板',
              )}
            />
          </div>
        </div>
        <Paragraph type="secondary" style={{ fontSize: 13 }}>
          将 {'<TOKEN>'} 替换为你创建的 Token 值。Token 在创建时仅显示一次，请妥善保管。
        </Paragraph>
        <Paragraph type="secondary" style={{ fontSize: 13 }}>
          也可以在 Token 列表中直接点击"复制命令"按钮获取完整命令。
        </Paragraph>
      </Modal>

      {/* 创建Token弹窗 */}
      <Modal
        title="创建 API Token"
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
            label="Token 名称"
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

      {/* Token 创建成功弹窗 */}
      <Modal
        title="Token 创建成功"
        open={!!tokenResult}
        onCancel={() => setTokenResult(null)}
        footer={<Button onClick={() => setTokenResult(null)}>关闭</Button>}
        width={560}
      >
        <div style={{ padding: '12px 16px', backgroundColor: '#fffbe6', borderRadius: 6, border: '1px solid #ffe58f', marginBottom: 16 }}>
          <Text style={{ color: '#ad6800' }}>请立即复制并妥善保管 Token，关闭后无法再次查看！</Text>
        </div>
        <div style={{ marginBottom: 12 }}>
          <Text type="secondary">Token 名称：{tokenResult?.name}</Text>
        </div>
        <div style={{
          padding: '8px 12px',
          backgroundColor: '#f5f5f5',
          borderRadius: 6,
          fontFamily: 'monospace',
          fontSize: 13,
          wordBreak: 'break-all',
          marginBottom: 16,
          border: '1px solid #d9d9d9',
        }}>
          {tokenResult?.token}
        </div>
        <Space>
          <Button
            type="primary"
            icon={copiedKey === 'token' ? <CheckOutlined /> : <CopyOutlined />}
            onClick={() => copyToClipboard(tokenResult?.token || '', 'token', '已复制 Token 到剪贴板')}
          >
            {copiedKey === 'token' ? '已复制 Token' : '复制 Token'}
          </Button>
          <Button
            icon={copiedKey === 'full-cmd' ? <CheckOutlined /> : <CopyOutlined />}
            onClick={() => copyToClipboard(
              `taskmanager auth ${serverUrl} ${tokenResult?.token || ''}`,
              'full-cmd',
              '已复制完整命令到剪贴板',
            )}
          >
            {copiedKey === 'full-cmd' ? '已复制命令' : '复制命令'}
          </Button>
        </Space>
      </Modal>
    </div>
  );
};

export default TokenManagement;
