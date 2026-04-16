/**
 * State Machine Types
 */

// StateTriggerGuide 状态触发器指南
export interface StateTriggerGuide {
  trigger: string;
  description?: string;
  condition?: string;
}

// State 状态节点
export interface State {
  id: string;
  name: string;
  is_final: boolean;
  // AI 指南相关字段（可选）
  ai_guide?: string;
  auto_init?: string;
  success_criteria?: string;
  failure_criteria?: string;
  triggers?: StateTriggerGuide[];
}

// TransitionHook 转换钩子
export interface TransitionHook {
  name: string;
  type: string;
  config: Record<string, unknown>;
  retry?: number;
  timeout?: number;
}

// Transition 转换规则
export interface Transition {
  id?: string;
  from: string;
  to: string;
  trigger: string;
  description?: string;
  hooks?: TransitionHook[];
}

// Config YAML 配置
export interface StateMachineConfig {
  name: string;
  description?: string;
  initial_state: string;
  states: State[];
  transitions: Transition[];
}

// StateMachine 状态机
export interface StateMachine {
  id: string;
  name: string;
  description: string;
  config: StateMachineConfig;
  created_at: string;
  updated_at: string;
}

// RequirementState 需求状态
export interface RequirementState {
  id: string;
  requirement_id: string;
  state_machine_id: string;
  current_state: string;
  current_state_name: string;
  created_at: string;
  updated_at: string;
}

// TransitionLog 转换日志
export interface TransitionLog {
  id: string;
  requirement_id: string;
  from_state: string;
  to_state: string;
  trigger: string;
  triggered_by: string;
  remark?: string;
  result: 'success' | 'failed';
  error_message?: string;
  created_at: string;
}

// CreateStateMachineRequest 创建状态机请求
export interface CreateStateMachineRequest {
  name: string;
  description: string;
  config: string; // YAML content
}

// UpdateStateMachineRequest 更新状态机请求
export interface UpdateStateMachineRequest extends CreateStateMachineRequest {
  id: string;
}

// TriggerTransitionRequest 触发转换请求
export interface TriggerTransitionRequest {
  trigger: string;
  triggered_by?: string;
  remark?: string;
}

// StateSummary 状态统计
export interface StateSummary {
  [state: string]: number;
}