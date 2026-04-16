package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrHeartbeatTemplateIDRequired   = errors.New("heartbeat template id is required")
	ErrHeartbeatTemplateNameRequired = errors.New("heartbeat template name is required")
)

type HeartbeatTemplateID struct {
	value string
}

func NewHeartbeatTemplateID(value string) HeartbeatTemplateID {
	return HeartbeatTemplateID{value: value}
}

func (id HeartbeatTemplateID) String() string {
	return id.value
}

type HeartbeatTemplate struct {
	id              HeartbeatTemplateID
	name            string
	mdContent       string
	requirementType string
	createdAt       time.Time
	updatedAt       time.Time
}

func NewHeartbeatTemplate(
	id HeartbeatTemplateID,
	name, mdContent, requirementType string,
) (*HeartbeatTemplate, error) {
	if strings.TrimSpace(id.String()) == "" {
		return nil, ErrHeartbeatTemplateIDRequired
	}
	if strings.TrimSpace(name) == "" {
		return nil, ErrHeartbeatTemplateNameRequired
	}
	if strings.TrimSpace(requirementType) == "" {
		requirementType = "heartbeat"
	}
	now := time.Now()
	return &HeartbeatTemplate{
		id:              id,
		name:            strings.TrimSpace(name),
		mdContent:       mdContent,
		requirementType: strings.TrimSpace(requirementType),
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

func (t *HeartbeatTemplate) ID() HeartbeatTemplateID     { return t.id }
func (t *HeartbeatTemplate) Name() string                { return t.name }
func (t *HeartbeatTemplate) MDContent() string           { return t.mdContent }
func (t *HeartbeatTemplate) RequirementType() string     { return t.requirementType }
func (t *HeartbeatTemplate) CreatedAt() time.Time        { return t.createdAt }
func (t *HeartbeatTemplate) UpdatedAt() time.Time        { return t.updatedAt }

func (t *HeartbeatTemplate) Update(name, mdContent, requirementType string) error {
	if strings.TrimSpace(name) == "" {
		return ErrHeartbeatTemplateNameRequired
	}
	if strings.TrimSpace(requirementType) == "" {
		requirementType = "heartbeat"
	}
	t.name = strings.TrimSpace(name)
	t.mdContent = mdContent
	t.requirementType = strings.TrimSpace(requirementType)
	t.updatedAt = time.Now()
	return nil
}

type HeartbeatTemplateSnapshot struct {
	ID              HeartbeatTemplateID
	Name            string
	MDContent       string
	RequirementType string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (t *HeartbeatTemplate) ToSnapshot() HeartbeatTemplateSnapshot {
	return HeartbeatTemplateSnapshot{
		ID:              t.id,
		Name:            t.name,
		MDContent:       t.mdContent,
		RequirementType: t.requirementType,
		CreatedAt:       t.createdAt,
		UpdatedAt:       t.updatedAt,
	}
}

func (t *HeartbeatTemplate) FromSnapshot(s HeartbeatTemplateSnapshot) error {
	if strings.TrimSpace(s.ID.String()) == "" {
		return ErrHeartbeatTemplateIDRequired
	}
	if strings.TrimSpace(s.Name) == "" {
		return ErrHeartbeatTemplateNameRequired
	}
	requirementType := strings.TrimSpace(s.RequirementType)
	if requirementType == "" {
		requirementType = "heartbeat"
	}
	t.id = s.ID
	t.name = strings.TrimSpace(s.Name)
	t.mdContent = s.MDContent
	t.requirementType = requirementType
	t.createdAt = s.CreatedAt
	t.updatedAt = s.UpdatedAt
	return nil
}
