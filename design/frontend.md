# 前端界面设计 (React + Ant Design)

前端采用 React + Ant Design 构建，提供任务管理界面。

## 目录

- [技术栈](#技术栈)
- [项目结构](#项目结构)
- [核心组件](#核心组件)
- [API 服务层](#api-服务层)
- [WebSocket 实时通信](#websocket-实时通信)
- [页面设计](#页面设计)
- [状态管理](#状态管理)

---

## 技术栈

```json
{
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-router-dom": "^6.x",
    "antd": "^5.x",
    "@ant-design/icons": "^5.x",
    "axios": "^1.x",
    "zustand": "^4.x"
  }
}
```

---

## 项目结构

```
frontend/
├── src/
│   ├── api/
│   │   ├── taskApi.ts          # 任务 HTTP API 调用
│   │   └── websocket.ts         # WebSocket 客户端
│   ├── components/
│   │   ├── TaskList/            # 任务列表组件
│   │   ├── TaskDetail/          # 任务详情组件
│   │   ├── TaskTree/            # 任务树组件
│   │   ├── TaskForm/            # 创建/编辑任务表单
│   │   ├── ProgressBar/          # 进度条组件
│   │   └── StatusBadge/         # 状态徽章组件
│   ├── pages/
│   │   ├── TaskDashboard.tsx    # 任务仪表板
│   │   ├── TaskListPage.tsx     # 任务列表页
│   │   ├── TaskDetailPage.tsx   # 任务详情页
│   │   └── TaskTreePage.tsx     # 任务树页
│   ├── stores/
│   │   └── taskStore.ts         # Zustand 状态管理
│   ├── hooks/
│   │   ├── useTaskWebSocket.ts  # WebSocket Hook
│   │   └── useTaskOperations.ts # 任务操作 Hook
│   ├── types/
│   │   └── task.ts              # TypeScript 类型定义
│   ├── App.tsx
│   └── main.tsx
├── package.json
└── vite.config.ts
```

---

## TypeScript 类型定义

```typescript
// src/types/task.ts

// 任务状态枚举
export type TaskStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

// 任务类型枚举
export type TaskType = 'data_processing' | 'file_operation' | 'api_call' | 'custom';

// 任务接口
export interface Task {
  id: string;
  trace_id: string;
  span_id: string;
  parent_id?: string;
  name: string;
  description: string;
  type: TaskType;
  status: TaskStatus;
  progress: Progress;
  result?: Result;
  error?: string;
  metadata: Record<string, unknown>;
  timeout: number;
  max_retries: number;
  priority: number;
  created_at: number;
  started_at?: number;
  finished_at?: number;
}

// 进度接口
export interface Progress {
  total: number;
  current: number;
  percentage: number;
  stage: string;
  detail: string;
  updated_at: number;
}

// 结果接口
export interface Result {
  data: unknown;
  message: string;
}

// 创建任务请求
export interface CreateTaskRequest {
  name: string;
  description: string;
  type: TaskType;
  timeout: number;
  max_retries: number;
  priority: number;
  metadata: Record<string, unknown>;
  parent_id?: string;
}

// WebSocket 消息类型
export interface WSMessage {
  type: string;
  trace_id: string;
  data: unknown;
  timestamp: number;
}

// 创建任务响应
export interface CreateTaskResponse {
  id: string;
  trace_id: string;
  span_id: string;
  status: TaskStatus;
}

// 任务列表响应
export interface TaskListResponse {
  tasks: Task[];
  total: number;
}

// 任务树节点
export interface TaskTreeNode {
  task: Task;
  children: TaskTreeNode[];
}
```

---

## API 服务层

### 任务 API 调用

```typescript
// src/api/taskApi.ts
import axios from 'axios';
import type {
  Task,
  TaskListResponse,
  CreateTaskRequest,
  CreateTaskResponse,
  TaskTreeNode
} from '../types/task';

const BASE_URL = '/api/v1';
const apiClient = axios.create({
  baseURL: BASE_URL,
  timeout: 10000,
});

// 创建任务
export async function createTask(request: CreateTaskRequest): Promise<CreateTaskResponse> {
  const response = await apiClient.post<CreateTaskResponse>('/tasks', request);
  return response.data;
}

// 获取单个任务
export async function getTask(taskId: string): Promise<Task> {
  const response = await apiClient.get<Task>('/tasks', {
    params: { id: taskId },
  });
  return response.data;
}

// 获取任务列表（按 trace_id）
export async function listTasksByTrace(traceId: string): Promise<TaskListResponse> {
  const response = await apiClient.get<TaskListResponse>(`/tasks/trace/${traceId}`);
  return response.data;
}

// 获取任务树
export async function getTaskTree(traceId: string): Promise<TaskTreeNode[]> {
  const response = await apiClient.get<TaskTreeNode[]>(`/traces/${traceId}/tree`);
  return response.data;
}

// 取消任务
export async function cancelTask(taskId: string): Promise<void> {
  await apiClient.post(`/tasks/${taskId}/cancel`);
}

export default apiClient;
```

---

## WebSocket 实时通信

### WebSocket 客户端

```typescript
// src/api/websocket.ts
import type { WSMessage } from '../types/task';

type MessageHandler = (message: WSMessage) => void;

class TaskWebSocketClient {
  private ws: WebSocket | null = null;
  private handlers: Map<string, Set<MessageHandler>> = new Map();
  private traceId: string = '';
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 3000;

  // 连接到 WebSocket 服务器
  connect(traceId: string): void {
    this.traceId = traceId;
    const wsUrl = `ws://localhost:8080/ws?trace_id=${traceId}`;

    this.ws = new WebSocket(wsUrl);

    this.ws.onopen = () => {
      console.log('WebSocket 连接已建立');
      this.reconnectAttempts = 0;
    };

    this.ws.onmessage = (event) => {
      try {
        const message: WSMessage = JSON.parse(event.data);
        this.dispatchMessage(message);
      } catch (error) {
        console.error('解析 WebSocket 消息失败:', error);
      }
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket 错误:', error);
    };

    this.ws.onclose = () => {
      console.log('WebSocket 连接已关闭');
      this.attemptReconnect();
    };
  }

  // 订阅特定类型的消息
  subscribe(eventType: string, handler: MessageHandler): () => void {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, new Set());
    }
    this.handlers.get(eventType)!.add(handler);

    // 返回取消订阅函数
    return () => {
      this.handlers.get(eventType)?.delete(handler);
    };
  }

  // 派发消息到对应的处理器
  private dispatchMessage(message: WSMessage): void {
    const handlers = this.handlers.get(message.type);
    if (handlers) {
      handlers.forEach((handler) => handler(message));
    }
    // 也派发到通配符处理器
    const wildcardHandlers = this.handlers.get('*');
    if (wildcardHandlers) {
      wildcardHandlers.forEach((handler) => handler(message));
    }
  }

  // 断开连接
  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.handlers.clear();
  }

  // 尝试重连
  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('达到最大重连次数，停止重连');
      return;
    }

    this.reconnectAttempts++;
    console.log(`${this.reconnectDelay / 1000}秒后尝试第${this.reconnectAttempts}次重连...`);

    setTimeout(() => {
      if (this.traceId) {
        this.connect(this.traceId);
      }
    }, this.reconnectDelay);
  }
}

