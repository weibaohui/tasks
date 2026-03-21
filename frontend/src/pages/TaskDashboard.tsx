/**
 * 任务仪表板页面
 * 只显示根任务，点击弹出详情抽屉
 */
import React, { useEffect, useState } from 'react';
import { Row, Col, Card, Statistic, Button, Space, Table, Tag, Modal } from 'antd';
import { PlusOutlined, ReloadOutlined, EyeOutlined } from '@ant-design/icons';
import { TaskForm } from '../components/TaskForm';
import { TaskDetailDrawer } from '../components/TaskDetailDrawer';
import { StatusBadge } from '../components/StatusBadge';
import { useTaskStore } from '../stores/taskStore';
import { useTaskOperations } from '../hooks/useTaskOperations';
import type { Task, TaskStatus } from '../types/task';

export const TaskDashboard: React.FC = () => {
  const { tasks, loading, fetchTasks } = useTaskStore();
  const { createTask, cancelTask } = useTaskOperations();
  const [modalVisible, setModalVisible] = useState(false);
  const [drawerTaskId, setDrawerTaskId] = useState<string | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);

  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  const rootTasks = tasks.filter((t) => !t.parent_id);

  const handleCreateTask = async (values: Parameters<typeof createTask>[0]) => {
    const result = await createTask(values);
    if (result) {
      setModalVisible(false);
      fetchTasks();
    }
  };

  const handleCancelTask = async (taskId: string) => {
    const success = await cancelTask(taskId);
    if (success) {
      fetchTasks();
    }
  };

  const handleViewDetail = (taskId: string) => {
    setDrawerTaskId(taskId);
    setDrawerOpen(true);
  };

  const handleDrawerClose = () => {
    setDrawerOpen(false);
    setDrawerTaskId(null);
  };

  const statusCounts = {
    pending: rootTasks.filter((t) => t.status === 'pending').length,
    running: rootTasks.filter((t) => t.status === 'running').length,
    completed: rootTasks.filter((t) => t.status === 'completed').length,
    failed: rootTasks.filter((t) => t.status === 'failed').length,
  };

  const columns = [
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => <StatusBadge status={status as TaskStatus} />,
    },
    {
      title: '任务名称',
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 120,
      render: (type: string) => <Tag color="blue">{type}</Tag>,
    },
    {
      title: '进度',
      key: 'progress',
      width: 150,
      render: (_: unknown, record: Task) => {
        const p = record.progress?.percentage || 0;
        return (
          <div style={{ width: 100 }}>
            <div style={{ fontSize: 12, marginBottom: 4 }}>
              {p}%
            </div>
            <div style={{ background: '#f0f0f0', height: 6, borderRadius: 3 }}>
              <div
                style={{
                  width: `${p}%`,
                  background: p === 100 ? '#52c41a' : '#1890ff',
                  height: 6,
                  borderRadius: 3,
                  transition: 'width 0.3s',
                }}
              />
            </div>
          </div>
        );
      },
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (time: number) => new Date(time).toLocaleString(),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: unknown, record: Task) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => handleViewDetail(record.id)}
          >
            查看
          </Button>
          {record.status === 'pending' && (
            <Button type="link" size="small" danger onClick={() => handleCancelTask(record.id)}>
              取消
            </Button>
          )}
        </Space>
      ),
    },
  ];

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
        title={`根任务列表 (${rootTasks.length})`}
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => fetchTasks()}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalVisible(true)}>
              创建任务
            </Button>
          </Space>
        }
      >
        <Table
          dataSource={rootTasks}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 10 }}
          size="middle"
        />
      </Card>

      <Modal
        title="创建任务"
        open={modalVisible}
        footer={null}
        onCancel={() => setModalVisible(false)}
        width={600}
      >
        <TaskForm onSubmit={handleCreateTask} onCancel={() => setModalVisible(false)} />
      </Modal>

      <TaskDetailDrawer taskId={drawerTaskId} open={drawerOpen} onClose={handleDrawerClose} />
    </div>
  );
};
