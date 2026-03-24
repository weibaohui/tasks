# API 客户端规范

## 1. API 客户端结构

```typescript
// api/client.ts
import axios from 'axios';

const client = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
});

// 请求拦截器
client.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// 响应拦截器
client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      useAuthStore.getState().logout();
    }
    return Promise.reject(error);
  }
);

export default client;
```

## 2. API 模块组织

```typescript
// api/userApi.ts
import client from './client';
import type { User, CreateUserInput, UpdateUserInput } from '../types/user';

export const userApi = {
  list: async (params?: { limit?: number; offset?: number }) => {
    const { data } = await client.get<User[]>('/users', { params });
    return data;
  },

  get: async (id: string) => {
    const { data } = await client.get<User>(`/users/${id}`);
    return data;
  },

  create: async (input: CreateUserInput) => {
    const { data } = await client.post<User>('/users', input);
    return data;
  },

  update: async (id: string, input: UpdateUserInput) => {
    const { data } = await client.put<User>(`/users/${id}`, input);
    return data;
  },

  delete: async (id: string) => {
    await client.delete(`/users/${id}`);
  },
};
```

## 3. 类型定义

```typescript
// types/user.ts
export interface User {
  id: string;
  name: string;
  email: string;
  created_at: number;
  updated_at: number;
}

export type CreateUserInput = Omit<User, 'id' | 'created_at' | 'updated_at'>;

export type UpdateUserInput = Partial<CreateUserInput>;
```

## 4. 分页查询

```typescript
// api/conversationRecordApi.ts
import type { ConversationRecord, ListQueryParams } from '../types/conversationRecord';

export const conversationRecordApi = {
  list: async (params: ListQueryParams) => {
    const { data } = await client.get<ConversationRecord[]>('/conversation-records', {
      params: {
        user_code: params.user_code,
        trace_id: params.trace_id,
        session_key: params.session_key,
        limit: params.limit ?? 50,
        offset: params.offset ?? 0,
      },
    });
    return data;
  },

  getByTrace: async (traceId: string) => {
    const { data } = await client.get<ConversationRecord[]>(`/conversation-records/trace/${traceId}`);
    return data;
  },

  getBySession: async (sessionKey: string) => {
    const { data } = await client.get<ConversationRecord[]>(`/conversation-records/session/${sessionKey}`);
    return data;
  },
};
```

## 5. 错误处理

```typescript
// types/errors.ts
export class ApiError extends Error {
  constructor(
    public code: number,
    message: string,
    public details?: string
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

// 使用
try {
  const user = await userApi.get(id);
} catch (error) {
  if (error instanceof ApiError) {
    message.error(`${error.message}: ${error.details}`);
  }
}
```

## 6. 响应类型

```typescript
// 分页响应
interface PaginatedResponse<T> {
  data: T[];
  total: number;
  limit: number;
  offset: number;
}

// 单个响应
interface SingleResponse<T> {
  data: T;
}

// 错误响应
interface ErrorResponse {
  code: number;
  message: string;
  details?: string;
}
```

## 7. API 封装模式

### 7.1 封装示例
```typescript
// api/agentApi.ts
import client from './client';
import type { Agent } from '../types/agent';

export const agentApi = {
  list: async (userCode: string) => {
    const { data } = await client.get<Agent[]>('/agents', {
      params: { user_code: userCode },
    });
    return data;
  },

  get: async (id: string) => {
    const { data } = await client.get<Agent>(`/agents`, {
      params: { id },
    });
    return data;
  },

  create: async (input: CreateAgentRequest) => {
    const { data } = await client.post<Agent>('/agents', input);
    return data;
  },

  update: async (id: string, input: UpdateAgentRequest) => {
    const { data } = await client.put<Agent>(`/agents`, input, {
      params: { id },
    });
    return data;
  },

  delete: async (id: string) => {
    await client.delete(`/agents`, { params: { id } });
  },
};
```

## 8. 请求取消

```typescript
// 使用 AbortController
const controller = new AbortController();

useEffect(() => {
  const fetchData = async () => {
    try {
      const data = await client.get('/users', {
        signal: controller.signal,
      });
      setUsers(data);
    } catch (error) {
      if (axios.isCancel(error)) {
        // 请求被取消
      }
    }
  };

  fetchData();

  return () => {
    controller.abort();
  };
}, []);
```

## 9. 重试机制

```typescript
// 简单重试
const fetchWithRetry = async (url: string, retries = 3) => {
  try {
    return await client.get(url);
  } catch (error) {
    if (retries > 0) {
      return fetchWithRetry(url, retries - 1);
    }
    throw error;
  }
};
```

## 10. 缓存策略

```typescript
// 使用 react-query
import { useQuery } from 'react-query';

const { data, isLoading } = useQuery(
  ['user', userId],
  () => userApi.get(userId),
  {
    staleTime: 5 * 60 * 1000, // 5 分钟
    cacheTime: 30 * 60 * 1000, // 30 分钟
  }
);
```
