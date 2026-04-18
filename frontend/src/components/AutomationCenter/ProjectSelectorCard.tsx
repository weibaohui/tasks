import React from 'react';
import { Card, Select, Space, Typography } from 'antd';
import type { Project } from '../../types/projectRequirement';

const { Text } = Typography;

interface ProjectSelectorCardProps {
  projects: Project[];
  selectedProjectId: string;
  onChange: (projectId: string) => void;
}

/**
 * ProjectSelectorCard 渲染项目切换卡片和当前场景摘要。
 */
export const ProjectSelectorCard: React.FC<ProjectSelectorCardProps> = ({
  projects,
  selectedProjectId,
  onChange,
}) => {
  const selectedProject = projects.find((project) => project.id === selectedProjectId);

  return (
    <Card size="small" style={{ marginBottom: 16 }}>
      <Space style={{ width: '100%', justifyContent: 'space-between' }} wrap>
        <Space>
          <Text strong>当前项目</Text>
          <Select
            style={{ minWidth: 280 }}
            placeholder="请选择项目"
            value={selectedProjectId || undefined}
            onChange={onChange}
            options={projects.map((project) => ({
              label: project.name,
              value: project.id,
            }))}
          />
        </Space>
        <Text type="secondary">
          当前场景：
          {selectedProject?.heartbeat_scenario_code || '未设置'}
        </Text>
      </Space>
    </Card>
  );
};
