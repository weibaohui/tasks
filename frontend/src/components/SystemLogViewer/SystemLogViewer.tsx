import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Empty,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Space,
  Switch,
  Tag,
  Typography,
  message,
} from 'antd';
import { ClearOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import {
  buildSystemLogStreamURL,
  clearSystemLog,
  getSystemLogConfig,
  getSystemLogTail,
  SystemLogLine,
  SystemLogStreamEvent,
} from '../../api/systemLogApi';
import './index.css';

const { Text } = Typography;

const DEFAULT_TAIL_LINES = 200;
const MIN_TAIL_LINES = 20;
const MAX_TAIL_LINES = 5000;
const MAX_BUFFER_LINES = 10000;
const ROW_HEIGHT = 26;
const OVERSCAN = 16;

interface LogRow extends SystemLogLine {
  id: number;
}

/**
 * SystemLogViewer 提供系统日志流式查看能力。
 */
export const SystemLogViewer: React.FC = () => {
  const [logPath, setLogPath] = useState('');
  const [rows, setRows] = useState<LogRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [tailLines, setTailLines] = useState(DEFAULT_TAIL_LINES);
  const [serverKeywordInput, setServerKeywordInput] = useState('');
  const [serverKeyword, setServerKeyword] = useState('');
  const [clientKeyword, setClientKeyword] = useState('');
  const [streamStatus, setStreamStatus] = useState<'connected' | 'connecting' | 'disconnected'>('connecting');
  const [streamError, setStreamError] = useState('');
  const nextRowIdRef = useRef(1);

  const viewportRef = useRef<HTMLDivElement | null>(null);
  const [scrollTop, setScrollTop] = useState(0);
  const reconnectTimerRef = useRef<number | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  const filteredRows = useMemo(() => {
    const keyword = clientKeyword.trim();
    if (!keyword) {
      return rows;
    }
    return rows.filter((item) => item.text.includes(keyword));
  }, [rows, clientKeyword]);

  const visibleRange = useMemo(() => {
    const viewportHeight = viewportRef.current?.clientHeight || 520;
    const start = Math.max(0, Math.floor(scrollTop / ROW_HEIGHT) - OVERSCAN);
    const count = Math.ceil(viewportHeight / ROW_HEIGHT) + OVERSCAN * 2;
    const end = Math.min(filteredRows.length, start + count);
    return { start, end };
  }, [scrollTop, filteredRows.length]);

  const visibleRows = useMemo(
    () => filteredRows.slice(visibleRange.start, visibleRange.end),
    [filteredRows, visibleRange.start, visibleRange.end]
  );

  /**
   * connectStream 建立日志 SSE 连接。
   */
  const connectStream = React.useCallback(() => {
    if (!autoRefresh) {
      return;
    }

    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }

    setStreamStatus('connecting');
    setStreamError('');
    const source = new EventSource(buildSystemLogStreamURL({ lines: tailLines, keyword: serverKeyword }));
    eventSourceRef.current = source;

    source.onopen = () => {
      setStreamStatus('connected');
    };

    source.onmessage = (event) => {
      let payload: SystemLogStreamEvent;
      try {
        payload = JSON.parse(event.data) as SystemLogStreamEvent;
      } catch {
        return;
      }

      if (payload.type === 'error') {
        setStreamError(payload.message || '日志流异常');
        return;
      }

      if (payload.type === 'reset') {
        message.warning('检测到日志文件被截断，已自动重置流读取');
        return;
      }

      if (payload.path) {
        setLogPath(payload.path);
      }

      if (!payload.lines || payload.lines.length === 0) {
        return;
      }

      const container = viewportRef.current;
      const nearBottom = container
        ? container.scrollHeight - (container.scrollTop + container.clientHeight) < 80
        : false;

      setRows((prev) => {
        const incoming = payload.lines || [];
        const startId = nextRowIdRef.current;
        const mapped: LogRow[] = incoming.map((item, index) => ({
          id: startId + index,
          line: item.line,
          text: item.text,
        }));
        const merged = payload.type === 'snapshot' ? mapped : [...prev, ...mapped];
        if (merged.length > MAX_BUFFER_LINES) {
          return merged.slice(merged.length - MAX_BUFFER_LINES);
        }
        return merged;
      });
      nextRowIdRef.current += payload.lines?.length || 0;

      if (nearBottom) {
        window.requestAnimationFrame(() => {
          if (viewportRef.current) {
            viewportRef.current.scrollTop = viewportRef.current.scrollHeight;
          }
        });
      }
    };

    source.onerror = () => {
      setStreamStatus('disconnected');
      source.close();
      eventSourceRef.current = null;
      if (!autoRefresh) {
        return;
      }
      setStreamError('日志流已断开，正在尝试重连...');
      reconnectTimerRef.current = window.setTimeout(() => {
        connectStream();
      }, 1500);
    };
  }, [autoRefresh, tailLines, serverKeyword]);

  /**
   * loadTail 拉取当前日志尾部数据。
   */
  const loadTail = React.useCallback(async () => {
    setLoading(true);
    try {
      const [config, tail] = await Promise.all([
        getSystemLogConfig(),
        getSystemLogTail({ lines: tailLines, keyword: serverKeyword }),
      ]);
      setLogPath(config.path);
      const mapped: LogRow[] = tail.lines.map((item, index) => ({
        id: index + 1,
        line: item.line,
        text: item.text,
      }));
      setRows(mapped);
      nextRowIdRef.current = mapped.length + 1;
      setStreamError('');
    } catch {
      message.error('读取日志失败');
    } finally {
      setLoading(false);
    }
  }, [tailLines, serverKeyword]);

  /**
   * handleClear 清空日志文件内容。
   */
  const handleClear = async () => {
    try {
      await clearSystemLog();
      message.success('日志文件已清空');
      await loadTail();
    } catch {
      message.error('清空日志失败');
    }
  };

  useEffect(() => {
    void loadTail();
  }, [loadTail]);

  useEffect(() => {
    if (autoRefresh) {
      connectStream();
    } else if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
      setStreamStatus('disconnected');
    }

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      if (reconnectTimerRef.current) {
        window.clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
    };
  }, [autoRefresh, connectStream]);

  return (
    <Card
      title="系统运行日志"
      className="system-log-card"
      extra={
        <Space size={8} wrap>
          <Tag color={streamStatus === 'connected' ? 'success' : streamStatus === 'connecting' ? 'processing' : 'default'}>
            {streamStatus === 'connected' ? '流已连接' : streamStatus === 'connecting' ? '连接中' : '已断开'}
          </Tag>
          <Switch checked={autoRefresh} onChange={setAutoRefresh} checkedChildren="自动刷新" unCheckedChildren="手动模式" />
        </Space>
      }
    >
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        <Text type="secondary" className="system-log-path">日志文件: {logPath || '加载中...'}</Text>

        {streamError ? (
          <Alert type="warning" showIcon message={streamError} />
        ) : null}

        <Form layout="vertical">
          <Space wrap size={12} style={{ width: '100%' }}>
            <Form.Item label="服务端过滤" style={{ marginBottom: 0 }}>
              <Input
                allowClear
                placeholder="按关键字过滤后再推流"
                value={serverKeywordInput}
                onChange={(e) => setServerKeywordInput(e.target.value)}
                style={{ width: 240 }}
              />
            </Form.Item>
            <Form.Item label="前端搜索" style={{ marginBottom: 0 }}>
              <Input
                allowClear
                prefix={<SearchOutlined />}
                placeholder="仅在当前缓冲区搜索"
                value={clientKeyword}
                onChange={(e) => setClientKeyword(e.target.value)}
                style={{ width: 220 }}
              />
            </Form.Item>
            <Form.Item label="显示行数" style={{ marginBottom: 0 }}>
              <InputNumber
                min={MIN_TAIL_LINES}
                max={MAX_TAIL_LINES}
                value={tailLines}
                onChange={(val) => setTailLines(Number(val || DEFAULT_TAIL_LINES))}
                style={{ width: 120 }}
              />
            </Form.Item>
            <Form.Item label="操作" style={{ marginBottom: 0 }}>
              <Space wrap>
                <Button
                  type="primary"
                  onClick={() => {
                    setServerKeyword(serverKeywordInput.trim());
                    setRows([]);
                    nextRowIdRef.current = 1;
                    void loadTail();
                    if (autoRefresh) {
                      connectStream();
                    }
                  }}
                >
                  应用过滤
                </Button>
                <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void loadTail()}>
                  刷新
                </Button>
                <Popconfirm
                  title="确认清空日志？"
                  description="只会清空文件内容，不会删除日志文件。"
                  okText="确认清空"
                  cancelText="取消"
                  okButtonProps={{ danger: true }}
                  onConfirm={handleClear}
                >
                  <Button danger icon={<ClearOutlined />}>
                    清空日志
                  </Button>
                </Popconfirm>
              </Space>
            </Form.Item>
          </Space>
        </Form>

        <div className="system-log-toolbar">
          <Text type="secondary">当前缓冲区 {rows.length} 行（上限 {MAX_BUFFER_LINES}）</Text>
          <Text type="secondary">过滤后显示 {filteredRows.length} 行</Text>
        </div>

        <div
          ref={viewportRef}
          className="system-log-viewport"
          onScroll={(e) => setScrollTop(e.currentTarget.scrollTop)}
        >
          {filteredRows.length === 0 ? (
            <div className="system-log-empty">
              <Empty description="暂无日志输出" />
            </div>
          ) : (
            <div style={{ height: filteredRows.length * ROW_HEIGHT, position: 'relative' }}>
              <div style={{ transform: `translateY(${visibleRange.start * ROW_HEIGHT}px)` }}>
                {visibleRows.map((row) => (
                  <div className="system-log-row" key={row.id}>
                    <span className="system-log-line">{row.line}</span>
                    <span className="system-log-text">{row.text || ' '}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </Space>
    </Card>
  );
};

export default SystemLogViewer;
