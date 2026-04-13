import React, { useCallback, useEffect, useState } from 'react';
import { message } from 'antd';
import { listRequirementsPaginated, type StatusStat } from '../../api/projectRequirementApi';
import type { Requirement } from '../../types/projectRequirement';
import { statusGroups, getStatusLabel, getStatusColor } from '../../constants/requirementStatus';
import { KanbanColumn } from './KanbanColumn';

interface RequirementKanbanProps {
  projectId: string;
  statusStats: StatusStat[];
  onRequirementClick: (requirement: Requirement) => void;
  refreshTrigger: number;
  onRefresh: () => void;
}

interface ColumnState {
  requirements: Requirement[];
  total: number;
  loaded: number;
  loading: boolean;
}

type ColumnsMap = Record<string, ColumnState>;

const PAGE_SIZE = 10;

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
}) => {
  const [columns, setColumns] = useState<ColumnsMap>({});
  const [initialLoading, setInitialLoading] = useState(false);

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
    statusGroups.forEach((group) => {
      group.statuses.forEach((status) => {
        newColumns[status] = {
          requirements: [],
          total: statusTotals[status] ?? 0,
          loaded: 0,
          loading: false,
        };
      });
    });
    setColumns(newColumns);

    // 发起所有状态的首屏请求
    const promises: Promise<void>[] = [];
    statusGroups.forEach((group) => {
      group.statuses.forEach((status) => {
        promises.push(loadColumn(status, 0));
      });
    });

    await Promise.all(promises);
    setInitialLoading(false);
  }, [projectId, statusStats, loadColumn]);

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
      {statusGroups.map((group) => {
        // 计算这个大组包含的需求总数
        const groupTotal = group.statuses.reduce((acc, status) => acc + (columns[status]?.total ?? 0), 0);

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
              {group.statuses.map((status) => (
                <KanbanColumn
                  key={status}
                  groupKey={status}
                  label={getStatusLabel(status)}
                  groupColor={getStatusColor(status)}
                  totalCount={columns[status]?.total ?? 0}
                  requirements={columns[status]?.requirements ?? []}
                  loadedCount={columns[status]?.loaded ?? 0}
                  loading={columns[status]?.loading ?? false}
                  onLoadMore={() => handleLoadMore(status)}
                  onCardClick={onRequirementClick}
                />
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
};
