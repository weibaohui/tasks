/**
 * StateMachine Detail Drawer Component
 */
import React, { useState, useEffect, useCallback } from 'react';
import {
  Drawer,
  Tabs,
  Table,
  Tag,
  Space,
  Button,
  Select,
  message,
  Descriptions,
  Alert,
  Form,
  Input,
} from 'antd';
import type { StateMachine, TransitionLog, RequirementState } from '../../../types/stateMachine';
import * as stateMachineApi from '../../../api/stateMachineApi';

interface StateMachineDetailDrawerProps {
  stateMachine: StateMachine | null;
  open: boolean;
  onClose: () => void;
  requirements: { id: string; title: string; requirement_type: string }[];
  onRefreshRequirements: () => void;
}

export const StateMachineDetailDrawer: React.FC<StateMachineDetailDrawerProps> = ({
  stateMachine,
  open,
  onClose,
  requirements,
  onRefreshRequirements,
}) => {
  const [transitionHistory, setTransitionHistory] = useState<TransitionLog[]>([]);
  const [loadingHistory, setLoadingHistory] = useState(false);
  const [selectedRequirement, setSelectedRequirement] = useState<string>('');
  const [requirementState, setRequirementState] = useState<RequirementState | null>(null);
  const [triggering, setTriggering] = useState(false);
  const [triggerForm] = Form.useForm();

  const fetchRequirementState = useCallback(async (requirementId: string) => {
    try {
      const state = await stateMachineApi.getRequirementState(requirementId);
      setRequirementState(state);
    } catch (err) {
      setRequirementState(null);
    }
  }, []);

  const fetchTransitionHistory = useCallback(async (requirementId: string) => {
    setLoadingHistory(true);
    try {
      const logs = await stateMachineApi.getTransitionHistory(requirementId);
      setTransitionHistory(logs);
    } catch (err) {
      setTransitionHistory([]);
    } finally {
      setLoadingHistory(false);
    }
  }, []);

  useEffect(() => {
    if (selectedRequirement) {
      fetchRequirementState(selectedRequirement);
      fetchTransitionHistory(selectedRequirement);
    } else {
      setRequirementState(null);
      setTransitionHistory([]);
    }
  }, [selectedRequirement, fetchRequirementState, fetchTransitionHistory]);

  const handleTriggerTransition = async (values: { trigger: string; remark?: string }) => {
    if (!selectedRequirement) {
      message.warning('请先选择需求');
      return;
    }
    setTriggering(true);
    try {
      const newState = await stateMachineApi.triggerTransition(
        selectedRequirement,
        values.trigger,
        'ui',
        values.remark || '',
      );
      setRequirementState(newState);
      message.success('转换成功');
      triggerForm.resetFields();
      fetchTransitionHistory(selectedRequirement);
      onRefreshRequirements();
    } catch (err) {
      message.error('转换失败');
      console.error(err);
    } finally {
      setTriggering(false);
    }
  };

  if (!stateMachine) {
    return null;
  }

  const availableTransitions = stateMachineApi.getAvailableTransitions(
    stateMachine,
    requirementState?.current_state || stateMachine.config.initial_state,
  );

  const currentStateInfo = stateMachine.config.states.find(
    (s) => s.id === (requirementState?.current_state || stateMachine.config.initial_state),
  );

  const historyColumns = [
    { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 160,
      render: (ts: string) => new Date(ts).toLocaleString() },
    { title: '源状态', dataIndex: 'from_state', key: 'from_state', width: 100 },
    { title: '目标状态', dataIndex: 'to_state', key: 'to_state', width: 100 },
    { title: '触发器', dataIndex: 'trigger', key: 'trigger', width: 100 },
    { title: '触发者', dataIndex: 'triggered_by', key: 'triggered_by', width: 80 },
    { title: '结果', dataIndex: 'result', key: 'result', width: 80,
      render: (result: string) => (
        <Tag color={result === 'success' ? 'green' : 'red'}>{result}</Tag>
      )
    },
    { title: '备注', dataIndex: 'remark', key: 'remark', ellipsis: true },
    { title: '错误', dataIndex: 'error_message', key: 'error_message', ellipsis: true },
  ];

  return (
    <Drawer
      title={`状态机详情 - ${stateMachine.name}`}
      placement="right"
      width={900}
      onClose={onClose}
      open={open}
    >
      <Tabs
        items={[
          {
            key: 'overview',
            label: '概览',
            children: (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
                <Descriptions column={2} bordered size="small">
                  <Descriptions.Item label="名称">{stateMachine.name}</Descriptions.Item>
                  <Descriptions.Item label="描述">{stateMachine.description || '-'}</Descriptions.Item>
                  <Descriptions.Item label="初始状态">
                    {stateMachine.config.states.find((s) => s.id === stateMachine.config.initial_state)?.name || stateMachine.config.initial_state}
                  </Descriptions.Item>
                  <Descriptions.Item label="状态数">
                    {stateMachine.config.states.length}
                  </Descriptions.Item>
                  <Descriptions.Item label="转换数">
                    {stateMachine.config.transitions.length}
                  </Descriptions.Item>
                  <Descriptions.Item label="创建时间">
                    {new Date(stateMachine.created_at).toLocaleString()}
                  </Descriptions.Item>
                </Descriptions>

                <div>
                  <h4>状态列表</h4>
                  <Space size={[4, 8]} wrap>
                    {stateMachine.config.states.map((s) => (
                      <Tag key={s.id} color={s.is_final ? 'green' : 'blue'}>
                        {s.name} {s.is_final ? '(终态)' : ''}
                      </Tag>
                    ))}
                  </Space>
                </div>

                <div>
                  <h4>转换规则</h4>
                  <Table
                    dataSource={stateMachine.config.transitions}
                    rowKey={(_, index) => `t-${index}`}
                    pagination={false}
                    size="small"
                    columns={[
                      { title: '源状态', dataIndex: 'from', key: 'from', width: 100,
                        render: (from: string) => stateMachine.config.states.find((s) => s.id === from)?.name || from },
                      { title: '目标状态', dataIndex: 'to', key: 'to', width: 100,
                        render: (to: string) => stateMachine.config.states.find((s) => s.id === to)?.name || to },
                      { title: '触发器', dataIndex: 'trigger', key: 'trigger', width: 100 },
                      { title: '描述', dataIndex: 'description', key: 'description' },
                    ]}
                  />
                </div>
              </div>
            ),
          },
          {
            key: 'transition',
            label: '状态转换',
            children: (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
                <Alert
                  message="状态转换测试"
                  description="选择一个需求，查看其当前状态，并触发状态转换"
                  type="info"
                  showIcon
                />

                <Form layout="vertical">
                  <Form.Item label="选择需求" required>
                    <Select
                      placeholder="请选择需求"
                      value={selectedRequirement || undefined}
                      onChange={(value) => setSelectedRequirement(value)}
                      style={{ width: 400 }}
                      showSearch
                      optionFilterProp="children"
                    >
                      {requirements.map((req) => (
                        <Select.Option key={req.id} value={req.id}>
                          {req.title} ({req.requirement_type})
                        </Select.Option>
                      ))}
                    </Select>
                  </Form.Item>
                </Form>

                {selectedRequirement && (
                  <>
                    <Descriptions column={2} bordered size="small">
                      <Descriptions.Item label="当前状态">
                        <Tag color="blue">{currentStateInfo?.name || requirementState?.current_state || '-'}</Tag>
                      </Descriptions.Item>
                      <Descriptions.Item label="状态ID">
                        {requirementState?.current_state || '-'}
                      </Descriptions.Item>
                    </Descriptions>

                    {availableTransitions.length > 0 ? (
                      <div>
                        <h4>可用转换</h4>
                        <Form
                          form={triggerForm}
                          layout="inline"
                          onFinish={handleTriggerTransition}
                        >
                          <Form.Item
                            name="trigger"
                            rules={[{ required: true, message: '请选择触发器' }]}
                          >
                            <Select
                              placeholder="选择触发器"
                              style={{ width: 200 }}
                            >
                              {availableTransitions.map((t) => (
                                <Select.Option key={t.trigger} value={t.trigger}>
                                  {t.trigger} → {stateMachine.config.states.find((s) => s.id === t.to)?.name || t.to}
                                </Select.Option>
                              ))}
                            </Select>
                          </Form.Item>
                          <Form.Item name="remark">
                            <Input placeholder="备注（可选）" style={{ width: 200 }} />
                          </Form.Item>
                          <Form.Item>
                            <Button type="primary" htmlType="submit" loading={triggering}>
                              触发转换
                            </Button>
                          </Form.Item>
                        </Form>
                      </div>
                    ) : (
                      <Alert
                        message="无可用转换"
                        description="当前状态没有可用的转换规则，或已是终态"
                        type="warning"
                        showIcon
                      />
                    )}

                    <div>
                      <h4>转换历史</h4>
                      <Table
                        dataSource={transitionHistory}
                        columns={historyColumns}
                        rowKey="id"
                        loading={loadingHistory}
                        pagination={{ pageSize: 10 }}
                        size="small"
                        scroll={{ x: 1000 }}
                      />
                    </div>
                  </>
                )}
              </div>
            ),
          },
        ]}
      />
    </Drawer>
  );
};