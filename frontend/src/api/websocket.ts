/**
 * WebSocket 客户端模块
 * 处理与后端的 WebSocket 实时通信
 */
import type { WSMessage } from '../types/task';

type MessageHandler = (message: WSMessage) => void;

/**
 * WebSocket 客户端类
 * 提供连接、订阅、取消订阅、重连等功能
 */
class TaskWebSocketClient {
  private ws: WebSocket | null = null;
  private handlers: Map<string, Set<MessageHandler>> = new Map();
  private traceId: string = '';
  private reconnectAttempts: number = 0;
  private maxReconnectAttempts: number = 5;
  private reconnectDelay: number = 3000;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;

  /**
   * 连接到 WebSocket 服务器
   */
  connect(traceId: string): void {
    this.traceId = traceId;
    const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws?trace_id=${traceId}`;
    console.log('正在连接 WebSocket:', wsUrl);

    this.ws = new WebSocket(wsUrl);

    this.ws.onopen = () => {
      console.log('WebSocket 连接已建立');
      this.reconnectAttempts = 0;
    };

    this.ws.onmessage = (event) => {
      try {
        const message: WSMessage = JSON.parse(event.data);
        this.dispatchMessage(message);
      } catch (error) {
        console.error('解析 WebSocket 消息失败:', error);
      }
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket 错误:', error);
    };

    this.ws.onclose = () => {
      console.log('WebSocket 连接已关闭');
      this.attemptReconnect();
    };
  }

  /**
   * 订阅特定类型的消息
   * @param eventType 事件类型
   * @param handler 处理函数
   * @returns 取消订阅函数
   */
  subscribe(eventType: string, handler: MessageHandler): () => void {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, new Set());
    }
    this.handlers.get(eventType)!.add(handler);

    return () => {
      this.handlers.get(eventType)?.delete(handler);
    };
  }

  /**
   * 派发消息到对应的处理器
   */
  private dispatchMessage(message: WSMessage): void {
    const handlers = this.handlers.get(message.type);
    if (handlers) {
      handlers.forEach((handler) => handler(message));
    }
    const wildcardHandlers = this.handlers.get('*');
    if (wildcardHandlers) {
      wildcardHandlers.forEach((handler) => handler(message));
    }
  }

  /**
   * 断开连接
   */
  disconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.handlers.clear();
  }

  /**
   * 尝试重连
   */
  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('达到最大重连次数，停止重连');
      return;
    }

    this.reconnectAttempts++;
    console.log(`${this.reconnectDelay / 1000}秒后尝试第${this.reconnectAttempts}次重连...`);

    this.reconnectTimer = setTimeout(() => {
      if (this.traceId) {
        this.connect(this.traceId);
      }
    }, this.reconnectDelay);
  }
}

/**
 * 导出 WebSocket 单例
 */
export const wsClient = new TaskWebSocketClient();