// 导出单例
export const wsClient = new TaskWebSocketClient();
```

---

## 核心组件

### TaskList 任务列表组件

```tsx
// src/components/TaskList/TaskList.tsx
import React from 'react';
import { Table, Tag, Space, Button } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { StatusBadge } from '../StatusBadge';
import { ProgressBar } from '../ProgressBar';
import type { Task } from '../../types/task';
import { useNavigate } from 'react-router-dom';

interface TaskListProps {
  tasks: Task[];
  loading?: boolean;
  onCancel?: (taskId: string) => void;
}

export const TaskList: React.FC<TaskListProps> = ({ tasks, loading, onCancel }) => {
  const navigate = useNavigate();

  const columns: ColumnsType<Task> = [
    {
      title: '任务名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: Task) => (
        <a onClick={() => navigate(`/tasks/${record.id}`)}>{name}</a>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: Task['status']) => <StatusBadge status={status} />,
    },
    {
      title: '进度',
      dataIndex: 'progress',
      key: 'progress',
      render: (progress: Task['progress']) => <ProgressBar progress={progress} />,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: Task['type']) => <Tag>{type}</Tag>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (timestamp: number) => new Date(timestamp).toLocaleString(),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: Task) => (
        <Space>
          <Button size="small" onClick={() => navigate(`/tasks/${record.id}`)}>
            详情
          </Button>
          {record.status === 'running' && (
            <Button size="small" danger onClick={() => onCancel?.(record.id)}>
              取消
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return <Table columns={columns} dataSource={tasks} rowKey="id" loading={loading} />;
};
```

### StatusBadge 状态徽章组件

```tsx
// src/components/StatusBadge/StatusBadge.tsx
import React from 'react';
import { Tag } from 'antd';
import type { TaskStatus } from '../../types/task';

const statusConfig: Record<TaskStatus, { color: string; label: string }> = {
  pending: { color: 'default', label: '待处理' },
  running: { color: 'processing', label: '运行中' },
  completed: { color: 'success', label: '已完成' },
  failed: { color: 'error', label: '失败' },
  cancelled: { color: 'warning', label: '已取消' },
};

interface StatusBadgeProps {
  status: TaskStatus;
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({ status }) => {
  const config = statusConfig[status] || statusConfig.pending;
  return <Tag color={config.color}>{config.label}</Tag>;
};
```

### ProgressBar 进度条组件

```tsx
// src/components/ProgressBar/ProgressBar.tsx
import React from 'react';
import { Progress } from 'antd';
import type { Progress as ProgressType } from '../../types/task';

interface ProgressBarProps {
  progress: ProgressType;
  showInfo?: boolean;
}

export const ProgressBar: React.FC<ProgressBarProps> = ({ progress, showInfo = true }) => {
  const formatPercent = (percent: number): string => {
    return `${Math.round(percent)}%`;
  };

  return (
    <Progress
      percent={progress.percentage}
      status={progress.percentage >= 100 ? 'success' : 'active'}
      format={showInfo ? formatPercent : undefined}
      strokeColor="#1890ff"
    />
  );
};
```

### TaskForm 创建任务表单

```tsx
// src/components/TaskForm/TaskForm.tsx
import React from 'react';
import { Form, Input, Select, InputNumber, Button, Space } from 'antd';
import type { CreateTaskRequest } from '../../types/task';

const { Option } = Select;

interface TaskFormProps {
  initialValues?: Partial<CreateTaskRequest>;
  onSubmit: (values: CreateTaskRequest) => void;
  onCancel: () => void;
  loading?: boolean;
}

export const TaskForm: React.FC<TaskFormProps> = ({
  initialValues,
  onSubmit,
  onCancel,
  loading,
}) => {
  const [form] = Form.useForm<CreateTaskRequest>();

  const handleFinish = (values: CreateTaskRequest): void => {
    onSubmit({
      ...values,
      timeout: values.timeout || 60000,
      max_retries: values.max_retries || 0,
      priority: values.priority || 0,
      metadata: values.metadata || {},
    });
  };

  return (
    <Form
      form={form}
      layout="vertical"
      initialValues={initialValues}
      onFinish={handleFinish}
    >
      <Form.Item
        name="name"
        label="任务名称"
        rules={[{ required: true, message: '请输入任务名称' }]}
      >
        <Input placeholder="请输入任务名称" />
      </Form.Item>

      <Form.Item name="description" label="任务描述">
        <Input.TextArea placeholder="请输入任务描述" rows={3} />
      </Form.Item>

      <Form.Item
        name="type"
        label="任务类型"
        rules={[{ required: true, message: '请选择任务类型' }]}
      >
        <Select placeholder="请选择任务类型">
          <Option value="data_processing">数据处理</Option>
          <Option value="file_operation">文件操作</Option>
          <Option value="api_call">API 调用</Option>
          <Option value="custom">自定义</Option>
        </Select>
      </Form.Item>

      <Form.Item name="timeout" label="超时时间 (ms)">
        <InputNumber min={1000} step={1000} defaultValue={60000} style={{ width: '100%' }} />
      </Form.Item>

      <Form.Item name="max_retries" label="最大重试次数">
        <InputNumber min={0} max={10} defaultValue={0} style={{ width: '100%' }} />
      </Form.Item>

      <Form.Item name="priority" label="优先级">
        <InputNumber min={0} max={100} defaultValue={0} style={{ width: '100%' }} />
      </Form.Item>

      <Form.Item>
        <Space>
          <Button type="primary" htmlType="submit" loading={loading}>
            创建任务
          </Button>
          <Button onClick={onCancel}>取消</Button>
        </Space>
      </Form.Item>
    </Form>
  );
};
```

### TaskTree 任务树组件

```tsx
// src/components/TaskTree/TaskTree.tsx
import React from 'react';
import { Tree } from 'antd';
import type { DataNode } from 'antd/es/tree';
import { StatusBadge } from '../StatusBadge';
import type { TaskTreeNode } from '../../types/task';

interface TaskTreeProps {
  treeData: TaskTreeNode[];
  onSelect?: (taskId: string) => void;
}

export const TaskTree: React.FC<TaskTreeProps> = ({ treeData, onSelect }) => {
  const convertToTreeData = (nodes: TaskTreeNode[]): DataNode[] => {
    return nodes.map((node) => ({
      key: node.task.id,
      title: (
        <span>
          <StatusBadge status={node.task.status} />
          {' '}
          {node.task.name}
        </span>
      ),
      children: node.children ? convertToTreeData(node.children) : undefined,
    }));
  };

  return (
    <Tree
      treeData={convertToTreeData(treeData)}
      onSelect={(selectedKeys) => {
        if (selectedKeys.length > 0) {
          onSelect?.(selectedKeys[0] as string);
        }
      }}
    />
  );
};
```

---

## 页面设计

### TaskDashboard 任务仪表板

```tsx
// src/pages/TaskDashboard.tsx
import React, { useEffect, useState } from 'react';
import { Row, Col, Card, Statistic, Button } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { TaskList } from '../components/TaskList';
import { useTaskWebSocket } from '../hooks/useTaskWebSocket';
import { taskStore } from '../stores/taskStore';
import { useNavigate } from 'react-router-dom';

export const TaskDashboard: React.FC = () => {
  const navigate = useNavigate();
  const { tasks, loading, fetchTasks } = taskStore();
  const traceId = 'default-trace-id'; // 或从 URL 获取

  // 建立 WebSocket 连接
  useTaskWebSocket(traceId);

  useEffect(() => {
    fetchTasks(traceId);
  }, [fetchTasks, traceId]);

  // 统计各状态任务数量
  const statusCounts = {
    pending: tasks.filter((t) => t.status === 'pending').length,
    running: tasks.filter((t) => t.status === 'running').length,
    completed: tasks.filter((t) => t.status === 'completed').length,
    failed: tasks.filter((t) => t.status === 'failed').length,
  };

  return (
    <div style={{ padding: 24 }}>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic title="待处理" value={statusCounts.pending} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="运行中" value={statusCounts.running} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="已完成" value={statusCounts.completed} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="失败" value={statusCounts.failed} />
          </Card>
        </Col>
      </Row>

      <Card
        title="任务列表"
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => fetchTasks(traceId)}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/tasks/new')}>
              创建任务
            </Button>
          </Space>
        }
      >
        <TaskList tasks={tasks} loading={loading} />
      </Card>
    </div>
  );
};
```

### TaskDetailPage 任务详情页

```tsx
// src/pages/TaskDetailPage.tsx
import React, { useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Card, Descriptions, Button, Space, Spin, message } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { StatusBadge } from '../components/StatusBadge';
import { ProgressBar } from '../components/ProgressBar';
import { taskStore } from '../stores/taskStore';
import { useTaskWebSocket } from '../hooks/useTaskWebSocket';
import { cancelTask as cancelTaskApi } from '../api/taskApi';

export const TaskDetailPage: React.FC = () => {
  const { taskId } = useParams<{ taskId: string }>();
  const navigate = useNavigate();
  const { currentTask, loading, fetchTask, updateTaskInList } = taskStore();

  // 建立 WebSocket 连接以接收实时更新
  useTaskWebSocket(currentTask?.trace_id || '');

  useEffect(() => {
    if (taskId) {
      fetchTask(taskId);
    }
  }, [taskId, fetchTask]);

  const handleCancel = async (): Promise<void> => {
    if (!taskId) return;
    try {
      await cancelTaskApi(taskId);
      message.success('任务已取消');
      fetchTask(taskId);
    } catch (error) {
      message.error('取消任务失败');
    }
  };

  if (loading || !currentTask) {
    return <Spin style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />;
  }

  return (
    <div style={{ padding: 24 }}>
      <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(-1)} style={{ marginBottom: 16 }}>
        返回
      </Button>

      <Card title="任务详情">
        <Descriptions column={2} bordered>
          <Descriptions.Item label="任务ID">{currentTask.id}</Descriptions.Item>
          <Descriptions.Item label="任务名称">{currentTask.name}</Descriptions.Item>
          <Descriptions.Item label="状态">
            <StatusBadge status={currentTask.status} />
          </Descriptions.Item>
          <Descriptions.Item label="类型">{currentTask.type}</Descriptions.Item>
          <Descriptions.Item label="优先级">{currentTask.priority}</Descriptions.Item>
          <Descriptions.Item label="超时时间">{currentTask.timeout}ms</Descriptions.Item>
          <Descriptions.Item label="创建时间">
            {new Date(currentTask.created_at).toLocaleString()}
          </Descriptions.Item>
          <Descriptions.Item label="开始时间">
            {currentTask.started_at ? new Date(currentTask.started_at).toLocaleString() : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="完成时间">
            {currentTask.finished_at ? new Date(currentTask.finished_at).toLocaleString() : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="重试次数">
            {currentTask.max_retries}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="执行进度" style={{ marginTop: 16 }}>
        <ProgressBar progress={currentTask.progress} />
        <p><strong>阶段：</strong>{currentTask.progress.stage || '-'}</p>
        <p><strong>详情：</strong>{currentTask.progress.detail || '-'}</p>
      </Card>

      {currentTask.error && (
        <Card title="错误信息" style={{ marginTop: 16 }}>
          <pre style={{ color: 'red' }}>{currentTask.error}</pre>
        </Card>
      )}

      {currentTask.result && (
        <Card title="执行结果" style={{ marginTop: 16 }}>
          <pre>{JSON.stringify(currentTask.result, null, 2)}</pre>
        </Card>
      )}

      {currentTask.status === 'running' && (
        <Card style={{ marginTop: 16 }}>
          <Space>
            <Button danger onClick={handleCancel}>
              取消任务
            </Button>
          </Space>
        </Card>
      )}
    </div>
  );
};
```

### TaskTreePage 任务树页面

```tsx
// src/pages/TaskTreePage.tsx
import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Card, Button, Spin } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { TaskTree } from '../components/TaskTree';
import { getTaskTree } from '../api/taskApi';
import type { TaskTreeNode } from '../types/task';

export const TaskTreePage: React.FC = () => {
  const { traceId } = useParams<{ traceId: string }>();
  const navigate = useNavigate();
  const [treeData, setTreeData] = useState<TaskTreeNode[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (traceId) {
      setLoading(true);
      getTaskTree(traceId)
        .then(setTreeData)
        .finally(() => setLoading(false));
    }
  }, [traceId]);

  return (
    <div style={{ padding: 24 }}>
      <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(-1)} style={{ marginBottom: 16 }}>
        返回
      </Button>

      <Card title={`任务树 - Trace: ${traceId}`}>
        {loading ? (
          <Spin />
        ) : (
          <TaskTree treeData={treeData} onSelect={(taskId) => navigate(`/tasks/${taskId}`)} />
        )}
      </Card>
    </div>
  );
};
```

---

## 状态管理 (Zustand)

```typescript
// src/stores/taskStore.ts
import { create } from 'zustand';
import type { Task } from '../types/task';
import * as taskApi from '../api/taskApi';

