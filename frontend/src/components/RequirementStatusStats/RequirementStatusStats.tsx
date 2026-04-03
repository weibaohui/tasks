import React from 'react';
import { Card, Row, Col, Badge } from 'antd';
import type { Requirement } from '../../types/projectRequirement';

interface RequirementStatusStatsProps {
  requirements: Requirement[];
  statusFilter: string;
  onStatusClick: (status: string) => void;
}

interface StatusConfig {
  key: string;
  label: string;
  color: string;
  bgColor: string;
  borderColor: string;
}

const statusConfigs: StatusConfig[] = [
  {
    key: 'todo',
    label: '待处理',
    color: '#666666',
    bgColor: '#f5f5f5',
    borderColor: '#d9d9d9',
  },
  {
    key: 'preparing',
    label: '准备中',
    color: '#d48806',
    bgColor: '#fffbe6',
    borderColor: '#ffd666',
  },
  {
    key: 'coding',
    label: '编码中',
    color: '#0958d9',
    bgColor: '#e6f4ff',
    borderColor: '#91caff',
  },
  {
    key: 'pr_opened',
    label: 'PR已开',
    color: '#389e0d',
    bgColor: '#f6ffed',
    borderColor: '#b7eb8f',
  },
  {
    key: 'failed',
    label: '失败',
    color: '#cf1322',
    bgColor: '#fff2f0',
    borderColor: '#ffccc7',
  },
  {
    key: 'completed',
    label: '已完成',
    color: '#52c41a',
    bgColor: '#f6ffed',
    borderColor: '#b7eb8f',
  },
  {
    key: 'done',
    label: '完成',
    color: '#237804',
    bgColor: '#d9f7be',
    borderColor: '#95de64',
  },
];

export const RequirementStatusStats: React.FC<RequirementStatusStatsProps> = ({
  requirements,
  statusFilter,
  onStatusClick,
}) => {
  const getStatusCount = (status: string) => {
    return requirements.filter((req) => req.status === status).length;
  };

  const getTotalCount = () => {
    return requirements.length;
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

        {/* 各状态卡片 */}
        {statusConfigs.map((config) => {
          const count = getStatusCount(config.key);
          const active = isActive(config.key);

          return (
            <Col xs={12} sm={8} md={6} lg={4} key={config.key}>
              <div
                onClick={() => onStatusClick(config.key)}
                style={{
                  cursor: 'pointer',
                  padding: '16px',
                  borderRadius: '8px',
                  border: `2px solid ${active ? config.color : config.borderColor}`,
                  backgroundColor: active ? config.bgColor : '#ffffff',
                  transition: 'all 0.3s ease',
                  textAlign: 'center',
                  opacity: count === 0 ? 0.6 : 1,
                }}
              >
                <div
                  style={{
                    fontSize: '24px',
                    fontWeight: 'bold',
                    color: config.color,
                    marginBottom: '4px',
                  }}
                >
                  {count}
                </div>
                <div
                  style={{
                    fontSize: '14px',
                    color: active ? config.color : '#595959',
                  }}
                >
                  <Badge color={config.color} text={config.label} />
                </div>
              </div>
            </Col>
          );
        })}
      </Row>
    </Card>
  );
};
