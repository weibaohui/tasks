/**
 * Hook 配置管理 Hook
 */
import { useState, useCallback } from 'react';
import { message } from 'antd';
import type { HookConfig, CreateHookConfigRequest, UpdateHookConfigRequest, HookActionLog } from '../../../types/hook';
import * as hookApi from '../../../api/hookApi';

export interface UseHookManagementReturn {
  // 数据
  items: HookConfig[];
  loading: boolean;
  saving: boolean;
  logs: HookActionLog[];
  logsLoading: boolean;

  // 编辑状态
  open: boolean;
  editing: HookConfig | null;

  // 操作方法
  fetchList: () => Promise<void>;
  openEditor: (record: HookConfig | null) => void;
  closeEditor: () => void;
  handleDelete: (id: string) => Promise<void>;
  handleToggleEnabled: (record: HookConfig) => Promise<void>;
  handleSubmit: (values: CreateHookConfigRequest | UpdateHookConfigRequest) => Promise<void>;
  fetchLogs: (requirementId?: string, hookConfigId?: string) => Promise<void>;
  clearLogs: () => void;
}

export interface UseHookManagementOptions {
  // 可以添加自定义选项
}

export function useHookManagement(_options?: UseHookManagementOptions): UseHookManagementReturn {
  const [items, setItems] = useState<HookConfig[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<HookConfig | null>(null);
  const [logs, setLogs] = useState<HookActionLog[]>([]);
  const [logsLoading, setLogsLoading] = useState(false);

  const fetchList = useCallback(async () => {
    setLoading(true);
    try {
      const data = await hookApi.listHookConfigs();
      setItems(data);
    } catch (err) {
      message.error('获取 Hook 配置列表失败');
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, []);

  const openEditor = useCallback((record: HookConfig | null) => {
    setEditing(record);
    setOpen(true);
  }, []);

  const closeEditor = useCallback(() => {
    setOpen(false);
    setEditing(null);
  }, []);

  const handleDelete = useCallback(async (id: string) => {
    try {
      await hookApi.deleteHookConfig(id);
      message.success('删除成功');
      fetchList();
    } catch (err) {
      message.error('删除失败');
      console.error(err);
    }
  }, [fetchList]);

  const handleToggleEnabled = useCallback(async (record: HookConfig) => {
    try {
      if (record.enabled) {
        await hookApi.disableHookConfig(record.id);
        message.success('已禁用');
      } else {
        await hookApi.enableHookConfig(record.id);
        message.success('已启用');
      }
      fetchList();
    } catch (err) {
      message.error('操作失败');
      console.error(err);
    }
  }, [fetchList]);

  const handleSubmit = useCallback(async (values: CreateHookConfigRequest | UpdateHookConfigRequest) => {
    setSaving(true);
    try {
      if (editing) {
        await hookApi.updateHookConfig(values as UpdateHookConfigRequest);
        message.success('更新成功');
      } else {
        await hookApi.createHookConfig(values as CreateHookConfigRequest);
        message.success('创建成功');
      }
      closeEditor();
      fetchList();
    } catch (err) {
      message.error(editing ? '更新失败' : '创建失败');
      console.error(err);
    } finally {
      setSaving(false);
    }
  }, [editing, closeEditor, fetchList]);

  const fetchLogs = useCallback(async (requirementId?: string, hookConfigId?: string) => {
    setLogsLoading(true);
    try {
      const data = await hookApi.listHookLogs(requirementId, hookConfigId);
      setLogs(data);
    } catch (err) {
      message.error('获取日志失败');
      console.error(err);
    } finally {
      setLogsLoading(false);
    }
  }, []);

  const clearLogs = useCallback(() => {
    setLogs([]);
  }, []);

  return {
    items,
    loading,
    saving,
    logs,
    logsLoading,
    open,
    editing,
    fetchList,
    openEditor,
    closeEditor,
    handleDelete,
    handleToggleEnabled,
    handleSubmit,
    fetchLogs,
    clearLogs,
  };
}