interface TaskState {
  tasks: Task[];
  currentTask: Task | null;
  loading: boolean;
  error: string | null;
  fetchTasks: (traceId: string) => Promise<void>;
  fetchTask: (taskId: string) => Promise<void>;
  updateTaskInList: (task: Task) => void;
  addTask: (task: Task) => void;
  clearError: () => void;
}

export const useTaskStore = create<TaskState>((set, get) => ({
  tasks: [],
  currentTask: null,
  loading: false,
  error: null,

  fetchTasks: async (traceId: string) => {
    set({ loading: true, error: null });
    try {
      const response = await taskApi.listTasksByTrace(traceId);
      set({ tasks: response.tasks, loading: false });
    } catch (error) {
      set({ error: '获取任务列表失败', loading: false });
    }
  },

  fetchTask: async (taskId: string) => {
    set({ loading: true, error: null });
    try {
      const task = await taskApi.getTask(taskId);
      set({ currentTask: task, loading: false });
    } catch (error) {
      set({ error: '获取任务详情失败', loading: false });
    }
  },

  updateTaskInList: (task: Task) => {
    const tasks = get().tasks.map((t) => (t.id === task.id ? task : t));
    set({ tasks, currentTask: task });
  },

  addTask: (task: Task) => {
    set({ tasks: [task, ...get().tasks] });
  },

  clearError: () => set({ error: null }),
}));
```

---

## 自定义 Hooks

### useTaskWebSocket

```typescript
// src/hooks/useTaskWebSocket.ts
import { useEffect } from 'react';
import { wsClient } from '../api/websocket';
import { useTaskStore } from '../stores/taskStore';
import type { WSMessage, Task } from '../types/task';

