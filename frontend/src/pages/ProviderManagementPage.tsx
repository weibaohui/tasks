/**
 * LLM 配置页面
 * 管理 LLM Provider（新增、编辑、删除、测试连接）
 */
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
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
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { createProvider, deleteProvider, listProviders, testProviderConnection, updateProvider } from '../api/providerApi';
import { useAuthStore } from '../stores/authStore';
import type { CreateProviderRequest, LLMProvider, ProviderModelInfo, UpdateProviderRequest } from '../types/provider';

type ProviderFormValues = {
  provider_key: string;
  provider_name: string;
  api_base: string;
  api_key: string;
  api_type: string;
  is_default: boolean;
  is_active: boolean;
  priority: number;
  auto_merge: boolean;
  default_model: string;
  supported_models: ProviderModelInfo[];
  extra_headers_json: string;
};

/**
 * 将 Extra Headers 的 JSON 文本解析为字符串键值对
 */
function toExtraHeaders(extraHeadersJson: string): Record<string, string> {
  if (!extraHeadersJson.trim()) {
    return {};
  }
  const parsed = JSON.parse(extraHeadersJson) as Record<string, unknown>;
  const out: Record<string, string> = {};
  for (const [k, v] of Object.entries(parsed)) {
    if (typeof v === 'string') {
      out[k] = v;
    }
  }
  return out;
}

/**
 * 将 Extra Headers 转换为用于表单编辑的 JSON 文本
 */
function toExtraHeadersJson(extraHeaders: Record<string, string> | undefined): string {
  if (!extraHeaders || Object.keys(extraHeaders).length === 0) {
    return '';
  }
  return JSON.stringify(extraHeaders, null, 2);
}

