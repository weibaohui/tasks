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
