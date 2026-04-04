package state_machine

import (
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// State 状态节点
type State struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	IsFinal  bool   `json:"is_final" yaml:"is_final"`
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
	ProjectID   string    `json:"project_id"`
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

// NewStateMachine 创建状态机
func NewStateMachine(projectID, name, description string, cfg *Config) *StateMachine {
	now := time.Now()
	return &StateMachine{
		ID:          uuid.New().String(),
		ProjectID:   projectID,
		Name:        name,
		Description: description,
		Config:      cfg,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