export const ProviderManagementPage: React.FC = () => {
  const { user } = useAuthStore();
  const userCode = user?.user_code || '';
  const [items, setItems] = useState<LLMProvider[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<LLMProvider | null>(null);
  const [form] = Form.useForm<ProviderFormValues>();

  /**
   * 拉取 Provider 列表
   */
  const fetchList = useCallback(async () => {
    if (!userCode) {
      setItems([]);
      return;
    }
    setLoading(true);
    try {
      const data = await listProviders(userCode);
      setItems(data);
    } catch (_error) {
      message.error('获取 Provider 列表失败');
    } finally {
      setLoading(false);
    }
  }, [userCode]);

  /**
   * 删除 Provider
   */
  const handleDelete = useCallback(async (id: string) => {
    try {
      await deleteProvider(id);
      message.success('删除成功');
      await fetchList();
    } catch (_error) {
      message.error('删除失败');
    }
  }, [fetchList]);

  /**
   * 保存（创建/更新）Provider
   */
  const handleSubmit = useCallback(async (values: ProviderFormValues) => {
    if (!userCode) {
      message.error('未获取到用户信息，请重新登录');
      return;
    }
    setSaving(true);
    try {
      const extraHeaders = toExtraHeaders(values.extra_headers_json);
      if (editing) {
        const req: UpdateProviderRequest = {
          provider_key: values.provider_key,
          provider_name: values.provider_name,
          api_base: values.api_base,
          api_key: values.api_key ? values.api_key : undefined,
          api_type: values.api_type,
          is_default: values.is_default,
          is_active: values.is_active,
          priority: values.priority,
          auto_merge: values.auto_merge,
          default_model: values.default_model,
          supported_models: values.supported_models || [],
          extra_headers: extraHeaders,
        };
        await updateProvider(editing.id, req);
        message.success('更新成功');
      } else {
        const req: CreateProviderRequest = {
          user_code: userCode,
          provider_key: values.provider_key,
          provider_name: values.provider_name,
          api_base: values.api_base,
          api_key: values.api_key,
          api_type: values.api_type,
          is_default: values.is_default,
          priority: values.priority,
          auto_merge: values.auto_merge,
          default_model: values.default_model,
          supported_models: values.supported_models || [],
          extra_headers: extraHeaders,
        };
        await createProvider(req);
        message.success('创建成功');
      }
      setCreateOpen(false);
      setEditing(null);
      form.resetFields();
      await fetchList();
    } catch (error) {
      if (error instanceof Error && error.message.includes('JSON')) {
        message.error('Extra Headers 不是合法 JSON');
      } else {
        message.error('保存失败');
      }
    } finally {
      setSaving(false);
    }
  }, [editing, fetchList, form, userCode]);

  const columns: ColumnsType<LLMProvider> = useMemo(
    () => [
      {
        title: 'Key',
        dataIndex: 'provider_key',
        key: 'provider_key',
        width: 160,
        render: (v: string) => <Tag color="blue">{v}</Tag>,
      },
      {
        title: '名称',
        dataIndex: 'provider_name',
        key: 'provider_name',
        ellipsis: true,
      },
      {
        title: 'Base URL',
        dataIndex: 'api_base',
        key: 'api_base',
        ellipsis: true,
      },
      {
        title: 'API类型',
        dataIndex: 'api_type',
        key: 'api_type',
        width: 120,
        render: (v: string) => {
          if (v === 'anthropic') return <Tag color="orange">Claude</Tag>;
          return <Tag color="blue">OpenAI</Tag>;
        },
      },
      {
        title: '默认',
        dataIndex: 'is_default',
        key: 'is_default',
        width: 80,
        render: (v: boolean) => (v ? <Tag color="green">是</Tag> : <Tag>否</Tag>),
      },
      {
        title: '启用',
        dataIndex: 'is_active',
        key: 'is_active',
        width: 80,
        render: (v: boolean) => (v ? <Tag color="green">启用</Tag> : <Tag color="red">停用</Tag>),
      },
      {
        title: '默认模型',
        dataIndex: 'default_model',
        key: 'default_model',
        width: 160,
        ellipsis: true,
      },
      {
        title: '优先级',
        dataIndex: 'priority',
        key: 'priority',
        width: 90,
      },
      {
        title: '操作',
        key: 'action',
        width: 260,
        render: (_: unknown, record: LLMProvider) => (
          <Space>
            <Button
              onClick={async () => {
                try {
                  const res = await testProviderConnection(record.id);
                  if (res.success) {
                    message.success(`连接测试成功：${res.message || 'ok'}`);
                  } else {
                    message.error(`连接测试失败：${res.message || '失败'}`);
                  }
                } catch (_error) {
                  message.error('连接测试失败');
                }
              }}
            >
              测试
            </Button>
            <Button
              type="primary"
              onClick={() => {
                setEditing(record);
                setCreateOpen(true);
                form.setFieldsValue({
                  provider_key: record.provider_key,
                  provider_name: record.provider_name,
                  api_base: record.api_base,
                  api_key: '',
                  api_type: record.api_type || 'openai',
                  is_default: record.is_default,
                  is_active: record.is_active,
                  priority: record.priority,
                  auto_merge: record.auto_merge,
                  default_model: record.default_model,
                  supported_models: record.supported_models || [],
                  extra_headers_json: toExtraHeadersJson(record.extra_headers),
                });
              }}
            >
              编辑
            </Button>
            <Popconfirm title="确认删除该 Provider？" onConfirm={() => handleDelete(record.id)}>
              <Button danger>删除</Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [form, handleDelete],
  );

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  return (
    <div style={{ padding: 24 }}>
      <Card
        title={`LLM Provider 配置 (${items.length})`}
        extra={
          <Space>
            <Button onClick={() => fetchList()}>刷新</Button>
            <Button
              type="primary"
              onClick={() => {
                setEditing(null);
                setCreateOpen(true);
                form.setFieldsValue({
                  provider_key: '',
                  provider_name: '',
                  api_base: '',
                  api_key: '',
                  api_type: 'openai',
                  is_default: false,
                  is_active: true,
                  priority: 0,
                  auto_merge: true,
                  default_model: '',
                  supported_models: [],
                  extra_headers_json: '',
                });
              }}
            >
              新建 Provider
            </Button>
          </Space>
        }
      >
        <Table<LLMProvider> rowKey="id" loading={loading} dataSource={items} columns={columns} />
      </Card>

      <Modal
        title={editing ? '编辑 Provider' : '新建 Provider'}
        open={createOpen}
        onCancel={() => {
          setCreateOpen(false);
          setEditing(null);
        }}
        footer={null}
        width={860}
      >
        <Form layout="vertical" form={form} onFinish={handleSubmit}>
          <Space style={{ width: '100%' }} align="start">
            <div style={{ flex: 1 }}>
              <Form.Item label="Provider Key" name="provider_key" rules={[{ required: true, message: '请输入 Provider Key' }]}>
                <Input placeholder="例如：openai / ollama / deepseek" />
              </Form.Item>
            </div>
            <div style={{ flex: 1 }}>
              <Form.Item label="Provider 名称" name="provider_name">
                <Input placeholder="展示名称（可选）" />
              </Form.Item>
            </div>
          </Space>

          <Space style={{ width: '100%' }} align="start">
            <div style={{ flex: 1 }}>
              <Form.Item label="API Base" name="api_base">
                <Input placeholder="例如：https://api.openai.com/v1 或 http://localhost:11434" />
              </Form.Item>
            </div>
            <div style={{ flex: 1 }}>
              <Form.Item
                label="API Key"
                name="api_key"
                rules={editing ? [] : [{ required: true, message: '请输入 API Key' }]}
              >
                <Input.Password placeholder={editing ? '留空则不更新' : '请输入 API Key'} />
              </Form.Item>
            </div>
            <div style={{ width: 160 }}>
              <Form.Item label="API 类型" name="api_type" initialValue="openai">
                <Select>
                  <Select.Option value="openai">OpenAI 格式</Select.Option>
                  <Select.Option value="anthropic">Anthropic 格式</Select.Option>
                </Select>
              </Form.Item>
            </div>
          </Space>

          <Space style={{ width: '100%' }} align="start">
            <div style={{ flex: 1 }}>
              <Form.Item label="默认模型" name="default_model">
                <Input placeholder="例如：gpt-4o-mini / llama3" />
              </Form.Item>
            </div>
            <div style={{ width: 180 }}>
              <Form.Item label="优先级" name="priority" initialValue={0}>
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </div>
            <div style={{ width: 120 }}>
              <Form.Item label="设为默认" name="is_default" valuePropName="checked">
                <Switch checkedChildren="是" unCheckedChildren="否" />
              </Form.Item>
            </div>
            <div style={{ width: 120 }}>
              <Form.Item label="启用" name="is_active" valuePropName="checked">
                <Switch checkedChildren="启用" unCheckedChildren="停用" />
              </Form.Item>
            </div>
            <div style={{ width: 140 }}>
              <Form.Item label="自动合并" name="auto_merge" valuePropName="checked">
                <Switch checkedChildren="开启" unCheckedChildren="关闭" />
              </Form.Item>
            </div>
          </Space>

          <Form.List name="supported_models">
            {(fields, { add, remove }) => (
              <div style={{ marginTop: 8 }}>
                <Space style={{ marginBottom: 8 }}>
                  <Typography.Text strong>支持模型</Typography.Text>
                  <Button onClick={() => add({ id: '', name: '', max_tokens: 0 })}>新增模型</Button>
                </Space>
                {fields.map((field) => (
                  <Space key={field.key} style={{ display: 'flex', marginBottom: 8 }} align="start">
                    <Form.Item
                      label="ID"
                      name={[field.name, 'id']}
                      rules={[{ required: true, message: '请输入模型 ID' }]}
                      style={{ width: 220 }}
                    >
                      <Input placeholder="例如：gpt-4o-mini" />
                    </Form.Item>
                    <Form.Item label="名称" name={[field.name, 'name']} style={{ width: 240 }}>
                      <Input placeholder="展示名称（可选）" />
                    </Form.Item>
                    <Form.Item label="Max Tokens" name={[field.name, 'max_tokens']} style={{ width: 160 }}>
                      <InputNumber min={0} style={{ width: '100%' }} />
                    </Form.Item>
                    <Button danger onClick={() => remove(field.name)} style={{ marginTop: 30 }}>
                      删除
                    </Button>
                  </Space>
                ))}
              </div>
            )}
          </Form.List>

          <Form.Item
            label="Extra Headers（JSON）"
            name="extra_headers_json"
            tooltip='例如：{"X-Custom-Header":"xxx"}'
          >
            <Input.TextArea rows={6} placeholder="可选，必须是 JSON 对象格式" />
          </Form.Item>

          <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button
              onClick={() => {
                setCreateOpen(false);
                setEditing(null);
              }}
            >
              取消
            </Button>
            <Button type="primary" htmlType="submit" loading={saving}>
              保存
            </Button>
          </Space>
        </Form>
      </Modal>
    </div>
  );
};
