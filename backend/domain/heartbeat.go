package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrHeartbeatIDRequired        = errors.New("heartbeat id is required")
	ErrHeartbeatProjectIDRequired = errors.New("heartbeat project id is required")
	ErrHeartbeatNameRequired      = errors.New("heartbeat name is required")
	ErrHeartbeatIntervalInvalid   = errors.New("heartbeat interval minutes must be >= 1")
)

type HeartbeatID struct {
	value string
}

func NewHeartbeatID(value string) HeartbeatID {
	return HeartbeatID{value: value}
}

func (id HeartbeatID) String() string {
	return id.value
}

type Heartbeat struct {
	id              HeartbeatID
	projectID       ProjectID
	name            string
	enabled         bool
	intervalMinutes int
	mdContent       string
	agentCode       string
	requirementType string
	sortOrder       int
	createdAt       time.Time
	updatedAt       time.Time
}

func NewHeartbeat(
	id HeartbeatID,
	projectID ProjectID,
	name string,
	intervalMinutes int,
	mdContent, agentCode, requirementType string,
) (*Heartbeat, error) {
	if id.String() == "" {
		return nil, ErrHeartbeatIDRequired
	}
	if projectID.String() == "" {
		return nil, ErrHeartbeatProjectIDRequired
	}
	if strings.TrimSpace(name) == "" {
		return nil, ErrHeartbeatNameRequired
	}
	if intervalMinutes < 1 {
		return nil, ErrHeartbeatIntervalInvalid
	}
	now := time.Now()
	return &Heartbeat{
		id:              id,
		projectID:       projectID,
		name:            strings.TrimSpace(name),
		enabled:         true,
		intervalMinutes: intervalMinutes,
		mdContent:       mdContent,
		agentCode:       strings.TrimSpace(agentCode),
		requirementType: strings.TrimSpace(requirementType),
		sortOrder:       0,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

func (h *Heartbeat) ID() HeartbeatID              { return h.id }
func (h *Heartbeat) ProjectID() ProjectID         { return h.projectID }
func (h *Heartbeat) Name() string                 { return h.name }
func (h *Heartbeat) Enabled() bool                { return h.enabled }
func (h *Heartbeat) IntervalMinutes() int         { return h.intervalMinutes }
func (h *Heartbeat) MDContent() string            { return h.mdContent }
func (h *Heartbeat) AgentCode() string            { return h.agentCode }
func (h *Heartbeat) RequirementType() string      { return h.requirementType }
func (h *Heartbeat) SortOrder() int               { return h.sortOrder }
func (h *Heartbeat) CreatedAt() time.Time         { return h.createdAt }
func (h *Heartbeat) UpdatedAt() time.Time         { return h.updatedAt }

func (h *Heartbeat) Update(name string, intervalMinutes int, mdContent, agentCode, requirementType string) error {
	if strings.TrimSpace(name) == "" {
		return ErrHeartbeatNameRequired
	}
	if intervalMinutes < 1 {
		return ErrHeartbeatIntervalInvalid
	}
	h.name = strings.TrimSpace(name)
	h.intervalMinutes = intervalMinutes
	h.mdContent = mdContent
	h.agentCode = strings.TrimSpace(agentCode)
	h.requirementType = strings.TrimSpace(requirementType)
	if h.requirementType == "" {
		h.requirementType = "heartbeat"
	}
	h.updatedAt = time.Now()
	return nil
}

func (h *Heartbeat) SetEnabled(v bool) {
	h.enabled = v
	h.updatedAt = time.Now()
}

func (h *Heartbeat) SetSortOrder(order int) {
	h.sortOrder = order
	h.updatedAt = time.Now()
}

func (h *Heartbeat) RenderPrompt(project *Project) string {
	if project == nil {
		return h.mdContent
	}
	result := h.mdContent
	result = strings.ReplaceAll(result, "${project.id}", project.ID().String())
	result = strings.ReplaceAll(result, "${project.name}", project.Name())
	result = strings.ReplaceAll(result, "${project.git_repo_url}", project.GitRepoURL())
	result = strings.ReplaceAll(result, "${project.default_branch}", project.DefaultBranch())
	result = strings.ReplaceAll(result, "${timestamp}", time.Now().Format("2006-01-02 15:04:05"))
	return result
}

type HeartbeatSnapshot struct {
	ID              HeartbeatID
	ProjectID       ProjectID
	Name            string
	Enabled         bool
	IntervalMinutes int
	MDContent       string
	AgentCode       string
	RequirementType string
	SortOrder       int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (h *Heartbeat) ToSnapshot() HeartbeatSnapshot {
	return HeartbeatSnapshot{
		ID:              h.id,
		ProjectID:       h.projectID,
		Name:            h.name,
		Enabled:         h.enabled,
		IntervalMinutes: h.intervalMinutes,
		MDContent:       h.mdContent,
		AgentCode:       h.agentCode,
		RequirementType: h.requirementType,
		SortOrder:       h.sortOrder,
		CreatedAt:       h.createdAt,
		UpdatedAt:       h.updatedAt,
	}
}

func (h *Heartbeat) FromSnapshot(s HeartbeatSnapshot) {
	h.id = s.ID
	h.projectID = s.ProjectID
	h.name = s.Name
	h.enabled = s.Enabled
	h.intervalMinutes = s.IntervalMinutes
	h.mdContent = s.MDContent
	h.agentCode = s.AgentCode
	h.requirementType = s.RequirementType
	h.sortOrder = s.SortOrder
	h.createdAt = s.CreatedAt
	h.updatedAt = s.UpdatedAt
}
