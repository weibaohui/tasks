import React, { useEffect, useState } from 'react';
import { Button, Card, Form, Input, Modal, Popconfirm, Space, Switch, Table, message } from 'antd';
import { createUser, deleteUser, listUsers, updateUser } from '../api/userApi';
import type { User } from '../types/user';
import { ActionGroup } from "@/components/ActionGroup";

export const UserManagementPage: React.FC = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();

  const fetchUsers = async () => {
    setLoading(true);
    try {
      const data = await listUsers();
      setUsers(data);
    } catch (_error) {
      message.error('获取用户列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, []);

  const handleCreate = async (values: { username: string; display_name: string; email: string; password: string }) => {
    try {
      await createUser(values);
      message.success('创建用户成功');
      setCreateOpen(false);
      createForm.resetFields();
      fetchUsers();
    } catch (_error) {
      message.error('创建用户失败');
    }
  };

  const handleEdit = (user: User) => {
    setEditingUser(user);
    editForm.setFieldsValue({
      display_name: user.display_name,
      email: user.email,
      is_active: user.is_active,
    });
  };

  const handleUpdate = async (values: { display_name: string; email: string; is_active: boolean }) => {
    if (!editingUser) {
      return;
    }
    try {
      await updateUser(editingUser.id, values);
      message.success('更新用户成功');
      setEditingUser(null);
      fetchUsers();
    } catch (_error) {
      message.error('更新用户失败');
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteUser(id);
      message.success('删除用户成功');
      fetchUsers();
    } catch (_error) {
      message.error('删除用户失败');
    }
  };

  const columns = [
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
    },
    {
      title: '显示名',
      dataIndex: 'display_name',
      key: 'display_name',
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      render: (isActive: boolean) => (isActive ? '启用' : '停用'),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time: number) => new Date(time).toLocaleString(),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: User) => (
        <ActionGroup>
          <Button onClick={() => handleEdit(record)} type="link" size="small" style={{ padding: 0 }}>
            编辑
          </Button>
          <Popconfirm title="确认删除该用户？" onConfirm={() => handleDelete(record.id)}>
            <Button danger type="link" size="small" style={{ padding: 0 }}>
              删除
            </Button>
          </Popconfirm>
        </ActionGroup>
      ),
        width: 100,
        fixed: 'left' as const
    },
  ];

  return (
    <div style={{ padding: 0 }}>
      <Card
        title={`用户管理 (${users.length})`}
        extra={
          <Space>
            <Button onClick={() => fetchUsers()}>刷新</Button>
            <Button type="primary" onClick={() => setCreateOpen(true)}>
              新建用户
            </Button>
          </Space>
        }
      >
        <Table<User> rowKey="id" loading={loading} dataSource={users} columns={columns} />
      </Card>

      <Modal title="新建用户" open={createOpen} footer={null} onCancel={() => setCreateOpen(false)}>
        <Form layout="vertical" form={createForm} onFinish={handleCreate}>
          <Form.Item label="用户名" name="username" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input />
          </Form.Item>
          <Form.Item label="显示名" name="display_name">
            <Input />
          </Form.Item>
          <Form.Item label="邮箱" name="email">
            <Input />
          </Form.Item>
          <Form.Item label="密码" name="password" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password />
          </Form.Item>
          <Button type="primary" htmlType="submit" block>
            创建
          </Button>
        </Form>
      </Modal>

      <Modal title="编辑用户" open={!!editingUser} footer={null} onCancel={() => setEditingUser(null)}>
        <Form layout="vertical" form={editForm} onFinish={handleUpdate}>
          <Form.Item label="显示名" name="display_name">
            <Input />
          </Form.Item>
          <Form.Item label="邮箱" name="email">
            <Input />
          </Form.Item>
          <Form.Item label="启用状态" name="is_active" valuePropName="checked">
            <Switch checkedChildren="启用" unCheckedChildren="停用" />
          </Form.Item>
          <Button type="primary" htmlType="submit" block>
            保存
          </Button>
        </Form>
      </Modal>
    </div>
  );
};
