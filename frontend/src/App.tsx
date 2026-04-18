/**
 * 应用入口组件
 * 配置路由和全局设置
 */
import React, { useState, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { ConfigProvider, Dropdown, Layout, Menu, Space, Typography, Drawer, Button } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import {
  ApiOutlined,
  ApartmentOutlined,
  BranchesOutlined,
  DashboardOutlined,
  DatabaseOutlined,
  DownOutlined,
  HeartOutlined,
  KeyOutlined,
  LogoutOutlined,
  MessageOutlined,
  FileTextOutlined,
  RobotOutlined,
  UserOutlined,
  NodeIndexOutlined,
  MenuOutlined,
} from '@ant-design/icons';
import { Dashboard } from './pages/Dashboard';
import { LoginPage } from './pages/LoginPage';
import { UserManagementPage } from './pages/UserManagementPage';
import { ProjectRequirementPage } from './pages/ProjectRequirementPage';
import { ProviderManagementPage } from './pages/ProviderManagementPage';
import AgentManagementPage from './pages/AgentManagementPage';
import { ChannelManagementPage } from './pages/ChannelManagementPage';
import { SessionManagementPage } from './pages/SessionManagementPage';
import { ConversationRecordsPage } from './pages/ConversationRecordsPage';
import { MCPManagementPage } from './pages/MCPManagementPage';
import { SkillsManagementPage } from './pages/SkillsManagementPage';
import StateMachineManagementPage from './pages/StateMachineManagementPage';
import { ApiTokensPage } from './pages/ApiTokensPage';
import { SystemLogsPage } from './pages/SystemLogsPage';
import { HeartbeatScenarioManagementPage } from './pages/HeartbeatScenarioManagementPage';
import { AutomationCenterPage } from './pages/AutomationCenterPage';
import { useAuthStore } from './stores/authStore';

const { Header, Sider, Content } = Layout;

// 响应式断点
const MOBILE_BREAKPOINT = 768;

const ProtectedRoute: React.FC<{ children: React.ReactElement }> = ({ children }) => {
  const { token } = useAuthStore();
  if (!token) {
    return <Navigate to="/login" replace />;
  }
  return children;
};

const MainLayout: React.FC = () => {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, logout } = useAuthStore();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [isMobileDevice, setIsMobileDevice] = useState(false);

  // 监听窗口大小变化
  useEffect(() => {
    const checkMobile = () => {
      const mobile = window.innerWidth < MOBILE_BREAKPOINT;
      setIsMobileDevice(mobile);
      if (mobile) {
        // 移动端不强制关闭，让用户自己关
      } else {
        // 切到桌面端时关闭抽屉
        setMobileOpen(false);
      }
    };
    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, []);

  const selectedKey = location.pathname.startsWith('/users')
    ? '/users'
    : location.pathname.startsWith('/providers')
      ? '/providers'
      : location.pathname.startsWith('/agents')
        ? '/agents'
        : location.pathname.startsWith('/mcp')
          ? '/mcp'
          : location.pathname.startsWith('/projects')
            ? '/projects'
            : location.pathname.startsWith('/skills')
            ? '/skills'
            : location.pathname.startsWith('/automation-center')
              ? '/automation-center'
              : location.pathname.startsWith('/heartbeat-scenarios')
                ? '/heartbeat-scenarios'
              : location.pathname.startsWith('/state-machines')
                ? '/state-machines'
                : location.pathname.startsWith('/channels')
                  ? '/channels'
                  : location.pathname.startsWith('/sessions')
                    ? '/sessions'
                    : location.pathname.startsWith('/conversation-records')
                      ? '/conversation-records'
                      : location.pathname.startsWith('/state-machines')
                        ? '/state-machines'
                        : location.pathname.startsWith('/dashboard')
                          ? '/dashboard'
                          : location.pathname.startsWith('/settings/api-tokens')
                            ? '/settings/api-tokens'
                            : location.pathname.startsWith('/settings/logs')
                              ? '/settings/logs'
                          : '/dashboard';

  const menuItems = [
    { key: '/dashboard', icon: <DashboardOutlined />, label: 'Dashboard' },
    { key: '/projects', icon: <BranchesOutlined />, label: '项目需求' },
    { key: '/conversation-records', icon: <MessageOutlined />, label: '对话记录' },
    { key: '/agents', icon: <RobotOutlined />, label: 'Agent 工坊' },
    { key: '/automation-center', icon: <HeartOutlined />, label: '自动化中心' },
    { key: '/state-machines', icon: <NodeIndexOutlined />, label: '状态机管理' },
    { key: '/channels', icon: <ApartmentOutlined />, label: '渠道管理' },
    { key: '/sessions', icon: <DatabaseOutlined />, label: '会话管理' },
    { key: '/providers', icon: <ApiOutlined />, label: 'LLM 配置' },
    { key: '/users', icon: <UserOutlined />, label: '用户管理' },
    { key: '/settings/api-tokens', icon: <KeyOutlined />, label: 'API令牌' },
    { key: '/settings/logs', icon: <FileTextOutlined />, label: '系统日志' },
  ];

  const handleMenuClick = (key: string) => {
    navigate(key);
    if (isMobileDevice) {
      setMobileOpen(false);
    }
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{
        background: '#fff',
        padding: isMobileDevice ? '0 12px' : '0 24px',
        borderBottom: '1px solid #f0f0f0',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        position: 'sticky',
        top: 0,
        zIndex: 100,
      }}>
        {isMobileDevice && (
          <Button
            type="text"
            icon={<MenuOutlined />}
            onClick={() => setMobileOpen(true)}
            style={{ fontSize: 18 }}
          />
        )}
        <Typography.Title level={5} style={{ margin: 0, flex: isMobileDevice ? 1 : 'none' }}>
          任务管理后台
        </Typography.Title>
        <Dropdown
          menu={{
            items: [
              { key: 'username', label: user?.display_name || user?.username, disabled: true },
              { type: 'divider' },
              { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', danger: true },
            ],
            onClick: ({ key }) => {
              if (key === 'logout') {
                logout();
                navigate('/login', { replace: true });
              }
            },
          }}
        >
          <Space style={{ cursor: 'pointer', color: '#666' }}>
            <UserOutlined />
            {!isMobileDevice && (user?.username || user?.user_code || '用户')}
            <DownOutlined style={{ fontSize: 10 }} />
          </Space>
        </Dropdown>
      </Header>
      <Layout>
        {/* 桌面端侧边栏 */}
        {!isMobileDevice && (
          <Sider width={220} theme="light" style={{ borderRight: '1px solid #f0f0f0', position: 'sticky', top: 64, height: 'calc(100vh - 64px)', overflow: 'auto' }}>
            <Menu
              mode="inline"
              selectedKeys={[selectedKey]}
              items={menuItems}
              onClick={(item) => handleMenuClick(item.key)}
            />
          </Sider>
        )}

        {/* 移动端抽屉导航 */}
        <Drawer
          title="导航菜单"
          placement="left"
          onClose={() => setMobileOpen(false)}
          open={mobileOpen}
          width={280}
          styles={{ body: { padding: 0 } }}
        >
          <Menu
            mode="inline"
            selectedKeys={[selectedKey]}
            items={menuItems}
            onClick={(item) => handleMenuClick(item.key)}
          />
        </Drawer>

        <Content style={{ background: '#f5f5f5', padding: isMobileDevice ? 12 : 24, minHeight: 'calc(100vh - 64px)' }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
};

const App: React.FC = () => {
  const { token, loadCurrentUser } = useAuthStore();

  React.useEffect(() => {
    if (token) {
      loadCurrentUser();
    }
  }, [token, loadCurrentUser]);

  return (
    <ConfigProvider locale={zhCN}>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Navigate to={token ? '/dashboard' : '/login'} replace />} />
          <Route path="/login" element={token ? <Navigate to="/dashboard" replace /> : <LoginPage />} />
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <MainLayout />
              </ProtectedRoute>
            }
          >
            <Route path="dashboard" element={<Dashboard />} />
            <Route path="projects" element={<ProjectRequirementPage />} />
            <Route path="conversation-records" element={<ConversationRecordsPage />} />
            <Route path="agents" element={<AgentManagementPage />} />
            <Route path="skills" element={<SkillsManagementPage />} />
            <Route path="automation-center" element={<AutomationCenterPage />} />
            <Route path="heartbeat-scenarios" element={<HeartbeatScenarioManagementPage />} />
            <Route path="state-machines" element={<StateMachineManagementPage />} />
            <Route path="channels" element={<ChannelManagementPage />} />
            <Route path="sessions" element={<SessionManagementPage />} />
            <Route path="providers" element={<ProviderManagementPage />} />
            <Route path="mcp" element={<MCPManagementPage />} />
            <Route path="users" element={<UserManagementPage />} />
            <Route path="settings" element={<Navigate to="/settings/logs" replace />} />
            <Route path="settings/api-tokens" element={<ApiTokensPage />} />
            <Route path="settings/logs" element={<SystemLogsPage />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  );
};

export default App;