export function useTaskWebSocket(traceId: string): void {
  const updateTaskInList = useTaskStore((state) => state.updateTaskInList);

  useEffect(() => {
    if (!traceId) return;

    // 连接到 WebSocket
    wsClient.connect(traceId);

    // 订阅任务更新消息
    const unsubscribe = wsClient.subscribe('TaskStatusChanged', (message: WSMessage) => {
      const updatedTask = message.data as Task;
      updateTaskInList(updatedTask);
    });

    // 订阅进度更新消息
    wsClient.subscribe('TaskProgressUpdated', (message: WSMessage) => {
      const progressData = message.data as { task_id: string; progress: Task['progress'] };
      const currentTasks = useTaskStore.getState().tasks;
      const task = currentTasks.find((t) => t.id === progressData.task_id);
      if (task) {
        updateTaskInList({ ...task, progress: progressData.progress });
      }
    });

    // 组件卸载时断开连接
    return () => {
      unsubscribe();
      wsClient.disconnect();
    };
  }, [traceId, updateTaskInList]);
}
```

### useTaskOperations

```typescript
// src/hooks/useTaskOperations.ts
import { useState } from 'react';
import { message } from 'antd';
import * as taskApi from '../api/taskApi';
import type { CreateTaskRequest, CreateTaskResponse } from '../types/task';
import { useTaskStore } from '../stores/taskStore';

