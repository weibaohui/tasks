/**
 * State Machine Management Page
 */
import React, { useEffect } from 'react';
import { Card, Typography, Space, Button } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useStateMachineManagement } from './hooks';
import { StateMachineTable } from './components/StateMachineTable';
import { StateMachineEditDrawer } from './components/StateMachineEditDrawer';
import { StateMachineInvokeDrawer } from './components/StateMachineInvokeDrawer';
import type { CreateStateMachineRequest, StateMachine } from '../../types/stateMachine';

const { Title } = Typography;

export const StateMachineManagementPage: React.FC = () => {
  const {
    items,
    loading,
    saving,
    open,
    editing,
    fetchList,
    openEditor,
    closeEditor,
    handleDelete,
    handleSubmit,
  } = useStateMachineManagement();

  // 调用抽屉状态
  const [invokeOpen, setInvokeOpen] = React.useState(false);
  const [invokingStateMachine, setInvokingStateMachine] = React.useState<StateMachine | null>(null);

  const handleOpenInvoke = (record: StateMachine) => {
    setInvokingStateMachine(record);
    setInvokeOpen(true);
  };

  const handleCloseInvoke = () => {
    setInvokeOpen(false);
    setInvokingStateMachine(null);
  };

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  return (
    <div style={{ padding: 0 }}>
      <Card
        title={<Title level={3} style={{ margin: 0 }}>状态机管理</Title>}
        extra={
          <Space>
            <Button onClick={() => fetchList()}>刷新</Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => openEditor(null)}
            >
              新建状态机
            </Button>
          </Space>
        }
      >
        <StateMachineTable
          items={items}
          loading={loading}
          onEdit={openEditor}
          onDelete={handleDelete}
          onInvoke={handleOpenInvoke}
        />
      </Card>

      {/* 编辑抽屉 */}
      <StateMachineEditDrawer
        open={open}
        editing={editing}
        saving={saving}
        onClose={closeEditor}
        onSubmit={handleSubmit as (values: CreateStateMachineRequest) => Promise<void>}
      />

      {/* 调用抽屉 */}
      <StateMachineInvokeDrawer
        open={invokeOpen}
        stateMachine={invokingStateMachine}
        onClose={handleCloseInvoke}
      />
    </div>
  );
};

export default StateMachineManagementPage;
