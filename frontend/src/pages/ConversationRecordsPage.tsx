/**
 * 对话记录页面
 * 支持按条件查询对话记录，以对话形式展示会话记录，以及链路树可视化
 */
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  Button,
  Card,
  Form,
  Input,
  Select,
  Space,
  Table,
  Tag,
  message,
  DatePicker,
  Tooltip,
} from 'antd';
import {
  FilterOutlined,
  ClearOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import dayjs from 'dayjs';
import {
  listConversationRecords,
} from '../api/conversationRecordApi';
import type { ConversationRecord, ListConversationRecordsQuery } from '../types/conversationRecord';
import { TraceViewer } from '../components/TraceViewer';
import { ActionGroup } from "@/components/ActionGroup";

type QueryFormValues = {
  trace_id?: string;
  session_key?: string;
  agent_code?: string;
  channel_code?: string;
  event_type?: string;
  role?: string;
  dateRange?: [dayjs.Dayjs, dayjs.Dayjs];
};

/**
 * 将毫秒时间戳格式化为可读字符串
 */
function formatTime(ms: number): string {
  return new Date(ms).toLocaleString();
}

function toRoleTag(role: string): React.ReactNode {
  const r = (role || '').toLowerCase();
  if (r === 'user') return <Tag color="blue">user</Tag>;
  if (r === 'assistant') return <Tag color="green">assistant</Tag>;
  if (r === 'system') return <Tag color="purple">system</Tag>;
  if (r === 'tool') return <Tag color="orange">tool</Tag>;
  if (r === 'tool_result') return <Tag color="cyan">tool_result</Tag>;
  return <Tag>{role || '-'}</Tag>;
}

export const ConversationRecordsPage: React.FC = () => {
  const [searchParams] = useSearchParams();
  const [items, setItems] = useState<ConversationRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm<QueryFormValues>();
  const [pagination, setPagination] = useState<TablePaginationConfig>({
    current: 1,
    pageSize: 50,
    showSizeChanger: true,
    pageSizeOptions: [20, 50, 100, 200],
    showTotal: (totalCount) => `共 ${totalCount} 条`,
  });

  // 链路可视化状态
  const [traceVisible, setTraceVisible] = useState(false);
  const [currentTraceId, setCurrentTraceId] = useState('');

  // 筛选面板状态
  const [filterVisible, setFilterVisible] = useState(false);

  // Agent 和 Channel 选项
  const [agentOptions, setAgentOptions] = useState<{ code: string; name: string }[]>([]);
  const [channelOptions, setChannelOptions] = useState<{ code: string; name: string }[]>([]);

  const roleOptions = [
    { value: 'user', label: '用户' },
    { value: 'assistant', label: '助手' },
    { value: 'system', label: '系统' },
    { value: 'tool', label: '工具' },
    { value: 'tool_result', label: '工具结果' },
  ];

  const extractOptions = (items: ConversationRecord[]) => {
    const agentMap = new Map<string, string>();
    const channelMap = new Map<string, string>();

    items.forEach(item => {
      if (item.agent_code) {
        agentMap.set(item.agent_code, item.agent_code);
      }
      if (item.channel_code) {
        channelMap.set(item.channel_code, item.channel_code);
      }
    });

    setAgentOptions(Array.from(agentMap.entries()).map(([code, name]) => ({ code, name })));
    setChannelOptions(Array.from(channelMap.entries()).map(([code, name]) => ({ code, name })));
  };

  /**
   * 拉取对话记录列表
   */
  const fetchList = useCallback(async (queryOverrides?: Partial<ListConversationRecordsQuery>) => {
    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const limit = pagination.pageSize || 50;
      const current = pagination.current || 1;

      // 处理时间范围
      let start_time: string | undefined;
      let end_time: string | undefined;
      if (values.dateRange && values.dateRange.length === 2) {
        start_time = values.dateRange[0].toISOString();
        end_time = values.dateRange[1].toISOString();
      }

      const query: ListConversationRecordsQuery = {
        trace_id: values.trace_id || undefined,
        session_key: values.session_key || undefined,
        agent_code: values.agent_code || undefined,
        channel_code: values.channel_code || undefined,
        event_type: values.event_type || undefined,
        role: values.role || undefined,
        start_time,
        end_time,
        limit,
        offset: (current - 1) * limit,
        ...queryOverrides,
      };
      const data = await listConversationRecords(query);
      setItems(data.items);
      setTotal(data.total);
      extractOptions(data.items);
    } catch (_error) {
      message.error('获取对话记录失败');
    } finally {
      setLoading(false);
    }
  }, [form, pagination]);

  // 构建会话聊天消息
  const columns: ColumnsType<ConversationRecord> = useMemo(
    () => [
          {
                  title: '操作',
                  key: 'action',
                  render: (_: unknown, record: ConversationRecord) => (
                    <ActionGroup size="small">
                      <Tooltip title="查看链路">
                        <Button
                          onClick={() => {
                            if (record.trace_id) {
                              setCurrentTraceId(record.trace_id);
                              setTraceVisible(true);
                            } else {
                              message.warning('该记录没有 trace_id');
                            }
                          }} type="link" size="small" style={{ padding: 0 }}
                        >查看</Button>
                      </Tooltip>
                    </ActionGroup>
                  ),
                    width: 100,
                    fixed: 'left' as const
              },
        {
        title: 'ID',
        dataIndex: 'id',
        key: 'id',
        width: 80,
        ellipsis: true,
      },
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
        title: 'Agent',
        dataIndex: 'agent_code',
        key: 'agent_code',
        width: 120,
        ellipsis: true,
        render: (v: string) => (v ? <Tag>{v}</Tag> : '-'),
      },
      {
        title: 'Channel',
        dataIndex: 'channel_code',
        key: 'channel_code',
        width: 120,
        ellipsis: true,
        render: (v: string) => (v ? <Tag>{v}</Tag> : '-'),
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
        width: 180,
        ellipsis: true,
        render: (v: string) => (v ? <Tag color="geekblue">{v.slice(0, 12)}...</Tag> : <Tag>-</Tag>),
      },
      {
        title: 'Session',
        dataIndex: 'session_key',
        key: 'session_key',
        width: 180,
        ellipsis: true,
        render: (v: string) => (v ? <Tag color="blue">{v.slice(0, 16)}...</Tag> : <Tag>-</Tag>),
      },
      {
        title: 'Tokens',
        dataIndex: 'total_tokens',
        key: 'total_tokens',
        width: 80,
        render: (v: number) => v || 0,
      },
      {
        title: '内容',
        dataIndex: 'content',
        key: 'content',
        ellipsis: true,
        render: (v: string) => v?.substring(0, 50) + (v?.length > 50 ? '...' : ''),
      }
    ],
    [setCurrentTraceId],
  );

  // 从 URL 参数读取 trace_id 并自动应用
  useEffect(() => {
    const traceIdFromUrl = searchParams.get('trace_id');
    if (traceIdFromUrl) {
      form.setFieldsValue({ trace_id: traceIdFromUrl });
      setFilterVisible(true);
    }
    fetchList();
  }, [fetchList, searchParams, form]);

  return (
    <div style={{ padding: 0 }}>
      <Card
        title={
          <Space>
            <span>对话记录</span>
            <Button
              type={filterVisible ? 'primary' : 'default'}
              icon={<FilterOutlined />}
              size="small"
              onClick={() => setFilterVisible(!filterVisible)}
            >
              筛选
            </Button>
          </Space>
        }
        extra={
          <Space>
            <Button onClick={() => fetchList()}>刷新</Button>
          </Space>
        }
      >
        {/* 筛选面板 */}
        {filterVisible && (
          <Card size="small" style={{ marginBottom: 16, background: '#f5f5f5' }}>
            <Form<QueryFormValues>
              form={form}
              layout="vertical"
              onFinish={() => {
                setPagination((p) => ({ ...p, current: 1 }));
                fetchList({ offset: 0 });
              }}
            >
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12 }}>
                <div style={{ minWidth: '200px', flex: '1 1 200px' }}>
                  <Form.Item label="时间范围" name="dateRange" style={{ marginBottom: 8 }}>
                    <DatePicker.RangePicker showTime style={{ width: '100%' }} />
                  </Form.Item>
                </div>
                <div style={{ minWidth: '150px', flex: '1 1 150px' }}>
                  <Form.Item label="Trace" name="trace_id" style={{ marginBottom: 8 }}>
                    <Input placeholder="trace_id" />
                  </Form.Item>
                </div>
                <div style={{ minWidth: '150px', flex: '1 1 150px' }}>
                  <Form.Item label="Session" name="session_key" style={{ marginBottom: 8 }}>
                    <Input placeholder="session_key" />
                  </Form.Item>
                </div>
                <div style={{ minWidth: '130px', flex: '1 1 130px' }}>
                  <Form.Item label="Agent" name="agent_code" style={{ marginBottom: 8 }}>
                    <Select
                      allowClear
                      options={agentOptions.map(a => ({ label: a.name || a.code, value: a.code }))}
                      placeholder="选择Agent"
                    />
                  </Form.Item>
                </div>
                <div style={{ minWidth: '130px', flex: '1 1 130px' }}>
                  <Form.Item label="Channel" name="channel_code" style={{ marginBottom: 8 }}>
                    <Select
                      allowClear
                      options={channelOptions.map(c => ({ label: c.name || c.code, value: c.code }))}
                      placeholder="选择Channel"
                    />
                  </Form.Item>
                </div>
                <div style={{ minWidth: '130px', flex: '1 1 130px' }}>
                  <Form.Item label="Role" name="role" style={{ marginBottom: 8 }}>
                    <Select
                      allowClear
                      options={roleOptions}
                      placeholder="选择角色"
                    />
                  </Form.Item>
                </div>
                <div style={{ minWidth: '130px', flex: '1 1 130px' }}>
                  <Form.Item label="Event" name="event_type" style={{ marginBottom: 8 }}>
                    <Input placeholder="event_type" />
                  </Form.Item>
                </div>
                <div style={{ display: 'flex', gap: 8, alignItems: 'flex-end' }}>
                  <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>
                    查询
                  </Button>
                  <Button
                    icon={<ClearOutlined />}
                    onClick={() => {
                      form.resetFields();
                      setPagination((p) => ({ ...p, current: 1 }));
                      fetchList({ offset: 0 });
                    }}
                  >
                    重置
                  </Button>
                </div>
              </div>
            </Form>
          </Card>
        )}

        <Table<ConversationRecord>
          rowKey="id"
          loading={loading}
          dataSource={items}
          columns={columns}
          pagination={{
            ...pagination,
            total,
            onChange: (page, pageSize) => {
              setPagination((p) => ({ ...p, current: page, pageSize }));
              fetchList({ limit: pageSize, offset: (page - 1) * pageSize });
            },
          }}
          scroll={{ x: 1500 }}
        />
      </Card>

      {/* 链路可视化弹窗 */}
      <TraceViewer
        traceId={currentTraceId}
        visible={traceVisible}
        onClose={() => {
          setTraceVisible(false);
          setCurrentTraceId('');
        }}
      />
    </div>
  );
};