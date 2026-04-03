/**
 * TraceViewer - 对话链路查看组件
 * 可复用的对话链路显示组件，传入 traceId 即可显示该链路的所有对话记录
 */
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  Card,
  Col,
  Divider,
  Modal,
  Row,
  Space,
  Statistic,
  Tag,
  Tree,
  Typography,
  message,
} from 'antd';
import {
  MessageOutlined,
  BranchesOutlined,
} from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import dayjs from 'dayjs';
import { getConversationRecordsByTrace } from '../../api/conversationRecordApi';
import type { ConversationRecord } from '../../types/conversationRecord';

const { Text } = Typography;

interface TraceViewerProps {
  traceId: string;
  visible: boolean;
  onClose: () => void;
  title?: string;
}

interface TraceNode {
  key: string;
  title: React.ReactNode;
  children?: TraceNode[];
  record: ConversationRecord;
  duration?: number;
}

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


function getTraceStats(records: ConversationRecord[]) {
  if (records.length === 0) {
    return { count: 0, totalTokens: 0, duration: 0 };
  }
  const sorted = [...records].sort((a, b) => (a.timestamp || 0) - (b.timestamp || 0));
  const totalTokens = records.reduce((sum, r) => sum + (r.total_tokens || 0), 0);
  const duration = Math.round(
    ((sorted[sorted.length - 1]?.timestamp || 0) - (sorted[0]?.timestamp || 0)) / 1000
  );
  return { count: records.length, totalTokens, duration };
}

function buildTraceTree(records: ConversationRecord[]): TraceNode[] {
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
    const eventDiff =
      (eventPriority[a.event_type || ''] || 1000) -
      (eventPriority[b.event_type || ''] || 1000);
    if (eventDiff !== 0) return eventDiff;
    const roleDiff =
      (rolePriority[a.role || ''] || 1000) - (rolePriority[b.role || ''] || 1000);
    if (roleDiff !== 0) return roleDiff;
    return a.id.localeCompare(b.id);
  };

  const sorted = [...records].sort(compareByOrder);

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
          {duration > 0 && (
            <Text type="success" style={{ fontSize: 12 }}>
              +{(duration / 1000).toFixed(1)}s
            </Text>
          )}
        </Space>
        <Text ellipsis style={{ maxWidth: 400, fontSize: 12 }}>
          {record.content?.substring(0, 100)}
          {record.content?.length > 100 ? '...' : ''}
        </Text>
      </Space>
    );

    nodeMap.set(record.span_id, {
      key: record.span_id,
      title,
      record,
      duration,
      children: [],
    });
  });

  const roots: TraceNode[] = [];
  sorted.forEach((record) => {
    const node = nodeMap.get(record.span_id);
    if (!node) return;
    if (record.parent_span_id && record.parent_span_id !== '' && nodeMap.has(record.parent_span_id)) {
      const parent = nodeMap.get(record.parent_span_id)!;
      parent.children = parent.children || [];
      parent.children.push(node);
    } else {
      roots.push(node);
    }
  });

  return roots;
}

