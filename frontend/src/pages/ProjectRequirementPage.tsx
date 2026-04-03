import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Card, Drawer, Form, Input, Modal, Popconfirm, Select, Space, Table, Tabs, Tag, Switch, message, Alert, Tooltip } from 'antd';
import { copyAndDispatchRequirement, createProject, createRequirement, deleteProject, dispatchRequirement, listProjects, listRequirements, updateProject, updateRequirement } from '../api/projectRequirementApi';
import { listAgents } from '../api/agentApi';
import { listChannels } from '../api/channelApi';
import { listHookConfigs, createHookConfig, updateHookConfig, deleteHookConfig, enableHookConfig, disableHookConfig } from '../api/hookApi';
import { useAuthStore } from '../stores/authStore';
import type { Agent } from '../types/agent';
import type { Channel } from '../types/channel';
import type { CreateProjectRequest, CreateRequirementRequest, Project, Requirement } from '../types/projectRequirement';
import type { HookConfig, CreateHookConfigRequest, UpdateHookConfigRequest } from '../types/hook';
import { TRIGGER_POINTS, ACTION_TYPES } from '../types/hook';
import { HeartbeatTemplateEditor } from '../components/HeartbeatTemplate';
import { TraceViewer } from '../components/TraceViewer';

const splitLines = (input: string): string[] => input.split('\n').map((item) => item.trim()).filter((item) => item !== '');

const joinLines = (items: string[]): string => items.join('\n');

const statusColorMap: Record<string, string> = {
  todo: 'default',
  preparing: 'gold',
  coding: 'blue',
  pr_opened: 'green',
  failed: 'red',
  completed: 'success',
  done: 'success',
};

const claudeRuntimeColorMap: Record<string, string> = {
  running: 'processing',
  completed: 'success',
  failed: 'error',
};

