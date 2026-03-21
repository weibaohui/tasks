/**
 * 任务仪表板页面
 * 显示任务统计和列表
 */
import React, { useEffect, useState } from 'react';
import { Row, Col, Card, Statistic, Button, Space, Modal, message } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { TaskList } from '../components/TaskList';
import { TaskForm } from '../components/TaskForm';
import { useTaskWebSocket } from '../hooks/useTaskWebSocket';
import { useTaskStore } from '../stores/taskStore';
import { useTaskOperations } from '../hooks/useTaskOperations';
import * as taskApi from '../api/taskApi';

export const TaskDashboard: React.FC = () => {
  const traceId = 'default-trace-id';
  const { tasks, loading, fetchTasks } = useTaskStore();
  const { createTask, cancelTask } = useTaskOperations();
  const [modalVisible, setModalVisible] = useState(false);

  useTaskWebSocket(traceId);

  useEffect(() => {
    fetchTasks(traceId);
  }, [fetchTasks, traceId]);

  const handleCreateTask = async (values: Parameters<typeof createTask>[0]) => {
    const result = await createTask({ ...values, trace_id: traceId });
    if (result) {
      setModalVisible(false);
      fetchTasks(traceId);
    }
  };

  const handleCancelTask = async (taskId: string) => {
    const success = await cancelTask(taskId);
    if (success) {
      fetchTasks(traceId);
    }
  };

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
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalVisible(true)}>
              创建任务
            </Button>
          </Space>
        }
      >
        <TaskList tasks={tasks} loading={loading} onCancel={handleCancelTask} />
      </Card>

      <Modal
        title="创建任务"
        open={modalVisible}
        footer={null}
        onCancel={() => setModalVisible(false)}
      >
        <TaskForm
          onSubmit={handleCreateTask}
          onCancel={() => setModalVisible(false)}
        />
      </Modal>
    </div>
  );
};
