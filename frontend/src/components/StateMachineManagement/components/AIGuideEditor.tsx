/**
 * StateMachine AI Guide Editor Component
 * AI Guide编辑Modal
 */
import React from 'react';
import { Modal, Form, Input, Button, Divider, Alert } from 'antd';
import { PlusOutlined } from '@ant-design/icons';

interface StateWithAIGuide {
  id: string;
  name: string;
  isFinal: boolean;
  aiGuide?: string;
  autoInit?: string;
  successCriteria?: string;
  failureCriteria?: string;
  triggers?: { trigger: string; description?: string; condition?: string }[];
}

interface AIGuideEditorProps {
  open: boolean;
  stateName: string;
  state: StateWithAIGuide;
  onSave: () => void;
  onCancel: () => void;
  onChange: (state: StateWithAIGuide) => void;
  onAddTrigger: () => void;
  onRemoveTrigger: (index: number) => void;
  onUpdateTrigger: (index: number, field: 'trigger' | 'description' | 'condition', value: string) => void;
}

export const AIGuideEditor: React.FC<AIGuideEditorProps> = ({
  open,
  stateName,
  state,
  onSave,
  onCancel,
  onChange,
  onAddTrigger,
  onRemoveTrigger,
  onUpdateTrigger,
}) => {
  return (
    <Modal
      title={`编辑 AI Guide: ${stateName}`}
      open={open}
      onOk={onSave}
      onCancel={onCancel}
      okText="保存"
      cancelText="取消"
      width={800}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16, maxHeight: '70vh', overflow: 'auto' }}>
        <Alert
          type="info"
          showIcon
          message="AI Guide 配置"
          description="配置当前状态的 AI 执行指南，包括任务说明、成功/失败判断标准、可用触发器等。"
        />

        <Form layout="vertical">
          <Form.Item label="AI 操作指南 (ai_guide)" extra="Markdown 格式，告诉 AI 当前阶段应该做什么">
            <Input.TextArea
              value={state.aiGuide}
              onChange={(e) => onChange({ ...state, aiGuide: e.target.value })}
              placeholder="## 当前阶段任务\n1. 分析需求\n2. 编写代码\n3. 运行测试"
              rows={8}
            />
          </Form.Item>

          <Form.Item label="自动初始化命令 (auto_init)" extra="进入此状态时自动执行的 Shell 命令（可选）">
            <Input.TextArea
              value={state.autoInit}
              onChange={(e) => onChange({ ...state, autoInit: e.target.value })}
              placeholder="#!/bin/bash\ngit clone {{git_repo_url}} ."
              rows={4}
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item label="成功判断标准 (success_criteria)" extra="AI 根据此标准判断任务是否成功完成">
            <Input.TextArea
              value={state.successCriteria}
              onChange={(e) => onChange({ ...state, successCriteria: e.target.value })}
              placeholder="测试全部通过，代码符合规范"
              rows={2}
            />
          </Form.Item>

          <Form.Item label="失败判断标准 (failure_criteria)" extra="AI 根据此标准判断任务是否失败">
            <Input.TextArea
              value={state.failureCriteria}
              onChange={(e) => onChange({ ...state, failureCriteria: e.target.value })}
              placeholder="无法实现需求或遇到技术障碍"
              rows={2}
            />
          </Form.Item>

          <Divider style={{ margin: '8px 0' }} />

          <Form.Item label="可用触发器">
            <div style={{ marginBottom: 8 }}>
              <Button size="small" onClick={onAddTrigger} icon={<PlusOutlined />}>
                添加触发器
              </Button>
            </div>
            {(state.triggers || []).map((t, index) => (
              <div key={index} style={{ display: 'flex', gap: 8, marginBottom: 8, alignItems: 'flex-start' }}>
                <Input
                  placeholder="触发器名称"
                  value={t.trigger}
                  onChange={(e) => onUpdateTrigger(index, 'trigger', e.target.value)}
                  style={{ width: 120 }}
                />
                <Input
                  placeholder="描述"
                  value={t.description}
                  onChange={(e) => onUpdateTrigger(index, 'description', e.target.value)}
                  style={{ flex: 1 }}
                />
                <Input
                  placeholder="触发条件"
                  value={t.condition}
                  onChange={(e) => onUpdateTrigger(index, 'condition', e.target.value)}
                  style={{ flex: 1 }}
                />
                <Button size="small" danger onClick={() => onRemoveTrigger(index)}>
                  删除
                </Button>
              </div>
            ))}
            {(state.triggers || []).length === 0 && (
              <div style={{ color: '#999', fontSize: 12 }}>暂无触发器配置</div>
            )}
          </Form.Item>
        </Form>
      </div>
    </Modal>
  );
};
