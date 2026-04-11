package statemachine

import "time"

// TransitionLog 状态转换审计日志
type TransitionLog struct {
	ID            string    `json:"id"`
	RequirementID string    `json:"requirement_id"`
	FromState     string    `json:"from_state"`
	ToState       string    `json:"to_state"`
	Trigger       string    `json:"trigger"`
	TriggeredBy   string    `json:"triggered_by"`
	Remark        string    `json:"remark,omitempty"`
	Result        string    `json:"result"`           // success, failed
	ErrorMessage  string    `json:"error_message,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// NewTransitionLog 创建转换日志
func NewTransitionLog(requirementID, fromState, toState, trigger, triggeredBy, remark string) *TransitionLog {
	return &TransitionLog{
		ID:            generateID(),
		RequirementID: requirementID,
		FromState:     fromState,
		ToState:       toState,
		Trigger:       trigger,
		TriggeredBy:   triggeredBy,
		Remark:        remark,
		Result:        "success",
		CreatedAt:     time.Now(),
	}
}

// MarkFailed 标记为失败
func (l *TransitionLog) MarkFailed(errMsg string) {
	l.Result = "failed"
	l.ErrorMessage = errMsg
}
