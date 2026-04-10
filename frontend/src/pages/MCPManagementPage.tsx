/**
 * MCP 管理页面
 * 支持服务器的增删改查、测试连接与刷新工具能力
 */
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Card, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, message } from 'antd';
import type { MCPTool } from '../types/mcp';
import type { ColumnsType } from 'antd/es/table';
import {
  createMCPServer,
  deleteMCPServer,
  getMCPErrorMessage,
  listMCPServers,
  refreshMCPServer,
  testMCPServer,
  updateMCPServer,
} from '../api/mcpApi';
import type { CreateMCPServerRequest, MCPServer, UpdateMCPServerRequest } from '../types/mcp';
import { ActionGroup } from "@/components/ActionGroup";

type FormValues = {
  code: string;
  name: string;
  description?: string;
  transport_type: 'stdio' | 'http' | 'sse';
  command?: string;
  args?: string[];
  url?: string;
  env_vars_kv?: Array<{ key: string; value: string }>;
};

/**
 * 将表单键值对转换为 env_vars 对象
 */
function kvToEnv(vars?: Array<{ key: string; value: string }>): Record<string, string> | undefined {
  if (!vars || vars.length === 0) return undefined;
  const m: Record<string, string> = {};
  for (const item of vars) {
    const k = (item.key || '').trim();
    if (!k) continue;
    m[k] = item.value || '';
  }
  return m;
}

/**
 * MCP 管理页面组件
 */
