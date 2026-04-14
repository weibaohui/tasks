/**
 * useAgentManagement - Agent 管理页面逻辑
 */
import { useCallback, useEffect, useMemo, useState } from 'react';
import { Form, message } from 'antd';
import { createAgent, deleteAgent, listAgents, patchAgent, updateAgent } from '../../../api/agentApi';
import { listProviders } from '../../../api/providerApi';
import { createBinding, deleteBinding, getMCPErrorMessage, listBindings, listMCPServers, listMCPTools, updateBinding } from '../../../api/mcpApi';
import { listBuiltInTools } from '../../../api/toolsApi';
import { listSkillsSimple, type Skill } from '../../../api/skillApi';
import { useAuthStore } from '../../../stores/authStore';
import type { Agent, ClaudeCodeConfig, OpenCodeConfig, CreateAgentRequest, PatchAgentRequest, UpdateAgentRequest } from '../../../types/agent';
import type { LLMProvider } from '../../../types/provider';
import type { AgentMCPBinding, MCPServer, MCPTool } from '../../../types/mcp';
import type { BuiltInTool } from '../../../types/task';
import type { FormInstance } from 'antd/es/form';

import {
  DEFAULT_IDENTITY_CONTENT,
  DEFAULT_SOUL_CONTENT,
  DEFAULT_AGENTS_CONTENT,
  DEFAULT_USER_CONTENT,
  DEFAULT_TOOLS_CONTENT,
} from '../constants/agentDefaults';

export type AgentFormValues = {
  name: string;
  agent_type?: string;
  description?: string;
  identity_content: string;
  soul_content: string;
  agents_content: string;
  user_content: string;
  tools_content: string;
  model?: string;
  llm_provider_id?: string;
  max_tokens?: number;
  temperature?: number;
  max_iterations?: number;
  history_messages?: number;
  skills_list: string[];
  tools_list: string[];
  is_default: boolean;
  is_active: boolean;
  enable_thinking_process: boolean;
  claude_code_config?: ClaudeCodeConfig;
  opencode_config?: OpenCodeConfig;
};

export function getDefaultAgentFormValues(_defaultModel?: string): Partial<AgentFormValues> {
  return {
    name: undefined,
    agent_type: undefined,
    description: undefined,
    identity_content: DEFAULT_IDENTITY_CONTENT,
    soul_content: DEFAULT_SOUL_CONTENT,
    agents_content: DEFAULT_AGENTS_CONTENT,
    user_content: DEFAULT_USER_CONTENT,
    tools_content: DEFAULT_TOOLS_CONTENT,
    model: undefined,
    llm_provider_id: undefined,
    max_tokens: undefined,
    temperature: undefined,
    max_iterations: undefined,
    history_messages: undefined,
    skills_list: [],
    tools_list: [],
    is_default: false,
    is_active: true,
    enable_thinking_process: false,
  };
}

export interface UseAgentManagementOptions {
  form: FormInstance<AgentFormValues>;
  mcpForm: FormInstance<{ mcp_server_id: string; is_active: boolean; auto_load: boolean }>;
  toolsForm: FormInstance<{ all_tools: boolean; enabled_tools: string[] }>;
}

export interface UseAgentManagementReturn {
  // State
  items: Agent[];
  loading: boolean;
  saving: boolean;
  open: boolean;
  editing: Agent | null;
  providers: LLMProvider[];
  providersLoading: boolean;
  activeProviders: LLMProvider[];
  modelOptions: Array<{ value: string; label: string }>;
  claudeCodeModelOptions: Array<{ value: string; label: string }>;
  llmProviderOptions: Array<{ value: string; label: string }>;
  watchedModel: string | undefined;
  activeTab: 'basic' | 'skills' | 'personality' | 'claudecode' | 'opencode';
  mcpLoading: boolean;
  mcpServers: MCPServer[];
  mcpBindings: AgentMCPBinding[];
  toolsDrawerOpen: boolean;
  toolsDrawerLoading: boolean;
  toolsForServer: MCPTool[];
  editingBinding: AgentMCPBinding | null;
  builtInTools: BuiltInTool[];
  skillsOptions: Skill[];
  editingSections: Record<string, boolean>;
  savingSections: Record<string, boolean>;
  defaultModelFromProviders: string | undefined;

