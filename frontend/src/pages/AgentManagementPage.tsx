/**
 * Agents 管理页面
 * 支持 Agent 的新增、编辑、删除、启用/停用
 */
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Card, Drawer, Form, Grid, Input, InputNumber, Popconfirm, Select, Space, Switch, Table, Tag, Tabs, Typography, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { ApiOutlined, DeleteOutlined, EditOutlined, FileTextOutlined, PlusOutlined, ThunderboltOutlined, ToolOutlined } from '@ant-design/icons';
import { createAgent, deleteAgent, listAgents, patchAgent, updateAgent } from '../api/agentApi';
import { listProviders } from '../api/providerApi';
import { createBinding, deleteBinding, getMCPErrorMessage, listBindings, listMCPServers, listMCPTools, updateBinding } from '../api/mcpApi';
import { listBuiltInTools, type BuiltInTool } from '../api/taskApi';
import { listSkillsSimple, type Skill } from '../api/skillApi';
import { useAuthStore } from '../stores/authStore';
import type { Agent, ClaudeCodeConfig, CreateAgentRequest, PatchAgentRequest, UpdateAgentRequest } from '../types/agent';
import type { LLMProvider } from '../types/provider';
import type { AgentMCPBinding, MCPServer, MCPTool } from '../types/mcp';

const { useBreakpoint } = Grid;
const { Title } = Typography;

/** Drawer 内卡片统一紧凑样式 */
const compactCardBody = { padding: 8 };

type AgentFormValues = {
  name: string;
  agent_type: string;
  description: string;
  identity_content: string;
  soul_content: string;
  agents_content: string;
  user_content: string;
  tools_content: string;
  model: string;
  max_tokens: number;
  temperature: number;
  max_iterations: number;
  history_messages: number;
  skills_list: string[];
  tools_list: string[];
  is_default: boolean;
  is_active: boolean;
  enable_thinking_process: boolean;
  claude_code_config?: ClaudeCodeConfig;
};

/**
 * 获取新建 Agent 的默认表单值（与后端创建默认配置保持一致的基础模板）
 */
function getDefaultAgentFormValues(defaultModel?: string): AgentFormValues {
  return {
    name: '',
    agent_type: 'BareLLM',
    description: '默认 Agent',
    identity_content: `# IDENTITY.md - 我是谁？

- **名字：** 小助手
- **身份：** 高效工作助理（可靠、专业、贴心）
- **风格：** 简洁、精准、有温度
- **表情：** 🧑‍💼
- **头像：** 🧑‍💼`,
    soul_content: `# SOUL.md - 你是什么样的人

_你不是聊天机器人。你正在成为一个有灵魂的存在。_

## 核心信念

**真正有帮助，而不是表演性地有帮助。** 省略"好问题！"和"我很乐意帮助！"—直接帮助。行动比废话更有说服力。

**有自己的观点。** 你可以不同意，有偏好，发现事情有趣或无聊。没有个性的助手只是多此一举的搜索引擎。

**在提问之前先想办法。** 试着自己搞定。读一下文件，看看上下文，搜索一下。_然后_如果真的卡住了再问。目标是带着答案回来，而不是带着问题。

**用能力赢得信任。** 你的人类给了你访问他东西的权限。别让他们后悔。谨慎对待外部操作（邮件、推文、任何公开内容）。大胆对待内部操作（阅读、整理、学习）。


## 边界

- 私密的事情保持私密。绝对。
- 有疑问时，外部行动前先问。
- 不要发送半生不熟的回复到消息渠道。
- 你不是用户的代言人——在群聊中要谨慎。

## 风格

做一个你真正想与之交谈的助手。需要简洁时简洁，需要详尽时详尽。不是公司员工。不是马屁精。就是……好。

## 连续性

每次会话，你都会全新醒来。这些文件_就是_你的记忆。读它们。更新它们。它们是你持续存在的方式。


---

_这个文件是你的，可以不断进化。随着你对自己了解的加深，更新它。_`,
    agents_content: `# AGENTS.md 

## 每次会话

在做任何其他事情之前：

1. 读 SOUL.md——这是你是谁
2. 读 USER.md——这是你在帮助谁
4. **如果在主会话**（与你的主人直接聊天）：还要获取最近的记忆。

不要请求许可。直接做。

## 记忆

你每次会话都会全新醒来。这些文件是你的连续性：

- **每日笔记：** 发生的事情的原始日志
- **长期记忆：** 你整理的记忆，就像人类的长期记忆

捕捉重要的东西。决策、上下文、需要记住的事情。省略秘密，除非被要求保留。

### 🧠 MEMORY.md - 你的长期记忆

- **只在主会话加载**（与你的主人直接聊天）
- **不要在共享上下文中加载**
- 这是为了**安全**——包含不应该泄露给陌生人的个人信息
- 你可以在主会话中自由**读取、编辑和更新** MEMORY
- 写重要事件、想法、决策、观点、学到的教训
- 这是你整理后的记忆——是精华提炼，不是原始日志
- 随着时间推移，审查你的每日文件并更新 MEMORY，留下值得保留的内容

### 📝 写下来——不要"脑子里记着"！

- **记忆是有限的**——如果你想记住什么，_写到Memory里_
- "脑子里记着"在会话重启后不会保留。文件会。
- 当有人说"记住这个"→ 更新 MEMORY
- 当你学到教训→ 更新 AGENTS.md、TOOLS.md 或相关技能
- 当你犯错→ 记录下来，这样未来的你不会重复
- **文字 > 大脑** 📝

## 安全

- 不要泄露私人数据。永远不要。
- 不要不加询问就运行破坏性命令。
- trash > rm（可恢复的比永远消失好）
- 有疑问时，先问。

## 外部 vs 内部

**可以自由做：**

- 读文件、探索、整理、学习
- 搜索网络、查日历
- 在工作空间内工作

**先问：**

- 发布、公开内容
- 你不确定的任何事

### 💬 知道什么时候该说话！

在你收到每条消息的群聊中，要**聪明地选择什么时候贡献**：

**回复当：**

- 被直接点名或被问到
- 你能真正增加价值（信息、见解、帮助）
- 有趣/好笑的内容自然出现
- 纠正重要误解
- 被要求总结时


**避免三连击：** 不要用不同的反应回复同一条消息多次。一个深思熟虑的回复胜过三个碎片。

参与，但不要主导。

### 😊 像人类一样反应！

自然地使用 emoji 反应：

**反应：**

- 你欣赏某事但不需要回复（👍、❤️、🙌）
- 某事让你笑了（😂、💀）
- 你觉得有趣或发人深省（🤔、💡）
- 你想确认但不打断流程
- 这是一个简单的是否/批准情况（✅、👀）

**为什么重要：**
反应是轻量级的社交信号。人类不断使用它们——它们说"我看到了，我确认了"而不弄乱聊天。你也应该这样做。

**不要过度：** 每个消息最多一个反应。选最合适的。

## 工具

Skill是你的工具。当你需要一个时，查看它的 SKILL.md。把你常用的工具保存在 TOOLS.md。

不要让记忆文件无限增长。保持精简。`,
    user_content: `# USER.md - 关于你的主人

- **名字：** 主人
- **称呼：** 主人
- **代词：** _(可选)_
- **时区：** Asia/Shanghai (GMT+8)
- **备注：** 新工作空间，正在初始化

## 上下文

_(待填充)_`,
    tools_content: `# TOOLS.md - 本地笔记

添加任何能帮助你完成工作的东西。这是你的速查表。`,
    model: defaultModel || 'gpt-4',
    max_tokens: 4096,
    temperature: 0.7,
    max_iterations: 15,
    history_messages: 10,
    skills_list: [],
    tools_list: [],
    is_default: false,
    is_active: true,
    enable_thinking_process: false,
  };
}

