/**
 * 任务树组件
 * 以树形结构展示父子任务关系
 */
import React from 'react';
import { Tree } from 'antd';
import type { DataNode } from 'antd/es/tree';
import { StatusBadge } from '../StatusBadge';
import type { TaskTreeNode } from '../../types/task';

interface TaskTreeProps {
  treeData: TaskTreeNode[];
  onSelect?: (taskId: string) => void;
}

export const TaskTree: React.FC<TaskTreeProps> = ({ treeData, onSelect }) => {
  const convertToTreeData = (nodes: TaskTreeNode[]): DataNode[] => {
    return nodes.map((node) => ({
      key: node.task.id,
      title: (
        <span>
          <StatusBadge status={node.task.status} />
          {' '}
          {node.task.name}
        </span>
      ),
      children: node.children ? convertToTreeData(node.children) : undefined,
    }));
  };

  return (
    <Tree
      treeData={convertToTreeData(treeData)}
      onSelect={(selectedKeys) => {
        if (selectedKeys.length > 0) {
          onSelect?.(selectedKeys[0] as string);
        }
      }}
    />
  );
};
