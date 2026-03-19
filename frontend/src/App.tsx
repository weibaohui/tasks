/**
 * 应用入口组件
 * 配置路由和全局设置
 */
import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { TaskDashboard } from './pages/TaskDashboard';
import { TaskDetailPage } from './pages/TaskDetailPage';
import { TaskTreePage } from './pages/TaskTreePage';

const App: React.FC = () => {
  return (
    <ConfigProvider locale={zhCN}>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Navigate to="/tasks" replace />} />
          <Route path="/tasks" element={<TaskDashboard />} />
          <Route path="/tasks/:taskId" element={<TaskDetailPage />} />
          <Route path="/tasks/trace/:traceId/tree" element={<TaskTreePage />} />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  );
};

export default App;
