import React, { useMemo } from 'react';
import { Tooltip } from 'antd';
import type { ConversationRecord } from '../../types/conversationRecord';
import { motion, AnimatePresence } from 'framer-motion';

export interface ConversationTimelineProps {
  records: ConversationRecord[];
  height?: number | string;
}

// 颜色映射：根据不同的 role 或 event_type 区分
function getBlockColor(record: ConversationRecord): string {
  const role = (record.role || '').toLowerCase();
  
  if (role === 'user') return '#1890ff'; // 蓝色 - 用户
  if (role === 'assistant') return '#52c41a'; // 绿色 - 助手
  if (role === 'system') return '#fa8c16'; // 橙色 - 系统
  if (role === 'tool') return '#eb2f96'; // 粉色 - 工具调用 (tool_call)
  if (role === 'tool_result') return '#722ed1'; // 紫色 - 工具结果
  
  return '#d9d9d9'; // 默认灰色
}

function getBlockLabel(record: ConversationRecord): string {
  const role = (record.role || '').toLowerCase();
  if (role === 'user') return '用户';
  if (role === 'assistant') return '助手';
  if (role === 'system') return '系统';
  if (role === 'tool') return '工具调用';
  if (role === 'tool_result') return '工具结果';
  return role || '未知';
}

export const ConversationTimeline: React.FC<ConversationTimelineProps> = ({ 
  records, 
  height = 30 
}) => {
  // 确保按时间戳排序
  const sortedRecords = useMemo(() => {
    return [...records].sort((a, b) => (a.timestamp || 0) - (b.timestamp || 0));
  }, [records]);

  if (sortedRecords.length === 0) {
    return (
      <div 
        style={{ 
          height, 
          width: '100%', 
          background: '#f0f0f0', 
          borderRadius: 4,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          color: '#bfbfbf',
          fontSize: 12
        }}
      >
        等待对话...
      </div>
    );
  }

  // 动态计算每个块的宽度（百分比）
  const itemWidthPercent = 100 / sortedRecords.length;

  return (
    <div 
      style={{ 
        width: '100%', 
        height, 
        display: 'flex', 
        borderRadius: 4, 
        overflow: 'hidden',
        background: '#f0f0f0',
        position: 'relative'
      }}
    >
      <AnimatePresence>
        {sortedRecords.map((record, index) => {
          const color = getBlockColor(record);
          const label = getBlockLabel(record);
          
          return (
            <Tooltip 
              key={record.id} 
              title={
                <div>
                  <div><strong>{label}</strong></div>
                  {record.event_type && <div>Event: {record.event_type}</div>}
                  {record.total_tokens > 0 && <div>Tokens: {record.total_tokens}</div>}
                  <div style={{ 
                    maxWidth: 300, 
                    overflow: 'hidden', 
                    textOverflow: 'ellipsis', 
                    whiteSpace: 'nowrap',
                    marginTop: 4,
                    opacity: 0.8,
                    fontSize: 12
                  }}>
                    {record.content || '(无内容)'}
                  </div>
                </div>
              }
            >
              <motion.div
                initial={{ width: 0, opacity: 0, x: 20 }}
                animate={{ 
                  width: `${itemWidthPercent}%`, 
                  opacity: 1, 
                  x: 0 
                }}
                transition={{ 
                  type: 'spring', 
                  stiffness: 300, 
                  damping: 30,
                  delay: index === sortedRecords.length - 1 ? 0.1 : 0 
                }}
                style={{
                  height: '100%',
                  background: color,
                  borderRight: index < sortedRecords.length - 1 ? '1px solid rgba(255,255,255,0.3)' : 'none',
                  cursor: 'pointer',
                  minWidth: 4
                }}
              />
            </Tooltip>
          );
        })}
      </AnimatePresence>
    </div>
  );
};
