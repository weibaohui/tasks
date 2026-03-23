import React from 'react';
import { useNavigate } from 'react-router-dom';
import { Card, Form, Input, Button, Typography, message } from 'antd';
import { useAuthStore } from '../stores/authStore';

export const LoginPage: React.FC = () => {
  const navigate = useNavigate();
  const { loginWithPassword, loading } = useAuthStore();

  const onFinish = async (values: { username: string; password: string }) => {
    const success = await loginWithPassword(values.username, values.password);
    if (!success) {
      message.error('登录失败，请检查用户名和密码');
      return;
    }
    message.success('登录成功');
    navigate('/tasks', { replace: true });
  };

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#f5f5f5',
      }}
    >
      <Card title="任务平台登录" style={{ width: 420 }}>
        <Form layout="vertical" onFinish={onFinish}>
          <Form.Item label="用户名" name="username" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input placeholder="请输入用户名" />
          </Form.Item>
          <Form.Item label="密码" name="password" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password placeholder="请输入密码" />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={loading} block>
            登录
          </Button>
        </Form>
        <Typography.Paragraph style={{ marginTop: 16, marginBottom: 0, color: '#999' }}>
          如需创建初始用户，请先调用用户创建接口。
        </Typography.Paragraph>
      </Card>
    </div>
  );
};
