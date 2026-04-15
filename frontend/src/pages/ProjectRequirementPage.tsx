import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Breadcrumb, Button, Card, Drawer, Dropdown, Form, Input, MenuProps, Modal, Popconfirm, Segmented, Select, Space, Table, Tabs, Tag, Switch, message, Alert, Tooltip, Row, Col, Progress, List } from 'antd';
import { CopyOutlined, SettingOutlined, EditOutlined, DeleteOutlined, CheckCircleOutlined, ClockCircleOutlined, SyncOutlined } from '@ant-design/icons';
import { batchDeleteRequirements, copyAndDispatchRequirement, createProject, createRequirement, deleteProject, deleteRequirement, dispatchRequirement, getRequirement, listProjects, listRequirements, updateProject, updateRequirement, updateRequirementStatus, getRequirementTransitionHistory, getStatusStats, type TransitionLog, type StatusStat } from '../api/projectRequirementApi';
import { listAgents } from '../api/agentApi';
import { listChannels } from '../api/channelApi';
import { useAuthStore } from '../stores/authStore';
import type { Agent } from '../types/agent';
import type { Channel } from '../types/channel';
import type { CreateProjectRequest, CreateRequirementRequest, Project, Requirement, ProgressData, TodoItem } from '../types/projectRequirement';
import { HeartbeatTemplateEditor } from '../components/HeartbeatTemplate';
import { TraceViewer } from '../components/TraceViewer';
import { RequirementStatusStats } from '../components/RequirementStatusStats';
import { RequirementKanban } from '../components/RequirementKanban';
import { ProjectStateMachineConfig } from '../components/ProjectStateMachineConfig';
import { RequirementTypeManagementPage } from '../components/RequirementTypeManagement';
import { requirementTypeApi, type RequirementType } from '../api/requirementTypeApi';
import { getProjectStateMachineByType } from '../api/projectStateMachineApi';
import { getStateMachine } from '../api/stateMachineApi';
import type { State } from '../types/stateMachine';
import { statusLabels, getStatusColor } from '../constants/requirementStatus';
import type { Breakpoint } from 'antd/es/_util/responsiveObserver';

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

const agentRuntimeColorMap: Record<string, string> = {
  running: 'processing',
  completed: 'success',
  failed: 'error',
};

const defaultDispatchSessionKey = 'feishu:ou_df798fe15d056000143691af8c1cdb55';

