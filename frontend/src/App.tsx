/**
 * 应用入口组件
 * 配置路由和全局设置
 */
import React from 'react';
import { BrowserRouter, Routes, Route, Navigate, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { ConfigProvider, Layout, Menu, Typography } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { AppstoreOutlined, UserOutlined } from '@ant-design/icons';
import { TaskDashboard } from './pages/TaskDashboard';
import { TaskDetailPage } from './pages/TaskDetailPage';
import { TaskTreePage } from './pages/TaskTreePage';
import { LoginPage } from './pages/LoginPage';
import { UserManagementPage } from './pages/UserManagementPage';
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
  const selectedKey = location.pathname.startsWith('/users') ? '/users' : '/tasks';

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
            { key: '/tasks', icon: <AppstoreOutlined />, label: '任务管理' },
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
          <Route path="/" element={<Navigate to={token ? '/tasks' : '/login'} replace />} />
          <Route path="/login" element={token ? <Navigate to="/tasks" replace /> : <LoginPage />} />
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <MainLayout />
              </ProtectedRoute>
            }
          >
            <Route path="tasks" element={<TaskDashboard />} />
            <Route path="tasks/:taskId" element={<TaskDetailPage />} />
            <Route path="tasks/trace/:traceId/tree" element={<TaskTreePage />} />
            <Route path="users" element={<UserManagementPage />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  );
};

export default App;
