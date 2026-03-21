/**
 * 进度条组件
 * 显示任务执行进度
 */
import React from 'react';
import { Progress } from 'antd';
import type { Progress as ProgressType } from '../../types/task';

interface ProgressBarProps {
  progress: ProgressType;
  showInfo?: boolean;
}

export const ProgressBar: React.FC<ProgressBarProps> = ({ progress, showInfo = true }) => {
  const formatPercent = (percent?: number): string => {
    return `${Math.round(percent || 0)}%`;
  };

  return (
    <Progress
      percent={progress.percentage}
      status={progress.percentage >= 100 ? 'success' : 'active'}
      format={showInfo ? formatPercent : undefined}
      strokeColor="#1890ff"
    />
  );
};