export const ProjectRequirementPage: React.FC = () => {
  const { user } = useAuthStore();
  const [projects, setProjects] = useState<Project[]>([]);
  const [requirements, setRequirements] = useState<Requirement[]>([]);
  const [statusStats, setStatusStats] = useState<StatusStat[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loadingProjects, setLoadingProjects] = useState(false);
  const [loadingRequirements, setLoadingRequirements] = useState(false);
  const [selectedProject, setSelectedProject] = useState<Project | null>(null);
  const selectedProjectId = selectedProject?.id || '';
  const [projectStatsMap, setProjectStatsMap] = useState<Record<string, StatusStat[]>>({});
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

  // 心跳配置相关状态
  const [heartbeatForm] = Form.useForm();
  const [savingHeartbeat, setSavingHeartbeat] = useState(false);

  // 需求详情抽屉状态
  const [requirementDetailDrawerOpen, setRequirementDetailDrawerOpen] = useState(false);

  // Trace Viewer 状态
  const [traceViewerVisible, setTraceViewerVisible] = useState(false);
  const [currentTraceId, setCurrentTraceId] = useState<string>('');
  const [detailRequirement, setDetailRequirement] = useState<Requirement | null>(null);
  // 需求状态转换历史
  const [transitionHistory, setTransitionHistory] = useState<TransitionLog[]>([]);

  // 进度详情弹窗状态
  const [progressModalOpen, setProgressModalOpen] = useState(false);
  const [progressModalData, setProgressModalData] = useState<ProgressData | null>(null);
  const [progressModalTitle, setProgressModalTitle] = useState('');
  const [progressModalReqId, setProgressModalReqId] = useState<string>('');
  const [progressModalLoading, setProgressModalLoading] = useState(false);
  const [agentProgressLoading, setAgentProgressLoading] = useState(false);

  // 需求状态过滤
  const [statusFilter, setStatusFilter] = useState<string>('');

  // 选中需求ID列表（用于批量删除）
  const [selectedRequirementKeys, setSelectedRequirementKeys] = useState<React.Key[]>([]);

  // 需求类型过滤
  const [typeFilter, setTypeFilter] = useState<string>('');

  // 视图切换：表格 / 看板
  const [viewMode, setViewMode] = useState<'表格' | '看板'>('表格');
  const [kanbanRefreshTrigger, setKanbanRefreshTrigger] = useState(0);

  // 需求类型列表（用于创建需求时选择）
  const [requirementTypes, setRequirementTypes] = useState<RequirementType[]>([]);

  // 可选状态列表（按需求类型存储）
  const [statesByType, setStatesByType] = useState<Record<string, State[]>>({});
  // 标记是否尝试加载过状态机（用于显示更准确的提示）
  const [stateMachineLoadAttempted, setStateMachineLoadAttempted] = useState(false);

  // 当项目变化时，预加载所有需求类型的状态机
  useEffect(() => {
    const fetchAllStateMachines = async () => {
      if (!selectedProjectId) {
        setStateMachineLoadAttempted(false);
        return;
      }
      const types = requirementTypes.map(t => t.code);
      const newStatesByType: Record<string, State[]> = {};
      for (const type of types) {
        try {
          const mapping = await getProjectStateMachineByType(selectedProjectId, type);
          if (mapping?.state_machine_id) {
            const sm = await getStateMachine(mapping.state_machine_id);
            // 按类型存储状态
            newStatesByType[type] = sm.config.states;
          }
        } catch {
          // ignore - API返回404表示该类型未配置状态机
        }
      }
      setStateMachineLoadAttempted(true);
      setStatesByType(newStatesByType);
    };
    fetchAllStateMachines();
  }, [selectedProjectId, requirementTypes]);

  // 根据状态和类型过滤后的需求列表
  const filteredRequirements = useMemo(() => {
    return requirements.filter((req) => {
      const matchStatus = !statusFilter || req.status === statusFilter;
      const matchType = !typeFilter || req.requirement_type === typeFilter;
      return matchStatus && matchType;
    });
  }, [requirements, statusFilter, typeFilter]);

  // 当项目或状态筛选变化时，清空选择
  useEffect(() => {
    setSelectedRequirementKeys([]);
  }, [selectedProjectId, statusFilter]);

  // 当过滤后的列表变化时，过滤掉不可见的选择项
  useEffect(() => {
    const visibleIds = new Set(filteredRequirements.map((req) => req.id));
    setSelectedRequirementKeys((prev) => prev.filter((key) => visibleIds.has(key as string)));
  }, [filteredRequirements]);

  const projectOptions = useMemo(
    () => projects.map((project) => ({ label: project.name, value: project.id })),
    [projects],
  );

  // 动态生成状态选项
  const statusOptions = [
    { label: '全部状态', value: '' },
    ...statusStats.map((stat) => ({
      label: `${statusLabels[stat.status] || stat.status} (${stat.status})`,
      value: stat.status,
    })),
  ];

  const fetchProjects = useCallback(async () => {
    setLoadingProjects(true);
    try {
      const data = await listProjects();
      setProjects(data);
      // 获取每个项目的状态统计
      const statsMap: Record<string, StatusStat[]> = {};
      await Promise.all(
        data.map(async (project) => {
          try {
            const stats = await getStatusStats(project.id);
            statsMap[project.id] = stats;
          } catch {
            statsMap[project.id] = [];
          }
        }),
      );
      setProjectStatsMap(statsMap);
    } catch (_error) {
      message.error('获取项目列表失败');
    } finally {
      setLoadingProjects(false);
    }
  }, []);

  const fetchRequirements = useCallback(async (projectId?: string) => {
    setLoadingRequirements(true);
    try {
      const [data, stats] = await Promise.all([
        listRequirements(projectId || selectedProjectId || undefined),
        getStatusStats(projectId || selectedProjectId || undefined),
      ]);
      setRequirements(data);
      setStatusStats(stats);
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
      setAgents(data.filter((agent) => ['CodingAgent', 'OpenCodeAgent'].includes(agent.agent_type)));
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

  const fetchRequirementTypes = useCallback(async (projectId: string) => {
    try {
      const data = await requirementTypeApi.list(projectId);
      setRequirementTypes(data);
    } catch (_error) {
      // 忽略错误，不影响其他功能
    }
  }, []);

  useEffect(() => {
    fetchProjects();
    fetchAgents();
    fetchChannels();
  }, [fetchAgents, fetchChannels, fetchProjects]);

  useEffect(() => {
    if (selectedProjectId) {
      fetchRequirementTypes(selectedProjectId);
    }
  }, [fetchRequirementTypes, selectedProjectId]);

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

  const submitRequirement = async (values: { project_id: string; title: string; description: string; acceptance_criteria: string; temp_workspace_root: string; requirement_type?: string }) => {
    const payload: CreateRequirementRequest = {
      project_id: values.project_id,
      title: values.title,
      description: values.description || '',
      acceptance_criteria: values.acceptance_criteria || '',
      temp_workspace_root: values.temp_workspace_root || '',
      requirement_type: values.requirement_type || (requirementTypes[0]?.code || ''),
    };
    try {
      if (editingRequirement) {
        await updateRequirement({
          id: editingRequirement.id,
          title: payload.title,
          description: payload.description,
          acceptance_criteria: payload.acceptance_criteria,
          temp_workspace_root: payload.temp_workspace_root,
          requirement_type: payload.requirement_type,
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
        setSelectedProject(null);
      }
      await fetchProjects();
    } catch (_error) {
      message.error('删除项目失败');
    }
  };

  const openCreateRequirement = () => {
    if (requirementTypes.length === 0) {
      message.warning('需求类型加载中，请稍候');
      return;
    }
    setEditingRequirement(null);
    requirementForm.resetFields();
    requirementForm.setFieldsValue({ project_id: selectedProjectId, temp_workspace_root: '/tmp/ai-devops', requirement_type: requirementTypes[0]?.code || '' });
    setRequirementModalOpen(true);
  };

  const handleViewRequirements = async (projectId: string) => {
    const project = projects.find((p) => p.id === projectId);
    if (project) {
      setSelectedProject(project);
      await fetchRequirements(projectId);
    }
  };

  const openEditRequirement = (item: Requirement) => {
    setEditingRequirement(item);
    requirementForm.setFieldsValue({
      project_id: item.project_id,
      title: item.title,
      description: item.description,
      acceptance_criteria: item.acceptance_criteria,
      temp_workspace_root: item.temp_workspace_root,
      requirement_type: item.requirement_type || '',
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

  // 删除单个需求
  const handleDeleteRequirement = async (item: Requirement) => {
    try {
      await deleteRequirement(item.id);
      message.success('删除需求成功');
      // 从选中列表中移除已删除的id
      setSelectedRequirementKeys((prev) => prev.filter((key) => key !== item.id));
      await fetchRequirements(selectedProjectId);
    } catch (_error) {
      message.error('删除需求失败');
    }
  };

  // 批量删除需求
  const handleBatchDeleteRequirements = async () => {
    if (selectedRequirementKeys.length === 0) {
      message.warning('请先选择要删除的需求');
      return;
    }
    try {
      await batchDeleteRequirements(selectedRequirementKeys as string[]);
      message.success(`成功删除 ${selectedRequirementKeys.length} 个需求`);
      setSelectedRequirementKeys([]);
      await fetchRequirements(selectedProjectId);
    } catch (_error) {
      message.error('批量删除需求失败');
    }
  };

  const openRequirementDetail = async (item: Requirement) => {
    setDetailRequirement(item);
    setRequirementDetailDrawerOpen(true);
    // 获取状态转换历史
    try {
      const history = await getRequirementTransitionHistory(item.id);
      setTransitionHistory(history);
    } catch {
      setTransitionHistory([]);
    }
  };

  // 项目配置相关处理
  const openProjectConfig = async (project: Project) => {
    setConfigProject(project);
    setProjectConfigDrawerOpen(true);

    // 设置心跳表单默认值
    heartbeatForm.setFieldsValue({
      heartbeat_enabled: project.heartbeat_enabled || false,
      heartbeat_interval_minutes: project.heartbeat_interval_minutes || 60,
      heartbeat_md_content: project.heartbeat_md_content || '',
      agent_code: project.agent_code || '',
    });
  };

  const closeProjectConfig = () => {
    setProjectConfigDrawerOpen(false);
    setConfigProject(null);
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

  // 解析 progress_data（后端可能是 JSON 字符串或对象）
  const parseProgressData = (item: Requirement): ProgressData | null => {
    if (!item.progress_data) return null;
    if (typeof item.progress_data === 'string') {
      try {
        return JSON.parse(item.progress_data) as ProgressData;
      } catch {
        return null;
      }
    }
    return item.progress_data as ProgressData;
  };

  const openProgressModal = (item: Requirement) => {
    const data = parseProgressData(item);
    if (!data) return;
    setProgressModalData(data);
    setProgressModalTitle(item.title);
    setProgressModalReqId(item.id);
    setProgressModalOpen(true);
  };

  const refreshProgressModal = async () => {
    if (!progressModalReqId) return;
    setProgressModalLoading(true);
    try {
      const item = await getRequirement(progressModalReqId);
      const data = parseProgressData(item);
      if (data) {
        setProgressModalData(data);
        setProgressModalTitle(item.title);
      }
      // 同时刷新列表中的数据
      if (selectedProjectId) {
        await fetchRequirements(selectedProjectId);
      }
    } catch (err) {
      message.error('刷新进度失败');
    } finally {
      setProgressModalLoading(false);
    }
  };

  const refreshAgentProgress = async () => {
    if (!detailRequirement) return;
    setAgentProgressLoading(true);
    try {
      const item = await getRequirement(detailRequirement.id);
      setDetailRequirement(item);
      if (selectedProjectId) {
        await fetchRequirements(selectedProjectId);
      }
      message.success('刷新进度成功');
    } catch (err) {
      message.error('刷新进度失败');
    } finally {
      setAgentProgressLoading(false);
    }
  };

  // 获取类型配置（用于显示）
  const getTypeDisplay = (code: string): { label: string; color: string } => {
    const typeConfig = requirementTypes.find((t) => t.code === code);
    if (typeConfig) {
      return { label: typeConfig.name || code, color: typeConfig.color || 'default' };
    }
    return { label: code || '-', color: 'default' };
  };

  const requirementColumns = [
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, item: Requirement) => {
        // 根据需求类型获取对应的状态
        const reqType = item.requirement_type || '';
        const availableStates = statesByType[reqType] || [];

        // 构建状态子菜单
        const statusSubMenuItems: MenuProps['items'] = availableStates.map((state) => ({
          key: `status-${state.id}`,
          label: state.name,
          disabled: item.status === state.id,
          onClick: async () => {
            try {
              await updateRequirementStatus(item.id, state.id);
              message.success(`状态已修改为: ${state.name}`);
              await fetchRequirements(selectedProjectId);
            } catch {
              message.error('修改状态失败');
            }
          },
        }));

        const menuItems: MenuProps['items'] = [
          { key: 'detail', label: '详情', onClick: () => openRequirementDetail(item) },
          { key: 'edit', label: '编辑', onClick: () => openEditRequirement(item) },
          {
            key: 'status',
            label: '修改状态',
            children: statusSubMenuItems.length > 0
              ? statusSubMenuItems
              : [
                  {
                    key: 'no-status',
                    label: stateMachineLoadAttempted && selectedProjectId
                      ? '请在项目设置中配置状态机'
                      : '暂无可用状态',
                    disabled: true,
                  },
                ],
          },
          { key: 'dispatch', label: '派发', disabled: item.status !== 'todo', onClick: () => openDispatchModal(item) },
          { key: 'copy', label: '复制并派发', disabled: item.status === 'todo', onClick: () => handleCopyAndDispatch(item) },
        ];
        if (item.trace_id) {
          menuItems.push({ key: 'trace', label: '对话链路', onClick: () => { setCurrentTraceId(item.trace_id!); setTraceViewerVisible(true); } });
        }
        menuItems.push(
          { type: 'divider' },
          {
            key: 'delete',
            label: (
              <Popconfirm
                title="确认删除"
                description={`确定要删除需求 "${item.title}" 吗？`}
                onConfirm={() => handleDeleteRequirement(item)}
                okText="确认"
                cancelText="取消"
              >
                <span style={{ color: '#ff4d4f' }}>删除</span>
              </Popconfirm>
            ),
            danger: true,
          }
        );
        return (
          <Dropdown menu={{ items: menuItems }} trigger={['click']}>
            <Button type="link" size="small" style={{ padding: 0 }}>操作</Button>
          </Dropdown>
        );
      },
        width: 80,
        fixed: 'left' as const
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
      minWidth: 150,
      render: (title: string, item: Requirement) => {
        if (!item.trace_id) return title;

        return (
          <Button
            type="link"
            style={{ padding: 0, height: 'auto', textAlign: 'left', whiteSpace: 'normal' }}
            onClick={() => {
              setCurrentTraceId(item.trace_id!);
              setTraceViewerVisible(true);
            }}
          >
            {title}
          </Button>
        );
      }
    },
    {
      title: '类型',
      key: 'requirement_type',
      width: 90,
      responsive: ['sm'] as Breakpoint[],
      render: (_: unknown, item: Requirement) => {
        const display = getTypeDisplay(item.requirement_type || '');
        return <Tag color={display.color}>{display.label}</Tag>;
      },
    },
    {
      title: '状态',
      key: 'status',
      width: 100,
      render: (_: unknown, item: Requirement) => (
        <Tag color={statusColorMap[item.status] || 'default'}>{item.status}</Tag>
      ),
    },
    {
      title: 'Agent',
      key: 'agent',
      width: 140,
      responsive: ['sm'] as Breakpoint[],
      render: (_: unknown, item: Requirement) => {
        const agent = item.replica_agent?.name
          ? item.replica_agent
          : item.assignee_agent?.name
            ? item.assignee_agent
            : null;
        if (!agent) {
          return <span>-</span>;
        }
        return (
          <Tooltip title={`${agent.name} (${agent.agent_code})`}>
            <Tag color="cyan">{agent.name}</Tag>
          </Tooltip>
        );
      },
    },
    {
      title: 'Agent状态',
      key: 'agent_runtime',
      width: 140,
      responsive: ['sm'] as Breakpoint[],
      render: (_: unknown, item: Requirement) => {
        const runtimeStatus = item.agent_runtime?.status || '';
        if (!runtimeStatus) {
          return <span>-</span>;
        }
        const isRunning = runtimeStatus === 'running';
        return (
          <Space size={4}>
            <Tag color={agentRuntimeColorMap[runtimeStatus] || 'default'}>{runtimeStatus}</Tag>
            {isRunning && <Tag color="processing">运行中</Tag>}
          </Space>
        );
      },
    },
    {
      title: '进度',
      key: 'progress',
      width: 120,
      responsive: ['sm'] as Breakpoint[],
      render: (_: unknown, item: Requirement) => {
        const data = parseProgressData(item);
        if (!data || !data.items || data.items.length === 0) {
          return <span>-</span>;
        }
        return (
          <div style={{ cursor: 'pointer', minWidth: 80 }} onClick={() => openProgressModal(item)}>
            <Progress percent={data.percent} size="small" strokeColor={data.percent === 100 ? '#52c41a' : '#1890ff'} />
          </div>
        );
      },
    },
    {
      title: 'Token消耗',
      key: 'tokens',
      width: 100,
      responsive: ['sm'] as Breakpoint[],
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
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      responsive: ['sm'] as Breakpoint[],
      render: (createdAt: string) => createdAt ? new Date(createdAt).toLocaleString() : '-',
    },
  ];

  // 渲染项目卡片列表
  const renderProjectCards = () => (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>项目列表</h2>
        <Space>
          <Button onClick={() => fetchProjects()} loading={loadingProjects}>刷新</Button>
          <Button type="primary" onClick={openCreateProject}>新建项目</Button>
        </Space>
      </div>
      <Row gutter={[16, 16]}>
        {loadingProjects && projects.length === 0 && (
          [1, 2, 3, 4].map((i) => (
            <Col xs={24} sm={12} md={8} lg={6} key={i}>
              <Card loading style={{ height: 220 }} />
            </Col>
          ))
        )}
        {!loadingProjects && projects.length === 0 && (
          <Col span={24}>
            <Card>
              <div style={{ textAlign: 'center', color: '#999', padding: 40 }}>
                暂无项目，点击"新建项目"创建第一个项目
              </div>
            </Card>
          </Col>
        )}
        {projects.map((project) => {
          const stats = projectStatsMap[project.id] || [];
          const total = stats.reduce((sum, s) => sum + s.count, 0);
          return (
            <Col xs={24} sm={12} md={8} lg={6} key={project.id}>
              <Card
                hoverable
                style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
                styles={{ body: { flex: 1 } }}
                onClick={() => handleViewRequirements(project.id)}
                actions={[
                  <SettingOutlined key="config" onClick={(e) => { e.stopPropagation(); openProjectConfig(project); }} />,
                  <EditOutlined key="edit" onClick={(e) => { e.stopPropagation(); openEditProject(project); }} />,
                  <Popconfirm key="delete" title="确认删除该项目？" description="删除后不可恢复，该项目下所有需求仍会保留" onConfirm={(e) => { e?.stopPropagation(); handleDeleteProject(project.id); }} onCancel={(e) => e?.stopPropagation()}>
                    <DeleteOutlined onClick={(e) => e.stopPropagation()} />
                  </Popconfirm>,
                ]}
              >
                <Card.Meta
                  title={project.name}
                  description={
                    <div>
                      <div style={{ fontSize: 12, color: '#999', marginBottom: 8, wordBreak: 'break-all' }}>
                        {project.git_repo_url}
                      </div>
                      <div style={{ fontSize: 12, color: '#666' }}>
                        分支: {project.default_branch}
                      </div>
                      {project.heartbeat_enabled && (
                        <Tag color="green" style={{ marginTop: 4, fontSize: 11 }}>心跳</Tag>
                      )}
                      <div style={{ marginTop: 8, borderTop: '1px solid #f0f0f0', paddingTop: 8 }}>
                        <div style={{ fontSize: 12, color: '#999', marginBottom: 4 }}>
                          需求总数: {total}
                        </div>
                        {stats.length > 0 && (
                          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                            {stats.map((stat) => {
                              const colors = getStatusColor(stat.status);
                              const label = statusLabels[stat.status] || stat.status;
                              return (
                                <Tag key={stat.status} style={{ fontSize: 11, margin: 0, color: colors.color, backgroundColor: colors.bgColor, borderColor: colors.borderColor }}>
                                  {label}: {stat.count}
                                </Tag>
                              );
                            })}
                          </div>
                        )}
                      </div>
                    </div>
                  }
                />
              </Card>
            </Col>
          );
        })}
      </Row>
    </div>
  );

  // 渲染需求管理视图
  const renderRequirements = () => (
    <div>
      <Breadcrumb
        style={{ marginBottom: 16 }}
        items={[
          {
            title: <a onClick={() => { setSelectedProject(null); setStatusFilter(''); setTypeFilter(''); setSelectedRequirementKeys([]); }}>项目列表</a>,
          },
          {
            title: selectedProject?.name || '',
          },
        ]}
      />
      <RequirementStatusStats
        statusStats={statusStats}
        statusFilter={statusFilter}
        onStatusClick={(status) => setStatusFilter(status)}
      />
      <Card
        title={viewMode === '表格' ? `需求列表 (${filteredRequirements.length})` : '需求看板'}
        extra={
          <Space>
            <Segmented
              value={viewMode}
              onChange={(v) => setViewMode(v as '表格' | '看板')}
              options={['表格', '看板']}
            />
            {viewMode === '表格' && (
              <>
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
                    ...requirementTypes.map((t) => ({ label: `${t.name} (${t.code})`, value: t.code })),
                  ]}
                  onChange={(value) => setTypeFilter(value || '')}
                  allowClear
                />
              </>
            )}
            <Button onClick={() => {
              if (viewMode === '看板') {
                setKanbanRefreshTrigger((t) => t + 1);
              } else {
                fetchRequirements(selectedProjectId);
              }
            }}>刷新</Button>
            <Button type="primary" onClick={openCreateRequirement}>
              新建需求
            </Button>
            {viewMode === '表格' && (
              <Popconfirm
                title="批量删除"
                description={`确定要删除选中的 ${selectedRequirementKeys.length} 个需求吗？`}
                onConfirm={handleBatchDeleteRequirements}
                okText="确认"
                cancelText="取消"
                disabled={selectedRequirementKeys.length === 0}
              >
                <Button danger disabled={selectedRequirementKeys.length === 0}>
                  批量删除 ({selectedRequirementKeys.length})
                </Button>
              </Popconfirm>
            )}
          </Space>
        }
      >
        {viewMode === '表格' ? (
          <Table<Requirement>
            rowKey="id"
            loading={loadingRequirements}
            dataSource={filteredRequirements}
            columns={requirementColumns}
            rowSelection={{
              type: 'checkbox',
              selectedRowKeys: selectedRequirementKeys,
              onChange: (selectedKeys) => setSelectedRequirementKeys(selectedKeys),
            }}
          />
        ) : (
          <RequirementKanban
            projectId={selectedProjectId}
            statusStats={statusStats}
            onRequirementClick={(req) => {
              setDetailRequirement(req);
              setRequirementDetailDrawerOpen(true);
            }}
            refreshTrigger={kanbanRefreshTrigger}
            onRefresh={() => fetchRequirements(selectedProjectId)}
            requirementTypes={requirementTypes}
          />
        )}
      </Card>
    </div>
  );

  return (
    <div style={{ padding: 0 }}>
      {selectedProject ? renderRequirements() : renderProjectCards()}

      <Drawer title={editingProject ? '编辑项目' : '新建项目'} open={projectModalOpen} onClose={() => setProjectModalOpen(false)} width={480}>
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
              options={agents.filter((a) => ['CodingAgent', 'OpenCodeAgent'].includes(a.agent_type)).map((a) => ({
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
      </Drawer>

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
          <Form.Item label="需求类型" name="requirement_type" initialValue={requirementTypes[0]?.code || ''}>
            <Select
              options={requirementTypes.map((t) => ({ label: `${t.name} (${t.code})`, value: t.code }))}
              placeholder="选择需求类型"
            />
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
                        options={agents.filter((a) => ['CodingAgent', 'OpenCodeAgent'].includes(a.agent_type)).map((a) => ({
                          label: `${a.name} (${a.agent_code})`,
                          value: a.agent_code,
                        }))}
                        placeholder="选择用于执行需求、心跳和 Hook 的默认 Agent"
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
            {
              key: 'stateMachine',
              label: '状态机配置',
              children: configProject ? <ProjectStateMachineConfig projectId={configProject.id} /> : null,
            },
            {
              key: 'requirementTypes',
              label: '需求类型',
              children: configProject ? <RequirementTypeManagementPage projectId={configProject.id} /> : null,
            },
          ]}
        />
      </Drawer>

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
                      <div>
                        <Tag color={getTypeDisplay(detailRequirement.requirement_type || '').color}>
                          {getTypeDisplay(detailRequirement.requirement_type || '').label}
                        </Tag>
                      </div>
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
                      <div>
                        {detailRequirement.assignee_agent?.name ? (
                          <Tag color="blue">{detailRequirement.assignee_agent.name}</Tag>
                        ) : (
                          <span style={{ fontFamily: 'monospace', fontSize: 13 }}>
                            {detailRequirement.assignee_agent_code || detailRequirement.replica_agent?.shadow_from || '-'}
                          </span>
                        )}
                      </div>
                    </div>
                    <div>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>分身Agent</div>
                      <div>
                        {detailRequirement.replica_agent?.name ? (
                          <Tag color="cyan">{detailRequirement.replica_agent.name}</Tag>
                        ) : (
                          <span style={{ fontFamily: 'monospace', fontSize: 13 }}>{detailRequirement.replica_agent_code || '-'}</span>
                        )}
                      </div>
                    </div>
                    <div>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>分身来源</div>
                      <div style={{ fontFamily: 'monospace', fontSize: 13 }}>
                        {detailRequirement.replica_agent?.shadow_from
                          ? `${detailRequirement.replica_agent.shadow_from}`
                          : '-'}
                      </div>
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
                label: 'Agent执行',
                children: (
                  <div>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 16 }}>
                      <div>
                        <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>执行状态</div>
                        <div>
                          {detailRequirement.agent_runtime?.status ? (
                            <Tag color={agentRuntimeColorMap[detailRequirement.agent_runtime.status] || 'default'}>
                              {detailRequirement.agent_runtime.status}
                            </Tag>
                          ) : (
                            '-'
                          )}
                        </div>
                      </div>
                      <div>
                        <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>运行状态</div>
                        <div>
                          {detailRequirement.agent_runtime?.status ? (
                            <Tag color={detailRequirement.agent_runtime.status === 'running' ? 'processing' : 'default'}>
                              {detailRequirement.agent_runtime.status === 'running' ? '运行中' : '已停止'}
                            </Tag>
                          ) : (
                            '-'
                          )}
                        </div>
                      </div>
                      <div>
                        <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>开始时间</div>
                        <div>
                          {detailRequirement.agent_runtime?.started_at
                            ? new Date(detailRequirement.agent_runtime.started_at).toLocaleString()
                            : '-'}
                        </div>
                      </div>
                      <div>
                        <div style={{ marginBottom: 8, color: '#666', fontSize: 12 }}>结束时间</div>
                        <div>
                          {detailRequirement.agent_runtime?.ended_at
                            ? new Date(detailRequirement.agent_runtime.ended_at).toLocaleString()
                            : '-'}
                        </div>
                      </div>
                    </div>

                    {/* 进度详情 */}
                    {(() => {
                      const progressData = parseProgressData(detailRequirement);
                      if (!progressData || progressData.items.length === 0) return null;
                      return (
                        <Card
                          title="执行进度"
                          size="small"
                          style={{ marginBottom: 16 }}
                          extra={
                            <Button
                              size="small"
                              icon={<SyncOutlined spin={agentProgressLoading} />}
                              loading={agentProgressLoading}
                              onClick={refreshAgentProgress}
                            >
                              刷新
                            </Button>
                          }
                        >
                          <div style={{ opacity: agentProgressLoading ? 0.6 : 1, transition: 'opacity 0.2s' }}>
                            <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                              <span style={{ fontSize: 14, color: '#666' }}>整体进度</span>
                              <span style={{ fontSize: 16, fontWeight: 600, color: progressData.percent === 100 ? '#52c41a' : '#1890ff' }}>
                                {progressData.percent}%
                              </span>
                            </div>
                            <Progress percent={progressData.percent} size="small" strokeColor={progressData.percent === 100 ? '#52c41a' : '#1890ff'} />
                            {progressData.updated_at && (
                              <div style={{ marginTop: 4, marginBottom: 12, fontSize: 12, color: '#999', textAlign: 'right' }}>
                                更新时间: {new Date(progressData.updated_at).toLocaleString()}
                              </div>
                            )}
                            <List
                              size="small"
                              dataSource={progressData.items}
                              renderItem={(todo: TodoItem) => {
                                const statusLower = (todo.status || '').toLowerCase();
                                const isDone = statusLower === 'completed' || statusLower === 'done';
                                const isRunning = statusLower === 'in_progress' || statusLower === 'running' || statusLower === 'doing';
                                const icon = isDone ? <CheckCircleOutlined style={{ color: '#52c41a' }} /> : isRunning ? <SyncOutlined spin style={{ color: '#1890ff' }} /> : <ClockCircleOutlined style={{ color: '#999' }} />;
                                return (
                                  <List.Item style={{ padding: '8px 0', borderBottom: '1px solid #f0f0f0' }}>
                                    <div style={{ display: 'flex', alignItems: 'center', gap: 10, width: '100%' }}>
                                      {icon}
                                      <span style={{ flex: 1, fontSize: 14, textDecoration: isDone ? 'line-through' : 'none', color: isDone ? '#999' : '#333' }}>
                                        {todo.content}
                                      </span>
                                      <Tag color={isDone ? 'success' : isRunning ? 'processing' : 'default'}>{todo.status || 'pending'}</Tag>
                                      {todo.priority && <Tag color="warning">{todo.priority}</Tag>}
                                    </div>
                                  </List.Item>
                                );
                              }}
                            />
                          </div>
                        </Card>
                      );
                    })()}

                    <div style={{ marginBottom: 16 }}>
                      <div style={{ marginBottom: 8, color: '#666', fontSize: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <span>执行提示词</span>
                        <Button
                          size="small"
                          icon={<CopyOutlined />}
                          onClick={() => {
                            navigator.clipboard.writeText(detailRequirement.agent_runtime?.prompt || '');
                            message.success('已复制到剪贴板');
                          }}
                        >
                          复制
                        </Button>
                      </div>
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
                        {detailRequirement.agent_runtime?.prompt || '无'}
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
                        {detailRequirement.agent_runtime?.result || '无'}
                      </pre>
                    </div>

                    {detailRequirement.agent_runtime?.last_error && (
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
                          {detailRequirement.agent_runtime.last_error}
                        </pre>
                      </div>
                    )}
                  </div>
                ),
              },
              {
                key: 'transition-history',
                label: '状态变更历史',
                children: (
                  <div>
                    {transitionHistory.length === 0 ? (
                      <div style={{ textAlign: 'center', color: '#999', padding: 40 }}>
                        暂无状态变更记录
                      </div>
                    ) : (
                      <Table
                        size="small"
                        dataSource={transitionHistory.map((log, index) => ({
                          key: index,
                          ...log,
                        }))}
                        columns={[
                          {
                            title: '序号',
                            dataIndex: 'key',
                            width: 60,
                            render: (_: unknown, __: unknown, index: number) => index + 1,
                          },
                          {
                            title: '从状态',
                            dataIndex: 'from_state',
                            width: 120,
                            render: (val: string) => <Tag>{val || '-'}</Tag>,
                          },
                          {
                            title: '到状态',
                            dataIndex: 'to_state',
                            width: 120,
                            render: (val: string) => <Tag color="blue">{val}</Tag>,
                          },
                          {
                            title: '触发方式',
                            dataIndex: 'trigger',
                            width: 100,
                          },
                          {
                            title: '触发者',
                            dataIndex: 'triggered_by',
                            width: 80,
                          },
                          {
                            title: '结果',
                            dataIndex: 'result',
                            width: 80,
                            render: (val: string) => (
                              <Tag color={val === 'success' ? 'green' : 'red'}>{val}</Tag>
                            ),
                          },
                          {
                            title: '说明',
                            dataIndex: 'remark',
                          },
                          {
                            title: '时间',
                            dataIndex: 'created_at',
                            width: 180,
                            render: (val: number) => new Date(val).toLocaleString(),
                          },
                        ]}
                        pagination={false}
                        scroll={{ y: 400 }}
                      />
                    )}
                  </div>
                ),
              },
            ]}
          />
        )}
      </Drawer>

      {/* 进度详情弹窗 */}
      <Modal
        title={
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', paddingRight: 32 }}>
            <span>进度详情 - {progressModalTitle}</span>
            <Button
              size="small"
              icon={<SyncOutlined spin={progressModalLoading} />}
              loading={progressModalLoading}
              onClick={refreshProgressModal}
            >
              刷新
            </Button>
          </div>
        }
        open={progressModalOpen}
        onCancel={() => setProgressModalOpen(false)}
        footer={null}
        width={560}
      >
        {progressModalData ? (
          <div style={{ opacity: progressModalLoading ? 0.6 : 1, transition: 'opacity 0.2s' }}>
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span style={{ fontSize: 14, color: '#666' }}>整体进度</span>
              <span style={{ fontSize: 16, fontWeight: 600, color: progressModalData.percent === 100 ? '#52c41a' : '#1890ff' }}>
                {progressModalData.percent}%
              </span>
            </div>
            <Progress percent={progressModalData.percent} size="small" strokeColor={progressModalData.percent === 100 ? '#52c41a' : '#1890ff'} />
            {progressModalData.updated_at && (
              <div style={{ marginTop: 4, marginBottom: 16, fontSize: 12, color: '#999', textAlign: 'right' }}>
                更新时间: {new Date(progressModalData.updated_at).toLocaleString()}
              </div>
            )}
            <List
              size="small"
              dataSource={progressModalData.items}
              renderItem={(todo: TodoItem) => {
                const statusLower = (todo.status || '').toLowerCase();
                const isDone = statusLower === 'completed' || statusLower === 'done';
                const isRunning = statusLower === 'in_progress' || statusLower === 'running' || statusLower === 'doing';
                const icon = isDone ? <CheckCircleOutlined style={{ color: '#52c41a' }} /> : isRunning ? <SyncOutlined spin style={{ color: '#1890ff' }} /> : <ClockCircleOutlined style={{ color: '#999' }} />;
                return (
                  <List.Item style={{ padding: '8px 0', borderBottom: '1px solid #f0f0f0' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 10, width: '100%' }}>
                      {icon}
                      <span style={{ flex: 1, fontSize: 14, textDecoration: isDone ? 'line-through' : 'none', color: isDone ? '#999' : '#333' }}>
                        {todo.content}
                      </span>
                      <Tag color={isDone ? 'success' : isRunning ? 'processing' : 'default'}>{todo.status || 'pending'}</Tag>
                      {todo.priority && <Tag color="warning">{todo.priority}</Tag>}
                    </div>
                  </List.Item>
                );
              }}
            />
          </div>
        ) : (
          <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>暂无进度数据</div>
        )}
      </Modal>

      {/* Trace Viewer 弹窗 */}
      <TraceViewer
        traceId={currentTraceId}
        visible={traceViewerVisible}
        onClose={() => setTraceViewerVisible(false)}
      />
    </div>
  );
};
