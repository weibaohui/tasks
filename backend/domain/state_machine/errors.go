package state_machine

import "fmt"

// 错误类型
type StateMachineError struct {
	Code    string
	Message string
}

func (e *StateMachineError) Error() string {
	return e.Message
}

// ErrInvalidConfig 配置错误
func ErrInvalidConfig(format string, args ...interface{}) error {
	return &StateMachineError{
		Code:    "INVALID_CONFIG",
		Message: fmt.Sprintf(format, args...),
	}
}

// ErrStateNotFound 状态不存在
func ErrStateNotFound(stateID string) error {
	return &StateMachineError{
		Code:    "STATE_NOT_FOUND",
		Message: fmt.Sprintf("state '%s' not found", stateID),
	}
}

// ErrTransitionNotFound 转换不存在
func ErrTransitionNotFound(fromState, trigger string) error {
	return &StateMachineError{
		Code:    "TRANSITION_NOT_FOUND",
		Message: fmt.Sprintf("transition from '%s' with trigger '%s' not found", fromState, trigger),
	}
}

// ErrFinalStateCannotTransition 终态不能转换
func ErrFinalStateCannotTransition(stateID string) error {
	return &StateMachineError{
		Code:    "FINAL_STATE_CANNOT_TRANSITION",
		Message: fmt.Sprintf("state '%s' is final and cannot transition", stateID),
	}
}

// ErrStateMachineNotFound 状态机不存在
func ErrStateMachineNotFound(id string) error {
	return &StateMachineError{
		Code:    "STATE_MACHINE_NOT_FOUND",
		Message: fmt.Sprintf("state machine '%s' not found", id),
	}
}

// ErrRequirementStateNotFound 需求状态不存在
func ErrRequirementStateNotFound(requirementID string) error {
	return &StateMachineError{
		Code:    "REQUIREMENT_STATE_NOT_FOUND",
		Message: fmt.Sprintf("requirement state for '%s' not found", requirementID),
	}
}
