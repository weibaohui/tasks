/**
 * 任务表单组件
 * 用于创建新任务
 */
import React, { useEffect, useState } from 'react';
import { Form, Input, Select, InputNumber, Button, Space } from 'antd';
import type { CreateTaskRequest, Task } from '../../types/task';
import { useTaskStore } from '../../stores/taskStore';
import { listUsers } from '../../api/userApi';
import { listChannels } from '../../api/channelApi';
import type { User } from '../../types/user';
import type { Channel } from '../../types/channel';

const { TextArea } = Input;

interface TaskFormProps {
  onSubmit: (values: CreateTaskRequest) => void;
  onCancel: () => void;
  loading?: boolean;
}

export const TaskForm: React.FC<TaskFormProps> = ({ onSubmit, onCancel, loading }) => {
  const [form] = Form.useForm<CreateTaskRequest>();
  const { tasks, fetchTasks } = useTaskStore();
  const [users, setUsers] = useState<User[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [selectedUser, setSelectedUser] = useState<string | undefined>();
  const [loadingUsers, setLoadingUsers] = useState(false);
  const [loadingChannels, setLoadingChannels] = useState(false);

  useEffect(() => {
    fetchTasks();
    fetchUsers();
  }, [fetchTasks]);

  const fetchUsers = async () => {
    setLoadingUsers(true);
    try {
      const userList = await listUsers();
      setUsers(userList);
    } catch (error) {
      console.error('获取用户列表失败:', error);
    } finally {
      setLoadingUsers(false);
    }
  };

  const fetchChannelsForUser = async (userCode: string) => {
    setLoadingChannels(true);
    try {
      const channelList = await listChannels(userCode);
      setChannels(channelList.filter((c) => c.is_active));
    } catch (error) {
      console.error('获取渠道列表失败:', error);
    } finally {
      setLoadingChannels(false);
    }
  };

  const handleUserChange = (userCode: string) => {
    setSelectedUser(userCode);
    setChannels([]);
    form.setFieldValue('channel_code', undefined);
    fetchChannelsForUser(userCode);
  };

  const handleFinish = (values: CreateTaskRequest): void => {
    onSubmit({
      ...values,
      type: 'agent',
      timeout: (values.timeout || 600) * 1e9,
      max_retries: values.max_retries || 0,
      priority: values.priority || 0,
      metadata: {
        ...values.metadata,
        user_code: values.user_code,
        channel_code: values.channel_code,
      },
    });
  };

  const pendingTasks = tasks.filter((t: Task) => t.status === 'pending' || t.status === 'running');

  return (
    <Form
      form={form}
      layout="vertical"
      onFinish={handleFinish}
      initialValues={{ type: 'agent' }}
    >
      <Form.Item
        name="name"
        label="任务名称"
        rules={[{ required: true, message: '请输入任务名称' }]}
      >
        <Input placeholder="请输入任务名称" />
      </Form.Item>

      <Form.Item name="description" label="任务描述">
        <TextArea placeholder="请输入任务描述" rows={3} />
      </Form.Item>

      <Form.Item name="type" label="任务类型">
        <Select disabled>
          <Select.Option value="agent">Agent</Select.Option>
        </Select>
      </Form.Item>

      <Form.Item name="user_code" label="用户" rules={[{ required: true, message: '请选择用户' }]}>
        <Select
          placeholder="请选择用户"
          allowClear
          loading={loadingUsers}
          onChange={handleUserChange}
          showSearch
          optionFilterProp="children"
        >
          {users.map((user) => (
            <Select.Option key={user.user_code} value={user.user_code}>
              {user.display_name || user.username} ({user.username})
            </Select.Option>
          ))}
        </Select>
      </Form.Item>

      <Form.Item name="channel_code" label="渠道" dependencies={['user_code']}>
        <Select
          placeholder="请先选择用户"
          allowClear
          loading={loadingChannels}
          disabled={!selectedUser}
          showSearch
          optionFilterProp="children"
        >
          {channels.map((channel) => (
            <Select.Option key={channel.channel_code} value={channel.channel_code}>
              {channel.name} ({channel.type})
            </Select.Option>
          ))}
        </Select>
      </Form.Item>

      <Form.Item name="parent_id" label="父任务 (可选)">
        <Select placeholder="无父任务" allowClear>
          {pendingTasks.map((task: Task) => (
            <Select.Option key={task.id} value={task.id}>
              {task.name} ({task.status})
            </Select.Option>
          ))}
        </Select>
      </Form.Item>

      <Form.Item name="timeout" label="超时时间 (秒)">
        <InputNumber min={1} step={10} defaultValue={600} style={{ width: '100%' }} />
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
