package state_machine

import "time"

// RequirementState 需求状态记录
type RequirementState struct {
	ID             string    `json:"id"`
	RequirementID  string    `json:"requirement_id"`
	StateMachineID string    `json:"state_machine_id"`
	CurrentState   string    `json:"current_state"`   // 状态ID
	CurrentStateName string  `json:"current_state_name"` // 状态名称
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// NewRequirementState 创建需求状态
func NewRequirementState(requirementID, stateMachineID, currentState, currentStateName string) *RequirementState {
	now := time.Now()
	return &RequirementState{
		ID:             generateID(),
		RequirementID:  requirementID,
		StateMachineID: stateMachineID,
		CurrentState:   currentState,
		CurrentStateName: currentStateName,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// Transition 转换状态
func (s *RequirementState) Transition(toState, toStateName string) {
	s.CurrentState = toState
	s.CurrentStateName = toStateName
	s.UpdatedAt = time.Now()
}

func generateID() string {
	// 简单的 ID 生成，实际使用 uuid
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
