/**
 * State Machine API 调用模块
 */
import apiClient from './client';
import type {
  StateMachine,
  CreateStateMachineRequest,
  UpdateStateMachineRequest,
  RequirementState,
  TransitionLog,
  Transition,
  StateSummary,
} from '../types/stateMachine';

/**
 * 获取状态机列表
 */
export async function listStateMachines(): Promise<StateMachine[]> {
  const response = await apiClient.get<StateMachine[]>(`/state-machines`);
  return response.data;
}

/**
 * 获取单个状态机
 */
export async function getStateMachine(id: string): Promise<StateMachine> {
  const response = await apiClient.get<StateMachine>(`/state-machines/${id}`);
  return response.data;
}

/**
 * 创建状态机
 */
export async function createStateMachine(
  request: CreateStateMachineRequest,
): Promise<StateMachine> {
  const response = await apiClient.post<StateMachine>(`/state-machines`, request);
  return response.data;
}

/**
 * 更新状态机
 */
export async function updateStateMachine(request: UpdateStateMachineRequest): Promise<StateMachine> {
  const response = await apiClient.put<StateMachine>(`/state-machines/${request.id}`, request);
  return response.data;
}

/**
 * 删除状态机
 */
export async function deleteStateMachine(id: string): Promise<void> {
  await apiClient.delete(`/state-machines/${id}`);
}

/**
 * 触发状态转换
 */
export async function triggerTransition(
  requirementId: string,
  trigger: string,
  triggeredBy?: string,
  remark?: string,
  metadata?: Record<string, unknown>,
): Promise<RequirementState> {
  const response = await apiClient.post<RequirementState>(
    `/requirements/${requirementId}/transitions`,
    {
      trigger,
      triggered_by: triggeredBy || 'api',
      remark: remark || '',
      metadata: metadata || {},
    },
  );
  return response.data;
}

/**
 * 获取需求当前状态
 */
export async function getRequirementState(requirementId: string): Promise<RequirementState> {
  const response = await apiClient.get<RequirementState>(
    `/requirements/${requirementId}/state`,
  );
  return response.data;
}

/**
 * 获取转换历史
 */
export async function getTransitionHistory(requirementId: string): Promise<TransitionLog[]> {
  const response = await apiClient.get<TransitionLog[]>(
    `/requirements/${requirementId}/transitions/history`,
  );
  return response.data;
}

/**
 * 获取状态统计
 */
export async function getStateSummary(): Promise<StateSummary> {
  const response = await apiClient.get<StateSummary>(`/requirements/states/summary`);
  return response.data;
}

/**
 * 获取状态机的可用转换
 */
export function getAvailableTransitions(
  stateMachine: StateMachine,
  currentState: string,
): Transition[] {
  return stateMachine.config.transitions.filter((t) => t.from === currentState);
}
