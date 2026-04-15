import React, { useCallback, useEffect, useState, useMemo } from 'react';
import { message } from 'antd';
import { listRequirementsPaginated, type StatusStat } from '../../api/projectRequirementApi';
import type { Requirement } from '../../types/projectRequirement';
import type { RequirementType } from '../../api/requirementTypeApi';
import { getStatusLabel, getStatusColor } from '../../constants/requirementStatus';
import { KanbanColumn } from './KanbanColumn';

interface RequirementKanbanProps {
  projectId: string;
  statusStats: StatusStat[];
  onRequirementClick: (requirement: Requirement) => void;
  refreshTrigger: number;
  onRefresh: () => void;
  requirementTypes: RequirementType[];
}

interface ColumnState {
  requirements: Requirement[];
  total: number;
  loaded: number;
  loading: boolean;
}

type ColumnsMap = Record<string, ColumnState>;

// 定义大分组结构
type MajorGroupKey = 'todo' | 'processing' | 'complete';
interface KanbanStatusDef {
  status: string;
  label: string;
  color: { color: string; bgColor: string; borderColor: string };
}
interface KanbanGroupDef {
  key: MajorGroupKey;
  label: string;
  color: { color: string; bgColor: string; borderColor: string };
  statuses: KanbanStatusDef[];
}

const PAGE_SIZE = 10;

/**
 * 动态计算看板分组结构
 * 逻辑：从数据库返回的现有状态统计(statusStats)中提取。
 * 1. todo 归为 "待办"
 * 2. completed, done 归为 "已完成"
 * 3. 其他所有状态均归为 "处理中" 的子状态
 */
function buildDynamicGroups(statusStats: StatusStat[]): KanbanGroupDef[] {
  const todoStatuses: KanbanStatusDef[] = [];
  const processingStatuses: KanbanStatusDef[] = [];
  const completeStatuses: KanbanStatusDef[] = [];

  // 遍历实际存在数据的状态，进行归类
  statusStats.forEach((stat) => {
    const s = stat.status;
    const def: KanbanStatusDef = {
      status: s,
      label: getStatusLabel(s),
      color: getStatusColor(s),
    };

    if (s === 'todo') {
      if (!todoStatuses.some((x) => x.status === s)) todoStatuses.push(def);
    } else if (s === 'completed' || s === 'done') {
      if (!completeStatuses.some((x) => x.status === s)) completeStatuses.push(def);
    } else {
      if (!processingStatuses.some((x) => x.status === s)) processingStatuses.push(def);
    }
  });

  return [
    {
      key: 'todo',
      label: '待办',
      color: { color: '#666666', bgColor: '#f5f5f5', borderColor: '#d9d9d9' },
      statuses: todoStatuses,
    },
    {
      key: 'processing',
      label: '处理中',
      color: { color: '#0958d9', bgColor: '#e6f4ff', borderColor: '#91caff' },
      statuses: processingStatuses,
    },
    {
      key: 'complete',
      label: '已完成',
      color: { color: '#389e0d', bgColor: '#f6ffed', borderColor: '#b7eb8f' },
      statuses: completeStatuses,
    },
  ];
}

/** 根据 statusStats 计算每个子状态的总数 */
function computeStatusTotals(statusStats: StatusStat[]): Record<string, number> {
  const totals: Record<string, number> = {};
  statusStats.forEach((stat) => {
    totals[stat.status] = stat.count;
  });
  return totals;
}

