/**
 * Dashboard 页面
 * 显示任务统计概览和对话用量分析
 */
import React, { useEffect, useState } from 'react';
import { Row, Col, Card, Statistic, DatePicker, Button, Space, Typography, message } from 'antd';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  LineChart,
  Line,
} from 'recharts';
import { ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { useTaskStore } from '../stores/taskStore';
import { getConversationStats } from '../api/conversationRecordApi';
import type { ConversationStats } from '../types/conversationRecord';

const { RangePicker } = DatePicker;
const { Title } = Typography;

// 颜色配置
const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884D8', '#82CA9D'];

export const Dashboard: React.FC = () => {
  // 任务统计
  const { tasks, fetchTasks } = useTaskStore();

  // 对话统计
  const [statsLoading, setStatsLoading] = useState(false);
  const [stats, setStats] = useState<ConversationStats | null>(null);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(7, 'day'),
    dayjs(),
  ]);

  useEffect(() => {
    fetchTasks();
    fetchStats();
  }, [fetchTasks]);

  const fetchStats = async () => {
    setStatsLoading(true);
    try {
      const [start, end] = dateRange;
      const res = await getConversationStats({
        start_time: start.toISOString(),
        end_time: end.toISOString(),
      });
      setStats(res);
    } catch (error) {
      message.error('获取统计数据失败');
    } finally {
      setStatsLoading(false);
    }
  };

  // 任务统计计算
  const rootTasks = tasks.filter((t) => !t.parent_id);
  const statusCounts = {
    pending: rootTasks.filter((t) => t.status === 'pending').length,
    running: rootTasks.filter((t) => t.status === 'running').length,
    completed: rootTasks.filter((t) => t.status === 'completed').length,
    failed: rootTasks.filter((t) => t.status === 'failed').length,
  };

  // Token 趋势图表数据
  const tokenTrendData = stats?.token_stats.daily_trends || [];

  // Agent 分布图表数据
  const agentDistData =
    stats?.agent_distribution.map((item) => ({
      name: item.name || item.code,
      count: item.count,
    })) || [];

  // Channel 分布图表数据
  const channelDistData =
    stats?.channel_distribution.map((item) => ({
      name: item.type || '未知',
      value: item.count,
    })) || [];

  // 角色分布图表数据
  const roleDistData =
    stats?.role_distribution.map((item) => ({
      name: getRoleLabel(item.role),
      value: item.count,
    })) || [];

  function getRoleLabel(role: string): string {
    const labels: Record<string, string> = {
      user: '用户',
      assistant: '助手',
      system: '系统',
      tool: '工具',
      tool_result: '工具结果',
    };
    return labels[role] || role;
  }

  return (
    <div style={{ padding: 24 }}>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={3} style={{ margin: 0 }}>Dashboard</Title>
        <Space>
          <RangePicker
            value={dateRange}
            onChange={(dates) => dates && setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs])}
          />
          <Button type="primary" icon={<ReloadOutlined />} onClick={fetchStats} loading={statsLoading}>
            刷新
          </Button>
        </Space>
      </div>

      {/* 任务统计 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic title="待处理" value={statusCounts.pending} valueStyle={{ color: '#faad14' }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="运行中" value={statusCounts.running} valueStyle={{ color: '#1890ff' }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="已完成" value={statusCounts.completed} valueStyle={{ color: '#52c41a' }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="失败" value={statusCounts.failed} valueStyle={{ color: '#f5222d' }} />
          </Card>
        </Col>
      </Row>

      {/* 对话核心指标 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总会话数"
              value={stats?.session_stats.total_sessions || 0}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="总 Token 数"
              value={stats?.token_stats.total_tokens || 0}
              valueStyle={{ color: '#52c41a' }}
              formatter={(value) => value.toLocaleString()}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="平均消息数/会话"
              value={stats?.session_stats.avg_messages_per_session || 0}
              precision={1}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="平均响应时间"
              value={stats?.session_stats.avg_response_time_ms || 0}
              precision={0}
              suffix="ms"
              valueStyle={{ color: '#722ed1' }}
            />
          </Card>
        </Col>
      </Row>

      {/* Token 趋势图 */}
      <Card title="Token 消耗趋势" style={{ marginBottom: 24 }}>
        <ResponsiveContainer width="100%" height={300}>
          <LineChart data={tokenTrendData}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="date" />
            <YAxis />
            <Tooltip formatter={(value) => (typeof value === 'number' ? value.toLocaleString() : value)} />
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

      {/* Agent / Channel 分布 */}
      <Row gutter={24} style={{ marginBottom: 24 }}>
        <Col span={12}>
          <Card title="Agent 使用分布">
            <ResponsiveContainer width="100%" height={250}>
              <BarChart data={agentDistData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip />
                <Bar dataKey="count" name="消息数" fill="#1890ff" />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col span={12}>
          <Card title="Channel 来源分布">
            <ResponsiveContainer width="100%" height={250}>
              <PieChart>
                <Pie
                  data={channelDistData}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={(props: any) =>
                    `${props.name || ''}: ${((props.percent || 0) * 100).toFixed(0)}%`
                  }
                  outerRadius={80}
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

      {/* 角色分布 / Token 统计 */}
      <Row gutter={24}>
        <Col span={12}>
          <Card title="角色消息分布">
            <ResponsiveContainer width="100%" height={250}>
              <PieChart>
                <Pie
                  data={roleDistData}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={(props: any) =>
                    `${props.name || ''}: ${((props.percent || 0) * 100).toFixed(0)}%`
                  }
                  outerRadius={80}
                  fill="#8884d8"
                  dataKey="value"
                >
                  {roleDistData.map((_entry, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col span={12}>
          <Card title="Token 使用统计">
            <Row gutter={16}>
              <Col span={8}>
                <Statistic
                  title="Prompt Tokens"
                  value={stats?.token_stats.total_prompt_tokens || 0}
                  formatter={(value) => value.toLocaleString()}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title="Completion Tokens"
                  value={stats?.token_stats.total_completion_tokens || 0}
                  formatter={(value) => value.toLocaleString()}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title="总 Token 数"
                  value={stats?.token_stats.total_tokens || 0}
                  formatter={(value) => value.toLocaleString()}
                  valueStyle={{ color: '#1890ff', fontWeight: 'bold' }}
                />
              </Col>
            </Row>
          </Card>
        </Col>
      </Row>
    </div>
  );
};