  // Actions
  fetchList: () => Promise<void>;
  openEditor: (agent: Agent | null) => Promise<void>;
  closeEditor: () => void;
  handleDelete: (id: string) => Promise<void>;
  handlePatchSection: (section: string, fields: PatchAgentRequest) => Promise<void>;
  handleSetDefault: (agent: Agent) => Promise<void>;
  handleToggleThinking: (agent: Agent, enabled: boolean) => Promise<void>;
  handleUpdateAgent: (id: string, fields: { name?: string; description?: string }) => Promise<void>;
  handleSubmit: () => Promise<void>;
  setActiveTab: (tab: 'basic' | 'skills' | 'personality' | 'claudecode' | 'opencode') => void;
  toggleSectionEdit: (section: string) => void;
  reloadMCP: () => Promise<void>;
  handleCreateBinding: (mcpServerId: string) => Promise<void>;
  handleUpdateBinding: (bindingId: string, fields: Record<string, unknown>) => Promise<void>;
  handleDeleteBinding: (bindingId: string) => Promise<void>;
  handleOpenToolsDrawer: (binding: AgentMCPBinding) => Promise<void>;
  handleCloseToolsDrawer: () => void;
  handleSaveTools: (enabledTools: string[]) => Promise<void>;
  setEditingSections: React.Dispatch<React.SetStateAction<Record<string, boolean>>>;
  setToolsDrawerOpen: (open: boolean) => void;
  setEditingBinding: (binding: AgentMCPBinding | null) => void;
  setToolsDrawerLoading: (loading: boolean) => void;
  setToolsForServer: (tools: MCPTool[]) => void;
  toolsForm: FormInstance<{ all_tools: boolean; enabled_tools: string[] }>;
  mcpForm: FormInstance<{ mcp_server_id: string; is_active: boolean; auto_load: boolean }>;
}

