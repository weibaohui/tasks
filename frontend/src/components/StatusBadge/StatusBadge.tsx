/**
 * 状态徽章组件
 * 根据任务状态显示不同颜色的标签
 */
import React from 'react';
import { Tag } from 'antd';
import type { TaskStatus } from '../../types/task';

const statusConfig: Record<TaskStatus, { color: string; label: string }> = {
  pending: { color: 'default', label: '待处理' },
  running: { color: 'processing', label: '运行中' },
  completed: { color: 'success', label: '已完成' },
  failed: { color: 'error', label: '失败' },
  cancelled: { color: 'warning', label: '已取消' },
};

interface StatusBadgeProps {
  status: TaskStatus;
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({ status }) => {
  const config = statusConfig[status] || statusConfig.pending;
  return <Tag color={config.color}>{config.label}</Tag>;
};
