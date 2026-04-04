/**
 * State Machine Management Page
 */
import React, { useEffect, useState, useCallback } from 'react';
import { Card, Typography, Space, Button, Select, message } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useStateMachineManagement } from './hooks';
import { StateMachineTable } from './components/StateMachineTable';
import { StateMachineEditDrawer } from './components/StateMachineEditDrawer';
import { StateMachineDetailDrawer } from './components/StateMachineDetailDrawer';
import type { StateMachine, CreateStateMachineRequest } from '../../types/stateMachine';
import { listProjects } from '../../api/projectRequirementApi';
import type { Project } from '../../types/projectRequirement';
import { listRequirements } from '../../api/projectRequirementApi';
import type { Requirement } from '../../types/projectRequirement';

const { Title } = Typography;

export const StateMachineManagementPage: React.FC = () => {
  const {
    items,
    loading,
    saving,
    open,
    editing,
    projectId,
    setProjectId,
    fetchList,
    openEditor,
    closeEditor,
    handleDelete,
    handleSubmit,
  } = useStateMachineManagement();

  const [projects, setProjects] = useState<Project[]>([]);
  const [requirements, setRequirements] = useState<Requirement[]>([]);
  const [loadingProjects, setLoadingProjects] = useState(false);
  const [detailDrawerOpen, setDetailDrawerOpen] = useState(false);
  const [selectedStateMachine, setSelectedStateMachine] = useState<StateMachine | null>(null);

  const fetchProjects = useCallback(async () => {
    setLoadingProjects(true);
    try {
      const data = await listProjects();
      setProjects(data);
      if (data.length > 0 && !projectId) {
        setProjectId(data[0].id);
      }
    } catch (err) {
      message.error('获取项目列表失败');
    } finally {
      setLoadingProjects(false);
    }
  }, [setProjectId, projectId]);

  const fetchRequirements = useCallback(async (pid: string) => {
    try {
      const data = await listRequirements(pid);
      setRequirements(data);
    } catch (err) {
      setRequirements([]);
    }
  }, []);

  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  useEffect(() => {
    if (projectId) {
      fetchList(projectId);
      fetchRequirements(projectId);
    }
  }, [projectId, fetchList, fetchRequirements]);

  const handleViewDetail = (record: StateMachine) => {
    setSelectedStateMachine(record);
    setDetailDrawerOpen(true);
  };

  const handleDetailClose = () => {
    setDetailDrawerOpen(false);
    setSelectedStateMachine(null);
  };

  const handleRefreshRequirements = () => {
    if (projectId) {
      fetchRequirements(projectId);
    }
  };

  const projectOptions = projects.map((p) => ({ label: p.name, value: p.id }));

  return (
    <div style={{ padding: 0 }}>
      <Card
        title={<Title level={3} style={{ margin: 0 }}>状态机管理</Title>}
        extra={
          <Space>
            <Select
              style={{ width: 200 }}
              placeholder="选择项目"
              value={projectId || undefined}
              options={projectOptions}
              onChange={(value) => setProjectId(value)}
              loading={loadingProjects}
            />
            <Button onClick={() => fetchList(projectId)}>刷新</Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => openEditor(null)}
              disabled={!projectId}
            >
              新建状态机
            </Button>
          </Space>
        }
      >
        {!projectId ? (
          <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>
            请先选择一个项目
          </div>
        ) : (
          <StateMachineTable
            items={items}
            loading={loading}
            onEdit={openEditor}
            onDelete={handleDelete}
            onViewDetail={handleViewDetail}
          />
        )}
      </Card>

      {/* 编辑抽屉 */}
      <StateMachineEditDrawer
        open={open}
        editing={editing}
        saving={saving}
        onClose={closeEditor}
        onSubmit={handleSubmit as (values: CreateStateMachineRequest) => Promise<void>}
      />

      {/* 详情抽屉 */}
      <StateMachineDetailDrawer
        stateMachine={selectedStateMachine}
        open={detailDrawerOpen}
        onClose={handleDetailClose}
        requirements={requirements.map((r) => ({
          id: r.id,
          title: r.title,
          requirement_type: r.requirement_type || 'normal',
        }))}
        onRefreshRequirements={handleRefreshRequirements}
      />
    </div>
  );
};

export default StateMachineManagementPage;