export const TraceViewer: React.FC<TraceViewerProps> = ({
  traceId,
  visible,
  onClose,
  title,
}) => {
  const [records, setRecords] = useState<ConversationRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [chatVisible, setChatVisible] = useState(false);
  const fetchIdRef = useRef(0);

  const fetchRecords = useCallback(async () => {
    if (!traceId) return;
    const fetchId = ++fetchIdRef.current;
    setRecords([]);
    setLoading(true);
    try {
      const data = await getConversationRecordsByTrace(traceId);
      if (fetchId === fetchIdRef.current) {
        setRecords(data);
      }
    } catch {
      if (fetchId === fetchIdRef.current) {
        message.error('获取对话链路数据失败');
      }
    } finally {
      if (fetchId === fetchIdRef.current) {
        setLoading(false);
      }
    }
  }, [traceId]);

  useEffect(() => {
    if (visible && traceId) {
      fetchRecords();
    }
    if (!visible) {
      fetchIdRef.current++;
      setRecords([]);
    }
  }, [visible, traceId, fetchRecords]);

  const traceStats = useMemo(() => getTraceStats(records), [records]);
  const traceTreeData = useMemo(() => buildTraceTree(records), [records]);

  // 递归转换树节点，支持任意层级
  const convertToTreeData = (nodes: TraceNode[]): DataNode[] =>
    nodes.map((node) => ({
      key: node.key,
      title: node.title,
      children: node.children ? convertToTreeData(node.children) : undefined,
    }));

  const treeData: DataNode[] = useMemo(
    () => convertToTreeData(traceTreeData),
    [traceTreeData]
  );

  const displayTitle = title || `对话链路 - ${traceId?.slice(0, 12)}...`;

  return (
    <>
      <Modal
        title={
          <Space>
            <BranchesOutlined />
            {displayTitle}
          </Space>
        }
        open={visible}
        onCancel={onClose}
        footer={
          <Space style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <Button
              icon={<MessageOutlined />}
              onClick={() => setChatVisible(true)}
              disabled={records.length === 0}
            >
              查看对话
            </Button>
            <Button onClick={onClose}>关闭</Button>
          </Space>
        }
        width={900}
      >
        {loading ? (
          <div style={{ textAlign: 'center', padding: 40 }}>加载中...</div>
        ) : records.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 40 }}>无数据</div>
        ) : (
          <div>
            <Card size="small" style={{ marginBottom: 16 }}>
              <Row gutter={16}>
                <Col span={8}>
                  <Statistic title="总消息数" value={traceStats.count} />
                </Col>
                <Col span={8}>
                  <Statistic title="总Token" value={traceStats.totalTokens} />
                </Col>
                <Col span={8}>
                  <Statistic
                    title="总耗时"
                    value={`${traceStats.duration}s`}
                  />
                </Col>
              </Row>
            </Card>
            <Divider style={{ margin: '12px 0' }} />
            <Tree
              treeData={treeData}
              showLine
              defaultExpandAll
              style={{
                background: '#fafafa',
                padding: 16,
                borderRadius: 8,
                maxHeight: 500,
                overflowY: 'auto',
              }}
            />
          </div>
        )}
      </Modal>

      {/* 对话详情弹窗 */}
      <Modal
        title={`对话详情 - ${traceId?.slice(0, 12) || ''}...`}
        open={chatVisible}
        onCancel={() => setChatVisible(false)}
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <Button onClick={() => setChatVisible(false)}>关闭</Button>
          </div>
        }
        width={800}
      >
        {records.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 40 }}>无数据</div>
        ) : (
          <div>
            <Row gutter={16} style={{ marginBottom: 16 }}>
              <Col span={8}>
                <Statistic title="消息数" value={records.length} />
              </Col>
              <Col span={8}>
                <Statistic
                  title="总Token"
                  value={records.reduce(
                    (sum, r) => sum + (r.total_tokens || 0),
                    0
                  )}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title="时长"
                  value={`${Math.round(
                    ((records[records.length - 1]?.timestamp || 0) -
                      (records[0]?.timestamp || 0)) /
                      1000
                  )}s`}
                />
              </Col>
            </Row>
            <Divider />
            <div
              style={{
                maxHeight: 500,
                overflowY: 'auto',
                padding: 16,
                background: '#f5f5f5',
                borderRadius: 8,
              }}
            >
              {records
                .sort((a, b) => (a.timestamp || 0) - (b.timestamp || 0))
                .map((r) => (
                  <div
                    key={r.id}
                    style={{
                      display: 'flex',
                      flexDirection: r.role === 'user' ? 'row-reverse' : 'row',
                      marginBottom: 16,
                      alignItems: 'flex-start',
                    }}
                  >
                    <div
                      style={{
                        maxWidth: '70%',
                        padding: '12px 16px',
                        borderRadius:
                          r.role === 'user'
                            ? '16px 16px 4px 16px'
                            : '16px 16px 16px 4px',
                        background: r.role === 'user' ? '#1890ff' : '#fff',
                        color: r.role === 'user' ? '#fff' : '#333',
                        boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
                      }}
                    >
                      <div
                        style={{
                          fontSize: 12,
                          opacity: 0.7,
                          marginBottom: 4,
                        }}
                      >
                        {getRoleLabel(r.role)} · {r.total_tokens || 0} tokens
                      </div>
                      <div
                        style={{
                          whiteSpace: 'pre-wrap',
                          wordBreak: 'break-word',
                        }}
                      >
                        {r.content || ''}
                      </div>
                      <div
                        style={{
                          fontSize: 11,
                          opacity: 0.5,
                          marginTop: 4,
                          textAlign: 'right',
                        }}
                      >
                        {dayjs(r.timestamp).format('HH:mm:ss')}
                      </div>
                    </div>
                  </div>
                ))}
            </div>
          </div>
        )}
      </Modal>
    </>
  );
};

export default TraceViewer;
