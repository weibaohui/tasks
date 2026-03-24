/**
 * 应用入口组件
 * 配置路由和全局设置
 */
import React from 'react';
import { BrowserRouter, Routes, Route, Navigate, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { ConfigProvider, Layout, Menu, Typography } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import {
  ApiOutlined,
  AppstoreOutlined,
  ApartmentOutlined,
  DashboardOutlined,
  DatabaseOutlined,
  MessageOutlined,
  RobotOutlined,
  ToolOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { Dashboard } from './pages/Dashboard';
import { TaskManagement } from './pages/TaskManagement';
import { TaskDetailPage } from './pages/TaskDetailPage';
import { TaskTreePage } from './pages/TaskTreePage';
import { LoginPage } from './pages/LoginPage';
import { UserManagementPage } from './pages/UserManagementPage';
import { ProviderManagementPage } from './pages/ProviderManagementPage';
import { AgentManagementPage } from './pages/AgentManagementPage';
import { ChannelManagementPage } from './pages/ChannelManagementPage';
import { SessionManagementPage } from './pages/SessionManagementPage';
import { ConversationRecordsPage } from './pages/ConversationRecordsPage';
import { MCPManagementPage } from './pages/MCPManagementPage';
import { useAuthStore } from './stores/authStore';

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
  const selectedKey = location.pathname.startsWith('/users')
    ? '/users'
    : location.pathname.startsWith('/providers')
      ? '/providers'
      : location.pathname.startsWith('/agents')
        ? '/agents'
        : location.pathname.startsWith('/mcp')
          ? '/mcp'
        : location.pathname.startsWith('/channels')
          ? '/channels'
          : location.pathname.startsWith('/sessions')
            ? '/sessions'
            : location.pathname.startsWith('/conversation-records')
              ? '/conversation-records'
              : location.pathname.startsWith('/dashboard')
                ? '/dashboard'
                : '/tasks';

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Layout.Sider width={220} theme="light" style={{ borderRight: '1px solid #f0f0f0' }}>
        <div style={{ height: 56, display: 'flex', alignItems: 'center', padding: '0 16px' }}>
          <Typography.Title level={5} style={{ margin: 0 }}>
            任务管理后台
          </Typography.Title>
        </div>
        <Menu
          mode="inline"
          selectedKeys={[selectedKey]}
          items={[
            { key: '/dashboard', icon: <DashboardOutlined />, label: 'Dashboard' },
            { key: '/tasks', icon: <AppstoreOutlined />, label: '任务管理' },
            { key: '/conversation-records', icon: <MessageOutlined />, label: '对话记录' },
            { key: '/agents', icon: <RobotOutlined />, label: 'Agents 管理' },
            { key: '/mcp', icon: <ToolOutlined />, label: 'MCP 管理' },
            { key: '/channels', icon: <ApartmentOutlined />, label: '渠道管理' },
            { key: '/sessions', icon: <DatabaseOutlined />, label: '会话管理' },
            { key: '/providers', icon: <ApiOutlined />, label: 'LLM 配置' },
            { key: '/users', icon: <UserOutlined />, label: '用户管理' },
          ]}
          onClick={(item) => navigate(item.key)}
        />
      </Layout.Sider>
      <Layout.Content style={{ background: '#f5f5f5' }}>
        <Outlet />
      </Layout.Content>
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
          <Route path="/login" element={token ? <Navigate to="/tasks" replace /> : <LoginPage />} />
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <MainLayout />
              </ProtectedRoute>
            }
          >
            <Route path="dashboard" element={<Dashboard />} />
            <Route path="tasks" element={<TaskManagement />} />
            <Route path="tasks/:taskId" element={<TaskDetailPage />} />
            <Route path="tasks/trace/:traceId/tree" element={<TaskTreePage />} />
            <Route path="conversation-records" element={<ConversationRecordsPage />} />
            <Route path="agents" element={<AgentManagementPage />} />
            <Route path="channels" element={<ChannelManagementPage />} />
            <Route path="sessions" element={<SessionManagementPage />} />
            <Route path="providers" element={<ProviderManagementPage />} />
            <Route path="mcp" element={<MCPManagementPage />} />
            <Route path="users" element={<UserManagementPage />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  );
};

export default App;