export const MCPManagementPage: React.FC = () => {
  const [items, setItems] = useState<MCPServer[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<MCPServer | null>(null);
  const [form] = Form.useForm<FormValues>();

  /**
   * 拉取列表
   */
  const fetchList = useCallback(async () => {
    setLoading(true);
    try {
      const data = await listMCPServers();
      setItems(data);
    } catch (e) {
      message.error(getMCPErrorMessage(e) || '获取 MCP 服务器列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  /**
   * 提交保存（创建/更新）
   */
  const handleSubmit = useCallback(async (values: FormValues) => {
    setSaving(true);
    try {
      if (editing) {
        const req: UpdateMCPServerRequest = {
          name: values.name,
          description: values.description,
          transport_type: values.transport_type,
          command: values.command,
          args: (values.args || []).filter(Boolean),
          url: values.url,
          env_vars: kvToEnv(values.env_vars_kv),
        };
        await updateMCPServer(editing.id, req);
        message.success('更新成功');
      } else {
        const req: CreateMCPServerRequest = {
          code: values.code,
          name: values.name,
          description: values.description,
          transport_type: values.transport_type,
          command: values.command,
          args: (values.args || []).filter(Boolean),
          url: values.url,
          env_vars: kvToEnv(values.env_vars_kv),
        };
        await createMCPServer(req);
        message.success('创建成功');
      }
      setOpen(false);
      setEditing(null);
      form.resetFields();
      await fetchList();
    } catch (e) {
      message.error(getMCPErrorMessage(e) || '保存失败');
    } finally {
      setSaving(false);
    }
  }, [editing, fetchList, form]);

  /**
   * 测试连接
   */
  const handleTest = useCallback(async (id: string) => {
    try {
      await testMCPServer(id);
      message.success('测试连接已完成');
      await fetchList();
    } catch (e) {
      message.error(getMCPErrorMessage(e) || '测试连接失败');
    }
  }, [fetchList]);

  /**
   * 刷新工具
   */
  const handleRefresh = useCallback(async (id: string) => {
    try {
      await refreshMCPServer(id);
      message.success('刷新工具能力成功');
      await fetchList();
    } catch (e) {
      message.error(getMCPErrorMessage(e) || '刷新失败');
    }
  }, [fetchList]);

  /**
   * 删除
   */
  const handleDelete = useCallback(async (id: string) => {
    try {
      await deleteMCPServer(id);
      message.success('删除成功');
      await fetchList();
    } catch (e) {
      message.error(getMCPErrorMessage(e) || '删除失败');
    }
  }, [fetchList]);

  const columns: ColumnsType<MCPServer> = useMemo(() => [
      {
            title: '操作',
            key: 'action',
            render: (_: unknown, record: MCPServer) => (
              <ActionGroup>
                <Button onClick={() => handleTest(record.id)} type="link" size="small" style={{ padding: 0 }}>测试</Button>
                <Button onClick={() => handleRefresh(record.id)} type="link" size="small" style={{ padding: 0 }}>刷新</Button>
                <Button
                  onClick={() => {
                    setEditing(record);
                    setOpen(true);
                    form.setFieldsValue({
                      code: record.code,
                      name: record.name,
                      description: record.description,
                      transport_type: record.transport_type,
                      command: record.command,
                      args: record.args || [],
                      url: record.url,
                      env_vars_kv: Object.entries(record.env_vars || {}).map(([k, v]) => ({ key: k, value: v })),
                    });
                  }} type="link" size="small" style={{ padding: 0 }}
                >
                  编辑
                </Button>
                <Popconfirm title="确认删除该服务器？" onConfirm={() => handleDelete(record.id)}>
                  <Button danger type="link" size="small" style={{ padding: 0 }}>删除</Button>
                </Popconfirm>
              </ActionGroup>
            ),
              width: 100,
              fixed: 'left' as const
          },
    { title: '名称', dataIndex: 'name', key: 'name', ellipsis: true },
    { title: 'Code', dataIndex: 'code', key: 'code', width: 120, render: (v: string) => <Tag color="blue">{v}</Tag> },
    { title: '传输', dataIndex: 'transport_type', key: 'transport_type', width: 100 },
    { title: '状态', dataIndex: 'status', key: 'status', width: 100, render: (v: string) =>
      v === 'active' ? <Tag color="green">active</Tag> : v === 'error' ? <Tag color="red">error</Tag> : <Tag>{v || 'inactive'}</Tag> },
    { title: '工具数', key: 'tools_count', width: 80, render: (_: unknown, record: MCPServer) =>
      record.capabilities && record.capabilities.length > 0
        ? <Tag color="cyan">{record.capabilities.length}</Tag>
        : <Tag color="default">0</Tag>
    },
    { title: '最后连接', dataIndex: 'last_connected', key: 'last_connected', width: 160, render: (ts: number | null) => ts ? new Date(ts).toLocaleString() : '-' }
  ], [form, handleDelete, handleRefresh, handleTest]);

  useEffect(() => { fetchList(); }, [fetchList]);

  return (
    <div style={{ padding: 0 }}>
      <Card
        title={`MCP 服务器管理（${items.length}）`}
        extra={
          <Space>
            <Button onClick={() => fetchList()}>刷新</Button>
            <Button
              type="primary"
              onClick={() => {
                setEditing(null);
                setOpen(true);
                form.resetFields();
              }}
            >
              新建服务器
            </Button>
          </Space>
        }
      >
        <Table<MCPServer>
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={items}
          expandable={{
            expandedRowRender: (record) => (
              <div style={{ padding: '8px 0' }}>
                <span style={{ fontWeight: 'bold' }}>工具列表（{record.capabilities?.length || 0}）：</span>
                {record.capabilities && record.capabilities.length > 0 ? (
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, marginTop: 8 }}>
                    {record.capabilities.map((tool: MCPTool) => (
                      <Tag key={tool.name} color="blue" style={{ margin: 0 }}>
                        {tool.name}
                      </Tag>
                    ))}
                  </div>
                ) : (
                  <span style={{ color: '#999', marginTop: 8, display: 'block' }}>
                    暂无工具，请点击「刷新工具」获取
                  </span>
                )}
              </div>
            ),
            rowExpandable: () => true,
          }}
        />
      </Card>

      <Modal
        title={editing ? '编辑 MCP 服务器' : '新建 MCP 服务器'}
        open={open}
        onCancel={() => {
          setOpen(false);
          setEditing(null);
        }}
        footer={null}
        width={720}
      >
        <Form layout="vertical" form={form} onFinish={handleSubmit}>
          {!editing && (
            <Form.Item label="编码（全局唯一）" name="code" rules={[{ required: true, message: '请输入编码' }]}>
              <Input placeholder="如: local-stdio" />
            </Form.Item>
          )}
          <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="服务器显示名称" />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item label="传输类型" name="transport_type" rules={[{ required: true, message: '请选择传输类型' }]}>
            <Select
              options={[
                { value: 'stdio', label: 'stdio（子进程）' },
                { value: 'http', label: 'http（流式）' },
                { value: 'sse', label: 'sse（服务端推送）' },
              ]}
            />
          </Form.Item>
          <Form.Item shouldUpdate noStyle>
            {() => {
              const t = form.getFieldValue('transport_type');
              return (
                <>
                  {t === 'stdio' && (
                    <>
                      <Form.Item label="命令" name="command">
                        <Input placeholder="例如: node" />
                      </Form.Item>
                      <Form.Item label="参数（逗号分隔）" name="args">
                        <Select mode="tags" tokenSeparators={[',']} placeholder="例如: server.js,--port,8080" />
                      </Form.Item>
                      <Form.List name="env_vars_kv">
                        {(fields, { add, remove }) => (
                          <div>
                            <div style={{ marginBottom: 8 }}>
                              环境变量
                              <Button type="link" onClick={() => add()} style={{ paddingLeft: 8 }}>
                                + 添加
                              </Button>
                            </div>
                            {fields.map((field) => (
                              <Space key={field.key} style={{ display: 'flex', marginBottom: 8 }} align="baseline">
                                <Form.Item {...field} name={[field.name, 'key']} rules={[{ required: true, message: 'Key 必填' }]}>
                                  <Input placeholder="KEY" />
                                </Form.Item>
                                <Form.Item {...field} name={[field.name, 'value']}>
                                  <Input placeholder="VALUE" />
                                </Form.Item>
                                <Button onClick={() => remove(field.name)}>移除</Button>
                              </Space>
                            ))}
                          </div>
                        )}
                      </Form.List>
                    </>
                  )}
                  {(t === 'http' || t === 'sse') && (
                    <Form.Item label="服务器 URL" name="url" rules={[{ required: true, message: '请输入服务器地址' }]}>
                      <Input placeholder="例如: http://localhost:8080/mcp/sse" />
                    </Form.Item>
                  )}
                </>
              );
            }}
          </Form.Item>

          <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button
              onClick={() => {
                setOpen(false);
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