export function useAgentManagement({
  form,
  mcpForm,
  toolsForm,
}: UseAgentManagementOptions): UseAgentManagementReturn {
  const { user } = useAuthStore();
  const userCode = user?.user_code || '';

  // State
  const [items, setItems] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<Agent | null>(null);
  const [providers, setProviders] = useState<LLMProvider[]>([]);
  const [providersLoading, setProvidersLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<'basic' | 'skills' | 'personality' | 'claudecode' | 'opencode'>('basic');
  const [mcpLoading, setMcpLoading] = useState(false);
  const [mcpServers, setMcpServers] = useState<MCPServer[]>([]);
  const [mcpBindings, setMcpBindings] = useState<AgentMCPBinding[]>([]);
  const [toolsDrawerOpen, setToolsDrawerOpen] = useState(false);
  const [toolsDrawerLoading, setToolsDrawerLoading] = useState(false);
  const [toolsForServer, setToolsForServer] = useState<MCPTool[]>([]);
  const [editingBinding, setEditingBinding] = useState<AgentMCPBinding | null>(null);
  const [builtInTools, setBuiltInTools] = useState<BuiltInTool[]>([]);
  const [skillsOptions, setSkillsOptions] = useState<Skill[]>([]);
  const [editingSections, setEditingSections] = useState<Record<string, boolean>>({});
  const [savingSections, setSavingSections] = useState<Record<string, boolean>>({});

  const watchedModel = Form.useWatch('model', form);

  // Computed
  const activeProviders = useMemo(() => providers.filter((p) => p.is_active), [providers]);

  const defaultModelFromProviders = useMemo(() => {
    const defaultProvider = activeProviders.find((p) => p.is_default) || activeProviders[0];
    return defaultProvider?.default_model?.trim() || undefined;
  }, [activeProviders]);

  const modelOptionsFromProviders = useMemo(() => {
    const seen = new Set<string>();
    const opts: Array<{ value: string; label: string }> = [];

    for (const p of activeProviders) {
      const providerLabel = p.provider_name || p.provider_key || 'Provider';
      const candidates: string[] = [];
      if (p.default_model) candidates.push(p.default_model);
      for (const m of p.supported_models || []) {
        if (m?.name) candidates.push(m.name);
        else if (m?.id) candidates.push(m.id);
      }
      for (const model of candidates) {
        const value = model.trim();
        if (!value || seen.has(value)) continue;
        seen.add(value);
        opts.push({ value, label: `${value}（${providerLabel}）` });
      }
    }
    opts.sort((a, b) => a.value.localeCompare(b.value));
    return opts;
  }, [activeProviders]);

  const modelOptions = useMemo(() => {
    if (!watchedModel || modelOptionsFromProviders.some((o) => o.value === watchedModel)) {
      return modelOptionsFromProviders;
    }
    return [{ value: watchedModel, label: watchedModel }, ...modelOptionsFromProviders];
  }, [modelOptionsFromProviders, watchedModel]);

  // ClaudeCode model options - use all active providers
  const claudeCodeModelOptions = useMemo(() => {
    const seen = new Set<string>();
    const opts: Array<{ value: string; label: string }> = [];

    for (const p of activeProviders) {
      const providerLabel = p.provider_name || p.provider_key || 'Provider';
      const candidates: string[] = [];
      if (p.default_model) candidates.push(p.default_model);
      for (const m of p.supported_models || []) {
        if (m?.name) candidates.push(m.name);
        else if (m?.id) candidates.push(m.id);
      }
      for (const model of candidates) {
        const value = model.trim();
        if (!value || seen.has(value)) continue;
        seen.add(value);
        opts.push({ value, label: `${value}（${providerLabel}）` });
      }
    }
    opts.sort((a, b) => a.value.localeCompare(b.value));
    return opts;
  }, [activeProviders]);

  // LLM Provider options for selection (by id)
  const llmProviderOptions = useMemo(() => {
    return activeProviders.map((p) => ({
      value: p.id,
      label: `${p.provider_name || p.provider_key} (${p.provider_key})`,
    }));
  }, [activeProviders]);

  // Actions
  const fetchList = useCallback(async () => {
    if (!userCode) { setItems([]); return; }
    setLoading(true);
    try {
      setItems(await listAgents(userCode));
    } catch { message.error('获取 Agent 列表失败'); }
    finally { setLoading(false); }
  }, [userCode]);

  const fetchProviders = useCallback(async () => {
    if (!userCode) { setProviders([]); return; }
    setProvidersLoading(true);
    try {
      setProviders(await listProviders(userCode));
    } catch { message.error('获取 LLM 配置失败'); setProviders([]); }
    finally { setProvidersLoading(false); }
  }, [userCode]);

  const buildUpdateRequestFromAgent = useCallback((agent: Agent, overrides: Partial<UpdateAgentRequest>): UpdateAgentRequest => ({
    name: agent.name, agent_type: agent.agent_type, description: agent.description,
    identity_content: agent.identity_content, soul_content: agent.soul_content,
    agents_content: agent.agents_content, user_content: agent.user_content,
    tools_content: agent.tools_content, model: agent.model,
    llm_provider_id: agent.llm_provider_id || '',
    max_tokens: agent.max_tokens, temperature: agent.temperature, max_iterations: agent.max_iterations,
    history_messages: agent.history_messages, skills_list: agent.skills_list || [],
    tools_list: agent.tools_list || [], is_active: agent.is_active,
    is_default: agent.is_default, enable_thinking_process: agent.enable_thinking_process,
    ...overrides,
  }), []);

  const reloadMCP = useCallback(async (agentId?: string) => {
    const id = agentId || editing?.id;
    if (!id) return;
    setMcpLoading(true);
    try {
      const [servers, bindings] = await Promise.all([listMCPServers(), listBindings(id)]);
      setMcpServers(servers);
      setMcpBindings(bindings);
    } catch (e) { message.error(getMCPErrorMessage(e) || '获取 MCP 绑定信息失败'); }
    finally { setMcpLoading(false); }
  }, [editing]);

  const openEditor = useCallback(async (agent: Agent | null) => {
    setEditing(agent);
    const isCoding = agent?.agent_type === 'CodingAgent';
    const isOpenCode = agent?.agent_type === 'OpenCodeAgent';
    setActiveTab(isCoding ? 'claudecode' : isOpenCode ? 'opencode' : 'basic');
    setEditingBinding(null);
    setToolsDrawerOpen(false);
    setToolsForServer([]);

    if (agent) {
      // Set form values BEFORE opening drawer so values are ready when form mounts
      form.setFieldsValue({
        name: agent.name, agent_type: agent.agent_type || 'BareLLM',
        description: agent.description, identity_content: agent.identity_content,
        soul_content: agent.soul_content, agents_content: agent.agents_content,
        user_content: agent.user_content, tools_content: agent.tools_content,
        model: agent.model, llm_provider_id: agent.llm_provider_id || '',
        max_tokens: agent.max_tokens, temperature: agent.temperature,
        max_iterations: agent.max_iterations, history_messages: agent.history_messages,
        skills_list: agent.skills_list || [], tools_list: agent.tools_list || [],
        is_default: agent.is_default, is_active: agent.is_active,
        enable_thinking_process: agent.enable_thinking_process,
        claude_code_config: agent.claude_code_config,
        opencode_config: agent.opencode_config,
      });
      // Pass agent.id explicitly to avoid stale closure issue
      await reloadMCP(agent.id);
    } else {
      try { setMcpServers(await listMCPServers()); }
      catch (e) { message.error(getMCPErrorMessage(e) || '获取 MCP 服务器列表失败'); }
      setMcpBindings([]);
    }

    // Open drawer AFTER setting form values
    setOpen(true);
  }, [form, reloadMCP]);

  const closeEditor = useCallback(() => {
    setOpen(false); setEditing(null); setActiveTab('basic');
    form.resetFields(); mcpForm.resetFields();
    setMcpBindings([]); setEditingBinding(null); setToolsDrawerOpen(false);
    setEditingSections({});
  }, [form, mcpForm]);

  const toggleSectionEdit = useCallback((section: string) => {
    setEditingSections((prev) => ({ ...prev, [section]: !prev[section] }));
  }, []);

  const handlePatchSection = useCallback(async (section: string, fields: PatchAgentRequest) => {
    if (!editing) return;
    setSavingSections((prev) => ({ ...prev, [section]: true }));
    try {
      await patchAgent(editing.id, fields);
      message.success('保存成功');
      setEditingSections((prev) => ({ ...prev, [section]: false }));
      await fetchList();
      const updated = await listAgents(userCode);
      const found = updated.find((a) => a.id === editing.id);
      if (found) {
        setEditing(found);
        form.setFieldsValue({
          name: found.name, agent_type: found.agent_type || 'BareLLM',
          description: found.description, identity_content: found.identity_content,
          soul_content: found.soul_content, agents_content: found.agents_content,
          user_content: found.user_content, tools_content: found.tools_content,
          model: found.model, llm_provider_id: found.llm_provider_id || '',
          max_tokens: found.max_tokens, temperature: found.temperature,
          max_iterations: found.max_iterations, history_messages: found.history_messages,
          skills_list: found.skills_list || [], tools_list: found.tools_list || [],
          is_default: found.is_default, is_active: found.is_active,
          enable_thinking_process: found.enable_thinking_process,
          claude_code_config: found.claude_code_config,
          opencode_config: found.opencode_config,
        });
      }
    } catch { message.error('保存失败'); }
    finally { setSavingSections((prev) => ({ ...prev, [section]: false })); }
  }, [editing, fetchList, form, userCode]);

  const handleSetDefault = useCallback(async (agent: Agent) => {
    try {
      await updateAgent(agent.id, buildUpdateRequestFromAgent(agent, { is_default: true }));
      message.success('已设为默认 Agent');
      await fetchList();
    } catch { message.error('设置默认失败'); }
  }, [buildUpdateRequestFromAgent, fetchList]);

  const handleToggleThinking = useCallback(async (agent: Agent, enabled: boolean) => {
    try {
      await updateAgent(agent.id, buildUpdateRequestFromAgent(agent, { enable_thinking_process: enabled }));
      message.success(enabled ? '已开启思考过程' : '已关闭思考过程');
      await fetchList();
    } catch { message.error('更新失败'); }
  }, [buildUpdateRequestFromAgent, fetchList]);

  const handleUpdateAgent = useCallback(async (id: string, fields: { name?: string; description?: string }) => {
    try {
      await patchAgent(id, { name: fields.name, description: fields.description });
      message.success('更新成功');
      await fetchList();
    } catch { message.error('更新失败'); }
  }, [fetchList]);

  const handleDelete = useCallback(async (id: string) => {
    try { await deleteAgent(id); message.success('删除成功'); await fetchList(); }
    catch { message.error('删除失败'); }
  }, [fetchList]);

  const handleSubmit = useCallback(async () => {
    if (!userCode) { message.error('未获取到用户信息，请重新登录'); return; }
    setSaving(true);
    try {
      const values = form.getFieldsValue() as AgentFormValues;
      // 根据 Agent 类型清理不相关的配置，避免脏数据提交
      const isCoding = values.agent_type === 'CodingAgent';
      const isOpenCode = values.agent_type === 'OpenCodeAgent';
      const claudeCodeConfig = isCoding ? values.claude_code_config : undefined;
      const opencodeConfig = isOpenCode ? values.opencode_config : undefined;

      if (editing) {
        await updateAgent(editing.id, {
          name: values.name, agent_type: values.agent_type, description: values.description,
          identity_content: values.identity_content || '', soul_content: values.soul_content || '',
          agents_content: values.agents_content || '', user_content: values.user_content || '',
          tools_content: values.tools_content || '', model: values.model ?? '',
          llm_provider_id: values.llm_provider_id,
          max_tokens: values.max_tokens, temperature: values.temperature,
          max_iterations: values.max_iterations, history_messages: values.history_messages,
          skills_list: values.skills_list || [], tools_list: values.tools_list || [],
          is_default: values.is_default, is_active: values.is_active,
          enable_thinking_process: values.enable_thinking_process,
          claude_code_config: claudeCodeConfig,
          opencode_config: opencodeConfig,
        } as UpdateAgentRequest);
        message.success('更新成功');
      } else {
        await createAgent({
          user_code: userCode, name: values.name, agent_type: values.agent_type,
          description: values.description, identity_content: values.identity_content,
          soul_content: values.soul_content, agents_content: values.agents_content,
          user_content: values.user_content, tools_content: values.tools_content,
          model: values.model ?? '',
          llm_provider_id: values.llm_provider_id,
          max_tokens: values.max_tokens, temperature: values.temperature,
          max_iterations: values.max_iterations, history_messages: values.history_messages,
          skills_list: values.skills_list || [], tools_list: values.tools_list || [],
          is_default: values.is_default, enable_thinking_process: values.enable_thinking_process,
          claude_code_config: claudeCodeConfig,
          opencode_config: opencodeConfig,
        } as CreateAgentRequest);
        message.success('创建成功');
      }
      closeEditor(); await fetchList();
    } catch { message.error('保存失败'); }
    finally { setSaving(false); }
  }, [userCode, editing, form, closeEditor, fetchList]);

  const handleCreateBinding = useCallback(async (mcpServerId: string) => {
    if (!editing) return;
    try {
      await createBinding({ agent_id: editing.id, mcp_server_id: mcpServerId, is_active: true, auto_load: false });
      message.success('绑定成功');
      mcpForm.resetFields();
      await reloadMCP();
    } catch (e) { message.error(getMCPErrorMessage(e) || '绑定失败'); }
  }, [editing, mcpForm, reloadMCP]);

  const handleUpdateBinding = useCallback(async (bindingId: string, fields: Record<string, unknown>) => {
    try {
      await updateBinding(bindingId, fields);
      message.success('更新成功');
      await reloadMCP();
    } catch (e) { message.error(getMCPErrorMessage(e) || '操作失败'); }
  }, [reloadMCP]);

  const handleDeleteBinding = useCallback(async (bindingId: string) => {
    try { await deleteBinding(bindingId); message.success('解绑成功'); await reloadMCP(); }
    catch (e) { message.error(getMCPErrorMessage(e) || '解绑失败'); }
  }, [reloadMCP]);

  const handleOpenToolsDrawer = useCallback(async (binding: AgentMCPBinding) => {
    setEditingBinding(binding);
    setToolsDrawerOpen(true);
    setToolsDrawerLoading(true);
    try {
      const tools = await listMCPTools(binding.mcp_server_id);
      setToolsForServer(tools);
      const current = binding.enabled_tools || [];
      toolsForm.setFieldsValue({ all_tools: current.length === 0, enabled_tools: current });
    } catch (e) { setToolsForServer([]); message.error(getMCPErrorMessage(e) || '获取工具列表失败'); }
    finally { setToolsDrawerLoading(false); }
  }, [toolsForm]);

  const handleCloseToolsDrawer = useCallback(() => {
    setToolsDrawerOpen(false); setEditingBinding(null); setToolsForServer([]);
    toolsForm.resetFields();
  }, [toolsForm]);

  const handleSaveTools = useCallback(async (enabledTools: string[]) => {
    if (!editingBinding) return;
    try {
      await updateBinding(editingBinding.id, { enabled_tools: enabledTools });
      message.success('已更新工具配置');
      await reloadMCP();
      handleCloseToolsDrawer();
    } catch (e) { message.error(getMCPErrorMessage(e) || '保存失败'); }
  }, [editingBinding, reloadMCP, handleCloseToolsDrawer]);

  // Effects
  useEffect(() => { fetchList(); }, [fetchList]);
  useEffect(() => { fetchProviders(); }, [fetchProviders]);
  useEffect(() => { listBuiltInTools().then(setBuiltInTools).catch(() => message.error('获取内置工具列表失败')); }, []);
  useEffect(() => { listSkillsSimple().then(setSkillsOptions).catch(() => message.error('获取技能列表失败')); }, []);

  // Auto-update tools list based on skills and MCP bindings
  useEffect(() => {
    if (!open || !editing) return;
    const currentTools = (form.getFieldValue('tools_list') as string[]) || [];
    const toolsSet = new Set(currentTools);
    let changed = false;

    const skills = (form.getFieldValue('skills_list') as string[]) || [];
    if (skills.length > 0) {
      if (!toolsSet.has('use_skill')) { toolsSet.add('use_skill'); changed = true; }
    } else {
      if (toolsSet.has('use_skill')) { toolsSet.delete('use_skill'); changed = true; }
    }

    const hasActiveMCP = mcpBindings.length > 0 && mcpBindings.some((b) => b.is_active);
    if (hasActiveMCP) {
      if (!toolsSet.has('use_mcp')) { toolsSet.add('use_mcp'); changed = true; }
      if (!toolsSet.has('call_mcp_tool')) { toolsSet.add('call_mcp_tool'); changed = true; }
    } else {
      if (toolsSet.has('use_mcp')) { toolsSet.delete('use_mcp'); changed = true; }
      if (toolsSet.has('call_mcp_tool')) { toolsSet.delete('call_mcp_tool'); changed = true; }
    }

    if (changed) form.setFieldsValue({ tools_list: Array.from(toolsSet) });
  }, [open, editing, mcpBindings, form]);

  return {
    items, loading, saving, open, editing, providers, providersLoading,
    activeProviders, modelOptions, claudeCodeModelOptions, llmProviderOptions, watchedModel, activeTab,
    mcpLoading, mcpServers, mcpBindings,
    toolsDrawerOpen, toolsDrawerLoading, toolsForServer, editingBinding,
    builtInTools, skillsOptions, editingSections, savingSections,
    defaultModelFromProviders,
    fetchList, openEditor, closeEditor, handleDelete, handlePatchSection,
    handleSetDefault, handleToggleThinking, handleUpdateAgent, handleSubmit,
    setActiveTab, toggleSectionEdit, reloadMCP, handleCreateBinding,
    handleUpdateBinding, handleDeleteBinding, handleOpenToolsDrawer,
    handleCloseToolsDrawer, handleSaveTools, setEditingSections,
    setToolsDrawerOpen, setEditingBinding, setToolsDrawerLoading, setToolsForServer,
    toolsForm, mcpForm,
  };
}
