# React 组件规范

## 1. 组件分类

| 类型 | 说明 | 示例 |
|------|------|------|
| Page | 页面组件 | `UserManagementPage` |
| Container | 容器组件 | `UserList` |
| Presentational | 展示组件 | `UserCard` |

## 2. 组件结构

```tsx
// components/UserCard.tsx

// 1. 导入
import React from 'react';
import { Button, Card } from 'antd';
import type { User } from '../types/user';

// 2. Props 类型定义
interface UserCardProps {
  user: User;
  onEdit?: (user: User) => void;
  onDelete?: (id: string) => void;
}

// 3. 组件定义
export const UserCard: React.FC<UserCardProps> = ({
  user,
  onEdit,
  onDelete,
}) => {
  // 4. Hooks（状态、副作用）
  // 5. 回调函数
  // 6. 渲染

  return (
    <Card>
      <UserCardContent user={user} />
      <Button onClick={() => onEdit?.(user)}>编辑</Button>
    </Card>
  );
};

// 7. 子组件（如果需要）
const UserCardContent: React.FC<{ user: User }> = ({ user }) => {
  return <div>{user.name}</div>;
};
```

## 3. Props 规范

### 3.1 Props 类型
```tsx
// ✅ 使用 interface
interface ButtonProps {
  label: string;
  onClick: () => void;
  variant?: 'primary' | 'secondary';
}

// ✅ 使用 type
type UserCardProps = {
  user: User;
  onSelect: (user: User) => void;
};
```

### 3.2 默认值
```tsx
interface ButtonProps {
  label: string;
  onClick: () => void;
  disabled?: boolean;
}

// 使用默认值
const Button: React.FC<ButtonProps> = ({
  label,
  onClick,
  disabled = false,
}) => {
  // ...
};
```

## 4. 状态管理

### 4.1 Local State
```tsx
// useState
const [count, setCount] = useState(0);
const [user, setUser] = useState<User | null>(null);
```

### 4.2 Global State（Zustand）
```tsx
// stores/authStore.ts
import { create } from 'zustand';

interface AuthState {
  user: User | null;
  token: string | null;
  login: (user: User, token: string) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  token: null,
  login: (user, token) => set({ user, token }),
  logout: () => set({ user: null, token: null }),
}));

// 使用
const { user, login } = useAuthStore();
```

## 5. 副作用（useEffect）

### 5.1 数据获取
```tsx
useEffect(() => {
  let cancelled = false;

  const fetchData = async () => {
    const data = await api.getUser(userId);
    if (!cancelled) {
      setUser(data);
    }
  };

  fetchData();

  return () => {
    cancelled = true;
  };
}, [userId]);
```

### 5.2 订阅
```tsx
useEffect(() => {
  const subscription = subscribe(handleEvent);

  return () => {
    subscription.unsubscribe();
  };
}, [handleEvent]);
```

## 6. 性能优化

### 6.1 React.memo
```tsx
const UserCard = React.memo<UserCardProps>(({ user }) => {
  return <div>{user.name}</div>;
});
```

### 6.2 useCallback
```tsx
const handleEdit = useCallback((user: User) => {
  setEditingUser(user);
}, []);

const handleDelete = useCallback((id: string) => {
  deleteUser(id);
}, [deleteUser]);
```

### 6.3 useMemo
```tsx
const sortedUsers = useMemo(() => {
  return [...users].sort((a, b) => a.name.localeCompare(b.name));
}, [users]);
```

## 7. 条件渲染

```tsx
// ✅ 正确：三元表达式
{isLoading ? <Spinner /> : <UserList users={users} />}

// ✅ 正确：&& 运算符
{hasPermission && <AdminPanel />}

// ❌ 错误：if 在组件中
const UserList = ({ users }) => {
  if (!users) return <Spinner />;  // 应该用条件表达式
  return <List users={users} />;
};
```

## 8. 列表渲染

```tsx
// ✅ 正确：提供 key
{users.map((user) => (
  <UserCard key={user.id} user={user} />
))}

// ✅ 正确：解构
{users.map(({ id, name }) => (
  <div key={id}>{name}</div>
))}
```

## 9. 表单处理

### 9.1 使用 Ant Design Form
```tsx
const [form] = Form.useForm<LoginFormValues>();

const onFinish = (values: LoginFormValues) => {
  login(values);
};

<Form form={form} onFinish={onFinish}>
  <Form.Item name="username" rules={[{ required: true }]}>
    <Input />
  </Form.Item>
</Form>
```

## 10. 组件文件组织

```
src/
├── components/
│   ├── UserCard/
│   │   ├── UserCard.tsx
│   │   ├── UserCard.test.tsx
│   │   └── index.ts
│   └── Button/
│       └── index.ts
├── pages/
│   └── UserManagementPage.tsx
├── hooks/
│   └── useUser.ts
├── stores/
│   └── authStore.ts
└── types/
    └── user.ts
```

## 11. 组件测试

```tsx
// UserCard.test.tsx
import { render, screen } from '@testing-library/react';
import { UserCard } from './UserCard';

test('renders user name', () => {
  const user = { id: '1', name: '张三' };
  render(<UserCard user={user} />);
  expect(screen.getByText('张三')).toBeInTheDocument();
});
```
