/**
 * 任务表单组件
 * 用于创建新任务
 */
import React, { useEffect } from 'react';
import { Form, Input, Select, InputNumber, Button, Space } from 'antd';
import type { CreateTaskRequest, Task } from '../../types/task';
import { useTaskStore } from '../../stores/taskStore';

const { TextArea } = Input;

interface TaskFormProps {
  onSubmit: (values: CreateTaskRequest) => void;
  onCancel: () => void;
  loading?: boolean;
}

export const TaskForm: React.FC<TaskFormProps> = ({ onSubmit, onCancel, loading }) => {
  const [form] = Form.useForm<CreateTaskRequest>();
  const { tasks, fetchTasks } = useTaskStore();

  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  const handleFinish = (values: CreateTaskRequest): void => {
    onSubmit({
      ...values,
      timeout: values.timeout || 60000,
      max_retries: values.max_retries || 0,
      priority: values.priority || 0,
      metadata: values.metadata || {},
    });
  };

  const pendingTasks = tasks.filter((t: Task) => t.status === 'pending' || t.status === 'running');

  return (
    <Form
      form={form}
      layout="vertical"
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
        <TextArea placeholder="请输入任务描述" rows={3} />
      </Form.Item>

      <Form.Item
        name="type"
        label="任务类型"
        rules={[{ required: true, message: '请选择任务类型' }]}
      >
        <Select placeholder="请选择任务类型">
          <Select.Option value="data_processing">数据处理</Select.Option>
          <Select.Option value="file_operation">文件操作</Select.Option>
          <Select.Option value="api_call">API 调用</Select.Option>
          <Select.Option value="custom">自定义</Select.Option>
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
