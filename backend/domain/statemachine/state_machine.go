package statemachine

import (
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// StateTriggerGuide 状态内触发器指南
type StateTriggerGuide struct {
	Trigger     string `json:"trigger" yaml:"trigger"`
	Description string `json:"description" yaml:"description"`
	Condition   string `json:"condition" yaml:"condition"`
}

// State 状态节点
type State struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	IsFinal  bool   `json:"is_final" yaml:"is_final"`

	// === AI 指南相关字段（可选）===

	// AIGuide AI操作指南（Markdown格式）
	// 说明当前阶段应该做什么、执行步骤、注意事项
	AIGuide string `json:"ai_guide,omitempty" yaml:"ai_guide,omitempty"`

	// AutoInit 自动初始化命令（可选）
	// 进入此状态时自动执行的 shell 命令
	AutoInit string `json:"auto_init,omitempty" yaml:"auto_init,omitempty"`

	// SuccessCriteria 成功判断标准
	// AI 根据此标准判断任务是否成功完成
	SuccessCriteria string `json:"success_criteria,omitempty" yaml:"success_criteria,omitempty"`

	// FailureCriteria 失败判断标准
	// AI 根据此标准判断任务是否失败
	FailureCriteria string `json:"failure_criteria,omitempty" yaml:"failure_criteria,omitempty"`

	// Triggers 可用的触发器说明
	// 告诉 AI 在什么条件下应该触发哪个转换
	Triggers []StateTriggerGuide `json:"triggers,omitempty" yaml:"triggers,omitempty"`
}

// Transition 转换规则
type Transition struct {
	FromState     string            `json:"from" yaml:"from"`
	ToState       string            `json:"to" yaml:"to"`
	Trigger       string            `json:"trigger" yaml:"trigger"`
	Description   string            `json:"description,omitempty" yaml:"description,omitempty"`
	Hooks         []TransitionHook `json:"hooks,omitempty" yaml:"hooks,omitempty"`
}

// TransitionHook 转换钩子
type TransitionHook struct {
	Name     string                 `json:"name" yaml:"name"`
	Type     string                 `json:"type" yaml:"type"`
	Config   map[string]interface{} `json:"config" yaml:"config"`
	Retry    int                    `json:"retry" yaml:"retry"`
	Timeout  int                    `json:"timeout" yaml:"timeout"`
}

// Config YAML 配置
type Config struct {
	Name          string      `json:"name" yaml:"name"`
	Description   string      `json:"description,omitempty" yaml:"description,omitempty"`
	InitialState  string      `json:"initial_state" yaml:"initial_state"`
	States       []State     `json:"states" yaml:"states"`
	Transitions  []Transition `json:"transitions" yaml:"transitions"`
}

// StateMachine 状态机聚合根
type StateMachine struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Config      *Config   `json:"config"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ParseConfig 解析 YAML 配置
func ParseConfig(yamlContent string) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal([]byte(yamlContent), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate 校验状态机配置
func (c *Config) Validate() error {
	if c.InitialState == "" {
		return ErrInvalidConfig("initial_state is required")
	}

	// 检查初始状态是否存在
	stateExists := false
	for _, s := range c.States {
		if s.ID == c.InitialState {
			stateExists = true
			break
		}
	}
	if !stateExists {
		return ErrInvalidConfig("initial_state '%s' not found in states", c.InitialState)
	}

	// 检查转换引用的状态是否存在
	stateMap := make(map[string]bool)
	for _, s := range c.States {
		stateMap[s.ID] = true
	}

	for _, t := range c.Transitions {
		if !stateMap[t.FromState] {
			return ErrInvalidConfig("transition from '%s': state not found", t.FromState)
		}
		if !stateMap[t.ToState] {
			return ErrInvalidConfig("transition to '%s': state not found", t.ToState)
		}
	}

	// 检查终态是否可以作为转换源
	for _, t := range c.Transitions {
		for _, s := range c.States {
			if s.ID == t.FromState && s.IsFinal {
				return ErrInvalidConfig("state '%s' is final and cannot be used as transition source", s.ID)
			}
		}
	}

	// 检查必须包含两态：todo, completed
	requiredStates := []string{"todo", "completed"}
	for _, required := range requiredStates {
		found := false
		for _, s := range c.States {
			if s.ID == required {
				found = true
				break
			}
		}
		if !found {
			return ErrInvalidConfig("state machine must contain mandatory state '%s'", required)
		}
	}

	// 检查 todo 至少有一条出向转换
	if len(c.GetAvailableTriggers("todo")) == 0 {
		return ErrInvalidConfig("mandatory state 'todo' must have at least one outgoing transition")
	}

	// 检查 completed 是否标记为终态
	completedState := c.GetState("completed")
	if completedState != nil && !completedState.IsFinal {
		return ErrInvalidConfig("mandatory state 'completed' should be marked as final (is_final: true)")
	}

	return nil
}

// FindTransition 查找转换规则
func (c *Config) FindTransition(currentStateID, trigger string) *Transition {
	for i := range c.Transitions {
		t := &c.Transitions[i]
		if t.FromState == currentStateID && t.Trigger == trigger {
			return t
		}
	}
	return nil
}

// FindFirstTransitionFrom 查找从指定状态出发的第一个转换
// 用于自动转换场景（如 dispatch 时从 todo 自动跳转到第一个处理状态）
func (c *Config) FindFirstTransitionFrom(fromState string) *Transition {
	for i := range c.Transitions {
		t := &c.Transitions[i]
		if t.FromState == fromState {
			return t
		}
	}
	return nil
}

// GetState 获取状态
func (c *Config) GetState(stateID string) *State {
	for i := range c.States {
		s := &c.States[i]
		if s.ID == stateID {
			return s
		}
	}
	return nil
}

// GetAvailableTriggers 获取指定状态可用的触发器
// 返回从该状态出发的所有转换对应的触发器指南
func (c *Config) GetAvailableTriggers(stateID string) []StateTriggerGuide {
	state := c.GetState(stateID)
	if state == nil {
		return nil
	}

	// 如果状态中已定义 triggers，直接返回
	if len(state.Triggers) > 0 {
		return state.Triggers
	}

	// 否则从 transitions 中推导
	var triggers []StateTriggerGuide
	for _, t := range c.Transitions {
		if t.FromState == stateID {
			triggers = append(triggers, StateTriggerGuide{
				Trigger:     t.Trigger,
				Description: t.Description,
				Condition:   "",
			})
		}
	}
	return triggers
}

// GetStateAIGuide 获取状态的 AI 指南
// 返回完整的 AI 执行指南信息
func (c *Config) GetStateAIGuide(stateID string) map[string]interface{} {
	state := c.GetState(stateID)
	if state == nil {
		return nil
	}

	return map[string]interface{}{
		"state":            state.ID,
		"name":             state.Name,
		"is_final":         state.IsFinal,
		"ai_guide":         state.AIGuide,
		"auto_init":        state.AutoInit,
		"success_criteria": state.SuccessCriteria,
		"failure_criteria": state.FailureCriteria,
		"triggers":         c.GetAvailableTriggers(stateID),
	}
}

// NewStateMachine 创建状态机
func NewStateMachine(name, description string, cfg *Config) *StateMachine {
	now := time.Now()
	return &StateMachine{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Config:      cfg,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
