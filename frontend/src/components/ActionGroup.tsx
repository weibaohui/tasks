import React, { ReactNode, useState } from 'react';
import { Popover, Button, Space, SpaceProps } from 'antd';
import { MoreOutlined } from '@ant-design/icons';

interface ActionGroupProps extends SpaceProps {
  children: ReactNode;
}

export const ActionGroup: React.FC<ActionGroupProps> = ({ children, ...restProps }) => {
  const [open, setOpen] = useState(false);
  const childrenArray = React.Children.toArray(children).filter(Boolean);

  if (childrenArray.length <= 1) {
    return <Space {...restProps}>{children}</Space>;
  }

  const firstChild = childrenArray[0];
  const restChildren = childrenArray.slice(1);

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen);
  };

  const handleContentClick = () => {
    // Optionally close the popover on child click
  };

  const content = (
    <Space 
      direction="vertical" 
      size="small" 
      style={{ display: 'flex', alignItems: 'flex-start' }}
      onClick={handleContentClick}
    >
      {restChildren}
    </Space>
  );

  return (
    <Space {...restProps}>
      {firstChild}
      <Popover 
        content={content} 
        trigger="click" 
        placement="bottomRight" 
        arrow={false}
        open={open}
        onOpenChange={handleOpenChange}
      >
        <Button type="link" size="small" style={{ padding: 0 }}>
          <MoreOutlined />
        </Button>
      </Popover>
    </Space>
  );
};
