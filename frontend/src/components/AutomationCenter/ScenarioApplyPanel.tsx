import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Alert, Button, Card, Empty, Modal, Select, Space, Tag, message } from 'antd';
import type { Project } from '../../types/projectRequirement';
import {
  applyHeartbeatScenario,
  listHeartbeatScenarios,
  previewApplyHeartbeatScenario,
  type HeartbeatScenario,
} from '../../api/heartbeatScenarioApi';

interface ScenarioApplyPanelProps {
  project: Project | null;
  onProjectScenarioChanged: (scenarioCode: string) => void;
}

/**
 * ScenarioApplyPanel 渲染项目场景应用面板，并提供影响预览确认。
 */
export const ScenarioApplyPanel: React.FC<ScenarioApplyPanelProps> = ({ project, onProjectScenarioChanged }) => {
  const [scenarios, setScenarios] = useState<HeartbeatScenario[]>([]);
  const [selectedScenarioCode, setSelectedScenarioCode] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [applying, setApplying] = useState(false);

  /**
   * fetchScenarios 加载可用场景列表。
   */
  const fetchScenarios = useCallback(async () => {
    setLoading(true);
    try {
      const data = await listHeartbeatScenarios();
      setScenarios(data);
    } catch {
      message.error('加载场景列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  /**
   * openPreviewAndApply 打开影响预览并在确认后执行应用。
   */
  const openPreviewAndApply = async () => {
    if (!project?.id || !selectedScenarioCode) {
      message.warning('请先选择项目和场景');
      return;
    }
    setApplying(true);
    try {
      const preview = await previewApplyHeartbeatScenario(project.id, selectedScenarioCode);
      setApplying(false);
      Modal.confirm({
        title: '确认应用心跳场景',
        width: 760,
        okText: '确认应用',
        cancelText: '取消',
        content: (
          <Space direction="vertical" style={{ width: '100%' }}>
            <Alert
              type="warning"
              showIcon
              message={`将应用场景：${preview.scenario_name}`}
              description={`预计删除 ${preview.delete_count} 条旧心跳，新增 ${preview.create_count} 条场景心跳。`}
            />
            <div>
              <div style={{ fontWeight: 600, marginBottom: 6 }}>将删除（最多展示5条）</div>
              {preview.to_delete.length === 0 ? (
                <div style={{ color: '#999' }}>无</div>
              ) : (
                <ul style={{ margin: 0, paddingLeft: 18 }}>
                  {preview.to_delete.slice(0, 5).map((item) => (
                    <li key={`delete-${item.id}`}>
                      {item.name}（{item.interval_minutes} 分钟）
                    </li>
                  ))}
                </ul>
              )}
            </div>
            <div>
              <div style={{ fontWeight: 600, marginBottom: 6 }}>将新增（最多展示5条）</div>
              {preview.to_create.length === 0 ? (
                <div style={{ color: '#999' }}>无</div>
              ) : (
                <ul style={{ margin: 0, paddingLeft: 18 }}>
                  {preview.to_create.slice(0, 5).map((item) => (
                    <li key={`create-${item.id}`}>
                      {item.name}（{item.interval_minutes} 分钟）
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </Space>
        ),
        onOk: async () => {
          setApplying(true);
          try {
            await applyHeartbeatScenario(project.id, selectedScenarioCode);
            message.success('场景应用成功');
            onProjectScenarioChanged(selectedScenarioCode);
          } catch {
            message.error('场景应用失败');
          } finally {
            setApplying(false);
          }
        },
      });
    } catch {
      message.error('预览场景影响失败');
      setApplying(false);
    }
  };

  const currentScenarioName = useMemo(() => {
    if (!project?.heartbeat_scenario_code) {
      return '';
    }
    const matched = scenarios.find((item) => item.code === project.heartbeat_scenario_code);
    return matched?.name || project.heartbeat_scenario_code;
  }, [project?.heartbeat_scenario_code, scenarios]);

  useEffect(() => {
    void fetchScenarios();
  }, [fetchScenarios]);

  if (!project) {
    return <Empty description="请先选择项目后再配置场景" />;
  }

  return (
    <Card
      size="small"
      title="项目场景应用"
      extra={
        <Button type="link" loading={loading} onClick={() => void fetchScenarios()}>
          刷新场景
        </Button>
      }
    >
      <Space direction="vertical" style={{ width: '100%' }}>
        <Space wrap>
          <Select
            style={{ minWidth: 300 }}
            placeholder="选择场景"
            value={selectedScenarioCode || undefined}
            options={scenarios.map((item) => ({ label: item.name, value: item.code }))}
            onChange={setSelectedScenarioCode}
            loading={loading}
            showSearch
            optionFilterProp="label"
          />
          <Button type="primary" onClick={() => void openPreviewAndApply()} loading={applying} disabled={!selectedScenarioCode}>
            预览并应用
          </Button>
        </Space>
        <div>
          <Tag color="blue">当前场景：{currentScenarioName || '未设置'}</Tag>
          <Tag color="purple">当前项目：{project.name}</Tag>
        </div>
      </Space>
    </Card>
  );
};
