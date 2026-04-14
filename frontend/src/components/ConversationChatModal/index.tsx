/**
 * 对话记录聊天弹窗组件
 * 根据 trace_id 加载对话记录，以聊天形式展示
 */
import React, { useCallback, useEffect, useState } from 'react';
import { Modal, Button, Row, Col, Statistic, Divider, Spin, message } from 'antd';
import { XMarkdown } from '@ant-design/x-markdown';
import { getConversationRecordsByTrace } from '../../api/conversationRecordApi';
import type { ConversationRecord } from '../../types/conversationRecord';
import dayjs from 'dayjs';

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

interface ConversationChatModalProps {
  traceId: string | null;
  open: boolean;
  onClose: () => void;
}

export const ConversationChatModal: React.FC<ConversationChatModalProps> = ({ traceId, open, onClose }) => {
  const [records, setRecords] = useState<ConversationRecord[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchRecords = useCallback(async (tid: string) => {
    setLoading(true);
    try {
      const data = await getConversationRecordsByTrace(tid);
      setRecords(data.sort((a, b) => (a.timestamp || 0) - (b.timestamp || 0)));
    } catch {
      message.error('获取对话记录失败');
      setRecords([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (open && traceId) {
      fetchRecords(traceId);
    }
    if (!open) {
      setRecords([]);
    }
  }, [open, traceId, fetchRecords]);

  const totalTokens = records.reduce((sum, r) => sum + (r.total_tokens || 0), 0);
  const durationSec = records.length >= 2
    ? Math.round(((records[records.length - 1].timestamp || 0) - (records[0].timestamp || 0)) / 1000)
    : 0;

  return (
    <Modal
      title={`对话记录 - ${traceId?.slice(0, 12) || ''}...`}
      open={open}
      onCancel={onClose}
      footer={
        <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
          <Button onClick={onClose}>关闭</Button>
        </div>
      }
      width={800}
    >
      {loading ? (
        <div style={{ textAlign: 'center', padding: 40 }}>
          <Spin size="large" />
        </div>
      ) : records.length === 0 ? (
        <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>暂无对话记录</div>
      ) : (
        <div>
          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={8}>
              <Statistic title="消息数" value={records.length} />
            </Col>
            <Col span={8}>
              <Statistic title="总Token" value={totalTokens} />
            </Col>
            <Col span={8}>
              <Statistic title="时长" value={`${durationSec}s`} />
            </Col>
          </Row>
          <Divider />
          <div style={{ maxHeight: 500, overflowY: 'auto', padding: 16, background: '#f5f5f5', borderRadius: 8 }}>
            {records.map((r) => (
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
                    borderRadius: r.role === 'user' ? '16px 16px 4px 16px' : '16px 16px 16px 4px',
                    background: r.role === 'user' ? '#1890ff' : '#fff',
                    color: r.role === 'user' ? '#fff' : '#333',
                    boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
                  }}
                >
                  <div style={{ fontSize: 12, opacity: 0.7, marginBottom: 4 }}>
                    {getRoleLabel(r.role)} · {r.total_tokens || 0} tokens
                  </div>
                  <div style={{ wordBreak: 'break-word' }}>
                    {r.role === 'user' ? (
                      <div style={{ whiteSpace: 'pre-wrap' }}>{r.content || ''}</div>
                    ) : (
                      <XMarkdown content={r.content || ''} />
                    )}
                  </div>
                  <div style={{ fontSize: 11, opacity: 0.5, marginTop: 4, textAlign: 'right' }}>
                    {dayjs(r.timestamp).format('HH:mm:ss')}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </Modal>
  );
};