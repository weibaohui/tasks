import React from 'react';
import { Card, Row, Col, Badge } from 'antd';
import type { StatusStat } from '../../api/projectRequirementApi';
import { statusLabels, getStatusColor } from '../../constants/requirementStatus';

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
      title={<span style={{ fontSize: 14 }}>状态统计</span>}
      size="small"
      style={{ marginBottom: 16 }}
      bodyStyle={{ padding: '12px' }}
    >
      <Row gutter={[12, 12]}>
        {/* 全部状态卡片 */}
        <Col xs={12} sm={8} md={6} lg={4}>
          <div
            onClick={() => onStatusClick('')}
            style={{
              cursor: 'pointer',
              padding: '8px 12px',
              borderRadius: '6px',
              border: `1px solid ${!statusFilter ? '#1890ff' : '#e8e8e8'}`,
              backgroundColor: !statusFilter ? '#e6f7ff' : '#fafafa',
              transition: 'all 0.3s ease',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between'
            }}
          >
            <div
              style={{
                fontSize: '12px',
                color: !statusFilter ? '#1890ff' : '#595959',
              }}
            >
              <Badge color="#1890ff" text="全部" />
            </div>
            <div
              style={{
                fontSize: '16px',
                fontWeight: 'bold',
                color: !statusFilter ? '#1890ff' : '#262626',
              }}
            >
              {getTotalCount()}
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
                  padding: '8px 12px',
                  borderRadius: '6px',
                  border: `1px solid ${active ? colors.color : colors.borderColor}`,
                  backgroundColor: active ? colors.bgColor : '#ffffff',
                  transition: 'all 0.3s ease',
                  opacity: stat.count === 0 ? 0.6 : 1,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between'
                }}
              >
                <div
                  style={{
                    fontSize: '12px',
                    color: active ? colors.color : '#595959',
                  }}
                >
                  <Badge color={colors.color} text={label} />
                </div>
                <div
                  style={{
                    fontSize: '16px',
                    fontWeight: 'bold',
                    color: colors.color,
                  }}
                >
                  {stat.count}
                </div>
              </div>
            </Col>
          );
        })}
      </Row>
    </Card>
  );
};
