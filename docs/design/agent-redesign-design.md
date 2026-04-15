# Agent 管理界面重构设计文档

## 概述

本设计文档描述如何将 Agent 管理界面从"单一列表+平铺菜单"重构为"按类型分组+两步创建+资源内嵌"的新结构。

## 架构约束

- 前端使用 React + TypeScript + Ant Design
- 不修改后端 Domain、Application、Infrastructure 层
- 仅调整前端路由、菜单、组件结构和页面交互

## 设计方案

### 1. 左侧菜单调整

#### 变更前
```
Dashboard | 项目需求 | 对话记录 | Agents 管理 | Skills 管理 | 状态机 | MCP 管理 | 渠道 | 会话 | LLM | 用户 | 设置
```

#### 变更后
```
Dashboard | 项目需求 | 对话记录 | Agent 工坊 | 状态机 | 渠道 | 会话 | LLM 配置 | 用户 | 设置
```

- 移除 `Skills 管理` 和 `MCP 管理` 两个一级菜单项
- `Agents 管理` 更名为 `Agent 工坊`（或保持原名称，根据产品偏好）

### 2. Agent 列表页结构

#### 页面布局
```
┌─────────────────────────────────────────────┐
│ Agent 工坊                        [新建 Agent] │
├─────────────────────────────────────────────┤
│ [全部 | 个人助理 | 编程 Agent]                │
├─────────────────────────────────────────────┤
│ 搜索框 + 刷新按钮                              │
├─────────────────────────────────────────────┤
│ ┌─────────┐ ┌─────────┐ ┌─────────┐        │
│ │ Agent A │ │ Agent B │ │ Agent C │ ...    │
│ └─────────┘ └─────────┘ └─────────┘        │
└─────────────────────────────────────────────┘
```

#### 类型分组映射
| 分组标签 | 匹配的 agent_type | 说明 |
|---------|------------------|------|
| 全部 | 全部 | 不过滤 |
| 个人助理 | `BareLLM` | 通用对话型 Agent |
| 编程 Agent | `CodingAgent`, `OpenCodeAgent` | 所有具备代码执行能力的 Agent |

> 未来扩展：新增 `OpenClawAgent`、`HermesAgent` 时，可归入现有分组或新增分组 Tab。

### 3. 新建 Agent 两步流程

#### Step 1: 选择类型
弹窗展示类型卡片网格，每个卡片包含：
- 图标/示意图
- 类型名称（个人助理 / Claude Code / OpenCode）
- 一句话描述
- 选中态高亮

```typescript
const AGENT_TYPE_CARDS = [
  {
    type: 'BareLLM',
    title: '个人助理',
    description: '通用对话型 Agent，可绑定 Skills 和 MCP 工具',
    icon: <RobotOutlined />,
  },
  {
    type: 'CodingAgent',
    title: 'Claude Code',
    description: '基于 Claude Code CLI 的编程 Agent',
    icon: <CodeOutlined />,
  },
  {
    type: 'OpenCodeAgent',
    title: 'OpenCode',
    description: '基于 OpenCode CLI 的编程 Agent',
    icon: <CodeOutlined />,
  },
];
```

#### Step 2: 配置表单
根据选中的类型，直接进入对应的编辑抽屉：
- `BareLLM` → 显示 基本信息 / 技能工具 / 人格属性
- `CodingAgent` → 显示 基本信息 / Claude Code 配置
- `OpenCodeAgent` → 显示 基本信息 / OpenCode 配置

**关键规则**：新建模式下，基本信息中的 `agent_type` 下拉框禁用或隐藏，避免用户在表单内切换类型导致配置项混乱。

### 4. 个人助理内嵌资源管理

#### 方案：标签页内快捷入口
在 `BareLLM` 编辑抽屉的"技能工具"标签页中，增加两个操作卡片：

```
┌─────────────────┐  ┌─────────────────┐
│   Skills 库     │  │   MCP 服务      │
│  管理技能列表    │  │  管理 MCP 绑定   │
│  [去管理 →]     │  │  [去管理 →]     │
└─────────────────┘  └─────────────────┘
```

点击后打开 Modal 抽屉，内嵌现有的 `SkillsManagement` 和 `MCPBindingManagement` 组件（或复用现有页面组件）。

这样无需从左侧菜单进入，也不会让 Agent 编辑页变得过于臃肿。

### 5. 组件变更清单

#### 新增组件
| 组件 | 路径 | 职责 |
|------|------|------|
| `AgentTypeSelector` | `components/AgentManagement/components/AgentTypeSelector.tsx` | 新建第一步的类型选择弹窗 |

#### 修改组件
| 组件 | 修改内容 |
|------|---------|
| `AgentManagementPage.tsx` | 增加 `Segmented` 分组器；过滤逻辑 |
| `AgentTable.tsx` | 无大改，接收过滤后的列表 |
| `AgentEditDrawer.tsx` | 新建模式下禁用/隐藏 `agent_type` 选择；调整默认 active tab |
| `useAgentManagement.ts` | 新增 `createStep`、`selectedCreateType` 状态；调整打开逻辑 |
| `App.tsx` | 移除 Skills、MCP 两个菜单项 |

#### 复用组件（从内嵌入口调出）
| 组件 | 来源 |
|------|------|
| `SkillManagementPage` | 现有 `frontend/src/pages/SkillManagementPage.tsx` |
| `MCPManagementPage` 或绑定组件 | 现有 MCP 相关页面/组件 |

### 6. 状态管理

列表页新增本地状态：
```typescript
// AgentManagementPage
const [activeGroup, setActiveGroup] = useState<'all' | 'assistant' | 'coding'>('all');

const groupedAgents = useMemo(() => {
  if (activeGroup === 'all') return agents;
  if (activeGroup === 'assistant') return agents.filter(a => a.agent_type === 'BareLLM');
  if (activeGroup === 'coding') return agents.filter(a => a.agent_type === 'CodingAgent' || a.agent_type === 'OpenCodeAgent');
  return agents;
}, [agents, activeGroup]);
```

### 7. 路由影响

- 无新增路由
- 移除 `/skills` 和 `/mcp` 的左侧菜单入口，但路由本身保留（通过个人助理的内嵌入口访问时可能需要）
- 若用户直接访问 `/skills` 或 `/mcp` URL，页面仍可正常渲染
