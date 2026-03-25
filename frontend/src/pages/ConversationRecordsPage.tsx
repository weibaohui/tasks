/**
 * 对话记录页面
 * 支持按条件查询对话记录，以对话形式展示会话记录，以及链路树可视化
 */
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
  Modal,
  Tree,
  Row,
  Col,
  Statistic,
  Divider,
  DatePicker,
  Tooltip,
} from 'antd';
import {
  EyeOutlined,
  MessageOutlined,
  FilterOutlined,
  ClearOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import type { DataNode } from 'antd/es/tree';
import dayjs from 'dayjs';
import {
  listConversationRecords,
  getConversationRecordsByTrace,
} from '../api/conversationRecordApi';
import type { ConversationRecord, ListConversationRecordsQuery } from '../types/conversationRecord';

const { Text } = Typography;

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

/**
 * 将角色映射为更易读的标签展示
 */
function getRoleColor(role?: string): string {
  const colors: Record<string, string> = {
    user: 'blue',
    assistant: 'green',
    system: 'orange',
    tool: 'purple',
    tool_result: 'cyan',
  };
  return colors[role || ''] || 'default';
}

function getRoleLabel(role?: string): string {
  const labels: Record<string, string> = {
    user: '用户',
    assistant: '助手',
    system: '系统',
    tool: '工具',
    tool_result: '工具结果',
  };
  return labels[role || ''] || role || '';
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

// 链路树节点类型
interface TraceNode {
  key: string;
  title: React.ReactNode;
  children?: TraceNode[];
  record: ConversationRecord;
  duration?: number;
}

export const ConversationRecordsPage: React.FC = () => {
  const [items, setItems] = useState<ConversationRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm<QueryFormValues>();
  const [pagination, setPagination] = useState<TablePaginationConfig>({
    current: 1,
    pageSize: 50,
    showSizeChanger: true,
    pageSizeOptions: [20, 50, 100, 200],
  });

  // 链路可视化状态
  const [traceVisible, setTraceVisible] = useState(false);
  const [traceRecords, setTraceRecords] = useState<ConversationRecord[]>([]);
  const [traceLoading, setTraceLoading] = useState(false);
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

      const query: ListConversationRecordsQuery = {
        trace_id: values.trace_id || undefined,
        session_key: values.session_key || undefined,
        agent_code: values.agent_code || undefined,
        channel_code: values.channel_code || undefined,
        event_type: values.event_type || undefined,
        role: values.role || undefined,
        limit,
        offset: (current - 1) * limit,
        ...queryOverrides,
      };
      const data = await listConversationRecords(query);
      setItems(data);
      extractOptions(data);
    } catch (_error) {
      message.error('获取对话记录失败');
    } finally {
      setLoading(false);
    }
  }, [form, pagination]);

  /**
   * 获取链路数据
   */
  const fetchTraceRecords = useCallback(async (traceId: string) => {
    setTraceLoading(true);
    try {
      const data = await getConversationRecordsByTrace(traceId);
      setTraceRecords(data);
    } catch (_error) {
      message.error('获取链路数据失败');
    } finally {
      setTraceLoading(false);
    }
  }, []);

  // 构建链路树
  const buildTraceTree = (records: ConversationRecord[]): TraceNode[] => {
    const nodeMap = new Map<string, TraceNode>();

    const eventPriority: Record<string, number> = {
      llm_call_end: 10,
      tool_completed: 20,
    };
    const rolePriority: Record<string, number> = {
      tool: 10,
      tool_result: 20,
    };
    const compareByOrder = (a: ConversationRecord, b: ConversationRecord) => {
      const timeDiff = (a.timestamp || 0) - (b.timestamp || 0);
      if (timeDiff !== 0) return timeDiff;
      const eventDiff = (eventPriority[a.event_type || ''] || 1000) - (eventPriority[b.event_type || ''] || 1000);
      if (eventDiff !== 0) return eventDiff;
      const roleDiff = (rolePriority[a.role || ''] || 1000) - (rolePriority[b.role || ''] || 1000);
      if (roleDiff !== 0) return roleDiff;
      return a.id.localeCompare(b.id);
    };

    // 按时间排序
    const sorted = [...records].sort(compareByOrder);
    const indexById = new Map<string, number>();
    sorted.forEach((record, index) => {
      indexById.set(record.id, index);
    });

    // 创建所有节点
    sorted.forEach((record, index) => {
      const nextRecord = sorted[index + 1];
      const duration = nextRecord
        ? (nextRecord.timestamp || 0) - (record.timestamp || 0)
        : 0;

      const title = (
        <Space direction="vertical" size={0} style={{ width: '100%' }}>
          <Space>
            <Tag color={getRoleColor(record.role)}>{getRoleLabel(record.role)}</Tag>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {record.event_type}
            </Text>
            {record.total_tokens > 0 && (
              <Tag color="blue">{record.total_tokens} tokens</Tag>
            )}
            {duration > 0 && duration < 300000 && (
              <Text type="success" style={{ fontSize: 12 }}>
                +{duration}ms
              </Text>
            )}
          </Space>
          <Text ellipsis style={{ maxWidth: 400, fontSize: 12 }}>
            {record.content?.substring(0, 100)}
            {record.content?.length > 100 ? '...' : ''}
          </Text>
        </Space>
      );

      nodeMap.set(record.id, {
        key: record.id,
        title,
        record,
        duration,
        children: [],
      });
    });

    const roots = sorted
      .map(record => nodeMap.get(record.id))
      .filter((node): node is TraceNode => !!node);

    const detachNode = (targetId: string) => {
      const rootIndex = roots.findIndex(node => node.record.id === targetId);
      if (rootIndex >= 0) {
        roots.splice(rootIndex, 1);
      }
      nodeMap.forEach(node => {
        if (!node.children || node.children.length === 0) return;
        node.children = node.children.filter(child => child.record.id !== targetId);
      });
    };

    sorted.forEach((record, index) => {
      if (record.role !== 'tool_result') return;
      const resultNode = nodeMap.get(record.id);
      if (!resultNode) return;

      let targetToolRecord: ConversationRecord | undefined;
      for (let i = index - 1; i >= 0; i -= 1) {
        const candidate = sorted[i];
        if (candidate.role !== 'tool') continue;
        if (record.parent_span_id && candidate.span_id === record.parent_span_id) {
          targetToolRecord = candidate;
          break;
        }
        if (record.span_id && candidate.span_id === record.span_id) {
          targetToolRecord = candidate;
          break;
        }
      }

      if (!targetToolRecord) {
        for (let i = index - 1; i >= 0; i -= 1) {
          if (sorted[i].role === 'tool') {
            targetToolRecord = sorted[i];
            break;
          }
        }
      }

      if (!targetToolRecord) return;
      const toolNode = nodeMap.get(targetToolRecord.id);
      if (!toolNode) return;
      const toolIndex = indexById.get(toolNode.record.id);
      const resultIndex = indexById.get(resultNode.record.id);
      if (toolIndex === undefined || resultIndex === undefined || toolIndex >= resultIndex) return;

      detachNode(resultNode.record.id);
      toolNode.children = toolNode.children || [];
      if (!toolNode.children.some(child => child.record.id === resultNode.record.id)) {
        toolNode.children.push(resultNode);
      }
    });

    const sortTreeNodes = (nodes: TraceNode[]) => {
      nodes.sort((a, b) => compareByOrder(a.record, b.record));
      nodes.forEach(node => {
        if (node.children && node.children.length > 0) {
          sortTreeNodes(node.children);
        }
      });
    };

    sortTreeNodes(roots);
    return roots;
  };

  // 计算链路统计
  const getTraceStats = (records: ConversationRecord[]) => {
    const totalTokens = records.reduce((sum, r) => sum + (r.total_tokens || 0), 0);
    const startTime = records.length > 0 ? records[0].timestamp : null;
    const endTime = records.length > 0 ? records[records.length - 1].timestamp : null;
    const duration = startTime && endTime ? (endTime - startTime) : 0;

    return { totalTokens, duration, count: records.length };
  };

  // 构建会话聊天消息
  const columns: ColumnsType<ConversationRecord> = useMemo(
    () => [
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
      },
      {
        title: '操作',
        key: 'action',
        width: 150,
        fixed: 'right' as const,
        render: (_: unknown, record: ConversationRecord) => (
          <Space size="small">
            <Tooltip title="查看详情">
              <Button
                type="text"
                icon={<EyeOutlined />}
                onClick={() => {
                  if (record.trace_id) {
                    setCurrentTraceId(record.trace_id);
                    fetchTraceRecords(record.trace_id);
                    setTraceVisible(true);
                  } else {
                    message.warning('该记录没有 trace_id');
                  }
                }}
              />
            </Tooltip>
            <Tooltip title="查看对话">
              <Button
                type="text"
                icon={<MessageOutlined />}
                onClick={() => {
                  if (record.trace_id) {
                    setCurrentTraceId(record.trace_id);
                    fetchTraceRecords(record.trace_id);
                    setTraceVisible(true);
                  } else {
                    message.warning('该记录没有 trace_id');
                  }
                }}
              />
            </Tooltip>
          </Space>
        ),
      },
    ],
    [fetchTraceRecords],
  );

  const traceStats = getTraceStats(traceRecords);
  const traceTreeData = buildTraceTree(traceRecords);

  // Convert TraceNode[] to DataNode[] for Ant Design Tree
  const treeData: DataNode[] = traceTreeData.map(node => ({
    key: node.key,
    title: node.title,
    children: node.children?.map(child => ({
      key: child.key,
      title: child.title,
      children: child.children?.map(grandChild => ({
        key: grandChild.key,
        title: grandChild.title,
      })),
    })),
  }));

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  return (
    <div style={{ padding: 24 }}>
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
              layout="inline"
              onFinish={() => {
                setPagination((p) => ({ ...p, current: 1 }));
                fetchList({ offset: 0 });
              }}
            >
              <Form.Item label="时间范围" name="dateRange">
                <DatePicker.RangePicker showTime />
              </Form.Item>
              <Form.Item label="Trace" name="trace_id">
                <Input placeholder="trace_id" style={{ width: 180 }} />
              </Form.Item>
              <Form.Item label="Session" name="session_key">
                <Input placeholder="session_key" style={{ width: 180 }} />
              </Form.Item>
              <Form.Item label="Agent" name="agent_code">
                <Select
                  allowClear
                  style={{ width: 140 }}
                  options={agentOptions.map(a => ({ label: a.name || a.code, value: a.code }))}
                  placeholder="选择Agent"
                />
              </Form.Item>
              <Form.Item label="Channel" name="channel_code">
                <Select
                  allowClear
                  style={{ width: 140 }}
                  options={channelOptions.map(c => ({ label: c.name || c.code, value: c.code }))}
                  placeholder="选择Channel"
                />
              </Form.Item>
              <Form.Item label="Role" name="role">
                <Select
                  allowClear
                  style={{ width: 140 }}
                  options={roleOptions}
                  placeholder="选择角色"
                />
              </Form.Item>
              <Form.Item label="Event" name="event_type">
                <Input placeholder="event_type" style={{ width: 140 }} />
              </Form.Item>
              <Form.Item>
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
                  style={{ marginLeft: 8 }}
                >
                  重置
                </Button>
              </Form.Item>
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
            onChange: (page, pageSize) => {
              setPagination((p) => ({ ...p, current: page, pageSize }));
              fetchList({ limit: pageSize, offset: (page - 1) * pageSize });
            },
          }}
          scroll={{ x: 1500 }}
        />
      </Card>

      {/* 链路可视化弹窗 */}
      <Modal
        title={`对话链路 - ${currentTraceId.slice(0, 12)}...`}
        open={traceVisible}
        onCancel={() => {
          setTraceVisible(false);
          setTraceRecords([]);
        }}
        footer={null}
        width={900}
      >
        {traceLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}>加载中...</div>
        ) : traceRecords.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 40 }}>无数据</div>
        ) : (
          <div>
            <Row gutter={16} style={{ marginBottom: 16 }}>
              <Col span={8}>
                <Statistic title="总消息数" value={traceStats.count} />
              </Col>
              <Col span={8}>
                <Statistic title="总Token" value={traceStats.totalTokens} />
              </Col>
              <Col span={8}>
                <Statistic title="总耗时" value={`${traceStats.duration}ms`} />
              </Col>
            </Row>
            <Divider />
            <Tree
              treeData={treeData}
              showLine
              defaultExpandAll
              style={{ background: '#fafafa', padding: 16, borderRadius: 8 }}
            />
          </div>
        )}
      </Modal>
    </div>
  );
};