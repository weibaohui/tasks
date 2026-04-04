package state_machine

import "time"

// TypeBinding 需求类型与状态机绑定
type TypeBinding struct {
	ID             string    `json:"id"`
	StateMachineID string    `json:"state_machine_id"`
	RequirementType string   `json:"requirement_type"`
	CreatedAt      time.Time `json:"created_at"`
}

// NewTypeBinding 创建类型绑定
func NewTypeBinding(stateMachineID, requirementType string) *TypeBinding {
	return &TypeBinding{
		ID:             generateID(),
		StateMachineID: stateMachineID,
		RequirementType: requirementType,
		CreatedAt:      time.Now(),
	}
}
