package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrHeartbeatScenarioIDRequired   = errors.New("heartbeat scenario id is required")
	ErrHeartbeatScenarioCodeRequired = errors.New("heartbeat scenario code is required")
	ErrHeartbeatScenarioNameRequired = errors.New("heartbeat scenario name is required")
)

type HeartbeatScenarioID struct {
	value string
}

func NewHeartbeatScenarioID(value string) HeartbeatScenarioID {
	return HeartbeatScenarioID{value: value}
}

func (id HeartbeatScenarioID) String() string {
	return id.value
}

// HeartbeatScenarioItem 场景中的单条心跳定义（值对象）
type HeartbeatScenarioItem struct {
	Name            string
	IntervalMinutes int
	MDContent       string
	AgentCode       string
	RequirementType string
	SortOrder       int
}

// HeartbeatScenario 心跳场景聚合根
type HeartbeatScenario struct {
	id          HeartbeatScenarioID
	code        string
	name        string
	description string
	items       []HeartbeatScenarioItem
	enabled     bool
	isBuiltIn   bool
	createdAt   time.Time
	updatedAt   time.Time
}

func NewHeartbeatScenario(
	id HeartbeatScenarioID,
	code, name, description string,
	items []HeartbeatScenarioItem,
) (*HeartbeatScenario, error) {
	if strings.TrimSpace(id.String()) == "" {
		return nil, ErrHeartbeatScenarioIDRequired
	}
	if strings.TrimSpace(code) == "" {
		return nil, ErrHeartbeatScenarioCodeRequired
	}
	if strings.TrimSpace(name) == "" {
		return nil, ErrHeartbeatScenarioNameRequired
	}
	now := time.Now()
	return &HeartbeatScenario{
		id:          id,
		code:        strings.TrimSpace(code),
		name:        strings.TrimSpace(name),
		description: strings.TrimSpace(description),
		items:       append([]HeartbeatScenarioItem(nil), items...),
		enabled:     true,
		isBuiltIn:   false,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

func (s *HeartbeatScenario) ID() HeartbeatScenarioID     { return s.id }
func (s *HeartbeatScenario) Code() string                { return s.code }
func (s *HeartbeatScenario) Name() string                { return s.name }
func (s *HeartbeatScenario) Description() string         { return s.description }
func (s *HeartbeatScenario) Items() []HeartbeatScenarioItem {
	return append([]HeartbeatScenarioItem(nil), s.items...)
}
func (s *HeartbeatScenario) Enabled() bool               { return s.enabled }
func (s *HeartbeatScenario) IsBuiltIn() bool             { return s.isBuiltIn }
func (s *HeartbeatScenario) CreatedAt() time.Time        { return s.createdAt }
func (s *HeartbeatScenario) UpdatedAt() time.Time        { return s.updatedAt }

func (s *HeartbeatScenario) Update(name, description string, items []HeartbeatScenarioItem) error {
	if strings.TrimSpace(name) == "" {
		return ErrHeartbeatScenarioNameRequired
	}
	s.name = strings.TrimSpace(name)
	s.description = strings.TrimSpace(description)
	s.items = append([]HeartbeatScenarioItem(nil), items...)
	s.updatedAt = time.Now()
	return nil
}

func (s *HeartbeatScenario) SetEnabled(v bool) {
	s.enabled = v
	s.updatedAt = time.Now()
}

func (s *HeartbeatScenario) SetIsBuiltIn(v bool) {
	s.isBuiltIn = v
	s.updatedAt = time.Now()
}

// ApplyToProject 将场景实例化为一组项目心跳
func (s *HeartbeatScenario) ApplyToProject(projectID ProjectID, idGen IDGenerator) ([]*Heartbeat, error) {
	if s.id.String() == "" {
		return nil, ErrHeartbeatScenarioIDRequired
	}
	if projectID.String() == "" {
		return nil, ErrHeartbeatProjectIDRequired
	}
	result := make([]*Heartbeat, 0, len(s.items))
	for _, item := range s.items {
		hb, err := NewHeartbeat(
			NewHeartbeatID(idGen.Generate()),
			projectID,
			s.name+" - "+item.Name,
			item.IntervalMinutes,
			item.MDContent,
			item.AgentCode,
			item.RequirementType,
		)
		if err != nil {
			return nil, err
		}
		hb.SetSortOrder(item.SortOrder)
		result = append(result, hb)
	}
	return result, nil
}

type HeartbeatScenarioSnapshot struct {
	ID          HeartbeatScenarioID
	Code        string
	Name        string
	Description string
	Items       []HeartbeatScenarioItem
	Enabled     bool
	IsBuiltIn   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (s *HeartbeatScenario) ToSnapshot() HeartbeatScenarioSnapshot {
	return HeartbeatScenarioSnapshot{
		ID:          s.id,
		Code:        s.code,
		Name:        s.name,
		Description: s.description,
		Items:       append([]HeartbeatScenarioItem(nil), s.items...),
		Enabled:     s.enabled,
		IsBuiltIn:   s.isBuiltIn,
		CreatedAt:   s.createdAt,
		UpdatedAt:   s.updatedAt,
	}
}

func (s *HeartbeatScenario) FromSnapshot(snap HeartbeatScenarioSnapshot) error {
	if strings.TrimSpace(snap.ID.String()) == "" {
		return ErrHeartbeatScenarioIDRequired
	}
	if strings.TrimSpace(snap.Code) == "" {
		return ErrHeartbeatScenarioCodeRequired
	}
	if strings.TrimSpace(snap.Name) == "" {
		return ErrHeartbeatScenarioNameRequired
	}
	s.id = snap.ID
	s.code = strings.TrimSpace(snap.Code)
	s.name = strings.TrimSpace(snap.Name)
	s.description = strings.TrimSpace(snap.Description)
	s.items = append([]HeartbeatScenarioItem(nil), snap.Items...)
	s.enabled = snap.Enabled
	s.isBuiltIn = snap.IsBuiltIn
	s.createdAt = snap.CreatedAt
	s.updatedAt = snap.UpdatedAt
	return nil
}
