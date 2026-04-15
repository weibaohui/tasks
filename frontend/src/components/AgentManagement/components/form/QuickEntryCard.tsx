/**
 * QuickEntryCard - 快捷入口卡片
 */
import React from 'react';
import { Card, Button, Typography } from 'antd';
import { ArrowRightOutlined } from '@ant-design/icons';

const { Text } = Typography;

interface QuickEntryCardProps {
  title: string;
  description: string;
  onClick: () => void;
}

export const QuickEntryCard: React.FC<QuickEntryCardProps> = ({
  title,
  description,
  onClick,
}) => {
  return (
    <Card
      size="small"
      style={{ flex: 1, cursor: 'pointer' }}
      onClick={onClick}
    >
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div>
          <Text strong style={{ display: 'block' }}>{title}</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>{description}</Text>
        </div>
        <Button type="link" size="small" icon={<ArrowRightOutlined />}>
          去管理
        </Button>
      </div>
    </Card>
  );
};

export default QuickEntryCard;
