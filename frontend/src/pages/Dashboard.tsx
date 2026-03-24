/**
 * Dashboard 页面
 * 显示任务统计概览
 */
import React, { useEffect } from 'react';
import { Row, Col, Card, Statistic } from 'antd';
import { useTaskStore } from '../stores/taskStore';

export const Dashboard: React.FC = () => {
  const { tasks, fetchTasks } = useTaskStore();

  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  const rootTasks = tasks.filter((t) => !t.parent_id);

  const statusCounts = {
    pending: rootTasks.filter((t) => t.status === 'pending').length,
    running: rootTasks.filter((t) => t.status === 'running').length,
    completed: rootTasks.filter((t) => t.status === 'completed').length,
    failed: rootTasks.filter((t) => t.status === 'failed').length,
  };

  return (
    <div style={{ padding: 24 }}>
      <div style={{ marginBottom: 24 }}>
        <h2 style={{ margin: 0 }}>Dashboard</h2>
      </div>

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
    </div>
  );
};