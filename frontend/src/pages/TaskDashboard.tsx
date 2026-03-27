/**
 * 任务仪表板页面
 * 显示任务列表和对话统计数据
 */
import React, { useCallback, useEffect, useState } from 'react';
import { Row, Col, Card, Statistic, Button, Space, Table, Tag, Modal, Popconfirm, message, DatePicker } from 'antd';
import { PlusOutlined, ReloadOutlined, EyeOutlined, BarChartOutlined, TeamOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
} from 'recharts';
import dayjs from 'dayjs';
import { TaskForm } from '../components/TaskForm';
import { TaskDetailDrawer } from '../components/TaskDetailDrawer';
import { StatusBadge } from '../components/StatusBadge';
import { useTaskStore } from '../stores/taskStore';
import { useTaskOperations } from '../hooks/useTaskOperations';
import type { Task, TaskStatus } from '../types/task';
import { clearAllTasks } from '../api/taskApi';
import { useAuthStore } from '../stores/authStore';
import { getConversationStats, StatsParams } from '../api/conversationRecordApi';
import type { ConversationStats } from '../types/conversationRecord';

const { RangePicker } = DatePicker;

// 颜色配置
const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884D8', '#82CA9D'];

export const TaskDashboard: React.FC = () => {
  const navigate = useNavigate();
  const { logout } = useAuthStore();
  const { tasks, loading, fetchTasks } = useTaskStore();
  const { createTask, cancelTask } = useTaskOperations();
  const [modalVisible, setModalVisible] = useState(false);
  const [drawerTaskId, setDrawerTaskId] = useState<string | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [clearing, setClearing] = useState(false);

  // 对话统计状态
  const [convStats, setConvStats] = useState<ConversationStats | null>(null);
  const [convLoading, setConvLoading] = useState(false);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(7, 'day'),
    dayjs(),
  ]);

  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  // 获取对话统计
  const fetchConversationStats = useCallback(async () => {
    setConvLoading(true);
    try {
      const [start, end] = dateRange;
      const params: StatsParams = {
        start_time: start.toISOString(),
        end_time: end.toISOString(),
      };
      const data = await getConversationStats(params);
      setConvStats(data);
    } catch (error) {
      console.error('获取对话统计失败:', error);
    } finally {
      setConvLoading(false);
    }
  }, [dateRange]);

  useEffect(() => {
    fetchConversationStats();
  }, [fetchConversationStats]);

  const rootTasks = tasks.filter((t) => !t.parent_id);

  const handleCreateTask = async (values: Parameters<typeof createTask>[0]) => {
    const result = await createTask(values);
    if (result) {
      setModalVisible(false);
      fetchTasks();
    }
  };

  const handleCancelTask = async (taskId: string) => {
    const success = await cancelTask(taskId);
    if (success) {
      fetchTasks();
    }
  };

  const handleViewDetail = (taskId: string) => {
    setDrawerTaskId(taskId);
    setDrawerOpen(true);
  };

  const handleDrawerClose = () => {
    setDrawerOpen(false);
    setDrawerTaskId(null);
  };

  const handleClearAllTasks = async () => {
    setClearing(true);
    try {
      const result = await clearAllTasks();
      message.success(`已清空 ${result.deleted} 个任务`);
      if (drawerOpen) {
        handleDrawerClose();
      }
      await fetchTasks();
    } catch (error) {
      message.error('清空任务失败');
    } finally {
      setClearing(false);
    }
  };

  const statusCounts = {
    pending: rootTasks.filter((t) => t.status === 'pending').length,
    running: rootTasks.filter((t) => t.status === 'running').length,
    completed: rootTasks.filter((t) => t.status === 'completed').length,
    failed: rootTasks.filter((t) => t.status === 'failed').length,
  };

  // Token 趋势图表数据
  const tokenTrendData = convStats?.token_stats.daily_trends || [];

  // Agent 分布图表数据
  const agentDistData =
    convStats?.agent_distribution.map((item) => ({
      name: item.name || item.code,
      value: item.count,
      count: item.count,
      tokens: item.tokens,
    })) || [];

  // Agent 分布表格列定义
  const agentColumns = [
    { title: 'Agent', dataIndex: 'name', key: 'name' },
    { title: '消息数', dataIndex: 'count', key: 'count' },
    {
      title: 'Token 数',
      dataIndex: 'tokens',
      key: 'tokens',
      render: (tokens: number) => tokens?.toLocaleString() || 0,
    },
  ];

  // Channel 分布图表数据
  const channelDistData =
    convStats?.channel_distribution.map((item) => ({
      name: item.type || '未知',
      value: item.count,
    })) || [];

  const columns = [
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => <StatusBadge status={status as TaskStatus} />,
    },
    {
      title: '任务名称',
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 120,
      render: (type: string) => <Tag color="blue">{type}</Tag>,
    },
    {
      title: '进度',
      key: 'progress',
      width: 150,
      render: (_: unknown, record: Task) => {
        const p = record.progress?.value || 0;
        return (
          <div style={{ width: 100 }}>
            <div style={{ fontSize: 12, marginBottom: 4 }}>
              {p}%
            </div>
            <div style={{ background: '#f0f0f0', height: 6, borderRadius: 3 }}>
              <div
                style={{
                  width: `${p}%`,
                  background: p === 100 ? '#52c41a' : '#1890ff',
                  height: 6,
                  borderRadius: 3,
                  transition: 'width 0.3s',
                }}
              />
            </div>
          </div>
        );
      },
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (time: number) => new Date(time).toLocaleString(),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: unknown, record: Task) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => handleViewDetail(record.id)}
          >
            查看
          </Button>
          {record.status === 'pending' && (
            <Button type="link" size="small" danger onClick={() => handleCancelTask(record.id)}>
              取消
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      {/* 任务状态统计 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic title="待处理" value={statusCounts.pending} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="运行中" value={statusCounts.running} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="已完成" value={statusCounts.completed} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="失败" value={statusCounts.failed} />
          </Card>
        </Col>
      </Row>

      {/* 对话统计 */}
      <Card
        title={
          <Space>
            <BarChartOutlined />
            <span>对话统计</span>
          </Space>
        }
        extra={
          <Space>
            <RangePicker
              value={dateRange}
              onChange={(dates) => dates && setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs])}
              style={{ width: 220 }}
            />
            <Button type="primary" onClick={fetchConversationStats} loading={convLoading}>
              刷新
            </Button>
            <Button onClick={() => navigate('/conversation-records')}>
              查看详情
            </Button>
          </Space>
        }
        style={{ marginBottom: 24 }}
      >
        {/* 对话核心指标 */}
        <Row gutter={16} style={{ marginBottom: 24 }}>
          <Col xs={24} sm={12} md={8} lg={6}>
            <Card bordered={false} style={{ background: '#f6ffed' }}>
              <Statistic
                title="总会话数"
                value={convStats?.session_stats.total_sessions || 0}
                prefix={<TeamOutlined />}
                valueStyle={{ color: '#52c41a' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} md={8} lg={6}>
            <Card bordered={false} style={{ background: '#e6f7ff' }}>
              <Statistic
                title="总 Token 数"
                value={convStats?.token_stats.total_tokens || 0}
                prefix={<ThunderboltOutlined />}
                valueStyle={{ color: '#1890ff' }}
                formatter={(value) => value.toLocaleString()}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} md={8} lg={6}>
            <Card bordered={false} style={{ background: '#fff7e6' }}>
              <Statistic
                title="平均消息数/会话"
                value={convStats?.session_stats.avg_messages_per_session || 0}
                precision={1}
                valueStyle={{ color: '#fa8c16' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} md={8} lg={6}>
            <Card bordered={false} style={{ background: '#f9f0ff' }}>
              <Statistic
                title="平均响应时间"
                value={convStats?.session_stats.avg_response_time_ms || 0}
                precision={0}
                suffix="ms"
                valueStyle={{ color: '#722ed1' }}
              />
            </Card>
          </Col>
        </Row>

        {/* Token 消耗趋势 */}
        <Card title="Token 消耗趋势" style={{ marginBottom: 16 }}>
          <ResponsiveContainer width="100%" height={250}>
            <LineChart data={tokenTrendData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="date" />
              <YAxis />
              <Tooltip formatter={(value) => typeof value === 'number' ? value.toLocaleString() : value} />
              <Legend />
              <Line
                type="monotone"
                dataKey="prompt_tokens"
                name="Prompt Tokens"
                stroke="#1890ff"
                strokeWidth={2}
              />
              <Line
                type="monotone"
                dataKey="complete_tokens"
                name="Completion Tokens"
                stroke="#52c41a"
                strokeWidth={2}
              />
              <Line
                type="monotone"
                dataKey="total_tokens"
                name="Total Tokens"
                stroke="#fa8c16"
                strokeWidth={2}
              />
            </LineChart>
          </ResponsiveContainer>
        </Card>

        {/* 分布图表 */}
        <Row gutter={16}>
          <Col xs={24} md={12}>
            <Card title="Agent 使用分布">
              <ResponsiveContainer width="100%" height={200}>
                <PieChart>
                  <Pie
                    data={agentDistData}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={(props: { name?: string; percent?: number }) => `${props.name || ''}: ${((props.percent || 0) * 100).toFixed(0)}%`}
                    outerRadius={70}
                    fill="#8884d8"
                    dataKey="value"
                  >
                    {agentDistData.map((_entry, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
              <Table
                dataSource={agentDistData}
                columns={agentColumns}
                rowKey="name"
                pagination={false}
                size="small"
                style={{ marginTop: 16 }}
              />
            </Card>
          </Col>
          <Col xs={24} md={12}>
            <Card title="Channel 来源分布">
              <ResponsiveContainer width="100%" height={200}>
                <PieChart>
                  <Pie
                    data={channelDistData}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={(props: { name?: string; percent?: number }) => `${props.name || ''}: ${((props.percent || 0) * 100).toFixed(0)}%`}
                    outerRadius={70}
                    fill="#8884d8"
                    dataKey="value"
                  >
                    {channelDistData.map((_entry, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
            </Card>
          </Col>
        </Row>
      </Card>

      {/* 任务列表 */}
      <Card
        title={`根任务列表 (${rootTasks.length})`}
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => fetchTasks()}>
              刷新
            </Button>
            <Popconfirm
              title="确认清空全部任务？"
              description="该操作会删除所有任务数据，无法恢复。"
              okText="确认清空"
              cancelText="取消"
              onConfirm={handleClearAllTasks}
            >
              <Button danger loading={clearing}>
                删除全部任务
              </Button>
            </Popconfirm>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalVisible(true)}>
              创建任务
            </Button>
            <Button
              danger
              onClick={() => {
                logout();
                navigate('/login', { replace: true });
              }}
            >
              退出登录
            </Button>
          </Space>
        }
      >
        <Table
          dataSource={rootTasks}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 10 }}
          size="middle"
        />
      </Card>

      <Modal
        title="创建任务"
        open={modalVisible}
        footer={null}
        onCancel={() => setModalVisible(false)}
        width={600}
      >
        <TaskForm onSubmit={handleCreateTask} onCancel={() => setModalVisible(false)} />
      </Modal>

      <TaskDetailDrawer taskId={drawerTaskId} open={drawerOpen} onClose={handleDrawerClose} />
    </div>
  );
};
