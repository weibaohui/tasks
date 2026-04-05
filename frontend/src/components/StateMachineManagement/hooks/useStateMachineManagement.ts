/**
 * State Machine Management Hook
 */
import { useState, useCallback } from 'react';
import { message } from 'antd';
import type {
  StateMachine,
  CreateStateMachineRequest,
  UpdateStateMachineRequest,
} from '../../../types/stateMachine';
import * as stateMachineApi from '../../../api/stateMachineApi';

export interface UseStateMachineManagementReturn {
  // 数据
  items: StateMachine[];
  loading: boolean;
  saving: boolean;

  // 编辑状态
  open: boolean;
  editing: StateMachine | null;

  // 操作方法
  fetchList: () => Promise<void>;
  openEditor: (record: StateMachine | null) => void;
  closeEditor: () => void;
  handleDelete: (id: string) => Promise<void>;
  handleSubmit: (values: CreateStateMachineRequest | UpdateStateMachineRequest) => Promise<void>;
}

export function useStateMachineManagement(): UseStateMachineManagementReturn {
  const [items, setItems] = useState<StateMachine[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<StateMachine | null>(null);

  const fetchList = useCallback(async () => {
    setLoading(true);
    try {
      const data = await stateMachineApi.listStateMachines();
      setItems(data);
    } catch (err) {
      message.error('获取状态机列表失败');
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, []);

  const openEditor = useCallback((record: StateMachine | null) => {
    setEditing(record);
    setOpen(true);
  }, []);

  const closeEditor = useCallback(() => {
    setOpen(false);
    setEditing(null);
  }, []);

  const handleDelete = useCallback(async (id: string) => {
    try {
      await stateMachineApi.deleteStateMachine(id);
      message.success('删除成功');
      fetchList();
    } catch (err) {
      message.error('删除失败');
      console.error(err);
    }
  }, [fetchList]);

  const handleSubmit = useCallback(
    async (values: CreateStateMachineRequest | UpdateStateMachineRequest) => {
      setSaving(true);
      try {
        if (editing) {
          await stateMachineApi.updateStateMachine(values as UpdateStateMachineRequest);
          message.success('更新成功');
        } else {
          await stateMachineApi.createStateMachine(values as CreateStateMachineRequest);
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
    },
    [editing, closeEditor, fetchList],
  );

  return {
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
  };
}