export function useTaskOperations() {
  const [creating, setCreating] = useState(false);
  const fetchTasks = useTaskStore((state) => state.fetchTasks);

  const createTask = async (
    request: CreateTaskRequest
  ): Promise<CreateTaskResponse | null> => {
    setCreating(true);
    try {
      const response = await taskApi.createTask(request);
      message.success('任务创建成功');
      return response;
    } catch (error) {
      message.error('创建任务失败');
      return null;
    } finally {
      setCreating(false);
    }
  };

  const cancelTask = async (taskId: string): Promise<boolean> => {
    try {
      await taskApi.cancelTask(taskId);
      message.success('任务已取消');
      return true;
    } catch (error) {
      message.error('取消任务失败');
      return false;
    }
  };

  return {
    creating,
    createTask,
    cancelTask,
  };
}
```

---

## 路由配置

```tsx
// src/App.tsx
import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { TaskDashboard } from './pages/TaskDashboard';
import { TaskDetailPage } from './pages/TaskDetailPage';
import { TaskTreePage } from './pages/TaskTreePage';

const App: React.FC = () => {
  return (
    <ConfigProvider locale={zhCN}>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Navigate to="/tasks" replace />} />
          <Route path="/tasks" element={<TaskDashboard />} />
          <Route path="/tasks/:taskId" element={<TaskDetailPage />} />
          <Route path="/tasks/trace/:traceId/tree" element={<TaskTreePage />} />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  );
};

export default App;
```

---

## 页面路由汇总

| 路径 | 页面 | 描述 |
|------|------|------|
| / | 跳转 | 重定向到 /tasks |
| /tasks | TaskDashboard | 任务仪表板，显示统计和任务列表 |
| /tasks/:taskId | TaskDetailPage | 任务详情页，显示任务详细信息 |
| /tasks/trace/:traceId/tree | TaskTreePage | 任务树页面，显示父子任务关系 |
