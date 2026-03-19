/**
 * WebSocket Hook
 * 管理 WebSocket 连接和消息订阅
 */
import { useEffect } from 'react';
import { wsClient } from '../api/websocket';
import { useTaskStore } from '../stores/taskStore';
import type { WSMessage, Task } from '../types/task';

export function useTaskWebSocket(traceId: string): void {
  const updateTaskInList = useTaskStore((state) => state.updateTaskInList);

  useEffect(() => {
    if (!traceId) return;

    wsClient.connect(traceId);

    const unsubscribeStatus = wsClient.subscribe('TaskStatusChanged', (message: WSMessage) => {
      const updatedTask = message.data as Task;
      updateTaskInList(updatedTask);
    });

    const unsubscribeProgress = wsClient.subscribe('TaskProgressUpdated', (message: WSMessage) => {
      const progressData = message.data as { task_id: string; progress: Task['progress'] };
      const currentTasks = useTaskStore.getState().tasks;
      const task = currentTasks.find((t) => t.id === progressData.task_id);
      if (task) {
        updateTaskInList({ ...task, progress: progressData.progress });
      }
    });

    return () => {
      unsubscribeStatus();
      unsubscribeProgress();
      wsClient.disconnect();
    };
  }, [traceId, updateTaskInList]);
}
