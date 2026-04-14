import React from 'react';
import { Button, Card, Spin, Tag, Typography, Empty, Tooltip } from 'antd';
import {
  ClockCircleOutlined,
  LoadingOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  RobotOutlined,
} from '@ant-design/icons';
import type { Requirement } from '../../types/projectRequirement';

const { Text, Paragraph } = Typography;

/** WIP 限制阈值：处理中列超过此数量显示警告 */
const WIP_LIMIT = 10;

interface KanbanColumnProps {
  groupKey: string;
  label: string;
  groupColor: { color: string; bgColor: string; borderColor: string };
  totalCount: number;
  groupTotal?: number;
  requirements: Requirement[];
  loadedCount: number;
  loading: boolean;
  onLoadMore: () => void;
  onCardClick: (requirement: Requirement) => void;
}

function formatTime(unixMilli: number | null): string {
  if (!unixMilli) return '';
  const d = new Date(unixMilli);
  return `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}

function getTypeConfig(type?: string): { label: string; color: string } {
  if (!type || type === 'normal') return { label: '普通', color: 'default' };
  if (type === 'heartbeat') return { label: '心跳', color: 'orange' };
  return { label: type, color: 'purple' };
}

function getRuntimeIcon(runtime?: Requirement['agent_runtime']): React.ReactNode {
  if (!runtime?.status) return null;
  switch (runtime.status) {
    case 'running':
      return <Tooltip title="运行中"><LoadingOutlined style={{ color: '#1677ff', fontSize: 12 }} spin /></Tooltip>;
    case 'completed':
      return <Tooltip title="运行完成"><CheckCircleOutlined style={{ color: '#52c41a', fontSize: 12 }} /></Tooltip>;
    case 'failed':
      return <Tooltip title="运行失败"><CloseCircleOutlined style={{ color: '#ff4d4f', fontSize: 12 }} /></Tooltip>;
    default:
      return null;
  }
}

export const KanbanColumn: React.FC<KanbanColumnProps> = ({
  groupKey,
  label,
  groupColor,
  totalCount,
  groupTotal,
  requirements,
  loadedCount,
  loading,
  onLoadMore,
  onCardClick,
}) => {
  const hasMore = loadedCount < totalCount;
  // 如果提供了 groupTotal，则基于大组总量判断 WIP 是否超限，否则基于单列总量
  const effectiveTotal = groupTotal !== undefined ? groupTotal : totalCount;
  const wipExceeded = groupKey !== 'todo' && groupKey !== 'completed' && groupKey !== 'done' && effectiveTotal > WIP_LIMIT;

  return (
    <div
      style={{
        width: 340,
        minWidth: 340,
        display: 'flex',
        flexDirection: 'column',
        backgroundColor: groupColor.bgColor,
        borderRadius: 8,
        border: `1px solid ${wipExceeded ? '#ff4d4f' : groupColor.borderColor}`,
        transition: 'background-color 0.2s, border-color 0.2s',
      }}
    >
      {/* 列头 */}
      <div style={{
        padding: '10px 12px',
        borderBottom: `2px solid ${wipExceeded ? '#ff4d4f' : groupColor.borderColor}`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        transition: 'border-color 0.2s',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <span style={{
            width: 8,
            height: 8,
            borderRadius: '50%',
            backgroundColor: wipExceeded ? '#ff4d4f' : groupColor.color,
            display: 'inline-block',
          }} />
          <Text strong style={{ color: wipExceeded ? '#ff4d4f' : groupColor.color, fontSize: 14 }}>{label}</Text>
        </div>
        <Tooltip title={wipExceeded ? `WIP 超限！当前 ${totalCount}，建议 ≤ ${WIP_LIMIT}` : undefined}>
          <Tag
            color={wipExceeded ? 'error' : groupColor.color}
            style={{ margin: 0, fontWeight: wipExceeded ? 700 : 400 }}
          >
            {totalCount}
          </Tag>
        </Tooltip>
      </div>

      {/* 卡片列表 */}
      <div style={{
        flex: 1,
        overflowY: 'auto',
        padding: '8px',
        display: 'flex',
        flexDirection: 'column',
        gap: '8px',
        minHeight: 200,
        maxHeight: 'calc(100vh - 320px)',
      }}>
        {loading && requirements.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 24 }}>
            <Spin />
          </div>
        ) : requirements.length === 0 ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无需求" />
        ) : (
          requirements.map((req) => (
            <RequirementCard
              key={req.id}
              requirement={req}
              groupKey={groupKey}
              groupColor={groupColor}
              onClick={() => onCardClick(req)}
            />
          ))
        )}
      </div>

      {/* 加载更多 */}
      {hasMore && (
        <div style={{ padding: '8px' }}>
          <Button
            block
            loading={loading}
            onClick={onLoadMore}
            style={{
              borderRadius: 6,
              borderStyle: 'dashed',
              color: groupColor.color,
            }}
          >
            加载更多 ({loadedCount}/{totalCount})
          </Button>
        </div>
      )}
    </div>
  );
};

/** 单个需求卡片 */
const RequirementCard: React.FC<{
  requirement: Requirement;
  groupKey: string;
  groupColor: { color: string; bgColor: string; borderColor: string };
  onClick: () => void;
}> = ({ requirement: req, groupColor, onClick }) => {
  const typeConfig = getTypeConfig(req.requirement_type);
  const runtimeIcon = getRuntimeIcon(req.agent_runtime);

  return (
    <Card
      size="small"
      hoverable
      style={{
        cursor: 'pointer',
        borderLeft: `3px solid ${groupColor.borderColor}`,
      }}
      bodyStyle={{ padding: '10px 12px' }}
      onClick={onClick}
    >
      <Tooltip title={req.description || req.title} placement="topLeft">
        <Paragraph
          ellipsis={{ rows: 2 }}
          style={{ margin: 0, fontSize: 13, fontWeight: 500, lineHeight: 1.4 }}
        >
          {req.title}
        </Paragraph>
      </Tooltip>

      <div style={{
        marginTop: 6,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        flexWrap: 'wrap',
        gap: 4,
      }}>
        <div style={{ display: 'flex', gap: 4, alignItems: 'center', flexWrap: 'wrap' }}>
          {/* 需求状态由列标题表示 */}
          <Tag color={typeConfig.color} style={{ fontSize: 11, lineHeight: '18px', padding: '0 4px' }}>
            {typeConfig.label}
          </Tag>
          {/* Agent */}
          {req.assignee_agent_code && (
            <Tooltip title={`Agent: ${req.assignee_agent_code}`}>
              <span style={{ display: 'inline-flex', alignItems: 'center' }}>
                <RobotOutlined style={{ fontSize: 11, color: '#8c8c8c' }} />
              </span>
            </Tooltip>
          )}
          {/* 运行状态 */}
          {runtimeIcon}
        </div>
        <Text type="secondary" style={{ fontSize: 11, display: 'inline-flex', alignItems: 'center', gap: 2 }}>
          <ClockCircleOutlined style={{ fontSize: 10 }} />
          {formatTime(req.created_at)}
        </Text>
      </div>
    </Card>
  );
};