export const RequirementKanban: React.FC<RequirementKanbanProps> = ({
  projectId,
  statusStats,
  onRequirementClick,
  refreshTrigger,
  requirementTypes,
}) => {
  const [columns, setColumns] = useState<ColumnsMap>({});
  const [initialLoading, setInitialLoading] = useState(false);

  // 动态生成分组结构
  const dynamicGroups = useMemo(() => buildDynamicGroups(statusStats), [statusStats]);

  const loadColumn = useCallback(async (status: string, offset: number) => {
    setColumns((prev) => ({
      ...prev,
      [status]: { ...prev[status], loading: true },
    }));

    try {
      const data = await listRequirementsPaginated({
        projectId,
        statuses: [status],
        limit: PAGE_SIZE,
        offset,
      });
      setColumns((prev) => {
        const existing = prev[status]?.requirements ?? [];
        return {
          ...prev,
          [status]: {
            requirements: offset === 0 ? data.items : [...existing, ...data.items],
            total: data.total,
            loaded: offset === 0 ? data.items.length : (prev[status]?.loaded ?? 0) + data.items.length,
            loading: false,
          },
        };
      });
    } catch {
      message.error(`加载数据失败: ${getStatusLabel(status)}`);
      setColumns((prev) => ({
        ...prev,
        [status]: { ...prev[status], loading: false },
      }));
    }
  }, [projectId]);

  const loadAllColumns = useCallback(async () => {
    if (!projectId) return;

    setInitialLoading(true);
    const statusTotals = computeStatusTotals(statusStats);
    const newColumns: ColumnsMap = {};
    
    // 初始化每个状态的数据结构
    dynamicGroups.forEach((group) => {
      group.statuses.forEach((statusDef) => {
        newColumns[statusDef.status] = {
          requirements: [],
          total: statusTotals[statusDef.status] ?? 0,
          loaded: 0,
          loading: false,
        };
      });
    });
    setColumns(newColumns);

    // 发起所有状态的首屏请求
    const promises: Promise<void>[] = [];
    dynamicGroups.forEach((group) => {
      group.statuses.forEach((statusDef) => {
        promises.push(loadColumn(statusDef.status, 0));
      });
    });

    await Promise.all(promises);
    setInitialLoading(false);
  }, [projectId, statusStats, loadColumn, dynamicGroups]);

  useEffect(() => {
    loadAllColumns();
  }, [loadAllColumns, refreshTrigger]);

  const handleLoadMore = (status: string) => {
    const col = columns[status];
    if (!col) return;
    loadColumn(status, col.loaded);
  };

  return (
    <div style={{
      display: 'flex',
      gap: 16,
      overflowX: 'auto',
      padding: '4px 0',
      minHeight: initialLoading ? 300 : undefined,
      opacity: initialLoading ? 0.6 : 1,
      transition: 'opacity 0.3s',
    }}>
      {dynamicGroups.map((group) => {
        // 计算这个大组包含的需求总数
        const groupTotal = group.statuses.reduce((acc, def) => acc + (columns[def.status]?.total ?? 0), 0);

        return (
          <div key={group.key} style={{
            display: 'flex',
            flexDirection: 'column',
            backgroundColor: '#fafafa',
            borderRadius: 12,
            padding: '16px',
            border: '1px solid #f0f0f0',
            minWidth: 'fit-content',
          }}>
            <div style={{
              marginBottom: 16,
              fontWeight: 600,
              fontSize: 16,
              color: group.color.color,
              display: 'flex',
              alignItems: 'center',
              gap: 8,
            }}>
              <span>{group.label}</span>
              <span style={{
                backgroundColor: group.color.bgColor,
                border: `1px solid ${group.color.borderColor}`,
                padding: '2px 8px',
                borderRadius: '12px',
                fontSize: 12,
              }}>
                {groupTotal}
              </span>
            </div>
            <div style={{ display: 'flex', gap: 12 }}>
              {group.statuses.map((statusDef) => (
                <KanbanColumn
                  key={statusDef.status}
                  groupKey={statusDef.status}
                  label={statusDef.label}
                  groupColor={statusDef.color}
                  totalCount={columns[statusDef.status]?.total ?? 0}
                  groupTotal={group.key === 'processing' ? groupTotal : undefined}
                  requirements={columns[statusDef.status]?.requirements ?? []}
                  loadedCount={columns[statusDef.status]?.loaded ?? 0}
                  loading={columns[statusDef.status]?.loading ?? false}
                  onLoadMore={() => handleLoadMore(statusDef.status)}
                  onCardClick={onRequirementClick}
                  requirementTypes={requirementTypes}
                />
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
};