const defaultDispatchSessionKey = 'feishu:ou_df798fe15d056000143691af8c1cdb55';

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
  const [projectForm] = Form.useForm();
  const [requirementForm] = Form.useForm();
  const [dispatchForm] = Form.useForm();

  // 项目配置抽屉状态
  const [projectConfigDrawerOpen, setProjectConfigDrawerOpen] = useState(false);
  const [configProject, setConfigProject] = useState<Project | null>(null);
  const [projectHooks, setProjectHooks] = useState<HookConfig[]>([]);
  const [loadingProjectHooks, setLoadingProjectHooks] = useState(false);
  const [hookModalOpen, setHookModalOpen] = useState(false);
  const [editingHook, setEditingHook] = useState<HookConfig | null>(null);
  const [hookForm] = Form.useForm();
  const [savingHook, setSavingHook] = useState(false);

  // 心跳配置相关状态
  const [heartbeatForm] = Form.useForm();
  const [savingHeartbeat, setSavingHeartbeat] = useState(false);

  // 需求详情抽屉状态
  const [requirementDetailDrawerOpen, setRequirementDetailDrawerOpen] = useState(false);

  // Trace Viewer 状态
  const [traceViewerVisible, setTraceViewerVisible] = useState(false);
  const [currentTraceId, setCurrentTraceId] = useState<string>('');
  const [detailRequirement, setDetailRequirement] = useState<Requirement | null>(null);

  // 需求状态过滤
  const [statusFilter, setStatusFilter] = useState<string>('');

  // 需求类型过滤
  const [typeFilter, setTypeFilter] = useState<string>('');

  // 根据状态和类型过滤后的需求列表
  const filteredRequirements = useMemo(() => {
    return requirements.filter((req) => {
      const matchStatus = !statusFilter || req.status === statusFilter;
      const matchType = !typeFilter || req.requirement_type === typeFilter || (typeFilter === 'normal' && !req.requirement_type);
      return matchStatus && matchType;
    });
  }, [requirements, statusFilter, typeFilter]);

  const projectOptions = useMemo(
    () => projects.map((project) => ({ label: project.name, value: project.id })),
    [projects],
  );

  const statusOptions = [
    { label: '全部状态', value: '' },
    { label: '待处理 (todo)', value: 'todo' },
    { label: '准备中 (preparing)', value: 'preparing' },
    { label: '编码中 (coding)', value: 'coding' },
    { label: 'PR已开 (pr_opened)', value: 'pr_opened' },
    { label: '失败 (failed)', value: 'failed' },
    { label: '已完成 (completed)', value: 'completed' },
    { label: '完成 (done)', value: 'done' },
  ];

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
      agent_code: project.agent_code || '',
      dispatch_channel_code: project.dispatch_channel_code || '',
      dispatch_session_key: project.dispatch_session_key || '',
    });
    setProjectModalOpen(true);
  };

  const submitProject = async (values: { name: string; git_repo_url: string; default_branch: string; init_steps_text: string; agent_code?: string; dispatch_channel_code?: string; dispatch_session_key?: string }) => {
    const payload: CreateProjectRequest = {
      name: values.name,
      git_repo_url: values.git_repo_url,
      default_branch: values.default_branch,
      init_steps: splitLines(values.init_steps_text || ''),
    };
    try {
      if (editingProject) {
        await updateProject({
          ...payload,
          id: editingProject.id,
          heartbeat_enabled: editingProject.heartbeat_enabled || false,
          heartbeat_interval_minutes: editingProject.heartbeat_interval_minutes || 60,
          heartbeat_md_content: editingProject.heartbeat_md_content || '',
          agent_code: values.agent_code || '',
          dispatch_channel_code: values.dispatch_channel_code || '',
          dispatch_session_key: values.dispatch_session_key || '',
        });
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

  const submitRequirement = async (values: { project_id: string; title: string; description: string; acceptance_criteria: string; temp_workspace_root: string }) => {
    const payload: CreateRequirementRequest = {
      project_id: values.project_id,
      title: values.title,
      description: values.description || '',
      acceptance_criteria: values.acceptance_criteria || '',
      temp_workspace_root: values.temp_workspace_root || '',
    };
    try {
      if (editingRequirement) {
        await updateRequirement({
          id: editingRequirement.id,
          title: payload.title,
          description: payload.description,
          acceptance_criteria: payload.acceptance_criteria,
          temp_workspace_root: payload.temp_workspace_root,
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
    requirementForm.setFieldsValue({ project_id: selectedProjectId, temp_workspace_root: '/tmp/ai-devops' });
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
      temp_workspace_root: item.temp_workspace_root,
    });
    setRequirementModalOpen(true);
  };

  const openDispatchModal = (item: Requirement) => {
    setDispatchRequirementItem(item);
    dispatchForm.resetFields();

    // 获取项目配置的派发渠道和 session_key
    const project = projects.find((p) => p.id === item.project_id);
    const projectAgentCode = project?.agent_code;
    const projectChannelCode = project?.dispatch_channel_code;
    const projectSessionKey = project?.dispatch_session_key;

    if (projectAgentCode) {
      dispatchForm.setFieldsValue({ agent_code: projectAgentCode });
    }
    if (projectChannelCode && projectSessionKey) {
      dispatchForm.setFieldsValue({ channel_code: projectChannelCode, session_key: projectSessionKey });
    } else if (channels.length > 0) {
      dispatchForm.setFieldsValue({ channel_code: channels[0]?.channel_code, session_key: defaultDispatchSessionKey });
    }
    setDispatchModalOpen(true);
  };

  const submitDispatch = async (values: { agent_code: string; channel_code: string; session_key: string }) => {
    if (!dispatchRequirementItem) {
      return;
    }
    try {
      const sessionKey = values.session_key.trim();
      const result = await dispatchRequirement(dispatchRequirementItem.id, values.agent_code, values.channel_code, sessionKey);
      message.success(`派发成功，任务ID: ${result.task_id}`);
      setDispatchModalOpen(false);
      await fetchRequirements(selectedProjectId);
    } catch (_error) {
      message.error('派发需求失败');
    }
  };

  // 复制需求并派发新副本
  const handleCopyAndDispatch = async (item: Requirement) => {
    try {
      const newReq = await copyAndDispatchRequirement(item.id);
      message.success(`已创建新需求并派发: ${newReq.title}`);
      await fetchRequirements(selectedProjectId);
    } catch (_error) {
      message.error('复制并派发失败');
    }
  };

  const openRequirementDetail = (item: Requirement) => {
    setDetailRequirement(item);
    setRequirementDetailDrawerOpen(true);
  };

  // 项目配置相关处理
  const openProjectConfig = async (project: Project) => {
    setConfigProject(project);
    setProjectConfigDrawerOpen(true);
    setLoadingProjectHooks(true);

    // 设置心跳表单默认值
    heartbeatForm.setFieldsValue({
      heartbeat_enabled: project.heartbeat_enabled || false,
      heartbeat_interval_minutes: project.heartbeat_interval_minutes || 60,
      heartbeat_md_content: project.heartbeat_md_content || '',
      agent_code: project.agent_code || '',
    });

    try {
      const hooks = await listHookConfigs(project.id);
      setProjectHooks(hooks);
    } catch (_error) {
      message.error('获取项目 Hook 配置失败');
    } finally {
      setLoadingProjectHooks(false);
    }
  };

  const closeProjectConfig = () => {
    setProjectConfigDrawerOpen(false);
    setConfigProject(null);
    setProjectHooks([]);
  };

  const openCreateHook = () => {
    setEditingHook(null);
    hookForm.resetFields();
    hookForm.setFieldsValue({
      project_id: configProject?.id,
      trigger_point: 'start_dispatch',
      action_type: 'coding_agent',
      enabled: true,
      priority: 50,
    });
    setHookModalOpen(true);
  };

  const openEditHook = (hook: HookConfig) => {
    setEditingHook(hook);
    const formValues: Record<string, unknown> = {
      project_id: hook.project_id,
      name: hook.name,
      trigger_point: hook.trigger_point,
      action_type: hook.action_type,
      enabled: hook.enabled,
      priority: hook.priority,
    };

    // 解析 action_config JSON
    if (hook.action_type === 'coding_agent') {
      try {
        const config = JSON.parse(hook.action_config || '{}');
        formValues.prompt_template = config.prompt_template || '';
      } catch {
        formValues.prompt_template = '';
      }
    } else {
      formValues.action_config = hook.action_config;
    }

    hookForm.setFieldsValue(formValues);
    setHookModalOpen(true);
  };

  const submitHook = async (values: CreateHookConfigRequest & { prompt_template?: string }) => {
    setSavingHook(true);
    try {
      const submitValues = { ...values };

      // 如果是 coding_agent，将 prompt_template 放入 action_config
      if (values.action_type === 'coding_agent') {
        const actionConfig = {
          prompt_template: values.prompt_template || '',
        };
        submitValues.action_config = JSON.stringify(actionConfig);
        delete submitValues.prompt_template;
      }

      if (editingHook) {
        await updateHookConfig({ ...submitValues, id: editingHook.id } as UpdateHookConfigRequest);
        message.success('更新 Hook 成功');
      } else {
        await createHookConfig(submitValues as CreateHookConfigRequest);
        message.success('创建 Hook 成功');
      }
      setHookModalOpen(false);
      // 刷新项目 Hook 列表
      if (configProject) {
        const hooks = await listHookConfigs(configProject.id);
        setProjectHooks(hooks);
      }
    } catch (_error) {
      message.error(editingHook ? '更新 Hook 失败' : '创建 Hook 失败');
    } finally {
      setSavingHook(false);
    }
  };

  const handleDeleteHook = async (id: string) => {
    try {
      await deleteHookConfig(id);
      message.success('删除 Hook 成功');
      if (configProject) {
        const hooks = await listHookConfigs(configProject.id);
        setProjectHooks(hooks);
      }
    } catch (_error) {
      message.error('删除 Hook 失败');
    }
  };

  const handleToggleHookEnabled = async (hook: HookConfig) => {
    try {
      if (hook.enabled) {
        await disableHookConfig(hook.id);
      } else {
        await enableHookConfig(hook.id);
      }
      if (configProject) {
        const hooks = await listHookConfigs(configProject.id);
        setProjectHooks(hooks);
      }
    } catch (_error) {
      message.error('操作失败');
    }
  };

  // 心跳配置保存
  const handleSaveHeartbeat = async () => {
    if (!configProject) return;

    setSavingHeartbeat(true);
    try {
      const values = heartbeatForm.getFieldsValue(true);
      await updateProject({
        id: configProject.id,
        name: configProject.name,
        git_repo_url: configProject.git_repo_url,
        default_branch: configProject.default_branch,
        init_steps: configProject.init_steps || [],
        heartbeat_enabled: values.heartbeat_enabled || false,
        heartbeat_interval_minutes: values.heartbeat_interval_minutes || 60,
        heartbeat_md_content: values.heartbeat_md_content || '',
        agent_code: values.agent_code || '',
        dispatch_channel_code: values.dispatch_channel_code || '',
        dispatch_session_key: values.dispatch_session_key || '',
      });
      message.success('心跳配置已保存');
      await fetchProjects();
    } catch (_error) {
      message.error('保存心跳配置失败');
    } finally {
      setSavingHeartbeat(false);
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
      title: '心跳',
      key: 'heartbeat',
      width: 80,
      render: (_: unknown, project: Project) =>
        project.heartbeat_enabled ? (
          <Tag color="green">启用</Tag>
        ) : (
          <Tag color="default">未启用</Tag>
        ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, project: Project) => (
        <Space>
          <Button type="link" onClick={() => openProjectConfig(project)}>
            配置
          </Button>
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
      title: '类型',
      key: 'requirement_type',
      render: (_: unknown, item: Requirement) => (
        <Tag color={item.requirement_type === 'heartbeat' ? 'orange' : 'default'}>
          {item.requirement_type === 'heartbeat' ? '心跳' : '普通'}
        </Tag>
      ),
    },
    {
      title: '状态',
      key: 'status',
      render: (_: unknown, item: Requirement) => (
        <Space>
          <Tag color={statusColorMap[item.status] || 'default'}>{item.status}</Tag>
        </Space>
      ),
    },
    {
      title: 'Claude状态',
      key: 'claude_runtime',
      render: (_: unknown, item: Requirement) => {
        const runtimeStatus = item.claude_runtime?.status || '';
        if (!runtimeStatus) {
          return <span>-</span>;
        }
        // 根据 status 字段计算 is_running
        const isRunning = runtimeStatus === 'running';
        return (
          <Space>
            <Tag color={claudeRuntimeColorMap[runtimeStatus] || 'default'}>{runtimeStatus}</Tag>
            {isRunning ? <Tag color="processing">运行中</Tag> : <Tag>已停止</Tag>}
          </Space>
        );
      },
    },
    {
      title: 'Token消耗',
      key: 'tokens',
      render: (_: unknown, item: Requirement) => {
        const totalTokens = item.total_tokens || 0;
        if (totalTokens === 0) {
          return <span>-</span>;
        }
        return (
          <Tooltip title={`Prompt: ${item.prompt_tokens || 0}, Completion: ${item.completion_tokens || 0}`}>
            <Tag color="blue">{totalTokens.toLocaleString()}</Tag>
          </Tooltip>
        );
      },
    },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, item: Requirement) => (
        <Space>
          <Button type="link" onClick={() => openRequirementDetail(item)}>
            详情
          </Button>
          <Button type="link" onClick={() => openEditRequirement(item)}>
            编辑
          </Button>
          <Button type="link" disabled={item.status !== 'todo'} onClick={() => openDispatchModal(item)}>
            派发
          </Button>
          <Button type="link" disabled={item.status === 'todo'} onClick={() => handleCopyAndDispatch(item)}>
            复制并派发
          </Button>
          {item.trace_id && (
            <Button type="link" onClick={() => { setCurrentTraceId(item.trace_id!); setTraceViewerVisible(true); }}>
              对话链路
            </Button>
          )}
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
                title={`需求列表 (${filteredRequirements.length})`}
                extra={
                  <Space>
                    <Select
                      style={{ width: 280 }}
                      placeholder="选择项目"
                      value={selectedProjectId || undefined}
                      options={projectOptions}
                      onChange={(value) => setSelectedProjectId(value)}
                    />
                    <Select
                      style={{ width: 180 }}
                      placeholder="按状态过滤"
                      value={statusFilter || undefined}
                      options={statusOptions}
                      onChange={(value) => setStatusFilter(value || '')}
                      allowClear
                    />
                    <Select
                      style={{ width: 150 }}
                      placeholder="按类型过滤"
                      value={typeFilter || undefined}
                      options={[
                        { label: '全部类型', value: '' },
                        { label: '普通', value: 'normal' },
                        { label: '心跳', value: 'heartbeat' },
                      ]}
                      onChange={(value) => setTypeFilter(value || '')}
                      allowClear
                    />
                    <Button onClick={() => fetchRequirements(selectedProjectId)}>刷新</Button>
                    <Button type="primary" disabled={!selectedProjectId} onClick={openCreateRequirement}>
                      新建需求
                    </Button>
                  </Space>
                }
              >
                <Table<Requirement> rowKey="id" loading={loadingRequirements} dataSource={filteredRequirements} columns={requirementColumns} />
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
          <Form.Item label="默认执行 Agent" name="agent_code">
            <Select
              options={agents.filter((a) => a.agent_type === 'CodingAgent').map((a) => ({
                label: `${a.name} (${a.agent_code})`,
                value: a.agent_code,
              }))}
              placeholder="选择默认执行 Agent"
              allowClear
            />
          </Form.Item>
          <Form.Item label="默认派发渠道" name="dispatch_channel_code">
            <Select
              options={channels.map((c) => ({
                label: `${c.name} (${c.type})`,
                value: c.channel_code,
              }))}
              placeholder="选择默认派发渠道"
              allowClear
            />
          </Form.Item>
          <Form.Item label="默认 SessionKey" name="dispatch_session_key">
            <Input placeholder="例如：feishu:ou_xxx" />
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
          <Form.Item label="临时工作目录根路径" name="temp_workspace_root" rules={[{ required: true, message: '请输入临时工作目录根路径' }]}>
            <Input placeholder="/tmp/ai-devops" />
          </Form.Item>
          <Button type="primary" htmlType="submit" block>
            保存
          </Button>
        </Form>
      </Modal>

      <Modal title="派发需求" open={dispatchModalOpen} footer={null} onCancel={() => setDispatchModalOpen(false)}>
        <Form layout="vertical" form={dispatchForm} onFinish={submitDispatch}>
          <Form.Item label="执行 Agent" name="agent_code" rules={[{ required: true, message: '请选择执行 Agent' }]}>
            <Select options={agents.map((agent) => ({ label: `${agent.name} (${agent.agent_code})`, value: agent.agent_code }))} />
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

      {/* 项目配置抽屉 - 包含 Hook 管理 和 心跳配置 */}
      <Drawer
        title={`项目配置 - ${configProject?.name || ''}`}
        placement="right"
        width={800}
        onClose={closeProjectConfig}
        open={projectConfigDrawerOpen}
      >
        <Tabs
          items={[
            {
              key: 'basic',
              label: '基本信息',
              children: (
                <>
                  <Form
                    layout="vertical"
                    initialValues={{
                      agent_code: configProject?.agent_code || '',
                      dispatch_channel_code: configProject?.dispatch_channel_code || '',
                      dispatch_session_key: configProject?.dispatch_session_key || '',
                    }}
                    onFinish={async (values) => {
                      if (!configProject) return;
                      try {
                        await updateProject({
                          id: configProject.id,
                          name: configProject.name,
                          git_repo_url: configProject.git_repo_url,
                          default_branch: configProject.default_branch,
                          init_steps: configProject.init_steps,
                          heartbeat_enabled: configProject.heartbeat_enabled || false,
                          heartbeat_interval_minutes: configProject.heartbeat_interval_minutes || 60,
                          heartbeat_md_content: configProject.heartbeat_md_content || '',
                          agent_code: values.agent_code,
                          dispatch_channel_code: values.dispatch_channel_code,
                          dispatch_session_key: values.dispatch_session_key,
                        });
                        message.success('基本信息保存成功');
                        fetchProjects();
                      } catch (_error) {
                        message.error('保存失败');
                      }
                    }}
                  >
                    <Form.Item label="默认执行 Agent" name="agent_code">
                      <Select
                        options={agents.filter((a) => a.agent_type === 'CodingAgent').map((a) => ({
                          label: `${a.name} (${a.agent_code})`,
                          value: a.agent_code,
                        }))}
                        placeholder="选择用于执行需求、心跳和 Hook 的默认 Coding Agent"
                        style={{ width: 300 }}
                        allowClear
                      />
                    </Form.Item>

                    <Form.Item label="默认派发渠道" name="dispatch_channel_code">
                      <Select
                        options={channels.map((c) => ({
                          label: `${c.name} (${c.type})`,
                          value: c.channel_code,
                        }))}
                        placeholder="选择默认派发渠道"
                        style={{ width: 300 }}
                        allowClear
                      />
                    </Form.Item>

                    <Form.Item label="默认 SessionKey" name="dispatch_session_key">
                      <Input placeholder="例如：feishu:ou_xxx" style={{ width: 400 }} />
                    </Form.Item>

                    <Form.Item>
                      <Button type="primary" htmlType="submit">
                        保存基本信息
                      </Button>
                    </Form.Item>
                  </Form>

                  <Alert
                    message="提示"
                    description="这些配置是项目的默认执行环境，用于需求派发、心跳任务和 Hook 触发。Hook 触发时将自动使用此处配置的 Agent、渠道和 SessionKey。"
                    type="info"
                    showIcon
                    style={{ marginTop: 16 }}
                  />
                </>
              ),
            },
            {
              key: 'hooks',
              label: 'Hook 配置',
              children: (
                <>
                  <div style={{ marginBottom: 16 }}>
                    <Button type="primary" onClick={openCreateHook}>
                      新建 Hook
                    </Button>
                  </div>
                  <Table
                    dataSource={projectHooks}
                    loading={loadingProjectHooks}
                    rowKey="id"
                    pagination={false}
                    size="small"
                    columns={[
                      { title: '名称', dataIndex: 'name', key: 'name', width: 120 },
                      { title: '触发点', dataIndex: 'trigger_point', key: 'trigger_point', width: 140 },
                      { title: '动作类型', dataIndex: 'action_type', key: 'action_type', width: 100 },
                      { title: '优先级', dataIndex: 'priority', key: 'priority', width: 60 },
                      {
                        title: '启用',
                        dataIndex: 'enabled',
                        key: 'enabled',
                        width: 60,
                        render: (enabled: boolean, hook: HookConfig) => (
                          <Switch checked={enabled} onChange={() => handleToggleHookEnabled(hook)} />
                        ),
                      },
                      {
                        title: '操作',
                        key: 'action',
                        width: 120,
                        render: (_: unknown, hook: HookConfig) => (
                          <Space size="small">
                            <Button type="link" size="small" onClick={() => openEditHook(hook)}>
                              编辑
                            </Button>
                            <Popconfirm title="确定删除此 Hook？" onConfirm={() => handleDeleteHook(hook.id)}>
                              <Button type="link" size="small" danger>
                                删除
                              </Button>
                            </Popconfirm>
                          </Space>
                        ),
                      },
                    ]}
                  />
                </>
              ),
            },
            {
              key: 'heartbeat',
              label: '心跳配置',
              children: (
                <>
                  <Form
                    form={heartbeatForm}
                    layout="vertical"
                    initialValues={{
                      heartbeat_enabled: configProject?.heartbeat_enabled || false,
                      heartbeat_interval_minutes: configProject?.heartbeat_interval_minutes || 60,
                      heartbeat_md_content: configProject?.heartbeat_md_content || '',
                    }}
                  >
                    <Form.Item label="启用心跳" name="heartbeat_enabled" valuePropName="checked">
                      <Switch />
                    </Form.Item>

                    <Form.Item label="心跳间隔（分钟）" name="heartbeat_interval_minutes">
                      <Select
                        options={[
                          { label: '15分钟', value: 15 },
                          { label: '30分钟', value: 30 },
                          { label: '1小时', value: 60 },
                          { label: '2小时', value: 120 },
                          { label: '6小时', value: 360 },
                          { label: '12小时', value: 720 },
                          { label: '24小时', value: 1440 },
                        ]}
                        style={{ width: 200 }}
                      />
                    </Form.Item>

                    <Form.Item
                      name="heartbeat_md_content"
                      hidden
                    >
                      <Input />
                    </Form.Item>
                    <HeartbeatTemplateEditor
                      value={heartbeatForm.getFieldValue('heartbeat_md_content')}
                      onChange={(value) => heartbeatForm.setFieldValue('heartbeat_md_content', value)}
                    />

                    <Form.Item>
                      <Space>
                        <Button
                          type="primary"
                          onClick={handleSaveHeartbeat}
                          loading={savingHeartbeat}
                        >
                          保存心跳配置
                        </Button>
                      </Space>
                    </Form.Item>
                  </Form>

                  <Alert
                    message="提示"
                    description="心跳任务会使用选定的 Agent 定期执行，分析项目 PR 并根据评论内容决定是否创建新的需求。没问题的 PR 会评论 /lgtm，需要处理的会创建需求让其他 AI 执行修复。"
                    type="info"
                    showIcon
                    style={{ marginTop: 16 }}
                  />
                </>
              ),
            },
          ]}
        />
      </Drawer>

      {/* Hook 编辑 Modal */}
      <Modal
        title={editingHook ? '编辑 Hook' : '新建 Hook'}
        open={hookModalOpen}
        footer={null}
        onCancel={() => setHookModalOpen(false)}
        width={600}
      >
        <Form layout="vertical" form={hookForm} onFinish={submitHook}>
          <Form.Item name="project_id" hidden>
            <Input />
          </Form.Item>
          <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="例如：派发时通知" />
          </Form.Item>
          <Form.Item label="触发点" name="trigger_point" rules={[{ required: true, message: '请选择触发点' }]}>
            <Select>
              {TRIGGER_POINTS.map((tp) => (
                <Select.Option key={tp.value} value={tp.value}>
                  {tp.label}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item label="动作类型" name="action_type" rules={[{ required: true, message: '请选择动作类型' }]}>
            <Select>
              {ACTION_TYPES.map((at) => (
                <Select.Option key={at.value} value={at.value}>
                  {at.label}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          {/* Coding Agent 配置 */}
          <Form.Item noStyle shouldUpdate={(prev, curr) => prev.action_type !== curr.action_type}>
            {({ getFieldValue }) => {
              const actionType = getFieldValue('action_type');
              if (actionType === 'coding_agent') {
                return (
                  <>
                    <Alert
                      message="使用项目默认配置"
                      description="Hook 将使用【基本信息】Tab 中配置的默认 Agent、渠道和 SessionKey 执行。"
                      type="info"
                      showIcon
                      style={{ marginBottom: 16 }}
                    />
                    <Form.Item
                      label="Prompt 模板"
                      name="prompt_template"
                      rules={[{ required: true, message: '请输入 Prompt 模板' }]}
                      extra={
                        <div style={{ marginTop: 8 }}>
                          <div style={{ fontSize: 12, color: '#666', marginBottom: 8 }}>
                            可用变量（会被替换为实际值）：
                          </div>
                          <pre style={{ fontSize: 11, background: '#f5f5f5', padding: 8, borderRadius: 4 }}>
{`\${requirement.id}           - 需求ID
\${requirement.title}         - 需求标题
\${requirement.description}     - 需求描述
\${requirement.status}          - 需求状态 (todo/preparing/coding/pr_opened/failed/completed/done)
\${requirement.temp_workspace_root} - 临时工作目录根路径
\${project.id}               - 项目ID
\${project.name}         - 项目名称
\${workspace.path}        - 工作目录路径
\${change.trigger}        - 触发事件名称
\${change.reason}        - 状态变更原因`}
                          </pre>
                          <div style={{ fontSize: 12, color: '#666', marginTop: 12, marginBottom: 8 }}>
                            完成后请执行以下 CLI 命令标记需求完成：
                          </div>
                          <pre style={{ fontSize: 11, background: '#f0f8ff', padding: 8, borderRadius: 4 }}>
{`taskmanager requirement complete --id \${requirement.id}`}
                          </pre>
                        </div>
                      }
                    >
                      <Input.TextArea rows={6} placeholder="请处理需求 {{requirement.title}}，描述：{{requirement.description}}。完成后执行 taskmanager requirement complete 命令。" />
                    </Form.Item>
                  </>
                );
              }
              return (
                <Form.Item
                  label="动作配置"
                  name="action_config"
                  rules={[{ required: true, message: '请输入动作配置' }]}
                  extra={
                    <pre style={{ fontSize: 11, background: '#f5f5f5', padding: 8, borderRadius: 4 }}>
{`{"key": "value"}`}
                    </pre>
                  }
                >
                  <Input.TextArea rows={4} placeholder='{"key": "value"}' />
                </Form.Item>
              );
            }}
          </Form.Item>

          <Form.Item label="优先级" name="priority" initialValue={50}>
            <Input type="number" placeholder="数值越大越优先执行" />
          </Form.Item>
          <Form.Item label="启用" name="enabled" valuePropName="checked" initialValue={true}>
            <Switch />
          </Form.Item>
          <Button type="primary" htmlType="submit" block loading={savingHook}>
            保存
          </Button>
        </Form>
      </Modal>

      {/* 需求详情抽屉 */}
      <Drawer
        title={`需求详情 - ${detailRequirement?.title || ''}`}
        placement="right"
        width={900}
        onClose={() => setRequirementDetailDrawerOpen(false)}
        open={requirementDetailDrawerOpen}
      >
        {detailRequirement && (
          <Tabs
            items={[
              {
                key: 'basic',
                label: '基础信息',
                children: (
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
                    <div>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>需求ID</div>
                      <div style={{ fontFamily: 'monospace', fontSize: 13 }}>{detailRequirement.id}</div>
                    </div>
                    <div>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>项目ID</div>
                      <div style={{ fontFamily: 'monospace', fontSize: 13 }}>{detailRequirement.project_id}</div>
                    </div>
                    <div>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>状态</div>
                      <div>
                        <Tag color={statusColorMap[detailRequirement.status] || 'default'}>{detailRequirement.status}</Tag>
                      </div>
                    </div>
                    <div>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>需求类型</div>
                      <div>{detailRequirement.requirement_type || 'normal'}</div>
                    </div>
                    <div>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>创建时间</div>
                      <div>{detailRequirement.created_at ? new Date(detailRequirement.created_at).toLocaleString() : '-'}</div>
                    </div>
                    <div>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>更新时间</div>
                      <div>{detailRequirement.updated_at ? new Date(detailRequirement.updated_at).toLocaleString() : '-'}</div>
                    </div>
                    <div>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>开始时间</div>
                      <div>{detailRequirement.started_at ? new Date(detailRequirement.started_at).toLocaleString() : '-'}</div>
                    </div>
                    <div>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>完成时间</div>
                      <div>{detailRequirement.completed_at ? new Date(detailRequirement.completed_at).toLocaleString() : '-'}</div>
                    </div>
                  </div>
                ),
              },
              {
                key: 'content',
                label: '需求内容',
                children: (
                  <div>
                    <div style={{ marginBottom: 16 }}>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>标题</div>
                      <div style={{ fontSize: 16, fontWeight: 500 }}>{detailRequirement.title}</div>
                    </div>
                    <div style={{ marginBottom: 16 }}>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>描述</div>
                      <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                        {detailRequirement.description || '无'}
                      </pre>
                    </div>
                    <div style={{ marginBottom: 16 }}>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>验收标准</div>
                      <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                        {detailRequirement.acceptance_criteria || '无'}
                      </pre>
                    </div>
                  </div>
                ),
              },
              {
                key: 'workspace',
                label: '工作区信息',
                children: (
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
                    <div style={{ gridColumn: '1 / -1' }}>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>临时工作目录根路径</div>
                      <div style={{ fontFamily: 'monospace', fontSize: 13 }}>{detailRequirement.temp_workspace_root || '-'}</div>
                    </div>
                    <div style={{ gridColumn: '1 / -1' }}>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>工作目录</div>
                      <div style={{ fontFamily: 'monospace', fontSize: 13 }}>{detailRequirement.workspace_path || '-'}</div>
                    </div>
                  </div>
                ),
              },
              {
                key: 'dispatch',
                label: '派发信息',
                children: (
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
                    <div>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>派发SessionKey</div>
                      <div style={{ fontFamily: 'monospace', fontSize: 13 }}>{detailRequirement.dispatch_session_key || '-'}</div>
                    </div>
                    <div>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>分配Agent</div>
                      <div style={{ fontFamily: 'monospace', fontSize: 13 }}>{detailRequirement.assignee_agent_code || '-'}</div>
                    </div>
                    <div style={{ gridColumn: '1 / -1' }}>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>分身Agent</div>
                      <div style={{ fontFamily: 'monospace', fontSize: 13 }}>{detailRequirement.replica_agent_code || '-'}</div>
                    </div>
                    <div style={{ gridColumn: '1 / -1' }}>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>最近错误</div>
                      <div style={{ color: '#ff4d4f' }}>{detailRequirement.last_error || '无'}</div>
                    </div>
                  </div>
                ),
              },
              {
                key: 'claude',
                label: 'Claude执行',
                children: (
                  <div>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 16 }}>
                      <div>
                        <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>执行状态</div>
                        <div>
                          {detailRequirement.claude_runtime?.status ? (
                            <Tag color={claudeRuntimeColorMap[detailRequirement.claude_runtime.status] || 'default'}>
                              {detailRequirement.claude_runtime.status}
                            </Tag>
                          ) : (
                            '-'
                          )}
                        </div>
                      </div>
                      <div>
                        <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>运行状态</div>
                        <div>
                          {detailRequirement.claude_runtime?.status ? (
                            <Tag color={detailRequirement.claude_runtime.status === 'running' ? 'processing' : 'default'}>
                              {detailRequirement.claude_runtime.status === 'running' ? '运行中' : '已停止'}
                            </Tag>
                          ) : (
                            '-'
                          )}
                        </div>
                      </div>
                      <div>
                        <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>开始时间</div>
                        <div>
                          {detailRequirement.claude_runtime?.started_at
                            ? new Date(detailRequirement.claude_runtime.started_at).toLocaleString()
                            : '-'}
                        </div>
                      </div>
                      <div>
                        <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>结束时间</div>
                        <div>
                          {detailRequirement.claude_runtime?.ended_at
                            ? new Date(detailRequirement.claude_runtime.ended_at).toLocaleString()
                            : '-'}
                        </div>
                      </div>
                    </div>

                    <div style={{ marginBottom: 16 }}>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>执行提示词</div>
                      <pre
                        style={{
                          background: '#f0f5ff',
                          padding: 12,
                          borderRadius: 4,
                          whiteSpace: 'pre-wrap',
                          wordBreak: 'break-word',
                          maxHeight: 300,
                          overflow: 'auto',
                          border: '1px solid #adc6ff',
                        }}
                      >
                        {detailRequirement.claude_runtime?.prompt || '无'}
                      </pre>
                    </div>

                    <div>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>执行结果</div>
                      <pre
                        style={{
                          background: '#f6ffed',
                          padding: 12,
                          borderRadius: 4,
                          whiteSpace: 'pre-wrap',
                          wordBreak: 'break-word',
                          maxHeight: 300,
                          overflow: 'auto',
                          border: '1px solid #b7eb8f',
                        }}
                      >
                        {detailRequirement.claude_runtime?.result || '无'}
                      </pre>
                    </div>

                    {detailRequirement.claude_runtime?.last_error && (
                      <div style={{ marginTop: 16 }}>
                        <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>执行错误</div>
                        <pre
                          style={{
                            background: '#fff2f0',
                            padding: 12,
                            borderRadius: 4,
                            whiteSpace: 'pre-wrap',
                            wordBreak: 'break-word',
                            border: '1px solid #ffccc7',
                            color: '#ff4d4f',
                          }}
                        >
                          {detailRequirement.claude_runtime.last_error}
                        </pre>
                      </div>
                    )}
                  </div>
                ),
              },
            ]}
          />
        )}
      </Drawer>

      {/* Trace Viewer 弹窗 */}
      <TraceViewer
        traceId={currentTraceId}
        visible={traceViewerVisible}
        onClose={() => setTraceViewerVisible(false)}
      />
    </div>
  );
};
