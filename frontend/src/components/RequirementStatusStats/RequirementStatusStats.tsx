import React from 'react';
import { Card, Row, Col, Badge } from 'antd';
import type { StatusStat } from '../../api/projectRequirementApi';

// 状态颜色配置
const statusColors: Record<string, { color: string; bgColor: string; borderColor: string }> = {
  todo: { color: '#666666', bgColor: '#f5f5f5', borderColor: '#d9d9d9' },
  preparing: { color: '#d48806', bgColor: '#fffbe6', borderColor: '#ffd666' },
  understanding: { color: '#722ed1', bgColor: '#f9f0ff', borderColor: '#d3adf7' },
  analyzing: { color: '#eb2f96', bgColor: '#fff0f6', borderColor: '#ffadd2' },
  implementing: { color: '#0958d9', bgColor: '#e6f4ff', borderColor: '#91caff' },
  submitting: { color: '#fa8c16', bgColor: '#fff7e6', borderColor: '#ffd591' },
  coding: { color: '#0958d9', bgColor: '#e6f4ff', borderColor: '#91caff' },
  pr_opened: { color: '#389e0d', bgColor: '#f6ffed', borderColor: '#b7eb8f' },
  failed: { color: '#cf1322', bgColor: '#fff2f0', borderColor: '#ffccc7' },
  completed: { color: '#52c41a', bgColor: '#f6ffed', borderColor: '#b7eb8f' },
  done: { color: '#237804', bgColor: '#d9f7be', borderColor: '#95de64' },
};

// 获取状态默认颜色
function getStatusColor(status: string) {
  return statusColors[status] || { color: '#8c8c8c', bgColor: '#f5f5f5', borderColor: '#d9d9d9' };
}

// 状态中文名映射
const statusLabels: Record<string, string> = {
  todo: '待处理',
  preparing: '准备中',
  understanding: '理解需求',
  analyzing: '分析方案',
  implementing: '编写代码',
  submitting: '提交PR',
  coding: '编码中',
  pr_opened: 'PR已开',
  failed: '失败',
  completed: '已完成',
  done: '完成',
};

interface RequirementStatusStatsProps {
  statusStats: StatusStat[];
  statusFilter: string;
  onStatusClick: (status: string) => void;
}

export const RequirementStatusStats: React.FC<RequirementStatusStatsProps> = ({
  statusStats,
  statusFilter,
  onStatusClick,
}) => {
  const getTotalCount = () => {
    return statusStats.reduce((sum, stat) => sum + stat.count, 0);
  };

  const isActive = (status: string) => {
    return statusFilter === status;
  };

  return (
    <Card
      title="状态统计"
      style={{ marginBottom: 16 }}
      bodyStyle={{ padding: '16px' }}
    >
      <Row gutter={[16, 16]}>
        {/* 全部状态卡片 */}
        <Col xs={12} sm={8} md={6} lg={4}>
          <div
            onClick={() => onStatusClick('')}
            style={{
              cursor: 'pointer',
              padding: '16px',
              borderRadius: '8px',
              border: `2px solid ${!statusFilter ? '#1890ff' : '#e8e8e8'}`,
              backgroundColor: !statusFilter ? '#e6f7ff' : '#fafafa',
              transition: 'all 0.3s ease',
              textAlign: 'center',
            }}
          >
            <div
              style={{
                fontSize: '24px',
                fontWeight: 'bold',
                color: !statusFilter ? '#1890ff' : '#262626',
                marginBottom: '4px',
              }}
            >
              {getTotalCount()}
            </div>
            <div
              style={{
                fontSize: '14px',
                color: !statusFilter ? '#1890ff' : '#595959',
              }}
            >
              <Badge color="#1890ff" text="全部" />
            </div>
          </div>
        </Col>

        {/* 各状态卡片 - 动态从数据库获取 */}
        {statusStats.map((stat) => {
          const colors = getStatusColor(stat.status);
          const active = isActive(stat.status);
          const label = statusLabels[stat.status] || stat.status;

          return (
            <Col xs={12} sm={8} md={6} lg={4} key={stat.status}>
              <div
                onClick={() => onStatusClick(stat.status)}
                style={{
                  cursor: 'pointer',
                  padding: '16px',
                  borderRadius: '8px',
                  border: `2px solid ${active ? colors.color : colors.borderColor}`,
                  backgroundColor: active ? colors.bgColor : '#ffffff',
                  transition: 'all 0.3s ease',
                  textAlign: 'center',
                  opacity: stat.count === 0 ? 0.6 : 1,
                }}
              >
                <div
                  style={{
                    fontSize: '24px',
                    fontWeight: 'bold',
                    color: colors.color,
                    marginBottom: '4px',
                  }}
                >
                  {stat.count}
                </div>
                <div
                  style={{
                    fontSize: '14px',
                    color: active ? colors.color : '#595959',
                  }}
                >
                  <Badge color={colors.color} text={label} />
                </div>
              </div>
            </Col>
          );
        })}
      </Row>
    </Card>
  );
};