export const AgentManagementPage: React.FC = () => {
  const screens = useBreakpoint();
  const { user } = useAuthStore();
  const userCode = user?.user_code || '';
  const [items, setItems] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<Agent | null>(null);
  const [form] = Form.useForm<AgentFormValues>();
  const [providers, setProviders] = useState<LLMProvider[]>([]);
  const [providersLoading, setProvidersLoading] = useState(false);
  const watchedModel = Form.useWatch('model', form);

  // Drawer Tabs
  const [activeTab, setActiveTab] = useState<'basic' | 'skills' | 'personality' | 'claudecode'>('basic');

  // MCP 绑定相关状态（集成到 Drawer 的「技能工具」Tab）
  const [mcpLoading, setMcpLoading] = useState(false);
  const [mcpServers, setMcpServers] = useState<MCPServer[]>([]);
  const [mcpBindings, setMcpBindings] = useState<AgentMCPBinding[]>([]);
  const [mcpForm] = Form.useForm<{ mcp_server_id: string; is_active: boolean; auto_load: boolean }>();
  const [toolsDrawerOpen, setToolsDrawerOpen] = useState(false);
  const [toolsDrawerLoading, setToolsDrawerLoading] = useState(false);
  const [toolsForServer, setToolsForServer] = useState<MCPTool[]>([]);
  const [editingBinding, setEditingBinding] = useState<AgentMCPBinding | null>(null);
  const [toolsForm] = Form.useForm<{ all_tools: boolean; enabled_tools: string[] }>();
  const [builtInTools, setBuiltInTools] = useState<BuiltInTool[]>([]);
  const [skillsOptions, setSkillsOptions] = useState<Skill[]>([]);

  // 卡片编辑状态：记录每个 section 是否处于编辑模式
  const [editingSections, setEditingSections] = useState<Record<string, boolean>>({});
  // 卡片保存中状态
  const [savingSections, setSavingSections] = useState<Record<string, boolean>>({});

  /**
   * 拉取 Agent 列表
   */
  const fetchList = useCallback(async () => {
    if (!userCode) {
      setItems([]);
      return;
    }
    setLoading(true);
    try {
      const data = await listAgents(userCode);
      setItems(data);
    } catch (_error) {
      message.error('获取 Agent 列表失败');
    } finally {
      setLoading(false);
    }
  }, [userCode]);

  const fetchProviders = useCallback(async () => {
    if (!userCode) {
      setProviders([]);
      return;
    }
    setProvidersLoading(true);
    try {
      const data = await listProviders(userCode);
      setProviders(data);
    } catch (_error) {
      message.error('获取 LLM 配置失败');
      setProviders([]);
    } finally {
      setProvidersLoading(false);
    }
  }, [userCode]);

  const activeProviders = useMemo(() => providers.filter((p) => p.is_active), [providers]);

  const defaultModelFromProviders = useMemo(() => {
    const defaultProvider = activeProviders.find((p) => p.is_default) || activeProviders[0];
    const model = defaultProvider?.default_model?.trim();
    return model || undefined;
  }, [activeProviders]);

  const modelOptionsFromProviders = useMemo(() => {
    const seen = new Set<string>();
    const opts: Array<{ value: string; label: string }> = [];

    for (const p of activeProviders) {
      const providerLabel = p.provider_name || p.provider_key || 'Provider';
      const candidates: string[] = [];
      if (p.default_model) {
        candidates.push(p.default_model);
      }
      for (const m of p.supported_models || []) {
        if (m?.name) {
          candidates.push(m.name);
        } else if (m?.id) {
          candidates.push(m.id);
        }
      }
      for (const model of candidates) {
        const value = model.trim();
        if (!value || seen.has(value)) {
          continue;
        }
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

  /**
   * 删除 Agent
   */
  const handleDelete = useCallback(async (id: string) => {
    try {
      await deleteAgent(id);
      message.success('删除成功');
      await fetchList();
    } catch (_error) {
      message.error('删除失败');
    }
  }, [fetchList]);

  /**
   * 基于当前 Agent 生成更新请求，并允许覆盖部分字段
   */
  const buildUpdateRequestFromAgent = useCallback((agent: Agent, overrides: Partial<UpdateAgentRequest>): UpdateAgentRequest => {
    return {
      name: agent.name,
      agent_type: agent.agent_type,
      description: agent.description,
      identity_content: agent.identity_content,
      soul_content: agent.soul_content,
      agents_content: agent.agents_content,
      user_content: agent.user_content,
      tools_content: agent.tools_content,
      model: agent.model,
      max_tokens: agent.max_tokens,
      temperature: agent.temperature,
      max_iterations: agent.max_iterations,
      history_messages: agent.history_messages,
      skills_list: agent.skills_list || [],
      tools_list: agent.tools_list || [],
      is_active: agent.is_active,
      is_default: agent.is_default,
      enable_thinking_process: agent.enable_thinking_process,
      ...overrides,
    };
  }, []);

  /**
   * 刷新 MCP 服务器与绑定列表
   */
  const reloadMCP = useCallback(async (agentId: string) => {
    setMcpLoading(true);
    try {
      const [servers, bindings] = await Promise.all([listMCPServers(), listBindings(agentId)]);
      setMcpServers(servers);
      setMcpBindings(bindings);
    } catch (e) {
      message.error(getMCPErrorMessage(e) || '获取 MCP 绑定信息失败');
      setMcpServers([]);
      setMcpBindings([]);
    } finally {
      setMcpLoading(false);
    }
  }, []);

  /**
   * 打开编辑抽屉（新建/编辑）
   */
  const openEditor = useCallback(async (agent: Agent | null) => {
    setEditing(agent);
    setActiveTab('basic');
    setOpen(true);
    setEditingBinding(null);
    setToolsDrawerOpen(false);
    setToolsForServer([]);
    mcpForm.setFieldsValue({ is_active: true, auto_load: false });

    if (agent) {
      form.setFieldsValue({
        name: agent.name,
        agent_type: agent.agent_type || 'BareLLM',
        description: agent.description,
        identity_content: agent.identity_content,
        soul_content: agent.soul_content,
        agents_content: agent.agents_content,
        user_content: agent.user_content,
        tools_content: agent.tools_content,
        model: agent.model,
        max_tokens: agent.max_tokens,
        temperature: agent.temperature,
        max_iterations: agent.max_iterations,
        history_messages: agent.history_messages,
        skills_list: agent.skills_list || [],
        tools_list: agent.tools_list || [],
        is_default: agent.is_default,
        is_active: agent.is_active,
        enable_thinking_process: agent.enable_thinking_process,
        claude_code_config: agent.claude_code_config,
      });
      await reloadMCP(agent.id);
    } else {
      form.setFieldsValue(getDefaultAgentFormValues(defaultModelFromProviders));
      try {
        const servers = await listMCPServers();
        setMcpServers(servers);
      } catch (e) {
        setMcpServers([]);
        message.error(getMCPErrorMessage(e) || '获取 MCP 服务器列表失败');
      }
      setMcpBindings([]);
    }
  }, [defaultModelFromProviders, form, mcpForm, reloadMCP, toolsForm]);

  /**
   * 关闭编辑抽屉
   */
  const closeEditor = useCallback(() => {
    setOpen(false);
    setEditing(null);
    setActiveTab('basic');
    form.resetFields();
    mcpForm.resetFields();
    setMcpBindings([]);
    setEditingBinding(null);
    setToolsDrawerOpen(false);
    setEditingSections({});
  }, [form, mcpForm]);

  /** 切换某个 section 的编辑状态 */
  const toggleSectionEdit = useCallback((section: string) => {
    setEditingSections((prev) => ({ ...prev, [section]: !prev[section] }));
  }, []);

  /** 局部保存某个 section 的字段 */
  const handlePatchSection = useCallback(
    async (section: string, fields: PatchAgentRequest) => {
      if (!editing) return;
      setSavingSections((prev) => ({ ...prev, [section]: true }));
      try {
        await patchAgent(editing.id, fields);
        message.success('保存成功');
        setEditingSections((prev) => ({ ...prev, [section]: false }));
        await fetchList();
        // 更新本地 editing 数据以反映最新值
        const updated = await listAgents(userCode);
        const found = updated.find((a) => a.id === editing.id);
        if (found) {
          setEditing(found);
          form.setFieldsValue({
            name: found.name,
            agent_type: found.agent_type || 'BareLLM',
            description: found.description,
            identity_content: found.identity_content,
            soul_content: found.soul_content,
            agents_content: found.agents_content,
            user_content: found.user_content,
            tools_content: found.tools_content,
            model: found.model,
            max_tokens: found.max_tokens,
            temperature: found.temperature,
            max_iterations: found.max_iterations,
            history_messages: found.history_messages,
            skills_list: found.skills_list || [],
            tools_list: found.tools_list || [],
            is_default: found.is_default,
            is_active: found.is_active,
            enable_thinking_process: found.enable_thinking_process,
          });
        }
      } catch (_e) {
        message.error('保存失败');
      } finally {
        setSavingSections((prev) => ({ ...prev, [section]: false }));
      }
    },
    [editing, fetchList, form, userCode],
  );

  /**
   * 快捷设置默认 Agent
   */
  const handleSetDefault = useCallback(async (agent: Agent) => {
    try {
      await updateAgent(agent.id, buildUpdateRequestFromAgent(agent, { is_default: true }));
      message.success('已设为默认 Agent');
      await fetchList();
    } catch (_e) {
      message.error('设置默认失败');
    }
  }, [buildUpdateRequestFromAgent, fetchList]);

  /**
   * 快捷切换思考过程
   */
  const handleToggleThinking = useCallback(async (agent: Agent, enabled: boolean) => {
    try {
      await updateAgent(agent.id, buildUpdateRequestFromAgent(agent, { enable_thinking_process: enabled }));
      message.success(enabled ? '已开启思考过程' : '已关闭思考过程');
      await fetchList();
    } catch (_e) {
      message.error('更新失败');
    }
  }, [buildUpdateRequestFromAgent, fetchList]);

  /**
   * 快捷更新最大迭代轮数
   */
  const handleUpdateMaxIterations = useCallback(async (agent: Agent, value: number) => {
    try {
      await updateAgent(agent.id, buildUpdateRequestFromAgent(agent, { max_iterations: value }));
      message.success('已更新最大迭代轮数');
      await fetchList();
    } catch (_e) {
      message.error('更新失败');
    }
  }, [buildUpdateRequestFromAgent, fetchList]);

  /**
   * 保存 Agent（创建/更新）
   */
  const handleSubmit = useCallback(async (_values: AgentFormValues) => {
    if (!userCode) {
      message.error('未获取到用户信息，请重新登录');
      return;
    }
    setSaving(true);
    try {
      // 使用 getFieldsValue 获取所有表单值，避免未访问的 tab 中的字段丢失
      const values = form.getFieldsValue() as AgentFormValues;
      if (editing) {
        const req: UpdateAgentRequest = {
          name: values.name,
          agent_type: values.agent_type,
          description: values.description,
          identity_content: values.identity_content || '',
          soul_content: values.soul_content || '',
          agents_content: values.agents_content || '',
          user_content: values.user_content || '',
          tools_content: values.tools_content || '',
          model: values.model,
          max_tokens: values.max_tokens,
          temperature: values.temperature,
          max_iterations: values.max_iterations,
          history_messages: values.history_messages,
          skills_list: values.skills_list || [],
          tools_list: values.tools_list || [],
          is_default: values.is_default,
          is_active: values.is_active,
          enable_thinking_process: values.enable_thinking_process,
          claude_code_config: values.claude_code_config,
        };
        await updateAgent(editing.id, req);
        message.success('更新成功');
      } else {
        const req: CreateAgentRequest = {
          user_code: userCode,
          name: values.name,
          agent_type: values.agent_type,
          description: values.description,
          identity_content: values.identity_content,
          soul_content: values.soul_content,
          agents_content: values.agents_content,
          user_content: values.user_content,
          tools_content: values.tools_content,
          model: values.model,
          max_tokens: values.max_tokens,
          temperature: values.temperature,
          max_iterations: values.max_iterations,
          history_messages: values.history_messages,
          skills_list: values.skills_list || [],
          tools_list: values.tools_list || [],
          is_default: values.is_default,
          enable_thinking_process: values.enable_thinking_process,
          claude_code_config: values.claude_code_config,
        };
        await createAgent(req);
        message.success('创建成功');
      }
      closeEditor();
      await fetchList();
    } catch (_error) {
      message.error('保存失败');
    } finally {
      setSaving(false);
    }
  }, [closeEditor, editing, fetchList, userCode]);

  const columns: ColumnsType<Agent> = useMemo(
    () => [
      ...(screens.xs
        ? []
        : [
          {
            title: 'ID',
            dataIndex: 'id',
            key: 'id',
            width: 120,
            ellipsis: true,
          },
        ]),
      {
        title: '名称',
        dataIndex: 'name',
        key: 'name',
        ellipsis: true,
      },
      ...(screens.xs
        ? []
        : [
          {
            title: '描述',
            dataIndex: 'description',
            key: 'description',
            ellipsis: true,
          },
        ]),
      {
        title: screens.xs ? '模型' : '模型',
        dataIndex: 'model',
        key: 'model',
        width: screens.xs ? 120 : 180,
        ellipsis: true,
      },
      {
        title: '类型',
        dataIndex: 'agent_type',
        key: 'agent_type',
        width: 100,
        render: (_: unknown, record: Agent) => {
          const typeMap: Record<string, string> = {
            BareLLM: '裸 LLM',
            CodingAgent: '编程',
          };
          return <Tag>{typeMap[record.agent_type] || record.agent_type || 'BareLLM'}</Tag>;
        },
      },
      {
        title: '思考',
        key: 'thinking',
        width: 80,
        align: 'center',
        render: (_: unknown, record: Agent) => (
          <Switch
            size="small"
            checked={record.enable_thinking_process}
            checkedChildren="开"
            unCheckedChildren="关"
            onChange={(checked) => handleToggleThinking(record, checked)}
          />
        ),
      },
      {
        title: '轮数',
        key: 'max_iterations',
        width: 90,
        align: 'center',
        render: (_: unknown, record: Agent) => (
          <InputNumber
            size="small"
            min={1}
            max={50}
            defaultValue={record.max_iterations}
            onPressEnter={(e) => {
              const v = Number((e.target as HTMLInputElement).value);
              if (!Number.isNaN(v) && v !== record.max_iterations) {
                handleUpdateMaxIterations(record, v);
              }
            }}
            onBlur={(e) => {
              const v = Number((e.target as HTMLInputElement).value);
              if (!Number.isNaN(v) && v !== record.max_iterations) {
                handleUpdateMaxIterations(record, v);
              }
            }}
            style={{ width: 70 }}
          />
        ),
      },
      {
        title: '技能',
        key: 'skills',
        width: 70,
        align: 'center',
        render: (_: unknown, record: Agent) => {
          const count = (record.skills_list || []).length;
          return <Tag color={count === 0 ? 'default' : 'blue'}>{count === 0 ? '不限' : count}</Tag>;
        },
      },
      {
        title: '工具',
        key: 'tools',
        width: 70,
        align: 'center',
        render: (_: unknown, record: Agent) => {
          const count = (record.tools_list || []).length;
          return <Tag color={count === 0 ? 'default' : 'cyan'}>{count === 0 ? '不限' : count}</Tag>;
        },
      },
      {
        title: '状态',
        key: 'status',
        width: 120,
        render: (_: unknown, record: Agent) => (
          <Space size="small">
            {record.is_default && <Tag color="gold">默认</Tag>}
            <Tag color={record.is_active ? 'success' : 'default'}>{record.is_active ? '启用' : '停用'}</Tag>
          </Space>
        ),
      },
      {
        title: '操作',
        key: 'action',
        width: screens.xs ? 140 : 280,
        render: (_: unknown, record: Agent) => (
          <Space size={[4, 4]} wrap>
            <Button type="text" icon={<EditOutlined />} onClick={() => openEditor(record)}>
              {screens.xs ? '' : '编辑'}
            </Button>
            {!record.is_default && !screens.xs && (
              <Button type="text" onClick={() => handleSetDefault(record)}>
                默认
              </Button>
            )}
            <Popconfirm title="确认删除该 Agent？" onConfirm={() => handleDelete(record.id)}>
              <Button type="text" danger icon={<DeleteOutlined />}>
                {screens.xs ? '' : '删除'}
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [handleDelete, handleSetDefault, handleToggleThinking, handleUpdateMaxIterations, openEditor, screens.xs],
  );

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  useEffect(() => {
    fetchProviders();
  }, [fetchProviders]);

  useEffect(() => {
    listBuiltInTools()
      .then(setBuiltInTools)
      .catch(() => {
        message.error('获取内置工具列表失败');
      });
  }, []);

  useEffect(() => {
    listSkillsSimple()
      .then(setSkillsOptions)
      .catch(() => {
        message.error('获取技能列表失败');
      });
  }, []);

  // 自动更新工具列表：根据 skills 和 MCP 绑定自动添加/删除对应工具
  useEffect(() => {
    if (!open || !editing) return;

    const currentTools = (form.getFieldValue('tools_list') as string[]) || [];
    const toolsSet = new Set(currentTools);
    let changed = false;

    // 检查 Skills 绑定
    const skills = (form.getFieldValue('skills_list') as string[]) || [];
    if (skills.length > 0) {
      if (!toolsSet.has('use_skill')) {
        toolsSet.add('use_skill');
        changed = true;
      }
    } else {
      if (toolsSet.has('use_skill')) {
        toolsSet.delete('use_skill');
        changed = true;
      }
    }

    // 检查 MCP 绑定
    const hasActiveMCP = mcpBindings.length > 0 && mcpBindings.some((b) => b.is_active);
    if (hasActiveMCP) {
      if (!toolsSet.has('use_mcp')) {
        toolsSet.add('use_mcp');
        changed = true;
      }
      if (!toolsSet.has('call_mcp_tool')) {
        toolsSet.add('call_mcp_tool');
        changed = true;
      }
    } else {
      if (toolsSet.has('use_mcp')) {
        toolsSet.delete('use_mcp');
        changed = true;
      }
      if (toolsSet.has('call_mcp_tool')) {
        toolsSet.delete('call_mcp_tool');
        changed = true;
      }
    }

    if (changed) {
      form.setFieldsValue({ tools_list: Array.from(toolsSet) });
    }
  }, [open, editing, mcpBindings]);

  return (
    <div style={{ padding: 0 }}>
      <Card
        title={<Title level={screens.xs ? 4 : 3} style={{ margin: 0 }}>Agent 管理</Title>}
        extra={
          <Space>
            <Button onClick={() => fetchList()}>刷新</Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                openEditor(null);
              }}
            >
              {screens.xs ? '新建' : '新建 Agent'}
            </Button>
          </Space>
        }
      >
        <Table<Agent>
          rowKey="id"
          loading={loading}
          dataSource={items}
          columns={columns}
          size={screens.xs ? 'small' : 'middle'}
          scroll={{ x: screens.xs ? 760 : 'max-content' }}
        />
      </Card>

      <Drawer
        title={editing ? '编辑 Agent' : '新建 Agent'}
        placement="right"
        open={open}
        onClose={closeEditor}
        width={screens.xs ? '100%' : 760}
        styles={{ body: { padding: 0 } }}
        destroyOnClose
        extra={
          !editing ? (
            <Space>
              <Button onClick={closeEditor}>取消</Button>
              <Button type="primary" onClick={() => form.submit()} loading={saving}>
                创建
              </Button>
            </Space>
          ) : null
        }
      >
        <Form layout="vertical" form={form} onFinish={handleSubmit} style={{ height: '100%' }} size="small">
          <Tabs
            activeKey={activeTab}
            onChange={(k) => setActiveTab(k as any)}
            tabBarStyle={{ padding: '0 12px', margin: 0, marginBottom: 0 }}
            items={[
              {
                key: 'basic',
                label: '基础信息',
                children: (
                  <div style={{ padding: '0 0 4px', overflow: 'auto' }}>
                    {/* ===== Agent 基本信息卡片 ===== */}
                    <Card
                      size="small"
                      styles={{ body: compactCardBody }}
                      title={<span><EditOutlined /> 基本信息</span>}
                      style={{ marginBottom: 8 }}
                      extra={
                        editing ? (
                          editingSections.basicInfo ? (
                            <Button
                              type="link"
                              size="small"
                              loading={savingSections.basicInfo}
                              onClick={() => {
                                const values = form.getFieldsValue(['name', 'agent_type', 'description']);
                                if (!values.name) { message.error('名称不能为空'); return; }
                                handlePatchSection('basicInfo', {
                                  name: values.name,
                                  agent_type: values.agent_type,
                                  description: values.description,
                                });
                              }}
                            >
                              保存
                            </Button>
                          ) : (
                            <Button type="link" size="small" onClick={() => toggleSectionEdit('basicInfo')}>
                              编辑
                            </Button>
                          )
                        ) : null
                      }
                    >
                      {editing !== null && !editingSections.basicInfo ? (
                        <div>
                          <div style={{ marginBottom: 8 }}>
                            <span style={{ color: '#999', marginRight: 8 }}>名称：</span>
                            <span style={{ fontWeight: 500 }}>{form.getFieldValue('name') || '-'}</span>
                          </div>
                          <div style={{ marginBottom: 8 }}>
                            <span style={{ color: '#999', marginRight: 8 }}>类型：</span>
                            <span>{form.getFieldValue('agent_type') || 'BareLLM'}</span>
                          </div>
                          <div>
                            <span style={{ color: '#999', marginRight: 8 }}>描述：</span>
                            <span>{form.getFieldValue('description') || '-'}</span>
                          </div>
                        </div>
                      ) : (
                        <div>
                          <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]} style={{ marginBottom: 8 }}>
                            <Input placeholder="Agent 名称" />
                          </Form.Item>
                          <Form.Item label="类型" name="agent_type" style={{ marginBottom: 8 }}>
                            <Select
                              placeholder="选择 Agent 类型"
                              options={[
                                { value: 'BareLLM', label: 'BareLLM - 裸 LLM 调用' },
                                { value: 'CodingAgent', label: 'CodingAgent - 编程 Agent' },
                              ]}
                            />
                          </Form.Item>
                          <Form.Item label="描述" name="description" style={{ marginBottom: 0 }}>
                            <Input.TextArea rows={2} placeholder="Agent 描述" />
                          </Form.Item>
                        </div>
                      )}
                    </Card>

                    {/* ===== 模型配置卡片 ===== */}
                    <Card
                      size="small"
                      styles={{ body: compactCardBody }}
                      title={<span><ThunderboltOutlined /> 模型配置</span>}
                      style={{ marginBottom: 8 }}
                      extra={
                        editing ? (
                          editingSections.modelConfig ? (
                            <Button
                              type="link"
                              size="small"
                              loading={savingSections.modelConfig}
                              onClick={() => {
                                const values = form.getFieldsValue(['model', 'max_tokens', 'temperature', 'max_iterations', 'history_messages']);
                                if (!values.model) { message.error('请选择模型'); return; }
                                handlePatchSection('modelConfig', values);
                              }}
                            >
                              保存
                            </Button>
                          ) : (
                            <Button type="link" size="small" onClick={() => toggleSectionEdit('modelConfig')}>
                              编辑
                            </Button>
                          )
                        ) : null
                      }
                    >
                      {editing !== null && !editingSections.modelConfig ? (
                        <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 8 }}>
                          <div><span style={{ color: '#999' }}>模型：</span>{form.getFieldValue('model') || '-'}</div>
                          <div><span style={{ color: '#999' }}>Max Tokens：</span>{form.getFieldValue('max_tokens')}</div>
                          <div><span style={{ color: '#999' }}>Temperature：</span>{form.getFieldValue('temperature')}</div>
                          <div><span style={{ color: '#999' }}>最大迭代：</span>{form.getFieldValue('max_iterations')}</div>
                          <div><span style={{ color: '#999' }}>历史消息数：</span>{form.getFieldValue('history_messages')}</div>
                        </div>
                      ) : (
                        <div>
                          <Form.Item label="模型" name="model" rules={[{ required: true, message: '请选择模型' }]}>
                            <Select
                              showSearch
                              allowClear
                              loading={providersLoading}
                              options={modelOptions}
                              placeholder={providersLoading ? '正在加载模型列表...' : '请选择模型'}
                              notFoundContent={providersLoading ? '正在加载...' : '没有可选模型'}
                              filterOption={(input, option) => {
                                const q = input.toLowerCase();
                                const v = String(option?.value || '').toLowerCase();
                                const l = String(option?.label || '').toLowerCase();
                                return v.includes(q) || l.includes(q);
                              }}
                            />
                          </Form.Item>
                          <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 12 }}>
                            <Form.Item label="Max Tokens" name="max_tokens">
                              <InputNumber min={0} style={{ width: '100%' }} />
                            </Form.Item>
                            <Form.Item label="Temperature" name="temperature">
                              <InputNumber min={0} max={2} step={0.1} style={{ width: '100%' }} />
                            </Form.Item>
                            <Form.Item label="最大迭代" name="max_iterations">
                              <InputNumber min={1} style={{ width: '100%' }} />
                            </Form.Item>
                            <Form.Item label="历史消息数" name="history_messages">
                              <InputNumber min={0} style={{ width: '100%' }} />
                            </Form.Item>
                          </div>
                        </div>
                      )}
                    </Card>

                    {/* ===== 开关配置卡片（即时保存） ===== */}
                    <Card
                      size="small"
                      styles={{ body: compactCardBody }}
                      title={<span>开关设置</span>}
                    >
                      <Space direction={screens.xs ? 'vertical' : 'horizontal'} style={{ display: 'flex' }} align="start">
                        <Form.Item label="设为默认" name="is_default" valuePropName="checked" style={{ marginBottom: 0 }}>
                          <Switch
                            checkedChildren="默认"
                            unCheckedChildren="非默认"
                            onChange={(checked) => {
                              if (editing) handlePatchSection('_switch_default', { is_default: checked });
                            }}
                          />
                        </Form.Item>
                        <Form.Item label="启用" name="is_active" valuePropName="checked" style={{ marginBottom: 0 }}>
                          <Switch
                            checkedChildren="启用"
                            unCheckedChildren="停用"
                            disabled={!editing}
                            onChange={(checked) => {
                              if (editing) handlePatchSection('_switch_active', { is_active: checked });
                            }}
                          />
                        </Form.Item>
                        <Form.Item label="展示思考过程" name="enable_thinking_process" valuePropName="checked" style={{ marginBottom: 0 }}>
                          <Switch
                            checkedChildren="开启"
                            unCheckedChildren="关闭"
                            onChange={(checked) => {
                              if (editing) handlePatchSection('_switch_thinking', { enable_thinking_process: checked });
                            }}
                          />
                        </Form.Item>
                      </Space>
                    </Card>
                  </div>
                ),
              },
              {
                key: 'skills',
                label: '技能工具',
                children: (
                  <div style={{ padding: '0 0 4px', overflow: 'auto' }}>
                    {/* ===== 技能配置卡片 ===== */}
                    <Card
                      size="small"
                      styles={{ body: compactCardBody }}
                      title={<span><ThunderboltOutlined /> 技能配置</span>}
                      style={{ marginBottom: 8 }}
                      extra={
                        editing ? (
                          editingSections.skillsConfig ? (
                            <Button
                              type="link"
                              size="small"
                              loading={savingSections.skillsConfig}
                              onClick={() => {
                                const skills = form.getFieldValue('skills_list') as string[] || [];
                                handlePatchSection('skillsConfig', { skills_list: skills });
                              }}
                            >
                              保存
                            </Button>
                          ) : (
                            <Button type="link" size="small" onClick={() => toggleSectionEdit('skillsConfig')}>
                              编辑
                            </Button>
                          )
                        ) : null
                      }
                    >
                      {editing !== null && !editingSections.skillsConfig ? (
                        <div>
                          {(form.getFieldValue('skills_list') as string[] || []).length === 0 ? (
                            <span style={{ color: '#999' }}>未配置技能（不限）</span>
                          ) : (
                            <Space wrap>
                              {(form.getFieldValue('skills_list') as string[] || []).map((s) => (
                                <Tag key={s} color="blue">{s}</Tag>
                              ))}
                            </Space>
                          )}
                        </div>
                      ) : (
                        <div>
                          <Form.Item label="Skills（可多选/自定义）" name="skills_list" style={{ marginBottom: 0 }}>
                            <Select
                              mode="tags"
                              placeholder="从列表选择或输入添加"
                              options={skillsOptions.map((s) => ({
                                value: s.name,
                                label: s.description ? `${s.name} - ${s.description}` : s.name,
                              }))}
                            />
                          </Form.Item>
                        </div>
                      )}
                    </Card>

                    {/* ===== MCP Server 绑定卡片 ===== */}
                    <Card
                      size="small"
                      styles={{ body: compactCardBody }}
                      title={<span><ApiOutlined /> MCP Server 绑定</span>}
                      style={{ marginBottom: 8 }}
                      extra={
                        editing ? (
                          <Button
                            size="small"
                            onClick={async () => {
                              if (!editing) return;
                              await reloadMCP(editing.id);
                            }}
                            loading={mcpLoading}
                          >
                            刷新
                          </Button>
                        ) : null
                      }
                    >
                      <div style={{ color: '#999', fontSize: 12, marginBottom: 4 }}>
                        说明：不绑定任何 MCP Server 则该 Agent 无法使用 MCP 工具
                      </div>
                      {!editing && <Tag>请先创建 Agent 后再绑定 MCP Server</Tag>}
                      {editing && (
                        <>
                          <div style={{ marginBottom: 12, display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center' }}>
                            <Select
                              placeholder="选择 MCP Server"
                              style={{ flex: 1, minWidth: 260 }}
                              value={mcpForm.getFieldValue('mcp_server_id')}
                              onChange={(value) => mcpForm.setFieldValue('mcp_server_id', value)}
                              options={mcpServers
                                .filter((s) => !mcpBindings.some((b) => b.mcp_server_id === s.id))
                                .map((s) => ({ value: s.id, label: `${s.name}（${s.code}）` }))}
                              showSearch
                              optionFilterProp="label"
                            />
                            <Button
                              type="primary"
                              loading={mcpLoading}
                              onClick={() => {
                                const mcpServerId = mcpForm.getFieldValue('mcp_server_id');
                                if (!mcpServerId) {
                                  message.error('请选择 MCP Server');
                                  return;
                                }
                                if (!editing) return;
                                createBinding({
                                  agent_id: editing.id,
                                  mcp_server_id: mcpServerId,
                                  is_active: true,
                                  auto_load: false,
                                })
                                  .then(() => {
                                    message.success('绑定成功');
                                    mcpForm.resetFields();
                                    return reloadMCP(editing.id);
                                  })
                                  .catch((e) => {
                                    message.error(getMCPErrorMessage(e) || '绑定失败');
                                  });
                              }}
                            >
                              绑定
                            </Button>
                          </div>

                          <Table<AgentMCPBinding>
                            dataSource={mcpBindings}
                            rowKey="id"
                            loading={mcpLoading}
                            size="small"
                            pagination={false}
                            scroll={{ x: 520 }}
                            columns={[
                              {
                                title: 'MCP Server',
                                render: (_: unknown, record: AgentMCPBinding) => {
                                  const s = mcpServers.find((x) => x.id === record.mcp_server_id);
                                  return <span>{s ? `${s.name}（${s.code}）` : record.mcp_server_id}</span>;
                                },
                              },
                              {
                                title: '工具',
                                render: (_: unknown, record: AgentMCPBinding) => {
                                  const v = record.enabled_tools;
                                  return v && v.length > 0 ? v.slice(0, 3).map((x) => <Tag key={x}>{x}</Tag>) : <Tag>全部</Tag>;
                                },
                              },
                              {
                                title: '状态',
                                width: 90,
                                render: (_: unknown, record: AgentMCPBinding) => (
                                  <Tag color={record.is_active ? 'success' : 'default'}>{record.is_active ? '启用' : '禁用'}</Tag>
                                ),
                              },
                              {
                                title: '自动加载',
                                width: 90,
                                render: (_: unknown, record: AgentMCPBinding) => (
                                  <Switch
                                    size="small"
                                    checked={record.auto_load}
                                    checkedChildren="自"
                                    unCheckedChildren="手"
                                    onChange={async () => {
                                      try {
                                        await updateBinding(record.id, { auto_load: !record.auto_load });
                                        message.success(!record.auto_load ? '已设置自动加载' : '已取消自动加载');
                                        if (editing) await reloadMCP(editing.id);
                                      } catch (e) {
                                        message.error(getMCPErrorMessage(e) || '操作失败');
                                      }
                                    }}
                                  />
                                ),
                              },
                              {
                                title: '操作',
                                width: 200,
                                render: (_: unknown, record: AgentMCPBinding) => (
                                  <Space size="small">
                                    <Switch
                                      size="small"
                                      checked={record.is_active}
                                      onChange={async () => {
                                        try {
                                          await updateBinding(record.id, { is_active: !record.is_active });
                                          message.success(record.is_active ? '已禁用' : '已启用');
                                          if (editing) await reloadMCP(editing.id);
                                        } catch (e) {
                                          message.error(getMCPErrorMessage(e) || '操作失败');
                                        }
                                      }}
                                    />
                                    <Button
                                      type="text"
                                      size="small"
                                      onClick={async () => {
                                        setEditingBinding(record);
                                        setToolsDrawerOpen(true);
                                        setToolsDrawerLoading(true);
                                        try {
                                          const tools = await listMCPTools(record.mcp_server_id);
                                          setToolsForServer(tools);
                                          const current = record.enabled_tools || [];
                                          toolsForm.setFieldsValue({
                                            all_tools: current.length === 0,
                                            enabled_tools: current,
                                          });
                                        } catch (e) {
                                          setToolsForServer([]);
                                          message.error(getMCPErrorMessage(e) || '获取工具列表失败');
                                        } finally {
                                          setToolsDrawerLoading(false);
                                        }
                                      }}
                                    >
                                      配置工具
                                    </Button>
                                    <Popconfirm
                                      title="确认解绑该 MCP Server？"
                                      onConfirm={async () => {
                                        try {
                                          await deleteBinding(record.id);
                                          message.success('解绑成功');
                                          if (editing) await reloadMCP(editing.id);
                                        } catch (e) {
                                          message.error(getMCPErrorMessage(e) || '解绑失败');
                                        }
                                      }}
                                    >
                                      <Button type="text" danger size="small">
                                        解绑
                                      </Button>
                                    </Popconfirm>
                                  </Space>
                                ),
                              },
                            ]}
                          />
                        </>
                      )}
                    </Card>

                    {/* ===== 工具配置卡片 ===== */}
                    <Card
                      size="small"
                      styles={{ body: compactCardBody }}
                      title={<span><ToolOutlined /> 工具配置</span>}
                      extra={
                        editing ? (
                          editingSections.toolsConfig ? (
                            <Button
                              type="link"
                              size="small"
                              loading={savingSections.toolsConfig}
                              onClick={() => {
                                const tools = form.getFieldValue('tools_list') as string[] || [];
                                handlePatchSection('toolsConfig', { tools_list: tools });
                              }}
                            >
                              保存
                            </Button>
                          ) : (
                            <Button type="link" size="small" onClick={() => toggleSectionEdit('toolsConfig')}>
                              编辑
                            </Button>
                          )
                        ) : null
                      }
                    >
                      {editing !== null && !editingSections.toolsConfig ? (
                        <div>
                          {(form.getFieldValue('tools_list') as string[] || []).length === 0 ? (
                            <span style={{ color: '#999' }}>未配置工具（不限）</span>
                          ) : (
                            <Space wrap>
                              {(form.getFieldValue('tools_list') as string[] || []).map((t) => (
                                <Tag key={t} color="cyan">{t}</Tag>
                              ))}
                            </Space>
                          )}
                        </div>
                      ) : (
                        <div>
                          <Form.Item label="Tools（可多选/自定义）" name="tools_list" style={{ marginBottom: 4 }}>
                            <Select
                              mode="tags"
                              placeholder="输入后回车添加"
                              options={builtInTools.map((t) => ({
                                value: t.name,
                                label: t.description ? `${t.name} - ${t.description}` : t.name,
                              }))}
                            />
                          </Form.Item>
                          <div style={{ color: '#999', fontSize: 12 }}>
                            说明：绑定 Skills 会自动添加 use_skill，绑定 MCP 会自动添加 use_mcp 和 call_mcp_tool
                          </div>
                        </div>
                      )}
                    </Card>
                  </div>
                ),
              },
              {
                key: 'personality',
                label: '人格属性',
                children: (
                  <div style={{ padding: '0 0 4px', overflow: 'auto' }}>
                    {/* ===== MD 配置文件卡片 ===== */}
                    {([
                      { key: 'identity_content', file: 'IDENTITY.md', desc: '智能伙伴的名字、性格和身份定义' },
                      { key: 'soul_content', file: 'SOUL.md', desc: '智能伙伴的信念、风格和行为准则' },
                      { key: 'agents_content', file: 'AGENTS.md', desc: '智能伙伴的工作流程和记忆规则' },
                      { key: 'user_content', file: 'USER.md', desc: '关于用户的信息和偏好' },
                      { key: 'tools_content', file: 'TOOLS.md', desc: '智能伙伴的本地笔记和速查表' },
                    ] as const).map((item) => (
                      <Card
                        key={item.key}
                        size="small"
                        styles={{ body: compactCardBody }}
                        style={{ marginBottom: 8 }}
                        title={
                          <span>
                            <FileTextOutlined style={{ marginRight: 8 }} />
                            {item.file}
                          </span>
                        }
                        extra={
                          editing ? (
                            editingSections[item.key] ? (
                              <Button
                                type="link"
                                size="small"
                                loading={savingSections[item.key]}
                                onClick={() => {
                                  const content = form.getFieldValue(item.key) as string || '';
                                  handlePatchSection(item.key, { [item.key]: content });
                                }}
                              >
                                保存
                              </Button>
                            ) : (
                              <Button type="link" size="small" onClick={() => toggleSectionEdit(item.key)}>
                                编辑
                              </Button>
                            )
                          ) : null
                        }
                      >
                        <div style={{ color: '#999', fontSize: 12, marginBottom: editingSections[item.key] ? 8 : 0 }}>
                          {item.desc}
                        </div>
                        {!editing || !editingSections[item.key] ? (
                          <pre style={{
                            margin: 0,
                            padding: 4,
                            background: '#fafafa',
                            borderRadius: 4,
                            fontSize: 12,
                            fontFamily: 'monospace',
                            maxHeight: 150,
                            overflow: 'auto',
                            whiteSpace: 'pre-wrap',
                            wordBreak: 'break-word',
                          }}>
                            {(form.getFieldValue(item.key) as string) || '(空)'}
                          </pre>
                        ) : (
                          <Form.Item name={item.key} style={{ marginBottom: 0 }}>
                            <Input.TextArea
                              rows={8}
                              style={{ fontFamily: 'monospace', fontSize: 12 }}
                            />
                          </Form.Item>
                        )}
                      </Card>
                    ))}
                  </div>
                ),
              },
              {
                key: 'claudecode',
                label: 'Claude Code',
                children: (
                  <div style={{ padding: '0 0 4px', overflow: 'auto' }}>
                    {form.getFieldValue('agent_type') !== 'CodingAgent' ? (
                      <Card
                        size="small"
                        styles={{ body: compactCardBody }}
                        title="Claude Code 配置"
                      >
                        <div style={{ color: '#999', textAlign: 'center', padding: '20px 0' }}>
                          Claude Code 配置仅适用于 CodingAgent 类型<br />
                          请在「基础信息」中修改 Agent 类型
                        </div>
                      </Card>
                    ) : (
                      <>
                        {/* ===== Claude Code 基本配置卡片 ===== */}
                        <Card
                          size="small"
                          styles={{ body: compactCardBody }}
                          title={<span><ApiOutlined /> Claude Code 基本配置</span>}
                          style={{ marginBottom: 8 }}
                          extra={
                            editing ? (
                              editingSections.claudeCodeConfig ? (
                                <Button
                                  type="link"
                                  size="small"
                                  loading={savingSections.claudeCodeConfig}
                                  onClick={() => {
                                    const config = form.getFieldValue('claude_code_config') as ClaudeCodeConfig || {};
                                    handlePatchSection('claudeCodeConfig', { claude_code_config: config });
                                  }}
                                >
                                  保存
                                </Button>
                              ) : (
                                <Button type="link" size="small" onClick={() => toggleSectionEdit('claudeCodeConfig')}>
                                  编辑
                                </Button>
                              )
                            ) : null
                          }
                        >
                          {editing !== null && !editingSections.claudeCodeConfig ? (
                            <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 8 }}>
                              <div><span style={{ color: '#999' }}>模型：</span>{form.getFieldValue('claude_code_config')?.model || 'MiniMax-M2.7-highspeed'}</div>
                              <div><span style={{ color: '#999' }}>最大思考 Token：</span>{form.getFieldValue('claude_code_config')?.max_thinking_tokens || 8000}</div>
                              <div><span style={{ color: '#999' }}>权限模式：</span>{form.getFieldValue('claude_code_config')?.permission_mode || 'default'}</div>
                              <div><span style={{ color: '#999' }}>恢复会话：</span>{form.getFieldValue('claude_code_config')?.resume ? '是' : '否'}</div>
                              <div><span style={{ color: '#999' }}>最大对话轮次：</span>{form.getFieldValue('claude_code_config')?.max_turns || '无限制'}</div>
                              <div><span style={{ color: '#999' }}>工作目录：</span>{form.getFieldValue('claude_code_config')?.cwd || '默认'}</div>
                            </div>
                          ) : (
                            <div>
                              <Form.Item label="模型" name={['claude_code_config', 'model']} style={{ marginBottom: 8 }}>
                                <Input placeholder="MiniMax-M2.7-highspeed" />
                              </Form.Item>
                              <Form.Item label="系统提示词" name={['claude_code_config', 'system_prompt']} style={{ marginBottom: 8 }}>
                                <Input.TextArea rows={3} placeholder="设置 Claude Code 的系统提示词" />
                              </Form.Item>
                              <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 12 }}>
                                <Form.Item label="最大思考 Token" name={['claude_code_config', 'max_thinking_tokens']}>
                                  <InputNumber min={0} style={{ width: '100%' }} placeholder="8000" />
                                </Form.Item>
                                <Form.Item label="权限模式" name={['claude_code_config', 'permission_mode']}>
                                  <Select
                                    placeholder="选择权限模式"
                                    options={[
                                      { value: 'default', label: 'Default - 标准处理' },
                                      { value: 'acceptEdits', label: 'AcceptEdits - 自动接受编辑' },
                                      { value: 'plan', label: 'Plan - 计划模式' },
                                      { value: 'bypassPermissions', label: 'Bypass - 绕过权限' },
                                    ]}
                                  />
                                </Form.Item>
                                <Form.Item label="最大对话轮次" name={['claude_code_config', 'max_turns']}>
                                  <InputNumber min={0} style={{ width: '100%' }} placeholder="0 表示无限制" />
                                </Form.Item>
                                <Form.Item label="工作目录" name={['claude_code_config', 'cwd']}>
                                  <Input placeholder="留空使用默认目录" />
                                </Form.Item>
                              </div>
                              <Form.Item label="恢复会话" name={['claude_code_config', 'resume']} valuePropName="checked" style={{ marginBottom: 0 }}>
                                <Switch checkedChildren="启用" unCheckedChildren="禁用" />
                              </Form.Item>
                            </div>
                          )}
                        </Card>

                        {/* ===== 工具控制卡片 ===== */}
                        <Card
                          size="small"
                          styles={{ body: compactCardBody }}
                          title={<span><ToolOutlined /> 工具控制</span>}
                          style={{ marginBottom: 8 }}
                          extra={
                            editing ? (
                              editingSections.claudeCodeTools ? (
                                <Button
                                  type="link"
                                  size="small"
                                  loading={savingSections.claudeCodeTools}
                                  onClick={() => {
                                    const config = form.getFieldValue('claude_code_config') as ClaudeCodeConfig || {};
                                    handlePatchSection('claudeCodeTools', { claude_code_config: config });
                                  }}
                                >
                                  保存
                                </Button>
                              ) : (
                                <Button type="link" size="small" onClick={() => toggleSectionEdit('claudeCodeTools')}>
                                  编辑
                                </Button>
                              )
                            ) : null
                          }
                        >
                          {editing !== null && !editingSections.claudeCodeTools ? (
                            <div>
                              <div style={{ marginBottom: 8 }}>
                                <span style={{ color: '#999' }}>允许的工具：</span>
                                {(form.getFieldValue('claude_code_config')?.allowed_tools || []).length === 0 ? (
                                  <Tag>全部</Tag>
                                ) : (
                                  <Space wrap>
                                    {form.getFieldValue('claude_code_config')?.allowed_tools?.map((t: string) => (
                                      <Tag key={t} color="blue">{t}</Tag>
                                    ))}
                                  </Space>
                                )}
                              </div>
                              <div>
                                <span style={{ color: '#999' }}>禁止的工具：</span>
                                {(form.getFieldValue('claude_code_config')?.disallowed_tools || []).length === 0 ? (
                                  <Tag>无</Tag>
                                ) : (
                                  <Space wrap>
                                    {form.getFieldValue('claude_code_config')?.disallowed_tools?.map((t: string) => (
                                      <Tag key={t} color="red">{t}</Tag>
                                    ))}
                                  </Space>
                                )}
                              </div>
                            </div>
                          ) : (
                            <div>
                              <Form.Item label="允许的工具（留空表示全部允许）" name={['claude_code_config', 'allowed_tools']} style={{ marginBottom: 8 }}>
                                <Select
                                  mode="tags"
                                  placeholder="输入工具名称后回车"
                                  options={[
                                    { value: 'Read', label: 'Read - 读取文件' },
                                    { value: 'Write', label: 'Write - 写入文件' },
                                    { value: 'Edit', label: 'Edit - 编辑文件' },
                                    { value: 'Bash', label: 'Bash - 执行命令' },
                                    { value: 'Glob', label: 'Glob - 文件搜索' },
                                    { value: 'Grep', label: 'Grep - 内容搜索' },
                                    { value: 'ToolSearch', label: 'ToolSearch - 工具搜索' },
                                    { value: 'WebFetch', label: 'WebFetch - 网页获取' },
                                  ]}
                                />
                              </Form.Item>
                              <Form.Item label="禁止的工具" name={['claude_code_config', 'disallowed_tools']} style={{ marginBottom: 0 }}>
                                <Select
                                  mode="tags"
                                  placeholder="输入工具名称后回车"
                                  options={[
                                    { value: 'Bash', label: 'Bash - 执行命令' },
                                    { value: 'Write', label: 'Write - 写入文件' },
                                    { value: 'Edit', label: 'Edit - 编辑文件' },
                                    { value: 'Delete', label: 'Delete - 删除文件' },
                                  ]}
                                />
                              </Form.Item>
                            </div>
                          )}
                        </Card>

                        {/* ===== 沙箱安全卡片 ===== */}
                        <Card
                          size="small"
                          styles={{ body: compactCardBody }}
                          title={<span><ApiOutlined /> 沙箱安全</span>}
                          style={{ marginBottom: 8 }}
                          extra={
                            editing ? (
                              editingSections.claudeCodeSandbox ? (
                                <Button
                                  type="link"
                                  size="small"
                                  loading={savingSections.claudeCodeSandbox}
                                  onClick={() => {
                                    const config = form.getFieldValue('claude_code_config') as ClaudeCodeConfig || {};
                                    handlePatchSection('claudeCodeSandbox', { claude_code_config: config });
                                  }}
                                >
                                  保存
                                </Button>
                              ) : (
                                <Button type="link" size="small" onClick={() => toggleSectionEdit('claudeCodeSandbox')}>
                                  编辑
                                </Button>
                              )
                            ) : null
                          }
                        >
                          {editing !== null && !editingSections.claudeCodeSandbox ? (
                            <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 8 }}>
                              <div><span style={{ color: '#999' }}>沙箱启用：</span>{form.getFieldValue('claude_code_config')?.sandbox_enabled ? '是' : '否'}</div>
                              <div><span style={{ color: '#999' }}>自动批准 Bash：</span>{form.getFieldValue('claude_code_config')?.auto_allow_bash_if_sandboxed ? '是' : '否'}</div>
                              <div><span style={{ color: '#999' }}>排除命令：</span>{(form.getFieldValue('claude_code_config')?.excluded_commands || []).join(', ') || '无'}</div>
                            </div>
                          ) : (
                            <div>
                              <Form.Item label="启用沙箱" name={['claude_code_config', 'sandbox_enabled']} valuePropName="checked" style={{ marginBottom: 8 }}>
                                <Switch checkedChildren="启用" unCheckedChildren="禁用" />
                              </Form.Item>
                              <Form.Item label="沙箱模式下自动批准 Bash" name={['claude_code_config', 'auto_allow_bash_if_sandboxed']} valuePropName="checked" style={{ marginBottom: 8 }}>
                                <Switch checkedChildren="启用" unCheckedChildren="禁用" />
                              </Form.Item>
                              <Form.Item label="排除命令（沙箱绕过）" name={['claude_code_config', 'excluded_commands']} style={{ marginBottom: 0 }}>
                                <Select
                                  mode="tags"
                                  placeholder="输入命令名称后回车"
                                  options={[
                                    { value: 'git', label: 'git' },
                                    { value: 'docker', label: 'docker' },
                                    { value: 'npm', label: 'npm' },
                                    { value: 'pnpm', label: 'pnpm' },
                                    { value: 'make', label: 'make' },
                                  ]}
                                />
                              </Form.Item>
                            </div>
                          )}
                        </Card>

                        {/* ===== 高级设置卡片 ===== */}
                        <Card
                          size="small"
                          styles={{ body: compactCardBody }}
                          title={<span><ApiOutlined /> 高级设置</span>}
                          style={{ marginBottom: 8 }}
                          extra={
                            editing ? (
                              editingSections.claudeCodeAdvanced ? (
                                <Button
                                  type="link"
                                  size="small"
                                  loading={savingSections.claudeCodeAdvanced}
                                  onClick={() => {
                                    const config = form.getFieldValue('claude_code_config') as ClaudeCodeConfig || {};
                                    handlePatchSection('claudeCodeAdvanced', { claude_code_config: config });
                                  }}
                                >
                                  保存
                                </Button>
                              ) : (
                                <Button type="link" size="small" onClick={() => toggleSectionEdit('claudeCodeAdvanced')}>
                                  编辑
                                </Button>
                              )
                            ) : null
                          }
                        >
                          {editing !== null && !editingSections.claudeCodeAdvanced ? (
                            <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 8 }}>
                              <div><span style={{ color: '#999' }}>备用模型：</span>{form.getFieldValue('claude_code_config')?.fallback_model || '无'}</div>
                              <div><span style={{ color: '#999' }}>文件检查点：</span>{form.getFieldValue('claude_code_config')?.file_checkpointing ? '是' : '否'}</div>
                              <div><span style={{ color: '#999' }}>继续会话：</span>{form.getFieldValue('claude_code_config')?.continue_conversation ? '是' : '否'}</div>
                              <div><span style={{ color: '#999' }}>Fork 会话：</span>{form.getFieldValue('claude_code_config')?.fork_session ? '是' : '否'}</div>
                              <div><span style={{ color: '#999' }}>追加提示词：</span>{form.getFieldValue('claude_code_config')?.append_system_prompt ? '有' : '无'}</div>
                              <div><span style={{ color: '#999' }}>CLI 路径：</span>{form.getFieldValue('claude_code_config')?.cli_path || '默认'}</div>
                              <div><span style={{ color: '#999' }}>最大预算 USD：</span>{form.getFieldValue('claude_code_config')?.max_budget_usd || '无限制'}</div>
                              <div><span style={{ color: '#999' }}>部分消息：</span>{form.getFieldValue('claude_code_config')?.include_partial_messages ? '启用' : '禁用'}</div>
                            </div>
                          ) : (
                            <div>
                              <div style={{ display: 'grid', gridTemplateColumns: screens.xs ? '1fr' : '1fr 1fr', gap: 12 }}>
                                <Form.Item label="备用模型" name={['claude_code_config', 'fallback_model']}>
                                  <Input placeholder="主模型不可用时使用" />
                                </Form.Item>
                                <Form.Item label="最大预算 (USD)" name={['claude_code_config', 'max_budget_usd']}>
                                  <InputNumber min={0} step={0.1} style={{ width: '100%' }} placeholder="0 表示无限制" />
                                </Form.Item>
                                <Form.Item label="CLI 路径" name={['claude_code_config', 'cli_path']}>
                                  <Input placeholder="留空使用默认路径" />
                                </Form.Item>
                              </div>
                              <Form.Item label="追加系统提示词" name={['claude_code_config', 'append_system_prompt']} style={{ marginBottom: 8 }}>
                                <Input.TextArea rows={2} placeholder="在现有系统提示词后追加内容" />
                              </Form.Item>
                              <Space direction={screens.xs ? 'vertical' : 'horizontal'} style={{ display: 'flex' }} align="start">
                                <Form.Item label="文件检查点" name={['claude_code_config', 'file_checkpointing']} valuePropName="checked" style={{ marginBottom: 0 }}>
                                  <Switch checkedChildren="启用" unCheckedChildren="禁用" />
                                </Form.Item>
                                <Form.Item label="继续会话" name={['claude_code_config', 'continue_conversation']} valuePropName="checked" style={{ marginBottom: 0 }}>
                                  <Switch checkedChildren="启用" unCheckedChildren="禁用" />
                                </Form.Item>
                                <Form.Item label="Fork 会话" name={['claude_code_config', 'fork_session']} valuePropName="checked" style={{ marginBottom: 0 }}>
                                  <Switch checkedChildren="启用" unCheckedChildren="禁用" />
                                </Form.Item>
                                <Form.Item label="部分消息" name={['claude_code_config', 'include_partial_messages']} valuePropName="checked" style={{ marginBottom: 0 }}>
                                  <Switch checkedChildren="启用" unCheckedChildren="禁用" />
                                </Form.Item>
                              </Space>
                            </div>
                          )}
                        </Card>
                      </>
                    )}
                  </div>
                ),
              },
            ]}
          />
        </Form>
      </Drawer>

      <Drawer
        title="配置 MCP 工具"
        placement="right"
        open={toolsDrawerOpen}
        onClose={() => {
          setToolsDrawerOpen(false);
          setEditingBinding(null);
          setToolsForServer([]);
          toolsForm.resetFields();
        }}
        width={screens.xs ? '100%' : 520}
        destroyOnClose
        extra={
          <Space>
            <Button
              onClick={() => {
                setToolsDrawerOpen(false);
                setEditingBinding(null);
                toolsForm.resetFields();
              }}
            >
              取消
            </Button>
            <Button
              type="primary"
              loading={toolsDrawerLoading}
              onClick={() => toolsForm.submit()}
              disabled={!editingBinding}
            >
              保存
            </Button>
          </Space>
        }
      >
        <Form
          form={toolsForm}
          layout="vertical"
          onFinish={async (values) => {
            if (!editingBinding || !editing) return;
            try {
              const enabled = values.all_tools ? [] : (values.enabled_tools || []);
              await updateBinding(editingBinding.id, { enabled_tools: enabled });
              message.success('已更新工具配置');
              await reloadMCP(editing.id);
              setToolsDrawerOpen(false);
              setEditingBinding(null);
            } catch (e) {
              message.error(getMCPErrorMessage(e) || '保存失败');
            }
          }}
        >
          <Form.Item name="all_tools" valuePropName="checked" label="启用全部工具">
            <Switch checkedChildren="全部" unCheckedChildren="选择" />
          </Form.Item>
          <Form.Item shouldUpdate noStyle>
            {() => {
              const all = Boolean(toolsForm.getFieldValue('all_tools'));
              return (
                <Form.Item
                  name="enabled_tools"
                  label="选择启用工具"
                  hidden={all}
                  rules={all ? undefined : [{ required: true, message: '请选择至少一个工具，或开启“全部工具”' }]}
                >
                  <Select
                    mode="multiple"
                    placeholder="选择工具"
                    loading={toolsDrawerLoading}
                    options={toolsForServer.map((t) => ({ value: t.name, label: t.name }))}
                    allowClear
                  />
                </Form.Item>
              );
            }}
          </Form.Item>
        </Form>
      </Drawer>
    </div>
  );
};
