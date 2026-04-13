/**
 * Dashboard 页面
 * 显示任务统计概览和对话用量分析
 */
import React, { useCallback, useEffect, useState } from 'react';
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
import { getConversationStats } from '../api/conversationRecordApi';
import type { ConversationStats } from '../types/conversationRecord';

const { RangePicker } = DatePicker;
const { Title } = Typography;

// 颜色配置
const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884D8', '#82CA9D'];

export const Dashboard: React.FC = () => {
  // 对话统计
  const [statsLoading, setStatsLoading] = useState(false);
  const [stats, setStats] = useState<ConversationStats | null>(null);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(7, 'day'),
    dayjs(),
  ]);

  const fetchStats = useCallback(async () => {
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
  }, [dateRange]);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

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

  // 项目 Token 分布图表数据
  const projectDistData =
    stats?.project_distribution.map((item) => ({
      name: item.name || '未命名项目',
      value: item.tokens,
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
    <div style={{ padding: 0 }}>
      <div style={{
        marginBottom: 24,
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        flexWrap: 'wrap',
        gap: 12
      }}>
        <Title level={3} style={{ margin: 0 }}>Dashboard</Title>
        <Space wrap>
          <RangePicker
            value={dateRange}
            onChange={(dates) => dates && setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs])}
            style={{ width: '100%', maxWidth: 280 }}
          />
          <Button type="primary" icon={<ReloadOutlined />} onClick={fetchStats} loading={statsLoading}>
            刷新
          </Button>
        </Space>
      </div>

      {/* 对话核心指标 - 响应式布局 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={12} sm={12} md={6} lg={6}>
          <Card>
            <Statistic
              title={<span style={{ fontSize: 12 }}>总会话数</span>}
              value={stats?.session_stats.total_sessions || 0}
              valueStyle={{ color: '#1890ff', fontSize: 20 }}
            />
          </Card>
        </Col>
        <Col xs={12} sm={12} md={6} lg={6}>
          <Card>
            <Statistic
              title={<span style={{ fontSize: 12 }}>总 Token 数</span>}
              value={stats?.token_stats.total_tokens || 0}
              valueStyle={{ color: '#52c41a', fontSize: 20 }}
              formatter={(value) => value.toLocaleString()}
            />
          </Card>
        </Col>
        <Col xs={12} sm={12} md={6} lg={6}>
          <Card>
            <Statistic
              title={<span style={{ fontSize: 12 }}>平均消息数/会话</span>}
              value={stats?.session_stats.avg_messages_per_session || 0}
              precision={1}
              valueStyle={{ color: '#fa8c16', fontSize: 20 }}
            />
          </Card>
        </Col>
        <Col xs={12} sm={12} md={6} lg={6}>
          <Card>
            <Statistic
              title={<span style={{ fontSize: 12 }}>平均响应时间</span>}
              value={stats?.session_stats.avg_response_time_ms || 0}
              precision={0}
              suffix="ms"
              valueStyle={{ color: '#722ed1', fontSize: 20 }}
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

      {/* 项目 Token 消耗分布 - 响应式布局 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} lg={12}>
          <Card title="项目 Token 消耗排行">
            <ResponsiveContainer width="100%" height={250}>
              <BarChart data={projectDistData} layout="vertical" margin={{ left: 20 }}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis type="number" />
                <YAxis dataKey="name" type="category" width={100} />
                <Tooltip formatter={(value) => (typeof value === 'number' ? value.toLocaleString() : value)} />
                <Bar dataKey="value" name="Token 数" fill="#1890ff" />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="项目 Token 消耗占比">
            <ResponsiveContainer width="100%" height={250}>
              <PieChart>
                <Pie
                  data={projectDistData}
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
                  {projectDistData.map((_entry, index) => (
                    <Cell key={`cell-project-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip formatter={(value: any) => typeof value === 'number' ? value.toLocaleString() : value} />
              </PieChart>
            </ResponsiveContainer>
          </Card>
        </Col>
      </Row>

      {/* Agent / Channel 分布 - 响应式布局 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} lg={12}>
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
        <Col xs={24} lg={12}>
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

      {/* 角色分布 / Token 统计 - 响应式布局 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
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
        <Col xs={24} lg={12}>
          <Card title="Token 使用统计">
            <Row gutter={[8, 8]}>
              <Col xs={8}>
                <Statistic
                  title={<span style={{ fontSize: 11 }}>Prompt Tokens</span>}
                  value={stats?.token_stats.total_prompt_tokens || 0}
                  formatter={(value) => value.toLocaleString()}
                />
              </Col>
              <Col xs={8}>
                <Statistic
                  title={<span style={{ fontSize: 11 }}>Completion Tokens</span>}
                  value={stats?.token_stats.total_completion_tokens || 0}
                  formatter={(value) => value.toLocaleString()}
                />
              </Col>
              <Col xs={8}>
                <Statistic
                  title={<span style={{ fontSize: 11 }}>总 Token 数</span>}
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
