package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrWebhookConfigIDRequired   = errors.New("webhook config id is required")
	ErrWebhookConfigProjectIDReq = errors.New("webhook config project id is required")
	ErrWebhookConfigRepoRequired = errors.New("webhook config repo is required")
	ErrWebhookEventLogIDRequired = errors.New("webhook event log id is required")
	ErrWebhookBindingIDRequired  = errors.New("webhook binding id is required")
	ErrWebhookBindingConfigIDReq = errors.New("webhook binding config id is required")
	ErrWebhookBindingEventReq    = errors.New("webhook binding event type is required")
	ErrWebhookBindingHeartbeatReq = errors.New("webhook binding heartbeat id is required")
)

type GitHubWebhookConfigID struct {
	value string
}

func NewGitHubWebhookConfigID(value string) GitHubWebhookConfigID {
	return GitHubWebhookConfigID{value: value}
}

func (id GitHubWebhookConfigID) String() string {
	return id.value
}

type GitHubWebhookConfig struct {
	id        GitHubWebhookConfigID
	projectID ProjectID
	repo      string
	enabled   bool
	webhookURL string
	createdAt time.Time
	updatedAt time.Time
}

func NewGitHubWebhookConfig(
	id GitHubWebhookConfigID,
	projectID ProjectID,
	repo string,
) (*GitHubWebhookConfig, error) {
	if id.String() == "" {
		return nil, ErrWebhookConfigIDRequired
	}
	if projectID.String() == "" {
		return nil, ErrWebhookConfigProjectIDReq
	}
	if strings.TrimSpace(repo) == "" {
		return nil, ErrWebhookConfigRepoRequired
	}
	now := time.Now()
	return &GitHubWebhookConfig{
		id:        id,
		projectID: projectID,
		repo:     strings.TrimSpace(repo),
		enabled:  false,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func (c *GitHubWebhookConfig) ID() GitHubWebhookConfigID { return c.id }
func (c *GitHubWebhookConfig) ProjectID() ProjectID     { return c.projectID }
func (c *GitHubWebhookConfig) Repo() string             { return c.repo }
func (c *GitHubWebhookConfig) Enabled() bool            { return c.enabled }
func (c *GitHubWebhookConfig) WebhookURL() string       { return c.webhookURL }
func (c *GitHubWebhookConfig) CreatedAt() time.Time   { return c.createdAt }
func (c *GitHubWebhookConfig) UpdatedAt() time.Time   { return c.updatedAt }

func (c *GitHubWebhookConfig) SetEnabled(v bool) {
	c.enabled = v
	c.updatedAt = time.Now()
}

func (c *GitHubWebhookConfig) SetWebhookURL(url string) {
	c.webhookURL = url
	c.updatedAt = time.Now()
}

func (c *GitHubWebhookConfig) UpdateRepo(repo string) error {
	if strings.TrimSpace(repo) == "" {
		return ErrWebhookConfigRepoRequired
	}
	c.repo = strings.TrimSpace(repo)
	c.updatedAt = time.Now()
	return nil
}

type GitHubWebhookConfigSnapshot struct {
	ID        GitHubWebhookConfigID
	ProjectID ProjectID
	Repo     string
	Enabled  bool
	WebhookURL string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c *GitHubWebhookConfig) ToSnapshot() GitHubWebhookConfigSnapshot {
	return GitHubWebhookConfigSnapshot{
		ID:        c.id,
		ProjectID: c.projectID,
		Repo:     c.repo,
		Enabled:  c.enabled,
		WebhookURL: c.webhookURL,
		CreatedAt: c.createdAt,
		UpdatedAt: c.updatedAt,
	}
}

func (c *GitHubWebhookConfig) FromSnapshot(s GitHubWebhookConfigSnapshot) {
	c.id = s.ID
	c.projectID = s.ProjectID
	c.repo = s.Repo
	c.enabled = s.Enabled
	c.webhookURL = s.WebhookURL
	c.createdAt = s.CreatedAt
	c.updatedAt = s.UpdatedAt
}

// WebhookEventStatus represents the status of a webhook event
type WebhookEventStatus string

const (
	WebhookEventStatusReceived  WebhookEventStatus = "received"
	WebhookEventStatusProcessed WebhookEventStatus = "processed"
	WebhookEventStatusFailed    WebhookEventStatus = "failed"
)

type WebhookEventLogID struct {
	value string
}

func NewWebhookEventLogID(value string) WebhookEventLogID {
	return WebhookEventLogID{value: value}
}

func (id WebhookEventLogID) String() string {
	return id.value
}

type WebhookEventLog struct {
	id                 WebhookEventLogID
	projectID          ProjectID
	eventType         string
	payload            string
	status             WebhookEventStatus
	triggerHeartbeatID string
	requirementID      string
	errorMessage       string
	receivedAt         time.Time
}

func NewWebhookEventLog(
	id WebhookEventLogID,
	projectID ProjectID,
	eventType string,
	payload string,
) (*WebhookEventLog, error) {
	if id.String() == "" {
		return nil, ErrWebhookEventLogIDRequired
	}
	if projectID.String() == "" {
		return nil, ErrWebhookConfigProjectIDReq
	}
	return &WebhookEventLog{
		id:         id,
		projectID:  projectID,
		eventType:  eventType,
		payload:    payload,
		status:     WebhookEventStatusReceived,
		receivedAt: time.Now(),
	}, nil
}

func (l *WebhookEventLog) ID() WebhookEventLogID    { return l.id }
func (l *WebhookEventLog) ProjectID() ProjectID     { return l.projectID }
func (l *WebhookEventLog) EventType() string        { return l.eventType }
func (l *WebhookEventLog) Payload() string          { return l.payload }
func (l *WebhookEventLog) Status() WebhookEventStatus { return l.status }
func (l *WebhookEventLog) TriggerHeartbeatID() string { return l.triggerHeartbeatID }
func (l *WebhookEventLog) ErrorMessage() string        { return l.errorMessage }
func (l *WebhookEventLog) ReceivedAt() time.Time      { return l.receivedAt }
func (l *WebhookEventLog) RequirementID() string     { return l.requirementID }

func (l *WebhookEventLog) SetProcessed(heartbeatID, requirementID string) {
	l.status = WebhookEventStatusProcessed
	l.triggerHeartbeatID = heartbeatID
	l.requirementID = requirementID
}

func (l *WebhookEventLog) SetFailed(errMsg string) {
	l.status = WebhookEventStatusFailed
	l.errorMessage = errMsg
}

type WebhookEventLogSnapshot struct {
	ID                 WebhookEventLogID
	ProjectID          ProjectID
	EventType          string
	Payload            string
	Status             WebhookEventStatus
	TriggerHeartbeatID string
	RequirementID      string
	ErrorMessage       string
	ReceivedAt         time.Time
}

func (l *WebhookEventLog) ToSnapshot() WebhookEventLogSnapshot {
	return WebhookEventLogSnapshot{
		ID:                 l.id,
		ProjectID:          l.projectID,
		EventType:          l.eventType,
		Payload:            l.payload,
		Status:             l.status,
		TriggerHeartbeatID: l.triggerHeartbeatID,
		RequirementID:      l.requirementID,
		ErrorMessage:       l.errorMessage,
		ReceivedAt:         l.receivedAt,
	}
}

func (l *WebhookEventLog) FromSnapshot(s WebhookEventLogSnapshot) {
	l.id = s.ID
	l.projectID = s.ProjectID
	l.eventType = s.EventType
	l.payload = s.Payload
	l.status = s.Status
	l.triggerHeartbeatID = s.TriggerHeartbeatID
	l.requirementID = s.RequirementID
	l.errorMessage = s.ErrorMessage
	l.receivedAt = s.ReceivedAt
}

// WebhookHeartbeatBindingID
type WebhookHeartbeatBindingID struct {
	value string
}

func NewWebhookHeartbeatBindingID(value string) WebhookHeartbeatBindingID {
	return WebhookHeartbeatBindingID{value: value}
}

func (id WebhookHeartbeatBindingID) String() string {
	return id.value
}

type WebhookHeartbeatBinding struct {
	id                 WebhookHeartbeatBindingID
	projectID          ProjectID
	configID           GitHubWebhookConfigID
	githubEventType    string
	heartbeatID        HeartbeatID
	enabled            bool
	createdAt          time.Time
}

func NewWebhookHeartbeatBinding(
	id WebhookHeartbeatBindingID,
	projectID ProjectID,
	configID GitHubWebhookConfigID,
	githubEventType string,
	heartbeatID HeartbeatID,
) (*WebhookHeartbeatBinding, error) {
	if id.String() == "" {
		return nil, ErrWebhookBindingIDRequired
	}
	if projectID.String() == "" {
		return nil, ErrWebhookConfigProjectIDReq
	}
	if configID.String() == "" {
		return nil, ErrWebhookBindingConfigIDReq
	}
	if strings.TrimSpace(githubEventType) == "" {
		return nil, ErrWebhookBindingEventReq
	}
	if heartbeatID.String() == "" {
		return nil, ErrWebhookBindingHeartbeatReq
	}
	return &WebhookHeartbeatBinding{
		id:              id,
		projectID:       projectID,
		configID:        configID,
		githubEventType: strings.TrimSpace(githubEventType),
		heartbeatID:     heartbeatID,
		enabled:         true,
		createdAt:       time.Now(),
	}, nil
}

func (b *WebhookHeartbeatBinding) ID() WebhookHeartbeatBindingID    { return b.id }
func (b *WebhookHeartbeatBinding) ProjectID() ProjectID            { return b.projectID }
func (b *WebhookHeartbeatBinding) ConfigID() GitHubWebhookConfigID { return b.configID }
func (b *WebhookHeartbeatBinding) GitHubEventType() string         { return b.githubEventType }
func (b *WebhookHeartbeatBinding) HeartbeatID() HeartbeatID        { return b.heartbeatID }
func (b *WebhookHeartbeatBinding) Enabled() bool                   { return b.enabled }
func (b *WebhookHeartbeatBinding) CreatedAt() time.Time            { return b.createdAt }

func (b *WebhookHeartbeatBinding) SetEnabled(v bool) {
	b.enabled = v
}

type WebhookHeartbeatBindingSnapshot struct {
	ID              WebhookHeartbeatBindingID
	ProjectID       ProjectID
	ConfigID        GitHubWebhookConfigID
	GitHubEventType string
	HeartbeatID     HeartbeatID
	Enabled         bool
	CreatedAt       time.Time
}

func (b *WebhookHeartbeatBinding) ToSnapshot() WebhookHeartbeatBindingSnapshot {
	return WebhookHeartbeatBindingSnapshot{
		ID:              b.id,
		ProjectID:       b.projectID,
		ConfigID:        b.configID,
		GitHubEventType: b.githubEventType,
		HeartbeatID:     b.heartbeatID,
		Enabled:         b.enabled,
		CreatedAt:       b.createdAt,
	}
}

func (b *WebhookHeartbeatBinding) FromSnapshot(s WebhookHeartbeatBindingSnapshot) {
	b.id = s.ID
	b.projectID = s.ProjectID
	b.configID = s.ConfigID
	b.githubEventType = s.GitHubEventType
	b.heartbeatID = s.HeartbeatID
	b.enabled = s.Enabled
	b.createdAt = s.CreatedAt
}
