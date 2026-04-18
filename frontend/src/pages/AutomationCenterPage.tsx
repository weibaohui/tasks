import React, { useEffect, useMemo, useState } from 'react';
import { Alert, Card, Drawer, Empty, Space, Tabs, Typography } from 'antd';
import { listProjects } from '../api/projectRequirementApi';
import { listAgents } from '../api/agentApi';
import { requirementTypeApi, type RequirementType } from '../api/requirementTypeApi';
import { listHeartbeatScenarios } from '../api/heartbeatScenarioApi';
import { useAuthStore } from '../stores/authStore';
import type { Project } from '../types/projectRequirement';
import type { Agent } from '../types/agent';
import { HeartbeatManagement } from '../components/HeartbeatManagement';
import { OverviewStats, ProjectRunsPanel, ProjectSelectorCard, ScenarioApplyPanel } from '../components/AutomationCenter';
import { HeartbeatScenarioManagementPage } from './HeartbeatScenarioManagementPage';
import { ProjectWebhookPage } from './ProjectWebhookPage';

const { Text } = Typography;

/**
 * AutomationCenterPage 作为自动化统一入口，聚合心跳实例、场景与 Webhook 管理能力。
 */
export const AutomationCenterPage: React.FC = () => {
  const { user } = useAuthStore();
  const [projects, setProjects] = useState<Project[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [selectedProjectId, setSelectedProjectId] = useState<string>('');
  const [requirementTypes, setRequirementTypes] = useState<RequirementType[]>([]);
  const [activeTab, setActiveTab] = useState<string>('overview');
  const [scenarioTemplateCount, setScenarioTemplateCount] = useState<number>(0);
  const [scenarioDrawerOpen, setScenarioDrawerOpen] = useState(false);

  /**
   * fetchBaseData 加载自动化中心基础依赖数据。
   */
  const fetchBaseData = async () => {
    const projectList = await listProjects();
    setProjects(projectList);
    if (!selectedProjectId && projectList.length > 0) {
      setSelectedProjectId(projectList[0].id);
    }
    if (user?.user_code) {
      const agentList = await listAgents(user.user_code);
      setAgents(agentList.filter((agent) => ['CodingAgent', 'OpenCodeAgent'].includes(agent.agent_type)));
    }
  };

  useEffect(() => {
    void fetchBaseData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.user_code]);

  /**
   * fetchRequirementTypes 加载当前项目可用需求类型。
   */
  const fetchRequirementTypes = async (projectId: string) => {
    if (!projectId) {
      setRequirementTypes([]);
      return;
    }
    try {
      const data = await requirementTypeApi.list(projectId);
      setRequirementTypes(data);
    } catch {
      setRequirementTypes([]);
    }
  };

  useEffect(() => {
    void fetchRequirementTypes(selectedProjectId);
  }, [selectedProjectId]);

  /**
   * fetchScenarioTemplateCount 拉取全局场景模板数量。
   */
  const fetchScenarioTemplateCount = async () => {
    try {
      const scenarios = await listHeartbeatScenarios();
      setScenarioTemplateCount(scenarios.length);
    } catch {
      setScenarioTemplateCount(0);
    }
  };

  useEffect(() => {
    void fetchScenarioTemplateCount();
  }, []);

  const selectedProject = useMemo(
    () => projects.find((project) => project.id === selectedProjectId) || null,
    [projects, selectedProjectId],
  );

  /**
   * handleProjectScenarioChanged 更新项目缓存中的场景编码，保持界面状态一致。
   */
  const handleProjectScenarioChanged = (scenarioCode: string) => {
    if (!selectedProjectId) {
      return;
    }
    setProjects((prev) =>
      prev.map((project) =>
        project.id === selectedProjectId
          ? { ...project, heartbeat_scenario_code: scenarioCode }
          : project,
      ),
    );
  };

  return (
    <div style={{ padding: 0 }}>
      <Card
        title="自动化中心"
        extra={
          <Space>
            <Text type="secondary">统一管理心跳实例、场景模板与 Webhook 事件触发</Text>
          </Space>
        }
      >
        <OverviewStats
          projects={projects}
          agents={agents}
          scenarioTemplateCount={scenarioTemplateCount}
          onOpenScenarioTemplates={() => setScenarioDrawerOpen(true)}
        />
        <ProjectSelectorCard projects={projects} selectedProjectId={selectedProjectId} onChange={setSelectedProjectId} />

        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'overview',
              label: '总览',
              children: (
                <Space direction="vertical" style={{ width: '100%' }}>
                  <ScenarioApplyPanel project={selectedProject} onProjectScenarioChanged={handleProjectScenarioChanged} />
                  <ProjectRunsPanel project={selectedProject} />
                </Space>
              ),
            },
            {
              key: 'heartbeats',
              label: '心跳实例',
              children: selectedProjectId ? (
                <HeartbeatManagement projectId={selectedProjectId} agents={agents} requirementTypes={requirementTypes} />
              ) : (
                <Empty description="请先选择项目后再管理心跳实例" />
              ),
            },
            {
              key: 'webhooks',
              label: 'Webhook 事件',
              children: <ProjectWebhookPage selectedProject={selectedProject} />,
            },
          ]}
        />
        <Drawer
          title="全局场景模板管理"
          width="86vw"
          destroyOnClose
          open={scenarioDrawerOpen}
          onClose={() => {
            setScenarioDrawerOpen(false);
            void fetchScenarioTemplateCount();
          }}
        >
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 12 }}
            message="场景模板是全局资产"
            description="模板与项目解耦。模板维护在这里完成，项目绑定请在“总览”页的“项目场景应用”中操作。"
          />
          <HeartbeatScenarioManagementPage />
        </Drawer>
      </Card>
    </div>
  );
};
