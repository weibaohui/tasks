import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Card, Form, Input, Modal, Popconfirm, Select, Space, Table, Tabs, Tag, message } from 'antd';
import { createProject, createRequirement, deleteProject, dispatchRequirement, listProjects, listRequirements, reportRequirementPROpened, updateProject, updateRequirement } from '../api/projectRequirementApi';
import { listAgents } from '../api/agentApi';
import { listChannels } from '../api/channelApi';
import { useAuthStore } from '../stores/authStore';
import type { Agent } from '../types/agent';
import type { Channel } from '../types/channel';
import type { CreateProjectRequest, CreateRequirementRequest, Project, Requirement } from '../types/projectRequirement';

const splitLines = (input: string): string[] => input.split('\n').map((item) => item.trim()).filter((item) => item !== '');

const joinLines = (items: string[]): string => items.join('\n');

const statusColorMap: Record<string, string> = {
  todo: 'default',
  in_progress: 'processing',
  done: 'success',
};

const devStateColorMap: Record<string, string> = {
  idle: 'default',
  preparing: 'gold',
  coding: 'blue',
  pr_opened: 'green',
  failed: 'red',
};

export const ProjectRequirementPage: React.FC = () => {
  const { user } = useAuthStore();
  const [projects, setProjects] = useState<Project[]>([]);
  const [requirements, setRequirements] = useState<Requirement[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loadingProjects, setLoadingProjects] = useState(false);
  const [loadingRequirements, setLoadingRequirements] = useState(false);
  const [selectedProjectId, setSelectedProjectId] = useState<string>('');
  const [activeTabKey, setActiveTabKey] = useState<string>('projects');
  const [projectModalOpen, setProjectModalOpen] = useState(false);
  const [editingProject, setEditingProject] = useState<Project | null>(null);
  const [requirementModalOpen, setRequirementModalOpen] = useState(false);
  const [editingRequirement, setEditingRequirement] = useState<Requirement | null>(null);
  const [dispatchModalOpen, setDispatchModalOpen] = useState(false);
  const [dispatchRequirementItem, setDispatchRequirementItem] = useState<Requirement | null>(null);
  const [prModalOpen, setPrModalOpen] = useState(false);
  const [prRequirementItem, setPrRequirementItem] = useState<Requirement | null>(null);
  const [projectForm] = Form.useForm();
  const [requirementForm] = Form.useForm();
  const [dispatchForm] = Form.useForm();
  const [prForm] = Form.useForm();

  const projectOptions = useMemo(
    () => projects.map((project) => ({ label: project.name, value: project.id })),
    [projects],
  );

  const fetchProjects = useCallback(async () => {
    setLoadingProjects(true);
    try {
      const data = await listProjects();
      setProjects(data);
      if (!selectedProjectId && data.length > 0) {
        setSelectedProjectId(data[0].id);
      }
    } catch (_error) {
      message.error('获取项目列表失败');
    } finally {
      setLoadingProjects(false);
    }
  }, [selectedProjectId]);

  const fetchRequirements = useCallback(async (projectId?: string) => {
    setLoadingRequirements(true);
    try {
      const data = await listRequirements(projectId || selectedProjectId || undefined);
      setRequirements(data);
    } catch (_error) {
      message.error('获取需求列表失败');
    } finally {
      setLoadingRequirements(false);
    }
  }, [selectedProjectId]);

  const fetchAgents = useCallback(async () => {
    if (!user?.user_code) {
      return;
    }
    try {
      const data = await listAgents(user.user_code);
      setAgents(data.filter((agent) => agent.agent_type === 'CodingAgent'));
    } catch (_error) {
      message.error('获取 Agent 列表失败');
    }
  }, [user?.user_code]);

  const fetchChannels = useCallback(async () => {
    if (!user?.user_code) {
      return;
    }
    try {
      const data = await listChannels(user.user_code);
      setChannels(data.filter((channel) => channel.is_active));
    } catch (_error) {
      message.error('获取渠道列表失败');
    }
  }, [user?.user_code]);

  useEffect(() => {
    fetchProjects();
    fetchAgents();
    fetchChannels();
  }, [fetchAgents, fetchChannels, fetchProjects]);

  useEffect(() => {
    if (selectedProjectId) {
      fetchRequirements(selectedProjectId);
    } else {
      setRequirements([]);
    }
  }, [fetchRequirements, selectedProjectId]);

  const openCreateProject = () => {
    setEditingProject(null);
    projectForm.resetFields();
    projectForm.setFieldsValue({ default_branch: 'main' });
    setProjectModalOpen(true);
  };

  const openEditProject = (project: Project) => {
    setEditingProject(project);
    projectForm.setFieldsValue({
      name: project.name,
      git_repo_url: project.git_repo_url,
      default_branch: project.default_branch,
      init_steps_text: joinLines(project.init_steps || []),
    });
    setProjectModalOpen(true);
  };

  const submitProject = async (values: { name: string; git_repo_url: string; default_branch: string; init_steps_text: string }) => {
    const payload: CreateProjectRequest = {
      name: values.name,
      git_repo_url: values.git_repo_url,
      default_branch: values.default_branch,
      init_steps: splitLines(values.init_steps_text || ''),
    };
    try {
      if (editingProject) {
        await updateProject({ ...payload, id: editingProject.id });
        message.success('更新项目成功');
      } else {
        await createProject(payload);
        message.success('创建项目成功');
      }
      setProjectModalOpen(false);
      await fetchProjects();
      if (selectedProjectId) {
        await fetchRequirements(selectedProjectId);
      }
    } catch (_error) {
      message.error(editingProject ? '更新项目失败' : '创建项目失败');
    }
  };

  const submitRequirement = async (values: { project_id: string; title: string; description: string; acceptance_criteria: string }) => {
    const payload: CreateRequirementRequest = {
      project_id: values.project_id,
      title: values.title,
      description: values.description || '',
      acceptance_criteria: values.acceptance_criteria || '',
    };
    try {
      if (editingRequirement) {
        await updateRequirement({
          id: editingRequirement.id,
          title: payload.title,
          description: payload.description,
          acceptance_criteria: payload.acceptance_criteria,
        });
        message.success('更新需求成功');
      } else {
        await createRequirement(payload);
        message.success('创建需求成功');
      }
      setRequirementModalOpen(false);
      await fetchRequirements(selectedProjectId);
    } catch (_error) {
      message.error(editingRequirement ? '更新需求失败' : '创建需求失败');
    }
  };

  const handleDeleteProject = async (id: string) => {
    try {
      await deleteProject(id);
      message.success('删除项目成功');
      if (selectedProjectId === id) {
        setSelectedProjectId('');
      }
      await fetchProjects();
    } catch (_error) {
      message.error('删除项目失败');
    }
  };

  const openCreateRequirement = () => {
    setEditingRequirement(null);
    requirementForm.resetFields();
    requirementForm.setFieldsValue({ project_id: selectedProjectId });
    setRequirementModalOpen(true);
  };

  const handleViewRequirements = async (projectId: string) => {
    setSelectedProjectId(projectId);
    setActiveTabKey('requirements');
    await fetchRequirements(projectId);
  };

  const openEditRequirement = (item: Requirement) => {
    setEditingRequirement(item);
    requirementForm.setFieldsValue({
      project_id: item.project_id,
      title: item.title,
      description: item.description,
      acceptance_criteria: item.acceptance_criteria,
    });
    setRequirementModalOpen(true);
  };

  const openDispatchModal = (item: Requirement) => {
    setDispatchRequirementItem(item);
    dispatchForm.resetFields();
    const defaultChannelCode = channels[0]?.channel_code;
    if (defaultChannelCode) {
      dispatchForm.setFieldsValue({ channel_code: defaultChannelCode });
    }
    setDispatchModalOpen(true);
  };

  const submitDispatch = async (values: { agent_id: string; channel_code: string; session_key: string }) => {
    if (!dispatchRequirementItem) {
      return;
    }
    try {
      const sessionKey = values.session_key.trim();
      const result = await dispatchRequirement(dispatchRequirementItem.id, values.agent_id, values.channel_code, sessionKey);
      message.success(`派发成功，任务ID: ${result.task_id}`);
      setDispatchModalOpen(false);
      await fetchRequirements(selectedProjectId);
    } catch (_error) {
      message.error('派发需求失败');
    }
  };

  const openReportPRModal = (item: Requirement) => {
    setPrRequirementItem(item);
    prForm.setFieldsValue({
      pr_url: item.pr_url || '',
      branch_name: item.branch_name || '',
    });
    setPrModalOpen(true);
  };

  const submitReportPR = async (values: { pr_url: string; branch_name: string }) => {
    if (!prRequirementItem) {
      return;
    }
    try {
      await reportRequirementPROpened(prRequirementItem.id, values.pr_url, values.branch_name || '');
      message.success('PR 状态更新成功');
      setPrModalOpen(false);
      await fetchRequirements(selectedProjectId);
    } catch (_error) {
      message.error('PR 状态更新失败');
    }
  };

  const projectColumns = [
    { title: '项目名称', dataIndex: 'name', key: 'name' },
    { title: '仓库地址', dataIndex: 'git_repo_url', key: 'git_repo_url' },
    { title: '默认分支', dataIndex: 'default_branch', key: 'default_branch' },
    {
      title: '初始化步骤',
      key: 'init_steps',
      render: (_: unknown, project: Project) => (project.init_steps || []).length,
    },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, project: Project) => (
        <Space>
          <Button type="link" onClick={() => handleViewRequirements(project.id)}>
            查看需求
          </Button>
          <Button type="link" onClick={() => openEditProject(project)}>
            编辑
          </Button>
          <Popconfirm title="确认删除该项目？" onConfirm={() => handleDeleteProject(project.id)}>
            <Button type="link" danger>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const requirementColumns = [
    { title: '标题', dataIndex: 'title', key: 'title' },
    {
      title: '状态',
      key: 'status',
      render: (_: unknown, item: Requirement) => (
        <Space>
          <Tag color={statusColorMap[item.status] || 'default'}>{item.status}</Tag>
          <Tag color={devStateColorMap[item.dev_state] || 'default'}>{item.dev_state}</Tag>
        </Space>
      ),
    },
    { title: '分支', dataIndex: 'branch_name', key: 'branch_name' },
    { title: 'PR', dataIndex: 'pr_url', key: 'pr_url', ellipsis: true },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, item: Requirement) => (
        <Space>
          <Button type="link" onClick={() => openEditRequirement(item)}>
            编辑
          </Button>
          <Button type="link" disabled={!(item.status === 'todo' && item.dev_state === 'idle')} onClick={() => openDispatchModal(item)}>
            派发
          </Button>
          <Button type="link" onClick={() => openReportPRModal(item)}>
            更新PR
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 0 }}>
      <Tabs
        activeKey={activeTabKey}
        onChange={(key) => setActiveTabKey(key)}
        items={[
          {
            key: 'projects',
            label: '项目管理',
            children: (
              <Card
                title={`项目列表 (${projects.length})`}
                extra={
                  <Space>
                    <Button onClick={() => fetchProjects()}>刷新</Button>
                    <Button type="primary" onClick={openCreateProject}>
                      新建项目
                    </Button>
                  </Space>
                }
              >
                <Table<Project> rowKey="id" loading={loadingProjects} dataSource={projects} columns={projectColumns} />
              </Card>
            ),
          },
          {
            key: 'requirements',
            label: '需求管理',
            children: (
              <Card
                title={`需求列表 (${requirements.length})`}
                extra={
                  <Space>
                    <Select
                      style={{ width: 280 }}
                      placeholder="选择项目"
                      value={selectedProjectId || undefined}
                      options={projectOptions}
                      onChange={(value) => setSelectedProjectId(value)}
                    />
                    <Button onClick={() => fetchRequirements(selectedProjectId)}>刷新</Button>
                    <Button type="primary" disabled={!selectedProjectId} onClick={openCreateRequirement}>
                      新建需求
                    </Button>
                  </Space>
                }
              >
                <Table<Requirement> rowKey="id" loading={loadingRequirements} dataSource={requirements} columns={requirementColumns} />
              </Card>
            ),
          },
        ]}
      />

      <Modal title={editingProject ? '编辑项目' : '新建项目'} open={projectModalOpen} footer={null} onCancel={() => setProjectModalOpen(false)}>
        <Form layout="vertical" form={projectForm} onFinish={submitProject}>
          <Form.Item label="项目名称" name="name" rules={[{ required: true, message: '请输入项目名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item label="Git 仓库地址" name="git_repo_url" rules={[{ required: true, message: '请输入仓库地址' }]}>
            <Input />
          </Form.Item>
          <Form.Item label="默认分支" name="default_branch" rules={[{ required: true, message: '请输入默认分支' }]}>
            <Input />
          </Form.Item>
          <Form.Item label="初始化步骤（每行一个）" name="init_steps_text">
            <Input.TextArea rows={5} />
          </Form.Item>
          <Button type="primary" htmlType="submit" block>
            保存
          </Button>
        </Form>
      </Modal>

      <Modal title={editingRequirement ? '编辑需求' : '新建需求'} open={requirementModalOpen} footer={null} onCancel={() => setRequirementModalOpen(false)}>
        <Form layout="vertical" form={requirementForm} onFinish={submitRequirement}>
          <Form.Item label="所属项目" name="project_id" rules={[{ required: true, message: '请选择所属项目' }]}>
            <Select options={projectOptions} />
          </Form.Item>
          <Form.Item label="需求标题" name="title" rules={[{ required: true, message: '请输入需求标题' }]}>
            <Input />
          </Form.Item>
          <Form.Item label="需求描述" name="description">
            <Input.TextArea rows={4} />
          </Form.Item>
          <Form.Item label="验收标准" name="acceptance_criteria">
            <Input.TextArea rows={4} />
          </Form.Item>
          <Button type="primary" htmlType="submit" block>
            保存
          </Button>
        </Form>
      </Modal>

      <Modal title="派发需求" open={dispatchModalOpen} footer={null} onCancel={() => setDispatchModalOpen(false)}>
        <Form layout="vertical" form={dispatchForm} onFinish={submitDispatch}>
          <Form.Item label="执行 Agent" name="agent_id" rules={[{ required: true, message: '请选择执行 Agent' }]}>
            <Select options={agents.map((agent) => ({ label: `${agent.name} (${agent.agent_code})`, value: agent.id }))} />
          </Form.Item>
          <Form.Item label="派发渠道" name="channel_code" rules={[{ required: true, message: '请选择渠道' }]}>
            <Select
              options={channels.map((channel) => ({
                label: `${channel.name} (${channel.type})`,
                value: channel.channel_code,
              }))}
              onChange={(channelCode) => {
                const selectedChannel = channels.find((channel) => channel.channel_code === channelCode);
                if (!selectedChannel) {
                  return;
                }
                const currentSessionKey = dispatchForm.getFieldValue('session_key') as string | undefined;
                if (!currentSessionKey || !currentSessionKey.includes(':')) {
                  dispatchForm.setFieldValue('session_key', `${selectedChannel.type}:`);
                }
              }}
            />
          </Form.Item>
          <Form.Item
            label="SessionKey"
            name="session_key"
            rules={[
              { required: true, message: '请输入 SessionKey，例如 feishu:ou_xxx' },
              {
                validator: async (_, value: string) => {
                  const channelCode = dispatchForm.getFieldValue('channel_code') as string | undefined;
                  const selectedChannel = channels.find((channel) => channel.channel_code === channelCode);
                  if (!value || !selectedChannel) {
                    return;
                  }
                  if (!value.startsWith(`${selectedChannel.type}:`)) {
                    throw new Error(`SessionKey 需要以 ${selectedChannel.type}: 开头`);
                  }
                },
              },
            ]}
          >
            <Input placeholder="例如：feishu:ou_df798fe15d056000143691af8c1cdb55" />
          </Form.Item>
          <Button type="primary" htmlType="submit" block>
            确认派发
          </Button>
        </Form>
      </Modal>

      <Modal title="更新 PR 状态" open={prModalOpen} footer={null} onCancel={() => setPrModalOpen(false)}>
        <Form layout="vertical" form={prForm} onFinish={submitReportPR}>
          <Form.Item label="PR 链接" name="pr_url" rules={[{ required: true, message: '请输入 PR 链接' }]}>
            <Input />
          </Form.Item>
          <Form.Item label="分支名" name="branch_name">
            <Input />
          </Form.Item>
          <Button type="primary" htmlType="submit" block>
            更新状态
          </Button>
        </Form>
      </Modal>
    </div>
  );
};
