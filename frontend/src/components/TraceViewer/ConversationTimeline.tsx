import React, { useMemo } from 'react';
import { Tooltip } from 'antd';
import type { ConversationRecord } from '../../types/conversationRecord';
import { motion, AnimatePresence } from 'framer-motion';

export interface ConversationTimelineProps {
  records: ConversationRecord[];
  height?: number | string;
}

// 颜色映射：根据给定的蓝色系调色板进行映射，并让起点与终点有明显的跳脱和区别
// 用户输入（起点）使用较深的品牌蓝
// 助手回复（终点/产出）使用绿色，代表成功、闭环
function getBlockColor(record: ConversationRecord): string {
  const role = (record.role || '').toLowerCase();
  
  if (role === 'user') return '#1890ff'; // blue-6 (Brand Color) - 用户的输入，整个链路的主体起点，足够醒目
  if (role === 'assistant') return '#52c41a'; // green-6 (Success) - 助手回复，使用绿色，代表最终的回答成功闭环
  if (role === 'system') return '#e6f7ff'; // blue-1 (Selected background) - 系统设定，背景感最弱
  if (role === 'tool') return '#096dd9'; // blue-7 (Click) - 工具调用，正在执行/最重最深的逻辑层
  if (role === 'tool_result') return '#69c0ff'; // blue-4 - 工具返回，处于工具调用和主色之间
  
  return '#f0f0f0'; // 默认灰
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
        flexDirection: 'row',
        justifyContent: 'flex-start',
        borderRadius: 4, 
        overflow: 'hidden',
        background: '#f0f0f0',
        position: 'relative'
      }}
    >
      <AnimatePresence mode="popLayout">
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
                // 关键改动：不再使用 width 展开动画（"不是展开"）
                // 而是将元素初始位置放到屏幕最右侧（x: 1000），然后逐个飞入
                initial={{ x: 1000, opacity: 0 }}
                animate={{ 
                  x: 0, 
                  opacity: 1
                }}
                exit={{ opacity: 0, scale: 0 }}
                transition={{ 
                  type: 'spring', 
                  stiffness: 250, 
                  damping: 25,
                  delay: index * 0.1 // 逐个延迟：第1个、第2个...依次进入
                }}
                style={{
                  height: '100%',
                  // 宽度固定平分，不参与动画，保持色块形态完整
                  width: `${itemWidthPercent}%`,
                  background: color,
                  borderRight: index < sortedRecords.length - 1 ? '1px solid #ffffff' : 'none',
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
