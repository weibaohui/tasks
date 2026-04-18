import React from 'react';
import { Card, Col, Row, Statistic } from 'antd';
import { BranchesOutlined, ClockCircleOutlined, HeartOutlined, LinkOutlined } from '@ant-design/icons';
import type { Project } from '../../types/projectRequirement';
import type { Agent } from '../../types/agent';

interface OverviewStatsProps {
  projects: Project[];
  agents: Agent[];
}

/**
 * OverviewStats 渲染自动化中心总览统计卡片。
 */
export const OverviewStats: React.FC<OverviewStatsProps> = ({ projects, agents }) => {
  return (
    <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
      <Col xs={24} sm={12} md={6}>
        <Card size="small">
          <Statistic title="项目数量" value={projects.length} prefix={<BranchesOutlined />} />
        </Card>
      </Col>
      <Col xs={24} sm={12} md={6}>
        <Card size="small">
          <Statistic title="可用 Agent" value={agents.length} prefix={<HeartOutlined />} />
        </Card>
      </Col>
      <Col xs={24} sm={12} md={6}>
        <Card size="small">
          <Statistic
            title="已配置场景项目"
            value={projects.filter((project) => !!project.heartbeat_scenario_code).length}
            prefix={<ClockCircleOutlined />}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} md={6}>
        <Card size="small">
          <Statistic title="Webhook 入口" value="已统一" prefix={<LinkOutlined />} />
        </Card>
      </Col>
    </Row>
  );
};
