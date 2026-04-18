import React from 'react';
import { Tabs } from 'antd';
import { useLocation, useNavigate } from 'react-router-dom';
import { TokenManagement } from '../components/TokenManagement/TokenManagement';
import { SystemLogViewer } from '../components/SystemLogViewer/SystemLogViewer';

/**
 * resolveTabFromPath 根据 URL 解析当前设置页 Tab。
 */
function resolveTabFromPath(pathname: string): 'api-tokens' | 'logs' {
  if (pathname.endsWith('/api-tokens')) {
    return 'api-tokens';
  }
  return 'logs';
}

/**
 * SettingsPage 系统设置页，按 URL 驱动 Tab，支持直达链接。
 */
export const SettingsPage: React.FC = () => {
  const location = useLocation();
  const navigate = useNavigate();
  const activeTab = resolveTabFromPath(location.pathname);

  return (
    <Tabs
      activeKey={activeTab}
      onChange={(key) => {
        const target = key === 'api-tokens' ? '/settings/api-tokens' : '/settings/logs';
        navigate(target);
      }}
      items={[
        {
          key: 'api-tokens',
          label: 'API 令牌',
          children: <TokenManagement />,
        },
        {
          key: 'logs',
          label: '系统日志',
          children: <SystemLogViewer />,
        },
      ]}
    />
  );
};

export default SettingsPage;
