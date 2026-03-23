/**
 * 对话记录页面
 * 支持按条件查询对话记录，并以对话形式展示指定会话的记录
 */
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Card, Drawer, Form, Input, Select, Space, Table, Tag, Typography, message } from 'antd';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import { listConversationRecords } from '../api/conversationRecordApi';
import { useAuthStore } from '../stores/authStore';
import type { ConversationRecord, ListConversationRecordsQuery } from '../types/conversationRecord';

type QueryFormValues = {
  trace_id?: string;
  session_key?: string;
  agent_code?: string;
  channel_code?: string;
  event_type?: string;
  role?: string;
};

/**
 * 将毫秒时间戳格式化为可读字符串
 */
function formatTime(ms: number): string {
  return new Date(ms).toLocaleString();
}

/**
 * 将角色映射为更易读的标签展示
 */
function toRoleTag(role: string): React.ReactNode {
  const r = (role || '').toLowerCase();
  if (r === 'user') return <Tag color="blue">user</Tag>;
  if (r === 'assistant') return <Tag color="green">assistant</Tag>;
  if (r === 'system') return <Tag color="purple">system</Tag>;
  if (r === 'tool') return <Tag color="orange">tool</Tag>;
  return <Tag>{role || '-'}</Tag>;
}

export const ConversationRecordsPage: React.FC = () => {
  const { user } = useAuthStore();
  const userCode = user?.user_code || '';
  const [items, setItems] = useState<ConversationRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [drawerSessionKey, setDrawerSessionKey] = useState<string | null>(null);
  const [drawerRecords, setDrawerRecords] = useState<ConversationRecord[]>([]);
  const [form] = Form.useForm<QueryFormValues>();
  const [pagination, setPagination] = useState<TablePaginationConfig>({
    current: 1,
    pageSize: 50,
    showSizeChanger: true,
    pageSizeOptions: [20, 50, 100, 200],
  });

  /**
   * 拉取对话记录列表
   */
  const fetchList = useCallback(async (queryOverrides?: Partial<ListConversationRecordsQuery>) => {
    if (!userCode) {
      setItems([]);
      return;
    }
    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const limit = pagination.pageSize || 50;
      const current = pagination.current || 1;
      const offset = (current - 1) * limit;
      const query: ListConversationRecordsQuery = {
        user_code: userCode,
        trace_id: values.trace_id || undefined,
        session_key: values.session_key || undefined,
        agent_code: values.agent_code || undefined,
        channel_code: values.channel_code || undefined,
        event_type: values.event_type || undefined,
        role: values.role || undefined,
        limit,
        offset,
        ...queryOverrides,
      };
      const data = await listConversationRecords(query);
      setItems(data);
    } catch (_error) {
      message.error('获取对话记录失败');
    } finally {
      setLoading(false);
    }
  }, [form, pagination, userCode]);

  /**
   * 打开会话抽屉（按 session_key 拉取更多记录并按时间排序）
   */
  const openSessionDrawer = useCallback(async (sessionKey: string) => {
    if (!userCode) {
      return;
    }
    setDrawerSessionKey(sessionKey);
    try {
      const data = await listConversationRecords({ user_code: userCode, session_key: sessionKey, limit: 200, offset: 0 });
      const sorted = [...data].sort((a, b) => (a.timestamp || 0) - (b.timestamp || 0));
      setDrawerRecords(sorted);
    } catch (_error) {
      setDrawerRecords([]);
      message.error('获取会话对话失败');
    }
  }, [userCode]);

  const columns: ColumnsType<ConversationRecord> = useMemo(
    () => [
      {
        title: '时间',
        dataIndex: 'timestamp',
        key: 'timestamp',
        width: 170,
        render: (v: number) => (v ? formatTime(v) : '-'),
      },
      {
        title: '角色',
        dataIndex: 'role',
        key: 'role',
        width: 110,
        render: (v: string) => toRoleTag(v),
      },
      {
        title: '事件',
        dataIndex: 'event_type',
        key: 'event_type',
        width: 140,
        render: (v: string) => (v ? <Tag>{v}</Tag> : <Tag>-</Tag>),
      },
      {
        title: 'Trace',
        dataIndex: 'trace_id',
        key: 'trace_id',
        width: 210,
        ellipsis: true,
        render: (v: string) => (v ? <Tag color="geekblue">{v}</Tag> : <Tag>-</Tag>),
      },
      {
        title: 'Session',
        dataIndex: 'session_key',
        key: 'session_key',
        width: 220,
        ellipsis: true,
        render: (v: string) => (v ? <Tag color="blue">{v}</Tag> : <Tag>-</Tag>),
      },
      {
        title: '内容',
        dataIndex: 'content',
        key: 'content',
        ellipsis: true,
      },
      {
        title: '操作',
        key: 'action',
        width: 120,
        render: (_: unknown, record: ConversationRecord) => (
          <Button
            onClick={() => {
              if (record.session_key) {
                openSessionDrawer(record.session_key);
              } else {
                message.warning('该记录没有 session_key，无法按会话展示');
              }
            }}
          >
            查看对话
          </Button>
        ),
      },
    ],
    [openSessionDrawer],
  );

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  return (
    <div style={{ padding: 24 }}>
      <Card
        title="对话记录"
        extra={
          <Space>
            <Button onClick={() => fetchList()}>刷新</Button>
          </Space>
        }
      >
        <Form<QueryFormValues>
          form={form}
          layout="inline"
          onFinish={() => {
            setPagination((p) => ({ ...p, current: 1 }));
            fetchList({ offset: 0 });
          }}
        >
          <Form.Item label="Trace" name="trace_id">
            <Input placeholder="trace_id" style={{ width: 220 }} />
          </Form.Item>
          <Form.Item label="Session" name="session_key">
            <Input placeholder="session_key" style={{ width: 220 }} />
          </Form.Item>
          <Form.Item label="Agent" name="agent_code">
            <Input placeholder="agent_code" style={{ width: 160 }} />
          </Form.Item>
          <Form.Item label="Channel" name="channel_code">
            <Input placeholder="channel_code" style={{ width: 160 }} />
          </Form.Item>
          <Form.Item label="Role" name="role">
            <Select
              allowClear
              style={{ width: 140 }}
              options={[
                { label: 'user', value: 'user' },
                { label: 'assistant', value: 'assistant' },
                { label: 'system', value: 'system' },
                { label: 'tool', value: 'tool' },
              ]}
            />
          </Form.Item>
          <Form.Item label="Event" name="event_type">
            <Input placeholder="event_type" style={{ width: 160 }} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">
              查询
            </Button>
          </Form.Item>
        </Form>

        <div style={{ marginTop: 16 }}>
          <Table<ConversationRecord>
            rowKey="id"
            loading={loading}
            dataSource={items}
            columns={columns}
            pagination={{
              ...pagination,
              onChange: (page, pageSize) => {
                setPagination((p) => ({ ...p, current: page, pageSize }));
                fetchList({ limit: pageSize, offset: (page - 1) * pageSize });
              },
            }}
          />
        </div>
      </Card>

      <Drawer
        title="会话对话展示"
        open={!!drawerSessionKey}
        onClose={() => {
          setDrawerSessionKey(null);
          setDrawerRecords([]);
        }}
        width={860}
      >
        <Space style={{ marginBottom: 12 }}>
          <Tag color="blue">{drawerSessionKey || '-'}</Tag>
          <Tag>共 {drawerRecords.length} 条</Tag>
        </Space>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {drawerRecords.map((r) => (
            <Card
              key={r.id}
              size="small"
              title={
                <Space>
                  {toRoleTag(r.role)}
                  <Tag>{r.event_type || '-'}</Tag>
                  <Typography.Text type="secondary">{formatTime(r.timestamp)}</Typography.Text>
                </Space>
              }
            >
              <pre style={{ margin: 0, whiteSpace: 'pre-wrap' }}>{r.content || ''}</pre>
            </Card>
          ))}
          {drawerRecords.length === 0 ? <Typography.Text type="secondary">暂无对话记录</Typography.Text> : null}
        </div>
      </Drawer>
    </div>
  );
};
