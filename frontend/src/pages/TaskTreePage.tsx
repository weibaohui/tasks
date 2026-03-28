/**
 * 任务树页面
 * 以树形结构展示父子任务关系
 */
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
    <div style={{ padding: 0 }}>
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
