import React, { useCallback, useEffect, useState } from 'react';
import { message, Modal } from 'antd';
import {
  DndContext,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
  DragOverlay,
  DragStartEvent,
} from '@dnd-kit/core';
import { listRequirementsPaginated, updateRequirementStatus, type StatusStat } from '../../api/projectRequirementApi';
import type { Requirement } from '../../types/projectRequirement';
import { statusGroups, getStatusGroup } from '../../constants/requirementStatus';
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

/** 每个分组的目标默认状态（拖入时使用） */
const groupDefaultStatus: Record<string, string> = {
  todo: 'todo',
  processing: 'preparing',
  complete: 'completed',
};

/** 根据 statusStats 计算每个分组的总数 */
function computeGroupTotals(statusStats: StatusStat[]): Record<string, number> {
  const totals: Record<string, number> = {};
  statusGroups.forEach((group) => {
    let count = 0;
    statusStats.forEach((stat) => {
      if (group.statuses.includes(stat.status)) {
        count += stat.count;
      }
    });
    totals[group.key] = count;
  });
  return totals;
}

export const RequirementKanban: React.FC<RequirementKanbanProps> = ({
  projectId,
  statusStats,
  onRequirementClick,
  refreshTrigger,
  onRefresh,
}) => {
  const [columns, setColumns] = useState<ColumnsMap>({});
  const [initialLoading, setInitialLoading] = useState(false);
  const [activeRequirement, setActiveRequirement] = useState<Requirement | null>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
  );

  const loadColumn = useCallback(async (groupKey: string, statuses: string[], offset: number) => {
    setColumns((prev) => ({
      ...prev,
      [groupKey]: { ...prev[groupKey], loading: true },
    }));

    try {
      const data = await listRequirementsPaginated({
        projectId,
        statuses,
        limit: PAGE_SIZE,
        offset,
      });
      setColumns((prev) => {
        const existing = prev[groupKey]?.requirements ?? [];
        return {
          ...prev,
          [groupKey]: {
            requirements: offset === 0 ? data.items : [...existing, ...data.items],
            total: data.total,
            loaded: offset === 0 ? data.items.length : (prev[groupKey]?.loaded ?? 0) + data.items.length,
            loading: false,
          },
        };
      });
    } catch {
      message.error('加载数据失败');
      setColumns((prev) => ({
        ...prev,
        [groupKey]: { ...prev[groupKey], loading: false },
      }));
    }
  }, [projectId]);

  const loadAllColumns = useCallback(async () => {
    if (!projectId) return;

    setInitialLoading(true);
    const groupTotals = computeGroupTotals(statusStats);
    const newColumns: ColumnsMap = {};
    statusGroups.forEach((group) => {
      newColumns[group.key] = {
        requirements: [],
        total: groupTotals[group.key] ?? 0,
        loaded: 0,
        loading: false,
      };
    });
    setColumns(newColumns);

    await Promise.all(
      statusGroups.map((group) => loadColumn(group.key, group.statuses, 0)),
    );
    setInitialLoading(false);
  }, [projectId, statusStats, loadColumn]);

  useEffect(() => {
    loadAllColumns();
  }, [loadAllColumns, refreshTrigger]);

  const handleLoadMore = (groupKey: string, statuses: string[]) => {
    const col = columns[groupKey];
    if (!col) return;
    loadColumn(groupKey, statuses, col.loaded);
  };

  /** 从所有列中查找需求 */
  const findRequirement = useCallback((reqId: string): Requirement | null => {
    for (const col of Object.values(columns)) {
      const found = col.requirements.find((r) => r.id === reqId);
      if (found) return found;
    }
    return null;
  }, [columns]);

  const handleDragStart = (event: DragStartEvent) => {
    const req = findRequirement(String(event.active.id));
    if (req) setActiveRequirement(req);
  };

  const handleDragEnd = async (event: DragEndEvent) => {
    setActiveRequirement(null);
    const { active, over } = event;
    if (!over) return;

    const reqId = String(active.id);
    const targetGroup = String(over.id);
    const req = findRequirement(reqId);
    if (!req || !targetGroup) return;

    // 检查是否在同一分组
    const currentGroup = getStatusGroup(req.status);
    if (currentGroup.key === targetGroup) return;

    const newStatus = groupDefaultStatus[targetGroup];
    if (!newStatus) return;

    // 确认弹窗
    const targetLabel = statusGroups.find((g) => g.key === targetGroup)?.label ?? targetGroup;
    Modal.confirm({
      title: '确认移动',
      content: `将需求「${req.title}」移至「${targetLabel}」？`,
      okText: '确认',
      cancelText: '取消',
      onOk: async () => {
        try {
          await updateRequirementStatus(req.id, newStatus);
          message.success(`已移至「${targetLabel}」`);
          // 刷新数据
          loadAllColumns();
          onRefresh();
        } catch {
          message.error('状态变更失败');
        }
      },
    });
  };

  return (
    <DndContext
      sensors={sensors}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
    >
      <div style={{
        display: 'flex',
        gap: 12,
        overflowX: 'auto',
        padding: '4px 0',
        minHeight: initialLoading ? 300 : undefined,
        opacity: initialLoading ? 0.6 : 1,
        transition: 'opacity 0.3s',
      }}>
        {statusGroups.map((group) => (
          <KanbanColumn
            key={group.key}
            groupKey={group.key}
            label={group.label}
            groupColor={group.color}
            totalCount={columns[group.key]?.total ?? 0}
            requirements={columns[group.key]?.requirements ?? []}
            loadedCount={columns[group.key]?.loaded ?? 0}
            loading={columns[group.key]?.loading ?? false}
            onLoadMore={() => handleLoadMore(group.key, group.statuses)}
            onCardClick={onRequirementClick}
          />
        ))}
      </div>
      <DragOverlay>
        {activeRequirement ? (
          <div style={{
            width: 320,
            padding: '8px 12px',
            backgroundColor: '#fff',
            borderRadius: 8,
            boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
            borderLeft: '3px solid #1677ff',
            opacity: 0.9,
          }}>
            <div style={{ fontSize: 13, fontWeight: 500 }}>{activeRequirement.title}</div>
          </div>
        ) : null}
      </DragOverlay>
    </DndContext>
  );
};
