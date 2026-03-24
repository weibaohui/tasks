# TypeScript 编码规范

## 1. 类型定义

### 1.1 接口 vs 类型别名

```typescript
// 接口：用于对象结构
interface User {
  id: string;
  name: string;
  email: string;
}

// 类型别名：用于联合类型、原始类型
type Status = 'active' | 'inactive';
type ID = string | number;
```

### 1.2 避免 any

```typescript
// ❌ 错误
function process(data: any) {
  return data.value;
}

// ✅ 正确
function process(data: User) {
  return data.value;
}

// 如果真的需要灵活类型
function process(data: unknown) {
  if (typeof data === 'object' && data !== null) {
    // 处理
  }
}
```

### 1.3 类型导出

```typescript
// types/user.ts
export interface User {
  id: string;
  name: string;
}

export type UserCreateInput = Omit<User, 'id'>;
```

## 2. 组件规范

### 2.1 组件文件结构

```typescript
// components/UserCard.tsx

// 1. 导入
import React from 'react';
import type { User } from '../types/user';

// 2. 类型定义
interface UserCardProps {
  user: User;
  onEdit: (id: string) => void;
}

// 3. 组件
export const UserCard: React.FC<UserCardProps> = ({ user, onEdit }) => {
  // ...
};

// 4. 导出
export default UserCard;
```

### 2.2 组件命名
- **文件**：`PascalCase.tsx`
- **组件**：`PascalCase`
- **函数**：`camelCase`

## 3. React Hooks

### 3.1 Hooks 规则
- 只在组件顶层调用
- 不在条件语句中调用
- 不在循环中调用

### 3.2 自定义 Hooks

```typescript
// hooks/useUser.ts
export const useUser = (userId: string) => {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchUser(userId).then(setUser).finally(() => setLoading(false));
  }, [userId]);

  return { user, loading };
};
```

### 3.3 常用 Hooks
```typescript
// 状态
const [state, setState] = useState<T>(initial);

// Effect
useEffect(() => {
  // cleanup
  return () => {};
}, [deps]);

// 回调
const handleClick = useCallback(() => {
  // ...
}, [deps]);

// Memo
const memoized = useMemo(() => computeExpensiveValue(a, b), [a, b]);

// Ref
const ref = useRef<T>(initial);
```

## 4. 导入导出

### 4.1 导入顺序
```typescript
// 1. React
import React from 'react';

// 2. 第三方库
import { Button } from 'antd';
import { useQuery } from 'react-query';

// 3. 内部模块
import { useAuthStore } from '../stores/authStore';
import type { User } from '../types/user';
import { api } from '../api/client';

// 4. 相对导入
import { UserCard } from './UserCard';
```

### 4.2 导出
```typescript
// 命名导出
export const UserCard: React.FC = () => {};
export { UserCard };

// 默认导出
export default UserCard;
```

## 5. 异步处理

### 5.1 async/await
```typescript
// ✅ 正确
async function fetchUser(id: string): Promise<User> {
  const response = await api.get(`/users/${id}`);
  return response.data;
}

// ❌ 错误
function fetchUser(id: string): Promise<User> {
  return api.get(`/users/${id}`).then(res => res.data);
}
```

### 5.2 错误处理
```typescript
try {
  const user = await fetchUser(id);
  setData(user);
} catch (error) {
  if (error instanceof ApiError) {
    message.error(error.message);
  }
}
```

## 6. 样式

### 6.1 CSS-in-JS
使用 Ant Design 的 `styled` 或内联样式：

```typescript
const Container = styled.div`
  padding: 16px;
  background: #fff;
`;

const InlineStyle = () => (
  <div style={{ padding: '16px', background: '#fff' }} />
);
```

### 6.2 类名
使用 `className` + CSS 模块或 Tailwind：

```tsx
<div className="p-4 bg-white">
  <Button type="primary">提交</Button>
</div>
```

## 7. 常量

```typescript
// constants/api.ts
export const API_BASE_URL = '/api/v1';
export const DEFAULT_PAGE_SIZE = 20;

// constants/status.ts
export const UserStatus = {
  ACTIVE: 'active',
  INACTIVE: 'inactive',
} as const;

export type UserStatus = typeof UserStatus[keyof typeof UserStatus];
```

## 8. 工具函数

```typescript
// utils/format.ts
export const formatDate = (date: Date): string => {
  return new Intl.DateTimeFormat('zh-CN').format(date);
};

// utils/validation.ts
export const isEmail = (email: string): boolean => {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
};
```

## 9. 枚举

```typescript
// 推荐：const 对象
enum Status {
  Active = 'active',
  Inactive = 'inactive',
}

// 或 as const
const Status = {
  Active: 'active',
  Inactive: 'inactive',
} as const;
type Status = typeof Status[keyof typeof Status];
```

## 10. import 类型

```typescript
// 导入类型（编译时去除）
import type { User } from './types';

// 导入值
import { UserCard } from './components';

// 两者都有
import { useState } from 'react';  // 值
import type { User } from './types';  // 类型
```
