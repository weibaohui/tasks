import React from 'react';
import { Card, Col, Row, Statistic } from 'antd';
import { AppstoreOutlined, BranchesOutlined, ClockCircleOutlined, HeartOutlined } from '@ant-design/icons';
import type { Project } from '../../types/projectRequirement';
import type { Agent } from '../../types/agent';

interface OverviewStatsProps {
  projects: Project[];
  agents: Agent[];
  scenarioTemplateCount: number;
  onOpenScenarioTemplates: () => void;
}

/**
 * OverviewStats 渲染自动化中心总览统计卡片。
 */
export const OverviewStats: React.FC<OverviewStatsProps> = ({ projects, agents, scenarioTemplateCount, onOpenScenarioTemplates }) => {
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
        <Card
          size="small"
          hoverable
          onClick={onOpenScenarioTemplates}
          style={{ cursor: 'pointer' }}
        >
          <Statistic title="场景模板数" value={scenarioTemplateCount} prefix={<AppstoreOutlined />} />
        </Card>
      </Col>
    </Row>
  );
};
