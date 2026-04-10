/**
 * 需求状态中文名称映射
 */
export const statusLabels: Record<string, string> = {
  todo: '待处理',
  preparing: '准备中',
  understanding: '理解需求',
  analyzing: '分析方案',
  implementing: '编写代码',
  submitting: '提交PR',
  coding: '编码中',
  pr_opened: 'PR已开',
  failed: '失败',
  completed: '已完成',
  done: '完成',
};

/**
 * 状态颜色配置
 */
export const statusColors: Record<string, { color: string; bgColor: string; borderColor: string }> = {
  todo: { color: '#666666', bgColor: '#f5f5f5', borderColor: '#d9d9d9' },
  preparing: { color: '#d48806', bgColor: '#fffbe6', borderColor: '#ffd666' },
  understanding: { color: '#722ed1', bgColor: '#f9f0ff', borderColor: '#d3adf7' },
  analyzing: { color: '#eb2f96', bgColor: '#fff0f6', borderColor: '#ffadd2' },
  implementing: { color: '#0958d9', bgColor: '#e6f4ff', borderColor: '#91caff' },
  submitting: { color: '#fa8c16', bgColor: '#fff7e6', borderColor: '#ffd591' },
  coding: { color: '#0958d9', bgColor: '#e6f4ff', borderColor: '#91caff' },
  pr_opened: { color: '#389e0d', bgColor: '#f6ffed', borderColor: '#b7eb8f' },
  failed: { color: '#cf1322', bgColor: '#fff2f0', borderColor: '#ffccc7' },
  completed: { color: '#52c41a', bgColor: '#f6ffed', borderColor: '#b7eb8f' },
  done: { color: '#237804', bgColor: '#d9f7be', borderColor: '#95de64' },
};

/**
 * 获取状态默认颜色
 */
export function getStatusColor(status: string) {
  return statusColors[status] || { color: '#8c8c8c', bgColor: '#f5f5f5', borderColor: '#d9d9d9' };
}

/**
 * 获取状态中文名称
 */
export function getStatusLabel(status: string): string {
  return statusLabels[status] || status;
}

/**
 * 看板三态分组定义
 * 所有需求按 todo / processing / complete 三态分组显示
 */
export interface StatusGroup {
  key: string;
  label: string;
  statuses: string[];
  color: { color: string; bgColor: string; borderColor: string };
}

export const statusGroups: StatusGroup[] = [
  {
    key: 'todo',
    label: '待办',
    statuses: ['todo'],
    color: { color: '#666666', bgColor: '#f5f5f5', borderColor: '#d9d9d9' },
  },
  {
    key: 'processing',
    label: '处理中',
    statuses: [
      'preparing', 'understanding', 'analyzing', 'implementing',
      'submitting', 'coding', 'pr_opened', 'failed',
    ],
    color: { color: '#0958d9', bgColor: '#e6f4ff', borderColor: '#91caff' },
  },
  {
    key: 'complete',
    label: '已完成',
    statuses: ['completed', 'done'],
    color: { color: '#389e0d', bgColor: '#f6ffed', borderColor: '#b7eb8f' },
  },
];

/** 根据实际状态找到所属分组 */
export function getStatusGroup(status: string): StatusGroup {
  return statusGroups.find((g) => g.statuses.includes(status)) ?? statusGroups[0];
}
